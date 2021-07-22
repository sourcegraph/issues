package repos

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/hashicorp/go-multierror"
	"github.com/inconshreveable/log15"
	"github.com/keegancsmith/sqlf"
	"github.com/lib/pq"
	"github.com/opentracing/opentracing-go"
	otlog "github.com/opentracing/opentracing-go/log"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/database/basestore"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
	"github.com/sourcegraph/sourcegraph/internal/logging"
	"github.com/sourcegraph/sourcegraph/internal/trace"
	"github.com/sourcegraph/sourcegraph/internal/types"
)

// A Store exposes methods to read and write repos and external services.
type Store struct {
	*basestore.Store

	// Logger used by the store. Defaults to log15.Root().
	Log logging.ErrorLogger
	// Metrics are sent to Prometheus by default.
	Metrics StoreMetrics
	// Used for tracing calls to store methods. Uses opentracing.GlobalTracer() by default.
	Tracer trace.Tracer
	// RepoStore is a database.RepoStore using the same database handle.
	RepoStore *database.RepoStore
	// ExternalServiceStore is a database.ExternalServiceStore using the same database handle.
	ExternalServiceStore *database.ExternalServiceStore

	txtrace *trace.Trace
	txctx   context.Context
}

// NewStore instantiates and returns a new DBStore with prepared statements.
func NewStore(db dbutil.DB, txOpts sql.TxOptions) *Store {
	s := basestore.NewWithDB(db, txOpts)
	return &Store{
		Store:                s,
		RepoStore:            database.ReposWith(s),
		ExternalServiceStore: database.ExternalServicesWith(s),
		Log:                  log15.Root(),
		Tracer:               trace.Tracer{Tracer: opentracing.GlobalTracer()},
	}
}

func (s *Store) With(other basestore.ShareableStore) *Store {
	return &Store{
		Store:                s.Store.With(other),
		RepoStore:            s.RepoStore.With(other),
		ExternalServiceStore: s.ExternalServiceStore.With(other),
		Log:                  s.Log,
		Metrics:              s.Metrics,
		Tracer:               s.Tracer,
	}
}

// Transact returns a TxStore whose methods operate within the context of a transaction.
func (s *Store) Transact(ctx context.Context) (stx *Store, err error) {
	tr, ctx := s.trace(ctx, "Store.Transact")

	defer func(began time.Time) {
		secs := time.Since(began).Seconds()
		s.Metrics.Transact.Observe(secs, 1, &err)
		logging.Log(s.Log, "store.transact", &err)
		if err != nil {
			tr.SetError(err)
			// Finish is called in Done in the non-error case
			tr.Finish()
		}
	}(time.Now())

	txBase, err := s.Store.Transact(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "starting transaction")
	}
	return &Store{
		Store:                txBase,
		RepoStore:            s.RepoStore.With(txBase),
		ExternalServiceStore: s.ExternalServiceStore.With(txBase),
		Log:                  s.Log,
		Metrics:              s.Metrics,
		Tracer:               s.Tracer,
		txtrace:              tr,
		txctx:                ctx,
	}, nil
}

// Done calls into the inner Store Done method.
func (s *Store) Done(err error) error {
	tr := s.txtrace
	tr.LogFields(otlog.String("event", "Store.Done"))

	defer func(began time.Time) {
		secs := time.Since(began).Seconds()
		done := false

		if err != nil {
			done = true
			tr.SetError(err)
			s.Metrics.Done.Observe(secs, 1, &err)
			logging.Log(s.Log, "store.done", &err)
		}

		if !done {
			s.Metrics.Done.Observe(secs, 1, nil)
		}

		tr.Finish()
	}(time.Now())

	return s.Store.Done(err)
}

func (s *Store) trace(ctx context.Context, family string) (*trace.Trace, context.Context) {
	txctx := s.txctx
	if txctx == nil {
		txctx = ctx
	}
	tr, txctx := s.Tracer.New(txctx, family, "")
	ctx = trace.CopyContext(ctx, txctx)
	return tr, ctx
}

// CountUserAddedRepos counts the total number of repos that have been added
// by user owned external services. If userIDs are specified, only repos owned by the given
// users are counted.
func (s *Store) CountUserAddedRepos(ctx context.Context, userIDs ...int32) (count uint64, err error) {
	tr, ctx := s.trace(ctx, "Store.CountUserAddedRepos")
	defer func(began time.Time) {
		secs := time.Since(began).Seconds()

		uids := fmt.Sprint(userIDs)
		tr.LogFields(otlog.String("user-ids", uids))
		s.Metrics.CountUserAddedRepos.Observe(secs, float64(count), &err)
		logging.Log(s.Log, "store.count-user-added-repos", &err, "count", count, "user-ids", uids)

		tr.SetError(err)
		tr.Finish()
	}(time.Now())

	var q *sqlf.Query
	if len(userIDs) > 0 {
		q = sqlf.Sprintf(countTotalUserAddedReposQueryFmtstr+"\nAND user_id = ANY(%s)", pq.Array(userIDs))
	} else {
		q = sqlf.Sprintf(countTotalUserAddedReposQueryFmtstr)
	}

	err = s.QueryRow(ctx, q).Scan(&count)
	return count, err
}

