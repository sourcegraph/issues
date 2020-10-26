package campaigns

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pkg/errors"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/cmd/repo-updater/repos"
	"github.com/sourcegraph/sourcegraph/internal/actor"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/campaigns"
	"github.com/sourcegraph/sourcegraph/internal/db"
	"github.com/sourcegraph/sourcegraph/internal/trace"
	"github.com/sourcegraph/sourcegraph/internal/vcs/git"
)

// ErrApplyClosedCampaign is returned by ApplyCampaign when the campaign
// matched by the campaign spec is already closed.
var ErrApplyClosedCampaign = errors.New("existing campaign matched by campaign spec is closed")

// ErrMatchingCampaign is returned by ApplyCampaign if a campaign matching the
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

	campaign, err = s.GetCampaignMatchingCampaignSpec(ctx, s.store, campaignSpec)
	if err != nil {
		return nil, err
	}
	if campaign == nil {
		campaign = &campaigns.Campaign{}
	} else if opts.FailIfCampaignExists {
		return nil, ErrMatchingCampaignExists
	}

	if opts.EnsureCampaignID != 0 && campaign.ID != opts.EnsureCampaignID {
		return nil, ErrEnsureCampaignFailed
	}

	if campaign.Closed() {
		return nil, ErrApplyClosedCampaign
	}

	if campaign.CampaignSpecID == campaignSpec.ID {
		return campaign, nil
	}

	// Before we write to the database in a transaction, we cancel all
	// currently enqueued/errored-and-retryable changesets the campaign might
	// have.
	// We do this so we don't continue to possibly create changesets on the
	// codehost while we're applying a new campaign spec.
	if err := s.store.CancelQueuedCampaignChangesets(ctx, campaign.ID); err != nil {
		return campaign, nil
	}

	tx, err := s.store.Transact(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { err = tx.Done(err) }()

	rstore := repos.NewDBStore(tx.DB(), sql.TxOptions{})

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

	if campaign.ID == 0 {
		err := tx.CreateCampaign(ctx, campaign)
		if err != nil {
			return nil, err
		}
	}

	// Now we need to wire up the ChangesetSpecs of the new CampaignSpec
	// correctly with the Changesets so that the reconciler can create/update
	// them.
	rewirer := &changesetRewirer{
		ctx:      ctx,
		tx:       tx,
		rstore:   rstore,
		campaign: campaign,
	}

	if err := rewirer.Rewire(); err != nil {
		return nil, err
	}

	return campaign, nil
}

type repoHeadRef struct {
	repo    api.RepoID
	headRef string
}

type repoExternalID struct {
	repo       api.RepoID
	externalID string
}

type changesetRewirer struct {
	ctx      context.Context
	campaign *campaigns.Campaign
	tx       *Store
	rstore   repos.Store

	// These fields are populated by loadAssociations
	changesets          campaigns.Changesets
	newChangesetSpecs   campaigns.ChangesetSpecs
	accessibleReposByID map[api.RepoID]*types.Repo

	// These are populated by indexAssociations
	changesetsByRepoHeadRef    map[repoHeadRef]*campaigns.Changeset
	changesetsByRepoExternalID map[repoExternalID]*campaigns.Changeset
	currentSpecsByChangeset    map[int64]*campaigns.ChangesetSpec
}

