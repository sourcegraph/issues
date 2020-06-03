package resolvers

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/authz"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/db"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/cmd/repo-updater/repos"
	ee "github.com/sourcegraph/sourcegraph/enterprise/internal/campaigns"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/campaigns/resolvers/apitest"
	"github.com/sourcegraph/sourcegraph/internal/actor"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/campaigns"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/db/dbconn"
	"github.com/sourcegraph/sourcegraph/internal/db/dbtesting"
	"github.com/sourcegraph/sourcegraph/internal/vcs/git"
	"github.com/sourcegraph/sourcegraph/schema"
)

func TestRepositoryPermissions(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	now := time.Now().UTC().Truncate(time.Microsecond)

	// We need to enable read access so that non-site-admin users can access
	// the API and we can check for their admin rights.
	// This can be removed once we enable campaigns for all users and only
	// check for permissions.
	readAccessEnabled := true
	conf.Mock(&conf.Unified{SiteConfiguration: schema.SiteConfiguration{
		CampaignsReadAccessEnabled: &readAccessEnabled,
	}})
	defer conf.Mock(nil)

	dbtesting.SetupGlobalTestDB(t)

	store := ee.NewStore(dbconn.Global)
	sr := &Resolver{store: store}
	s, err := graphqlbackend.NewSchema(sr, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	testRev := "b69072d5f687b31b9f6ae3ceafdc24c259c4b9ec"
	mockBackendCommit(t, testRev)

	// Global test data that we reuse in every test
	// adminID := insertTestUser(t, dbconn.Global, "perm-level-admin", true)
	userID := insertTestUser(t, dbconn.Global, "perm-level-user", false)
	// otherUserID := insertTestUser(t, dbconn.Global, "perm-level-user-2", false)

	reposStore := repos.NewDBStore(dbconn.Global, sql.TxOptions{})

	// Create 4 repositories
	repos := make([]*repos.Repo, 0, 4)
	for i := 0; i < cap(repos); i++ {
		name := fmt.Sprintf("github.com/sourcegraph/repo-%d", i)
		r := newGitHubTestRepo(name, i)
		if err := reposStore.UpsertRepos(ctx, r); err != nil {
			t.Fatal(err)
		}
		repos = append(repos, r)
	}

	// Create 2 changesets for 2 repositories
	changesets := make([]*campaigns.Changeset, 0, 2)
	changesetIDs := make([]int64, 0, cap(changesets))
	for _, r := range repos[0:2] {
		c := &campaigns.Changeset{
			RepoID:              r.ID,
			ExternalServiceType: "github",
			ExternalID:          fmt.Sprintf("external-%d", r.ID),
		}
		if err := store.CreateChangesets(ctx, c); err != nil {
			t.Fatal(err)
		}
		changesets = append(changesets, c)
		changesetIDs = append(changesetIDs, c.ID)
	}

	patchSet := &campaigns.PatchSet{UserID: userID}
	if err := store.CreatePatchSet(ctx, patchSet); err != nil {
		t.Fatal(err)
	}

	// Create 2 patches for the other repositories
	patches := make([]*campaigns.Patch, 0, 2)
	for _, r := range repos[2:4] {
		p := &campaigns.Patch{
			PatchSetID: patchSet.ID,
			RepoID:     r.ID,
			Rev:        api.CommitID(testRev),
			BaseRef:    "refs/heads/master",
			Diff:       "+ foo - bar",
		}
		if err := store.CreatePatch(ctx, p); err != nil {
			t.Fatal(err)
		}
		patches = append(patches, p)
	}

	campaign := &campaigns.Campaign{
		PatchSetID:      patchSet.ID,
		Name:            "my campaign",
		AuthorID:        userID,
		NamespaceUserID: userID,
		// We attach the two changesets to the campaign
		// Note: we are mixing a "manual" and "non-manual" campaign here, but
		// that shouldn't matter for the purposes of this test.
		ChangesetIDs: changesetIDs,
	}
	if err := store.CreateCampaign(ctx, campaign); err != nil {
		t.Fatal(err)
	}
	for _, c := range changesets {
		c.CampaignIDs = []int64{campaign.ID}
	}
	if err := store.UpdateChangesets(ctx, changesets...); err != nil {
		t.Fatal(err)
	}

	// Create 2 failed ChangesetJobs for the patchess to produce error messages
	// on the campaign.
	changesetJobs := make([]*campaigns.ChangesetJob, 0, 2)
	for _, p := range patches {
		job := &campaigns.ChangesetJob{
			CampaignID: campaign.ID,
			PatchID:    p.ID,
			Error:      fmt.Sprintf("error patch %d", p.ID),
			StartedAt:  now,
			FinishedAt: now,
		}
		if err := store.CreateChangesetJob(ctx, job); err != nil {
			t.Fatal(err)
		}

		changesetJobs = append(changesetJobs, job)
	}

	type want struct {
		patchTypes     map[string]int
		changesetTypes map[string]int
		errors         []string
	}

	testCampaign := func(t *testing.T, ctx context.Context, id int64, w want) {
		t.Helper()

		var response struct{ Node apitest.Campaign }
		query := fmt.Sprintf(queryCampaignPermLevels, campaigns.MarshalCampaignID(id))

		apitest.MustExec(ctx, t, s, nil, &response, query)

		if have, want := response.Node.ID, string(campaigns.MarshalCampaignID(id)); have != want {
			t.Fatalf("campaign id is wrong. have %q, want %q", have, want)
		}

		if diff := cmp.Diff(response.Node.Status.Errors, w.errors); diff != "" {
			t.Fatalf("unexpected status errors (-want +got):\n%s", diff)
		}

		changesetTypes := map[string]int{}
		for _, c := range response.Node.Changesets.Nodes {
			changesetTypes[c.Typename]++
		}
		if diff := cmp.Diff(w.changesetTypes, changesetTypes); diff != "" {
			t.Fatalf("unexpected changesettypes (-want +got):\n%s", diff)
		}

		patchTypes := map[string]int{}
		for _, p := range response.Node.Patches.Nodes {
			patchTypes[p.Typename]++
		}
		if diff := cmp.Diff(w.patchTypes, patchTypes); diff != "" {
			t.Fatalf("unexpected patch types (-want +got):\n%s", diff)
		}
	}

	// Query campaign and check that we get all changesets and all patches
	userCtx := actor.WithActor(ctx, actor.FromUser(userID))
	testCampaign(t, userCtx, campaign.ID, want{
		changesetTypes: map[string]int{"ExternalChangeset": 2},
		errors: []string{
			fmt.Sprintf("error patch %d", patches[0].ID),
			fmt.Sprintf("error patch %d", patches[1].ID),
		},
		patchTypes: map[string]int{"Patch": 2},
	})

	// Now we add the authzFilter and filter out 2 repositories
	filteredRepoIDs := map[api.RepoID]bool{
		patches[0].RepoID:    true,
		changesets[0].RepoID: true,
	}

	db.MockAuthzFilter = func(ctx context.Context, repos []*types.Repo, p authz.Perms) ([]*types.Repo, error) {
		var filtered []*types.Repo
		for _, r := range repos {
			if _, ok := filteredRepoIDs[r.ID]; ok {
				continue
			}
			filtered = append(filtered, r)
		}
		return filtered, nil
	}
	defer func() { db.MockAuthzFilter = nil }()

	// Send query again and check that for each filtered repository we get a
	// HiddenChangeset/HiddenPatch and that errors are filtered out
	testCampaign(t, userCtx, campaign.ID, want{
		changesetTypes: map[string]int{
			"ExternalChangeset":       1,
			"HiddenExternalChangeset": 1,
		},
		errors: []string{
			// patches[0] is filtered out
			fmt.Sprintf("error patch %d", patches[1].ID),
		},
		patchTypes: map[string]int{
			"Patch":       1,
			"HiddenPatch": 1,
		},
	})

	// TODO: Test that the diffStat on `patchset` doesn't include the filtered patches diff stats
	// TODO: Test that the diffStat on `campaign` doesn't include the filtered changesets diff stats
	// TODO: Test that ChangesetByID and PatchByID don't return the filtered out changesets/patches
}

const queryCampaignPermLevels = `
query {
  node(id: %q) {
    ... on Campaign {
      id

	  status {
	    state
		errors
	  }

      changesets(first: 100) {
        nodes {
          __typename
          ... on HiddenExternalChangeset {
            id
          }
          ... on ExternalChangeset {
            id
            repository {
              id
              name
            }
          }
        }
      }

      patches(first: 100) {
        nodes {
          __typename
          ... on HiddenPatch {
            id
          }
          ... on Patch {
            id
            repository {
              id
              name
            }
          }
        }
      }
    }
  }
}
`

func mockBackendCommit(t *testing.T, rev string) {
	t.Helper()

	backend.Mocks.Repos.ResolveRev = func(_ context.Context, _ *types.Repo, rev string) (api.CommitID, error) {
		if rev != rev {
			t.Fatalf("ResolveRev received wrong rev: %q", rev)
		}
		return api.CommitID(rev), nil
	}
	t.Cleanup(func() { backend.Mocks.Repos.ResolveRev = nil })

	backend.Mocks.Repos.GetCommit = func(_ context.Context, _ *types.Repo, id api.CommitID) (*git.Commit, error) {
		if string(id) != rev {
			t.Fatalf("GetCommit received wrong ID: %s", id)
		}
		return &git.Commit{ID: id}, nil
	}
	t.Cleanup(func() { backend.Mocks.Repos.GetCommit = nil })
}