const countTotalUserAddedReposQueryFmtstr = `
SELECT COUNT(DISTINCT(repo_id))
FROM external_service_repos
WHERE user_id IS NOT NULL`

// DeleteExternalServiceReposNotIn calls DeleteExternalServiceRepo for every repo not in the given ids that is owned
// by the given external service. We run one query per repo rather than one batch query in order to reduce the chances
// of this whole operation blocking on locks other queries acquire when referencing external_service_repos or repo.
// Since the syncer runs periodically, it's better to fail to delete some repos and try to delete them again in the
// next run, than to have one failure prevent all deletes from happening.
func (s *Store) DeleteExternalServiceReposNotIn(ctx context.Context, svc *types.ExternalService, ids map[api.RepoID]struct{}) (deleted []api.RepoID, err error) {
	tr, ctx := s.trace(ctx, "Store.DeleteExternalServiceReposNotIn")
	tr.LogFields(
		otlog.Int("len(ids)", len(ids)),
		otlog.Int64("external_service_id", svc.ID),
	)

	defer func(began time.Time) {
		secs := time.Since(began).Seconds()

		s.Metrics.DeleteExternalServiceReposNotIn.Observe(secs, 1, &err)
		logging.Log(s.Log, "store.delete-external-service-repos-not-in", &err, "external-service-id", svc.ID, "len(ids)", len(ids))

		tr.SetError(err)
		tr.Finish()
	}(time.Now())

	set := make(pq.Int64Array, 0, len(ids))
	for id := range ids {
		set = append(set, int64(id))
	}

	sort.Slice(set, func(a, b int) bool { return set[a] < set[b] })

	var toDelete pq.Int64Array
	if err = s.QueryRow(ctx, sqlf.Sprintf(listExternalServiceReposNotInQuery, svc.ID, set)).Scan(&toDelete); err != nil {
		return nil, errors.Wrap(err, "failed to list external service repo ids")
	}

	var errs multierror.Error
	for _, id := range toDelete {
		if err = s.DeleteExternalServiceRepo(ctx, svc, api.RepoID(id)); err != nil {
			multierror.Append(&errs, errors.Wrapf(err, "failed to delete external service repo (%d, %d)", svc.ID, id))
		} else {
			deleted = append(deleted, api.RepoID(id))
		}
	}

	return deleted, errs.ErrorOrNil()
}

const listExternalServiceReposNotInQuery = `
SELECT array_agg(repo_id)
FROM external_service_repos
WHERE external_service_id = %s AND repo_id != ALL(%s)
`

// DeleteExternalServiceRepo deletes a repo's association to an external service and the repo itself if there are no
// more associations to that repo by any other external service.
func (s *Store) DeleteExternalServiceRepo(ctx context.Context, svc *types.ExternalService, id api.RepoID) (err error) {
	tr, ctx := s.trace(ctx, "Store.DeleteExternalServiceRepo")
	tr.LogFields(
		otlog.Int32("id", int32(id)),
		otlog.Int64("external_service_id", svc.ID),
	)

	defer func(began time.Time) {
		secs := time.Since(began).Seconds()

		s.Metrics.DeleteExternalServiceRepo.Observe(secs, 1, &err)
		logging.Log(s.Log, "store.delete-external-service-repo", &err, "external-service-id", svc.ID, "repo-id", id)

		tr.SetError(err)
		tr.Finish()
	}(time.Now())

	if !s.InTransaction() {
		s, err = s.Transact(ctx)
		if err != nil {
			return err
		}
		defer func() { s.Done(err) }()
	}

	err = s.Exec(ctx, sqlf.Sprintf(deleteExternalServiceRepoQuery, svc.ID, id))
	if err != nil {
		return errors.Wrap(err, "failed to delete external service repo")
	}

	err = s.Exec(ctx, sqlf.Sprintf(deleteRepoIfOrphanQuery, id, id))
	if err != nil {
		return errors.Wrap(err, "failed to delete orphaned repo")
	}

	return nil
}

const deleteExternalServiceRepoQuery = `
DELETE FROM external_service_repos
WHERE external_service_id = %s AND repo_id = %s
`

const deleteRepoIfOrphanQuery = `
UPDATE repo
SET name = soft_deleted_repository_name(name), deleted_at = now()
WHERE id = %s AND NOT EXISTS (
	SELECT FROM external_service_repos
	WHERE repo_id = %s LIMIT 1
)
`

