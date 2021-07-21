package indexing

import (
	"context"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/hashicorp/go-multierror"

	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/stores/dbstore"
	"github.com/sourcegraph/sourcegraph/internal/actor"
	"github.com/sourcegraph/sourcegraph/internal/workerutil"
	"github.com/sourcegraph/sourcegraph/internal/workerutil/dbworker"
	dbworkerstore "github.com/sourcegraph/sourcegraph/internal/workerutil/dbworker/store"
	"github.com/sourcegraph/sourcegraph/lib/codeintel/semantic"
)

// NewDependencyIndexingScheduler returns a new worker instance that processes
// records from lsif_dependency_indexing_jobs.
func NewDependencyIndexingScheduler(
	dbStore DBStore,
	workerStore dbworkerstore.Store,
	enqueuer IndexEnqueuer,
	pollInterval time.Duration,
	numProcessorRoutines int,
	workerMetrics workerutil.WorkerMetrics,
) *workerutil.Worker {
	rootContext := actor.WithActor(context.Background(), &actor.Actor{Internal: true})

	handler := &dependencyIndexingSchedulerHandler{
		dbStore:       dbStore,
		indexEnqueuer: enqueuer,
	}

	return dbworker.NewWorker(rootContext, workerStore, handler, workerutil.WorkerOptions{
		Name:              "precise_code_intel_dependency_indexing_scheduler_worker",
		NumHandlers:       numProcessorRoutines,
		Interval:          pollInterval,
		Metrics:           workerMetrics,
		HeartbeatInterval: 1 * time.Second,
	})
}

type dependencyIndexingSchedulerHandler struct {
	dbStore       DBStore
	indexEnqueuer IndexEnqueuer
}

var _ workerutil.Handler = &dependencyIndexingSchedulerHandler{}

// Handle iterates all import monikers associated with a given upload that has
// recently completed processing. Each moniker is interpreted according to its
// scheme to determine the dependent repository and commit. A set of indexing
// jobs are enqueued for each repository and commit pair.
func (h *dependencyIndexingSchedulerHandler) Handle(ctx context.Context, record workerutil.Record) error {
	job := record.(dbstore.DependencyIndexingJob)

	if ok, err := h.shouldIndexDependencies(ctx, h.dbStore, job.UploadID); err != nil || !ok {
		return err
	}

	scanner, err := h.dbStore.ReferencesForUpload(ctx, job.UploadID)
	if err != nil {
		return errors.Wrap(err, "dbstore.ReferencesForUpload")
	}
	defer func() {
		if closeErr := scanner.Close(); closeErr != nil {
			err = multierror.Append(err, errors.Wrap(closeErr, "dbstore.ReferenceIDsAndFilters.Close"))
		}
	}()

	var errs []error
	for {
		packageReference, exists, err := scanner.Next()
		if err != nil {
			return errors.Wrap(err, "dbstore.ReferencesForUpload.Next")
		}
		if !exists {
			break
		}

		pkg := semantic.Package{
			Scheme:  packageReference.Package.Scheme,
			Name:    packageReference.Package.Name,
			Version: packageReference.Package.Version,
		}
		if err := h.indexEnqueuer.QueueIndexesForPackage(ctx, pkg); err != nil {
			errs = append(errs, errors.Wrap(err, "enqueuer.QueueIndexesForPackage"))
		}
	}

	if len(errs) == 0 {
		return nil
	}
	if len(errs) == 1 {
		return errs[0]
	}

	return multierror.Append(nil, errs...)
}

var dependencyIndexingRepositoryIDs = []int{
	35703861, // github.com/sourcegraph/jsonx on cloud
	36146693, // github.com/sourcegraph/src-clion cloud
	36809250, // github.com/sourcegraph/sourcegraph on cloud
	38967070, // github.com/sourcegraph/lsif-go on cloud
}

// shouldIndexDependencies returns true if the given upload should undergo dependency
// indexing. Currently, we're only enabling dependency indexing for a small, hard-coded
// list of repository identifiers in the Cloud env.
func (h *dependencyIndexingSchedulerHandler) shouldIndexDependencies(ctx context.Context, store DBStore, uploadID int) (bool, error) {
	upload, _, err := store.GetUploadByID(ctx, uploadID)
	if err != nil {
		return false, errors.Wrap(err, "dbstore.GetUploadByID")
	}

	for _, repositoryID := range dependencyIndexingRepositoryIDs {
		if upload.RepositoryID == repositoryID {
			return true, nil
		}
	}

	return true, nil // false, nil
}
