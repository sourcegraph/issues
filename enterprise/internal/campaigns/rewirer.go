package campaigns

import (
	"context"

	"github.com/pkg/errors"
	"github.com/sourcegraph/sourcegraph/cmd/repo-updater/repos"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/campaigns"
	"github.com/sourcegraph/sourcegraph/internal/db"
	"github.com/sourcegraph/sourcegraph/internal/types"
)

// RewirerMapping maps a connection between ChangesetSpec and Changeset.
// If the ChangesetSpec doesn't match a Changeset (ie. it describes a to-be-created Changeset), ChangesetID is 0.
// If the ChangesetSpec is 0, the Changeset will be non-zero and means "to be closed".
// If both are non-zero values, the changeset should be updated with the changeset spec in the mapping.
type RewirerMapping struct {
	ChangesetSpecID int64
	ChangesetID     int64
	RepoID          api.RepoID
}

type RewirerMappings []*RewirerMapping

// ChangesetIDs returns a list of unique changeset IDs in the slice of mappings.
func (rm RewirerMappings) ChangesetIDs() []int64 {
	changesetIDMap := make(map[int64]struct{})
	for _, m := range rm {
		if m.ChangesetID != 0 {
			changesetIDMap[m.ChangesetID] = struct{}{}
		}
	}
	changesetIDs := make([]int64, len(changesetIDMap))
	for id := range changesetIDMap {
		changesetIDs = append(changesetIDs, id)
	}
	return changesetIDs
}

// ChangesetSpecIDs returns a list of unique changeset spec IDs in the slice of mappings.
func (rm RewirerMappings) ChangesetSpecIDs() []int64 {
	changesetSpecIDMap := make(map[int64]struct{})
	for _, m := range rm {
		if m.ChangesetSpecID != 0 {
			changesetSpecIDMap[m.ChangesetSpecID] = struct{}{}
		}
	}
	changesetSpecIDs := make([]int64, len(changesetSpecIDMap))
	for id := range changesetSpecIDMap {
		changesetSpecIDs = append(changesetSpecIDs, id)
	}
	return changesetSpecIDs
}

// RepoIDs returns a list of unique repo IDs in the slice of mappings.
func (rm RewirerMappings) RepoIDs() []api.RepoID {
	repoIDMap := make(map[api.RepoID]struct{})
	for _, m := range rm {
		repoIDMap[m.RepoID] = struct{}{}
	}
	repoIDs := make([]api.RepoID, len(repoIDMap))
	for id := range repoIDMap {
		repoIDs = append(repoIDs, id)
	}
	return repoIDs
}

type changesetRewirer struct {
	mappings RewirerMappings
	campaign *campaigns.Campaign
	tx       *Store
	rstore   repos.Store
}

func NewChangesetRewirer(mappings RewirerMappings, campaign *campaigns.Campaign, tx *Store, rstore repos.Store) *changesetRewirer {
	return &changesetRewirer{
		mappings: mappings,
		campaign: campaign,
		tx:       tx,
		rstore:   rstore,
	}
}

