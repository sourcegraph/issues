package reconciler

import (
	"fmt"
	"sort"
	"strings"

	btypes "github.com/sourcegraph/sourcegraph/enterprise/internal/batches/types"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/bitbucketserver"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/github"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/gitlab"
)

var operationPrecedence = map[btypes.ReconcilerOperation]int{
	btypes.ReconcilerOperationPush:         0,
	btypes.ReconcilerOperationDetach:       0,
	btypes.ReconcilerOperationArchive:      0,
	btypes.ReconcilerOperationImport:       1,
	btypes.ReconcilerOperationPublish:      1,
	btypes.ReconcilerOperationPublishDraft: 1,
	btypes.ReconcilerOperationClose:        1,
	btypes.ReconcilerOperationReopen:       2,
	btypes.ReconcilerOperationUndraft:      3,
	btypes.ReconcilerOperationRedraft:      3,
	btypes.ReconcilerOperationUpdate:       4,
	btypes.ReconcilerOperationSleep:        5,
	btypes.ReconcilerOperationSync:         6,
}

type Operations []btypes.ReconcilerOperation

func (ops Operations) IsNone() bool {
	return len(ops) == 0
}

func (ops Operations) Equal(b Operations) bool {
	if len(ops) != len(b) {
		return false
	}
	bEntries := make(map[btypes.ReconcilerOperation]struct{})
	for _, e := range b {
		bEntries[e] = struct{}{}
	}

	for _, op := range ops {
		if _, ok := bEntries[op]; !ok {
			return false
		}
	}

	return true
}

func (ops Operations) String() string {
	if ops.IsNone() {
		return "No operations required"
	}
	eo := ops.ExecutionOrder()
	ss := make([]string, len(eo))
	for i, val := range eo {
		ss[i] = strings.ToLower(string(val))
	}
	return strings.Join(ss, " => ")
}

func (ops Operations) ExecutionOrder() []btypes.ReconcilerOperation {
	uniqueOps := []btypes.ReconcilerOperation{}

	// Make sure ops are unique.
	seenOps := make(map[btypes.ReconcilerOperation]struct{})
	for _, op := range ops {
		if _, ok := seenOps[op]; ok {
			continue
		}

		seenOps[op] = struct{}{}
		uniqueOps = append(uniqueOps, op)
	}

	sort.Slice(uniqueOps, func(i, j int) bool {
		return operationPrecedence[uniqueOps[i]] < operationPrecedence[uniqueOps[j]]
	})

	return uniqueOps
}

// Plan represents the possible operations the reconciler needs to do
// to reconcile the current and the desired state of a changeset.
type Plan struct {
	// The changeset that is targeted in this plan.
	Changeset *btypes.Changeset

	// The changeset spec that is used in this plan.
	ChangesetSpec *btypes.ChangesetSpec

	// The operations that need to be done to reconcile the changeset.
	Ops Operations

	// The Delta between a possible previous ChangesetSpec and the current
	// ChangesetSpec.
	Delta *ChangesetSpecDelta
}

func (p *Plan) AddOp(op btypes.ReconcilerOperation) { p.Ops = append(p.Ops, op) }
func (p *Plan) SetOp(op btypes.ReconcilerOperation) { p.Ops = Operations{op} }

