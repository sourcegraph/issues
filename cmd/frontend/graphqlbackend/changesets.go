package graphqlbackend

import (
	"context"
	"sync"

	graphql "github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"
	"github.com/pkg/errors"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/db"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend/graphqlutil"
	"github.com/sourcegraph/sourcegraph/pkg/a8n"
	"github.com/sourcegraph/sourcegraph/pkg/api"
)

func (r *schemaResolver) CreateChangeSet(ctx context.Context, args *struct {
	Repository graphql.ID
	ExternalID string
}) (_ *changesetResolver, err error) {
	user, err := db.Users.GetByCurrentAuthUser(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "%v", backend.ErrNotAuthenticated)
	}

	// 🚨 SECURITY: Only site admins may create a changeset for now.
	if !user.SiteAdmin {
		return nil, backend.ErrMustBeSiteAdmin
	}

	repoID, err := unmarshalRepositoryID(args.Repository)
	if err != nil {
		return nil, err
	}

	changeset := &a8n.ChangeSet{
		RepoID:     int32(repoID),
		ExternalID: args.ExternalID,
	}

	if err = r.A8NStore.CreateChangeSet(ctx, changeset); err != nil {
		return nil, err
	}

	// TODO(tsenart): Sync change-set metadata.

	return &changesetResolver{store: r.A8NStore, ChangeSet: changeset}, nil
}

func (r *schemaResolver) ChangeSets(ctx context.Context, args *struct {
	graphqlutil.ConnectionArgs
}) (*changesetsConnectionResolver, error) {
	// 🚨 SECURITY: Only site admins may read external services (they have secrets).
	if err := backend.CheckCurrentUserIsSiteAdmin(ctx); err != nil {
		return nil, err
	}

	return &changesetsConnectionResolver{
		store: r.A8NStore,
		opts: a8n.ListChangeSetsOpts{
			Limit: int(args.ConnectionArgs.GetFirst()),
		},
	}, nil
}

type changesetsConnectionResolver struct {
	store *a8n.Store
	opts  a8n.ListChangeSetsOpts

	// cache results because they are used by multiple fields
	once       sync.Once
	changesets []*a8n.ChangeSet
	next       int64
	err        error
}

func (r *changesetsConnectionResolver) Nodes(ctx context.Context) ([]*changesetResolver, error) {
	changesets, _, err := r.compute(ctx)
	if err != nil {
		return nil, err
	}
	resolvers := make([]*changesetResolver, 0, len(changesets))
	for _, c := range changesets {
		resolvers = append(resolvers, &changesetResolver{ChangeSet: c})
	}
	return resolvers, nil
}

func (r *changesetsConnectionResolver) TotalCount(ctx context.Context) (int32, error) {
	opts := a8n.CountChangeSetsOpts{CampaignID: r.opts.CampaignID}
	count, err := r.store.CountChangeSets(ctx, opts)
	return int32(count), err
}

func (r *changesetsConnectionResolver) PageInfo(ctx context.Context) (*graphqlutil.PageInfo, error) {
	_, next, err := r.compute(ctx)
	if err != nil {
		return nil, err
	}
	return graphqlutil.HasNextPage(next != 0), nil
}

func (r *changesetsConnectionResolver) compute(ctx context.Context) ([]*a8n.ChangeSet, int64, error) {
	r.once.Do(func() {
		r.changesets, r.next, r.err = r.store.ListChangeSets(ctx, r.opts)
	})
	return r.changesets, r.next, r.err
}

type changesetResolver struct {
	store *a8n.Store
	*a8n.ChangeSet
}

const changesetIDKind = "ChangeSet"

func marshalChangeSetID(id int64) graphql.ID {
	return relay.MarshalID(changesetIDKind, id)
}

func unmarshalChangeSetID(id graphql.ID) (cid int64, err error) {
	err = relay.UnmarshalSpec(id, &cid)
	return
}

func (r *changesetResolver) ID() graphql.ID {
	return marshalChangeSetID(r.ChangeSet.ID)
}

func (r *changesetResolver) Repository(ctx context.Context) (*RepositoryResolver, error) {
	return repositoryByIDInt32(ctx, api.RepoID(r.ChangeSet.RepoID))
}

func (r *changesetResolver) Campaigns(ctx context.Context, args struct {
	graphqlutil.ConnectionArgs
}) *campaignsConnectionResolver {
	return &campaignsConnectionResolver{
		store: r.store,
		opts: a8n.ListCampaignsOpts{
			ChangeSetID: r.ChangeSet.ID,
			Limit:       int(args.ConnectionArgs.GetFirst()),
		},
	}
}

func (r *changesetResolver) CreatedAt() DateTime {
	return DateTime{Time: r.ChangeSet.CreatedAt}
}

func (r *changesetResolver) UpdatedAt() DateTime {
	return DateTime{Time: r.ChangeSet.UpdatedAt}
}

func issueURLToRepoURL(url string) string {
	// TODO: here be dragons
	// 1. Parse URL
	// 2. Determine code host
	// 3. According to which code host it is, go from issue URL to repoURL (i.e. cut off "issues/1")
	return "github.com/sourcegraph/sourcegraph"
}
