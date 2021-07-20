package codeintel

import (
	"context"
	"database/sql"
	"time"

	"github.com/keegancsmith/sqlf"

	apiserver "github.com/sourcegraph/sourcegraph/enterprise/cmd/executor-queue/internal/server"
	store "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/stores/dbstore"
	apiclient "github.com/sourcegraph/sourcegraph/enterprise/internal/executor"
	"github.com/sourcegraph/sourcegraph/internal/database/basestore"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
	"github.com/sourcegraph/sourcegraph/internal/observation"
	"github.com/sourcegraph/sourcegraph/internal/workerutil"
	dbworkerstore "github.com/sourcegraph/sourcegraph/internal/workerutil/dbworker/store"
)

// HeartbeatInterval is the duration between heartbeat updates to the job records.
const HeartbeatInterval = time.Second

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
	recordTransformer := func(ctx context.Context, record workerutil.Record) (apiclient.Job, error) {
		return transformRecord(record.(store.Index), config)
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
		Name:              "precise_code_intel_index_worker_store",
		TableName:         "lsif_indexes",
		ViewName:          "lsif_indexes_with_repository_name u",
		ColumnExpressions: store.IndexColumnsWithNullRank,
		Scan:              store.ScanFirstIndexRecord,
		OrderByExpression: sqlf.Sprintf("u.queued_at, u.id"),
		HeartbeatInterval: HeartbeatInterval,
		StalledMaxAge:     StalledJobMaximumAge,
		MaxNumResets:      MaximumNumResets,
	}

	return dbworkerstore.NewWithMetrics(handle, options, observationContext)
}
