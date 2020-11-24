package background

import (
	"context"
	"database/sql"

	"github.com/sourcegraph/sourcegraph/cmd/repo-updater/repos"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/campaigns"
	"github.com/sourcegraph/sourcegraph/internal/gitserver"
	"github.com/sourcegraph/sourcegraph/internal/goroutine"
	"github.com/sourcegraph/sourcegraph/internal/httpcli"
)

func StartBackgroundJobs(ctx context.Context, db *sql.DB, campaignsStore *campaigns.Store, repoStore repos.Store, cf *httpcli.Factory) {
	sourcer := repos.NewSourcer(cf)

	metrics := newMetrics()

	routines := []goroutine.BackgroundRoutine{
		newWorker(ctx, campaignsStore, gitserver.DefaultClient, sourcer, metrics),
		newWorkerResetter(campaignsStore, metrics),
		newSpecExpireWorker(ctx, campaignsStore),
		newHistoryMigrator(ctx, campaignsStore),
	}

	go goroutine.MonitorBackgroundRoutines(ctx, routines...)
}
