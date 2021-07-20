package dbstore

import (
	"context"
	"database/sql"
	"strconv"
	"time"

	"github.com/keegancsmith/sqlf"
	"github.com/lib/pq"
	"github.com/opentracing/opentracing-go/log"

	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/database/basestore"
	"github.com/sourcegraph/sourcegraph/internal/observation"
	"github.com/sourcegraph/sourcegraph/internal/workerutil"
	dbworkerstore "github.com/sourcegraph/sourcegraph/internal/workerutil/dbworker/store"
)

// Index is a subset of the lsif_indexes table and stores both processed and unprocessed
// records.
type Index struct {
	ID                 int                            `json:"id"`
	Commit             string                         `json:"commit"`
	QueuedAt           time.Time                      `json:"queuedAt"`
	State              string                         `json:"state"`
	FailureMessage     *string                        `json:"failureMessage"`
	StartedAt          *time.Time                     `json:"startedAt"`
	FinishedAt         *time.Time                     `json:"finishedAt"`
	ProcessAfter       *time.Time                     `json:"processAfter"`
	NumResets          int                            `json:"numResets"`
	NumFailures        int                            `json:"numFailures"`
	RepositoryID       int                            `json:"repositoryId"`
	LocalSteps         []string                       `json:"local_steps"`
	RepositoryName     string                         `json:"repositoryName"`
	DockerSteps        []DockerStep                   `json:"docker_steps"`
	Root               string                         `json:"root"`
	Indexer            string                         `json:"indexer"`
	IndexerArgs        []string                       `json:"indexer_args"` // TODO - convert this to `IndexCommand string`
	Outfile            string                         `json:"outfile"`
	ExecutionLogs      []workerutil.ExecutionLogEntry `json:"execution_logs"`
	Rank               *int                           `json:"placeInQueue"`
	AssociatedUploadID *int                           `json:"associatedUpload"`
}

func (i Index) RecordID() int {
	return i.ID
}

// scanIndexes scans a slice of indexes from the return value of `*Store.query`.
func scanIndexes(rows *sql.Rows, queryErr error) (_ []Index, err error) {
	if queryErr != nil {
		return nil, queryErr
	}
	defer func() { err = basestore.CloseRows(rows, err) }()

	var indexes []Index
	for rows.Next() {
		var index Index
		var executionLogs []dbworkerstore.ExecutionLogEntry

		if err := rows.Scan(
			&index.ID,
			&index.Commit,
			&index.QueuedAt,
			&index.State,
			&index.FailureMessage,
			&index.StartedAt,
			&index.FinishedAt,
			&index.ProcessAfter,
			&index.NumResets,
			&index.NumFailures,
			&index.RepositoryID,
			&index.RepositoryName,
			pq.Array(&index.DockerSteps),
			&index.Root,
			&index.Indexer,
			pq.Array(&index.IndexerArgs),
			&index.Outfile,
			pq.Array(&executionLogs),
			&index.Rank,
			pq.Array(&index.LocalSteps),
			&index.AssociatedUploadID,
		); err != nil {
			return nil, err
		}

		for _, entry := range executionLogs {
			index.ExecutionLogs = append(index.ExecutionLogs, workerutil.ExecutionLogEntry(entry))
		}

		indexes = append(indexes, index)
	}

	return indexes, nil
}

// scanFirstIndex scans a slice of indexes from the return value of `*Store.query` and returns the first.
func scanFirstIndex(rows *sql.Rows, err error) (Index, bool, error) {
	indexes, err := scanIndexes(rows, err)
	if err != nil || len(indexes) == 0 {
		return Index{}, false, err
	}
	return indexes[0], true, nil
}

// scanFirstIndexInterface scans a slice of indexes from the return value of `*Store.query` and returns the first.
func scanFirstIndexRecord(rows *sql.Rows, err error) (workerutil.Record, bool, error) {
	return scanFirstIndex(rows, err)
}

var ScanFirstIndexRecord = scanFirstIndexRecord