// Rewire loads the current changesets of the given campaign, the changeset
// specs attached to the new campaign spec and rewires them so that the
// changesets are associated with the correct changeset specs and with the
// campaign.
//
// It also updates the ChangesetIDs on the campaign.
func (r *changesetRewirer) Rewire() (err error) {
	// First we need to load the associations
	if err := r.loadAssociations(); err != nil {
		return err
	}

	// Now we put them into buckets so we can match easily
	if err := r.indexAssociations(); err != nil {
		return err
	}

	// Now we have two lists, the current changesets and the new changeset specs:

	// ┌───────────────────────────────────────┐   ┌───────────────────────────────┐
	// │Changeset 1 | Repo A | #111 | run-gofmt│   │  Spec 1 | Repo A | run-gofmt  │
	// └───────────────────────────────────────┘   └───────────────────────────────┘
	// ┌───────────────────────────────────────┐   ┌───────────────────────────────┐
	// │Changeset 2 | Repo B |      | run-gofmt│   │  Spec 2 | Repo B | run-gofmt  │
	// └───────────────────────────────────────┘   └───────────────────────────────┘
	// ┌───────────────────────────────────────┐   ┌───────────────────────────────────┐
	// │Changeset 3 | Repo C | #222 | run-gofmt│   │  Spec 3 | Repo C | run-goimports  │
	// └───────────────────────────────────────┘   └───────────────────────────────────┘
	// ┌───────────────────────────────────────┐   ┌───────────────────────────────┐
	// │Changeset 4 | Repo C | #333 | older-pr │   │    Spec 4 | Repo C | #333     │
	// └───────────────────────────────────────┘   └───────────────────────────────┘

	// We need to:
	// 1. Find out whether our new specs should _update_ an existing
	//    changeset, or whether we need to create a new one.
	// 2. Since we can have multiple changesets per repository, we need to match
	//    based on repo and external ID.
	// 3. But if a changeset wasn't published yet, it doesn't have an external ID.
	//    In that case, we need to check whether the branch on which we _might_
	//    push the commit (because the changeset might not be published
	//    yet) is the same.

	// What we want:
	//
	// ┌───────────────────────────────────────┐    ┌───────────────────────────────┐
	// │Changeset 1 | Repo A | #111 | run-gofmt│───▶│  Spec 1 | Repo A | run-gofmt  │
	// └───────────────────────────────────────┘    └───────────────────────────────┘
	// ┌───────────────────────────────────────┐    ┌───────────────────────────────┐
	// │Changeset 2 | Repo B |      | run-gofmt│───▶│  Spec 2 | Repo B | run-gofmt  │
	// └───────────────────────────────────────┘    └───────────────────────────────┘
	// ┌───────────────────────────────────────┐
	// │Changeset 3 | Repo C | #222 | run-gofmt│
	// └───────────────────────────────────────┘
	// ┌───────────────────────────────────────┐    ┌───────────────────────────────┐
	// │Changeset 4 | Repo C | #333 | older-pr │───▶│    Spec 4 | Repo C | #333     │
	// └───────────────────────────────────────┘    └───────────────────────────────┘
	// ┌───────────────────────────────────────┐    ┌───────────────────────────────────┐
	// │Changeset 5 | Repo C | | run-goimports │───▶│  Spec 3 | Repo C | run-goimports  │
	// └───────────────────────────────────────┘    └───────────────────────────────────┘
	//
	// Spec 1 should be attached to Changeset 1 and (possibly) update its title/body/diff.
	// Spec 2 should be attached to Changeset 2 and publish it on the code host.
	// Spec 3 should get a new Changeset, since its branch doesn't match Changeset 3's branch.
	// Spec 4 should be attached to Changeset 4, since it tracks PR #333 in Repo C.
	// Changeset 3 doesn't have a matching spec and should be detached from the campaign (and closed).

	attachedChangesets := map[int64]bool{}
	for _, spec := range r.newChangesetSpecs {
		// If we don't have access to a repository, we return an error. Why not
		// simply skip the repository? If we skip it, the user can't reapply
		// the same campaign spec, since it's already applied and re-applying
		// would require a new spec.
		repo, ok := r.accessibleReposByID[spec.RepoID]
		if !ok {
			return &db.RepoNotFoundErr{ID: spec.RepoID}
		}

		if err := checkRepoSupported(repo); err != nil {
			return err
		}

		// If we need to track a changeset, we need to find it.
		if spec.Spec.IsImportingExisting() {
			k := repoExternalID{repo: spec.RepoID, externalID: spec.Spec.ExternalID}

			c, ok := r.changesetsByRepoExternalID[k]
			if ok {
				// If it's already attached to the campaign and errored, we re-enqueue it.
				if c.ReconcilerState == campaigns.ReconcilerStateErrored {
					if err := r.updateAndReenqueue(c); err != nil {
						return err
					}
				}
			} else {
				// If we don't have a changeset attached to the campaign, we need to find or create one with the externalID in that repository.
				c, err = r.updateOrCreateTrackingChangeset(repo, k.externalID)
				if err != nil {
					return err
				}
			}
			// If it's already attached to the campaign, we need to keep it
			// there. And if it's new, we want to attach it:
			attachedChangesets[c.ID] = true

			// We handled both cases for "track existing changeset" spec:
			// 1. Add existing changeset to campaign
			// 2. Create new changeset and sync it
			continue
		}

		// What we're now looking at is a spec that says:
		//   1. Create a PR on this branch in this repo with this title/body/diff
		// or, if the a PR on this branch with this repo already exists:
		//   2. Update the PR on this branch in this repo to have this new title/body/diff
		//
		// So, let's check:
		// Do we already have a changeset on this branch in this repo?
		k := repoHeadRef{repo: spec.RepoID, headRef: git.EnsureRefPrefix(spec.Spec.HeadRef)}
		c, ok := r.changesetsByRepoHeadRef[k]
		if !ok {
			// No, we don't have a changeset on that branch in this repo.
			// We're going to create one so the changeset reconciler picks it up,
			// creates a commit and pushes it to the branch.
			// Except, of course, if spec.Spec.Published is false, then it doesn't do anything.
			c, err = r.createChangesetForSpec(repo, spec)
			if err != nil {
				return err
			}
		} else {
			// But if we already have a changeset in the given repository with
			// the given branch, we need to update it to have the new spec
			// and possibly re-attach it to the campaign:
			if err = r.updateChangesetToNewSpec(c, spec); err != nil {
				return err
			}
		}
		// In both cases we want to attach it to the campaign
		attachedChangesets[c.ID] = true
	}

	// We went through all the new changeset specs and either created or
	// updated a changeset.
	// Their IDs are all the IDs of changesets that should be in the campaign:
	r.campaign.ChangesetIDs = []int64{}
	for changesetID := range attachedChangesets {
		r.campaign.ChangesetIDs = append(r.campaign.ChangesetIDs, changesetID)
	}

	// But it's possible that changesets are now detached, like Changeset 3 in
	// the example above.
	// This we need to detach and close.
	for _, c := range r.changesets {
		if _, ok := attachedChangesets[c.ID]; ok {
			continue
		}

		// If we don't have access to a repository, we don't detach nor close the changeset.
		_, ok := r.accessibleReposByID[c.RepoID]
		if !ok {
			continue
		}

		if c.CurrentSpecID != 0 && c.OwnedByCampaignID == r.campaign.ID {
			// If we have a current spec ID and the changeset was created by
			// _this_ campaign that means we should detach and close it.

			// But only if it was created on the code host:
			if c.Published() {
				c.Closing = true
				c.ReconcilerState = campaigns.ReconcilerStateQueued
			} else {
				// otherwise we simply delete it.
				if err = r.tx.DeleteChangeset(r.ctx, c.ID); err != nil {
					return err
				}
				continue
			}
		}

		c.RemoveCampaignID(r.campaign.ID)
		if err = r.tx.UpdateChangeset(r.ctx, c); err != nil {
			return err
		}
	}

	return r.tx.UpdateCampaign(r.ctx, r.campaign)
}

