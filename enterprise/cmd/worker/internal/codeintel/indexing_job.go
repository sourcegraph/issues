package codeintel

import (
	"context"

	"github.com/inconshreveable/log15"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/sourcegraph/sourcegraph/cmd/worker/shared"
	"github.com/sourcegraph/sourcegraph/enterprise/cmd/worker/internal/codeintel/indexing"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/autoindex/enqueuer"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/env"
	"github.com/sourcegraph/sourcegraph/internal/goroutine"
	"github.com/sourcegraph/sourcegraph/internal/observation"
	"github.com/sourcegraph/sourcegraph/internal/repoupdater"
	"github.com/sourcegraph/sourcegraph/internal/trace"
	"github.com/sourcegraph/sourcegraph/internal/workerutil"
)

type indexingJob struct{}

func NewIndexingJob() shared.Job {
	return &indexingJob{}
}

func (j *indexingJob) Config() []env.Config {
	return []env.Config{indexingConfigInst}
}

func (j *indexingJob) Routines(ctx context.Context) ([]goroutine.BackgroundRoutine, error) {
	observationContext := &observation.Context{
		Logger:     log15.Root(),
		Tracer:     &trace.Tracer{Tracer: opentracing.GlobalTracer()},
		Registerer: prometheus.DefaultRegisterer,
	}

	db, err := shared.InitDatabase()
	if err != nil {
		return nil, err
	}

	dbStore, err := InitDBStore()
	if err != nil {
		return nil, err
	}

	gitserverClient, err := InitGitserverClient()
	if err != nil {
		return nil, err
	}

	dependencyIndexStore, err := InitDependencyIndexStore()
	if err != nil {
		return nil, err
	}

	dbStoreShim := &indexing.DBStoreShim{Store: dbStore}
	enqueuerDBStoreShim := &enqueuer.DBStoreShim{Store: dbStore}
	indexEnqueuer := enqueuer.NewIndexEnqueuer(enqueuerDBStoreShim, gitserverClient, repoupdater.DefaultClient, observationContext)
	metrics := workerutil.NewMetrics(observationContext, "codeintel_dependency_index_processor", nil)

	settingStore := database.Settings(db)
	repoStore := database.Repos(db)

	prometheus.DefaultRegisterer.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "src_codeintel_dependency_index_total",
		Help: "Total number of jobs in the queued state.",
	}, func() float64 {
		count, err := dependencyIndexStore.QueuedCount(context.Background(), nil)
		if err != nil {
			log15.Error("Failed to get queued job count", "error", err)
		}

		return float64(count)
	}))

	routines := []goroutine.BackgroundRoutine{
		indexing.NewIndexScheduler(dbStoreShim, settingStore, repoStore, indexEnqueuer, indexingConfigInst.AutoIndexingTaskInterval, observationContext),
		indexing.NewDependencyIndexingScheduler(dbStoreShim, dependencyIndexStore, indexEnqueuer, indexingConfigInst.DependencyIndexerSchedulerPollInterval, indexingConfigInst.DependencyIndexerSchedulerConcurrency, metrics),
	}

	return routines, nil
}