// GetIndexByID returns an index by its identifier and boolean flag indicating its existence.
func (s *Store) GetIndexByID(ctx context.Context, id int) (_ Index, _ bool, err error) {
	ctx, endObservation := s.operations.getIndexByID.With(ctx, &err, observation.Args{LogFields: []log.Field{
		log.Int("id", id),
	}})
	defer endObservation(1, observation.Args{})

	authzConds, err := database.AuthzQueryConds(ctx, s.Store.Handle().DB())
	if err != nil {
		return Index{}, false, err
	}

	return scanFirstIndex(s.Store.Query(ctx, sqlf.Sprintf(getIndexByIDQuery, id, authzConds)))
}

const indexAssociatedUploadIDQueryFragment = `
(
	SELECT MAX(id) FROM lsif_uploads WHERE associated_index_id = u.id
) AS associated_upload_id
`

const indexRankQueryFragment = `
SELECT
	r.id,
	ROW_NUMBER() OVER (ORDER BY COALESCE(r.process_after, r.queued_at), r.id) as rank
FROM lsif_indexes_with_repository_name r
WHERE r.state = 'queued'
`

const getIndexByIDQuery = `
-- source: enterprise/internal/codeintel/stores/dbstore/indexes.go:GetIndexByID
SELECT
	u.id,
	u.commit,
	u.queued_at,
	u.state,
	u.failure_message,
	u.started_at,
	u.finished_at,
	u.process_after,
	u.num_resets,
	u.num_failures,
	u.repository_id,
	u.repository_name,
	u.docker_steps,
	u.root,
	u.indexer,
	u.indexer_args,
	u.outfile,
	u.execution_logs,
	s.rank,
	u.local_steps,
	` + indexAssociatedUploadIDQueryFragment + `
FROM lsif_indexes_with_repository_name u
LEFT JOIN (` + indexRankQueryFragment + `) s
ON u.id = s.id
JOIN repo ON repo.id = u.repository_id
WHERE u.id = %s AND %s
`

// GetIndexesByIDs returns an index for each of the given identifiers. Not all given ids will necessarily
// have a corresponding element in the returned list.
func (s *Store) GetIndexesByIDs(ctx context.Context, ids ...int) (_ []Index, err error) {
	ctx, endObservation := s.operations.getIndexesByIDs.With(ctx, &err, observation.Args{LogFields: []log.Field{
		log.String("ids", intsToString(ids)),
	}})
	defer endObservation(1, observation.Args{})

	if len(ids) == 0 {
		return nil, nil
	}

	authzConds, err := database.AuthzQueryConds(ctx, s.Store.Handle().DB())
	if err != nil {
		return nil, err
	}

	queries := make([]*sqlf.Query, 0, len(ids))
	for _, id := range ids {
		queries = append(queries, sqlf.Sprintf("%d", id))
	}

	return scanIndexes(s.Store.Query(ctx, sqlf.Sprintf(getIndexesByIDsQuery, sqlf.Join(queries, ", "), authzConds)))
}

const getIndexesByIDsQuery = `
-- source: enterprise/internal/codeintel/stores/dbstore/indexes.go:GetIndexesByIDs
SELECT
	u.id,
	u.commit,
	u.queued_at,
	u.state,
	u.failure_message,
	u.started_at,
	u.finished_at,
	u.process_after,
	u.num_resets,
	u.num_failures,
	u.repository_id,
	u.repository_name,
	u.docker_steps,
	u.root,
	u.indexer,
	u.indexer_args,
	u.outfile,
	u.execution_logs,
	s.rank,
	u.local_steps,
	` + indexAssociatedUploadIDQueryFragment + `
FROM lsif_indexes_with_repository_name u
LEFT JOIN (` + indexRankQueryFragment + `) s
ON u.id = s.id
JOIN repo ON repo.id = u.repository_id
WHERE u.id IN (%s) AND %s
`

type GetIndexesOptions struct {
	RepositoryID int
	State        string
	Term         string
	Limit        int
	Offset       int
}

