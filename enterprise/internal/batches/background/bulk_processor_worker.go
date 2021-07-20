package background

import (
	"context"
	"database/sql"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/keegancsmith/sqlf"

	"github.com/sourcegraph/sourcegraph/enterprise/internal/batches/sources"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/batches/store"
	btypes "github.com/sourcegraph/sourcegraph/enterprise/internal/batches/types"
	"github.com/sourcegraph/sourcegraph/internal/workerutil"
	"github.com/sourcegraph/sourcegraph/internal/workerutil/dbworker"
	dbworkerstore "github.com/sourcegraph/sourcegraph/internal/workerutil/dbworker/store"
)

// bulkProcessorMaxNumRetries is the maximum number of attempts the bulkProcessor
// makes to process a changeset job when it fails.
const bulkProcessorMaxNumRetries = 10

// bulkProcessorMaxNumResets is the maximum number of attempts the bulkProcessor
// makes to process a changeset job when it stalls (process crashes, etc.).
const bulkProcessorMaxNumResets = 60

// newBulkOperationWorker creates a dbworker.Worker that fetches enqueued changeset_jobs
// from the database and passes them to the bulk executor for processing.
func newBulkOperationWorker(
	ctx context.Context,
	s *store.Store,
	sourcer sources.Sourcer,
	metrics batchChangesMetrics,
) *workerutil.Worker {
	r := &bulkProcessorWorker{sourcer: sourcer, store: s}

	options := workerutil.WorkerOptions{
		Name:        "batches_bulk_processor",
		NumHandlers: 5,
		Interval:    5 * time.Second,
		Metrics:     metrics.bulkProcessorWorkerMetrics,
	}

	workerStore := createBulkOperationDBWorkerStore(s)

	worker := dbworker.NewWorker(ctx, workerStore, r.HandlerFunc(), options)
	return worker
}

// newBulkOperationWorkerResetter creates a dbworker.Resetter that reenqueues lost jobs
// for processing.
func newBulkOperationWorkerResetter(s *store.Store, metrics batchChangesMetrics) *dbworker.Resetter {
	workerStore := createBulkOperationDBWorkerStore(s)

	options := dbworker.ResetterOptions{
		Name:     "batches_bulk_worker_resetter",
		Interval: 1 * time.Minute,
		Metrics:  metrics.bulkProcessorWorkerResetterMetrics,
	}

	resetter := dbworker.NewResetter(workerStore, options)
	return resetter
}

func createBulkOperationDBWorkerStore(s *store.Store) dbworkerstore.Store {
	return dbworkerstore.New(s.Handle(), dbworkerstore.Options{
		Name:              "batches_bulk_worker_store",
		TableName:         "changeset_jobs",
		ColumnExpressions: store.ChangesetJobColumns.ToSqlf(),
		Scan:              scanFirstChangesetJobRecord,

		OrderByExpression: sqlf.Sprintf("changeset_jobs.state = 'errored', changeset_jobs.updated_at DESC"),

		HeartbeatInterval: 15 * time.Second,
		StalledMaxAge:     60 * time.Second,
		MaxNumResets:      bulkProcessorMaxNumResets,

		RetryAfter:    5 * time.Second,
		MaxNumRetries: bulkProcessorMaxNumRetries,
	})
}

// scanFirstChangesetJobRecord wraps store.ScanFirstChangesetJob to return a
// generic workerutil.Record.
func scanFirstChangesetJobRecord(rows *sql.Rows, err error) (workerutil.Record, bool, error) {
	return store.ScanFirstChangesetJob(rows, err)
}

// bulkProcessorWorker is a wrapper for the workerutil handlerfunc to create a
// bulkProcessor with a source and store.
type bulkProcessorWorker struct {
	store   *store.Store
	sourcer sources.Sourcer
}

func (b *bulkProcessorWorker) HandlerFunc() workerutil.HandlerFunc {
	return func(ctx context.Context, record workerutil.Record) (err error) {
		tx, err := b.store.Transact(ctx)
		if err != nil {
			return err
		}
		defer func() {
			doneErr := tx.Done(nil)
			if err != nil && doneErr != nil {
				err = multierror.Append(err, doneErr)
			}
			if doneErr != nil {
				err = doneErr
			}
		}()

		processor := &bulkProcessor{
			tx:      tx,
			sourcer: b.sourcer,
		}
		return processor.process(ctx, record.(*btypes.ChangesetJob))
	}
}
