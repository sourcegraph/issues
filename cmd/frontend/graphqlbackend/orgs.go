package graphqlbackend

import (
	"context"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend/graphqlutil"
	"github.com/sourcegraph/sourcegraph/internal/db"
)

func (r *schemaResolver) Organizations(args *struct {
	graphqlutil.ConnectionArgs
	Query *string
}) *orgConnectionResolver {
	var opt db.OrgsListOptions
	if args.Query != nil {
		opt.Query = *args.Query
	}
	args.ConnectionArgs.Set(&opt.LimitOffset)
	return &orgConnectionResolver{opt: opt}
}

type orgConnectionResolver struct {
	opt db.OrgsListOptions
}

func (r *orgConnectionResolver) Nodes(ctx context.Context) ([]*OrgResolver, error) {
	// 🚨 SECURITY: Only site admins can list orgs.
	if err := backend.CheckCurrentUserIsSiteAdmin(ctx); err != nil {
		return nil, err
	}

	orgs, err := db.Orgs.List(ctx, &r.opt)
	if err != nil {
		return nil, err
	}

	var l []*OrgResolver
	for _, org := range orgs {
		l = append(l, &OrgResolver{
			org: org,
		})
	}
	return l, nil
}

func (r *orgConnectionResolver) TotalCount(ctx context.Context) (int32, error) {
	// 🚨 SECURITY: Only site admins can count orgs.
	if err := backend.CheckCurrentUserIsSiteAdmin(ctx); err != nil {
		return 0, err
	}

	count, err := db.Orgs.Count(ctx, r.opt)
	return int32(count), err
}

type orgConnectionStaticResolver struct {
	nodes []*OrgResolver
}

func (r *orgConnectionStaticResolver) Nodes() []*OrgResolver { return r.nodes }
func (r *orgConnectionStaticResolver) TotalCount() int32     { return int32(len(r.nodes)) }
func (r *orgConnectionStaticResolver) PageInfo() *graphqlutil.PageInfo {
	return graphqlutil.HasNextPage(false)
}
