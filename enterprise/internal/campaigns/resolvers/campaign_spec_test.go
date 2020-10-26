package resolvers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend"
	"github.com/sourcegraph/sourcegraph/cmd/repo-updater/repos"
	ee "github.com/sourcegraph/sourcegraph/enterprise/internal/campaigns"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/campaigns/resolvers/apitest"
	ct "github.com/sourcegraph/sourcegraph/enterprise/internal/campaigns/testing"
	"github.com/sourcegraph/sourcegraph/internal/campaigns"
	"github.com/sourcegraph/sourcegraph/internal/db"
	"github.com/sourcegraph/sourcegraph/internal/db/dbconn"
	"github.com/sourcegraph/sourcegraph/internal/db/dbtesting"
)

func TestCampaignSpecResolver(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	ctx := backend.WithAuthzBypass(context.Background())
	dbtesting.SetupGlobalTestDB(t)

	store := ee.NewStore(dbconn.Global)

	reposStore := repos.NewDBStore(dbconn.Global, sql.TxOptions{})

	repo := newGitHubTestRepo("github.com/sourcegraph/sourcegraph", newGitHubExternalService(t, reposStore))
	if err := reposStore.InsertRepos(ctx, repo); err != nil {
		t.Fatal(err)
	}
	repoID := graphqlbackend.MarshalRepositoryID(repo.ID)

	username := "campaign-spec-by-id-user-name"
	orgname := "test-org"
	userID := insertTestUser(t, dbconn.Global, username, false)
	org, err := db.Orgs.Create(ctx, orgname, nil)
	if err != nil {
		t.Fatal(err)
	}
	orgID := org.ID

	spec, err := campaigns.NewCampaignSpecFromRaw(ct.TestRawCampaignSpec)
	if err != nil {
		t.Fatal(err)
	}
	spec.UserID = userID
	spec.NamespaceOrgID = orgID
	if err := store.CreateCampaignSpec(ctx, spec); err != nil {
		t.Fatal(err)
	}

	changesetSpec, err := campaigns.NewChangesetSpecFromRaw(ct.NewRawChangesetSpecGitBranch(repoID, "deadb33f"))
	if err != nil {
		t.Fatal(err)
	}
	changesetSpec.CampaignSpecID = spec.ID
	changesetSpec.UserID = userID
	changesetSpec.RepoID = repo.ID

	if err := store.CreateChangesetSpec(ctx, changesetSpec); err != nil {
		t.Fatal(err)
	}

	matchingCampaign := &campaigns.Campaign{
		Name:             spec.Spec.Name,
		NamespaceOrgID:   orgID,
		InitialApplierID: userID,
		LastApplierID:    userID,
		LastAppliedAt:    time.Now(),
		CampaignSpecID:   spec.ID,
	}
	if err := store.CreateCampaign(ctx, matchingCampaign); err != nil {
		t.Fatal(err)
	}

	s, err := graphqlbackend.NewSchema(&Resolver{store: store}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	apiID := string(marshalCampaignSpecRandID(spec.RandID))
	userAPIID := string(graphqlbackend.MarshalUserID(userID))
	orgAPIID := string(graphqlbackend.MarshalOrgID(orgID))

	var unmarshaled interface{}
	err = json.Unmarshal([]byte(spec.RawSpec), &unmarshaled)
	if err != nil {
		t.Fatal(err)
	}

	want := apitest.CampaignSpec{
		Typename: "CampaignSpec",
		ID:       apiID,

		OriginalInput: spec.RawSpec,
		ParsedInput:   graphqlbackend.JSONValue{Value: unmarshaled},

		ApplyURL:            fmt.Sprintf("/organizations/%s/campaigns/apply/%s", orgname, apiID),
		Namespace:           apitest.UserOrg{ID: orgAPIID, Name: orgname},
		Creator:             &apitest.User{ID: userAPIID, DatabaseID: userID},
		ViewerCanAdminister: true,

		CreatedAt: graphqlbackend.DateTime{Time: spec.CreatedAt.Truncate(time.Second)},
		ExpiresAt: &graphqlbackend.DateTime{Time: spec.ExpiresAt().Truncate(time.Second)},

		ChangesetSpecs: apitest.ChangesetSpecConnection{
			TotalCount: 1,
			Nodes: []apitest.ChangesetSpec{
				{
					ID:       string(marshalChangesetSpecRandID(changesetSpec.RandID)),
					Typename: "VisibleChangesetSpec",
					Description: apitest.ChangesetSpecDescription{
						BaseRepository: apitest.Repository{
							ID:   string(repoID),
							Name: repo.Name,
						},
					},
				},
			},
		},

		DiffStat: apitest.DiffStat{
			Added:   changesetSpec.DiffStatAdded,
			Changed: changesetSpec.DiffStatChanged,
			Deleted: changesetSpec.DiffStatDeleted,
		},

		AppliesToCampaign: apitest.Campaign{
			ID: string(marshalCampaignID(matchingCampaign.ID)),
		},
	}

	input := map[string]interface{}{"campaignSpec": apiID}
	{
		var response struct{ Node apitest.CampaignSpec }
		apitest.MustExec(ctx, t, s, input, &response, queryCampaignSpecNode)

		if diff := cmp.Diff(want, response.Node); diff != "" {
			t.Fatalf("unexpected response (-want +got):\n%s", diff)
		}
	}

	// Now soft-delete the creator and check that the campaign spec is still retrievable.
	err = db.Users.Delete(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	{
		var response struct{ Node apitest.CampaignSpec }
		apitest.MustExec(ctx, t, s, input, &response, queryCampaignSpecNode)

		// Expect creator to not be returned anymore.
		want.Creator = nil

		if diff := cmp.Diff(want, response.Node); diff != "" {
			t.Fatalf("unexpected response (-want +got):\n%s", diff)
		}
	}

	// Now hard-delete the creator and check that the campaign spec is still retrievable.
	err = db.Users.HardDelete(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	{
		var response struct{ Node apitest.CampaignSpec }
		apitest.MustExec(ctx, t, s, input, &response, queryCampaignSpecNode)

		// Expect creator to not be returned anymore.
		want.Creator = nil

		if diff := cmp.Diff(want, response.Node); diff != "" {
			t.Fatalf("unexpected response (-want +got):\n%s", diff)
		}
	}
}

const queryCampaignSpecNode = `
fragment u on User { id, databaseID, siteAdmin }
fragment o on Org  { id, name }

query($campaignSpec: ID!) {
  node(id: $campaignSpec) {
    __typename

    ... on CampaignSpec {
      id
      originalInput
      parsedInput

      creator  { ...u }
      namespace {
        ... on User { ...u }
        ... on Org  { ...o }
      }

      applyURL
      viewerCanAdminister

      createdAt
      expiresAt

      diffStat { added, deleted, changed }

      appliesToCampaign { id }

      changesetSpecs(first: 100) {
        totalCount

        nodes {
          __typename
          type

          ... on HiddenChangesetSpec {
            id
          }

          ... on VisibleChangesetSpec {
            id

            description {
              ... on ExistingChangesetReference {
                baseRepository {
                  id
                  name
                }
              }

              ... on GitBranchChangesetDescription {
                baseRepository {
                  id
                  name
                }
              }
            }
          }
        }
	  }
    }
  }
}
`
