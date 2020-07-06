package main

import (
	"context"
	"database/sql"
	"time"

	"github.com/inconshreveable/log15"
	ossAuthz "github.com/sourcegraph/sourcegraph/cmd/frontend/authz"
	ossDB "github.com/sourcegraph/sourcegraph/cmd/frontend/db"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/globals"
	"github.com/sourcegraph/sourcegraph/cmd/repo-updater/repos"
	"github.com/sourcegraph/sourcegraph/cmd/repo-updater/repoupdater"
	"github.com/sourcegraph/sourcegraph/cmd/repo-updater/shared"
	frontendAuthz "github.com/sourcegraph/sourcegraph/enterprise/cmd/frontend/authz"
	frontendDB "github.com/sourcegraph/sourcegraph/enterprise/cmd/frontend/db"
	"github.com/sourcegraph/sourcegraph/enterprise/cmd/repo-updater/authz"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/campaigns"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/db/dbconn"
	"github.com/sourcegraph/sourcegraph/internal/db/dbutil"
	"github.com/sourcegraph/sourcegraph/internal/debugserver"
	"github.com/sourcegraph/sourcegraph/internal/gitserver"
	"github.com/sourcegraph/sourcegraph/internal/httpcli"
	"github.com/sourcegraph/sourcegraph/internal/ratelimit"
	"github.com/sourcegraph/sourcegraph/internal/servicecmdutil"
)

func main() {
	servicecmdutil.Init()
	shared.Main(enterpriseInit)
}

func enterpriseInit(
	db *sql.DB,
	repoStore repos.Store,
	cf *httpcli.Factory,
	server *repoupdater.Server,
) (debugDumpers []debugserver.Dumper) {
	ctx := context.Background()
	campaignsStore := campaigns.NewStore(db)

	syncRegistry := campaigns.NewSyncRegistry(ctx, campaignsStore, repoStore, cf)
	if server != nil {
		server.ChangesetSyncRegistry = syncRegistry
	}

	clock := func() time.Time {
		return time.Now().UTC().Truncate(time.Microsecond)
	}

	sourcer := repos.NewSourcer(cf)
	go campaigns.RunWorkers(ctx, campaignsStore, clock, gitserver.DefaultClient, sourcer, 5*time.Second)

	// Set up expired patch set deletion
	go func() {
		for {
			err := campaignsStore.DeleteExpiredPatchSets(ctx)
			if err != nil {
				log15.Error("DeleteExpiredPatchSets", "error", err)
			}
			time.Sleep(2 * time.Minute)
		}
	}()

	// TODO(jchen): This is an unfortunate compromise to not rewrite ossDB.ExternalServices for now.
	dbconn.Global = db
	permsStore := frontendDB.NewPermsStore(db, clock)
	permsSyncer := authz.NewPermsSyncer(repoStore, permsStore, clock, ratelimit.DefaultRegistry)
	go startBackgroundPermsSync(ctx, permsSyncer, db)
	debugDumpers = append(debugDumpers, permsSyncer)
	if server != nil {
		server.PermsSyncer = permsSyncer
	}

	return debugDumpers
}

// startBackgroundPermsSync sets up background permissions syncing.
func startBackgroundPermsSync(ctx context.Context, syncer *authz.PermsSyncer, db dbutil.DB) {
	globals.WatchPermissionsBackgroundSync()
	go func() {
		t := time.NewTicker(5 * time.Second)
		for range t.C {
			allowAccessByDefault, authzProviders, _, _ :=
				frontendAuthz.ProvidersFromConfig(ctx, conf.Get(), ossDB.ExternalServices, db)
			ossAuthz.SetProviders(allowAccessByDefault, authzProviders)
		}
	}()

	go syncer.Run(ctx)
}
