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
	"github.com/sourcegraph/sourcegraph/internal/actor"
	"github.com/sourcegraph/sourcegraph/internal/campaigns"
	"github.com/sourcegraph/sourcegraph/internal/db/dbconn"
	"github.com/sourcegraph/sourcegraph/internal/db/dbtesting"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/github"
)

func TestChangesetEventConnectionResolver(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	ctx := backend.WithAuthzBypass(context.Background())
	dbtesting.SetupGlobalTestDB(t)

	userID := insertTestUser(t, dbconn.Global, "changeset-event-connection-resolver", true)

	now := time.Now().UTC().Truncate(time.Microsecond)
	clock := func() time.Time {
		return now.UTC().Truncate(time.Microsecond)
	}
	store := ee.NewStoreWithClock(dbconn.Global, clock)
	rstore := repos.NewDBStore(dbconn.Global, sql.TxOptions{})

	repo := newGitHubTestRepo("github.com/sourcegraph/sourcegraph", newGitHubExternalService(t, rstore))
	if err := rstore.InsertRepos(ctx, repo); err != nil {
		t.Fatal(err)
	}

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

	changeset := createChangeset(t, ctx, store, testChangesetOpts{
		repo:                repo.ID,
		externalServiceType: "github",
		publicationState:    campaigns.ChangesetPublicationStateUnpublished,
		externalReviewState: campaigns.ChangesetReviewStatePending,
		ownedByCampaign:     campaign.ID,
		campaign:            campaign.ID,
		metadata: &github.PullRequest{
			TimelineItems: []github.TimelineItem{
				{Type: "PullRequestCommit", Item: &github.PullRequestCommit{
					Commit: github.Commit{
						OID: "d34db33f",
					},
				}},
				{Type: "LabeledEvent", Item: &github.LabelEvent{
					Label: github.Label{
						ID:    "label-event",
						Name:  "cool-label",
						Color: "blue",
					},
				}},
			},
		},
	})

	addChangeset(t, ctx, store, campaign, changeset.ID)

	s, err := graphqlbackend.NewSchema(&Resolver{store: store}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	changesetAPIID := string(marshalChangesetID(changeset.ID))
	nodes := []apitest.ChangesetEvent{
		{
			ID:        string(marshalChangesetEventID(1)),
			Changeset: struct{ ID string }{ID: changesetAPIID},
			CreatedAt: marshalDateTime(t, now),
		},
		{
			ID:        string(marshalChangesetEventID(2)),
			Changeset: struct{ ID string }{ID: changesetAPIID},
			CreatedAt: marshalDateTime(t, now),
		},
	}

	tests := []struct {
		firstParam      int
		wantHasNextPage bool
		wantTotalCount  int
		wantNodes       []apitest.ChangesetEvent
	}{
		{firstParam: 1, wantHasNextPage: true, wantTotalCount: 2, wantNodes: nodes[:1]},
		{firstParam: 2, wantHasNextPage: false, wantTotalCount: 2, wantNodes: nodes},
		{firstParam: 3, wantHasNextPage: false, wantTotalCount: 2, wantNodes: nodes},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("first=%d", tc.firstParam), func(t *testing.T) {
			input := map[string]interface{}{"changeset": changesetAPIID, "first": int64(tc.firstParam)}
			var response struct{ Node apitest.Changeset }
			apitest.MustExec(actor.WithActor(context.Background(), actor.FromUser(userID)), t, s, input, &response, queryChangesetEventConnection)

			wantEvents := apitest.ChangesetEventConnection{
				TotalCount: tc.wantTotalCount,
				PageInfo: apitest.PageInfo{
					HasNextPage: tc.wantHasNextPage,
					// This test doesn't check on the cursors, the below test does that.
					EndCursor: response.Node.Events.PageInfo.EndCursor,
				},
				Nodes: tc.wantNodes,
			}

			if diff := cmp.Diff(wantEvents, response.Node.Events); diff != "" {
				t.Fatalf("wrong changesets response (-want +got):\n%s", diff)
			}
		})
	}

	var endCursor *string
	for i := range nodes {
		input := map[string]interface{}{"changeset": changesetAPIID, "first": 1}
		if endCursor != nil {
			input["after"] = *endCursor
		}
		wantHasNextPage := i != len(nodes)-1

		var response struct{ Node apitest.Changeset }
		apitest.MustExec(ctx, t, s, input, &response, queryChangesetEventConnection)

		events := response.Node.Events
		if diff := cmp.Diff(1, len(events.Nodes)); diff != "" {
			t.Fatalf("unexpected number of nodes (-want +got):\n%s", diff)
		}

		if diff := cmp.Diff(len(nodes), events.TotalCount); diff != "" {
			t.Fatalf("unexpected total count (-want +got):\n%s", diff)
		}

		if diff := cmp.Diff(wantHasNextPage, events.PageInfo.HasNextPage); diff != "" {
			t.Fatalf("unexpected hasNextPage (-want +got):\n%s", diff)
		}

		endCursor = events.PageInfo.EndCursor
		if want, have := wantHasNextPage, endCursor != nil; have != want {
			t.Fatalf("unexpected endCursor existence. want=%t, have=%t", want, have)
		}
	}
}

const queryChangesetEventConnection = `
query($changeset: ID!, $first: Int, $after: String){
  node(id: $changeset) {
    ... on ExternalChangeset {
      events(first: $first, after: $after) {
        totalCount
        pageInfo {
          hasNextPage
          endCursor
        }
        nodes {
         id
         createdAt
         changeset {
           id
         }
        }
      }
    }
  }
}
`
