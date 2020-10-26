package graphqlbackend

import (
	"context"
	"testing"

	"github.com/graph-gophers/graphql-go/gqltesting"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/internal/db"
	"github.com/sourcegraph/sourcegraph/internal/repoupdater"
	"github.com/sourcegraph/sourcegraph/internal/repoupdater/protocol"
)

func TestStatusMessages(t *testing.T) {
	graphqlQuery := `
		query StatusMessages {
			statusMessages {
				__typename

				... on CloningProgress {
					message
				}

				... on SyncError {
					message
				}

				... on ExternalServiceSyncError {
					message
					externalService {
						id
						displayName
					}
				}
			}
		}
	`

	resetMocks()
	t.Run("unauthenticated", func(t *testing.T) {
		result, err := (&schemaResolver{}).StatusMessages(context.Background())
		if want := backend.ErrNotAuthenticated; err != want {
			t.Errorf("got err %v, want %v", err, want)
		}
		if result != nil {
			t.Errorf("got result %v, want nil", result)
		}
	})

	t.Run("authenticated as non-site-admin", func(t *testing.T) {
		db.Mocks.Users.GetByCurrentAuthUser = func(ctx context.Context) (*types.User, error) {
			return &types.User{ID: 1, SiteAdmin: false}, nil
		}
		defer func() { db.Mocks.Users.GetByCurrentAuthUser = nil }()

		result, err := (&schemaResolver{}).StatusMessages(context.Background())
		if want := backend.ErrMustBeSiteAdmin; err != want {
			t.Errorf("got err %v, want %v", err, want)
		}
		if result != nil {
			t.Errorf("got result %v, want nil", result)
		}
	})

	t.Run("no messages", func(t *testing.T) {
		db.Mocks.Users.GetByCurrentAuthUser = func(ctx context.Context) (*types.User, error) {
			return &types.User{ID: 1, SiteAdmin: true}, nil
		}
		defer func() { db.Mocks.Users.GetByCurrentAuthUser = nil }()

		repoupdater.MockStatusMessages = func(_ context.Context) (*protocol.StatusMessagesResponse, error) {
			res := &protocol.StatusMessagesResponse{Messages: []protocol.StatusMessage{}}
			return res, nil
		}
		defer func() { repoupdater.MockStatusMessages = nil }()

		gqltesting.RunTests(t, []*gqltesting.Test{
			{
				Schema: mustParseGraphQLSchema(t),
				Query:  graphqlQuery,
				ExpectedResult: `
				{
					"statusMessages": []
				}
			`,
			},
		})
	})

	t.Run("messages", func(t *testing.T) {
		db.Mocks.Users.GetByCurrentAuthUser = func(ctx context.Context) (*types.User, error) {
			return &types.User{ID: 1, SiteAdmin: true}, nil
		}
		defer func() { db.Mocks.Users.GetByCurrentAuthUser = nil }()

		db.Mocks.ExternalServices.GetByID = func(id int64) (*types.ExternalService, error) {
			return &types.ExternalService{ID: 1, DisplayName: "GitHub.com testing"}, nil
		}
		defer func() { db.Mocks.ExternalServices.GetByID = nil }()

		repoupdater.MockStatusMessages = func(_ context.Context) (*protocol.StatusMessagesResponse, error) {
			res := &protocol.StatusMessagesResponse{Messages: []protocol.StatusMessage{
				{
					Cloning: &protocol.CloningProgress{
						Message: "Currently cloning 5 repositories in parallel...",
					},
				},
				{
					ExternalServiceSyncError: &protocol.ExternalServiceSyncError{
						Message:           "Authentication failed. Please check credentials.",
						ExternalServiceId: 1,
					},
				},
				{
					SyncError: &protocol.SyncError{
						Message: "Could not save to database",
					},
				},
			}}
			return res, nil
		}
		defer func() { repoupdater.MockStatusMessages = nil }()

		gqltesting.RunTests(t, []*gqltesting.Test{
			{
				Schema: mustParseGraphQLSchema(t),
				Query:  graphqlQuery,
				ExpectedResult: `
					{
						"statusMessages": [
							{
								"__typename": "CloningProgress",
								"message": "Currently cloning 5 repositories in parallel..."
							},
							{
								"__typename": "ExternalServiceSyncError",
								"externalService": {
									"displayName": "GitHub.com testing",
									"id": "RXh0ZXJuYWxTZXJ2aWNlOjE="
								},
								"message": "Authentication failed. Please check credentials."
							},
							{
								"__typename": "SyncError",
								"message": "Could not save to database"
							}
						]
					}
				`,
			},
		})
	})
}
