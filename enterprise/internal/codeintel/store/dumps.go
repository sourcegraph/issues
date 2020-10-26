package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/keegancsmith/sqlf"
	"github.com/sourcegraph/sourcegraph/internal/db/basestore"
)

// Dump is a subset of the lsif_uploads table (queried via the lsif_dumps_with_repository_name view)
// and stores only processed records.
type Dump struct {
	ID             int        `json:"id"`
	Commit         string     `json:"commit"`
	Root           string     `json:"root"`
	VisibleAtTip   bool       `json:"visibleAtTip"`
	UploadedAt     time.Time  `json:"uploadedAt"`
	State          string     `json:"state"`
	FailureMessage *string    `json:"failureMessage"`
	StartedAt      *time.Time `json:"startedAt"`
	FinishedAt     *time.Time `json:"finishedAt"`
	ProcessAfter   *time.Time `json:"processAfter"`
	NumResets      int        `json:"numResets"`
	NumFailures    int        `json:"numFailures"`
	RepositoryID   int        `json:"repositoryId"`
	RepositoryName string     `json:"repositoryName"`
	Indexer        string     `json:"indexer"`
}

// scanDumps scans a slice of dumps from the return value of `*store.query`.
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
		); err != nil {
			return nil, err
		}

		dumps = append(dumps, dump)
	}

	return dumps, nil
}

// scanFirstDump scans a slice of dumps from the return value of `*store.query` and returns the first.
func scanFirstDump(rows *sql.Rows, err error) (Dump, bool, error) {
	dumps, err := scanDumps(rows, err)
	if err != nil || len(dumps) == 0 {
		return Dump{}, false, err
	}
	return dumps[0], true, nil
}

// GetDumpByID returns a dump by its identifier and boolean flag indicating its existence.
func (s *store) GetDumpByID(ctx context.Context, id int) (Dump, bool, error) {
	return scanFirstDump(s.Store.Query(ctx, sqlf.Sprintf(`
		SELECT
			d.id,
			d.commit,
			d.root,
			EXISTS (SELECT 1 FROM lsif_uploads_visible_at_tip where repository_id = d.repository_id and upload_id = d.id) AS visible_at_tip,
			d.uploaded_at,
			d.state,
			d.failure_message,
			d.started_at,
			d.finished_at,
			d.process_after,
			d.num_resets,
			d.num_failures,
			d.repository_id,
			d.repository_name,
			d.indexer
		FROM lsif_dumps_with_repository_name d WHERE d.id = %s
	`, id)))
}

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
func (s *store) FindClosestDumps(ctx context.Context, repositoryID int, commit, path string, rootMustEnclosePath bool, indexer string) (_ []Dump, err error) {
	conds := makeFindClosestDumpConditions(path, rootMustEnclosePath, indexer)

	return scanDumps(s.Store.Query(
		ctx,
		sqlf.Sprintf(`
			SELECT
				d.id,
				d.commit,
				d.root,
				EXISTS (SELECT 1 FROM lsif_uploads_visible_at_tip where repository_id = d.repository_id and upload_id = d.id) AS visible_at_tip,
				d.uploaded_at,
				d.state,
				d.failure_message,
				d.started_at,
				d.finished_at,
				d.process_after,
				d.num_resets,
				d.num_failures,
				d.repository_id,
				d.repository_name,
				d.indexer
			FROM lsif_nearest_uploads u
			JOIN lsif_dumps_with_repository_name d ON d.id = u.upload_id
			WHERE u.repository_id = %s AND u.commit = %s AND NOT u.overwritten AND %s
		`, repositoryID, commit, sqlf.Join(conds, " AND ")),
	))
}

// FindClosestDumpsFromGraphFragment returns the set of dumps that can most accurately answer queries for the given repository, commit,
// path, and optional indexer by only considering the given fragment of the full git graph. See FindClosestDumps for additional details.
func (s *store) FindClosestDumpsFromGraphFragment(ctx context.Context, repositoryID int, commit, path string, rootMustEnclosePath bool, indexer string, graph map[string][]string) ([]Dump, error) {
	if len(graph) == 0 {
		return nil, nil
	}

	commits := make([]*sqlf.Query, 0, len(graph))
	for commit := range graph {
		commits = append(commits, sqlf.Sprintf("%s", commit))
	}

	uploadMeta, err := scanUploadMeta(s.Store.Query(ctx, sqlf.Sprintf(`
		SELECT nu.upload_id, nu.commit, u.root, u.indexer, nu.distance, nu.ancestor_visible, nu.overwritten
		FROM lsif_nearest_uploads nu
		JOIN lsif_uploads u ON u.id = nu.upload_id
		WHERE nu.repository_id = %s AND nu.commit IN (%s) AND nu.ancestor_visible
	`, repositoryID, sqlf.Join(commits, ", "))))
	if err != nil {
		return nil, err
	}

	visibleUploads, err := calculateVisibleUploads(graph, uploadMeta)
	if err != nil {
		return nil, err
	}

	var ids []*sqlf.Query
	for _, uploadMeta := range visibleUploads[commit] {
		if uploadMeta.Overwritten == false {
			ids = append(ids, sqlf.Sprintf("%d", uploadMeta.UploadID))
		}
	}
	if len(ids) == 0 {
		return nil, nil
	}

	conds := makeFindClosestDumpConditions(path, rootMustEnclosePath, indexer)

	return scanDumps(s.Store.Query(
		ctx,
		sqlf.Sprintf(`
			SELECT
				d.id,
				d.commit,
				d.root,
				EXISTS (SELECT 1 FROM lsif_uploads_visible_at_tip where repository_id = d.repository_id and upload_id = d.id) AS visible_at_tip,
				d.uploaded_at,
				d.state,
				d.failure_message,
				d.started_at,
				d.finished_at,
				d.process_after,
				d.num_resets,
				d.num_failures,
				d.repository_id,
				d.repository_name,
				d.indexer
			FROM lsif_dumps_with_repository_name d
			WHERE d.id IN (%s) AND %s
		`, sqlf.Join(ids, ","), sqlf.Join(conds, " AND ")),
	))
}

