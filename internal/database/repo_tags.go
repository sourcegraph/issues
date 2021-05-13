package database

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"time"

	"github.com/keegancsmith/sqlf"

	"github.com/sourcegraph/sourcegraph/internal/database/basestore"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
	"github.com/sourcegraph/sourcegraph/internal/types"
)

type RepoTagsStore struct {
	*basestore.Store
}

func RepoTags(db dbutil.DB) *RepoTagsStore {
	store := basestore.NewWithDB(db, sql.TxOptions{})
	return &RepoTagsStore{store}
}

func RepoTagsWith(other basestore.ShareableStore) *RepoTagsStore {
	return &RepoTagsStore{basestore.NewWithHandle(other.Handle())}
}

func (s *RepoTagsStore) With(other basestore.ShareableStore) *RepoTagsStore {
	return &RepoTagsStore{s.Store.With(other)}
}

func (s *RepoTagsStore) Transact(ctx context.Context) (*RepoTagsStore, error) {
	tx, err := s.Store.Transact(ctx)
	return &RepoTagsStore{tx}, err
}

type RepoTagNotFoundErr struct{ args []interface{} }

func (err RepoTagNotFoundErr) Error() string {
	return fmt.Sprintf("repo tag not found: %v", err.args)
}
func (RepoTagNotFoundErr) NotFound() bool { return true }

func (s *RepoTagsStore) Create(ctx context.Context, repoID int, tag string) (*types.RepoTag, error) {
	if Mocks.RepoTags.Create != nil {
		return Mocks.RepoTags.Create(ctx, repoID, tag)
	}

	q := sqlf.Sprintf(
		repoTagsCreateQueryFmtstr,
		repoID,
		tag,
		sqlf.Join(repoTagsColumns, ","),
	)

	rt := types.RepoTag{}
	row := s.QueryRow(ctx, q)
	if err := scanRepoTag(&rt, row); err != nil {
		return nil, err
	}

	return &rt, nil
}

func (s *RepoTagsStore) Delete(ctx context.Context, rt *types.RepoTag) error {
	if Mocks.RepoTags.Delete != nil {
		return Mocks.RepoTags.Delete(ctx, rt)
	}

	now := time.Now()
	rt.DeletedAt = &now
	_, err := s.Update(ctx, rt)
	return err
}

func (s *RepoTagsStore) GetByID(ctx context.Context, id int64) (*types.RepoTag, error) {
	if Mocks.RepoTags.GetByID != nil {
		return Mocks.RepoTags.GetByID(ctx, id)
	}

	q := sqlf.Sprintf(
		repoTagsGetByIDQueryFmtstr,
		sqlf.Join(repoTagsColumns, ","),
		id,
	)

	rt := types.RepoTag{}
	row := s.QueryRow(ctx, q)
	if err := scanRepoTag(&rt, row); err == sql.ErrNoRows {
		return nil, RepoTagNotFoundErr{args: []interface{}{id}}
	} else if err != nil {
		return nil, err
	}

	return &rt, nil
}

func (s *RepoTagsStore) GetByRepoAndTag(ctx context.Context, repoID int, tag string) (*types.RepoTag, error) {
	if Mocks.RepoTags.GetByRepoAndTag != nil {
		return Mocks.RepoTags.GetByRepoAndTag(ctx, repoID, tag)
	}

	q := sqlf.Sprintf(
		repoTagsGetByRepoAndTagQueryFmtstr,
		sqlf.Join(repoTagsColumns, ","),
		repoID,
		tag,
	)

	rt := types.RepoTag{}
	row := s.QueryRow(ctx, q)
	if err := scanRepoTag(&rt, row); err == sql.ErrNoRows {
		return nil, RepoTagNotFoundErr{args: []interface{}{repoID, tag}}
	} else if err != nil {
		return nil, err
	}

	return &rt, nil
}

type RepoTagsStoreListOpts struct {
	*LimitOffset
	RepoIDs []int
	Tags    []RepoTagsStoreListSearchTerm
}

type RepoTagsStoreListSearchTerm struct {
	Term string
	Not  bool
}

func (opts *RepoTagsStoreListOpts) preds() *sqlf.Query {
	preds := []*sqlf.Query{sqlf.Sprintf("deleted_at IS NULL")}
	if len(opts.RepoIDs) != 0 {
		ids := make([]*sqlf.Query, len(opts.RepoIDs))
		for i, id := range opts.RepoIDs {
			ids[i] = sqlf.Sprintf("%s", id)
		}

		preds = append(preds, sqlf.Sprintf("repo_id IN (%s)", sqlf.Join(ids, ",")))
	}
	if len(opts.Tags) != 0 {
		// TODO: unify with textSearchTermToClause
		for _, tag := range opts.Tags {
			var textOp *sqlf.Query
			if tag.Not {
				textOp = sqlf.Sprintf("!~*")
			} else {
				textOp = sqlf.Sprintf("~*")
			}

			preds = append(preds, sqlf.Sprintf(
				`(tag %s ('\m'||%s||'\M'))`,
				textOp,
				regexp.QuoteMeta(tag.Term),
			))
		}
	}

	return sqlf.Join(preds, "AND")
}

