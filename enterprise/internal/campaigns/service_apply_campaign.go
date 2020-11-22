package campaigns

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pkg/errors"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/cmd/repo-updater/repos"
	"github.com/sourcegraph/sourcegraph/internal/actor"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/campaigns"
	"github.com/sourcegraph/sourcegraph/internal/db"
	"github.com/sourcegraph/sourcegraph/internal/trace"
	"github.com/sourcegraph/sourcegraph/internal/types"
)

// ErrApplyClosedCampaign is returned by ApplyCampaign when the campaign
// matched by the campaign spec is already closed.
var ErrApplyClosedCampaign = errors.New("existing campaign matched by campaign spec is closed")

// ErrMatchingCampaignExists is returned by ApplyCampaign if a campaign matching the
// campaign spec already exists and FailIfExists was set.
var ErrMatchingCampaignExists = errors.New("a campaign matching the given campaign spec already exists")

type ApplyCampaignOpts struct {
	CampaignSpecRandID string
	EnsureCampaignID   int64

	// When FailIfCampaignExists is true, ApplyCampaign will fail if a Campaign
	// matching the given CampaignSpec already exists.
	FailIfCampaignExists bool
}

func (o ApplyCampaignOpts) String() string {
	return fmt.Sprintf(
		"CampaignSpec %s, EnsureCampaignID %d",
		o.CampaignSpecRandID,
		o.EnsureCampaignID,
	)
}

