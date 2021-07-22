package codeintel

import (
	"context"

	"github.com/inconshreveable/log15"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/sourcegraph/sourcegraph/cmd/worker/shared"
	"github.com/sourcegraph/sourcegraph/enterprise/cmd/worker/internal/codeintel/janitor"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/stores/dbstore"
	"github.com/sourcegraph/sourcegraph/internal/env"
	"github.com/sourcegraph/sourcegraph/internal/goroutine"
	"github.com/sourcegraph/sourcegraph/internal/observation"
	"github.com/sourcegraph/sourcegraph/internal/trace"
)

type janitorJob struct{}

func NewJanitorJob() shared.Job {
	return &janitorJob{}
}

func (j *janitorJob) Config() []env.Config {
	return []env.Config{janitorConfigInst}
}

func (j *janitorJob) Routines(ctx context.Context) ([]goroutine.BackgroundRoutine, error) {
	observationContext := &observation.Context{
		Logger:     log15.Root(),
		Tracer:     &trace.Tracer{Tracer: opentracing.GlobalTracer()},
		Registerer: prometheus.DefaultRegisterer,
	}

	dbStore, err := InitDBStore()
	if err != nil {
		return nil, err
	}

	lsifStore, err := InitLSIFStore()
	if err != nil {
		return nil, err
	}

	dependencyIndexStore, err := InitDependencyIndexStore()
	if err != nil {
		return nil, err
	}

	dbStoreShim := &janitor.DBStoreShim{Store: dbStore}
	uploadWorkerStore := dbstore.WorkerutilUploadStore(dbStoreShim, observationContext)
	indexWorkerStore := dbstore.WorkerutilIndexStore(dbStoreShim, observationContext)
	metrics := janitor.NewMetrics(observationContext)

	routines := []goroutine.BackgroundRoutine{
		janitor.NewAbandonedUploadJanitor(dbStoreShim, janitorConfigInst.UploadTimeout, janitorConfigInst.CleanupTaskInterval, metrics),
		janitor.NewDeletedRepositoryJanitor(dbStoreShim, janitorConfigInst.CleanupTaskInterval, metrics),
		janitor.NewHardDeleter(dbStoreShim, lsifStore, janitorConfigInst.CleanupTaskInterval, metrics),
		janitor.NewRecordExpirer(dbStoreShim, janitorConfigInst.DataTTL, janitorConfigInst.CleanupTaskInterval, metrics),
		janitor.NewUploadResetter(uploadWorkerStore, janitorConfigInst.CleanupTaskInterval, metrics, observationContext),
		janitor.NewIndexResetter(indexWorkerStore, janitorConfigInst.CleanupTaskInterval, metrics, observationContext),
		janitor.NewDependencyIndexResetter(dependencyIndexStore, janitorConfigInst.CleanupTaskInterval, metrics, observationContext),
		janitor.NewUnknownCommitJanitor(dbStoreShim, janitorConfigInst.CommitResolverMinimumTimeSinceLastCheck, janitorConfigInst.CommitResolverBatchSize, janitorConfigInst.CommitResolverTaskInterval, metrics),
	}

	return routines, nil
}
