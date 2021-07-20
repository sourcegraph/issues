package background

import (
	"context"
	"fmt"

	"github.com/cockroachdb/errors"
	"github.com/inconshreveable/log15"

	"github.com/sourcegraph/sourcegraph/enterprise/internal/batches/global"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/batches/service"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/batches/sources"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/batches/state"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/batches/store"
	btypes "github.com/sourcegraph/sourcegraph/enterprise/internal/batches/types"
	"github.com/sourcegraph/sourcegraph/internal/actor"
	"github.com/sourcegraph/sourcegraph/internal/errcode"
	"github.com/sourcegraph/sourcegraph/internal/types"
)

// unknownJobTypeErr is returned when a ChangesetJob record is of an unknown type
// and hence cannot be executed.
type unknownJobTypeErr struct {
	jobType string
}

func (e unknownJobTypeErr) Error() string {
	return fmt.Sprintf("invalid job type %q", e.jobType)
}

func (e unknownJobTypeErr) NonRetryable() bool {
	return true
}

type bulkProcessor struct {
	tx      *store.Store
	sourcer sources.Sourcer

	css  sources.ChangesetSource
	repo *types.Repo
	ch   *btypes.Changeset
}

func (b *bulkProcessor) process(ctx context.Context, job *btypes.ChangesetJob) (err error) {
	// Use the acting user for the operation to enforce repository permissions.
	ctx = actor.WithActor(ctx, actor.FromUser(job.UserID))

	// Load changeset.
	b.ch, err = b.tx.GetChangeset(ctx, store.GetChangesetOpts{ID: job.ChangesetID})
	if err != nil {
		return errors.Wrap(err, "loading changeset")
	}

	// Load repo.
	b.repo, err = b.tx.Repos().Get(ctx, b.ch.RepoID)
	if err != nil {
		return errors.Wrap(err, "loading repo")
	}

	// Construct changeset source.
	b.css, err = b.sourcer.ForRepo(ctx, b.tx, b.repo)
	if err != nil {
		return errors.Wrap(err, "loading ChangesetSource")
	}
	b.css, err = sources.WithAuthenticatorForUser(ctx, b.tx, b.css, job.UserID, b.repo)
	if err != nil {
		return errors.Wrap(err, "authenticating ChangesetSource")
	}

	log15.Info("processing changeset job", "type", job.JobType)

	switch job.JobType {

	case btypes.ChangesetJobTypeComment:
		return b.comment(ctx, job)
	case btypes.ChangesetJobTypeDetach:
		return b.detach(ctx, job)
	case btypes.ChangesetJobTypeReenqueue:
		return b.reenqueueChangeset(ctx, job)
	case btypes.ChangesetJobTypeMerge:
		return b.mergeChangeset(ctx, job)
	case btypes.ChangesetJobTypeClose:
		return b.closeChangeset(ctx, job)
	case btypes.ChangesetJobTypePublish:
		return b.publishChangeset(ctx, job)

	default:
		return &unknownJobTypeErr{jobType: string(job.JobType)}
	}
}

func (b *bulkProcessor) comment(ctx context.Context, job *btypes.ChangesetJob) error {
	typedPayload, ok := job.Payload.(*btypes.ChangesetJobCommentPayload)
	if !ok {
		return errors.Errorf("invalid payload type for changeset_job, want=%T have=%T", &btypes.ChangesetJobCommentPayload{}, job.Payload)
	}
	cs := &sources.Changeset{
		Changeset: b.ch,
		Repo:      b.repo,
	}
	return b.css.CreateComment(ctx, cs, typedPayload.Message)
}

func (b *bulkProcessor) detach(ctx context.Context, job *btypes.ChangesetJob) error {
	// Try to detach the changeset from the batch change of the job.
	var detached bool
	for i, assoc := range b.ch.BatchChanges {
		if assoc.BatchChangeID == job.BatchChangeID {
			if !b.ch.BatchChanges[i].Detach {
				b.ch.BatchChanges[i].Detach = true
				detached = true
			}
		}
	}

	if !detached {
		return nil
	}

	// If we successfully marked the record as to-be-detached, trigger a reconciler run.

	// We do two `UPDATE` queries here: first we update all of the changesets
	// columns to refelct the changes made here in the bulkProcessor. Then,
	// with `EnqueueChangeset`, we do a second UPDATE that only updates the
	// worker/reconciler-related columns.
	if err := b.tx.UpdateChangeset(ctx, b.ch); err != nil {
		return err
	}

	return b.tx.EnqueueChangeset(ctx, b.ch, global.DefaultReconcilerEnqueueState(), "")
}

