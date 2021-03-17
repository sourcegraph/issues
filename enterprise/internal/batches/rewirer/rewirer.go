package rewirer

import (
	"fmt"

	"github.com/sourcegraph/sourcegraph/enterprise/internal/batches/store"
	"github.com/sourcegraph/sourcegraph/internal/batches"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/types"
)

type ChangesetRewirer struct {
	// The mappings need to be hydrated for the ChangesetRewirer to consume them.
	mappings      store.RewirerMappings
	batchChangeID int64

	// Feature flag that can be removed once this is the default behaviour
	ArchiveInsteadOfDetach bool
}

func New(mappings store.RewirerMappings, batchChangeID int64) *ChangesetRewirer {
	return &ChangesetRewirer{
		mappings:      mappings,
		batchChangeID: batchChangeID,
	}
}

// Rewire uses RewirerMappings (mapping ChangesetSpecs to matching Changesets) generated by Store.GetRewirerMappings to update the Changesets
// for consumption by the background reconciler.
//
// It also updates the ChangesetIDs on the batch change.
func (r *ChangesetRewirer) Rewire() (changesets []*batches.Changeset, err error) {
	changesets = []*batches.Changeset{}

	for _, m := range r.mappings {
		// If a Changeset that's currently attached to the batch change wasn't matched to a ChangesetSpec, it needs to be closed/detached.
		if m.ChangesetSpec == nil {
			changeset := m.Changeset

			// If we don't have access to a repository, we don't detach nor close the changeset.
			if m.Repo == nil {
				continue
			}

			// If the changeset is currently not attached to this batch change, we don't want to modify it.
			if !changeset.AttachedTo(r.batchChangeID) {
				continue
			}

			r.closeChangeset(changeset)
			changesets = append(changesets, changeset)

			continue
		}

		spec := m.ChangesetSpec

		// If we don't have access to a repository, we return an error. Why not
		// simply skip the repository? If we skip it, the user can't reapply
		// the same batch spec, since it's already applied and re-applying
		// would require a new spec.
		repo := m.Repo
		if repo == nil {
			return nil, &database.RepoNotFoundErr{ID: m.RepoID}
		}

		if err := checkRepoSupported(repo); err != nil {
			return nil, err
		}

		var changeset *batches.Changeset

		if m.Changeset != nil {
			changeset = m.Changeset
			if spec.Spec.IsImportingExisting() {
				r.attachTrackingChangeset(changeset)
			} else if spec.Spec.IsBranch() {
				r.updateChangesetToNewSpec(changeset, spec)
			}
		} else {
			if spec.Spec.IsImportingExisting() {
				changeset = r.createTrackingChangeset(repo, spec.Spec.ExternalID)
			} else if spec.Spec.IsBranch() {
				changeset = r.createChangesetForSpec(repo, spec)
			}
		}
		changesets = append(changesets, changeset)
	}

	return changesets, nil
}

func (r *ChangesetRewirer) createChangesetForSpec(repo *types.Repo, spec *batches.ChangesetSpec) *batches.Changeset {
	newChangeset := &batches.Changeset{
		RepoID:              spec.RepoID,
		ExternalServiceType: repo.ExternalRepo.ServiceType,

		BatchChanges:         []batches.BatchChangeAssoc{{BatchChangeID: r.batchChangeID}},
		OwnedByBatchChangeID: r.batchChangeID,
		CurrentSpecID:        spec.ID,

		PublicationState: batches.ChangesetPublicationStateUnpublished,
		ReconcilerState:  batches.ReconcilerStateQueued,
	}

	// Copy over diff stat from the spec.
	diffStat := spec.DiffStat()
	newChangeset.SetDiffStat(&diffStat)

	return newChangeset
}

func (r *ChangesetRewirer) updateChangesetToNewSpec(c *batches.Changeset, spec *batches.ChangesetSpec) {
	if c.ReconcilerState == batches.ReconcilerStateCompleted {
		c.PreviousSpecID = c.CurrentSpecID
	}
	c.CurrentSpecID = spec.ID

	// Ensure that the changeset is attached to the batch change
	c.Attach(r.batchChangeID)

	// We need to enqueue it for the changeset reconciler, so the
	// reconciler wakes up, compares old and new spec and, if
	// necessary, updates the changesets accordingly.
	c.ResetQueued()
}

