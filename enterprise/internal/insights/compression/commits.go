package compression

import (
	"context"
	"database/sql"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/keegancsmith/sqlf"

	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/database/basestore"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
	"github.com/sourcegraph/sourcegraph/internal/vcs/git"
)

type DBCommitStore struct {
	*basestore.Store
}

type CommitStore interface {
	Save(ctx context.Context, id api.RepoID, commit *git.Commit) error
	Get(ctx context.Context, id api.RepoID, start time.Time, end time.Time) ([]CommitStamp, error)
	GetMetadata(ctx context.Context, id api.RepoID) (CommitIndexMetadata, error)
	UpsertMetadataStamp(ctx context.Context, id api.RepoID) (CommitIndexMetadata, error)
	InsertCommits(ctx context.Context, id api.RepoID, commits []*git.Commit) error
}

func NewCommitStore(db dbutil.DB) *DBCommitStore {
	return &DBCommitStore{
		Store: basestore.NewWithDB(db, sql.TxOptions{}),
	}
}

func (c *DBCommitStore) With(other basestore.ShareableStore) *DBCommitStore {
	return &DBCommitStore{Store: c.Store.With(other)}
}

func (c *DBCommitStore) Transact(ctx context.Context) (*DBCommitStore, error) {
	txBase, err := c.Store.Transact(ctx)
	return &DBCommitStore{Store: txBase}, err
}

func (c *DBCommitStore) Save(ctx context.Context, id api.RepoID, commit *git.Commit) error {
	commitID := commit.ID
	if err := c.Exec(ctx, sqlf.Sprintf(insertCommitIndexStr, id, dbutil.CommitBytea(commitID), commit.Committer.Date)); err != nil {
		return errors.Errorf("error saving commit for repo_id: %v commit_id %v: %w", id, commitID, err)
	}

	return nil
}

func (c *DBCommitStore) InsertCommits(ctx context.Context, id api.RepoID, commits []*git.Commit) (err error) {
	tx, err := c.Transact(ctx)
	if err != nil {
		return err
	}

	defer func() { err = tx.Store.Done(err) }()

	for _, commit := range commits {
		if err = tx.Save(ctx, id, commit); err != nil {
			return err
		}
	}

	if _, err = tx.UpsertMetadataStamp(ctx, id); err != nil {
		return err
	}

	return nil
}

// Get Fetch all commits that occur for a specific repository and fall in a specific time range. The time range
// is start inclusive and end exclusive [start, end)
func (c *DBCommitStore) Get(ctx context.Context, id api.RepoID, start time.Time, end time.Time) (_ []CommitStamp, err error) {
	rows, err := c.Query(ctx, sqlf.Sprintf(getCommitsInRangeStr, id, start, end))
	if err != nil {
		return []CommitStamp{}, err
	}
	defer func() { err = basestore.CloseRows(rows, err) }()

	results := make([]CommitStamp, 0)
	for rows.Next() {
		var stamp CommitStamp
		if err := rows.Scan(&stamp.RepoID, &stamp.Commit, &stamp.CommittedAt); err != nil {
			return []CommitStamp{}, err
		}

		results = append(results, stamp)
	}

	return results, nil
}

// GetMetadata Returns commit index metadata for a given repository
func (c *DBCommitStore) GetMetadata(ctx context.Context, id api.RepoID) (CommitIndexMetadata, error) {
	row := c.QueryRow(ctx, sqlf.Sprintf(getCommitIndexMetadataStr, id))

	var metadata CommitIndexMetadata
	if err := row.Scan(&metadata.RepoId, &metadata.Enabled, &metadata.LastIndexedAt); err != nil {
		return CommitIndexMetadata{}, err
	}

	return metadata, nil
}

// UpsertMetadataStamp inserts (or updates, if the row already exists) the index metadata timestamp for a given repository
func (c *DBCommitStore) UpsertMetadataStamp(ctx context.Context, id api.RepoID) (CommitIndexMetadata, error) {
	row := c.QueryRow(ctx, sqlf.Sprintf(upsertCommitIndexMetadataStampStr, id))

	var metadata CommitIndexMetadata
	if err := row.Scan(&metadata.RepoId, &metadata.Enabled, &metadata.LastIndexedAt); err != nil {
		return CommitIndexMetadata{}, err
	}

	return metadata, nil
}

type CommitStamp struct {
	RepoID      int
	Commit      dbutil.CommitBytea
	CommittedAt time.Time
}

type CommitIndexMetadata struct {
	RepoId        int
	Enabled       bool
	LastIndexedAt time.Time
}

const getCommitsInRangeStr = `
-- source: enterprise/internal/insights/compression/commits.go:Get
SELECT repo_id, commit_bytea, committed_at FROM commit_index WHERE repo_id = %s AND committed_at >= %s AND committed_at < %s ORDER BY committed_at desc;
`

const insertCommitIndexStr = `
-- source: enterprise/internal/insights/compression/commits.go:Save
INSERT INTO commit_index(repo_id, commit_bytea, committed_at) VALUES (%s, %s, %s);
`

const getCommitIndexMetadataStr = `
-- source: enterprise/internal/insights/compression/commits.go:GetMetadata
SELECT repo_id, enabled, last_indexed_at FROM commit_index_metadata WHERE repo_id = %s;
`

const upsertCommitIndexMetadataStampStr = `
-- source: enterprise/internal/insights/compression/commits.go:UpsertMetadataStamp
INSERT INTO commit_index_metadata(repo_id)
VALUES (%v)
ON CONFLICT (repo_id) DO UPDATE
SET last_indexed_at = CURRENT_TIMESTAMP
RETURNING repo_id, enabled, last_indexed_at;
`
