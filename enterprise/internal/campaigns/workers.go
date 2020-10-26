package campaigns

import (
	"context"
	"database/sql"
	"time"

	"github.com/inconshreveable/log15"
	"github.com/keegancsmith/sqlf"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sourcegraph/sourcegraph/cmd/repo-updater/repos"
	"github.com/sourcegraph/sourcegraph/internal/metrics"
	"github.com/sourcegraph/sourcegraph/internal/observation"
	"github.com/sourcegraph/sourcegraph/internal/trace"
	"github.com/sourcegraph/sourcegraph/internal/workerutil"
	"github.com/sourcegraph/sourcegraph/internal/workerutil/dbworker"
	dbworkerstore "github.com/sourcegraph/sourcegraph/internal/workerutil/dbworker/store"
)

// RunWorkers starts a dbworker.NewWorker that fetches enqueued changesets
// from the database and passes them to the changeset reconciler for
// processing.
func RunWorkers(
	ctx context.Context,
	s *Store,
	gitClient GitserverClient,
	sourcer repos.Sourcer,
) {
	r := &reconciler{gitserverClient: gitClient, sourcer: sourcer, store: s}

	options := workerutil.WorkerOptions{
		NumHandlers: 5,
		Interval:    5 * time.Second,
		Metrics: workerutil.WorkerMetrics{
			HandleOperation: newObservationOperation(),
		},
	}

	workerStore := dbworkerstore.NewStore(s.Handle(), dbworkerstore.StoreOptions{
		TableName:            "changesets",
		AlternateColumnNames: map[string]string{"state": "reconciler_state"},
		ColumnExpressions:    changesetColumns,
		Scan:                 scanFirstChangesetRecord,

		// Order changesets by state, so that freshly enqueued changesets have
		// higher priority.
		// If state is equal, prefer the newer ones.
		OrderByExpression: sqlf.Sprintf("reconciler_state = 'errored', changesets.updated_at DESC"),

		StalledMaxAge: 60 * time.Second,
		MaxNumResets:  reconcilerMaxNumResets,

		RetryAfter:    5 * time.Second,
		MaxNumRetries: reconcilerMaxNumRetries,
	})

	worker := dbworker.NewWorker(ctx, workerStore, r.HandlerFunc(), options)
	worker.Start()
}

// reconcilerMaxNumRetries is the maximum number of attempts the reconciler
// makes to process a changeset when it fails.
const reconcilerMaxNumRetries = 60

// reconcilerMaxNumResets is the maximum number of attempts the reconciler
// makes to process a changeset when it stalls (process crashes, etc.).
const reconcilerMaxNumResets = 60

func scanFirstChangesetRecord(rows *sql.Rows, err error) (workerutil.Record, bool, error) {
	return scanFirstChangeset(rows, err)
}

func newObservationOperation() *observation.Operation {
	observationContext := &observation.Context{
		Logger:     log15.Root(),
		Tracer:     &trace.Tracer{Tracer: opentracing.GlobalTracer()},
		Registerer: prometheus.DefaultRegisterer,
	}

	metrics := metrics.NewOperationMetrics(
		observationContext.Registerer,
		"campaigns_reconciler",
		metrics.WithLabels("op"),
		metrics.WithCountHelp("Total number of results returned"),
	)

	return observationContext.Operation(observation.Op{
		Name:         "Reconciler.Process",
		MetricLabels: []string{"process"},
		Metrics:      metrics,
	})
}