func (r *ChangesetRewirer) createTrackingChangeset(repo *types.Repo, externalID string) *batches.Changeset {
	newChangeset := &batches.Changeset{
		RepoID:              repo.ID,
		ExternalServiceType: repo.ExternalRepo.ServiceType,

		BatchChanges: []batches.BatchChangeAssoc{{BatchChangeID: r.batchChangeID}},
		ExternalID:   externalID,
		// Note: no CurrentSpecID, because we merely track this one

		PublicationState: batches.ChangesetPublicationStateUnpublished,

		// Enqueue it so the reconciler syncs it.
		ReconcilerState: batches.ReconcilerStateQueued,
	}

	return newChangeset
}

func (r *ChangesetRewirer) attachTrackingChangeset(changeset *batches.Changeset) {
	// We already have a changeset with the given repoID and
	// externalID, so we can track it.
	changeset.Attach(r.batchChangeID)

	// If it's errored and not created by another batch change, we re-enqueue it.
	if changeset.OwnedByBatchChangeID == 0 && (changeset.ReconcilerState == batches.ReconcilerStateErrored || changeset.ReconcilerState == batches.ReconcilerStateFailed) {
		changeset.ResetQueued()
	}
}

func (r *ChangesetRewirer) closeChangeset(changeset *batches.Changeset) {
	reset := false
	if changeset.CurrentSpecID != 0 && changeset.OwnedByBatchChangeID == r.batchChangeID {
		// If we have a current spec ID and the changeset was created by
		// _this_ batch change that means we should detach and close it.
		if changeset.Published() {
			// Store the current spec also as the previous spec.
			//
			// Why?
			//
			// When a changeset with (prev: A, curr: B) should be closed but
			// closing failed, it will still have (prev: A, curr: B) set.
			//
			// If someone then applies a new batch spec and re-attaches that
			// changeset with changeset spec C, the changeset would end up with
			// (prev: A, curr: C), because we don't rotate specs on errors in
			// `updateChangesetToNewSpec`.
			//
			// That would mean, though, that the delta between A and C tells us
			// to repush and update the changeset on the code host, in addition
			// to 'reopen', which would actually be the only required action.
			//
			// So, when we mark a changeset as to-be-closed, we also rotate the
			// specs, so that it changeset is saved as (prev: B, curr: B) and
			// when somebody re-attaches it it's (prev: B, curr: C).
			// But we only rotate the spec, if applying the currentSpecID was
			// successful:
			if changeset.ReconcilerState == batches.ReconcilerStateCompleted {
				changeset.PreviousSpecID = changeset.CurrentSpecID
			}
			changeset.Closing = true
			reset = true
		}
	}

	// Disassociate the changeset with the batch change.
	if wasAttached := changeset.Detach(r.batchChangeID); wasAttached {
		reset = true
	}

	if reset {
		changeset.ResetQueued()
	}
}

// ErrRepoNotSupported is thrown by the rewirer when it encounters a mapping
// targetting a repo on a code host that's not supported by batches.
type ErrRepoNotSupported struct {
	ServiceType string
	RepoName    string
}

func (e ErrRepoNotSupported) Error() string {
	return fmt.Sprintf(
		"Code host type %s of repository %q is currently not supported for use with Batch Changes",
		e.ServiceType,
		e.RepoName,
	)
}

var _ error = ErrRepoNotSupported{}

// checkRepoSupported checks whether the given repository is supported by batch
// changes and if not it returns an error.
func checkRepoSupported(repo *types.Repo) error {
	if batches.IsRepoSupported(&repo.ExternalRepo) {
		return nil
	}

	return &ErrRepoNotSupported{
		ServiceType: repo.ExternalRepo.ServiceType,
		RepoName:    string(repo.Name),
	}
}
