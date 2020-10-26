package graphqlbackend

import (
	"context"

	"github.com/graph-gophers/graphql-go"
	"github.com/pkg/errors"
)

type AuthzResolver interface {
	// Mutations
	SetRepositoryPermissionsForUsers(ctx context.Context, args *RepoPermsArgs) (*EmptyResponse, error)
	ScheduleRepositoryPermissionsSync(ctx context.Context, args *RepositoryIDArgs) (*EmptyResponse, error)
	ScheduleUserPermissionsSync(ctx context.Context, args *UserIDArgs) (*EmptyResponse, error)

	// Queries
	AuthorizedUserRepositories(ctx context.Context, args *AuthorizedRepoArgs) (RepositoryConnectionResolver, error)
	UsersWithPendingPermissions(ctx context.Context) ([]string, error)
	AuthorizedUsers(ctx context.Context, args *RepoAuthorizedUserArgs) (UserConnectionResolver, error)

	// Helpers
	RepositoryPermissionsInfo(ctx context.Context, repoID graphql.ID) (PermissionsInfoResolver, error)
	UserPermissionsInfo(ctx context.Context, userID graphql.ID) (PermissionsInfoResolver, error)
}

var authzInEnterprise = errors.New("authorization mutations and queries are only available in enterprise")

type defaultAuthzResolver struct{}

var DefaultAuthzResolver AuthzResolver = defaultAuthzResolver{}

func (defaultAuthzResolver) SetRepositoryPermissionsForUsers(ctx context.Context, args *RepoPermsArgs) (*EmptyResponse, error) {
	return nil, authzInEnterprise
}

func (defaultAuthzResolver) ScheduleRepositoryPermissionsSync(ctx context.Context, args *RepositoryIDArgs) (*EmptyResponse, error) {
	return nil, authzInEnterprise
}

func (defaultAuthzResolver) ScheduleUserPermissionsSync(ctx context.Context, args *UserIDArgs) (*EmptyResponse, error) {
	return nil, authzInEnterprise
}

func (defaultAuthzResolver) AuthorizedUserRepositories(ctx context.Context, args *AuthorizedRepoArgs) (RepositoryConnectionResolver, error) {
	return nil, authzInEnterprise
}

func (defaultAuthzResolver) UsersWithPendingPermissions(ctx context.Context) ([]string, error) {
	return nil, authzInEnterprise
}

func (defaultAuthzResolver) AuthorizedUsers(ctx context.Context, args *RepoAuthorizedUserArgs) (UserConnectionResolver, error) {
	return nil, authzInEnterprise
}

func (defaultAuthzResolver) RepositoryPermissionsInfo(ctx context.Context, repoID graphql.ID) (PermissionsInfoResolver, error) {
	return nil, authzInEnterprise
}

func (defaultAuthzResolver) UserPermissionsInfo(ctx context.Context, userID graphql.ID) (PermissionsInfoResolver, error) {
	// NOTE: Both OSS and enterprise web app use the same GraphQL query to get user information
	// (including "permissionsInfo" field). Since the web app won't show "Permissions" tab in
	// OSS version anyway, it is OK to return empty information about user permissions (as if
	// no permissions available in enterprise version) to not fail the GraphQL query entirely.
	return nil, nil
}

type RepositoryIDArgs struct {
	Repository graphql.ID
}

type UserIDArgs struct {
	User graphql.ID
}

type RepoPermsArgs struct {
	Repository      graphql.ID
	UserPermissions []struct {
		BindID     string
		Permission string
	}
}

type AuthorizedRepoArgs struct {
	Username *string
	Email    *string
	Perm     string
	First    int32
	After    *string
}

type PermissionsInfoResolver interface {
	Permissions() []string
	SyncedAt() *DateTime
	UpdatedAt() DateTime
}
