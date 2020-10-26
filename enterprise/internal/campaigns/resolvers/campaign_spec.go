package resolvers

import (
	"context"
	"strconv"
	"sync"

	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"
	"github.com/pkg/errors"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend"
	ee "github.com/sourcegraph/sourcegraph/enterprise/internal/campaigns"
	"github.com/sourcegraph/sourcegraph/internal/campaigns"
	"github.com/sourcegraph/sourcegraph/internal/errcode"
	"github.com/sourcegraph/sourcegraph/internal/httpcli"
)

func marshalCampaignSpecRandID(id string) graphql.ID {
	return relay.MarshalID("CampaignSpec", id)
}

func unmarshalCampaignSpecID(id graphql.ID) (campaignSpecRandID string, err error) {
	err = relay.UnmarshalSpec(id, &campaignSpecRandID)
	return
}

var _ graphqlbackend.CampaignSpecResolver = &campaignSpecResolver{}

type campaignSpecResolver struct {
	store       *ee.Store
	httpFactory *httpcli.Factory

	campaignSpec *campaigns.CampaignSpec

	// We cache the namespace on the resolver, since it's accessed more than once.
	namespaceOnce sync.Once
	namespace     *graphqlbackend.NamespaceResolver
	namespaceErr  error
}

func (r *campaignSpecResolver) ID() graphql.ID {
	// 🚨 SECURITY: This needs to be the RandID! We can't expose the
	// sequential, guessable ID.
	return marshalCampaignSpecRandID(r.campaignSpec.RandID)
}

func (r *campaignSpecResolver) OriginalInput() (string, error) {
	return r.campaignSpec.RawSpec, nil
}

func (r *campaignSpecResolver) ParsedInput() (graphqlbackend.JSONValue, error) {
	return graphqlbackend.JSONValue{Value: r.campaignSpec.Spec}, nil
}

func (r *campaignSpecResolver) ChangesetSpecs(ctx context.Context, args *graphqlbackend.ChangesetSpecsConnectionArgs) (graphqlbackend.ChangesetSpecConnectionResolver, error) {
	opts := ee.ListChangesetSpecsOpts{CampaignSpecID: r.campaignSpec.ID}
	if err := validateFirstParamDefaults(args.First); err != nil {
		return nil, err
	}
	opts.Limit = int(args.First)
	if args.After != nil {
		id, err := strconv.Atoi(*args.After)
		if err != nil {
			return nil, err
		}
		opts.Cursor = int64(id)
	}

	return &changesetSpecConnectionResolver{
		store:       r.store,
		httpFactory: r.httpFactory,
		opts:        opts,
	}, nil
}

func (r *campaignSpecResolver) Description() graphqlbackend.CampaignDescriptionResolver {
	return &campaignDescriptionResolver{
		name:        r.campaignSpec.Spec.Name,
		description: r.campaignSpec.Spec.Description,
	}
}

func (r *campaignSpecResolver) Creator(ctx context.Context) (*graphqlbackend.UserResolver, error) {
	user, err := graphqlbackend.UserByIDInt32(ctx, r.campaignSpec.UserID)
	if errcode.IsNotFound(err) {
		return nil, nil
	}
	return user, err
}

func (r *campaignSpecResolver) Namespace(ctx context.Context) (*graphqlbackend.NamespaceResolver, error) {
	return r.computeNamespace(ctx)
}

func (r *campaignSpecResolver) computeNamespace(ctx context.Context) (*graphqlbackend.NamespaceResolver, error) {
	r.namespaceOnce.Do(func() {
		var (
			err error
			n   = &graphqlbackend.NamespaceResolver{}
		)

		if r.campaignSpec.NamespaceUserID != 0 {
			n.Namespace, err = graphqlbackend.UserByIDInt32(ctx, r.campaignSpec.NamespaceUserID)
		} else {
			n.Namespace, err = graphqlbackend.OrgByIDInt32(ctx, r.campaignSpec.NamespaceOrgID)
		}

		if errcode.IsNotFound(err) {
			r.namespace = nil
			r.namespaceErr = errors.New("namespace of campaign spec has been deleted")
			return
		}

		r.namespace = n
		r.namespaceErr = err
	})
	return r.namespace, r.namespaceErr
}

func (r *campaignSpecResolver) ApplyURL(ctx context.Context) (string, error) {
	n, err := r.computeNamespace(ctx)
	if err != nil {
		return "", err
	}
	return campaignsApplyURL(n, r), nil
}

func (r *campaignSpecResolver) CreatedAt() graphqlbackend.DateTime {
	return graphqlbackend.DateTime{Time: r.campaignSpec.CreatedAt}
}

func (r *campaignSpecResolver) ExpiresAt() *graphqlbackend.DateTime {
	return &graphqlbackend.DateTime{Time: r.campaignSpec.ExpiresAt()}
}

func (r *campaignSpecResolver) ViewerCanAdminister(ctx context.Context) (bool, error) {
	return checkSiteAdminOrSameUser(ctx, r.campaignSpec.UserID)
}

type campaignDescriptionResolver struct {
	name, description string
}

func (r *campaignDescriptionResolver) Name() string {
	return r.name
}

func (r *campaignDescriptionResolver) Description() string {
	return r.description
}

func (r *campaignSpecResolver) DiffStat(ctx context.Context) (*graphqlbackend.DiffStat, error) {
	specsConnection := &changesetSpecConnectionResolver{
		store:       r.store,
		httpFactory: r.httpFactory,
		opts: ee.ListChangesetSpecsOpts{
			CampaignSpecID: r.campaignSpec.ID,
		},
	}

	specs, err := specsConnection.Nodes(ctx)
	if err != nil {
		return nil, err
	}

	totalStat := &graphqlbackend.DiffStat{}
	for _, spec := range specs {
		// If we can't convert it, that means it's hidden from the user and we
		// can simply skip it.
		if _, ok := spec.ToVisibleChangesetSpec(); !ok {
			continue
		}

		resolver, ok := spec.(*changesetSpecResolver)
		if !ok {
			// This should never happen.
			continue
		}

		stat := resolver.changesetSpec.DiffStat()
		totalStat.AddStat(stat)
	}

	return totalStat, nil
}

func (r *campaignSpecResolver) AppliesToCampaign(ctx context.Context) (graphqlbackend.CampaignResolver, error) {
	svc := ee.NewService(r.store, r.httpFactory)
	campaign, err := svc.GetCampaignMatchingCampaignSpec(ctx, r.store, r.campaignSpec)
	if err != nil {
		return nil, err
	}
	if campaign == nil {
		return nil, nil
	}

	return &campaignResolver{
		store:       r.store,
		httpFactory: r.httpFactory,
		Campaign:    campaign,
	}, nil
}