func (r *changesetRewirer) createChangesetForSpec(repo *types.Repo, spec *campaigns.ChangesetSpec) (*campaigns.Changeset, error) {
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

	return newChangeset, r.tx.CreateChangeset(r.ctx, newChangeset)
}

func (r *changesetRewirer) updateChangesetToNewSpec(c *campaigns.Changeset, spec *campaigns.ChangesetSpec) error {
	c.PreviousSpecID = c.CurrentSpecID
	c.CurrentSpecID = spec.ID

	// Ensure that the changeset is attached to the campaign
	c.CampaignIDs = append(c.CampaignIDs, r.campaign.ID)

	// Copy over diff stat from the new spec.
	diffStat := spec.DiffStat()
	c.SetDiffStat(&diffStat)

	// We need to enqueue it for the changeset reconciler, so the
	// reconciler wakes up, compares old and new spec and, if
	// necessary, updates the changesets accordingly.
	return r.updateAndReenqueue(c)
}

// loadAssociations populates the chagnesets, newChangesetSpecs and
// accessibleReposByID on changesetRewirer.
func (r *changesetRewirer) loadAssociations() (err error) {
	// Load all of the new ChangesetSpecs
	r.newChangesetSpecs, _, err = r.tx.ListChangesetSpecs(r.ctx, ListChangesetSpecsOpts{
		CampaignSpecID: r.campaign.CampaignSpecID,
	})
	if err != nil {
		return err
	}

	// Load all Changesets attached to this campaign, or owned by this campaign but detached.
	r.changesets, err = r.tx.ListChangesetsAttachedOrOwnedByCampaign(r.ctx, r.campaign.ID)
	if err != nil {
		return err
	}

	repoIDs := append(r.newChangesetSpecs.RepoIDs(), r.changesets.RepoIDs()...)
	// 🚨 SECURITY: db.Repos.GetRepoIDsSet uses the authzFilter under the hood and
	// filters out repositories that the user doesn't have access to.
	r.accessibleReposByID, err = db.Repos.GetReposSetByIDs(r.ctx, repoIDs...)
	return err
}

