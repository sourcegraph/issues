package dbstore

import (
	"context"
	"database/sql"
	"time"

	"github.com/keegancsmith/sqlf"
	"github.com/opentracing/opentracing-go/log"

	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/commitgraph"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/gitserver"
	"github.com/sourcegraph/sourcegraph/internal/database/basestore"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
	"github.com/sourcegraph/sourcegraph/internal/observation"
)

// Dump is a subset of the lsif_uploads table (queried via the lsif_dumps_with_repository_name view)
// and stores only processed records.
type Dump struct {
	ID                int        `json:"id"`
	Commit            string     `json:"commit"`
	Root              string     `json:"root"`
	VisibleAtTip      bool       `json:"visibleAtTip"`
	UploadedAt        time.Time  `json:"uploadedAt"`
	State             string     `json:"state"`
	FailureMessage    *string    `json:"failureMessage"`
	StartedAt         *time.Time `json:"startedAt"`
	FinishedAt        *time.Time `json:"finishedAt"`
	ProcessAfter      *time.Time `json:"processAfter"`
	NumResets         int        `json:"numResets"`
	NumFailures       int        `json:"numFailures"`
	RepositoryID      int        `json:"repositoryId"`
	RepositoryName    string     `json:"repositoryName"`
	Indexer           string     `json:"indexer"`
	AssociatedIndexID *int       `json:"associatedIndex"`
}

// scanDumps scans a slice of dumps from the return value of `*Store.query`.
func scanDumps(rows *sql.Rows, queryErr error) (_ []Dump, err error) {
	if queryErr != nil {
		return nil, queryErr
	}
	defer func() { err = basestore.CloseRows(rows, err) }()

	var dumps []Dump
	for rows.Next() {
		var dump Dump
		if err := rows.Scan(
			&dump.ID,
			&dump.Commit,
			&dump.Root,
			&dump.VisibleAtTip,
			&dump.UploadedAt,
			&dump.State,
			&dump.FailureMessage,
			&dump.StartedAt,
			&dump.FinishedAt,
			&dump.ProcessAfter,
			&dump.NumResets,
			&dump.NumFailures,
			&dump.RepositoryID,
			&dump.RepositoryName,
			&dump.Indexer,
			&dump.AssociatedIndexID,
		); err != nil {
			return nil, err
		}

		dumps = append(dumps, dump)
	}

	return dumps, nil
}

// GetDumpsByIDs returns a set of dumps by identifiers.
func (s *Store) GetDumpsByIDs(ctx context.Context, ids []int) (_ []Dump, err error) {
	ctx, traceLog, endObservation := s.operations.getDumpsByIDs.WithAndLogger(ctx, &err, observation.Args{LogFields: []log.Field{
		log.Int("numIDs", len(ids)),
		log.String("ids", intsToString(ids)),
	}})
	defer endObservation(1, observation.Args{})

	if len(ids) == 0 {
		return nil, nil
	}

	var idx []*sqlf.Query
	for _, id := range ids {
		idx = append(idx, sqlf.Sprintf("%s", id))
	}

	dumps, err := scanDumps(s.Store.Query(ctx, sqlf.Sprintf(getDumpsByIDsQuery, sqlf.Join(idx, ", "))))
	if err != nil {
		return nil, err
	}
	traceLog(log.Int("numDumps", len(dumps)))

	return dumps, nil
}

const getDumpsByIDsQuery = `
-- source: enterprise/internal/codeintel/stores/dbstore/dumps.go:GetDumpsByIDs
SELECT
	u.id,
	u.commit,
	u.root,
	EXISTS (` + visibleAtTipSubselectQuery + `) AS visible_at_tip,
	u.uploaded_at,
	u.state,
	u.failure_message,
	u.started_at,
	u.finished_at,
	u.process_after,
	u.num_resets,
	u.num_failures,
	u.repository_id,
	u.repository_name,
	u.indexer,
	u.associated_index_id
FROM lsif_dumps_with_repository_name u WHERE u.id IN (%s)
`

