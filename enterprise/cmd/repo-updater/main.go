package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/envvar"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/globals"
	"github.com/sourcegraph/sourcegraph/cmd/repo-updater/repoupdater"
	"github.com/sourcegraph/sourcegraph/cmd/repo-updater/shared"
	"github.com/sourcegraph/sourcegraph/enterprise/cmd/repo-updater/internal/authz"
	frontendAuthz "github.com/sourcegraph/sourcegraph/enterprise/internal/authz"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/batches"
	codemonitorsBackground "github.com/sourcegraph/sourcegraph/enterprise/internal/codemonitors/background"
	edb "github.com/sourcegraph/sourcegraph/enterprise/internal/database"
	insightsBackground "github.com/sourcegraph/sourcegraph/enterprise/internal/insights/background"
	"github.com/sourcegraph/sourcegraph/internal/actor"
	ossAuthz "github.com/sourcegraph/sourcegraph/internal/authz"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	ossDB "github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
	"github.com/sourcegraph/sourcegraph/internal/debugserver"
	"github.com/sourcegraph/sourcegraph/internal/encryption/keyring"
	"github.com/sourcegraph/sourcegraph/internal/httpcli"
	"github.com/sourcegraph/sourcegraph/internal/ratelimit"
	"github.com/sourcegraph/sourcegraph/internal/repos"
	"github.com/sourcegraph/sourcegraph/internal/timeutil"
)

func main() {
	debug, _ := strconv.ParseBool(os.Getenv("DEBUG"))
	if debug {
		log.Println("enterprise edition")
	}
	shared.Main(enterpriseInit)
}

func enterpriseInit(
	db *sql.DB,
	repoStore *repos.Store,
	keyring keyring.Ring,
	cf *httpcli.Factory,
	server *repoupdater.Server,
) (debugDumpers []debugserver.Dumper) {
	// NOTE: Internal actor is required to have full visibility of the repo table
	// 	(i.e. bypass repository authorization).
	ctx := actor.WithInternalActor(context.Background())

	codemonitorsBackground.StartBackgroundJobs(ctx, db)
	insightsBackground.StartBackgroundJobs(ctx, db)

	// No Batch Changes on dotcom, so we don't need to spawn the
	// background jobs for this feature.
	if !envvar.SourcegraphDotComMode() {
		syncRegistry := batches.InitBackgroundJobs(ctx, db, keyring.BatchChangesCredentialKey, cf)
		if server != nil {
			server.ChangesetSyncRegistry = syncRegistry
		}
	}

	permsStore := edb.Perms(db, timeutil.Now)
	permsSyncer := authz.NewPermsSyncer(repoStore, permsStore, timeutil.Now, ratelimit.DefaultRegistry)
	go startBackgroundPermsSync(ctx, permsSyncer, db)
	debugDumpers = append(debugDumpers, permsSyncer)
	if server != nil {
		server.PermsSyncer = permsSyncer
	}

	return debugDumpers
}

// startBackgroundPermsSync sets up background permissions syncing.
func startBackgroundPermsSync(ctx context.Context, syncer *authz.PermsSyncer, db dbutil.DB) {
	globals.WatchPermissionsUserMapping()
	go func() {
		t := time.NewTicker(5 * time.Second)
		for range t.C {
			allowAccessByDefault, authzProviders, _, _ :=
				frontendAuthz.ProvidersFromConfig(
					ctx,
					conf.Get(),
					ossDB.ExternalServices(db),
				)
			ossAuthz.SetProviders(allowAccessByDefault, authzProviders)
		}
	}()

	go syncer.Run(ctx)
}
