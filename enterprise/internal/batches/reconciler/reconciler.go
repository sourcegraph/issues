package reconciler

import (
	"context"

	"github.com/hashicorp/go-multierror"
	"github.com/inconshreveable/log15"

	"github.com/sourcegraph/sourcegraph/enterprise/internal/batches/sources"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/batches/store"
	btypes "github.com/sourcegraph/sourcegraph/enterprise/internal/batches/types"
	"github.com/sourcegraph/sourcegraph/internal/gitserver/protocol"
	"github.com/sourcegraph/sourcegraph/internal/workerutil"
)

type GitserverClient interface {
	CreateCommitFromPatch(ctx context.Context, req protocol.CreateCommitFromPatchRequest) (string, error)
}

// Reconciler processes changesets and reconciles their current state — in
// Sourcegraph or on the code host — with that described in the current
// ChangesetSpec associated with the changeset.
type Reconciler struct {
	gitserverClient GitserverClient
	sourcer         sources.Sourcer
	store           *store.Store

	// This is used to disable a time.Sleep for operationSleep so that the
	// tests don't run slower.
	noSleepBeforeSync bool
}

func New(gitClient GitserverClient, sourcer sources.Sourcer, store *store.Store) *Reconciler {
	return &Reconciler{
		gitserverClient: gitClient,
		sourcer:         sourcer,
		store:           store,
	}
}

// HandlerFunc returns a dbworker.HandlerFunc that can be passed to a
// workerutil.Worker to process queued changesets.
func (r *Reconciler) HandlerFunc() workerutil.HandlerFunc {
	return func(ctx context.Context, record workerutil.Record) (err error) {
		tx, err := r.store.Transact(ctx)
		if err != nil {
			return err
		}
		defer func() {
			doneErr := tx.Done(nil)
			if err != nil && doneErr != nil {
				err = multierror.Append(err, doneErr)
			}
			if doneErr != nil {
				err = doneErr
			}
		}()

		return r.process(ctx, tx, record.(*btypes.Changeset))
	}
}

// process is the main entry point of the reconciler and processes changesets
// that were marked as queued in the database.
//
// For each changeset, the reconciler computes an execution plan to run to reconcile a
// possible divergence between the changeset's current state and the desired
// state (for example expressed in a changeset spec).
//
// To do that, the reconciler looks at the changeset's current state
// (publication state, external state, sync state, ...), its (if set) current
// ChangesetSpec, and (if it exists) its previous ChangesetSpec.
//
// If an error is returned, the workerutil.Worker that called this function
// (through the HandlerFunc) will set the changeset's ReconcilerState to
// errored and set its FailureMessage to the error.
func (r *Reconciler) process(ctx context.Context, tx *store.Store, ch *btypes.Changeset) error {
	// Reset the error message.
	ch.FailureMessage = nil

	prev, curr, err := loadChangesetSpecs(ctx, tx, ch)
	if err != nil {
		return nil
	}

	plan, err := DeterminePlan(prev, curr, ch)
	if err != nil {
		return err
	}

	log15.Info("Reconciler processing changeset", "changeset", ch.ID, "operations", plan.Ops)

	return executePlan(
		ctx,
		r.gitserverClient,
		r.sourcer,
		r.noSleepBeforeSync,
		tx,
		plan,
	)
}

func loadChangesetSpecs(ctx context.Context, tx *store.Store, ch *btypes.Changeset) (prev, curr *btypes.ChangesetSpec, err error) {
	if ch.CurrentSpecID != 0 {
		curr, err = tx.GetChangesetSpecByID(ctx, ch.CurrentSpecID)
		if err != nil {
			return
		}
	}
	if ch.PreviousSpecID != 0 {
		prev, err = tx.GetChangesetSpecByID(ctx, ch.PreviousSpecID)
		if err != nil {
			return
		}
	}
	return
}