func (opts *RepoTagsStoreListOpts) sql() *sqlf.Query {
	if opts.LimitOffset == nil || opts.Limit == 0 {
		return &sqlf.Query{}
	}

	return (&LimitOffset{Limit: opts.Limit + 1, Offset: opts.Offset}).SQL()
}

func (s *RepoTagsStore) Count(ctx context.Context, opts RepoTagsStoreListOpts) (int, error) {
	if Mocks.RepoTags.Count != nil {
		return Mocks.RepoTags.Count(ctx, opts)
	}

	q := sqlf.Sprintf(
		repoTagsCountQueryFmtstr,
		opts.preds(),
	)

	count, ok, err := basestore.ScanFirstInt(s.Query(ctx, q))
	if err != nil || !ok {
		return count, err
	}
	return count, nil
}

func (s *RepoTagsStore) List(ctx context.Context, opts RepoTagsStoreListOpts) (tags []*types.RepoTag, next int, err error) {
	if Mocks.RepoTags.List != nil {
		return Mocks.RepoTags.List(ctx, opts)
	}

	q := sqlf.Sprintf(
		repoTagsListQueryFmtstr,
		sqlf.Join(repoTagsColumns, ","),
		opts.preds(),
		opts.sql(),
	)

	rows, err := s.Query(ctx, q)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		rt := types.RepoTag{}
		if err := scanRepoTag(&rt, rows); err != nil {
			return nil, 0, err
		}
		tags = append(tags, &rt)
	}

	// Check if there were more results than the limit: if so, then we need to
	// set the return cursor and lop off the extra credential that we retrieved.
	if opts.LimitOffset != nil && opts.Limit != 0 && len(tags) == opts.Limit+1 {
		next = opts.Offset + opts.Limit
		tags = tags[:len(tags)-1]
	}

	return tags, next, nil
}

func (s *RepoTagsStore) Update(ctx context.Context, rt *types.RepoTag) (*types.RepoTag, error) {
	if Mocks.RepoTags.Update != nil {
		return Mocks.RepoTags.Update(ctx, rt)
	}

	rt.UpdatedAt = time.Now()
	q := sqlf.Sprintf(
		repoTagsUpdateQueryFmtstr,
		rt.RepoID,
		rt.Tag,
		rt.CreatedAt,
		rt.UpdatedAt,
		&dbutil.NullTime{Time: rt.DeletedAt},
		rt.ID,
		sqlf.Join(repoTagsColumns, ","),
	)

	updated := types.RepoTag{}
	row := s.QueryRow(ctx, q)
	if err := scanRepoTag(&updated, row); err == sql.ErrNoRows {
		return nil, RepoTagNotFoundErr{args: []interface{}{*rt}}
	} else if err != nil {
		return nil, err
	}

	return &updated, nil
}

var repoTagsColumns = []*sqlf.Query{
	sqlf.Sprintf("id"),
	sqlf.Sprintf("repo_id"),
	sqlf.Sprintf("tag"),
	sqlf.Sprintf("created_at"),
	sqlf.Sprintf("updated_at"),
	sqlf.Sprintf("deleted_at"),
}

const repoTagsCreateQueryFmtstr = `
-- source: internal/database/repo_tags.go:Create
INSERT INTO repo_tags (repo_id, tag)
VALUES (%s, %s)
RETURNING %s
`

const repoTagsGetByIDQueryFmtstr = `
-- source: internal/database/repo_tags.go:GetByID
SELECT %s
FROM repo_tags
WHERE id = %s AND deleted_at IS NULL
`

const repoTagsGetByRepoAndTagQueryFmtstr = `
-- source: internal/database/repo_tags.go:GetByRepoAndTag
SELECT %s
FROM repo_tags
WHERE
	repo_id = %s
	AND LOWER(tag) = %s
	AND deleted_at IS NULL
`

const repoTagsCountQueryFmtstr = `
-- source: internal/database/repo_tags.go:Count
SELECT COUNT(id)
FROM repo_tags
WHERE %s
`

const repoTagsListQueryFmtstr = `
-- source: internal/database/repo_tags.go:List
SELECT %s
FROM repo_tags
WHERE %s
ORDER BY repo_id ASC, tag ASC
%s  -- LIMIT clause
`

const repoTagsUpdateQueryFmtstr = `
-- source: internal/database/repo_tags.go:Update
UPDATE repo_tags
SET
	repo_id = %s,
	tag = %s,
	created_at = %s,
	updated_at = %s,
	deleted_at = %s
WHERE
	id = %s
RETURNING %s
`

func scanRepoTag(tag *types.RepoTag, s interface {
	Scan(...interface{}) error
}) error {
	var deleted time.Time

	if err := s.Scan(
		&tag.ID,
		&tag.RepoID,
		&tag.Tag,
		&tag.CreatedAt,
		&tag.UpdatedAt,
		&dbutil.NullTime{Time: &deleted},
	); err != nil {
		return err
	}

	if deleted.IsZero() {
		tag.DeletedAt = nil
	} else {
		tag.DeletedAt = &deleted
	}

	return nil
}