func makeFindClosestDumpConditions(path string, rootMustEnclosePath bool, indexer string) (conds []*sqlf.Query) {
	if rootMustEnclosePath {
		// Ensure that the root is a prefix of the path
		conds = append(conds, sqlf.Sprintf(`%s LIKE (d.root || '%%%%')`, path))
	} else {
		// Ensure that the root is a prefix of the path or vice versa
		conds = append(conds, sqlf.Sprintf(`(%s LIKE (d.root || '%%%%') OR d.root LIKE (%s || '%%%%'))`, path, path))
	}
	if indexer != "" {
		conds = append(conds, sqlf.Sprintf("indexer = %s", indexer))
	}

	return conds
}

func scanFirstIntPair(rows *sql.Rows, queryErr error) (_ int, _ int, _ bool, err error) {
	if queryErr != nil {
		return 0, 0, false, queryErr
	}
	defer func() { err = basestore.CloseRows(rows, err) }()

	if rows.Next() {
		var value1 int
		var value2 int
		if err := rows.Scan(&value1, &value2); err != nil {
			return 0, 0, false, err
		}

		return value1, value2, true, nil
	}

	return 0, 0, false, nil
}

// DeleteOldestDump deletes the oldest dump that is not currently visible at the tip of its repository's default branch.
// This method returns the deleted dump's identifier and a flag indicating its (previous) existence. The associated repository
// will be marked as dirty so that its commit graph will be updated in the background.
func (s *store) DeleteOldestDump(ctx context.Context) (_ int, _ bool, err error) {
	tx, err := s.transact(ctx)
	if err != nil {
		return 0, false, err
	}
	defer func() { err = tx.Done(err) }()

	id, repositoryID, deleted, err := scanFirstIntPair(tx.Store.Query(ctx, sqlf.Sprintf(`
		UPDATE lsif_uploads
		SET state = 'deleted'
		WHERE id IN (
			SELECT d.id FROM lsif_dumps_with_repository_name d
			WHERE NOT EXISTS (SELECT 1 FROM lsif_uploads_visible_at_tip WHERE repository_id = d.repository_id AND upload_id = d.id)
			ORDER BY d.uploaded_at
			LIMIT 1
		)
		RETURNING id, repository_id
	`)))
	if err != nil {
		return 0, false, err
	}
	if !deleted {
		return 0, false, nil
	}

	if err := tx.MarkRepositoryAsDirty(ctx, repositoryID); err != nil {
		return 0, false, err
	}

	return id, true, nil
}

// SoftDeleteOldDumps marks dumps older than the given age that are not visible at the tip of the default branch
// as deleted. The associated repositories will be marked as dirty so that their commit graphs are updated in the
// background.
func (s *store) SoftDeleteOldDumps(ctx context.Context, maxAge time.Duration, now time.Time) (count int, err error) {
	tx, err := s.transact(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { err = tx.Done(err) }()

	repositoryIDs, err := scanCounts(tx.Store.Query(ctx, sqlf.Sprintf(`
		WITH u AS (
			UPDATE lsif_uploads u
				SET state = 'deleted'
				WHERE
					%s - u.finished_at > (%s || ' second')::interval AND
					u.id NOT IN (SELECT uv.upload_id FROM lsif_uploads_visible_at_tip uv WHERE uv.repository_id = u.repository_id)
				RETURNING id, repository_id
		)
		SELECT u.repository_id, count(*) FROM u GROUP BY u.repository_id
	`, now, maxAge/time.Second)))
	if err != nil {
		return 0, err
	}

	for repositoryID, numUpdated := range repositoryIDs {
		if err := tx.MarkRepositoryAsDirty(ctx, repositoryID); err != nil {
			return 0, err
		}

		count += numUpdated
	}

	return count, nil
}

// DeleteOverlapapingDumps deletes all completed uploads for the given repository with the same
// commit, root, and indexer. This is necessary to perform during conversions before changing
// the state of a processing upload to completed as there is a unique index on these four columns.
func (s *store) DeleteOverlappingDumps(ctx context.Context, repositoryID int, commit, root, indexer string) (err error) {
	return s.Store.Exec(ctx, sqlf.Sprintf(`
		UPDATE lsif_uploads
		SET state = 'deleted'
		WHERE repository_id = %s AND commit = %s AND root = %s AND indexer = %s AND state = 'completed'
	`, repositoryID, commit, root, indexer))
}
