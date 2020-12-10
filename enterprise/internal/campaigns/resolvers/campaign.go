package resolvers

import (
	"context"
	"sync"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"
	"github.com/pkg/errors"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend"
	ee "github.com/sourcegraph/sourcegraph/enterprise/internal/campaigns"
	"github.com/sourcegraph/sourcegraph/internal/campaigns"
	"github.com/sourcegraph/sourcegraph/internal/errcode"
	"github.com/sourcegraph/sourcegraph/internal/httpcli"
)

var _ graphqlbackend.CampaignResolver = &campaignResolver{}

type campaignResolver struct {
	store       *ee.Store
	httpFactory *httpcli.Factory
	*campaigns.Campaign

	// Cache the namespace on the resolver, since it's accessed more than once.
	namespaceOnce sync.Once
	namespace     graphqlbackend.NamespaceResolver
	namespaceErr  error
}

const campaignIDKind = "Campaign"

func marshalCampaignID(id int64) graphql.ID {
	return relay.MarshalID(campaignIDKind, id)
}

func unmarshalCampaignID(id graphql.ID) (campaignID int64, err error) {
	err = relay.UnmarshalSpec(id, &campaignID)
	return
}

func (r *campaignResolver) ID() graphql.ID {
	return marshalCampaignID(r.Campaign.ID)
}

func (r *campaignResolver) Name() string {
	return r.Campaign.Name
}

func (r *campaignResolver) Description() *string {
	if r.Campaign.Description == "" {
		return nil
	}
	return &r.Campaign.Description
}

func (r *campaignResolver) InitialApplier(ctx context.Context) (*graphqlbackend.UserResolver, error) {
	user, err := graphqlbackend.UserByIDInt32(ctx, r.Campaign.InitialApplierID)
	if errcode.IsNotFound(err) {
		return nil, nil
	}
	return user, err
}

func (r *campaignResolver) LastApplier(ctx context.Context) (*graphqlbackend.UserResolver, error) {
	user, err := graphqlbackend.UserByIDInt32(ctx, r.Campaign.LastApplierID)
	if errcode.IsNotFound(err) {
		return nil, nil
	}
	return user, err
}

func (r *campaignResolver) LastAppliedAt() graphqlbackend.DateTime {
	return graphqlbackend.DateTime{Time: r.Campaign.LastAppliedAt}
}

func (r *campaignResolver) SpecCreator(ctx context.Context) (*graphqlbackend.UserResolver, error) {
	spec, err := r.store.GetCampaignSpec(ctx, ee.GetCampaignSpecOpts{
		ID: r.Campaign.CampaignSpecID,
	})
	if err != nil {
		return nil, err
	}
	user, err := graphqlbackend.UserByIDInt32(ctx, spec.UserID)
	if errcode.IsNotFound(err) {
		return nil, nil
	}
	return user, err
}

func (r *campaignResolver) ViewerCanAdminister(ctx context.Context) (bool, error) {
	return checkSiteAdminOrSameUser(ctx, r.Campaign.InitialApplierID)
}

func (r *campaignResolver) URL(ctx context.Context) (string, error) {
	n, err := r.Namespace(ctx)
	if err != nil {
		return "", err
	}
	return campaignURL(n, r), nil
}

func (r *campaignResolver) Namespace(ctx context.Context) (graphqlbackend.NamespaceResolver, error) {
	return r.computeNamespace(ctx)
}

func (r *campaignResolver) computeNamespace(ctx context.Context) (graphqlbackend.NamespaceResolver, error) {
	r.namespaceOnce.Do(func() {
		if r.Campaign.NamespaceUserID != 0 {
			r.namespace.Namespace, r.namespaceErr = graphqlbackend.UserByIDInt32(
				ctx,
				r.Campaign.NamespaceUserID,
			)
		} else {
			r.namespace.Namespace, r.namespaceErr = graphqlbackend.OrgByIDInt32(
				ctx,
				r.Campaign.NamespaceOrgID,
			)
		}
		if errcode.IsNotFound(r.namespaceErr) {
			r.namespace.Namespace = nil
			r.namespaceErr = errors.New("namespace of campaign has been deleted")
		}
	})

	return r.namespace, r.namespaceErr
}

func (r *campaignResolver) CreatedAt() graphqlbackend.DateTime {
	return graphqlbackend.DateTime{Time: r.Campaign.CreatedAt}
}

func (r *campaignResolver) UpdatedAt() graphqlbackend.DateTime {
	return graphqlbackend.DateTime{Time: r.Campaign.UpdatedAt}
}

