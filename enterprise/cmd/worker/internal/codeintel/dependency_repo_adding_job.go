package codeintel

import (
	"context"
	"time"

	"github.com/inconshreveable/log15"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/sourcegraph/sourcegraph/cmd/worker/shared"
	"github.com/sourcegraph/sourcegraph/enterprise/cmd/worker/internal/codeintel/dependencyrepos"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/stores/dbstore"
	"github.com/sourcegraph/sourcegraph/internal/env"
	"github.com/sourcegraph/sourcegraph/internal/goroutine"
	"github.com/sourcegraph/sourcegraph/internal/observation"
	"github.com/sourcegraph/sourcegraph/internal/trace"
	"github.com/sourcegraph/sourcegraph/internal/workerutil"
)

type dependencyRepoAddingJob struct{}

func NewDependencyRepoAddingJob() shared.Job {
	return &dependencyRepoAddingJob{}
}

func (j *dependencyRepoAddingJob) Config() []env.Config {
	return []env.Config{}
}

func (j *dependencyRepoAddingJob) Routines(ctx context.Context) ([]goroutine.BackgroundRoutine, error) {
	observationContext := &observation.Context{
		Logger:     log15.Root(),
		Tracer:     &trace.Tracer{Tracer: opentracing.GlobalTracer()},
		Registerer: prometheus.DefaultRegisterer,
	}

	dbStore, err := InitDBStore()
	if err != nil {
		return nil, err
	}

	dbStoreShim := &dependencyrepos.DBStoreShim{Store: dbStore}
	metrics := workerutil.NewMetrics(observationContext, "codeintel_dependency_repo_adder", nil)

	return []goroutine.BackgroundRoutine{
		dependencyrepos.NewDependencyRepoAdder(
			dbStoreShim,
			dbstore.WorkerutilDependencyRepoAddingJobStore(dbStore, observationContext),
			time.Minute*1,
			1,
			metrics),
	}, nil
}
