package resolvers

import (
	"context"
	"database/sql"
	"os"
	"reflect"
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
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/campaigns"
	"github.com/sourcegraph/sourcegraph/internal/db/dbconn"
	"github.com/sourcegraph/sourcegraph/internal/db/dbtesting"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/httptestutil"
	"github.com/sourcegraph/sourcegraph/internal/rcache"
	"github.com/sourcegraph/sourcegraph/internal/repoupdater/protocol"
	"github.com/sourcegraph/sourcegraph/schema"
)

func TestChangesetCountsOverTimeResolver(t *testing.T) {
	counts := &ee.ChangesetCounts{
		Time:                 time.Now(),
		Total:                10,
		Merged:               9,
		Closed:               8,
		Open:                 7,
		OpenApproved:         6,
		OpenChangesRequested: 5,
		OpenPending:          4,
	}

	resolver := changesetCountsResolver{counts: counts}

	tests := []struct {
		name   string
		method func() int32
		want   int32
	}{
		{name: "Total", method: resolver.Total, want: counts.Total},
		{name: "Merged", method: resolver.Merged, want: counts.Merged},
		{name: "Closed", method: resolver.Closed, want: counts.Closed},
		{name: "Open", method: resolver.Open, want: counts.Open},
		{name: "OpenApproved", method: resolver.OpenApproved, want: counts.OpenApproved},
		{name: "OpenChangesRequested", method: resolver.OpenChangesRequested, want: counts.OpenChangesRequested},
		{name: "OpenPending", method: resolver.OpenPending, want: counts.OpenPending},
	}

	for _, tc := range tests {
		if have := tc.method(); have != tc.want {
			t.Errorf("resolver.%s wrong. want=%d, have=%d", tc.name, tc.want, have)
		}
	}
}

func TestChangesetCountsOverTimeIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	ctx := backend.WithAuthzBypass(context.Background())
	dbtesting.SetupGlobalTestDB(t)
	rcache.SetupForTest(t)

	cf, save := httptestutil.NewGitHubRecorderFactory(t, *update, "test-changeset-counts-over-time")
	defer save()

	userID := insertTestUser(t, dbconn.Global, "changeset-counts-over-time", false)

	repoStore := repos.NewDBStore(dbconn.Global, sql.TxOptions{})
	githubExtSvc := &repos.ExternalService{
		Kind:        extsvc.KindGitHub,
		DisplayName: "GitHub",
		Config: marshalJSON(t, &schema.GitHubConnection{
			Url:   "https://github.com",
			Token: os.Getenv("GITHUB_TOKEN"),
			Repos: []string{"sourcegraph/sourcegraph"},
		}),
	}

	err := repoStore.UpsertExternalServices(ctx, githubExtSvc)
	if err != nil {
		t.Fatal(t)
	}

	githubSrc, err := repos.NewGithubSource(githubExtSvc, cf)
	if err != nil {
		t.Fatal(t)
	}

	githubRepo, err := githubSrc.GetRepo(ctx, "sourcegraph/sourcegraph")
	if err != nil {
		t.Fatal(err)
	}

	err = repoStore.InsertRepos(ctx, githubRepo)
	if err != nil {
		t.Fatal(err)
	}

	mockState := ct.MockChangesetSyncState(&protocol.RepoInfo{
		Name: api.RepoName(githubRepo.Name),
		VCS:  protocol.VCSInfo{URL: githubRepo.URI},
	})
	defer mockState.Unmock()

	sourcer := repos.NewSourcer(cf)

	store := ee.NewStore(dbconn.Global)

	spec := &campaigns.CampaignSpec{
		NamespaceUserID: userID,
		UserID:          userID,
	}
	if err := store.CreateCampaignSpec(ctx, spec); err != nil {
		t.Fatal(err)
	}

	campaign := &campaigns.Campaign{
		Name:             "Test campaign",
		Description:      "Testing changeset counts",
		InitialApplierID: userID,
		NamespaceUserID:  userID,
		LastApplierID:    userID,
		LastAppliedAt:    time.Now(),
		CampaignSpecID:   spec.ID,
	}

	err = store.CreateCampaign(ctx, campaign)
	if err != nil {
		t.Fatal(err)
	}

	changesets := []*campaigns.Changeset{
		{
			RepoID:              githubRepo.ID,
			ExternalID:          "5834",
			ExternalServiceType: githubRepo.ExternalRepo.ServiceType,
			CampaignIDs:         []int64{campaign.ID},
			PublicationState:    campaigns.ChangesetPublicationStatePublished,
		},
		{
			RepoID:              githubRepo.ID,
			ExternalID:          "5849",
			ExternalServiceType: githubRepo.ExternalRepo.ServiceType,
			CampaignIDs:         []int64{campaign.ID},
			PublicationState:    campaigns.ChangesetPublicationStatePublished,
		},
	}

	for _, c := range changesets {
		if err = store.CreateChangeset(ctx, c); err != nil {
			t.Fatal(err)
		}

		campaign.ChangesetIDs = append(campaign.ChangesetIDs, c.ID)

		if err := ee.SyncChangeset(ctx, repoStore, store, sourcer, c); err != nil {
			t.Fatal(err)
		}
	}

	err = store.UpdateCampaign(ctx, campaign)
	if err != nil {
		t.Fatal(err)
	}

	s, err := graphqlbackend.NewSchema(&Resolver{store: store}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Date when PR #5834 was created: "2019-10-02T14:49:31Z"
	// We start exactly one day earlier
	// Date when PR #5849 was created: "2019-10-03T15:03:21Z"
	start := parseJSONTime(t, "2019-10-01T14:49:31Z")
	// Date when PR #5834 was merged:  "2019-10-07T13:13:45Z"
	// Date when PR #5849 was merged:  "2019-10-04T08:55:21Z"
	end := parseJSONTime(t, "2019-10-07T13:13:45Z")
	daysBeforeEnd := func(days int) time.Time {
		return end.AddDate(0, 0, -days)
	}

	input := map[string]interface{}{
		"campaign": string(marshalCampaignID(campaign.ID)),
		"from":     start,
		"to":       end,
	}

	var response struct{ Node apitest.Campaign }

	apitest.MustExec(actor.WithActor(context.Background(), actor.FromUser(userID)), t, s, input, &response, queryChangesetCountsConnection)

	wantCounts := []apitest.ChangesetCounts{
		{Date: marshalDateTime(t, daysBeforeEnd(5)), Total: 0, Open: 0, OpenPending: 0},
		{Date: marshalDateTime(t, daysBeforeEnd(4)), Total: 1, Draft: 1},
		{Date: marshalDateTime(t, daysBeforeEnd(3)), Total: 2, Open: 1, OpenPending: 1, Merged: 1},
		{Date: marshalDateTime(t, daysBeforeEnd(2)), Total: 2, Open: 1, OpenPending: 1, Merged: 1},
		{Date: marshalDateTime(t, daysBeforeEnd(1)), Total: 2, Open: 1, OpenPending: 1, Merged: 1},
		{Date: marshalDateTime(t, end), Total: 2, Merged: 2},
	}

	if !reflect.DeepEqual(response.Node.ChangesetCountsOverTime, wantCounts) {
		t.Errorf("wrong counts listed. diff=%s", cmp.Diff(response.Node.ChangesetCountsOverTime, wantCounts))
	}
}

const queryChangesetCountsConnection = `
query($campaign: ID!, $from: DateTime!, $to: DateTime!) {
  node(id: $campaign) {
    ... on Campaign {
	  changesetCountsOverTime(from: $from, to: $to) {
        date
        total
        merged
        draft
        closed
        open
        openApproved
        openChangesRequested
        openPending
      }
    }
  }
}
`
