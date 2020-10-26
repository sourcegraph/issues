package graphqlbackend

import (
	"context"
	"testing"

	"github.com/graph-gophers/graphql-go/gqltesting"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/internal/db"
)

func TestSetUserEmailVerified(t *testing.T) {
	resetMocks()
	db.Mocks.Users.GetByCurrentAuthUser = func(context.Context) (*types.User, error) {
		return &types.User{SiteAdmin: true}, nil
	}
	db.Mocks.UserEmails.SetVerified = func(context.Context, int32, string, bool) error {
		return nil
	}

	tests := []struct {
		name                                string
		gqlTests                            []*gqltesting.Test
		expectCalledGrantPendingPermissions bool
	}{
		{
			name: "set an email to be verified",
			gqlTests: []*gqltesting.Test{
				{
					Schema: mustParseGraphQLSchema(t),
					Query: `
				mutation {
					setUserEmailVerified(user: "VXNlcjox", email: "alice@example.com", verified: true) {
						alwaysNil
					}
				}
			`,
					ExpectedResult: `
				{
					"setUserEmailVerified": {
						"alwaysNil": null
    				}
				}
			`,
				},
			},
			expectCalledGrantPendingPermissions: true,
		},
		{
			name: "set an email to be unverified",
			gqlTests: []*gqltesting.Test{
				{
					Schema: mustParseGraphQLSchema(t),
					Query: `
				mutation {
					setUserEmailVerified(user: "VXNlcjox", email: "alice@example.com", verified: false) {
						alwaysNil
					}
				}
			`,
					ExpectedResult: `
				{
					"setUserEmailVerified": {
						"alwaysNil": null
    				}
				}
			`,
				},
			},
			expectCalledGrantPendingPermissions: false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			calledGrantPendingPermissions := false
			db.Mocks.Authz.GrantPendingPermissions = func(context.Context, *db.GrantPendingPermissionsArgs) error {
				calledGrantPendingPermissions = true
				return nil
			}

			gqltesting.RunTests(t, test.gqlTests)

			if test.expectCalledGrantPendingPermissions != calledGrantPendingPermissions {
				t.Fatalf("calledGrantPendingPermissions: want %v but got %v", test.expectCalledGrantPendingPermissions, calledGrantPendingPermissions)
			}
		})
	}
}
