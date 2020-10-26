package resolvers

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend"
	"github.com/sourcegraph/sourcegraph/cmd/repo-updater/repos"
	ee "github.com/sourcegraph/sourcegraph/enterprise/internal/campaigns"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/campaigns/resolvers/apitest"
	ct "github.com/sourcegraph/sourcegraph/enterprise/internal/campaigns/testing"
	"github.com/sourcegraph/sourcegraph/internal/campaigns"
	"github.com/sourcegraph/sourcegraph/internal/db/dbconn"
	"github.com/sourcegraph/sourcegraph/internal/db/dbtesting"
)

func TestChangesetSpecConnectionResolver(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	ctx := backend.WithAuthzBypass(context.Background())
	dbtesting.SetupGlobalTestDB(t)

	userID := insertTestUser(t, dbconn.Global, "changeset-spec-connection-resolver", false)

	store := ee.NewStore(dbconn.Global)

	campaignSpec := &campaigns.CampaignSpec{
		UserID:          userID,
		NamespaceUserID: userID,
	}
	if err := store.CreateCampaignSpec(ctx, campaignSpec); err != nil {
		t.Fatal(err)
	}

	reposStore := repos.NewDBStore(dbconn.Global, sql.TxOptions{})

	repos := make([]*repos.Repo, 0, 3)
	for i := 0; i < cap(repos); i++ {
		name := fmt.Sprintf("github.com/sourcegraph/repo-%d", i)
		r := newGitHubTestRepo(name, newGitHubExternalService(t, reposStore))
		if err := reposStore.InsertRepos(ctx, r); err != nil {
			t.Fatal(err)
		}
		repos = append(repos, r)
	}

	changesetSpecs := make([]*campaigns.ChangesetSpec, 0, len(repos))
	for _, r := range repos {
		repoID := graphqlbackend.MarshalRepositoryID(r.ID)
		s, err := campaigns.NewChangesetSpecFromRaw(ct.NewRawChangesetSpecGitBranch(repoID, "d34db33f"))
		if err != nil {
			t.Fatal(err)
		}
		s.CampaignSpecID = campaignSpec.ID
		s.UserID = userID
		s.RepoID = r.ID

		if err := store.CreateChangesetSpec(ctx, s); err != nil {
			t.Fatal(err)
		}

		changesetSpecs = append(changesetSpecs, s)
	}

	s, err := graphqlbackend.NewSchema(&Resolver{store: store}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	apiID := string(marshalCampaignSpecRandID(campaignSpec.RandID))

	tests := []struct {
		first int

		wantTotalCount  int
		wantHasNextPage bool
	}{
		{first: 1, wantTotalCount: 3, wantHasNextPage: true},
		{first: 2, wantTotalCount: 3, wantHasNextPage: true},
		{first: 3, wantTotalCount: 3, wantHasNextPage: false},
	}

	for _, tc := range tests {
		input := map[string]interface{}{"campaignSpec": apiID, "first": tc.first}
		var response struct{ Node apitest.CampaignSpec }
		apitest.MustExec(ctx, t, s, input, &response, queryChangesetSpecConnection)

		specs := response.Node.ChangesetSpecs
		if diff := cmp.Diff(tc.wantTotalCount, specs.TotalCount); diff != "" {
			t.Fatalf("first=%d, unexpected total count (-want +got):\n%s", tc.first, diff)
		}

		if diff := cmp.Diff(tc.wantHasNextPage, specs.PageInfo.HasNextPage); diff != "" {
			t.Fatalf("first=%d, unexpected hasNextPage (-want +got):\n%s", tc.first, diff)
		}
	}

	var endCursor *string
	for i := range changesetSpecs {
		input := map[string]interface{}{"campaignSpec": apiID, "first": 1}
		if endCursor != nil {
			input["after"] = *endCursor
		}
		wantHasNextPage := i != len(changesetSpecs)-1

		var response struct{ Node apitest.CampaignSpec }
		apitest.MustExec(ctx, t, s, input, &response, queryChangesetSpecConnection)

		specs := response.Node.ChangesetSpecs
		if diff := cmp.Diff(1, len(specs.Nodes)); diff != "" {
			t.Fatalf("unexpected number of nodes (-want +got):\n%s", diff)
		}

		if diff := cmp.Diff(len(changesetSpecs), specs.TotalCount); diff != "" {
			t.Fatalf("unexpected total count (-want +got):\n%s", diff)
		}

		if diff := cmp.Diff(wantHasNextPage, specs.PageInfo.HasNextPage); diff != "" {
			t.Fatalf("unexpected hasNextPage (-want +got):\n%s", diff)
		}

		endCursor = specs.PageInfo.EndCursor
		if want, have := wantHasNextPage, endCursor != nil; have != want {
			t.Fatalf("unexpected endCursor existence. want=%t, have=%t", want, have)
		}
	}
}

const queryChangesetSpecConnection = `
query($campaignSpec: ID!, $first: Int!, $after: String) {
  node(id: $campaignSpec) {
    __typename

    ... on CampaignSpec {
      id

      changesetSpecs(first: $first, after: $after) {
        totalCount
        pageInfo { hasNextPage, endCursor }

        nodes {
          __typename
          ... on HiddenChangesetSpec { id }
          ... on VisibleChangesetSpec { id }
        }
      }
    }
  }
}
`