// GetIndexes returns a list of indexes and the total count of records matching the given conditions.
func (s *Store) GetIndexes(ctx context.Context, opts GetIndexesOptions) (_ []Index, _ int, err error) {
	ctx, traceLog, endObservation := s.operations.getIndexes.WithAndLogger(ctx, &err, observation.Args{LogFields: []log.Field{
		log.Int("repositoryID", opts.RepositoryID),
		log.String("state", opts.State),
		log.String("term", opts.Term),
		log.Int("limit", opts.Limit),
		log.Int("offset", opts.Offset),
	}})
	defer endObservation(1, observation.Args{})

	tx, err := s.transact(ctx)
	if err != nil {
		return nil, 0, err
	}
	defer func() { err = tx.Done(err) }()

	var conds []*sqlf.Query

	if opts.RepositoryID != 0 {
		conds = append(conds, sqlf.Sprintf("u.repository_id = %s", opts.RepositoryID))
	}
	if opts.Term != "" {
		conds = append(conds, makeIndexSearchCondition(opts.Term))
	}
	if opts.State != "" {
		conds = append(conds, sqlf.Sprintf("u.state = %s", opts.State))
	}

	authzConds, err := database.AuthzQueryConds(ctx, tx.Store.Handle().DB())
	if err != nil {
		return nil, 0, err
	}
	conds = append(conds, authzConds)

	totalCount, _, err := basestore.ScanFirstInt(tx.Store.Query(ctx, sqlf.Sprintf(getIndexesCountQuery, sqlf.Join(conds, " AND "))))
	if err != nil {
		return nil, 0, err
	}

	indexes, err := scanIndexes(tx.Store.Query(ctx, sqlf.Sprintf(getIndexesQuery, sqlf.Join(conds, " AND "), opts.Limit, opts.Offset)))
	if err != nil {
		return nil, 0, err
	}
	traceLog(
		log.Int("totalCount", totalCount),
		log.Int("numIndexes", len(indexes)),
	)

	return indexes, totalCount, nil
}

const getIndexesCountQuery = `
-- source: enterprise/internal/codeintel/stores/dbstore/indexes.go:GetIndexes
SELECT COUNT(*)
FROM lsif_indexes_with_repository_name u
JOIN repo ON repo.id = u.repository_id
WHERE %s
`

const getIndexesQuery = `
-- source: enterprise/internal/codeintel/stores/dbstore/indexes.go:GetIndexes
SELECT
	u.id,
	u.commit,
	u.queued_at,
	u.state,
	u.failure_message,
	u.started_at,
	u.finished_at,
	u.process_after,
	u.num_resets,
	u.num_failures,
	u.repository_id,
	u.repository_name,
	u.docker_steps,
	u.root,
	u.indexer,
	u.indexer_args,
	u.outfile,
	u.execution_logs,
	s.rank,
	u.local_steps,
	` + indexAssociatedUploadIDQueryFragment + `
FROM lsif_indexes_with_repository_name u
LEFT JOIN (` + indexRankQueryFragment + `) s
ON u.id = s.id
JOIN repo ON repo.id = u.repository_id
WHERE %s ORDER BY queued_at DESC LIMIT %d OFFSET %d
`

// makeIndexSearchCondition returns a disjunction of LIKE clauses against all searchable columns of an index.
func makeIndexSearchCondition(term string) *sqlf.Query {
	searchableColumns := []string{
		"u.commit",
		"(u.state)::text",
		"u.failure_message",
		`u.repository_name`,
		"u.root",
		"u.indexer",
	}

	var termConds []*sqlf.Query
	for _, column := range searchableColumns {
		termConds = append(termConds, sqlf.Sprintf(column+" ILIKE %s", "%"+term+"%"))
	}

	return sqlf.Sprintf("(%s)", sqlf.Join(termConds, " OR "))
}

// IsQueued returns true if there is an index or an upload for the repository and commit.
func (s *Store) IsQueued(ctx context.Context, repositoryID int, commit string) (_ bool, err error) {
	ctx, endObservation := s.operations.isQueued.With(ctx, &err, observation.Args{LogFields: []log.Field{
		log.Int("repositoryID", repositoryID),
		log.String("commit", commit),
	}})
	defer endObservation(1, observation.Args{})

	count, _, err := basestore.ScanFirstInt(s.Store.Query(ctx, sqlf.Sprintf(isQueuedQuery, repositoryID, commit, repositoryID, commit)))
	return count > 0, err
}