// DeterminePlan looks at the given changeset to determine what action the
// reconciler should take.
// It consumes the current and the previous changeset spec, if they exist. If
// the current ChangesetSpec is not applied to a batch change, it returns an
// error.
func DeterminePlan(previousSpec, currentSpec *btypes.ChangesetSpec, ch *btypes.Changeset) (*Plan, error) {
	pl := &Plan{
		Changeset:     ch,
		ChangesetSpec: currentSpec,
	}

	wantDetach := false
	wantClose := false
	wantArchive := false
	isArchived := false
	isStillAttached := false
	wantDetachFromOwnerBatchChange := false
	for _, assoc := range ch.BatchChanges {
		if assoc.Detach {
			wantDetach = true
			if assoc.BatchChangeID == ch.OwnedByBatchChangeID {
				wantDetachFromOwnerBatchChange = true
			}
		} else if assoc.Archive && assoc.BatchChangeID == ch.OwnedByBatchChangeID && ch.Published() {
			wantArchive = !assoc.IsArchived
			isArchived = assoc.IsArchived
			if wantArchive {
				// If the changeset hasn't been closed/merged yet, we close it.
				// Marking it as Closing would be a noop, but it's weird to show a
				// changeset as will-be-closed on the preview page when it's
				// already closed.
				if ch.Published() && ch.Closeable() {
					wantClose = true
				}
			}
		} else {
			isStillAttached = true
		}
	}
	if wantDetach {
		pl.SetOp(btypes.ReconcilerOperationDetach)
	}

	if wantArchive {
		pl.AddOp(btypes.ReconcilerOperationArchive)
	}

	if wantClose {
		pl.AddOp(btypes.ReconcilerOperationClose)
		// Close is a final operation, nothing else should overwrite it.
		return pl, nil
	} else if wantDetachFromOwnerBatchChange || wantArchive || isArchived {
		// If the owner batch change detaches the changeset, we don't need to do
		// any additional writing operations, we can just return operation
		// "detach".
		// If some other batch change detached, but the owner batch change
		// didn't, detach, update is a valid combination, since we'll detach
		// from one batch change but still update the changeset because the
		// owning batch change changed the spec.
		return pl, nil
	}

	// If it doesn't have a spec, it's an imported changeset and we can't do
	// anything.
	if currentSpec == nil {
		// If still more than one remains attached, we still want to import the changeset.
		if ch.Unpublished() && isStillAttached {
			pl.AddOp(btypes.ReconcilerOperationImport)
		}
		return pl, nil
	}

	delta, err := compareChangesetSpecs(previousSpec, currentSpec)
	if err != nil {
		return pl, nil
	}
	pl.Delta = delta

	switch ch.PublicationState {
	case btypes.ChangesetPublicationStateUnpublished:
		if currentSpec.Spec.Published.True() {
			pl.SetOp(btypes.ReconcilerOperationPublish)
			pl.AddOp(btypes.ReconcilerOperationPush)
		} else if currentSpec.Spec.Published.Draft() && ch.SupportsDraft() {
			// If configured to be opened as draft, and the changeset supports
			// draft mode, publish as draft. Otherwise, take no action.
			pl.SetOp(btypes.ReconcilerOperationPublishDraft)
			pl.AddOp(btypes.ReconcilerOperationPush)
		}

	case btypes.ChangesetPublicationStatePublished:
		// Don't take any actions for merged changesets.
		if ch.ExternalState == btypes.ChangesetExternalStateMerged {
			return pl, nil
		}
		if reopenAfterDetach(ch, delta) {
			pl.SetOp(btypes.ReconcilerOperationReopen)
		}

		// If was set to "draft" and now "true", need to undraft the changeset.
		if previousSpec.Spec.Published.Draft() && currentSpec.Spec.Published.True() {
			if ch.SupportsDraft() {
				pl.AddOp(btypes.ReconcilerOperationUndraft)
			}
		} else if previousSpec.Spec.Published.True() && currentSpec.Spec.Published.Draft() {
			if ch.SupportsDraft() {
				pl.AddOp(btypes.ReconcilerOperationRedraft)
			}
		} else if !previousSpec.Spec.Published.False() && currentSpec.Spec.Published.False() {
			pl.AddOp(btypes.ReconcilerOperationClose)
		} else if previousSpec.Spec.Published.False() && !currentSpec.Spec.Published.False() {
			pl.AddOp(btypes.ReconcilerOperationReopen)
			if ch.SupportsDraft() {
				isDraft, err := changesetIsDraft(ch)
				if err != nil {
					return nil, err
				}
				if currentSpec.Spec.Published.Draft() {
					if !isDraft {
						pl.AddOp(btypes.ReconcilerOperationRedraft)
					}
				} else if isDraft {
					pl.AddOp(btypes.ReconcilerOperationUndraft)
				}
			}
		}

		if delta.AttributesChanged() {
			if delta.NeedCommitUpdate() {
				pl.AddOp(btypes.ReconcilerOperationPush)
			}

			// If we only need to update the diff and we didn't change the state of the changeset,
			// we're done, because we already pushed the commit. We don't need to
			// update anything on the codehost.
			if !delta.NeedCodeHostUpdate() {
				// But we need to sync the changeset so that it has the new commit.
				//
				// The problem: the code host might not have updated the changeset to
				// have the new commit SHA as its head ref oid (and the check states,
				// ...).
				//
				// That's why we give them 3 seconds to update the changesets.
				//
				// Why 3 seconds? Well... 1 or 2 seem to be too short and 4 too long?
				pl.AddOp(btypes.ReconcilerOperationSleep)
				pl.AddOp(btypes.ReconcilerOperationSync)
			} else {
				// Otherwise, we need to update the pull request on the code host or, if we
				// need to reopen it, update it to make sure it has the newest state.
				pl.AddOp(btypes.ReconcilerOperationUpdate)
			}
		}

	default:
		return pl, fmt.Errorf("unknown changeset publication state: %s", ch.PublicationState)
	}

	return pl, nil
}

