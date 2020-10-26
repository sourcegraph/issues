package graphqlbackend

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/gqltesting"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/internal/actor"
	"github.com/sourcegraph/sourcegraph/internal/authz"
	"github.com/sourcegraph/sourcegraph/internal/db"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
)

func TestDeleteUser(t *testing.T) {
	t.Run("authenticated as non-admin", func(t *testing.T) {
		resetMocks()
		db.Mocks.Users.GetByCurrentAuthUser = func(context.Context) (*types.User, error) {
			return &types.User{}, nil
		}

		ctx := actor.WithActor(context.Background(), &actor.Actor{UID: 1})
		result, err := (&schemaResolver{}).DeleteUser(ctx, &struct {
			User graphql.ID
			Hard *bool
		}{
			User: MarshalUserID(1),
		})
		if want := backend.ErrMustBeSiteAdmin; err != want {
			t.Errorf("err: want %q but got %v", want, err)
		}
		if result != nil {
			t.Errorf("result: want nil but got %v", result)
		}
	})

	t.Run("delete current user", func(t *testing.T) {
		resetMocks()
		db.Mocks.Users.GetByCurrentAuthUser = func(context.Context) (*types.User, error) {
			return &types.User{ID: 1, SiteAdmin: true}, nil
		}

		ctx := actor.WithActor(context.Background(), &actor.Actor{UID: 1})
		_, err := (&schemaResolver{}).DeleteUser(ctx, &struct {
			User graphql.ID
			Hard *bool
		}{
			User: MarshalUserID(1),
		})
		want := "unable to delete current user"
		if err == nil || err.Error() != want {
			t.Fatalf("err: want %q but got %v", want, err)
		}
	})

	// Mocking all database interactions here, but they are all thoroughly tested in the lower layer in "db" package.
	resetMocks()
	db.Mocks.Users.GetByCurrentAuthUser = func(context.Context) (*types.User, error) {
		return &types.User{SiteAdmin: true}, nil
	}
	db.Mocks.Users.GetByID = func(_ context.Context, id int32) (*types.User, error) {
		return &types.User{ID: id, Username: "alice"}, nil
	}
	db.Mocks.Users.Delete = func(context.Context, int32) error {
		return nil
	}
	db.Mocks.Users.HardDelete = func(context.Context, int32) error {
		return nil
	}
	db.Mocks.UserEmails.ListByUser = func(context.Context, db.UserEmailsListOptions) ([]*db.UserEmail, error) {
		return []*db.UserEmail{
			{Email: "alice@example.com"},
		}, nil
	}
	db.Mocks.ExternalAccounts.List = func(db.ExternalAccountsListOptions) ([]*extsvc.Account, error) {
		return []*extsvc.Account{
			{
				AccountSpec: extsvc.AccountSpec{
					ServiceType: extsvc.TypeGitLab,
					ServiceID:   "https://gitlab.com/",
					AccountID:   "alice_gitlab",
				},
			},
		}, nil
	}
	db.Mocks.Authz.RevokeUserPermissions = func(_ context.Context, args *db.RevokeUserPermissionsArgs) error {
		if args.UserID != 6 {
			return fmt.Errorf("args.UserID: want 6 but got %v", args.UserID)
		}

		expAccounts := []*extsvc.Accounts{
			{
				ServiceType: extsvc.TypeGitLab,
				ServiceID:   "https://gitlab.com/",
				AccountIDs:  []string{"alice_gitlab"},
			},
			{
				ServiceType: authz.SourcegraphServiceType,
				ServiceID:   authz.SourcegraphServiceID,
				AccountIDs:  []string{"alice@example.com", "alice"},
			},
		}
		if diff := cmp.Diff(expAccounts, args.Accounts); diff != "" {
			t.Fatalf("args.Accounts: %v", diff)
		}
		return nil
	}

	tests := []struct {
		name     string
		gqlTests []*gqltesting.Test
	}{
		{
			name: "soft delete a user",
			gqlTests: []*gqltesting.Test{
				{
					Schema: mustParseGraphQLSchema(t),
					Query: `
				mutation {
					deleteUser(user: "VXNlcjo2") {
						alwaysNil
					}
				}
			`,
					ExpectedResult: `
				{
					"deleteUser": {
						"alwaysNil": null
					}
				}
			`,
				},
			},
		},
		{
			name: "hard delete a user",
			gqlTests: []*gqltesting.Test{
				{
					Schema: mustParseGraphQLSchema(t),
					Query: `
				mutation {
					deleteUser(user: "VXNlcjo2", hard: true) {
						alwaysNil
					}
				}
			`,
					ExpectedResult: `
				{
					"deleteUser": {
						"alwaysNil": null
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
