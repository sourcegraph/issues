package compression

import (
	"context"
	"database/sql"
	"time"

	"github.com/cockroachdb/errors"

	"github.com/sourcegraph/sourcegraph/internal/types"

	"github.com/inconshreveable/log15"
	"golang.org/x/time/rate"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/envvar"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/insights/discovery"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/database/dbcache"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
	"github.com/sourcegraph/sourcegraph/internal/goroutine"
	"github.com/sourcegraph/sourcegraph/internal/vcs/git"
)

type RepoStore interface {
	GetByName(ctx context.Context, repoName api.RepoName) ([]*types.Repo, error)
}

type CommitIndexer struct {
	limiter           *rate.Limiter
	allReposIterator  func(ctx context.Context, each func(repoName string) error) error
	getRepoID         func(ctx context.Context, name api.RepoName) (*types.Repo, error)
	getCommits        func(ctx context.Context, name api.RepoName, after time.Time) ([]*git.Commit, error)
	commitStore       CommitStore
	maxHistoricalTime time.Time
	background        context.Context
}

func NewCommitIndexer(background context.Context, base dbutil.DB, insights dbutil.DB) *CommitIndexer {
	//TODO(insights): add a setting for historical index length
	startTime := time.Now().AddDate(-1, 0, 0)

	repoStore := database.Repos(base)

	commitStore := NewCommitStore(insights)

	iterator := discovery.AllReposIterator{
		IndexableReposLister:  dbcache.NewIndexableReposLister(repoStore),
		RepoStore:             repoStore,
		Clock:                 time.Now,
		SourcegraphDotComMode: envvar.SourcegraphDotComMode(),

		// If a new repository is added to Sourcegraph, it can take 0-15m for it to be picked
		// up for backfilling.
		RepositoryListCacheTime: 15 * time.Minute,
	}
	limiter := rate.NewLimiter(10, 1)

	indexer := CommitIndexer{
		limiter:           limiter,
		allReposIterator:  iterator.ForEach,
		getRepoID:         repoStore.GetByName,
		commitStore:       commitStore,
		maxHistoricalTime: startTime,
		background:        background,
		getCommits:        getCommits,
	}
	return &indexer
}

func NewCommitIndexerWorker(ctx context.Context, base dbutil.DB, insights dbutil.DB) goroutine.BackgroundRoutine {
	indexer := NewCommitIndexer(ctx, base, insights)

	return indexer.Handler(ctx)
}

func (i *CommitIndexer) Handler(ctx context.Context) goroutine.BackgroundRoutine {
	//TODO(insights) consider adding setting for index interval
	interval := time.Hour * 1

	//TODO(insights) convert this to a metrics generating goroutine
	return goroutine.NewPeriodicGoroutine(ctx, interval,
		goroutine.NewHandlerWithErrorMessage("commit_indexer_handler", func(ctx context.Context) error {
			return i.indexAll(ctx)
		}))
}

func (i *CommitIndexer) indexAll(ctx context.Context) error {
	err := i.allReposIterator(ctx, i.indexRepository)
	if err != nil {
		return err
	}

	return nil
}

// indexRepository attempts to index the commits given a repository name. This method will absorb any errors that
// occur during execution and skip the index for this repository.
// If this repository already has some commits indexed, only commits made more recently than the previous index will be added.
func (i *CommitIndexer) indexRepository(name string) error {
	err := i.index(name)
	if err != nil {
		log15.Error(err.Error())
	}
	return nil
}

func (i *CommitIndexer) index(name string) error {
	ctx, cancel := context.WithTimeout(i.background, time.Second*45)
	defer cancel()

	err := i.limiter.Wait(ctx)
	if err != nil {
		return nil
	}

	logger := log15.Root().New("worker", "insights-commit-indexer")

	repoName := api.RepoName(name)

	repo, err := i.getRepoID(ctx, repoName)
	if err != nil {
		return errors.Wrapf(err, "unable to resolve repository id repo_name: %v", repoName)
	}
	repoId := repo.ID

	metadata, err := getMetadata(ctx, repoId, i.commitStore)
	if err != nil {
		return errors.Wrapf(err, "unable to fetch commit index metadata repo_id: %v", repoId)
	}

	if !metadata.Enabled {
		logger.Debug("commit indexing disabled", "repo_id", repoId)
		return nil
	}

	searchTime := max(i.maxHistoricalTime, metadata.LastIndexedAt)

	logger.Debug("fetching commits", "repo_id", repoId, "after", searchTime)
	commits, err := i.getCommits(ctx, repoName, searchTime)
	if err != nil {
		return errors.Wrapf(err, "error fetching commits from gitserver repo_id: %v", repoId)
	}

	if len(commits) == 0 {
		logger.Debug("commit index up to date", "repo_id", repoId)

		if _, err = i.commitStore.UpsertMetadataStamp(ctx, repoId); err != nil {
			return err

		}

		return nil
	}

	log15.Debug("indexing commits", "repo_id", repoId, "count", len(commits))
	err = i.commitStore.InsertCommits(ctx, repoId, commits)
	if err != nil {
		return errors.Wrapf(err, "unable to update commit index repo_id: %v", repoId)
	}

	return nil
}

//getCommits fetches the commits from the remote gitserver for a repository after a certain time.
func getCommits(ctx context.Context, name api.RepoName, after time.Time) ([]*git.Commit, error) {
	return git.Commits(ctx, name, git.CommitsOptions{N: 0, DateOrder: true, NoEnsureRevision: true, After: after.Format(time.RFC3339)})
}

//getMetadata gets the index metadata for a repository. The metadata will be generated if it doesn't already exist, such as
// in the case of a newly installed repository.
func getMetadata(ctx context.Context, id api.RepoID, store CommitStore) (CommitIndexMetadata, error) {
	metadata, err := store.GetMetadata(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		metadata, err = store.UpsertMetadataStamp(ctx, id)
	}
	if err != nil {
		return CommitIndexMetadata{}, err
	}

	return metadata, nil
}

func max(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}