// Rewire uses RewirerMappings (mapping ChangesetSpecs to matching Changesets) generated by Store.GetRewirerMappings to update the Changesets
// for consumption by the background reconciler.
//
// It also updates the ChangesetIDs on the campaign.
func (r *changesetRewirer) Rewire(ctx context.Context) (changesets []*campaigns.Changeset, err error) {
	// First we need to load the associations.
	associations, err := r.loadAssociations(ctx)
	if err != nil {
		return nil, err
	}

	changesets = []*campaigns.Changeset{}

	for _, m := range r.mappings {
		// If a Changeset that's currently attached to the campaign wasn't matched to a ChangesetSpec, it needs to be closed/detached.
		if m.ChangesetSpecID == 0 {
			changeset, ok := associations.changesetsByID[m.ChangesetID]
			if !ok {
				// This should never happen.
				return nil, errors.New("changeset not found")
			}

			// If we don't have access to a repository, we don't detach nor close the changeset.
			_, ok = associations.accessibleReposByID[m.RepoID]
			if !ok {
				continue
			}

			if err := r.closeChangeset(ctx, changeset); err != nil {
				return nil, err
			}

			continue
		}

		spec, ok := associations.changesetSpecsByID[m.ChangesetSpecID]
		if !ok {
			// This should never happen.
			return nil, errors.New("spec not found")
		}

		// If we don't have access to a repository, we return an error. Why not
		// simply skip the repository? If we skip it, the user can't reapply
		// the same campaign spec, since it's already applied and re-applying
		// would require a new spec.
		repo, ok := associations.accessibleReposByID[m.RepoID]
		if !ok {
			return nil, &db.RepoNotFoundErr{ID: m.RepoID}
		}

		if err := checkRepoSupported(repo); err != nil {
			return nil, err
		}

		var changeset *campaigns.Changeset

		if m.ChangesetID != 0 {
			changeset, ok = associations.changesetsByID[m.ChangesetID]
			if !ok {
				// This should never happen.
				return nil, errors.New("changeset not found")
			}
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

func (r *changesetRewirer) createChangesetForSpec(repo *types.Repo, spec *campaigns.ChangesetSpec) *campaigns.Changeset {
	newChangeset := &campaigns.Changeset{
		RepoID:              spec.RepoID,
		ExternalServiceType: repo.ExternalRepo.ServiceType,

		CampaignIDs:       []int64{r.campaign.ID},
		OwnedByCampaignID: r.campaign.ID,
		CurrentSpecID:     spec.ID,

		PublicationState: campaigns.ChangesetPublicationStateUnpublished,
		ReconcilerState:  campaigns.ReconcilerStateQueued,
	}

	// Copy over diff stat from the spec.
	diffStat := spec.DiffStat()
	newChangeset.SetDiffStat(&diffStat)

	return newChangeset
}

func (r *changesetRewirer) updateChangesetToNewSpec(c *campaigns.Changeset, spec *campaigns.ChangesetSpec) {
	if c.ReconcilerState == campaigns.ReconcilerStateCompleted {
		c.PreviousSpecID = c.CurrentSpecID
	}
	c.CurrentSpecID = spec.ID

	// Ensure that the changeset is attached to the campaign
	c.CampaignIDs = append(c.CampaignIDs, r.campaign.ID)

	// Copy over diff stat from the new spec.
	diffStat := spec.DiffStat()
	c.SetDiffStat(&diffStat)

	// We need to enqueue it for the changeset reconciler, so the
	// reconciler wakes up, compares old and new spec and, if
	// necessary, updates the changesets accordingly.
	c.ResetQueued()
}

func (r *changesetRewirer) createTrackingChangeset(repo *types.Repo, externalID string) *campaigns.Changeset {
	newChangeset := &campaigns.Changeset{
		RepoID:              repo.ID,
		ExternalServiceType: repo.ExternalRepo.ServiceType,

		CampaignIDs:     []int64{r.campaign.ID},
		ExternalID:      externalID,
		AddedToCampaign: true,
		// Note: no CurrentSpecID, because we merely track this one

		PublicationState: campaigns.ChangesetPublicationStatePublished,

		// Enqueue it so the reconciler syncs it.
		ReconcilerState: campaigns.ReconcilerStateQueued,
		Unsynced:        true,
	}

	return newChangeset
}

func (r *changesetRewirer) attachTrackingChangeset(changeset *campaigns.Changeset) {
	// We already have a changeset with the given repoID and
	// externalID, so we can track it.
	changeset.AddedToCampaign = true
	changeset.CampaignIDs = append(changeset.CampaignIDs, r.campaign.ID)

	// If it's errored and not created by another campaign, we re-enqueue it.
	if changeset.OwnedByCampaignID == 0 && changeset.ReconcilerState == campaigns.ReconcilerStateErrored {
		changeset.ResetQueued()
	}
}

func (r *changesetRewirer) closeChangeset(ctx context.Context, changeset *campaigns.Changeset) error {
	if changeset.CurrentSpecID != 0 && changeset.OwnedByCampaignID == r.campaign.ID {
		// If we have a current spec ID and the changeset was created by
		// _this_ campaign that means we should detach and close it.

		// But only if it was created on the code host:
		if changeset.Published() {
			// Store the current spec also as the previous spec.
			// Reason:
			// When a changeset with (prev: 0, curr: 1) should be closed but closing failed, it will still have (0, 1) set.
			// Now when someone comes by and reattaches that changeset, the changeset would have (prev: 0, curr: 1), because
			// we don't rotate specs on errors. That would mean, though, that the delta is big and we would repush and update on the code host,
			// in addition to 'reopen', which would actually be the only required action.
			changeset.PreviousSpecID = changeset.CurrentSpecID
			changeset.Closing = true
			changeset.ResetQueued()
		} else {
			// otherwise we simply delete it.
			return r.tx.DeleteChangeset(ctx, changeset.ID)
		}
	}

	// Disassociate the changeset with the campaign.
	changeset.RemoveCampaignID(r.campaign.ID)
	return r.tx.UpdateChangeset(ctx, changeset)
}

type rewirerAssociations struct {
	accessibleReposByID map[api.RepoID]*types.Repo
	changesetsByID      map[int64]*campaigns.Changeset
	changesetSpecsByID  map[int64]*campaigns.ChangesetSpec
}

// loadAssociations retrieves all entities required to rewire the changesets in a campaign.
func (r *changesetRewirer) loadAssociations(ctx context.Context) (associations *rewirerAssociations, err error) {
	// Fetch the changeset specs involved in this rewiring. This should always be the same as omitting the `IDs` section,
	// we just make sure people know why that is the case here.
	changesetSpecs, _, err := r.tx.ListChangesetSpecs(ctx, ListChangesetSpecsOpts{
		CampaignSpecID: r.campaign.CampaignSpecID,
		IDs:            r.mappings.ChangesetSpecIDs(),
	})
	if err != nil {
		return nil, err
	}

	// Then fetch the changesets involved in this rewiring.
	changesets, _, err := r.tx.ListChangesets(ctx, ListChangesetsOpts{IDs: r.mappings.ChangesetIDs()})
	if err != nil {
		return nil, err
	}

	associations = &rewirerAssociations{}
	// Fetch all repos involved. We use them later to enforce repo permissions.
	//
	// 🚨 SECURITY: db.Repos.GetRepoIDsSet uses the authzFilter under the hood and
	// filters out repositories that the user doesn't have access to.
	associations.accessibleReposByID, err = db.Repos.GetReposSetByIDs(ctx, r.mappings.RepoIDs()...)
	if err != nil {
		return nil, err
	}

	associations.changesetsByID = map[int64]*campaigns.Changeset{}
	associations.changesetSpecsByID = map[int64]*campaigns.ChangesetSpec{}

	for _, c := range changesets {
		associations.changesetsByID[c.ID] = c
	}
	for _, c := range changesetSpecs {
		associations.changesetSpecsByID[c.ID] = c
	}

	return associations, nil
}
