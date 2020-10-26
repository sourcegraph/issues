package commitupdater

import (
	"context"
	"time"

	"github.com/inconshreveable/log15"
	"github.com/pkg/errors"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/commits"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/store"
	"github.com/sourcegraph/sourcegraph/internal/goroutine"
)

// Updater periodically re-calculates the commit and upload visibility graph for repositories
// that are marked as dirty by the worker process. This is done out-of-band from the rest of
// the upload processing as it is likely that we are processing multiple uploads concurrently
// for the same repository and should not repeat the work since the last calculation performed
// will always be the one we want.
type Updater struct {
	store   store.Store
	updater commits.Updater
}

var _ goroutine.Handler = &Updater{}

type UpdaterOptions struct {
	Interval time.Duration
}

func NewUpdater(store store.Store, updater commits.Updater, options UpdaterOptions) goroutine.BackgroundRoutine {
	return goroutine.NewPeriodicGoroutine(context.Background(), options.Interval, &Updater{
		store:   store,
		updater: updater,
	})
}

// Handle checks for dirty repositories and invokes the underlying updater on each one.
func (u *Updater) Handle(ctx context.Context) error {
	repositoryIDs, err := u.store.DirtyRepositories(ctx)
	if err != nil {
		return errors.Wrap(err, "store.DirtyRepositories")
	}

	for repositoryID, dirtyFlag := range repositoryIDs {
		if err := u.updater.TryUpdate(ctx, repositoryID, dirtyFlag); err != nil {
			log15.Warn("Failed to update commit graph", "err", err)
		}
	}

	return nil
}

func (u *Updater) HandleError(err error) {
	log15.Error("Failed to run update process", "err", err)
}
