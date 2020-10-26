package usagestats

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/cmd/repo-updater/repos"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/db"
	"github.com/sourcegraph/sourcegraph/internal/db/dbconn"
	"github.com/sourcegraph/sourcegraph/internal/db/dbtesting"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
)

func TestCampaignsUsageStatistics(t *testing.T) {
	ctx := context.Background()
	dbtesting.SetupGlobalTestDB(t)

	// Create stub repo.
	rstore := repos.NewDBStore(dbconn.Global, sql.TxOptions{})
	now := time.Now()
	svc := repos.ExternalService{
		Kind:        extsvc.KindGitHub,
		DisplayName: "Github - Test",
		Config:      `{"url": "https://github.com"}`,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := rstore.UpsertExternalServices(ctx, &svc); err != nil {
		t.Fatalf("failed to insert external services: %v", err)
	}
	repo := &repos.Repo{
		Name: "test/repo",
		ExternalRepo: api.ExternalRepoSpec{
			ID:          fmt.Sprintf("external-id-%d", svc.ID),
			ServiceType: extsvc.TypeGitHub,
			ServiceID:   "https://github.com/",
		},
		Sources: map[string]*repos.SourceInfo{
			svc.URN(): {
				ID:       svc.URN(),
				CloneURL: "https://secrettoken@test/repo",
			},
		},
	}
	if err := rstore.InsertRepos(ctx, repo); err != nil {
		t.Fatal(err)
	}

	// Create a user.
	user, err := db.Users.Create(ctx, db.NewUser{Username: "test"})
	if err != nil {
		t.Fatal(err)
	}

	// Create campaign specs 1, 2.
	_, err = dbconn.Global.Exec(`
		INSERT INTO campaign_specs
			(id, rand_id, raw_spec, namespace_user_id)
		VALUES
			(1, '123', '{}', $1),
			(2, '456', '{}', $1)
	`, user.ID)
	if err != nil {
		t.Fatal(err)
	}

	// Create campaigns 1, 2.
	_, err = dbconn.Global.Exec(`
		INSERT INTO campaigns
			(id, name, campaign_spec_id, last_applied_at, namespace_user_id)
		VALUES
			(1, 'test', 1, NOW(), $1),
			(2, 'test-2', 2, NOW(), $1)
	`, user.ID)
	if err != nil {
		t.Fatal(err)
	}

	// Create 4 changesets, 2 tracked, 2 created by a campaign. For each pair, one OPEN, one MERGED.
	_, err = dbconn.Global.Exec(`
		INSERT INTO changesets
			(id, repo_id, external_service_type, added_to_campaign, owned_by_campaign_id, external_state, publication_state)
		VALUES
			(1, $1, 'github', true, NULL, 'OPEN', 'PUBLISHED'),
			(2, $1, 'github', true, NULL, 'MERGED', 'PUBLISHED'),
			(3, $1, 'github', false, 1, 'OPEN', 'PUBLISHED'),
			(4, $1, 'github', false, 2, 'MERGED', 'PUBLISHED')
	`, repo.ID)
	if err != nil {
		t.Fatal(err)
	}
	have, err := GetCampaignsUsageStatistics(ctx)
	if err != nil {
		t.Fatal(err)
	}
	want := &types.CampaignsUsageStatistics{
		CampaignsCount:              2,
		ActionChangesetsCount:       2,
		ActionChangesetsMergedCount: 1,
		ManualChangesetsCount:       2,
		ManualChangesetsMergedCount: 1,
	}
	if diff := cmp.Diff(want, have); diff != "" {
		t.Fatal(diff)
	}
}