// FindClosestDumps returns the set of dumps that can most accurately answer queries for the given repository, commit, path, and
// optional indexer. If rootMustEnclosePath is true, then only dumps with a root which is a prefix of path are returned. Otherwise,
// any dump with a root intersecting the given path is returned.
//
// This method should be used when the commit is known to exist in the lsif_nearest_uploads table. If it doesn't, then this method
// will return no dumps (as the input commit is not reachable from anything with an upload). The nearest uploads table must be
// refreshed before calling this method when the commit is unknown.
//
// Because refreshing the commit graph can be very expensive, we also provide FindClosestDumpsFromGraphFragment. That method should
// be used instead in low-latency paths. It should be supplied a small fragment of the commit graph that contains the input commit
// as well as a commit that is likely to exist in the lsif_nearest_uploads table. This is enough to propagate the correct upload
// visibility data down the graph fragment.
//
// The graph supplied to FindClosestDumpsFromGraphFragment will also determine whether or not it is possible to produce a partial set
// of visible uploads (ideally, we'd like to return the complete set of visible uploads, or fail). If the graph fragment is complete
// by depth (e.g. if the graph contains an ancestor at depth d, then the graph also contains all other ancestors up to depth d), then
// we get the ideal behavior. Only if we contain a partial row of ancestors will we return partial results.
func (s *Store) FindClosestDumps(ctx context.Context, repositoryID int, commit, path string, rootMustEnclosePath bool, indexer string) (_ []Dump, err error) {
	ctx, traceLog, endObservation := s.operations.findClosestDumps.WithAndLogger(ctx, &err, observation.Args{
		LogFields: []log.Field{
			log.Int("repositoryID", repositoryID),
			log.String("commit", commit),
			log.String("path", path),
			log.Bool("rootMustEnclosePath", rootMustEnclosePath),
			log.String("indexer", indexer),
		},
	})
	defer endObservation(1, observation.Args{})

	conds := makeFindClosestDumpConditions(path, rootMustEnclosePath, indexer)
	query := sqlf.Sprintf(findClosestDumpsQuery, makeVisibleUploadsQuery(repositoryID, commit), sqlf.Join(conds, " AND "))

	dumps, err := scanDumps(s.Store.Query(ctx, query))
	if err != nil {
		return nil, err
	}
	traceLog(log.Int("numDumps", len(dumps)))

	return dumps, nil
}

const findClosestDumpsQuery = `
-- source: enterprise/internal/codeintel/stores/dbstore/dumps.go:FindClosestDumps
WITH
visible_uploads AS (%s)
SELECT
	u.id,
	u.commit,
	u.root,
	EXISTS (` + visibleAtTipSubselectQuery + `) AS visible_at_tip,
	u.uploaded_at,
	u.state,
	u.failure_message,
	u.started_at,
	u.finished_at,
	u.process_after,
	u.num_resets,
	u.num_failures,
	u.repository_id,
	u.repository_name,
	u.indexer,
	u.associated_index_id
FROM visible_uploads vu
JOIN lsif_dumps_with_repository_name u ON u.id = vu.upload_id
WHERE %s
`

