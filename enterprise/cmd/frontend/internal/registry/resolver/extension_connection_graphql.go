package resolver

import (
	"context"
	"sort"
	"strings"
	"sync"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend/graphqlutil"
	"github.com/sourcegraph/sourcegraph/enterprise/cmd/frontend/internal/registry/service"
	"github.com/sourcegraph/sourcegraph/enterprise/cmd/frontend/internal/registry/store"
	"github.com/sourcegraph/sourcegraph/enterprise/cmd/frontend/internal/registry/types"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
)

// makePrioritizeExtensionIDsSet returns a set whose values are the elements of
// args.PrioritizeExtensionIDs.
func makePrioritizeExtensionIDsSet(args graphqlbackend.RegistryExtensionConnectionArgs) map[string]struct{} {
	if args.PrioritizeExtensionIDs == nil {
		return nil
	}
	set := make(map[string]struct{}, len(*args.PrioritizeExtensionIDs))
	for _, id := range *args.PrioritizeExtensionIDs {
		set[id] = struct{}{}
	}
	return set
}

func (r *extensionRegistryResolver) Extensions(ctx context.Context, args *graphqlbackend.RegistryExtensionConnectionArgs) (graphqlbackend.RegistryExtensionConnection, error) {
	return &registryExtensionConnectionResolver{args: *args, db: r.db}, nil
}

// registryExtensionConnectionResolver resolves a list of registry extensions.
type registryExtensionConnectionResolver struct {
	args graphqlbackend.RegistryExtensionConnectionArgs

	// cache results because they are used by multiple fields
	once               sync.Once
	registryExtensions []graphqlbackend.RegistryExtension
	err                error
	db                 dbutil.DB
}

var (
	// ListLocalRegistryExtensions lists and returns local registry extensions according to the args. If
	// there is no local extension registry, it is not implemented.
	ListLocalRegistryExtensions func(context.Context, dbutil.DB, graphqlbackend.RegistryExtensionConnectionArgs) ([]graphqlbackend.RegistryExtension, error)

	// CountLocalRegistryExtensions returns the count of local registry extensions according to the
	// args. Pagination-related args are ignored. If there is no local extension registry, it is not
	// implemented.
	CountLocalRegistryExtensions func(context.Context, dbutil.DB, graphqlbackend.RegistryExtensionConnectionArgs) (int, error)
)

func (r *registryExtensionConnectionResolver) compute(ctx context.Context) ([]graphqlbackend.RegistryExtension, error) {
	r.once.Do(func() {
		args2 := r.args
		if args2.First != nil {
			tmp := *args2.First
			tmp++ // so we can detect if there is a next page
			args2.First = &tmp
		}

		var query string
		if args2.Query != nil {
			query = *args2.Query
		}

		// Query local registry extensions.
		var local []graphqlbackend.RegistryExtension
		if r.args.Local && ListLocalRegistryExtensions != nil {
			local, r.err = ListLocalRegistryExtensions(ctx, r.db, args2)
			if r.err != nil {
				return
			}
		}

		// Query remote registry extensions, if filters would match any.
		var remote []*types.Extension
		if args2.Publisher == nil && r.args.Remote {
			xs, err := listRemoteRegistryExtensions(ctx, query)
			if err != nil {
				// Continue execution even if r.err != nil so that partial (local) results are returned
				// even when the remote registry is inaccessible.
				r.err = err
			}
			remote = append(remote, xs...)
		}

		r.registryExtensions = make([]graphqlbackend.RegistryExtension, len(local)+len(remote))
		copy(r.registryExtensions, local)
		for i, x := range remote {
			r.registryExtensions[len(local)+i] = &registryExtensionRemoteResolver{v: x}
		}

		// Sort WIP extensions last. (The local extensions list is already sorted in that way, but
		// the remote extensions list isn't, so therefore the combined list isn't.)
		sort.SliceStable(r.registryExtensions, func(i, j int) bool {
			return !r.registryExtensions[i].IsWorkInProgress() && r.registryExtensions[j].IsWorkInProgress()
		})

		if r.args.PrioritizeExtensionIDs != nil && len(*r.args.PrioritizeExtensionIDs) > 0 {
			// Sort prioritized extension IDs first.
			set := makePrioritizeExtensionIDsSet(r.args)
			sort.SliceStable(r.registryExtensions, func(i, j int) bool {
				_, pi := set[r.registryExtensions[i].ExtensionID()]
				_, pj := set[r.registryExtensions[j].ExtensionID()]
				return pi && !pj
			})
		}
	})
	return r.registryExtensions, r.err
}

func (r *registryExtensionConnectionResolver) Nodes(ctx context.Context) ([]graphqlbackend.RegistryExtension, error) {
	// See (*registryExtensionConnectionResolver).Error for why we ignore the error.
	xs, _ := r.compute(ctx)
	if r.args.First != nil && len(xs) > int(*r.args.First) {
		xs = xs[:int(*r.args.First)]
	}
	return xs, nil
}

