package enqueuer

import (
	"context"

	"github.com/cockroachdb/errors"
	"github.com/inconshreveable/log15"
	"github.com/opentracing/opentracing-go/log"

	store "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/stores/dbstore"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/errcode"
	"github.com/sourcegraph/sourcegraph/internal/observation"
	"github.com/sourcegraph/sourcegraph/lib/codeintel/autoindex/config"
	"github.com/sourcegraph/sourcegraph/lib/codeintel/autoindex/inference"
	"github.com/sourcegraph/sourcegraph/lib/codeintel/semantic"
)

type IndexEnqueuer struct {
	dbStore          DBStore
	gitserverClient  GitserverClient
	repoUpdater      RepoUpdaterClient
	maxJobsPerCommit int
	operations       *operations
}

const defaultMaxJobsPerCommit = 25

func NewIndexEnqueuer(dbStore DBStore, gitClient GitserverClient, repoUpdater RepoUpdaterClient, observationContext *observation.Context) *IndexEnqueuer {
	return &IndexEnqueuer{
		dbStore:          dbStore,
		gitserverClient:  gitClient,
		repoUpdater:      repoUpdater,
		maxJobsPerCommit: defaultMaxJobsPerCommit,
		operations:       newOperations(observationContext),
	}
}

// QueueIndexesForRepository attempts to queue an index for the lastest commit on the default branch of the given
// repository. If this repository and commit already has an index or upload record associated with it, this method
// does nothing.
func (s *IndexEnqueuer) QueueIndexesForRepository(ctx context.Context, repositoryID int) error {
	return s.queueIndexForRepository(ctx, repositoryID, false)
}

// ForceQueueIndexesForRepository attempts to queue an index for the lastest commit on the default branch of the given
// repository. If this repository and commit already has an index or upload record associated with it, a new index job
// record will still be enqueued.
func (s *IndexEnqueuer) ForceQueueIndexesForRepository(ctx context.Context, repositoryID int) error {
	return s.queueIndexForRepository(ctx, repositoryID, true)
}

// InferIndexConfiguration looks at the repository contents at the lastest commit on the default branch of the given
// repository and determines an index configuration that is likely to succeed.
func (s *IndexEnqueuer) InferIndexConfiguration(ctx context.Context, repositoryID int) (_ *config.IndexConfiguration, err error) {
	ctx, traceLog, endObservation := s.operations.InferIndexConfiguration.WithAndLogger(ctx, &err, observation.Args{
		LogFields: []log.Field{
			log.Int("repositoryID", repositoryID),
		},
	})
	defer endObservation(1, observation.Args{})

	commit, ok, err := s.gitserverClient.Head(ctx, repositoryID)
	if err != nil || !ok {
		return nil, errors.Wrap(err, "gitserver.Head")
	}
	traceLog(log.String("commit", commit))

	indexJobs, err := s.inferIndexJobsFromRepositoryStructure(ctx, repositoryID, commit)
	if err != nil || len(indexJobs) == 0 {
		return nil, err
	}

	return &config.IndexConfiguration{
		IndexJobs: indexJobs,
	}, nil
}

// QueueIndexesForPackage enqueues index jobs for a dependency of a recently-processed precise code intelligence
// index. Currently we only support recognition of "gomod" import monikers.
func (s *IndexEnqueuer) QueueIndexesForPackage(ctx context.Context, pkg semantic.Package) (err error) {
	ctx, traceLog, endObservation := s.operations.QueueIndexForPackage.WithAndLogger(ctx, &err, observation.Args{
		LogFields: []log.Field{
			log.String("scheme", pkg.Scheme),
			log.String("name", pkg.Name),
			log.String("version", pkg.Version),
		},
	})
	defer endObservation(1, observation.Args{})

	repoName, revision, ok := InferGoRepositoryAndRevision(pkg)
	if !ok {
		return nil
	}
	traceLog(log.String("repoName", repoName))
	traceLog(log.String("revision", revision))

	resp, err := s.repoUpdater.EnqueueRepoUpdate(ctx, api.RepoName(repoName))
	if err != nil {
		if errcode.IsNotFound(err) {
			return nil
		}

		return errors.Wrap(err, "repoUpdater.EnqueueRepoUpdate")
	}

	commit, err := s.gitserverClient.ResolveRevision(ctx, int(resp.ID), revision)
	if err != nil {
		if errcode.IsNotFound(err) {
			return nil
		}

		return errors.Wrap(err, "gitserverClient.ResolveRevision")
	}

	return s.queueIndexForRepositoryAndCommit(ctx, int(resp.ID), string(commit), false, traceLog)
}