// FindClosestDumpsFromGraphFragment returns the set of dumps that can most accurately answer queries for the given repository, commit,
// path, and optional indexer by only considering the given fragment of the full git graph. See FindClosestDumps for additional details.
func (s *Store) FindClosestDumpsFromGraphFragment(ctx context.Context, repositoryID int, commit, path string, rootMustEnclosePath bool, indexer string, commitGraph *gitserver.CommitGraph) (_ []Dump, err error) {
	ctx, traceLog, endObservation := s.operations.findClosestDumpsFromGraphFragment.WithAndLogger(ctx, &err, observation.Args{
		LogFields: []log.Field{
			log.Int("repositoryID", repositoryID),
			log.String("commit", commit),
			log.String("path", path),
			log.Bool("rootMustEnclosePath", rootMustEnclosePath),
			log.String("indexer", indexer),
			log.Int("numCommitGraphKeys", len(commitGraph.Order())),
		},
	})
	defer endObservation(1, observation.Args{})

	if len(commitGraph.Order()) == 0 {
		return nil, nil
	}

	commits := make([]string, 0, len(commitGraph.Graph()))
	for commit := range commitGraph.Graph() {
		commits = append(commits, commit)
	}

	commitGraphView, err := scanCommitGraphView(s.Store.Query(ctx, sqlf.Sprintf(
		findClosestDumpsFromGraphFragmentCommitGraphQuery,
		makeVisibleUploadCandidatesQuery(repositoryID, commits...)),
	))
	if err != nil {
		return nil, err
	}
	traceLog(
		log.Int("numCommitGraphViewMetaKeys", len(commitGraphView.Meta)),
		log.Int("numCommitGraphViewTokenKeys", len(commitGraphView.Tokens)),
	)

	var ids []*sqlf.Query
	for _, uploadMeta := range commitgraph.NewGraph(commitGraph, commitGraphView).UploadsVisibleAtCommit(commit) {
		ids = append(ids, sqlf.Sprintf("%d", uploadMeta.UploadID))
	}
	if len(ids) == 0 {
		return nil, nil
	}

	conds := makeFindClosestDumpConditions(path, rootMustEnclosePath, indexer)
	query := sqlf.Sprintf(findClosestDumpsFromGraphFragmentQuery, sqlf.Join(ids, ","), sqlf.Join(conds, " AND "))

	dumps, err := scanDumps(s.Store.Query(ctx, query))
	if err != nil {
		return nil, err
	}
	traceLog(log.Int("numDumps", len(dumps)))

	return dumps, nil
}

const findClosestDumpsFromGraphFragmentCommitGraphQuery = `
-- source: enterprise/internal/codeintel/stores/dbstore/dumps.go:FindClosestDumpsFromGraphFragment
WITH
visible_uploads AS (%s)
SELECT
	vu.upload_id,
	encode(vu.commit_bytea, 'hex'),
	md5(u.root || ':' || u.indexer) as token,
	vu.distance
FROM visible_uploads vu
JOIN lsif_uploads u ON u.id = vu.upload_id
`

const findClosestDumpsFromGraphFragmentQuery = `
-- source: enterprise/internal/codeintel/stores/dbstore/dumps.go:FindClosestDumpsFromGraphFragment
SELECT
	u.id,
	u.commit,
	u.root,
	EXISTS (` + visibleAtTipSubselectQuery + `) AS visible_at_tip,
	u.uploaded_at,
	u.state,
	u.failure_message,
	u.started_at,
	u.finished_at,
	u.process_after,
	u.num_resets,
	u.num_failures,
	u.repository_id,
	u.repository_name,
	u.indexer,
	u.associated_index_id
FROM lsif_dumps_with_repository_name u
WHERE u.id IN (%s) AND %s
`

// makeVisibleUploadCandidatesQuery returns a SQL query returning the set of uploads
// visible from the given commits. This is done by looking at each commit's row in the
// lsif_nearest_uploads, and the (adjusted) set of uploads visible from each commit's
// nearest ancestor according to data compressed in the links table.
//
// NB: A commit should be present in at most one of these tables.
func makeVisibleUploadCandidatesQuery(repositoryID int, commits ...string) *sqlf.Query {
	if len(commits) == 0 {
		panic("No commits supplied to makeVisibleUploadCandidatesQuery.")
	}

	commitQueries := make([]*sqlf.Query, 0, len(commits))
	for _, commit := range commits {
		commitQueries = append(commitQueries, sqlf.Sprintf("%s", dbutil.CommitBytea(commit)))
	}

	return sqlf.Sprintf(visibleUploadCandidatesQuery, repositoryID, sqlf.Join(commitQueries, ", "), repositoryID, sqlf.Join(commitQueries, ", "))
}

