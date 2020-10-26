package graphqlbackend

import (
	"context"
	"testing"

	"github.com/graph-gophers/graphql-go/gqltesting"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/internal/db"
)

func TestCreateUser(t *testing.T) {
	resetMocks()
	db.Mocks.Users.GetByCurrentAuthUser = func(context.Context) (*types.User, error) {
		return &types.User{SiteAdmin: true}, nil
	}
	db.Mocks.Users.Create = func(context.Context, db.NewUser) (*types.User, error) {
		return &types.User{ID: 1, Username: "alice"}, nil
	}

	calledGrantPendingPermissions := false
	db.Mocks.Authz.GrantPendingPermissions = func(context.Context, *db.GrantPendingPermissionsArgs) error {
		calledGrantPendingPermissions = true
		return nil
	}

	gqltesting.RunTests(t, []*gqltesting.Test{
		{
			Schema: mustParseGraphQLSchema(t),
			Query: `
				mutation {
					createUser(username: "alice") {
						user {
							id
						}
					}
				}
			`,
			ExpectedResult: `
				{
					"createUser": {
						"user": {
							"id": "VXNlcjox"
						}
					}
				}
			`,
		},
	})
	if !calledGrantPendingPermissions {
		t.Fatal("!calledGrantPendingPermissions")
	}
}
