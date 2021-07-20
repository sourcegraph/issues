package resolvers

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/RoaringBitmap/roaring"
	"github.com/cockroachdb/errors"
	"github.com/google/go-cmp/cmp"
	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/gqltesting"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/globals"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend"
	edb "github.com/sourcegraph/sourcegraph/enterprise/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/actor"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/authz"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/database/basestore"
	"github.com/sourcegraph/sourcegraph/internal/database/dbtest"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/repoupdater/protocol"
	"github.com/sourcegraph/sourcegraph/internal/timeutil"
	"github.com/sourcegraph/sourcegraph/internal/types"
	"github.com/sourcegraph/sourcegraph/schema"
)

var now = timeutil.Now().UnixNano()

func clock() time.Time {
	return time.Unix(0, atomic.LoadInt64(&now))
}

var (
	parseSchemaOnce sync.Once
	parseSchemaErr  error
	parsedSchema    *graphql.Schema
)

func mustParseGraphQLSchema(t *testing.T, db dbutil.DB) *graphql.Schema {
	t.Helper()

	parseSchemaOnce.Do(func() {
		parsedSchema, parseSchemaErr = graphqlbackend.NewSchema(db, nil, nil, nil, NewResolver(db, clock), nil, nil, nil)
	})
	if parseSchemaErr != nil {
		t.Fatal(parseSchemaErr)
	}

	return parsedSchema
}

func TestResolver_SetRepositoryPermissionsForUsers(t *testing.T) {
	t.Run("authenticated as non-admin", func(t *testing.T) {
		database.Mocks.Users.GetByCurrentAuthUser = func(context.Context) (*types.User, error) {
			return &types.User{}, nil
		}
		defer func() {
			database.Mocks.Users.GetByCurrentAuthUser = nil
		}()

		ctx := actor.WithActor(context.Background(), &actor.Actor{UID: 1})
		result, err := (&Resolver{store: &edb.PermsStore{Store: basestore.NewWithDB(nil, sql.TxOptions{})}}).SetRepositoryPermissionsForUsers(ctx, &graphqlbackend.RepoPermsArgs{})
		if want := backend.ErrMustBeSiteAdmin; err != want {
			t.Errorf("err: want %q but got %v", want, err)
		}
		if result != nil {
			t.Errorf("result: want nil but got %v", result)
		}
	})

	tests := []struct {
		name               string
		config             *schema.PermissionsUserMapping
		mockVerifiedEmails []*database.UserEmail
		mockUsers          []*types.User
		gqlTests           []*gqltesting.Test
		expUserIDs         []uint32
		expAccounts        *extsvc.Accounts
	}{
		{
			name: "set permissions via email",
			config: &schema.PermissionsUserMapping{
				BindID: "email",
			},
			mockVerifiedEmails: []*database.UserEmail{
				{
					UserID: 1,
					Email:  "alice@example.com",
				},
			},
			gqlTests: []*gqltesting.Test{
				{
					Schema: mustParseGraphQLSchema(t, nil),
					Query: `
				mutation {
					setRepositoryPermissionsForUsers(
						repository: "UmVwb3NpdG9yeTox",
						userPermissions: [
							{ bindID: "alice@example.com"},
							{ bindID: "bob"}
						]) {
						alwaysNil
					}
				}
			`,
					ExpectedResult: `
				{
					"setRepositoryPermissionsForUsers": {
						"alwaysNil": null
    				}
				}
			`,
				},
			},
			expUserIDs: []uint32{1},
			expAccounts: &extsvc.Accounts{
				ServiceType: authz.SourcegraphServiceType,
				ServiceID:   authz.SourcegraphServiceID,
				AccountIDs:  []string{"bob"},
			},
		},
		{
			name: "set permissions via username",
			config: &schema.PermissionsUserMapping{
				BindID: "username",
			},
			mockUsers: []*types.User{
				{
					ID:       1,
					Username: "alice",
				},
			},
			gqlTests: []*gqltesting.Test{
				{
					Schema: mustParseGraphQLSchema(t, nil),
					Query: `
				mutation {
					setRepositoryPermissionsForUsers(
						repository: "UmVwb3NpdG9yeTox",
						userPermissions: [
							{ bindID: "alice"},
							{ bindID: "bob"}
						]) {
						alwaysNil
					}
				}
			`,
					ExpectedResult: `
				{
					"setRepositoryPermissionsForUsers": {
						"alwaysNil": null
					}
				}
			`,
				},
			},
			expUserIDs: []uint32{1},
			expAccounts: &extsvc.Accounts{
				ServiceType: authz.SourcegraphServiceType,
				ServiceID:   authz.SourcegraphServiceID,
				AccountIDs:  []string{"bob"},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			globals.SetPermissionsUserMapping(test.config)

			database.Mocks.Users.GetByCurrentAuthUser = func(context.Context) (*types.User, error) {
				return &types.User{SiteAdmin: true}, nil
			}
			database.Mocks.Users.GetByUsernames = func(context.Context, ...string) ([]*types.User, error) {
				return test.mockUsers, nil
			}
			database.Mocks.UserEmails.GetVerifiedEmails = func(context.Context, ...string) ([]*database.UserEmail, error) {
				return test.mockVerifiedEmails, nil
			}
			database.Mocks.Repos.Get = func(_ context.Context, id api.RepoID) (*types.Repo, error) {
				return &types.Repo{ID: id}, nil
			}
			edb.Mocks.Perms.Transact = func(_ context.Context) (*edb.PermsStore, error) {
				return &edb.PermsStore{}, nil
			}
			edb.Mocks.Perms.SetRepoPermissions = func(_ context.Context, p *authz.RepoPermissions) error {
				ids := p.UserIDs.ToArray()
				if diff := cmp.Diff(test.expUserIDs, ids); diff != "" {
					return errors.Errorf("p.UserIDs: %v", diff)
				}
				return nil
			}
			edb.Mocks.Perms.SetRepoPendingPermissions = func(_ context.Context, accounts *extsvc.Accounts, _ *authz.RepoPermissions) error {
				if diff := cmp.Diff(test.expAccounts, accounts); diff != "" {
					return errors.Errorf("accounts: %v", diff)
				}
				return nil
			}
			defer func() {
				database.Mocks.UserEmails = database.MockUserEmails{}
				database.Mocks.Users = database.MockUsers{}
				database.Mocks.Repos = database.MockRepos{}
				edb.Mocks.Perms = edb.MockPerms{}
			}()

			gqltesting.RunTests(t, test.gqlTests)
		})
	}
}

