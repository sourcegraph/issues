package resolvers

import (
	"context"
	"sync"

	"github.com/RoaringBitmap/roaring"
	"github.com/graph-gophers/graphql-go"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend/graphqlutil"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/db"
)

var _ graphqlbackend.UserConnectionResolver = &userConnectionResolver{}

// userConnectionResolver resolves a list of user from the roaring bitmap with pagination.
type userConnectionResolver struct {
	ids *roaring.Bitmap

	first int32
	after *string

	// cache results because they are used by multiple fields
	once     sync.Once
	users    []*types.User
	pageInfo *graphqlutil.PageInfo
	err      error
}

// 🚨 SECURITY: It is the caller's responsibility to ensure the current authenticated user
// is the site admin because this method computes data from all available information in
// the database.
func (r *userConnectionResolver) compute(ctx context.Context) ([]*types.User, *graphqlutil.PageInfo, error) {
	r.once.Do(func() {
		// Create the bitmap iterator and advance to the next value of r.after.
		var afterID api.RepoID
		if r.after != nil {
			afterID, r.err = graphqlbackend.UnmarshalRepositoryID(graphql.ID(*r.after))
			if r.err != nil {
				return
			}
		}
		iter := r.ids.Iterator()
		iter.AdvanceIfNeeded(uint32(afterID) + 1) // Plus 1 since r.after should be excluded.

		userIDs := make([]int32, 0, r.first)
		for iter.HasNext() {
			userIDs = append(userIDs, int32(iter.Next()))
			if len(userIDs) >= int(r.first) {
				break
			}
		}

		r.users, r.err = db.Users.List(ctx, &db.UsersListOptions{
			UserIDs: userIDs,
		})
		if r.err != nil {
			return
		}

		if iter.HasNext() {
			endCursor := string(graphqlbackend.MarshalUserID(userIDs[len(userIDs)-1]))
			r.pageInfo = graphqlutil.NextPageCursor(endCursor)
		} else {
			r.pageInfo = graphqlutil.HasNextPage(false)
		}
	})
	return r.users, r.pageInfo, r.err
}

func (r *userConnectionResolver) Nodes(ctx context.Context) ([]*graphqlbackend.UserResolver, error) {
	// 🚨 SECURITY: Only site admins may access this method.
	if err := backend.CheckCurrentUserIsSiteAdmin(ctx); err != nil {
		return nil, err
	}

	users, _, err := r.compute(ctx)
	if err != nil {
		return nil, err
	}
	resolvers := make([]*graphqlbackend.UserResolver, len(users))
	for i := range users {
		resolvers[i] = graphqlbackend.NewUserResolver(users[i])
	}
	return resolvers, nil
}

func (r *userConnectionResolver) TotalCount(ctx context.Context) (int32, error) {
	// 🚨 SECURITY: Only site admins may access this method.
	if err := backend.CheckCurrentUserIsSiteAdmin(ctx); err != nil {
		return -1, err
	}

	return int32(r.ids.GetCardinality()), nil
}

func (r *userConnectionResolver) PageInfo(ctx context.Context) (*graphqlutil.PageInfo, error) {
	// 🚨 SECURITY: Only site admins may access this method.
	if err := backend.CheckCurrentUserIsSiteAdmin(ctx); err != nil {
		return nil, err
	}

	_, pageInfo, err := r.compute(ctx)
	return pageInfo, err
}
