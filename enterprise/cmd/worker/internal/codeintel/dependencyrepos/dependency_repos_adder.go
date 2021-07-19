package dependencyrepos

import (
	"context"
	"fmt"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/hashicorp/go-multierror"

	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/stores/dbstore"
	"github.com/sourcegraph/sourcegraph/internal/actor"
	"github.com/sourcegraph/sourcegraph/internal/workerutil"
	"github.com/sourcegraph/sourcegraph/internal/workerutil/dbworker"
	dbworkerstore "github.com/sourcegraph/sourcegraph/internal/workerutil/dbworker/store"
)

func NewDependencyRepoAdder(
	dbStore DBStore,
	workerStore dbworkerstore.Store,
	pollInterval time.Duration,
	numProcessorRoutines int,
	workerMetrics workerutil.WorkerMetrics,
) *workerutil.Worker {
	rootContext := actor.WithActor(context.Background(), &actor.Actor{Internal: true})

	handler := &dependencyRepoAddingHandler{
		dbStore: dbStore,
	}

	return dbworker.NewWorker(rootContext, workerStore, handler, workerutil.WorkerOptions{
		Name:        "code_intel_dependency_repo_adding_worker",
		NumHandlers: numProcessorRoutines,
		Interval:    pollInterval,
		Metrics:     workerMetrics,
	})
}

type dependencyRepoAddingHandler struct {
	dbStore DBStore
}

var _ workerutil.Handler = &dependencyRepoAddingHandler{}

type DependencyRepoInfo struct {
	Identifier, Version, Scheme string
}

func (h *dependencyRepoAddingHandler) Handle(ctx context.Context, record workerutil.Record) error {
	job := record.(dbstore.DependencyRepoAddingJob)

	scanner, err := h.dbStore.ReferencesForUpload(ctx, job.UploadID)
	if err != nil {
		return errors.Wrap(err, "dbstore.ReferencesForUpload")
	}
	defer func() {
		if closeErr := scanner.Close(); closeErr != nil {
			err = multierror.Append(err, errors.Wrap(closeErr, "dbstore.ReferenceIDsAndFilters.Close"))
		}
	}()

	var dependencies []dbstore.DependencyRepoInfo

	var errs []error
	for {
		packageReference, exists, err := scanner.Next()
		fmt.Println("ONCE")
		if err != nil {
			return errors.Wrap(err, "dbstore.ReferencesForUpload.Next")
		}
		if !exists {
			break
		}

		parser, ok := schemeToParser[packageReference.Scheme]
		if !ok {
			return errors.Newf("no parser found for schema %s", packageReference.Scheme)
		}

		result, err := parser(packageReference.Package)
		if err != nil {
			errs = append(errs, errors.Wrap(err, fmt.Sprintf("dependencyrepo.PackageInformationParser: failed to parse package information data %#v", packageReference)))
			continue
		}

		dependencies = append(dependencies, dbstore.DependencyRepoInfo{
			Identifier: result.Identifier,
			Version:    result.Version,
			Scheme:     result.Scheme,
		})
	}

	fmt.Printf("ERRORS %d DEPS %d %v\n", len(errs), len(dependencies), dependencies)

	if len(errs) == 1 {
		return errs[0]
	}

	if err = h.dbStore.InsertCloneableDependencyRepos(ctx, dependencies); err != nil {
		return errors.Wrap(err, "dbstore.InsertCloneableDependencyRepos")
	}

	return err
}
