package repos

import (
	"context"
	"database/sql"
	"time"

	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/inconshreveable/log15"
	"github.com/keegancsmith/sqlf"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/sourcegraph/sourcegraph/internal/database/basestore"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
	"github.com/sourcegraph/sourcegraph/internal/observation"
	"github.com/sourcegraph/sourcegraph/internal/trace"
	"github.com/sourcegraph/sourcegraph/internal/workerutil"
	"github.com/sourcegraph/sourcegraph/internal/workerutil/dbworker"
	"github.com/sourcegraph/sourcegraph/internal/workerutil/dbworker/store"
)

type SyncWorkerOptions struct {
	NumHandlers            int                   // defaults to 3
	WorkerInterval         time.Duration         // defaults to 10s
	PrometheusRegisterer   prometheus.Registerer // if non-nil, metrics will be collected
	CleanupOldJobs         bool                  // run a background process to cleanup old jobs
	CleanupOldJobsInterval time.Duration         // defaults to 1h
}

// NewSyncWorker creates a new external service sync worker.
func NewSyncWorker(ctx context.Context, db dbutil.DB, handler workerutil.Handler, opts SyncWorkerOptions) (*workerutil.Worker, *dbworker.Resetter) {
	if opts.NumHandlers == 0 {
		opts.NumHandlers = 3
	}
	if opts.WorkerInterval == 0 {
		opts.WorkerInterval = 10 * time.Second
	}
	if opts.CleanupOldJobsInterval == 0 {
		opts.CleanupOldJobsInterval = time.Hour
	}

	dbHandle := basestore.NewHandleWithDB(db, sql.TxOptions{
		// Change the isolation level for every transaction created by the worker
		// so that multiple workers can modify the same rows without conflicts.
		Isolation: sql.LevelReadCommitted,
	})

	syncJobColumns := []*sqlf.Query{
		sqlf.Sprintf("id"),
		sqlf.Sprintf("state"),
		sqlf.Sprintf("failure_message"),
		sqlf.Sprintf("started_at"),
		sqlf.Sprintf("finished_at"),
		sqlf.Sprintf("process_after"),
		sqlf.Sprintf("num_resets"),
		sqlf.Sprintf("num_failures"),
		sqlf.Sprintf("execution_logs"),
		sqlf.Sprintf("external_service_id"),
		sqlf.Sprintf("next_sync_at"),
	}

	store := store.New(dbHandle, store.Options{
		Name:              "repo_sync_worker_store",
		TableName:         "external_service_sync_jobs",
		ViewName:          "external_service_sync_jobs_with_next_sync_at",
		Scan:              scanSingleJob,
		OrderByExpression: sqlf.Sprintf("next_sync_at"),
		ColumnExpressions: syncJobColumns,
		HeartbeatInterval: 15 * time.Second,
		StalledMaxAge:     30 * time.Second,
		MaxNumResets:      5,
		MaxNumRetries:     0,
	})

	worker := dbworker.NewWorker(ctx, store, handler, workerutil.WorkerOptions{
		Name:        "repo_sync_worker",
		NumHandlers: opts.NumHandlers,
		Interval:    opts.WorkerInterval,
		Metrics:     newWorkerMetrics(opts.PrometheusRegisterer),
	})

	resetter := dbworker.NewResetter(store, dbworker.ResetterOptions{
		Name:     "repo_sync_worker_resetter",
		Interval: 5 * time.Minute,
		Metrics:  newResetterMetrics(opts.PrometheusRegisterer),
	})

	if opts.CleanupOldJobs {
		go runJobCleaner(ctx, db, opts.CleanupOldJobsInterval)
	}

	return worker, resetter
}

func newWorkerMetrics(r prometheus.Registerer) workerutil.WorkerMetrics {
	var observationContext *observation.Context

	if r == nil {
		observationContext = &observation.TestContext
	} else {
		observationContext = &observation.Context{
			Logger:     log15.Root(),
			Tracer:     &trace.Tracer{Tracer: opentracing.GlobalTracer()},
			Registerer: r,
		}
	}

	return workerutil.NewMetrics(observationContext, "repo_updater_external_service_syncer", nil)
}

func newResetterMetrics(r prometheus.Registerer) dbworker.ResetterMetrics {
	return dbworker.ResetterMetrics{
		RecordResets: promauto.With(r).NewCounter(prometheus.CounterOpts{
			Name: "src_external_service_queue_resets_total",
			Help: "Total number of external services put back into queued state",
		}),
		RecordResetFailures: promauto.With(r).NewCounter(prometheus.CounterOpts{
			Name: "src_external_service_queue_max_resets_total",
			Help: "Total number of external services that exceed the max number of resets",
		}),
		Errors: promauto.With(r).NewCounter(prometheus.CounterOpts{
			Name: "src_external_service_queue_reset_errors_total",
			Help: "Total number of errors when running the external service resetter",
		}),
	}
}

func runJobCleaner(ctx context.Context, db dbutil.DB, interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()

	for {
		_, err := db.ExecContext(ctx, `
-- source: internal/repos/sync_worker.go:runJobCleaner
DELETE FROM external_service_sync_jobs
WHERE
  finished_at < now() - INTERVAL '1 day'
  AND state IN ('completed', 'errored')
`)
		if err != nil && err != context.Canceled {
			log15.Error("error while running job cleaner", "err", err)
		}

		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

func scanSingleJob(rows *sql.Rows, err error) (workerutil.Record, bool, error) {
	if err != nil {
		return nil, false, err
	}

	jobs, err := scanJobs(rows)
	if err != nil {
		return nil, false, err
	}

	var job SyncJob

	if len(jobs) > 0 {
		job = jobs[0]
	}

	return &job, true, nil
}

// SyncJob represents an external service that needs to be synced
type SyncJob struct {
	ID                int
	State             string
	FailureMessage    sql.NullString
	StartedAt         sql.NullTime
	FinishedAt        sql.NullTime
	ProcessAfter      sql.NullTime
	NumResets         int
	NumFailures       int
	ExternalServiceID int64
	NextSyncAt        sql.NullTime
}

// RecordID implements workerutil.Record and indicates the queued item id
func (s *SyncJob) RecordID() int {
	return s.ID
}