func reopenAfterDetach(ch *btypes.Changeset, delta *ChangesetSpecDelta) bool {
	closed := ch.ExternalState == btypes.ChangesetExternalStateClosed
	if !closed {
		return false
	}

	// Sanity check: if it's not owned by a batch change, it's simply being tracked.
	if ch.OwnedByBatchChangeID == 0 {
		return false
	}
	// Sanity check 2: if it's marked as to-be-closed, then we don't reopen it.
	if ch.Closing {
		return false
	}

	// At this point the changeset is closed and not marked as to-be-closed.

	// TODO: What if somebody closed the changeset on purpose on the codehost?
	return ch.AttachedTo(ch.OwnedByBatchChangeID) && delta.Reopen
}

func compareChangesetSpecs(previous, current *btypes.ChangesetSpec) (*ChangesetSpecDelta, error) {
	delta := &ChangesetSpecDelta{}

	if previous == nil {
		return delta, nil
	}

	if previous.Spec.Title != current.Spec.Title {
		delta.TitleChanged = true
	}
	if previous.Spec.Body != current.Spec.Body {
		delta.BodyChanged = true
	}
	if previous.Spec.BaseRef != current.Spec.BaseRef {
		delta.BaseRefChanged = true
	}

	// Diff
	currentDiff, err := current.Spec.Diff()
	if err != nil {
		return nil, nil
	}
	previousDiff, err := previous.Spec.Diff()
	if err != nil {
		return nil, err
	}
	if previousDiff != currentDiff {
		delta.DiffChanged = true
	}

	// CommitMessage
	currentCommitMessage, err := current.Spec.CommitMessage()
	if err != nil {
		return nil, nil
	}
	previousCommitMessage, err := previous.Spec.CommitMessage()
	if err != nil {
		return nil, err
	}
	if previousCommitMessage != currentCommitMessage {
		delta.CommitMessageChanged = true
	}

	// AuthorName
	currentAuthorName, err := current.Spec.AuthorName()
	if err != nil {
		return nil, nil
	}
	previousAuthorName, err := previous.Spec.AuthorName()
	if err != nil {
		return nil, err
	}
	if previousAuthorName != currentAuthorName {
		delta.AuthorNameChanged = true
	}

	// AuthorEmail
	currentAuthorEmail, err := current.Spec.AuthorEmail()
	if err != nil {
		return nil, nil
	}
	previousAuthorEmail, err := previous.Spec.AuthorEmail()
	if err != nil {
		return nil, err
	}
	if previousAuthorEmail != currentAuthorEmail {
		delta.AuthorEmailChanged = true
	}

	return delta, nil
}

type ChangesetSpecDelta struct {
	TitleChanged         bool
	BodyChanged          bool
	BaseRefChanged       bool
	DiffChanged          bool
	CommitMessageChanged bool
	AuthorNameChanged    bool
	AuthorEmailChanged   bool
}

func (d *ChangesetSpecDelta) String() string { return fmt.Sprintf("%#v", d) }

func (d *ChangesetSpecDelta) NeedCommitUpdate() bool {
	return d.DiffChanged || d.CommitMessageChanged || d.AuthorNameChanged || d.AuthorEmailChanged
}

func (d *ChangesetSpecDelta) NeedCodeHostUpdate() bool {
	return d.TitleChanged || d.BodyChanged || d.BaseRefChanged
}

func (d *ChangesetSpecDelta) AttributesChanged() bool {
	return d.NeedCommitUpdate() || d.NeedCodeHostUpdate()
}

func changesetIsDraft(ch *btypes.Changeset) (bool, error) {
	switch m := ch.Metadata.(type) {
	case *github.PullRequest:
		return m.IsDraft, nil
	case *gitlab.MergeRequest:
		return m.WorkInProgress, nil
	case *bitbucketserver.PullRequest:
		return false, nil

	default:
		return false, fmt.Errorf("invalid changeset metadata %T", ch.Metadata)
	}
}
