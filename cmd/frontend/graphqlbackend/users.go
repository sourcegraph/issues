package graphqlbackend

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend/graphqlutil"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/internal/usagestats"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/internal/db"
)

func (r *schemaResolver) Users(args *struct {
	graphqlutil.ConnectionArgs
	Query        *string
	Tag          *string
	ActivePeriod *string
}) *userConnectionResolver {
	var opt db.UsersListOptions
	if args.Query != nil {
		opt.Query = *args.Query
	}
	if args.Tag != nil {
		opt.Tag = *args.Tag
	}
	args.ConnectionArgs.Set(&opt.LimitOffset)
	return &userConnectionResolver{opt: opt, activePeriod: args.ActivePeriod}
}

type UserConnectionResolver interface {
	Nodes(ctx context.Context) ([]*UserResolver, error)
	TotalCount(ctx context.Context) (int32, error)
	PageInfo(ctx context.Context) (*graphqlutil.PageInfo, error)
}

var _ UserConnectionResolver = &userConnectionResolver{}

type userConnectionResolver struct {
	opt          db.UsersListOptions
	activePeriod *string

	// cache results because they are used by multiple fields
	once       sync.Once
	users      []*types.User
	totalCount int
	err        error
}

// compute caches results from the more expensive user list creation that occurs when activePeriod
// is set to a specific length of time.
//
// Because usage statistics data isn't stored in PostgreSQL (but rather in Redis), adding this
// parameter requires accessing a second data store.
func (r *userConnectionResolver) compute(ctx context.Context) ([]*types.User, int, error) {
	if r.activePeriod == nil {
		return nil, 0, errors.New("activePeriod must not be nil")
	}
	r.once.Do(func() {
		var err error
		switch *r.activePeriod {
		case "TODAY":
			r.opt.UserIDs, err = usagestats.ListRegisteredUsersToday(ctx)
		case "THIS_WEEK":
			r.opt.UserIDs, err = usagestats.ListRegisteredUsersThisWeek(ctx)
		case "THIS_MONTH":
			r.opt.UserIDs, err = usagestats.ListRegisteredUsersThisMonth(ctx)
		default:
			err = fmt.Errorf("unknown user active period %s", *r.activePeriod)
		}
		if err != nil {
			r.err = err
			return
		}

		r.users, err = db.Users.List(ctx, &r.opt)
		if err != nil {
			r.err = err
			return
		}
		r.totalCount, r.err = db.Users.Count(ctx, &r.opt)
	})
	return r.users, r.totalCount, r.err
}

func (r *userConnectionResolver) Nodes(ctx context.Context) ([]*UserResolver, error) {
	var users []*types.User
	var err error
	if r.useCache() {
		users, _, err = r.compute(ctx)
	} else {
		users, err = db.Users.List(ctx, &r.opt)
	}
	if err != nil {
		return nil, err
	}

	var l []*UserResolver
	for _, user := range users {
		l = append(l, &UserResolver{
			user: user,
		})
	}
	return l, nil
}

func (r *userConnectionResolver) TotalCount(ctx context.Context) (int32, error) {
	var count int
	var err error
	if r.useCache() {
		_, count, err = r.compute(ctx)
	} else {
		count, err = db.Users.Count(ctx, &r.opt)
	}
	return int32(count), err
}

func (r *userConnectionResolver) PageInfo(ctx context.Context) (*graphqlutil.PageInfo, error) {
	count, err := r.TotalCount(ctx)
	if err != nil {
		return nil, err
	}
	return graphqlutil.HasNextPage(r.opt.LimitOffset != nil && int(count) > r.opt.Limit), nil
}

func (r *userConnectionResolver) useCache() bool {
	return r.activePeriod != nil && *r.activePeriod != "ALL_TIME"
}

// staticUserConnectionResolver implements the GraphQL type UserConnection based on an underlying
// list of users that is computed statically.
type staticUserConnectionResolver struct {
	users []*types.User
}

func (r *staticUserConnectionResolver) Nodes() []*UserResolver {
	resolvers := make([]*UserResolver, len(r.users))
	for i, user := range r.users {
		resolvers[i] = &UserResolver{user: user}
	}
	return resolvers
}

func (r *staticUserConnectionResolver) TotalCount() int32 { return int32(len(r.users)) }

func (r *staticUserConnectionResolver) PageInfo() *graphqlutil.PageInfo {
	return graphqlutil.HasNextPage(false) // not paginated
}