// CreateExternalServiceRepo inserts a single repo and its association to an external service, respectively in the repo and
// external_service_repos table. The associated external service must already exist.
func (s *Store) CreateExternalServiceRepo(ctx context.Context, svc *types.ExternalService, r *types.Repo) (err error) {
	tr, ctx := s.trace(ctx, "Store.CreateExternalServiceRepo")
	tr.LogFields(
		otlog.String("name", string(r.Name)),
		otlog.Int64("external_service_id", svc.ID),
		otlog.String("external_repo_spec", r.ExternalRepo.String()),
	)

	defer func(began time.Time) {
		secs := time.Since(began).Seconds()

		s.Metrics.CreateExternalServiceRepo.Observe(secs, 1, &err)
		logging.Log(s.Log, "store.create-external-service-repo", &err,
			"external-service-id", svc.ID,
			"name", r.Name,
			"external-repo-spec", r.ExternalRepo.String(),
		)

		tr.SetError(err)
		tr.Finish()
	}(time.Now())

	metadata, err := json.Marshal(r.Metadata)
	if err != nil {
		return err
	}

	q := sqlf.Sprintf(createRepoQuery,
		r.Name,
		r.URI,
		r.Description,
		r.ExternalRepo.ServiceType,
		r.ExternalRepo.ServiceID,
		r.ExternalRepo.ID,
		r.Archived,
		r.Fork,
		r.Stars,
		r.Private,
		metadata,
	)

	src := r.Sources[svc.URN()]
	if src == nil || src.CloneURL == "" {
		return errors.New("CreateExternalServiceRepo: repo missing source info for external service")
	}

	if !s.InTransaction() {
		s, err = s.Transact(ctx)
		if err != nil {
			return err
		}
		defer func() { s.Done(err) }()
	}

	if err = s.QueryRow(ctx, q).Scan(&r.ID, &r.CreatedAt); err != nil {
		return err
	}

	return s.Exec(ctx, sqlf.Sprintf(upsertExternalServiceRepoQuery,
		svc.ID,
		r.ID,
		svc.NamespaceUserID,
		src.CloneURL,
	))
}

const createRepoQuery = `
INSERT INTO repo (
	name,
	uri,
	description,
	external_service_type,
	external_service_id,
	external_id,
	archived,
	fork,
	stars,
	private,
	metadata,
	created_at
)
VALUES (%s, NULLIF(%s, ''), %s, %s, %s, %s, %s, %s, %s, %s, %s, now())
RETURNING id, created_at
`

const upsertExternalServiceRepoQuery = `
INSERT INTO external_service_repos (
	external_service_id,
	repo_id,
	user_id,
	clone_url
)
VALUES (%s, %s, NULLIF(%s, 0), %s)
ON CONFLICT (external_service_id, repo_id)
DO UPDATE SET
	clone_url = excluded.clone_url,
	user_id   = excluded.user_id
WHERE
	external_service_repos.clone_url != excluded.clone_url OR
	external_service_repos.user_id   != excluded.user_id
`

// UpdateExternalServiceRepo updates a single repo and its association to an external service, respectively in the repo and
// external_service_repos table. The associated external service must already exist.
func (s *Store) UpdateExternalServiceRepo(ctx context.Context, svc *types.ExternalService, r *types.Repo) (err error) {
	tr, ctx := s.trace(ctx, "Store.UpdateExternalServiceRepo")
	tr.LogFields(
		otlog.String("name", string(r.Name)),
		otlog.Int64("external_service_id", svc.ID),
		otlog.String("external_repo_spec", r.ExternalRepo.String()),
	)

	defer func(began time.Time) {
		secs := time.Since(began).Seconds()

		s.Metrics.UpdateExternalServiceRepo.Observe(secs, 1, &err)
		logging.Log(s.Log, "store.update-external-service-repo", &err,
			"external-service-id", svc.ID,
			"name", r.Name,
			"external-repo-spec", r.ExternalRepo.String(),
		)

		tr.SetError(err)
		tr.Finish()
	}(time.Now())

	if r.ID == 0 {
		return errors.New("empty repo id in update")
	}

	metadata, err := metadataColumn(r.Metadata)
	if err != nil {
		return errors.Wrapf(err, "metadata marshalling failed")
	}

	q := sqlf.Sprintf(updateRepoQuery,
		r.Name,
		r.URI,
		r.Description,
		r.ExternalRepo.ServiceType,
		r.ExternalRepo.ServiceID,
		r.ExternalRepo.ID,
		r.Archived,
		r.Fork,
		r.Stars,
		r.Private,
		metadata,
		r.ID,
	)

	src := r.Sources[svc.URN()]
	if src == nil || src.CloneURL == "" {
		return errors.New("UpdateExternalServiceRepo: repo missing source info for external service")
	}

	if !s.InTransaction() {
		s, err = s.Transact(ctx)
		if err != nil {
			return err
		}
		defer func() { s.Done(err) }()
	}

	if err = s.QueryRow(ctx, q).Scan(&r.UpdatedAt); err != nil {
		return err
	}

	return s.Exec(ctx, sqlf.Sprintf(upsertExternalServiceRepoQuery,
		svc.ID,
		r.ID,
		svc.NamespaceUserID,
		src.CloneURL,
	))
}