// queueIndexForRepository determines the head of the default branch of the given repository and attempts to
// determine a set of index jobs to enqueue.
//
// If the force flag is false, then the presence of an upload or index record for this given repository and commit
// will cause this method to no-op. Note that this is NOT a guarantee that there will never be any duplicate records
// when the flag is false.
func (s *IndexEnqueuer) queueIndexForRepository(ctx context.Context, repositoryID int, force bool) (err error) {
	ctx, traceLog, endObservation := s.operations.QueueIndex.WithAndLogger(ctx, &err, observation.Args{
		LogFields: []log.Field{
			log.Int("repositoryID", repositoryID),
		},
	})
	defer endObservation(1, observation.Args{})

	commit, ok, err := s.gitserverClient.Head(ctx, repositoryID)
	if err != nil || !ok {
		return errors.Wrap(err, "gitserver.Head")
	}
	traceLog(log.String("commit", commit))

	return s.queueIndexForRepositoryAndCommit(ctx, repositoryID, commit, force, traceLog)
}

// queueIndexForRepositoryAndCommit determines a set of index jobs to enqueue for the given repository and commit.
//
// If the force flag is false, then the presence of an upload or index record for this given repository and commit
// will cause this method to no-op. Note that this is NOT a guarantee that there will never be any duplicate records
// when the flag is false.
func (s *IndexEnqueuer) queueIndexForRepositoryAndCommit(ctx context.Context, repositoryID int, commit string, force bool, traceLog observation.TraceLogger) error {
	if !force {
		isQueued, err := s.dbStore.IsQueued(ctx, repositoryID, commit)
		if err != nil {
			return errors.Wrap(err, "dbstore.IsQueued")
		}
		if isQueued {
			return nil
		}
	}

	indexes, err := s.getIndexRecords(ctx, repositoryID, commit)
	if err != nil {
		return err
	}
	if len(indexes) == 0 {
		return nil
	}
	traceLog(log.Int("numIndexes", len(indexes)))

	return s.queueIndexes(ctx, repositoryID, commit, indexes)
}

// queueIndexes inserts a set of index records into the database. It is assumed that the given repository id an
// commit are the same for each given index record. In the same transaction as the insert, the repository's row
// is updated in the lsif_indexable_repositories table as a crude form of rate limiting.
func (s *IndexEnqueuer) queueIndexes(ctx context.Context, repositoryID int, commit string, indexes []store.Index) (err error) {
	tx, err := s.dbStore.Transact(ctx)
	if err != nil {
		return errors.Wrap(err, "dbstore.Transact")
	}
	defer func() {
		err = tx.Done(err)
	}()

	for _, index := range indexes {
		id, err := tx.InsertIndex(ctx, index)
		if err != nil {
			return errors.Wrap(err, "dbstore.QueueIndex")
		}

		log15.Info(
			"Enqueued index",
			"id", id,
			"repository_id", repositoryID,
			"commit", commit,
		)
	}

	return nil
}

// inferIndexJobsFromRepositoryStructure collects the result of  InferIndexJobs over all registered recognizers.
func (s *IndexEnqueuer) inferIndexJobsFromRepositoryStructure(ctx context.Context, repositoryID int, commit string) ([]config.IndexJob, error) {
	paths, err := s.gitserverClient.ListFiles(ctx, repositoryID, commit, inference.Patterns)
	if err != nil {
		return nil, errors.Wrap(err, "gitserver.ListFiles")
	}

	gitclient := newGitClient(s.gitserverClient, repositoryID, commit)

	var indexes []config.IndexJob
	for _, recognizer := range inference.Recognizers {
		indexes = append(indexes, recognizer.InferIndexJobs(gitclient, paths)...)
	}

	if len(indexes) > s.maxJobsPerCommit {
		log15.Info("Too many inferred roots. Scheduling no index jobs for repository.", "repository_id", repositoryID)
		return nil, nil
	}

	return indexes, nil
}