func (r *campaignResolver) ClosedAt() *graphqlbackend.DateTime {
	if !r.Campaign.Closed() {
		return nil
	}
	return &graphqlbackend.DateTime{Time: r.Campaign.ClosedAt}
}

func (r *campaignResolver) ChangesetsStats(ctx context.Context) (graphqlbackend.ChangesetsStatsResolver, error) {
	stats, err := r.store.GetChangesetsStats(ctx, ee.GetChangesetsStatsOpts{
		CampaignID: r.Campaign.ID,
	})
	if err != nil {
		return nil, err
	}
	return &changesetsStatsResolver{stats: stats}, nil
}

func (r *campaignResolver) Changesets(
	ctx context.Context,
	args *graphqlbackend.ListChangesetsArgs,
) (graphqlbackend.ChangesetsConnectionResolver, error) {
	opts, safe, err := listChangesetOptsFromArgs(args, r.Campaign.ID)
	if err != nil {
		return nil, err
	}
	opts.CampaignID = r.Campaign.ID
	return &changesetsConnectionResolver{
		store:    r.store,
		opts:     opts,
		optsSafe: safe,
	}, nil
}

func (r *campaignResolver) ChangesetCountsOverTime(
	ctx context.Context,
	args *graphqlbackend.ChangesetCountsArgs,
) ([]graphqlbackend.ChangesetCountsResolver, error) {
	resolvers := []graphqlbackend.ChangesetCountsResolver{}

	publishedState := campaigns.ChangesetPublicationStatePublished
	opts := ee.ListChangesetsOpts{
		CampaignID:       r.Campaign.ID,
		PublicationState: &publishedState,
		// Only load fully-synced changesets, so that the data we use for computing the changeset counts is complete.
		OnlySynced: true,
	}
	cs, _, err := r.store.ListChangesets(ctx, opts)
	if err != nil {
		return resolvers, err
	}

	now := r.store.Clock()()

	weekAgo := now.Add(-7 * 24 * time.Hour)
	start := r.Campaign.CreatedAt.UTC()
	if start.After(weekAgo) {
		start = weekAgo
	}
	if args.From != nil {
		start = args.From.Time.UTC()
	}

	end := now.UTC()
	if args.To != nil && args.To.Time.Before(end) {
		end = args.To.Time.UTC()
	}

	eventsOpts := ee.ListChangesetEventsOpts{ChangesetIDs: cs.IDs(), Kinds: ee.RequiredEventTypesForHistory}
	es, _, err := r.store.ListChangesetEvents(ctx, eventsOpts)
	if err != nil {
		return resolvers, err
	}

	// sort.Sort(cs)
	if args.From == nil {
		for _, c := range cs {
			t := c.ExternalCreatedAt()
			if !t.IsZero() {
				start = t
				break
			}
		}
	}

	counts, err := ee.CalcCounts(start, end, cs, es...)
	if err != nil {
		return resolvers, err
	}

	for _, c := range counts {
		resolvers = append(resolvers, &changesetCountsResolver{counts: c})
	}

	return resolvers, nil
}

func (r *campaignResolver) DiffStat(ctx context.Context) (*graphqlbackend.DiffStat, error) {
	changesetsConnection := &changesetsConnectionResolver{
		store: r.store,
		opts: ee.ListChangesetsOpts{
			CampaignID: r.Campaign.ID,
		},
		optsSafe: true,
	}

	changesets, err := changesetsConnection.Nodes(ctx)
	if err != nil {
		return nil, err
	}

	totalStat := &graphqlbackend.DiffStat{}
	for _, cs := range changesets {
		// Not being able to convert is OK; it just means there's a hidden
		// changeset that we can't use the stats from.
		if external, ok := cs.ToExternalChangeset(); ok && external != nil {
			stat, err := external.DiffStat(ctx)
			if err != nil {
				return nil, err
			}
			if stat != nil {
				totalStat.AddDiffStat(stat)
			}
		}
	}

	return totalStat, nil
}

func (r *campaignResolver) CurrentSpec(ctx context.Context) (graphqlbackend.CampaignSpecResolver, error) {
	campaignSpec, err := r.store.GetCampaignSpec(ctx, ee.GetCampaignSpecOpts{ID: r.Campaign.CampaignSpecID})
	if err != nil {
		// This spec should always exist, so fail hard on not found errors as well.
		return nil, err
	}

	return &campaignSpecResolver{store: r.store, httpFactory: r.httpFactory, campaignSpec: campaignSpec}, nil
}