const updateRepoQuery = `
UPDATE repo
SET
	name                  = %s,
	uri                   = NULLIF(%s, ''),
	description           = %s,
	external_service_type = %s,
	external_service_id   = %s,
	external_id           = %s,
	archived              = %s,
	fork                  = %s,
	stars                 = %s,
	private               = %s,
	metadata              = %s,
	updated_at            = now(),
	deleted_at            = NULL
WHERE id = %s
RETURNING updated_at
`

// EnqueueSingleSyncJob enqueues a single sync job for the given external
// service if it is not already queued or processing.
func (s *Store) EnqueueSingleSyncJob(ctx context.Context, id int64) (err error) {
	q := sqlf.Sprintf(`
INSERT INTO external_service_sync_jobs (external_service_id)
SELECT %s
WHERE NOT EXISTS(
        SELECT 1
        FROM external_service_sync_jobs
        WHERE external_service_id = %s
          AND state IN ('queued', 'processing'))
`, id, id)
	return s.Exec(ctx, q)
}

// EnqueueSyncJobs enqueues sync jobs for all external services that are due.
func (s *Store) EnqueueSyncJobs(ctx context.Context, isCloud bool) (err error) {
	tr, ctx := s.trace(ctx, "Store.EnqueueSyncJobs")

	defer func(began time.Time) {
		secs := time.Since(began).Seconds()
		s.Metrics.EnqueueSyncJobs.Observe(secs, 0, &err)
		tr.SetError(err)
		tr.Finish()
	}(time.Now())

	filter := "TRUE"
	// On Cloud we don't sync our default sources in the background, they are synced
	// on demand instead.
	if isCloud {
		filter = "cloud_default = false"
	}
	q := sqlf.Sprintf(enqueueSyncJobsQueryFmtstr, sqlf.Sprintf(filter))
	return s.Exec(ctx, q)
}

// We ignore Phabricator repos here as they are currently synced using
// RunPhabricatorRepositorySyncWorker
const enqueueSyncJobsQueryFmtstr = `
WITH due AS (
    SELECT id
    FROM external_services
    WHERE (next_sync_at <= clock_timestamp() OR next_sync_at IS NULL)
    AND deleted_at IS NULL
    AND LOWER(kind) != 'phabricator'
    AND %s
),
busy AS (
    SELECT DISTINCT external_service_id id FROM external_service_sync_jobs
    WHERE state = 'queued'
    OR state = 'processing'
)
INSERT INTO external_service_sync_jobs (external_service_id)
SELECT id from due EXCEPT SELECT id from busy
`

// ListSyncJobs returns all sync jobs.
func (s *Store) ListSyncJobs(ctx context.Context) ([]SyncJob, error) {
	q := sqlf.Sprintf(`
		SELECT
			id,
			state,
			failure_message,
			started_at,
			finished_at,
			process_after,
			num_resets,
			num_failures,
			execution_logs,
			external_service_id,
			next_sync_at
		FROM external_service_sync_jobs_with_next_sync_at
	`)
	rows, err := s.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanJobs(rows)
}

func scanJobs(rows *sql.Rows) ([]SyncJob, error) {
	var jobs []SyncJob

	for rows.Next() {
		// required field for the sync worker, but
		// the value is thrown out here
		var executionLogs *[]interface{}

		var job SyncJob
		if err := rows.Scan(
			&job.ID,
			&job.State,
			&job.FailureMessage,
			&job.StartedAt,
			&job.FinishedAt,
			&job.ProcessAfter,
			&job.NumResets,
			&job.NumFailures,
			&executionLogs,
			&job.ExternalServiceID,
			&job.NextSyncAt,
		); err != nil {
			return nil, err
		}

		jobs = append(jobs, job)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return jobs, nil
}

func metadataColumn(metadata interface{}) (msg json.RawMessage, err error) {
	switch m := metadata.(type) {
	case nil:
		msg = json.RawMessage("{}")
	case string:
		msg = json.RawMessage(m)
	case []byte:
		msg = m
	case json.RawMessage:
		msg = m
	default:
		msg, err = json.MarshalIndent(m, "        ", "    ")
	}
	return
}