const isQueuedQuery = `
-- source: enterprise/internal/codeintel/stores/dbstore/indexes.go:IsQueued
SELECT COUNT(*) WHERE EXISTS (
	SELECT id FROM lsif_uploads_with_repository_name WHERE state != 'deleted' AND repository_id = %s AND commit = %s
	UNION
	SELECT id FROM lsif_indexes_with_repository_name WHERE repository_id = %s AND commit = %s
)
`

// InsertIndex inserts a new index and returns its identifier.
func (s *Store) InsertIndex(ctx context.Context, index Index) (id int, err error) {
	ctx, endObservation := s.operations.insertIndex.With(ctx, &err, observation.Args{})
	defer func() {
		endObservation(1, observation.Args{LogFields: []log.Field{
			log.Int("id", id),
		}})
	}()

	if index.DockerSteps == nil {
		index.DockerSteps = []DockerStep{}
	}
	if index.IndexerArgs == nil {
		index.IndexerArgs = []string{}
	}
	if index.LocalSteps == nil {
		index.LocalSteps = []string{}
	}

	id, _, err = basestore.ScanFirstInt(s.Store.Query(
		ctx,
		sqlf.Sprintf(
			insertIndexQuery,
			index.State,
			index.Commit,
			index.RepositoryID,
			pq.Array(index.DockerSteps),
			pq.Array(index.LocalSteps),
			index.Root,
			index.Indexer,
			pq.Array(index.IndexerArgs),
			index.Outfile,
			pq.Array(dbworkerstore.ExecutionLogEntries(index.ExecutionLogs)),
		),
	))

	return id, err
}

const insertIndexQuery = `
-- source: enterprise/internal/codeintel/stores/dbstore/indexes.go:InsertIndex
INSERT INTO lsif_indexes (
	state,
	commit,
	repository_id,
	docker_steps,
	local_steps,
	root,
	indexer,
	indexer_args,
	outfile,
	execution_logs
) VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
RETURNING id
`

var indexColumnsWithNullRank = []*sqlf.Query{
	sqlf.Sprintf("u.id"),
	sqlf.Sprintf("u.commit"),
	sqlf.Sprintf("u.queued_at"),
	sqlf.Sprintf("u.state"),
	sqlf.Sprintf("u.failure_message"),
	sqlf.Sprintf("u.started_at"),
	sqlf.Sprintf("u.finished_at"),
	sqlf.Sprintf("u.process_after"),
	sqlf.Sprintf("u.num_resets"),
	sqlf.Sprintf("u.num_failures"),
	sqlf.Sprintf("u.repository_id"),
	sqlf.Sprintf(`u.repository_name`),
	sqlf.Sprintf(`u.docker_steps`),
	sqlf.Sprintf(`u.root`),
	sqlf.Sprintf(`u.indexer`),
	sqlf.Sprintf(`u.indexer_args`),
	sqlf.Sprintf(`u.outfile`),
	sqlf.Sprintf(`u.execution_logs`),
	sqlf.Sprintf("NULL"),
	sqlf.Sprintf(`u.local_steps`),
	sqlf.Sprintf(indexAssociatedUploadIDQueryFragment),
}

var IndexColumnsWithNullRank = indexColumnsWithNullRank

// DeleteIndexByID deletes an index by its identifier.
func (s *Store) DeleteIndexByID(ctx context.Context, id int) (_ bool, err error) {
	ctx, endObservation := s.operations.deleteIndexByID.With(ctx, &err, observation.Args{LogFields: []log.Field{
		log.Int("id", id),
	}})
	defer endObservation(1, observation.Args{})

	tx, err := s.transact(ctx)
	if err != nil {
		return false, err
	}
	defer func() { err = tx.Done(err) }()

	_, exists, err := basestore.ScanFirstInt(tx.Store.Query(ctx, sqlf.Sprintf(deleteIndexByIDQuery, id)))
	return exists, err
}