func TestResolver_ScheduleRepositoryPermissionsSync(t *testing.T) {
	db := dbtest.NewDB(t, "")

	t.Run("authenticated as non-admin", func(t *testing.T) {
		database.Mocks.Users.GetByCurrentAuthUser = func(context.Context) (*types.User, error) {
			return &types.User{}, nil
		}
		t.Cleanup(func() {
			database.Mocks.Users = database.MockUsers{}
		})

		ctx := actor.WithActor(context.Background(), &actor.Actor{UID: 1})
		result, err := (&Resolver{store: edb.Perms(db, timeutil.Now)}).ScheduleRepositoryPermissionsSync(ctx, &graphqlbackend.RepositoryIDArgs{})
		if want := backend.ErrMustBeSiteAdmin; err != want {
			t.Errorf("err: want %q but got %v", want, err)
		}
		if result != nil {
			t.Errorf("result: want nil but got %v", result)
		}
	})

	database.Mocks.Users.GetByCurrentAuthUser = func(context.Context) (*types.User, error) {
		return &types.User{SiteAdmin: true}, nil
	}
	t.Cleanup(func() {
		database.Mocks.Users = database.MockUsers{}
	})

	r := &Resolver{
		store: edb.Perms(db, timeutil.Now),
		repoupdaterClient: &fakeRepoupdaterClient{
			mockSchedulePermsSync: func(ctx context.Context, args protocol.PermsSyncRequest) error {
				if len(args.RepoIDs) != 1 {
					return errors.Errorf("RepoIDs: want 1 id but got %d", len(args.RepoIDs))
				}
				return nil
			},
		},
	}
	_, err := r.ScheduleRepositoryPermissionsSync(context.Background(), &graphqlbackend.RepositoryIDArgs{
		Repository: graphqlbackend.MarshalRepositoryID(1),
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestResolver_ScheduleUserPermissionsSync(t *testing.T) {
	db := dbtest.NewDB(t, "")

	t.Run("authenticated as non-admin", func(t *testing.T) {
		database.Mocks.Users.GetByCurrentAuthUser = func(context.Context) (*types.User, error) {
			return &types.User{}, nil
		}
		t.Cleanup(func() {
			database.Mocks.Users = database.MockUsers{}
		})

		ctx := actor.WithActor(context.Background(), &actor.Actor{UID: 1})
		result, err := (&Resolver{store: edb.Perms(db, timeutil.Now)}).ScheduleUserPermissionsSync(ctx, &graphqlbackend.UserIDArgs{})
		if want := backend.ErrMustBeSiteAdmin; err != want {
			t.Errorf("err: want %q but got %v", want, err)
		}
		if result != nil {
			t.Errorf("result: want nil but got %v", result)
		}
	})

	database.Mocks.Users.GetByCurrentAuthUser = func(context.Context) (*types.User, error) {
		return &types.User{SiteAdmin: true}, nil
	}
	t.Cleanup(func() {
		database.Mocks.Users = database.MockUsers{}
	})

	r := &Resolver{
		store: edb.Perms(db, timeutil.Now),
		repoupdaterClient: &fakeRepoupdaterClient{
			mockSchedulePermsSync: func(ctx context.Context, args protocol.PermsSyncRequest) error {
				if len(args.UserIDs) != 1 {
					return errors.Errorf("UserIDs: want 1 id but got %d", len(args.UserIDs))
				}
				return nil
			},
		},
	}
	_, err := r.ScheduleUserPermissionsSync(context.Background(), &graphqlbackend.UserIDArgs{
		User: graphqlbackend.MarshalUserID(1),
	})
	if err != nil {
		t.Fatal(err)
	}
}

type fakeRepoupdaterClient struct {
	mockSchedulePermsSync func(ctx context.Context, args protocol.PermsSyncRequest) error
}

func (c *fakeRepoupdaterClient) SchedulePermsSync(ctx context.Context, args protocol.PermsSyncRequest) error {
	return c.mockSchedulePermsSync(ctx, args)
}

func TestResolver_AuthorizedUserRepositories(t *testing.T) {
	db := dbtest.NewDB(t, "")

	t.Run("authenticated as non-admin", func(t *testing.T) {
		database.Mocks.Users.GetByCurrentAuthUser = func(context.Context) (*types.User, error) {
			return &types.User{}, nil
		}
		defer func() {
			database.Mocks.Users.GetByCurrentAuthUser = nil
		}()

		ctx := actor.WithActor(context.Background(), &actor.Actor{UID: 1})
		result, err := (&Resolver{store: edb.Perms(db, timeutil.Now)}).AuthorizedUserRepositories(ctx, &graphqlbackend.AuthorizedRepoArgs{})
		if want := backend.ErrMustBeSiteAdmin; err != want {
			t.Errorf("err: want %q but got %v", want, err)
		}
		if result != nil {
			t.Errorf("result: want nil but got %v", result)
		}
	})

	database.Mocks.Users.GetByCurrentAuthUser = func(context.Context) (*types.User, error) {
		return &types.User{SiteAdmin: true}, nil
	}
	database.Mocks.Users.GetByVerifiedEmail = func(_ context.Context, email string) (*types.User, error) {
		if email == "alice@example.com" {
			return &types.User{ID: 1}, nil
		}
		return nil, database.MockUserNotFoundErr
	}
	database.Mocks.Users.GetByUsername = func(_ context.Context, username string) (*types.User, error) {
		if username == "alice" {
			return &types.User{ID: 1}, nil
		}
		return nil, database.MockUserNotFoundErr
	}
	database.Mocks.Repos.GetByIDs = func(_ context.Context, ids ...api.RepoID) ([]*types.Repo, error) {
		repos := make([]*types.Repo, len(ids))
		for i, id := range ids {
			repos[i] = &types.Repo{ID: id}
		}
		return repos, nil
	}
	edb.Mocks.Perms.LoadUserPermissions = func(_ context.Context, p *authz.UserPermissions) error {
		p.IDs = roaring.NewBitmap()
		p.IDs.Add(1)
		return nil
	}
	edb.Mocks.Perms.LoadUserPendingPermissions = func(_ context.Context, p *authz.UserPendingPermissions) error {
		p.IDs = roaring.NewBitmap()
		p.IDs.Add(2)
		return nil
	}
	defer func() {
		database.Mocks.Users = database.MockUsers{}
		edb.Mocks.Perms = edb.MockPerms{}
	}()

	tests := []struct {
		name     string
		gqlTests []*gqltesting.Test
	}{
		{
			name: "check authorized repos via email",
			gqlTests: []*gqltesting.Test{
				{
					Schema: mustParseGraphQLSchema(t, nil),
					Query: `
				{
					authorizedUserRepositories(
						first: 10,
						email: "alice@example.com") {
						nodes {
							id
						}
					}
				}
			`,
					ExpectedResult: `
				{
					"authorizedUserRepositories": {
						"nodes": [
							{"id":"UmVwb3NpdG9yeTox"}
						]
    				}
				}
			`,
				},
			},
		},
		{
			name: "check authorized repos via username",
			gqlTests: []*gqltesting.Test{
				{
					Schema: mustParseGraphQLSchema(t, nil),
					Query: `
				{
					authorizedUserRepositories(
						first: 10,
						username: "alice") {
						nodes {
							id
						}
					}
				}
			`,
					ExpectedResult: `
				{
					"authorizedUserRepositories": {
						"nodes": [
							{"id":"UmVwb3NpdG9yeTox"}
						]
    				}
				}
			`,
				},
			},
		},
		{
			name: "check pending authorized repos via email",
			gqlTests: []*gqltesting.Test{
				{
					Schema: mustParseGraphQLSchema(t, nil),
					Query: `
				{
					authorizedUserRepositories(
						first: 10,
						email: "bob@example.com") {
						nodes {
							id
						}
					}
				}
			`,
					ExpectedResult: `
				{
					"authorizedUserRepositories": {
						"nodes": [
							{"id":"UmVwb3NpdG9yeToy"}
						]
    				}
				}
			`,
				},
			},
		},
		{
			name: "check pending authorized repos via username",
			gqlTests: []*gqltesting.Test{
				{
					Schema: mustParseGraphQLSchema(t, nil),
					Query: `
				{
					authorizedUserRepositories(
						first: 10,
						username: "bob") {
						nodes {
							id
						}
					}
				}
			`,
					ExpectedResult: `
				{
					"authorizedUserRepositories": {
						"nodes": [
							{"id":"UmVwb3NpdG9yeToy"}
						]
    				}
				}
			`,
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gqltesting.RunTests(t, test.gqlTests)
		})
	}
}

func TestResolver_UsersWithPendingPermissions(t *testing.T) {
	db := dbtest.NewDB(t, "")

	t.Run("authenticated as non-admin", func(t *testing.T) {
		database.Mocks.Users.GetByCurrentAuthUser = func(context.Context) (*types.User, error) {
			return &types.User{}, nil
		}
		defer func() {
			database.Mocks.Users.GetByCurrentAuthUser = nil
		}()

		ctx := actor.WithActor(context.Background(), &actor.Actor{UID: 1})
		result, err := (&Resolver{store: edb.Perms(db, timeutil.Now)}).UsersWithPendingPermissions(ctx)
		if want := backend.ErrMustBeSiteAdmin; err != want {
			t.Errorf("err: want %q but got %v", want, err)
		}
		if result != nil {
			t.Errorf("result: want nil but got %v", result)
		}
	})

	database.Mocks.Users.GetByCurrentAuthUser = func(context.Context) (*types.User, error) {
		return &types.User{SiteAdmin: true}, nil
	}
	edb.Mocks.Perms.ListPendingUsers = func(context.Context) ([]string, error) {
		return []string{"alice", "bob"}, nil
	}
	defer func() {
		database.Mocks.Users = database.MockUsers{}
		edb.Mocks.Perms = edb.MockPerms{}
	}()

	tests := []struct {
		name     string
		gqlTests []*gqltesting.Test
	}{
		{
			name: "list pending users with their bind IDs",
			gqlTests: []*gqltesting.Test{
				{
					Schema: mustParseGraphQLSchema(t, nil),
					Query: `
				{
					usersWithPendingPermissions
				}
			`,
					ExpectedResult: `
				{
					"usersWithPendingPermissions": [
						"alice",
						"bob"
					]
				}
			`,
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gqltesting.RunTests(t, test.gqlTests)
		})
	}
}

func TestResolver_AuthorizedUsers(t *testing.T) {
	db := dbtest.NewDB(t, "")

	t.Run("authenticated as non-admin", func(t *testing.T) {
		database.Mocks.Users.GetByCurrentAuthUser = func(context.Context) (*types.User, error) {
			return &types.User{}, nil
		}
		defer func() {
			database.Mocks.Users.GetByCurrentAuthUser = nil
		}()

		ctx := actor.WithActor(context.Background(), &actor.Actor{UID: 1})
		result, err := (&Resolver{store: edb.Perms(db, timeutil.Now)}).AuthorizedUsers(ctx, &graphqlbackend.RepoAuthorizedUserArgs{})
		if want := backend.ErrMustBeSiteAdmin; err != want {
			t.Errorf("err: want %q but got %v", want, err)
		}
		if result != nil {
			t.Errorf("result: want nil but got %v", result)
		}
	})

	database.Mocks.Users.GetByCurrentAuthUser = func(context.Context) (*types.User, error) {
		return &types.User{SiteAdmin: true}, nil
	}
	database.Mocks.Users.List = func(_ context.Context, opt *database.UsersListOptions) ([]*types.User, error) {
		users := make([]*types.User, len(opt.UserIDs))
		for i, id := range opt.UserIDs {
			users[i] = &types.User{ID: id}
		}
		return users, nil
	}
	database.Mocks.Repos.GetByName = func(_ context.Context, repo api.RepoName) (*types.Repo, error) {
		return &types.Repo{ID: 1, Name: repo}, nil
	}
	database.Mocks.Repos.Get = func(_ context.Context, id api.RepoID) (*types.Repo, error) {
		return &types.Repo{ID: id}, nil
	}
	edb.Mocks.Perms.LoadRepoPermissions = func(_ context.Context, p *authz.RepoPermissions) error {
		p.UserIDs = roaring.NewBitmap()
		p.UserIDs.Add(1)
		return nil
	}
	defer func() {
		database.Mocks.Users = database.MockUsers{}
		database.Mocks.Repos = database.MockRepos{}
		edb.Mocks.Perms = edb.MockPerms{}
	}()

	tests := []struct {
		name     string
		gqlTests []*gqltesting.Test
	}{
		{
			name: "get authorized users",
			gqlTests: []*gqltesting.Test{
				{
					Schema: mustParseGraphQLSchema(t, nil),
					Query: `
				{
					repository(name: "github.com/owner/repo") {
						authorizedUsers(first: 10) {
							nodes {
								id
							}
						}
					}
				}
			`,
					ExpectedResult: `
				{
					"repository": {
						"authorizedUsers": {
							"nodes":[
								{"id":"VXNlcjox"}
							]
						}
    				}
				}
			`,
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gqltesting.RunTests(t, test.gqlTests)
		})
	}
}

func TestResolver_RepositoryPermissionsInfo(t *testing.T) {
	db := dbtest.NewDB(t, "")

	t.Run("authenticated as non-admin", func(t *testing.T) {
		database.Mocks.Users.GetByCurrentAuthUser = func(context.Context) (*types.User, error) {
			return &types.User{}, nil
		}
		t.Cleanup(func() {
			database.Mocks.Users.GetByCurrentAuthUser = nil
		})

		ctx := actor.WithActor(context.Background(), &actor.Actor{UID: 1})
		result, err := (&Resolver{store: edb.Perms(db, timeutil.Now)}).RepositoryPermissionsInfo(ctx, graphqlbackend.MarshalRepositoryID(1))
		if want := backend.ErrMustBeSiteAdmin; err != want {
			t.Errorf("err: want %q but got %v", want, err)
		}
		if result != nil {
			t.Errorf("result: want nil but got %v", result)
		}
	})

	database.Mocks.Users.GetByCurrentAuthUser = func(context.Context) (*types.User, error) {
		return &types.User{SiteAdmin: true}, nil
	}
	database.Mocks.Repos.GetByName = func(_ context.Context, repo api.RepoName) (*types.Repo, error) {
		return &types.Repo{ID: 1, Name: repo}, nil
	}
	database.Mocks.Repos.Get = func(_ context.Context, id api.RepoID) (*types.Repo, error) {
		return &types.Repo{ID: id}, nil
	}
	edb.Mocks.Perms.LoadRepoPermissions = func(_ context.Context, p *authz.RepoPermissions) error {
		p.UpdatedAt = clock()
		p.SyncedAt = clock()
		return nil
	}
	defer func() {
		database.Mocks.Users = database.MockUsers{}
		database.Mocks.Repos = database.MockRepos{}
		edb.Mocks.Perms = edb.MockPerms{}
	}()
	tests := []struct {
		name     string
		gqlTests []*gqltesting.Test
	}{
		{
			name: "get permissions information",
			gqlTests: []*gqltesting.Test{
				{
					Schema: mustParseGraphQLSchema(t, nil),
					Query: `
				{
					repository(name: "github.com/owner/repo") {
						permissionsInfo {
							permissions
							syncedAt
							updatedAt
						}
					}
				}
			`,
					ExpectedResult: fmt.Sprintf(`
				{
					"repository": {
						"permissionsInfo": {
							"permissions": ["READ"],
							"syncedAt": "%[1]s",
							"updatedAt": "%[1]s"
						}
    				}
				}
			`, clock().Format(time.RFC3339)),
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gqltesting.RunTests(t, test.gqlTests)
		})
	}
}

func TestResolver_UserPermissionsInfo(t *testing.T) {
	db := dbtest.NewDB(t, "")

	t.Run("authenticated as non-admin", func(t *testing.T) {
		database.Mocks.Users.GetByCurrentAuthUser = func(context.Context) (*types.User, error) {
			return &types.User{}, nil
		}
		t.Cleanup(func() {
			database.Mocks.Users.GetByCurrentAuthUser = nil
		})

		ctx := actor.WithActor(context.Background(), &actor.Actor{UID: 1})
		result, err := (&Resolver{store: edb.Perms(db, timeutil.Now)}).UserPermissionsInfo(ctx, graphqlbackend.MarshalRepositoryID(1))
		if want := backend.ErrMustBeSiteAdmin; err != want {
			t.Errorf("err: want %q but got %v", want, err)
		}
		if result != nil {
			t.Errorf("result: want nil but got %v", result)
		}
	})

	database.Mocks.Users.GetByCurrentAuthUser = func(context.Context) (*types.User, error) {
		return &types.User{SiteAdmin: true}, nil
	}
	database.Mocks.Users.GetByID = func(ctx context.Context, id int32) (*types.User, error) {
		return &types.User{ID: id}, nil
	}
	edb.Mocks.Perms.LoadUserPermissions = func(_ context.Context, p *authz.UserPermissions) error {
		p.UpdatedAt = clock()
		p.SyncedAt = clock()
		return nil
	}
	defer func() {
		database.Mocks.Users = database.MockUsers{}
		edb.Mocks.Perms = edb.MockPerms{}
	}()
	tests := []struct {
		name     string
		gqlTests []*gqltesting.Test
	}{
		{
			name: "get permissions information",
			gqlTests: []*gqltesting.Test{
				{
					Schema: mustParseGraphQLSchema(t, nil),
					Query: `
				{
					currentUser {
						permissionsInfo {
							permissions
							syncedAt
							updatedAt
						}
					}
				}
			`,
					ExpectedResult: fmt.Sprintf(`
				{
					"currentUser": {
						"permissionsInfo": {
							"permissions": ["READ"],
							"syncedAt": "%[1]s",
							"updatedAt": "%[1]s"
						}
    				}
				}
			`, clock().Format(time.RFC3339)),
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gqltesting.RunTests(t, test.gqlTests)
		})
	}
}