func (b *bulkProcessor) reenqueueChangeset(ctx context.Context, job *btypes.ChangesetJob) error {
	svc := service.New(b.tx)
	_, _, err := svc.ReenqueueChangeset(ctx, b.ch.ID)
	return err
}

func (b *bulkProcessor) mergeChangeset(ctx context.Context, job *btypes.ChangesetJob) (err error) {
	typedPayload, ok := job.Payload.(*btypes.ChangesetJobMergePayload)
	if !ok {
		return errors.Errorf("invalid payload type for changeset_job, want=%T have=%T", &btypes.ChangesetJobMergePayload{}, job.Payload)
	}

	cs := &sources.Changeset{
		Changeset: b.ch,
		Repo:      b.repo,
	}
	if err := b.css.MergeChangeset(ctx, cs, typedPayload.Squash); err != nil {
		return err
	}

	events, err := cs.Changeset.Events()
	if err != nil {
		log15.Error("Events", "err", err)
		return errcode.MakeNonRetryable(err)
	}
	state.SetDerivedState(ctx, b.tx.Repos(), cs.Changeset, events)

	if err := b.tx.UpsertChangesetEvents(ctx, events...); err != nil {
		log15.Error("UpsertChangesetEvents", "err", err)
		return errcode.MakeNonRetryable(err)
	}

	if err := b.tx.UpdateChangesetCodeHostState(ctx, cs.Changeset); err != nil {
		log15.Error("UpdateChangeset", "err", err)
		return errcode.MakeNonRetryable(err)
	}

	return nil
}

func (b *bulkProcessor) closeChangeset(ctx context.Context, job *btypes.ChangesetJob) (err error) {
	cs := &sources.Changeset{
		Changeset: b.ch,
		Repo:      b.repo,
	}
	if err := b.css.CloseChangeset(ctx, cs); err != nil {
		return err
	}

	events, err := cs.Changeset.Events()
	if err != nil {
		log15.Error("Events", "err", err)
		return errcode.MakeNonRetryable(err)
	}
	state.SetDerivedState(ctx, b.tx.Repos(), cs.Changeset, events)

	if err := b.tx.UpsertChangesetEvents(ctx, events...); err != nil {
		log15.Error("UpsertChangesetEvents", "err", err)
		return errcode.MakeNonRetryable(err)
	}

	if err := b.tx.UpdateChangesetCodeHostState(ctx, cs.Changeset); err != nil {
		log15.Error("UpdateChangeset", "err", err)
		return errcode.MakeNonRetryable(err)
	}

	return nil
}

func (b *bulkProcessor) publishChangeset(ctx context.Context, job *btypes.ChangesetJob) (err error) {
	typedPayload, ok := job.Payload.(*btypes.ChangesetJobPublishPayload)
	if !ok {
		return errors.Errorf("invalid payload type for changeset_job, want=%T have=%T", &btypes.ChangesetJobPublishPayload{}, job.Payload)
	}

	// We can't publish an imported changeset.
	if b.ch.CurrentSpecID == 0 {
		return errcode.MakeNonRetryable(errors.New("cannot publish an imported changeset"))
	}

	// We can't publish a changeset with its publication state set in the spec.
	spec, err := b.tx.GetChangesetSpecByID(ctx, b.ch.CurrentSpecID)
	if err != nil {
		log15.Error("GetChangesetSpecByID", "err", err)
		return errcode.MakeNonRetryable(errors.Wrapf(err, "getting changeset spec for changeset %d", b.ch.ID))
	} else if spec == nil {
		return errcode.MakeNonRetryable(errors.Newf("no changeset spec for changeset %d", b.ch.ID))
	}

	if !spec.Spec.Published.Nil() {
		return errcode.MakeNonRetryable(errors.New("cannot publish a changeset that has a published value set in its changesetTemplate"))
	}

	// Changesets that are currently processing should be retried at a later stage.
	if b.ch.ReconcilerState == btypes.ReconcilerStateProcessing {
		return errors.New("cannot update a changeset that is currently being processed; will retry")
	}

	// Set the desired UI publication state.
	if typedPayload.Draft {
		b.ch.UiPublicationState = &btypes.ChangesetUiPublicationStateDraft
	} else {
		b.ch.UiPublicationState = &btypes.ChangesetUiPublicationStatePublished
	}

	// Reset the reconciler state.
	b.ch.ResetReconcilerState(global.DefaultReconcilerEnqueueState())

	// And finally update the changeset record.
	if err := b.tx.UpdateChangeset(ctx, b.ch); err != nil {
		log15.Error("UpdateChangeset", "err", err)
		return errcode.MakeNonRetryable(err)
	}
	return nil
}