// ApplyCampaign creates the CampaignSpec.
func (s *Service) ApplyCampaign(ctx context.Context, opts ApplyCampaignOpts) (campaign *campaigns.Campaign, err error) {
	tr, ctx := trace.New(ctx, "Service.ApplyCampaign", opts.String())
	defer func() {
		tr.SetError(err)
		tr.Finish()
	}()

	campaignSpec, err := s.store.GetCampaignSpec(ctx, GetCampaignSpecOpts{
		RandID: opts.CampaignSpecRandID,
	})
	if err != nil {
		return nil, err
	}

	// 🚨 SECURITY: Only site-admins or the creator of campaignSpec can apply
	// campaignSpec.
	if err := backend.CheckSiteAdminOrSameUser(ctx, campaignSpec.UserID); err != nil {
		return nil, err
	}

	campaign, err = s.ReconcileCampaign(ctx, campaignSpec)
	if err != nil {
		return nil, err
	}

	if campaign.ID != 0 && opts.FailIfCampaignExists {
		return nil, ErrMatchingCampaignExists
	}

	if opts.EnsureCampaignID != 0 && campaign.ID != opts.EnsureCampaignID {
		return nil, ErrEnsureCampaignFailed
	}

	if campaign.Closed() {
		return nil, ErrApplyClosedCampaign
	}

	// if campaign.CampaignSpecID == campaignSpec.ID {
	// 	return campaign, nil
	// }

	// Before we write to the database in a transaction, we cancel all
	// currently enqueued/errored-and-retryable changesets the campaign might
	// have.
	// We do this so we don't continue to possibly create changesets on the
	// codehost while we're applying a new campaign spec.
	// This is blocking, because the changeset rows currently being processed by the
	// reconciler are locked.
	if err := s.store.CancelQueuedCampaignChangesets(ctx, campaign.ID); err != nil {
		return campaign, nil
	}

	tx, err := s.store.Transact(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { err = tx.Done(err) }()

	if campaign.ID == 0 {
		err := tx.CreateCampaign(ctx, campaign)
		if err != nil {
			return nil, err
		}
	}

	rstore := repos.NewDBStore(tx.DB(), sql.TxOptions{})

	// Now we need to wire up the ChangesetSpecs of the new CampaignSpec
	// correctly with the Changesets so that the reconciler can create/update
	// them.

	// Load the mapping between changeset specs and existing changesets in the target campaign.
	mappings, err := tx.GetRewirerMappings(ctx, GetRewirerMappingsOpts{
		CampaignSpecID: campaign.CampaignSpecID,
		CampaignID:     campaign.ID,
	})
	if err != nil {
		return nil, err
	}

	// And execute the mapping.
	rewirer := &ChangesetRewirer{
		TX:       tx,
		RStore:   rstore,
		Campaign: campaign,
		Mappings: mappings,
	}
	changesets, err := rewirer.Rewire(ctx)
	if err != nil {
		return nil, err
	}

	for _, changeset := range changesets {
		if err := tx.UpsertChangeset(ctx, changeset); err != nil {
			return nil, err
		}
		campaign.ChangesetIDs = append(campaign.ChangesetIDs, changeset.ID)
	}

	if err := tx.UpdateCampaign(ctx, campaign); err != nil {
		return nil, err
	}

	return campaign, nil
}

func (s *Service) ReconcileCampaign(ctx context.Context, campaignSpec *campaigns.CampaignSpec) (*campaigns.Campaign, error) {
	campaign, err := s.GetCampaignMatchingCampaignSpec(ctx, campaignSpec)
	if err != nil {
		return nil, err
	}
	if campaign == nil {
		campaign = &campaigns.Campaign{}
	}
	// Populate the campaign with the values from the campaign spec.
	campaign.CampaignSpecID = campaignSpec.ID
	campaign.NamespaceOrgID = campaignSpec.NamespaceOrgID
	campaign.NamespaceUserID = campaignSpec.NamespaceUserID
	campaign.Name = campaignSpec.Spec.Name
	actor := actor.FromContext(ctx)
	if campaign.InitialApplierID == 0 {
		campaign.InitialApplierID = actor.UID
	}
	campaign.LastApplierID = actor.UID
	campaign.LastAppliedAt = s.clock()
	campaign.Description = campaignSpec.Spec.Description
	return campaign, nil
}

type ChangesetRewirer struct {
	Mappings campaigns.RewirerMappings
	Campaign *campaigns.Campaign
	TX       *Store
	RStore   repos.Store
}

// Rewire uses mappings generated by Store.GetRewirerMappings to prepare the changeset rows
// for consumption by the background reconciler.
//
// It also updates the ChangesetIDs on the campaign.
func (r *ChangesetRewirer) Rewire(ctx context.Context) (changesets []*campaigns.Changeset, err error) {
	// First we need to load the associations.
	accessibleReposByID, changesetsByID, changesetSpecsByID, err := r.loadAssociations(ctx)
	if err != nil {
		return nil, err
	}

	// Reset the attached changesets. We will add all we encounter while processing the mappings to this list again.
	r.Campaign.ChangesetIDs = []int64{}

	changesets = []*campaigns.Changeset{}

	for _, m := range r.Mappings {
		// If no changeset spec matched, a changeset must have matched, and it needs to be closed/detached.
		if m.ChangesetSpecID == 0 {
			changeset, ok := changesetsByID[m.ChangesetID]
			if !ok {
				// This should never happen.
				return nil, errors.New("changeset not found")
			}

			// If we don't have access to a repository, we don't detach nor close the changeset.
			_, ok = accessibleReposByID[m.RepoID]
			if !ok {
				continue
			}

			if c, err := r.closeChangeset(ctx, changeset); err != nil {
				return nil, err
			} else if c != nil {
				changesets = append(changesets, c)
			}

			continue
		}

		spec, ok := changesetSpecsByID[m.ChangesetSpecID]
		if !ok {
			// This should never happen.
			return nil, errors.New("spec not found")
		}

		// If we don't have access to a repository, we return an error. Why not
		// simply skip the repository? If we skip it, the user can't reapply
		// the same campaign spec, since it's already applied and re-applying
		// would require a new spec.
		repo, ok := accessibleReposByID[m.RepoID]
		if !ok {
			return nil, &db.RepoNotFoundErr{ID: m.RepoID}
		}

		if err := checkRepoSupported(repo); err != nil {
			return nil, err
		}

		var changeset *campaigns.Changeset

		if m.ChangesetID != 0 {
			changeset, ok = changesetsByID[m.ChangesetID]
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

func (r *ChangesetRewirer) createChangesetForSpec(repo *types.Repo, spec *campaigns.ChangesetSpec) *campaigns.Changeset {
	newChangeset := &campaigns.Changeset{
		RepoID:              spec.RepoID,
		ExternalServiceType: repo.ExternalRepo.ServiceType,

		CampaignIDs:       []int64{r.Campaign.ID},
		OwnedByCampaignID: r.Campaign.ID,
		CurrentSpecID:     spec.ID,

		PublicationState: campaigns.ChangesetPublicationStateUnpublished,
		ReconcilerState:  campaigns.ReconcilerStateQueued,
	}

	// Copy over diff stat from the spec.
	diffStat := spec.DiffStat()
	newChangeset.SetDiffStat(&diffStat)

	return newChangeset
}

func (r *ChangesetRewirer) updateChangesetToNewSpec(c *campaigns.Changeset, spec *campaigns.ChangesetSpec) {
	if c.ReconcilerState != campaigns.ReconcilerStateErrored && c.ReconcilerState != campaigns.ReconcilerStateQueued {
		c.PreviousSpecID = c.CurrentSpecID
	}

	c.CurrentSpecID = spec.ID

	// Ensure that the changeset is attached to the campaign
	c.CampaignIDs = append(c.CampaignIDs, r.Campaign.ID)

	// Copy over diff stat from the new spec.
	diffStat := spec.DiffStat()
	c.SetDiffStat(&diffStat)

	// We need to enqueue it for the changeset reconciler, so the
	// reconciler wakes up, compares old and new spec and, if
	// necessary, updates the changesets accordingly.
	c.ResetQueued()
}

func (r *ChangesetRewirer) createTrackingChangeset(repo *types.Repo, externalID string) *campaigns.Changeset {
	newChangeset := &campaigns.Changeset{
		RepoID:              repo.ID,
		ExternalServiceType: repo.ExternalRepo.ServiceType,

		CampaignIDs:     []int64{r.Campaign.ID},
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

func (r *ChangesetRewirer) attachTrackingChangeset(changeset *campaigns.Changeset) {
	// We already have a changeset with the given repoID and
	// externalID, so we can track it.
	changeset.AddedToCampaign = true
	changeset.CampaignIDs = append(changeset.CampaignIDs, r.Campaign.ID)

	// If it's errored and not created by another campaign, we re-enqueue it.
	if changeset.OwnedByCampaignID == 0 && changeset.ReconcilerState == campaigns.ReconcilerStateErrored {
		changeset.ResetQueued()
	}
}

func (r *ChangesetRewirer) closeChangeset(ctx context.Context, changeset *campaigns.Changeset) (*campaigns.Changeset, error) {
	if changeset.CurrentSpecID != 0 && changeset.OwnedByCampaignID == r.Campaign.ID {
		// If we have a current spec ID and the changeset was created by
		// _this_ campaign that means we should detach and close it.

		// But only if it was created on the code host:
		if changeset.Published() && changeset.ExternalState == campaigns.ChangesetExternalStateOpen {
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
			return nil, r.TX.DeleteChangeset(ctx, changeset.ID)
		}
	}

	// Disassociate the changeset with the campaign.
	changeset.RemoveCampaignID(r.Campaign.ID)
	return changeset, nil
}

// loadAssociations retrieves all entities required to rewire a campaign.
func (r *ChangesetRewirer) loadAssociations(ctx context.Context) (
	accessibleReposByID map[api.RepoID]*types.Repo,
	changesetsByID map[int64]*campaigns.Changeset,
	changesetSpecsByID map[int64]*campaigns.ChangesetSpec,
	err error,
) {
	// Fetch the changeset specs involved in this rewiring.
	changesetSpecs, _, err := r.TX.ListChangesetSpecs(ctx, ListChangesetSpecsOpts{
		CampaignSpecID: r.Campaign.CampaignSpecID,
		IDs:            r.Mappings.ChangesetSpecIDs(),
	})
	if err != nil {
		return accessibleReposByID, changesetsByID, changesetSpecsByID, err
	}

	// Then fetch the changesets involved in this rewiring.
	changesets, _, err := r.TX.ListChangesets(ctx, ListChangesetsOpts{IDs: r.Mappings.ChangesetIDs()})
	if err != nil {
		return accessibleReposByID, changesetsByID, changesetSpecsByID, err
	}

	// Fetch all repos involved. We use them later to enforce repo permissions.
	//
	// 🚨 SECURITY: db.Repos.GetRepoIDsSet uses the authzFilter under the hood and
	// filters out repositories that the user doesn't have access to.
	accessibleReposByID, err = db.Repos.GetReposSetByIDs(ctx, r.Mappings.RepoIDs()...)
	if err != nil {
		return accessibleReposByID, changesetsByID, changesetSpecsByID, err
	}

	changesetsByID = map[int64]*campaigns.Changeset{}
	changesetSpecsByID = map[int64]*campaigns.ChangesetSpec{}

	for _, c := range changesets {
		changesetsByID[c.ID] = c
	}
	for _, c := range changesetSpecs {
		changesetSpecsByID[c.ID] = c
	}

	return accessibleReposByID, changesetsByID, changesetSpecsByID, nil
}