func (r *changesetRewirer) indexAssociations() (err error) {
	r.changesetsByRepoHeadRef = map[repoHeadRef]*campaigns.Changeset{}
	r.changesetsByRepoExternalID = map[repoExternalID]*campaigns.Changeset{}
	r.currentSpecsByChangeset = map[int64]*campaigns.ChangesetSpec{}

	for _, c := range r.changesets {
		// This is an n+1
		s, err := r.tx.GetChangesetSpecByID(r.ctx, c.CurrentSpecID)
		if err != nil {
			return err
		}
		r.currentSpecsByChangeset[c.ID] = s

		if c.ExternalID != "" {
			k := repoExternalID{repo: c.RepoID, externalID: c.ExternalID}
			r.changesetsByRepoExternalID[k] = c

			// If it has an externalID but no CurrentSpecID, it is a tracked
			// changeset, and we're done and don't need to match it by HeadRef
			if c.CurrentSpecID == 0 {
				continue
			}
		}

		k := repoHeadRef{repo: c.RepoID}
		if c.ExternalBranch != "" {
			k.headRef = git.EnsureRefPrefix(c.ExternalBranch)
			r.changesetsByRepoHeadRef[k] = c
			continue
		}

		// If we don't have an ExternalBranch, the changeset hasn't been
		// published yet (or hasn't been synced yet).
		if c.CurrentSpecID != 0 {
			// If we're here, the changeset doesn't have an external branch
			//
			// So we load the spec to get the branch where we _would_ push
			// the commit.

			k.headRef = git.EnsureRefPrefix(s.Spec.HeadRef)
			r.changesetsByRepoHeadRef[k] = c
		}
	}

	return nil
}

func (r *changesetRewirer) updateOrCreateTrackingChangeset(repo *types.Repo, externalID string) (*campaigns.Changeset, error) {
	existing, err := r.tx.GetChangeset(r.ctx, GetChangesetOpts{
		RepoID:              repo.ID,
		ExternalID:          externalID,
		ExternalServiceType: repo.ExternalRepo.ServiceType,
	})
	if err != nil && err != ErrNoResults {
		return nil, err
	}

	if existing != nil {
		// We already have a changeset with the given repoID and
		// externalID, so we can track it.
		existing.AddedToCampaign = true
		existing.CampaignIDs = append(existing.CampaignIDs, r.campaign.ID)

		// If it errored, we re-enqueue it.
		if existing.ReconcilerState == campaigns.ReconcilerStateErrored {
			return existing, r.updateAndReenqueue(existing)
		}

		return existing, r.tx.UpdateChangeset(r.ctx, existing)
	}

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

	return newChangeset, r.tx.CreateChangeset(r.ctx, newChangeset)
}

func (r *changesetRewirer) updateAndReenqueue(ch *campaigns.Changeset) error {
	ch.ResetQueued()
	return r.tx.UpdateChangeset(r.ctx, ch)
}
