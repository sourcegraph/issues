package resolvers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/graph-gophers/graphql-go"
	"github.com/sourcegraph/campaignutils/overridable"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend"
	"github.com/sourcegraph/sourcegraph/cmd/repo-updater/repos"
	ee "github.com/sourcegraph/sourcegraph/enterprise/internal/campaigns"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/campaigns/resolvers/apitest"
	ct "github.com/sourcegraph/sourcegraph/enterprise/internal/campaigns/testing"
	"github.com/sourcegraph/sourcegraph/internal/actor"
	"github.com/sourcegraph/sourcegraph/internal/campaigns"
	"github.com/sourcegraph/sourcegraph/internal/db"
	"github.com/sourcegraph/sourcegraph/internal/db/dbconn"
	"github.com/sourcegraph/sourcegraph/internal/db/dbtesting"
)

func TestNullIDResilience(t *testing.T) {
	sr := &Resolver{store: ee.NewStore(dbconn.Global)}

	s, err := graphqlbackend.NewSchema(sr, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	ctx := backend.WithAuthzBypass(context.Background())

	ids := []graphql.ID{
		marshalCampaignID(0),
		marshalChangesetID(0),
		marshalCampaignSpecRandID(""),
		marshalChangesetSpecRandID(""),
	}

	for _, id := range ids {
		var response struct{ Node struct{ ID string } }

		query := fmt.Sprintf(`query { node(id: %q) { id } }`, id)
		apitest.MustExec(ctx, t, s, nil, &response, query)

		if have, want := response.Node.ID, ""; have != want {
			t.Fatalf("node has wrong ID. have=%q, want=%q", have, want)
		}
	}

	mutations := []string{
		fmt.Sprintf(`mutation { closeCampaign(campaign: %q) { id } }`, marshalCampaignID(0)),
		fmt.Sprintf(`mutation { deleteCampaign(campaign: %q) { alwaysNil } }`, marshalCampaignID(0)),
		fmt.Sprintf(`mutation { syncChangeset(changeset: %q) { alwaysNil } }`, marshalChangesetID(0)),
		fmt.Sprintf(`mutation { applyCampaign(campaignSpec: %q) { id } }`, marshalCampaignSpecRandID("")),
		fmt.Sprintf(`mutation { createCampaign(campaignSpec: %q) { id } }`, marshalCampaignSpecRandID("")),
		fmt.Sprintf(`mutation { moveCampaign(campaign: %q, newName: "foobar") { id } }`, marshalCampaignID(0)),
	}

	for _, m := range mutations {
		var response struct{}
		errs := apitest.Exec(ctx, t, s, nil, &response, m)
		if len(errs) == 0 {
			t.Fatalf("expected errors but none returned (mutation: %q)", m)
		}
		if have, want := errs[0].Error(), fmt.Sprintf("graphql: %s", ErrIDIsZero.Error()); have != want {
			t.Fatalf("wrong errors. have=%s, want=%s (mutation: %q)", have, want, m)
		}
	}
}

func TestCreateCampaignSpec(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	ctx := context.Background()
	dbtesting.SetupGlobalTestDB(t)

	username := "create-campaign-spec-username"
	userID := insertTestUser(t, dbconn.Global, username, true)

	store := ee.NewStore(dbconn.Global)
	reposStore := repos.NewDBStore(dbconn.Global, sql.TxOptions{})

	repo := newGitHubTestRepo("github.com/sourcegraph/sourcegraph", newGitHubExternalService(t, reposStore))
	if err := reposStore.InsertRepos(ctx, repo); err != nil {
		t.Fatal(err)
	}

	changesetSpec := &campaigns.ChangesetSpec{
		Spec: &campaigns.ChangesetSpecDescription{
			BaseRepository: graphqlbackend.MarshalRepositoryID(repo.ID),
		},
		RepoID: repo.ID,
		UserID: userID,
	}
	if err := store.CreateChangesetSpec(ctx, changesetSpec); err != nil {
		t.Fatal(err)
	}

	r := &Resolver{store: store}
	s, err := graphqlbackend.NewSchema(r, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	userAPIID := string(graphqlbackend.MarshalUserID(userID))
	changesetSpecID := marshalChangesetSpecRandID(changesetSpec.RandID)
	rawSpec := ct.TestRawCampaignSpec

	input := map[string]interface{}{
		"namespace":      userAPIID,
		"campaignSpec":   rawSpec,
		"changesetSpecs": []graphql.ID{changesetSpecID},
	}

	var response struct{ CreateCampaignSpec apitest.CampaignSpec }

	actorCtx := actor.WithActor(ctx, actor.FromUser(userID))
	apitest.MustExec(actorCtx, t, s, input, &response, mutationCreateCampaignSpec)

	var unmarshaled interface{}
	err = json.Unmarshal([]byte(rawSpec), &unmarshaled)
	if err != nil {
		t.Fatal(err)
	}
	have := response.CreateCampaignSpec

	want := apitest.CampaignSpec{
		ID:            have.ID,
		CreatedAt:     have.CreatedAt,
		ExpiresAt:     have.ExpiresAt,
		OriginalInput: rawSpec,
		ParsedInput:   graphqlbackend.JSONValue{Value: unmarshaled},
		ApplyURL:      fmt.Sprintf("/users/%s/campaigns/apply/%s", username, have.ID),
		Namespace:     apitest.UserOrg{ID: userAPIID, DatabaseID: userID, SiteAdmin: true},
		Creator:       &apitest.User{ID: userAPIID, DatabaseID: userID, SiteAdmin: true},
		ChangesetSpecs: apitest.ChangesetSpecConnection{
			Nodes: []apitest.ChangesetSpec{
				{
					Typename: "VisibleChangesetSpec",
					ID:       string(changesetSpecID),
				},
			},
		},
	}

	if diff := cmp.Diff(want, have); diff != "" {
		t.Fatalf("unexpected response (-want +got):\n%s", diff)
	}
}

const mutationCreateCampaignSpec = `
fragment u on User { id, databaseID, siteAdmin }
fragment o on Org  { id, name }

mutation($namespace: ID!, $campaignSpec: String!, $changesetSpecs: [ID!]!){
  createCampaignSpec(namespace: $namespace, campaignSpec: $campaignSpec, changesetSpecs: $changesetSpecs) {
    id
    originalInput
    parsedInput

    creator  { ...u }
    namespace {
      ... on User { ...u }
      ... on Org  { ...o }
    }

    applyURL

	changesetSpecs {
	  nodes {
		  __typename
		  ... on VisibleChangesetSpec {
			  id
		  }
	  }
	}

    createdAt
    expiresAt
  }
}
`

func TestCreateChangesetSpec(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	ctx := context.Background()
	dbtesting.SetupGlobalTestDB(t)

	userID := insertTestUser(t, dbconn.Global, "create-changeset-spec", true)

	store := ee.NewStore(dbconn.Global)
	reposStore := repos.NewDBStore(dbconn.Global, sql.TxOptions{})

	repo := newGitHubTestRepo("github.com/sourcegraph/sourcegraph", newGitHubExternalService(t, reposStore))
	if err := reposStore.InsertRepos(ctx, repo); err != nil {
		t.Fatal(err)
	}

	r := &Resolver{store: store}
	s, err := graphqlbackend.NewSchema(r, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	input := map[string]interface{}{
		"changesetSpec": ct.NewRawChangesetSpecGitBranch(graphqlbackend.MarshalRepositoryID(repo.ID), "d34db33f"),
	}

	var response struct{ CreateChangesetSpec apitest.ChangesetSpec }

	actorCtx := actor.WithActor(ctx, actor.FromUser(userID))
	apitest.MustExec(actorCtx, t, s, input, &response, mutationCreateChangesetSpec)

	have := response.CreateChangesetSpec

	want := apitest.ChangesetSpec{
		Typename:  "VisibleChangesetSpec",
		ID:        have.ID,
		ExpiresAt: have.ExpiresAt,
	}

	if diff := cmp.Diff(want, have); diff != "" {
		t.Fatalf("unexpected response (-want +got):\n%s", diff)
	}

	randID, err := unmarshalChangesetSpecID(graphql.ID(want.ID))
	if err != nil {
		t.Fatal(err)
	}

	cs, err := store.GetChangesetSpec(ctx, ee.GetChangesetSpecOpts{RandID: randID})
	if err != nil {
		t.Fatal(err)
	}

	if have, want := cs.RepoID, repo.ID; have != want {
		t.Fatalf("wrong RepoID. want=%d, have=%d", want, have)
	}
}

const mutationCreateChangesetSpec = `
mutation($changesetSpec: String!){
  createChangesetSpec(changesetSpec: $changesetSpec) {
	__typename
	... on VisibleChangesetSpec {
		id
		expiresAt
	}
  }
}
`

func TestApplyCampaign(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	ctx := context.Background()
	dbtesting.SetupGlobalTestDB(t)

	userID := insertTestUser(t, dbconn.Global, "apply-campaign", true)

	now := time.Now().UTC().Truncate(time.Microsecond)
	clock := func() time.Time {
		return now.UTC().Truncate(time.Microsecond)
	}
	store := ee.NewStoreWithClock(dbconn.Global, clock)
	reposStore := repos.NewDBStore(dbconn.Global, sql.TxOptions{})

	repo := newGitHubTestRepo("github.com/sourcegraph/sourcegraph", newGitHubExternalService(t, reposStore))
	if err := reposStore.InsertRepos(ctx, repo); err != nil {
		t.Fatal(err)
	}

	repoAPIID := graphqlbackend.MarshalRepositoryID(repo.ID)

	campaignSpec := &campaigns.CampaignSpec{
		RawSpec: ct.TestRawCampaignSpec,
		Spec: campaigns.CampaignSpecFields{
			Name:        "my-campaign",
			Description: "My description",
			ChangesetTemplate: campaigns.ChangesetTemplate{
				Title:  "Hello there",
				Body:   "This is the body",
				Branch: "my-branch",
				Commit: campaigns.CommitTemplate{
					Message: "Add hello world",
				},
				Published: overridable.FromBoolOrString(false),
			},
		},
		UserID:          userID,
		NamespaceUserID: userID,
	}
	if err := store.CreateCampaignSpec(ctx, campaignSpec); err != nil {
		t.Fatal(err)
	}

	changesetSpec := &campaigns.ChangesetSpec{
		CampaignSpecID: campaignSpec.ID,
		Spec: &campaigns.ChangesetSpecDescription{
			BaseRepository: repoAPIID,
		},
		RepoID: repo.ID,
		UserID: userID,
	}
	if err := store.CreateChangesetSpec(ctx, changesetSpec); err != nil {
		t.Fatal(err)
	}

	r := &Resolver{store: store}
	s, err := graphqlbackend.NewSchema(r, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	userAPIID := string(graphqlbackend.MarshalUserID(userID))
	input := map[string]interface{}{
		"campaignSpec": string(marshalCampaignSpecRandID(campaignSpec.RandID)),
	}

	var response struct{ ApplyCampaign apitest.Campaign }
	actorCtx := actor.WithActor(ctx, actor.FromUser(userID))
	apitest.MustExec(actorCtx, t, s, input, &response, mutationApplyCampaign)

	apiUser := &apitest.User{
		ID:         userAPIID,
		DatabaseID: userID,
		SiteAdmin:  true,
	}

	have := response.ApplyCampaign
	want := apitest.Campaign{
		ID:          have.ID,
		Name:        campaignSpec.Spec.Name,
		Description: campaignSpec.Spec.Description,
		Namespace: apitest.UserOrg{
			ID:         userAPIID,
			DatabaseID: userID,
			SiteAdmin:  true,
		},
		InitialApplier: apiUser,
		LastApplier:    apiUser,
		LastAppliedAt:  marshalDateTime(t, now),
		Changesets: apitest.ChangesetConnection{
			Nodes: []apitest.Changeset{
				{Typename: "ExternalChangeset", ReconcilerState: "QUEUED"},
			},
			TotalCount: 1,
		},
	}

	if diff := cmp.Diff(want, have); diff != "" {
		t.Fatalf("unexpected response (-want +got):\n%s", diff)
	}

	// Now we execute it again and make sure we get the same campaign back
	apitest.MustExec(actorCtx, t, s, input, &response, mutationApplyCampaign)
	have2 := response.ApplyCampaign
	if diff := cmp.Diff(want, have2); diff != "" {
		t.Fatalf("unexpected response (-want +got):\n%s", diff)
	}

	// Execute it again with ensureCampaign set to correct campaign's ID
	input["ensureCampaign"] = have2.ID
	apitest.MustExec(actorCtx, t, s, input, &response, mutationApplyCampaign)
	have3 := response.ApplyCampaign
	if diff := cmp.Diff(want, have3); diff != "" {
		t.Fatalf("unexpected response (-want +got):\n%s", diff)
	}

	// Execute it again but ensureCampaign set to wrong campaign ID
	campaignID, err := unmarshalCampaignID(graphql.ID(have3.ID))
	if err != nil {
		t.Fatal(err)
	}
	input["ensureCampaign"] = marshalCampaignID(campaignID + 999)
	errs := apitest.Exec(actorCtx, t, s, input, &response, mutationApplyCampaign)
	if len(errs) == 0 {
		t.Fatalf("expected errors, got none")
	}
}

const mutationApplyCampaign = `
fragment u on User { id, databaseID, siteAdmin }
fragment o on Org  { id, name }

mutation($campaignSpec: ID!, $ensureCampaign: ID){
  applyCampaign(campaignSpec: $campaignSpec, ensureCampaign: $ensureCampaign) {
    id, name, description
    initialApplier    { ...u }
    lastApplier       { ...u }
    lastAppliedAt
    namespace {
        ... on User { ...u }
        ... on Org  { ...o }
    }

    changesets {
      nodes {
        __typename

        ... on ExternalChangeset {
          reconcilerState
        }
        ... on HiddenExternalChangeset {
          reconcilerState
        }
      }

      totalCount
    }
  }
}
`

func TestCreateCampaign(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	ctx := context.Background()
	dbtesting.SetupGlobalTestDB(t)

	userID := insertTestUser(t, dbconn.Global, "apply-campaign", true)

	store := ee.NewStore(dbconn.Global)

	campaignSpec := &campaigns.CampaignSpec{
		RawSpec: ct.TestRawCampaignSpec,
		Spec: campaigns.CampaignSpecFields{
			Name:        "my-campaign",
			Description: "My description",
		},
		UserID:          userID,
		NamespaceUserID: userID,
	}
	if err := store.CreateCampaignSpec(ctx, campaignSpec); err != nil {
		t.Fatal(err)
	}

	r := &Resolver{store: store}
	s, err := graphqlbackend.NewSchema(r, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	input := map[string]interface{}{
		"campaignSpec": string(marshalCampaignSpecRandID(campaignSpec.RandID)),
	}

	var response struct{ CreateCampaign apitest.Campaign }
	actorCtx := actor.WithActor(ctx, actor.FromUser(userID))

	// First time it should work, because no campaign exists
	apitest.MustExec(actorCtx, t, s, input, &response, mutationCreateCampaign)

	if response.CreateCampaign.ID == "" {
		t.Fatalf("expected campaign to be created, but was not")
	}

	// Second time it should fail
	errors := apitest.Exec(actorCtx, t, s, input, &response, mutationCreateCampaign)

	if len(errors) != 1 {
		t.Fatalf("expected single errors, but got none")
	}
	if have, want := errors[0].Message, ee.ErrMatchingCampaignExists.Error(); have != want {
		t.Fatalf("wrong error. want=%q, have=%q", want, have)
	}
}

const mutationCreateCampaign = `
mutation($campaignSpec: ID!){
  createCampaign(campaignSpec: $campaignSpec) { id }
}
`

func TestMoveCampaign(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	ctx := context.Background()
	dbtesting.SetupGlobalTestDB(t)

	username := "move-campaign-username"
	userID := insertTestUser(t, dbconn.Global, username, true)

	org, err := db.Orgs.Create(ctx, "org", nil)
	if err != nil {
		t.Fatal(err)
	}

	store := ee.NewStore(dbconn.Global)

	campaignSpec := &campaigns.CampaignSpec{
		RawSpec:         ct.TestRawCampaignSpec,
		UserID:          userID,
		NamespaceUserID: userID,
	}
	if err := store.CreateCampaignSpec(ctx, campaignSpec); err != nil {
		t.Fatal(err)
	}

	campaign := &campaigns.Campaign{
		CampaignSpecID:   campaignSpec.ID,
		Name:             "old-name",
		InitialApplierID: userID,
		LastApplierID:    userID,
		LastAppliedAt:    time.Now(),
		NamespaceUserID:  campaignSpec.UserID,
	}
	if err := store.CreateCampaign(ctx, campaign); err != nil {
		t.Fatal(err)
	}

	r := &Resolver{store: store}
	s, err := graphqlbackend.NewSchema(r, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Move to a new name
	campaignAPIID := string(marshalCampaignID(campaign.ID))
	newCampaignName := "new-name"
	input := map[string]interface{}{
		"campaign": campaignAPIID,
		"newName":  newCampaignName,
	}

	var response struct{ MoveCampaign apitest.Campaign }
	actorCtx := actor.WithActor(ctx, actor.FromUser(userID))
	apitest.MustExec(actorCtx, t, s, input, &response, mutationMoveCampaign)

	haveCampaign := response.MoveCampaign
	if diff := cmp.Diff(input["newName"], haveCampaign.Name); diff != "" {
		t.Fatalf("unexpected name (-want +got):\n%s", diff)
	}

	wantURL := fmt.Sprintf("/users/%s/campaigns/%s", username, newCampaignName)
	if diff := cmp.Diff(wantURL, haveCampaign.URL); diff != "" {
		t.Fatalf("unexpected URL (-want +got):\n%s", diff)
	}

	// Move to a new namespace
	orgAPIID := graphqlbackend.MarshalOrgID(org.ID)
	input = map[string]interface{}{
		"campaign":     string(marshalCampaignID(campaign.ID)),
		"newNamespace": orgAPIID,
	}

	apitest.MustExec(actorCtx, t, s, input, &response, mutationMoveCampaign)

	haveCampaign = response.MoveCampaign
	if diff := cmp.Diff(string(orgAPIID), haveCampaign.Namespace.ID); diff != "" {
		t.Fatalf("unexpected namespace (-want +got):\n%s", diff)
	}
	wantURL = fmt.Sprintf("/organizations/%s/campaigns/%s", org.Name, newCampaignName)
	if diff := cmp.Diff(wantURL, haveCampaign.URL); diff != "" {
		t.Fatalf("unexpected URL (-want +got):\n%s", diff)
	}
}

const mutationMoveCampaign = `
fragment u on User { id, databaseID, siteAdmin }
fragment o on Org  { id, name }

mutation($campaign: ID!, $newName: String, $newNamespace: ID){
  moveCampaign(campaign: $campaign, newName: $newName, newNamespace: $newNamespace) {
	id, name, description
	initialApplier  { ...u }
	namespace {
		... on User { ...u }
		... on Org  { ...o }
	}
	url
  }
}
`

func TestListChangesetOptsFromArgs(t *testing.T) {
	var wantFirst int32 = 10
	wantPublicationStates := []campaigns.ChangesetPublicationState{
		"PUBLISHED",
		"INVALID",
	}
	reconcilerStates := [][]campaigns.ReconcilerState{
		{"PROCESSING"},
		{campaigns.ReconcilerStateProcessing},
		{"INVALID"},
	}
	wantExternalStates := []campaigns.ChangesetExternalState{"OPEN", "INVALID"}
	wantReviewStates := []campaigns.ChangesetReviewState{"APPROVED", "INVALID"}
	wantCheckStates := []campaigns.ChangesetCheckState{"PENDING", "INVALID"}
	wantOnlyPublishedByThisCampaign := []bool{true}
	var campaignID int64 = 1

	tcs := []struct {
		args       *graphqlbackend.ListChangesetsArgs
		wantSafe   bool
		wantErr    string
		wantParsed ee.ListChangesetsOpts
	}{
		// No args given.
		{
			args:       nil,
			wantSafe:   true,
			wantParsed: ee.ListChangesetsOpts{},
		},
		// First argument is set in opts, and considered safe.
		{
			args: &graphqlbackend.ListChangesetsArgs{
				First: wantFirst,
			},
			wantSafe:   true,
			wantParsed: ee.ListChangesetsOpts{LimitOpts: ee.LimitOpts{Limit: 10}},
		},
		// Setting publication state is safe and transferred to opts.
		{
			args: &graphqlbackend.ListChangesetsArgs{
				PublicationState: &wantPublicationStates[0],
			},
			wantSafe: true,
			wantParsed: ee.ListChangesetsOpts{
				PublicationState: &wantPublicationStates[0],
			},
		},
		// Setting invalid publication state fails.
		{
			args: &graphqlbackend.ListChangesetsArgs{
				PublicationState: &wantPublicationStates[1],
			},
			wantErr: "changeset publication state not valid",
		},
		// Setting reconciler state is safe and transferred to opts as lowercase version.
		{
			args: &graphqlbackend.ListChangesetsArgs{
				ReconcilerState: &reconcilerStates[0],
			},
			wantSafe: true,
			wantParsed: ee.ListChangesetsOpts{
				ReconcilerStates: reconcilerStates[1],
			},
		},
		// Setting invalid reconciler state fails.
		{
			args: &graphqlbackend.ListChangesetsArgs{
				ReconcilerState: &reconcilerStates[2],
			},
			wantErr: "changeset reconciler state not valid",
		},
		// Setting external state is safe and transferred to opts.
		{
			args: &graphqlbackend.ListChangesetsArgs{
				ExternalState: &wantExternalStates[0],
			},
			wantSafe:   true,
			wantParsed: ee.ListChangesetsOpts{ExternalState: &wantExternalStates[0]},
		},
		// Setting invalid external state fails.
		{
			args: &graphqlbackend.ListChangesetsArgs{
				ExternalState: &wantExternalStates[1],
			},
			wantErr: "changeset external state not valid",
		},
		// Setting review state is not safe and transferred to opts.
		{
			args: &graphqlbackend.ListChangesetsArgs{
				ReviewState: &wantReviewStates[0],
			},
			wantSafe:   false,
			wantParsed: ee.ListChangesetsOpts{ExternalReviewState: &wantReviewStates[0]},
		},
		// Setting invalid review state fails.
		{
			args: &graphqlbackend.ListChangesetsArgs{
				ReviewState: &wantReviewStates[1],
			},
			wantErr: "changeset review state not valid",
		},
		// Setting check state is not safe and transferred to opts.
		{
			args: &graphqlbackend.ListChangesetsArgs{
				CheckState: &wantCheckStates[0],
			},
			wantSafe:   false,
			wantParsed: ee.ListChangesetsOpts{ExternalCheckState: &wantCheckStates[0]},
		},
		// Setting invalid check state fails.
		{
			args: &graphqlbackend.ListChangesetsArgs{
				CheckState: &wantCheckStates[1],
			},
			wantErr: "changeset check state not valid",
		},
		// Setting OnlyPublishedByThisCampaign true.
		{
			args: &graphqlbackend.ListChangesetsArgs{
				OnlyPublishedByThisCampaign: &wantOnlyPublishedByThisCampaign[0],
			},
			wantSafe: true,
			wantParsed: ee.ListChangesetsOpts{
				PublicationState:  &wantPublicationStates[0],
				OwnedByCampaignID: campaignID,
			},
		},
	}
	for i, tc := range tcs {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			haveParsed, haveSafe, err := listChangesetOptsFromArgs(tc.args, campaignID)
			if tc.wantErr == "" && err != nil {
				t.Fatal(err)
			}
			haveErr := fmt.Sprintf("%v", err)
			wantErr := tc.wantErr
			if wantErr == "" {
				wantErr = "<nil>"
			}
			if have, want := haveErr, wantErr; have != want {
				t.Errorf("wrong error returned. have=%q want=%q", have, want)
			}
			if diff := cmp.Diff(haveParsed, tc.wantParsed); diff != "" {
				t.Errorf("wrong args returned. diff=%s", diff)
			}
			if have, want := haveSafe, tc.wantSafe; have != want {
				t.Errorf("wrong safe value returned. have=%t want=%t", have, want)
			}
		})
	}
}