const deleteIndexByIDQuery = `
-- source: enterprise/internal/codeintel/stores/dbstore/indexes.go:DeleteIndexByID
DELETE FROM lsif_indexes WHERE id = %s RETURNING repository_id
`

// DeleteIndexesWithoutRepository deletes indexes associated with repositories that were deleted at least
// DeletedRepositoryGracePeriod ago. This returns the repository identifier mapped to the number of indexes
// that were removed for that repository.
func (s *Store) DeleteIndexesWithoutRepository(ctx context.Context, now time.Time) (_ map[int]int, err error) {
	ctx, traceLog, endObservation := s.operations.deleteIndexesWithoutRepository.WithAndLogger(ctx, &err, observation.Args{})
	defer endObservation(1, observation.Args{})

	// TODO(efritz) - this would benefit from an index on repository_id. We currently have
	// a similar one on this index, but only for uploads that are completed or visible at tip.

	repositories, err := scanCounts(s.Store.Query(ctx, sqlf.Sprintf(deleteIndexesWithoutRepositoryQuery, now.UTC(), DeletedRepositoryGracePeriod/time.Second)))
	if err != nil {
		return nil, err
	}

	count := 0
	for _, numDeleted := range repositories {
		count += numDeleted
	}
	traceLog(
		log.Int("count", count),
		log.Int("numRepositories", len(repositories)),
	)

	return repositories, nil
}

const deleteIndexesWithoutRepositoryQuery = `
-- source: enterprise/internal/codeintel/stores/dbstore/indexes.go:DeleteIndexesWithoutRepository
WITH
candidates AS (
	SELECT u.id
	FROM repo r
	JOIN lsif_indexes u ON u.repository_id = r.id
	WHERE %s - r.deleted_at >= %s * interval '1 second'

	-- Lock these rows in a deterministic order so that we don't
	-- deadlock with other processes updating the lsif_indexes table.
	ORDER BY u.id FOR UPDATE
),
deleted AS (
	DELETE FROM lsif_indexes u
	WHERE id IN (SELECT id FROM candidates)
	RETURNING u.id, u.repository_id
)
SELECT d.repository_id, COUNT(*) FROM deleted d GROUP BY d.repository_id
`

// DeleteOldIndexes deletes indexes older than the given age.
func (s *Store) DeleteOldIndexes(ctx context.Context, maxAge time.Duration, now time.Time) (count int, err error) {
	ctx, traceLog, endObservation := s.operations.deleteOldIndexes.WithAndLogger(ctx, &err, observation.Args{LogFields: []log.Field{
		log.String("maxAge", maxAge.String()),
	}})
	defer endObservation(1, observation.Args{})

	tx, err := s.transact(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { err = tx.Done(err) }()

	repositoryIDs, err := scanCounts(tx.Store.Query(ctx, sqlf.Sprintf(deleteOldIndexesQuery, now, strconv.Itoa(int(maxAge/time.Second)))))
	if err != nil {
		return 0, err
	}

	for _, numDeleted := range repositoryIDs {
		count += numDeleted
	}
	traceLog(
		log.Int("count", count),
		log.Int("numRepositories", len(repositoryIDs)),
	)

	return count, nil
}

const deleteOldIndexesQuery = `
-- source: enterprise/internal/codeintel/stores/dbstore/indexes.go:DeleteOldIndexes
WITH
candidates AS (
	SELECT u.id
	FROM lsif_indexes u
	WHERE %s - u.queued_at > (%s || ' second')::interval

	-- Lock these rows in a deterministic order so that we don't
	-- deadlock with other processes updating the lsif_indexes table.
	ORDER BY u.id FOR UPDATE
),
deleted AS (
	DELETE FROM lsif_indexes u
	WHERE id IN (SELECT id FROM candidates)
	RETURNING u.id, u.repository_id
)
SELECT d.repository_id, COUNT(*) FROM deleted d GROUP BY d.repository_id
`
