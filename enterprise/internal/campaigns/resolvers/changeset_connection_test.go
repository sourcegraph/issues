package resolvers

import (
	"context"
	"database/sql"
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
	"github.com/sourcegraph/sourcegraph/internal/actor"
	"github.com/sourcegraph/sourcegraph/internal/campaigns"
	"github.com/sourcegraph/sourcegraph/internal/db/dbconn"
	"github.com/sourcegraph/sourcegraph/internal/db/dbtesting"
)

func TestChangesetConnectionResolver(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	ctx := backend.WithAuthzBypass(context.Background())
	dbtesting.SetupGlobalTestDB(t)

	userID := insertTestUser(t, dbconn.Global, "changeset-connection-resolver", true)

	store := ee.NewStore(dbconn.Global)
	rstore := repos.NewDBStore(dbconn.Global, sql.TxOptions{})

	repo := newGitHubTestRepo("github.com/sourcegraph/sourcegraph", newGitHubExternalService(t, rstore))
	inaccessibleRepo := newGitHubTestRepo("github.com/sourcegraph/private", newGitHubExternalService(t, rstore))
	if err := rstore.InsertRepos(ctx, repo, inaccessibleRepo); err != nil {
		t.Fatal(err)
	}
	ct.AuthzFilterRepos(t, inaccessibleRepo.ID)

	spec := &campaigns.CampaignSpec{
		NamespaceUserID: userID,
		UserID:          userID,
	}
	if err := store.CreateCampaignSpec(ctx, spec); err != nil {
		t.Fatal(err)
	}

	campaign := &campaigns.Campaign{
		Name:             "my-unique-name",
		NamespaceUserID:  userID,
		InitialApplierID: userID,
		LastApplierID:    userID,
		LastAppliedAt:    time.Now(),
		CampaignSpecID:   spec.ID,
	}
	if err := store.CreateCampaign(ctx, campaign); err != nil {
		t.Fatal(err)
	}

	changeset1 := createChangeset(t, ctx, store, testChangesetOpts{
		repo:                repo.ID,
		externalServiceType: "github",
		publicationState:    campaigns.ChangesetPublicationStateUnpublished,
		externalReviewState: campaigns.ChangesetReviewStatePending,
		ownedByCampaign:     campaign.ID,
		campaign:            campaign.ID,
	})

	changeset2 := createChangeset(t, ctx, store, testChangesetOpts{
		repo:                repo.ID,
		externalServiceType: "github",
		externalID:          "12345",
		externalBranch:      "open-pr",
		externalState:       campaigns.ChangesetExternalStateOpen,
		publicationState:    campaigns.ChangesetPublicationStatePublished,
		externalReviewState: campaigns.ChangesetReviewStatePending,
		ownedByCampaign:     campaign.ID,
		campaign:            campaign.ID,
	})

	changeset3 := createChangeset(t, ctx, store, testChangesetOpts{
		repo:                repo.ID,
		externalServiceType: "github",
		externalID:          "56789",
		externalBranch:      "merged-pr",
		externalState:       campaigns.ChangesetExternalStateMerged,
		publicationState:    campaigns.ChangesetPublicationStatePublished,
		externalReviewState: campaigns.ChangesetReviewStatePending,
		ownedByCampaign:     campaign.ID,
		campaign:            campaign.ID,
	})
	changeset4 := createChangeset(t, ctx, store, testChangesetOpts{
		repo:                inaccessibleRepo.ID,
		externalServiceType: "github",
		externalID:          "987651",
		externalBranch:      "open-hidden-pr",
		externalState:       campaigns.ChangesetExternalStateOpen,
		publicationState:    campaigns.ChangesetPublicationStatePublished,
		externalReviewState: campaigns.ChangesetReviewStatePending,
		ownedByCampaign:     campaign.ID,
		campaign:            campaign.ID,
	})

	addChangeset(t, ctx, store, campaign, changeset1.ID)
	addChangeset(t, ctx, store, campaign, changeset2.ID)
	addChangeset(t, ctx, store, campaign, changeset3.ID)
	addChangeset(t, ctx, store, campaign, changeset4.ID)

	s, err := graphqlbackend.NewSchema(&Resolver{store: store}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	campaignAPIID := string(marshalCampaignID(campaign.ID))
	nodes := []apitest.Changeset{
		{
			Typename:   "ExternalChangeset",
			ID:         string(marshalChangesetID(changeset1.ID)),
			Repository: apitest.Repository{Name: repo.Name},
		},
		{
			Typename:   "ExternalChangeset",
			ID:         string(marshalChangesetID(changeset2.ID)),
			Repository: apitest.Repository{Name: repo.Name},
		},
		{
			Typename:   "ExternalChangeset",
			ID:         string(marshalChangesetID(changeset3.ID)),
			Repository: apitest.Repository{Name: repo.Name},
		},
		{
			Typename: "HiddenExternalChangeset",
			ID:       string(marshalChangesetID(changeset4.ID)),
		},
	}

	tests := []struct {
		firstParam      int
		useUnsafeOpts   bool
		wantHasNextPage bool
		wantEndCursor   string
		wantTotalCount  int
		wantOpen        int
		wantNodes       []apitest.Changeset
	}{
		{firstParam: 1, wantHasNextPage: true, wantEndCursor: "1", wantTotalCount: 4, wantOpen: 2, wantNodes: nodes[:1]},
		{firstParam: 2, wantHasNextPage: true, wantEndCursor: "2", wantTotalCount: 4, wantOpen: 2, wantNodes: nodes[:2]},
		{firstParam: 3, wantHasNextPage: true, wantEndCursor: "3", wantTotalCount: 4, wantOpen: 2, wantNodes: nodes[:3]},
		{firstParam: 4, wantHasNextPage: false, wantTotalCount: 4, wantOpen: 2, wantNodes: nodes[:4]},
		// Expect only 3 changesets to be returned when an unsafe filter is applied.
		{firstParam: 1, useUnsafeOpts: true, wantEndCursor: "1", wantHasNextPage: true, wantTotalCount: 3, wantOpen: 1, wantNodes: nodes[:1]},
		{firstParam: 2, useUnsafeOpts: true, wantEndCursor: "2", wantHasNextPage: true, wantTotalCount: 3, wantOpen: 1, wantNodes: nodes[:2]},
		{firstParam: 3, useUnsafeOpts: true, wantHasNextPage: false, wantTotalCount: 3, wantOpen: 1, wantNodes: nodes[:3]},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("Unsafe opts %t, first %d", tc.useUnsafeOpts, tc.firstParam), func(t *testing.T) {
			input := map[string]interface{}{"campaign": campaignAPIID, "first": int64(tc.firstParam)}
			if tc.useUnsafeOpts {
				input["reviewState"] = campaigns.ChangesetReviewStatePending
			}
			var response struct{ Node apitest.Campaign }
			apitest.MustExec(actor.WithActor(context.Background(), actor.FromUser(userID)), t, s, input, &response, queryChangesetConnection)

			var wantEndCursor *string
			if tc.wantEndCursor != "" {
				wantEndCursor = &tc.wantEndCursor
			}

			wantChangesets := apitest.ChangesetConnection{
				Stats: apitest.ChangesetConnectionStats{
					Unpublished: 1,
					Draft:       0,
					Open:        tc.wantOpen,
					Merged:      1,
					Closed:      0,
					Deleted:     0,
					Total:       tc.wantTotalCount,
				},
				TotalCount: tc.wantTotalCount,
				PageInfo: apitest.PageInfo{
					EndCursor:   wantEndCursor,
					HasNextPage: tc.wantHasNextPage,
				},
				Nodes: tc.wantNodes,
			}

			if diff := cmp.Diff(wantChangesets, response.Node.Changesets); diff != "" {
				t.Fatalf("wrong changesets response (-want +got):\n%s", diff)
			}
		})
	}

	var endCursor *string
	for i := range nodes {
		input := map[string]interface{}{"campaign": campaignAPIID, "first": 1}
		if endCursor != nil {
			input["after"] = *endCursor
		}
		wantHasNextPage := i != len(nodes)-1

		var response struct{ Node apitest.Campaign }
		apitest.MustExec(actor.WithActor(context.Background(), actor.FromUser(userID)), t, s, input, &response, queryChangesetConnection)

		changesets := response.Node.Changesets
		if diff := cmp.Diff(1, len(changesets.Nodes)); diff != "" {
			t.Fatalf("unexpected number of nodes (-want +got):\n%s", diff)
		}

		if diff := cmp.Diff(len(nodes), changesets.TotalCount); diff != "" {
			t.Fatalf("unexpected total count (-want +got):\n%s", diff)
		}

		if diff := cmp.Diff(wantHasNextPage, changesets.PageInfo.HasNextPage); diff != "" {
			t.Fatalf("unexpected hasNextPage (-want +got):\n%s", diff)
		}

		endCursor = changesets.PageInfo.EndCursor
		if want, have := wantHasNextPage, endCursor != nil; have != want {
			t.Fatalf("unexpected endCursor existence. want=%t, have=%t", want, have)
		}
	}
}

const queryChangesetConnection = `
query($campaign: ID!, $first: Int, $after: String, $reviewState: ChangesetReviewState){
  node(id: $campaign) {
    ... on Campaign {
      changesets(first: $first, after: $after, reviewState: $reviewState) {
        totalCount
        stats { unpublished, draft, open, merged, closed, deleted, total }
        nodes {
          __typename

          ... on ExternalChangeset {
            id
            repository { name }
            nextSyncAt
          }
          ... on HiddenExternalChangeset {
            id
            nextSyncAt
          }
        }
        pageInfo {
          endCursor
          hasNextPage
        }
      }
    }
  }
}
`