const visibleUploadCandidatesQuery = `
-- source: enterprise/internal/codeintel/stores/dbstore/dumps.go:makeVisibleUploadCandidatesQuery
SELECT
	nu.repository_id,
	upload_id::integer,
	nu.commit_bytea,
	u_distance::text::integer as distance
FROM lsif_nearest_uploads nu
CROSS JOIN jsonb_each(nu.uploads) as u(upload_id, u_distance)
WHERE nu.repository_id = %s AND nu.commit_bytea IN (%s)
UNION (
	SELECT
		nu.repository_id,
		upload_id::integer,
		ul.commit_bytea,
		u_distance::text::integer + ul.distance as distance
	FROM lsif_nearest_uploads_links ul
	JOIN lsif_nearest_uploads nu ON nu.repository_id = ul.repository_id AND nu.commit_bytea = ul.ancestor_commit_bytea
	CROSS JOIN jsonb_each(nu.uploads) as u(upload_id, u_distance)
	WHERE nu.repository_id = %s AND ul.commit_bytea IN (%s)
)
`

// makeVisibleUploadsQuery returns a SQL query returning the set of identifiers of uploads
// visible from the given commit. This is done by removing the "shadowed" values created
// by looking at a commit and it's ancestors visible commits.
func makeVisibleUploadsQuery(repositoryID int, commit string) *sqlf.Query {
	return sqlf.Sprintf(visibleUploadsQuery, makeVisibleUploadCandidatesQuery(repositoryID, commit))
}

const visibleUploadsQuery = `
-- source: enterprise/internal/codeintel/stores/dbstore/dumps.go:makeVisibleUploadsQuery
SELECT
	t.upload_id
FROM (
	SELECT
		t.*,
		row_number() OVER (PARTITION BY root, indexer ORDER BY distance) AS r
	FROM (%s) t
	JOIN lsif_uploads u ON u.id = upload_id
) t
WHERE t.r <= 1
`

func makeFindClosestDumpConditions(path string, rootMustEnclosePath bool, indexer string) (conds []*sqlf.Query) {
	if rootMustEnclosePath {
		// Ensure that the root is a prefix of the path
		conds = append(conds, sqlf.Sprintf(`%s LIKE (u.root || '%%%%')`, path))
	} else {
		// Ensure that the root is a prefix of the path or vice versa
		conds = append(conds, sqlf.Sprintf(`(%s LIKE (u.root || '%%%%') OR u.root LIKE (%s || '%%%%'))`, path, path))
	}
	if indexer != "" {
		conds = append(conds, sqlf.Sprintf("indexer = %s", indexer))
	}

	return conds
}

// DeleteOverlapapingDumps deletes all completed uploads for the given repository with the same
// commit, root, and indexer. This is necessary to perform during conversions before changing
// the state of a processing upload to completed as there is a unique index on these four columns.
func (s *Store) DeleteOverlappingDumps(ctx context.Context, repositoryID int, commit, root, indexer string) (err error) {
	ctx, traceLog, endObservation := s.operations.deleteOverlappingDumps.WithAndLogger(ctx, &err, observation.Args{LogFields: []log.Field{
		log.Int("repositoryID", repositoryID),
		log.String("commit", commit),
		log.String("root", root),
		log.String("indexer", indexer),
	}})
	defer endObservation(1, observation.Args{})

	count, _, err := basestore.ScanFirstInt(s.Store.Query(ctx, sqlf.Sprintf(deleteOverlappingDumpsQuery, repositoryID, commit, root, indexer)))
	if err != nil {
		return err
	}
	traceLog(log.Int("count", count))

	return nil
}

const deleteOverlappingDumpsQuery = `
-- source: enterprise/internal/codeintel/stores/dbstore/dumps.go:DeleteOverlappingDumps
WITH
candidates AS (
	SELECT u.id
	FROM lsif_uploads u
	WHERE
		u.state = 'completed' AND
		u.repository_id = %s AND
		u.commit = %s AND
		u.root = %s AND
		u.indexer = %s

	-- Lock these rows in a deterministic order so that we don't
	-- deadlock with other processes updating the lsif_uploads table.
	ORDER BY u.id FOR UPDATE
),
updated AS (
	UPDATE lsif_uploads u
	SET state = 'deleted'
	WHERE id IN (SELECT id FROM candidates)
	RETURNING 1
)
SELECT COUNT(*) FROM updated
`
