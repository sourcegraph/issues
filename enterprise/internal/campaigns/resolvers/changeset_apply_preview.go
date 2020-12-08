package resolvers

import (
	"context"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/campaigns/reconciler"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/campaigns/rewirer"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/campaigns/service"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/campaigns/store"
	"github.com/sourcegraph/sourcegraph/internal/campaigns"
)

type changesetApplyPreviewResolver struct {
	store *store.Store

	mapping           *store.RewirerMapping
	preloadedNextSync time.Time
	preloadedCampaign *campaigns.Campaign
	campaignSpecID    int64

	planOnce sync.Once
	plan     *reconciler.Plan
	planErr  error

	campaignOnce sync.Once
	campaign     *campaigns.Campaign
	campaignErr  error
}

var _ graphqlbackend.ChangesetApplyPreviewResolver = &changesetApplyPreviewResolver{}

func (r *changesetApplyPreviewResolver) repoAccessible() bool {
	// The repo is accessible when it was returned by the db when the mapping was hydrated.
	return r.mapping.Repo != nil
}

func (r *changesetApplyPreviewResolver) Operations(ctx context.Context) ([]campaigns.ReconcilerOperation, error) {
	if !r.repoAccessible() {
		// If the repo is inaccessible, no operations would be taken, since the changeset is not created/updated.
		return []campaigns.ReconcilerOperation{}, nil
	}

	plan, err := r.computePlan(ctx)
	if err != nil {
		return nil, err
	}
	ops := plan.Ops.ExecutionOrder()
	return ops, nil
}

func (r *changesetApplyPreviewResolver) Delta(ctx context.Context) (graphqlbackend.ChangesetSpecDeltaResolver, error) {
	if !r.repoAccessible() {
		// If the repo is inaccessible, no comparison is made, since the changeset is not created/updated.
		return &changesetSpecDeltaResolver{}, nil
	}

	plan, err := r.computePlan(ctx)
	if err != nil {
		return nil, err
	}
	if plan.Delta == nil {
		return &changesetSpecDeltaResolver{}, nil
	}
	return &changesetSpecDeltaResolver{delta: *plan.Delta}, nil
}

func (r *changesetApplyPreviewResolver) computePlan(ctx context.Context) (*reconciler.Plan, error) {
	r.planOnce.Do(func() {
		campaign, err := r.computeCampaign(ctx)
		if err != nil {
			r.planErr = err
			return
		}

		// Then, dry-run the rewirer to simulate how the changeset would look like _after_ an apply operation.
		rewirer := rewirer.New(store.RewirerMappings{r.mapping}, campaign.ID)
		changesets, err := rewirer.Rewire()
		if err != nil {
			r.planErr = err
			return
		}
		if len(changesets) != 1 {
			r.planErr = errors.New("rewirer did not return changeset")
			return
		}
		changeset := changesets[0]

		// Detached changesets would still appear here, but since they'll never match one of the new specs, they don't actually appear here.
		// Once we have a way to have changeset specs for detached changesets, this would be the place to do a "will be detached" check.
		// TBD: How we represent that in the API.

		// The rewirer takes previous and current spec into account to determine actions to take,
		// so we need to find out which specs we need to pass to the planner.

		// This means that we currently won't show "attach to tracking changeset" and "detach changeset" in this preview API. Close and import non-existing work, though.
		var previousSpec, currentSpec *campaigns.ChangesetSpec
		if changeset.PreviousSpecID != 0 {
			previousSpec, err = r.store.GetChangesetSpecByID(ctx, changeset.PreviousSpecID)
			if err != nil {
				r.planErr = err
				return
			}
		}
		if changeset.CurrentSpecID != 0 {
			// If the current spec was not unset by the rewirer, it will be this resolvers spec.
			currentSpec = r.mapping.ChangesetSpec
		}
		r.plan, r.planErr = reconciler.DeterminePlan(previousSpec, currentSpec, changeset)
	})
	return r.plan, r.planErr
}

func (r *changesetApplyPreviewResolver) computeCampaign(ctx context.Context) (*campaigns.Campaign, error) {
	r.campaignOnce.Do(func() {
		if r.preloadedCampaign != nil {
			r.campaign = r.preloadedCampaign
			return
		}
		svc := service.New(r.store)
		campaignSpec, err := r.store.GetCampaignSpec(ctx, store.GetCampaignSpecOpts{ID: r.campaignSpecID})
		if err != nil {
			r.planErr = err
			return
		}
		// Dry-run reconcile the campaign with the new campaign spec.
		r.campaign, _, r.campaignErr = svc.ReconcileCampaign(ctx, campaignSpec)
	})
	return r.campaign, r.campaignErr
}

func (r *changesetApplyPreviewResolver) ChangesetSpec(ctx context.Context) (graphqlbackend.ChangesetSpecResolver, error) {
	if r.mapping.ChangesetSpec == nil {
		return nil, nil
	}
	return NewChangesetSpecResolverWithRepo(r.store, r.mapping.Repo, r.mapping.ChangesetSpec), nil
}

func (r *changesetApplyPreviewResolver) Changeset(ctx context.Context) (graphqlbackend.ChangesetResolver, error) {
	if r.mapping.Changeset == nil {
		return nil, nil
	}
	return NewChangesetResolverWithNextSync(r.store, r.mapping.Changeset, r.mapping.Repo, r.preloadedNextSync), nil
}

type changesetSpecDeltaResolver struct {
	delta reconciler.ChangesetSpecDelta
}

var _ graphqlbackend.ChangesetSpecDeltaResolver = &changesetSpecDeltaResolver{}

func (c *changesetSpecDeltaResolver) TitleChanged() bool {
	return c.delta.TitleChanged
}
func (c *changesetSpecDeltaResolver) BodyChanged() bool {
	return c.delta.BodyChanged
}
func (c *changesetSpecDeltaResolver) Undraft() bool {
	return c.delta.Undraft
}
func (c *changesetSpecDeltaResolver) BaseRefChanged() bool {
	return c.delta.BaseRefChanged
}
func (c *changesetSpecDeltaResolver) DiffChanged() bool {
	return c.delta.DiffChanged
}
func (c *changesetSpecDeltaResolver) CommitMessageChanged() bool {
	return c.delta.CommitMessageChanged
}
func (c *changesetSpecDeltaResolver) AuthorNameChanged() bool {
	return c.delta.AuthorNameChanged
}
func (c *changesetSpecDeltaResolver) AuthorEmailChanged() bool {
	return c.delta.AuthorEmailChanged
}
