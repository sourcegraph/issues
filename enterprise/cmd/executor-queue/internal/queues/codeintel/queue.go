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
		Name:              "codeintel",
		Store:             newWorkerStore(db, observationContext),
		RecordTransformer: recordTransformer,
	}
}

// newWorkerStore creates a dbworker store that wraps the lsif_indexes table.
func newWorkerStore(db dbutil.DB, observationContext *observation.Context) apiserver.QueueStore {
	handle := basestore.NewHandleWithDB(db, sql.TxOptions{})
	options := dbworkerstore.Options{
		Name:              "precise_code_intel_index_worker_store",
		TableName:         "lsif_indexes",
		ViewName:          "lsif_indexes_with_repository_name u",
		ColumnExpressions: store.IndexColumnsWithNullRank,
		Scan:              store.ScanFirstIndexRecord,
		OrderByExpression: sqlf.Sprintf("u.queued_at, u.id"),
		StalledMaxAge:     StalledJobMaximumAge,
		MaxNumResets:      MaximumNumResets,
	}

	return queueStoreWrapper{Store: dbworkerstore.NewWithMetrics(handle, options, observationContext), options: options}
}

type queueStoreWrapper struct {
	dbworkerstore.Store
	options dbworkerstore.Options
}

func (s queueStoreWrapper) ExecutorLastUpdate(ctx context.Context, executorName string) (time.Time, error) {
	q := sqlf.Sprintf(`
	SELECT
		max(last_heartbeat_at)
	FROM
		lsif_indexes
	WHERE
		worker_hostname = %s AND last_heartbeat_at IS NOT NULL
	GROUP BY
		worker_hostname
	`, executorName)
	time, _, err := basestore.ScanFirstTime(s.Store.Handle().DB().QueryContext(ctx, q.Query(sqlf.PostgresBindVar), q.Args()...))
	return time, err
}

func (s queueStoreWrapper) RecordStartedAt(ctx context.Context, executorName string, recordID int) (time.Time, error) {
	q := sqlf.Sprintf(`
	SELECT
		started_at
	FROM
		lsif_indexes
	WHERE
		worker_hostname = %s AND id = %s AND state = 'processing'
	`, executorName, recordID)
	time, _, err := basestore.ScanFirstTime(s.Store.Handle().DB().QueryContext(ctx, q.Query(sqlf.PostgresBindVar), q.Args()...))
	return time, err
}

func (s queueStoreWrapper) HeartbeatRecords(ctx context.Context, executorName string, recordIDs []int) ([]int, error) {
	ids := make([]*sqlf.Query, 0, len(recordIDs))
	for _, id := range recordIDs {
		ids = append(ids, sqlf.Sprintf("%s", id))
	}
	q := sqlf.Sprintf(`
	WITH alive_candidates AS (
		SELECT
			id
		FROM
			lsif_indexes
		WHERE
			id IN (%s)
			AND
			state = 'processing'
			AND
			worker_hostname = %s
		FOR UPDATE
	)
	UPDATE
		lsif_indexes
	SET
		last_heartbeat_at = %s
	WHERE
		id IN (SELECT id FROM alive_candidates)
	RETURNING id
	`, sqlf.Join(ids, ","), executorName, time.Now())
	rows, err := s.Store.Handle().DB().QueryContext(ctx, q.Query(sqlf.PostgresBindVar), q.Args()...)
	if err != nil {
		return []int{}, err
	}
	defer func() { err = basestore.CloseRows(rows, err) }()

	resultIDs := make([]int, 0)
	for rows.Next() {
		var value int
		if err := rows.Scan(&value); err != nil {
			return []int{}, err
		}
		resultIDs = append(resultIDs, value)
	}
	return resultIDs, err
}
