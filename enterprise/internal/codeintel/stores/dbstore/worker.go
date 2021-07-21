package dbstore

import (
	"time"

	"github.com/keegancsmith/sqlf"

	"github.com/sourcegraph/sourcegraph/internal/database/basestore"
	"github.com/sourcegraph/sourcegraph/internal/observation"
	dbworkerstore "github.com/sourcegraph/sourcegraph/internal/workerutil/dbworker/store"
)

// UploadHeartbeatInterval is the duration between heartbeat updates to the upload job records.
const UploadHeartbeatInterval = time.Second

// StalledUploadMaxAge is the maximum allowable duration between updating the state of an
// upload as "processing" and locking the upload row during processing. An unlocked row that
// is marked as processing likely indicates that the worker that dequeued the upload has died.
// There should be a nearly-zero delay between these states during normal operation.
const StalledUploadMaxAge = time.Second * 5

// UploadMaxNumResets is the maximum number of times an upload can be reset. If an upload's
// failed attempts counter reaches this threshold, it will be moved into "errored" rather than
// "queued" on its next reset.
const UploadMaxNumResets = 3

var uploadWorkerStoreOptions = dbworkerstore.Options{
	Name:              "precise_code_intel_upload_worker_store",
	TableName:         "lsif_uploads",
	ViewName:          "lsif_uploads_with_repository_name u",
	ColumnExpressions: uploadColumnsWithNullRank,
	Scan:              scanFirstUploadRecord,
	OrderByExpression: sqlf.Sprintf("u.uploaded_at, u.id"),
	HeartbeatInterval: UploadHeartbeatInterval,
	StalledMaxAge:     StalledUploadMaxAge,
	MaxNumResets:      UploadMaxNumResets,
}

func WorkerutilUploadStore(s basestore.ShareableStore, observationContext *observation.Context) dbworkerstore.Store {
	return dbworkerstore.NewWithMetrics(s.Handle(), uploadWorkerStoreOptions, observationContext)
}

// IndexHeartbeatInterval is the duration between heartbeat updates to the index job records.
const IndexHeartbeatInterval = time.Second

// StalledIndexMaxAge is the maximum allowable duration between updating the state of an
// index as "processing" and locking the index row during processing. An unlocked row that
// is marked as processing likely indicates that the indexer that dequeued the index has
// died. There should be a nearly-zero delay between these states during normal operation.
const StalledIndexMaxAge = time.Second * 5

// IndexMaxNumResets is the maximum number of times an index can be reset. If an index's
// failed attempts counter reaches this threshold, it will be moved into "errored" rather than
// "queued" on its next reset.
const IndexMaxNumResets = 3

var indexWorkerStoreOptions = dbworkerstore.Options{
	Name:              "precise_code_intel_index_worker_store",
	TableName:         "lsif_indexes",
	ViewName:          "lsif_indexes_with_repository_name u",
	ColumnExpressions: indexColumnsWithNullRank,
	Scan:              scanFirstIndexRecord,
	OrderByExpression: sqlf.Sprintf("u.queued_at, u.id"),
	HeartbeatInterval: IndexHeartbeatInterval,
	StalledMaxAge:     StalledIndexMaxAge,
	MaxNumResets:      IndexMaxNumResets,
}

func WorkerutilIndexStore(s basestore.ShareableStore, observationContext *observation.Context) dbworkerstore.Store {
	return dbworkerstore.NewWithMetrics(s.Handle(), indexWorkerStoreOptions, observationContext)
}

// StalledDependencyIndexingJobMaxAge is the maximum allowable duration between updating
// the state of a dependency indexing job as "processing" and locking the job row during
// processing. An unlocked row that is marked as processing likely indicates that the worker
// that dequeued the job has died. There should be a nearly-zero delay between these states
// during normal operation.
const StalledDependencyIndexingJobMaxAge = time.Second * 5

// DependencyIndexingJobMaxNumResets is the maximum number of times a dependency indexing
// job can be reset. If an job's failed attempts counter reaches this threshold, it will be
// moved into "errored" rather than "queued" on its next reset.
const DependencyIndexingJobMaxNumResets = 3

var dependencyIndexingJobWorkerStoreOptions = dbworkerstore.Options{
	Name:              "precise_code_intel_dependency_indexing_scheduler_worker_store",
	TableName:         "lsif_dependency_indexing_jobs j",
	ColumnExpressions: dependencyIndexingJobColumns,
	Scan:              scanFirstDependencyIndexingJobRecord,
	OrderByExpression: sqlf.Sprintf("j.queued_at, j.upload_id"),
	StalledMaxAge:     StalledDependencyIndexingJobMaxAge,
	MaxNumResets:      DependencyIndexingJobMaxNumResets,
}

func WorkerutilDependencyIndexingJobStore(s basestore.ShareableStore, observationContext *observation.Context) dbworkerstore.Store {
	return dbworkerstore.NewWithMetrics(s.Handle(), dependencyIndexingJobWorkerStoreOptions, observationContext)
}

// StalledDependencyRepoAddingJobMaxAge is the maximum allowable duration between updating
// the state of a dependency repo adding job as "processing" and locking the job row during
// processing. An unlocked row that is marked as processing likely indicates that the worker
// that dequeued the job has died. There should be a nearly-zero delay between these states
// during normal operation.
const StalledDependencyRepoAddingJobMaxAge = time.Second * 5

// DependencyRepoAddingJobMaxNumResets is the maximum number of times a dependency repo adding
// job can be reset. If an job's failed attempts counter reaches this threshold, it will be
// moved into "errored" rather than "queued" on its next reset.
const DependencyRepoAddingJobMaxNumResets = 3

var dependencyRepoAddingJobWorkerStoreOptions = dbworkerstore.Options{
	Name:              "dependency_repo_adding_scheduler_worker_store",
	TableName:         "codeintel_dependency_repo_adding_jobs j",
	ColumnExpressions: dependencyRepoAddingJobColumns,
	Scan:              scanFirstDependencyRepoAddingJobRecord,
	OrderByExpression: sqlf.Sprintf("j.upload_id"),
	StalledMaxAge:     StalledDependencyRepoAddingJobMaxAge,
	MaxNumResets:      DependencyRepoAddingJobMaxNumResets,
}

func WorkerutilDependencyRepoAddingJobStore(s basestore.ShareableStore, observationContext *observation.Context) dbworkerstore.Store {
	return dbworkerstore.NewWithMetrics(s.Handle(), dependencyRepoAddingJobWorkerStoreOptions, observationContext)
}
