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
	"github.com/sourcegraph/sourcegraph/internal/db"
	"github.com/sourcegraph/sourcegraph/internal/db/dbconn"
	"github.com/sourcegraph/sourcegraph/internal/db/dbtesting"
)

func TestCampaignConnectionResolver(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	ctx := backend.WithAuthzBypass(context.Background())
	dbtesting.SetupGlobalTestDB(t)

	userID := insertTestUser(t, dbconn.Global, "campaign-connection-resolver", true)

	store := ee.NewStore(dbconn.Global)
	rstore := repos.NewDBStore(dbconn.Global, sql.TxOptions{})

	repo := newGitHubTestRepo("github.com/sourcegraph/sourcegraph", newGitHubExternalService(t, rstore))
	if err := rstore.InsertRepos(ctx, repo); err != nil {
		t.Fatal(err)
	}

	spec1 := &campaigns.CampaignSpec{
		NamespaceUserID: userID,
		UserID:          userID,
	}
	if err := store.CreateCampaignSpec(ctx, spec1); err != nil {
		t.Fatal(err)
	}
	spec2 := &campaigns.CampaignSpec{
		NamespaceUserID: userID,
		UserID:          userID,
	}
	if err := store.CreateCampaignSpec(ctx, spec2); err != nil {
		t.Fatal(err)
	}

	campaign1 := &campaigns.Campaign{
		Name:             "my-unique-name",
		NamespaceUserID:  userID,
		InitialApplierID: userID,
		LastApplierID:    userID,
		LastAppliedAt:    time.Now(),
		CampaignSpecID:   spec1.ID,
	}
	if err := store.CreateCampaign(ctx, campaign1); err != nil {
		t.Fatal(err)
	}
	campaign2 := &campaigns.Campaign{
		Name:             "my-other-unique-name",
		NamespaceUserID:  userID,
		InitialApplierID: userID,
		LastApplierID:    userID,
		LastAppliedAt:    time.Now(),
		CampaignSpecID:   spec2.ID,
	}
	if err := store.CreateCampaign(ctx, campaign2); err != nil {
		t.Fatal(err)
	}

	s, err := graphqlbackend.NewSchema(&Resolver{store: store}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Campaigns are returned in reverse order.
	nodes := []apitest.Campaign{
		{
			ID: string(marshalCampaignID(campaign2.ID)),
		},
		{
			ID: string(marshalCampaignID(campaign1.ID)),
		},
	}

	tests := []struct {
		firstParam      int
		wantHasNextPage bool
		wantTotalCount  int
		wantNodes       []apitest.Campaign
	}{
		{firstParam: 1, wantHasNextPage: true, wantTotalCount: 2, wantNodes: nodes[:1]},
		{firstParam: 2, wantHasNextPage: false, wantTotalCount: 2, wantNodes: nodes},
		{firstParam: 3, wantHasNextPage: false, wantTotalCount: 2, wantNodes: nodes},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("first=%d", tc.firstParam), func(t *testing.T) {
			input := map[string]interface{}{"first": int64(tc.firstParam)}
			var response struct{ Campaigns apitest.CampaignConnection }
			apitest.MustExec(actor.WithActor(context.Background(), actor.FromUser(userID)), t, s, input, &response, queryCampaignsConnection)

			wantConnection := apitest.CampaignConnection{
				TotalCount: tc.wantTotalCount,
				PageInfo: apitest.PageInfo{
					HasNextPage: tc.wantHasNextPage,
					// We don't test on the cursor here.
					EndCursor: response.Campaigns.PageInfo.EndCursor,
				},
				Nodes: tc.wantNodes,
			}

			if diff := cmp.Diff(wantConnection, response.Campaigns); diff != "" {
				t.Fatalf("wrong campaigns response (-want +got):\n%s", diff)
			}
		})
	}

	t.Run("Cursor based pagination", func(t *testing.T) {
		var endCursor *string
		for i := range nodes {
			input := map[string]interface{}{"first": 1}
			if endCursor != nil {
				input["after"] = *endCursor
			}
			wantHasNextPage := i != len(nodes)-1

			var response struct{ Campaigns apitest.CampaignConnection }
			apitest.MustExec(ctx, t, s, input, &response, queryCampaignsConnection)

			if diff := cmp.Diff(1, len(response.Campaigns.Nodes)); diff != "" {
				t.Fatalf("unexpected number of nodes (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(len(nodes), response.Campaigns.TotalCount); diff != "" {
				t.Fatalf("unexpected total count (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(wantHasNextPage, response.Campaigns.PageInfo.HasNextPage); diff != "" {
				t.Fatalf("unexpected hasNextPage (-want +got):\n%s", diff)
			}

			endCursor = response.Campaigns.PageInfo.EndCursor
			if want, have := wantHasNextPage, endCursor != nil; have != want {
				t.Fatalf("unexpected endCursor existence. want=%t, have=%t", want, have)
			}
		}
	})
}

const queryCampaignsConnection = `
query($first: Int, $after: String) {
  campaigns(first: $first, after: $after) {
    totalCount
    pageInfo {
	  hasNextPage
	  endCursor
    }
    nodes {
      id
    }
  }
}
`

func TestCampaignsListing(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	ctx := context.Background()
	dbtesting.SetupGlobalTestDB(t)

	userID := insertTestUser(t, dbconn.Global, "campaigns-listing", true)
	actorCtx := actor.WithActor(ctx, actor.FromUser(userID))

	org, err := db.Orgs.Create(ctx, "org", nil)
	if err != nil {
		t.Fatal(err)
	}

	store := ee.NewStore(dbconn.Global)

	r := &Resolver{store: store}
	s, err := graphqlbackend.NewSchema(r, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	createCampaignSpec := func(t *testing.T, spec *campaigns.CampaignSpec) {
		t.Helper()

		spec.UserID = userID
		spec.NamespaceUserID = userID
		if err := store.CreateCampaignSpec(ctx, spec); err != nil {
			t.Fatal(err)
		}
	}

	createCampaign := func(t *testing.T, c *campaigns.Campaign) {
		t.Helper()

		c.Name = "n"
		c.InitialApplierID = userID
		if err := store.CreateCampaign(ctx, c); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("listing a users campaigns", func(t *testing.T) {
		spec := &campaigns.CampaignSpec{}
		createCampaignSpec(t, spec)

		campaign := &campaigns.Campaign{
			NamespaceUserID: userID,
			CampaignSpecID:  spec.ID,
			LastApplierID:   userID,
			LastAppliedAt:   time.Now(),
		}
		createCampaign(t, campaign)

		userAPIID := string(graphqlbackend.MarshalUserID(userID))
		input := map[string]interface{}{"node": userAPIID}

		var response struct{ Node apitest.User }
		apitest.MustExec(actorCtx, t, s, input, &response, listNamespacesCampaigns)

		want := apitest.User{
			ID: userAPIID,
			Campaigns: apitest.CampaignConnection{
				TotalCount: 1,
				Nodes: []apitest.Campaign{
					{ID: string(marshalCampaignID(campaign.ID))},
				},
			},
		}

		if diff := cmp.Diff(want, response.Node); diff != "" {
			t.Fatalf("wrong campaign response (-want +got):\n%s", diff)
		}
	})

	t.Run("listing an orgs campaigns", func(t *testing.T) {
		spec := &campaigns.CampaignSpec{}
		createCampaignSpec(t, spec)

		campaign := &campaigns.Campaign{
			NamespaceOrgID: org.ID,
			CampaignSpecID: spec.ID,
			LastApplierID:  userID,
			LastAppliedAt:  time.Now(),
		}
		createCampaign(t, campaign)

		orgAPIID := string(graphqlbackend.MarshalOrgID(org.ID))
		input := map[string]interface{}{"node": orgAPIID}

		var response struct{ Node apitest.Org }
		apitest.MustExec(actorCtx, t, s, input, &response, listNamespacesCampaigns)

		want := apitest.Org{
			ID: orgAPIID,
			Campaigns: apitest.CampaignConnection{
				TotalCount: 1,
				Nodes: []apitest.Campaign{
					{ID: string(marshalCampaignID(campaign.ID))},
				},
			},
		}

		if diff := cmp.Diff(want, response.Node); diff != "" {
			t.Fatalf("wrong campaign response (-want +got):\n%s", diff)
		}
	})
}

const listNamespacesCampaigns = `
query($node: ID!) {
  node(id: $node) {
    ... on User {
      id
      campaigns {
        totalCount
        nodes {
          id
        }
      }
    }

    ... on Org {
      id
      campaigns {
        totalCount
        nodes {
          id
        }
      }
    }
  }
}
`