func (r *registryExtensionConnectionResolver) TotalCount(ctx context.Context) (int32, error) {
	var total int

	if r.args.Local && CountLocalRegistryExtensions != nil {
		dbCount, err := CountLocalRegistryExtensions(ctx, r.db, r.args)
		if err != nil {
			return 0, err
		}
		total += dbCount
	}

	// Count remote extensions. Performing an actual fetch is necessary.
	//
	// See (*registryExtensionConnectionResolver).Error for why we ignore the error.
	xs, _ := r.compute(ctx)
	for _, x := range xs {
		if _, isRemote := x.(*registryExtensionRemoteResolver); isRemote {
			total++
		}
	}

	return int32(total), nil
}

func (r *registryExtensionConnectionResolver) PageInfo(ctx context.Context) (*graphqlutil.PageInfo, error) {
	// See (*registryExtensionConnectionResolver).Error for why we ignore the error.
	registryExtensions, _ := r.compute(ctx)
	return graphqlutil.HasNextPage(r.args.First != nil && len(registryExtensions) > int(*r.args.First)), nil
}

func (r *registryExtensionConnectionResolver) URL(ctx context.Context) (*string, error) {
	if r.args.Publisher == nil || RegistryPublisherByID == nil {
		return nil, nil
	}

	publisher, err := RegistryPublisherByID(ctx, *r.args.Publisher)
	if err != nil {
		return nil, err
	}
	return publisher.RegistryExtensionConnectionURL()
}

func (r *registryExtensionConnectionResolver) Error(ctx context.Context) *string {
	// See the GraphQL API schema documentation for this field for an explanation of why we return
	// errors in this way.
	//
	// TODO(sqs): When https://github.com/graph-gophers/graphql-go/pull/219 or similar is merged, we
	// can make the other fields return data *and* an error, instead of using this separate error
	// field.
	_, err := r.compute(ctx)
	if err == nil {
		return nil
	}
	errStr := err.Error()
	return &errStr
}

func listLocalRegistryExtensions(ctx context.Context, db dbutil.DB, args graphqlbackend.RegistryExtensionConnectionArgs) ([]graphqlbackend.RegistryExtension, error) {
	if args.PrioritizeExtensionIDs != nil {
		ids := filterStripLocalExtensionIDs(*args.PrioritizeExtensionIDs)
		args.PrioritizeExtensionIDs = &ids
	}
	opt, err := toDBExtensionsListOptions(args)
	if err != nil {
		return nil, err
	}

	vs, err := store.NewDBExtensions(db).List(ctx, opt)
	if err != nil {
		return nil, err
	}
	if err := prefixLocalExtensionID(vs...); err != nil {
		return nil, err
	}

	svc := service.New(db)
	releasesByExtensionID, err := svc.GetLatestForBatch(ctx, vs)
	if err != nil {
		return nil, err
	}
	var ys []graphqlbackend.RegistryExtension
	for _, v := range vs {
		ys = append(ys, &extensionDBResolver{db: db, v: v, r: releasesByExtensionID[v.ID]})
	}
	return ys, nil
}

func countLocalRegistryExtensions(ctx context.Context, db dbutil.DB, args graphqlbackend.RegistryExtensionConnectionArgs) (int, error) {
	opt, err := toDBExtensionsListOptions(args)
	if err != nil {
		return 0, err
	}
	return store.NewDBExtensions(db).Count(ctx, opt)
}

func toDBExtensionsListOptions(args graphqlbackend.RegistryExtensionConnectionArgs) (store.DBExtensionsListOptions, error) {
	var opt store.DBExtensionsListOptions
	args.ConnectionArgs.Set(&opt.LimitOffset)
	if args.Publisher != nil {
		p, err := unmarshalRegistryPublisherID(*args.Publisher)
		if err != nil {
			return opt, err
		}
		opt.Publisher.UserID = p.userID
		opt.Publisher.OrgID = p.orgID
	}
	if args.Query != nil {
		opt.Query, opt.Category, opt.Tag = types.ParseExtensionQuery(*args.Query)
	}
	if args.PrioritizeExtensionIDs != nil {
		opt.PrioritizeExtensionIDs = *args.PrioritizeExtensionIDs
	}
	return opt, nil
}

// filterStripLocalExtensionIDs filters to local extension IDs and strips the
// host prefix.
func filterStripLocalExtensionIDs(extensionIDs []string) []string {
	prefix := GetLocalRegistryExtensionIDPrefix()
	local := []string{}
	for _, id := range extensionIDs {
		parts := strings.SplitN(id, "/", 3)
		if prefix != nil && len(parts) == 3 && parts[0] == *prefix {
			local = append(local, parts[1]+"/"+parts[2])
		} else if (prefix == nil || *prefix == "") && len(parts) == 2 {
			local = append(local, id)
		}
	}
	return local
}
