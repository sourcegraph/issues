package batches

import (
	"database/sql"
	"time"

	"github.com/keegancsmith/sqlf"

	apiserver "github.com/sourcegraph/sourcegraph/enterprise/cmd/executor-queue/internal/server"
	store "github.com/sourcegraph/sourcegraph/enterprise/internal/batches/store"
	btypes "github.com/sourcegraph/sourcegraph/enterprise/internal/batches/types"
	apiclient "github.com/sourcegraph/sourcegraph/enterprise/internal/executor"
	"github.com/sourcegraph/sourcegraph/internal/database/basestore"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
	"github.com/sourcegraph/sourcegraph/internal/observation"
	"github.com/sourcegraph/sourcegraph/internal/workerutil"
	dbworkerstore "github.com/sourcegraph/sourcegraph/internal/workerutil/dbworker/store"
)

// StalledJobMaximumAge is the maximum allowable duration between updating the state of a
// job as "processing" and locking the record during processing. An unlocked row that is
// marked as processing likely indicates that the executor that dequeued the job has died.
// There should be a nearly-zero delay between these states during normal operation.
const StalledJobMaximumAge = time.Second * 5

// MaximumNumResets is the maximum number of times a job can be reset. If a job's failed
// attempts counter reaches this threshold, it will be moved into "errored" rather than
// "queued" on its next reset.
const MaximumNumResets = 3

func QueueOptions(db dbutil.DB, config *Config, observationContext *observation.Context) apiserver.QueueOptions {
	recordTransformer := func(record workerutil.Record) (apiclient.Job, error) {
		bej := record.(batchExecutorJobRecord).BatchExecutorJob
		bej.Job.ID = int(bej.ID)

		return bej.Job, nil
	}

	return apiserver.QueueOptions{
		Store:             newWorkerStore(db, observationContext),
		RecordTransformer: recordTransformer,
	}
}

// newWorkerStore creates a dbworker store that wraps the lsif_indexes table.
func newWorkerStore(db dbutil.DB, observationContext *observation.Context) dbworkerstore.Store {
	handle := basestore.NewHandleWithDB(db, sql.TxOptions{})
	options := dbworkerstore.Options{
		Name:              "batch_executor_job_worker_store",
		TableName:         "batch_executor_jobs",
		ColumnExpressions: store.BatchExecutorJobColumns,
		Scan: func(rows *sql.Rows, err error) (workerutil.Record, bool, error) {
			bej, exists, err := store.ScanFirstBatchExecutorJob(rows, err)
			return batchExecutorJobRecord{bej}, exists, err
		},
		OrderByExpression: sqlf.Sprintf("created_at, id"),
		StalledMaxAge:     StalledJobMaximumAge,
		MaxNumResets:      MaximumNumResets,
	}

	return dbworkerstore.NewWithMetrics(handle, options, observationContext)
}

type batchExecutorJobRecord struct{ *btypes.BatchExecutorJob }

var _ workerutil.Record = batchExecutorJobRecord{}

func (r batchExecutorJobRecord) RecordID() int {
	return int(r.ID)
}
