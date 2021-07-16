package versions

import (
	"context"
	"time"

	"github.com/inconshreveable/log15"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/envvar"
	"github.com/sourcegraph/sourcegraph/cmd/worker/shared"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
	"github.com/sourcegraph/sourcegraph/internal/env"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/goroutine"
	"github.com/sourcegraph/sourcegraph/internal/httpcli"
	"github.com/sourcegraph/sourcegraph/internal/repos"
	"github.com/sourcegraph/sourcegraph/internal/types"
)

const syncInterval = 24 * time.Hour

func NewSyncingJob(db *repos.Store) shared.Job {
	return &syncingJob{}
}

type syncingJob struct {
	db *repos.Store
}

func (j *syncingJob) Config() []env.Config {
	return []env.Config{}
}

func (j *syncingJob) Routines(_ context.Context) ([]goroutine.BackgroundRoutine, error) {
	if envvar.SourcegraphDotComMode() {
		// If we're on sourcegraph.com we don't want to run this
		return nil, nil
	}

	db, err := shared.InitDatabase()
	if err != nil {
		return nil, err
	}

	cf := httpcli.NewExternalHTTPClientFactory()
	sourcer := repos.NewSourcer(cf, j.db)

	handler := goroutine.NewHandlerWithErrorMessage("sync versions of external services", func(ctx context.Context) error {
		versions, err := loadVersions(ctx, db, sourcer)
		if err != nil {
			return err
		}
		return storeVersions(versions)
	})

	return []goroutine.BackgroundRoutine{
		// Pass a fresh context, see docs for shared.Job
		goroutine.NewPeriodicGoroutine(context.Background(), syncInterval, handler),
	}, nil
}

func loadVersions(ctx context.Context, db dbutil.DB, sourcer repos.Sourcer) ([]*Version, error) {
	var versions []*Version

	es, err := database.ExternalServices(db).List(ctx, database.ExternalServicesListOptions{})
	if err != nil {
		return versions, err
	}

	// Group the external services by the code host instance they point at so
	// we don't send >1 requests to the same instance.
	unique := make(map[string]*types.ExternalService)
	for _, svc := range es {
		ident, err := extsvc.UniqueCodeHostIdentifier(svc.Kind, svc.Config)
		if err != nil {
			return versions, err
		}

		if _, ok := unique[ident]; ok {
			continue
		}
		unique[ident] = svc
	}

	for _, svc := range unique {
		sources, err := sourcer(svc)
		if err != nil {
			return versions, err
		}
		src := sources[0]

		versionSrc, ok := src.(repos.VersionSource)
		if !ok {
			log15.Debug("external service source does not implement VersionSource interface", "kind", svc.Kind)
			continue
		}

		v, err := versionSrc.Version(ctx)
		if err != nil {
			log15.Warn("failed to fetch version of code host", "version", v, "error", err)
			continue
		}

		versions = append(versions, &Version{
			ExternalServiceKind: svc.Kind,
			Version:             v,
		})
	}

	return versions, nil
}
