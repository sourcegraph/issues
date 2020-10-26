package graphqlbackend

import (
	"context"
	"reflect"
	"testing"

	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/gqltesting"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/internal/actor"
	"github.com/sourcegraph/sourcegraph/internal/authz"
	"github.com/sourcegraph/sourcegraph/internal/db"
)

// 🚨 SECURITY: This tests that users can't create tokens for users they aren't allowed to do so for.
func TestMutation_CreateAccessToken(t *testing.T) {
	mockAccessTokensCreate := func(t *testing.T, wantCreatorUserID int32, wantScopes []string) {
		db.Mocks.AccessTokens.Create = func(subjectUserID int32, scopes []string, note string, creatorUserID int32) (int64, string, error) {
			if want := int32(1); subjectUserID != want {
				t.Errorf("got %v, want %v", subjectUserID, want)
			}
			if !reflect.DeepEqual(scopes, wantScopes) {
				t.Errorf("got %q, want %q", scopes, wantScopes)
			}
			if want := "n"; note != want {
				t.Errorf("got %q, want %q", note, want)
			}
			if creatorUserID != wantCreatorUserID {
				t.Errorf("got %v, want %v", creatorUserID, wantCreatorUserID)
			}
			return 1, "t", nil
		}
	}

	const uid1GQLID = "VXNlcjox"

	t.Run("authenticated as user", func(t *testing.T) {
		resetMocks()
		mockAccessTokensCreate(t, 1, []string{authz.ScopeUserAll})
		gqltesting.RunTests(t, []*gqltesting.Test{
			{
				Context: actor.WithActor(context.Background(), &actor.Actor{UID: 1}),
				Schema:  mustParseGraphQLSchema(t),
				Query: `
				mutation {
					createAccessToken(user: "` + uid1GQLID + `", scopes: ["user:all"], note: "n") {
						id
						token
					}
				}
			`,
				ExpectedResult: `
				{
					"createAccessToken": {
						"id": "QWNjZXNzVG9rZW46MQ==",
						"token": "t"
					}
				}
			`,
			},
		})
	})

	t.Run("authenticated as user, using invalid scopes", func(t *testing.T) {
		resetMocks()

		ctx := actor.WithActor(context.Background(), &actor.Actor{UID: 1})
		result, err := (&schemaResolver{}).CreateAccessToken(ctx, &createAccessTokenInput{User: uid1GQLID /* no scopes */, Note: "n"})
		if err == nil {
			t.Error("err == nil")
		}
		if result != nil {
			t.Errorf("got result %v, want nil", result)
		}
	})

	t.Run("authenticated as user, using site-admin-only scopes", func(t *testing.T) {
		resetMocks()
		db.Mocks.Users.GetByCurrentAuthUser = func(ctx context.Context) (*types.User, error) {
			return &types.User{ID: 1, SiteAdmin: false}, nil
		}
		defer func() { db.Mocks.Users.GetByCurrentAuthUser = nil }()

		ctx := actor.WithActor(context.Background(), &actor.Actor{UID: 1})
		result, err := (&schemaResolver{}).CreateAccessToken(ctx, &createAccessTokenInput{
			User:   uid1GQLID,
			Scopes: []string{authz.ScopeUserAll, authz.ScopeSiteAdminSudo},
			Note:   "n",
		})
		if want := backend.ErrMustBeSiteAdmin; err != want {
			t.Errorf("got err %v, want %v", err, want)
		}
		if result != nil {
			t.Errorf("got result %v, want nil", result)
		}
	})

	t.Run("authenticated as site admin, using site-admin-only scopes", func(t *testing.T) {
		resetMocks()
		mockAccessTokensCreate(t, 1, []string{authz.ScopeSiteAdminSudo, authz.ScopeUserAll})
		db.Mocks.Users.GetByCurrentAuthUser = func(ctx context.Context) (*types.User, error) {
			return &types.User{ID: 1, SiteAdmin: true}, nil
		}
		defer func() { db.Mocks.Users.GetByCurrentAuthUser = nil }()

		gqltesting.RunTests(t, []*gqltesting.Test{
			{
				Context: actor.WithActor(context.Background(), &actor.Actor{UID: 1}),
				Schema:  mustParseGraphQLSchema(t),
				Query: `
				mutation {
					createAccessToken(user: "` + uid1GQLID + `", scopes: ["user:all", "site-admin:sudo"], note: "n") {
						id
						token
					}
				}
			`,
				ExpectedResult: `
				{
					"createAccessToken": {
						"id": "QWNjZXNzVG9rZW46MQ==",
						"token": "t"
					}
				}
			`,
			},
		})
	})

	t.Run("authenticated as different user who is a site-admin", func(t *testing.T) {
		resetMocks()
		const differentSiteAdminUID = 234
		mockAccessTokensCreate(t, differentSiteAdminUID, []string{authz.ScopeUserAll})
		db.Mocks.Users.GetByCurrentAuthUser = func(ctx context.Context) (*types.User, error) {
			return &types.User{ID: differentSiteAdminUID, SiteAdmin: true}, nil
		}
		defer func() { db.Mocks.Users.GetByCurrentAuthUser = nil }()

		gqltesting.RunTests(t, []*gqltesting.Test{
			{
				Context: actor.WithActor(context.Background(), &actor.Actor{UID: differentSiteAdminUID}),
				Schema:  mustParseGraphQLSchema(t),
				Query: `
				mutation {
					createAccessToken(user: "` + uid1GQLID + `", scopes: ["user:all"], note: "n") {
						id
						token
					}
				}
			`,
				ExpectedResult: `
				{
					"createAccessToken": {
						"id": "QWNjZXNzVG9rZW46MQ==",
						"token": "t"
					}
				}
			`,
			},
		})
	})

	t.Run("unauthenticated", func(t *testing.T) {
		resetMocks()
		db.Mocks.Users.GetByCurrentAuthUser = func(ctx context.Context) (*types.User, error) { return nil, db.ErrNoCurrentUser }
		defer func() { db.Mocks.Users.GetByCurrentAuthUser = nil }()
		db.Mocks.Users.GetByID = func(_ context.Context, userID int32) (*types.User, error) {
			return &types.User{Username: "username"}, nil
		}
		defer func() { db.Mocks.Users.GetByID = nil }()

		ctx := actor.WithActor(context.Background(), nil)
		result, err := (&schemaResolver{}).CreateAccessToken(ctx, &createAccessTokenInput{User: uid1GQLID, Note: "n"})
		if err == nil {
			t.Error("Expected error, but there was none")
		}
		if result != nil {
			t.Errorf("got result %v, want nil", result)
		}
	})

	t.Run("authenticated as different non-site-admin user", func(t *testing.T) {
		resetMocks()
		const differentNonSiteAdminUID = 456
		db.Mocks.Users.GetByCurrentAuthUser = func(ctx context.Context) (*types.User, error) { return &types.User{ID: differentNonSiteAdminUID}, nil }
		defer func() { db.Mocks.Users.GetByCurrentAuthUser = nil }()
		db.Mocks.Users.GetByID = func(_ context.Context, userID int32) (*types.User, error) {
			return &types.User{Username: "username"}, nil
		}
		defer func() { db.Mocks.Users.GetByID = nil }()

		ctx := actor.WithActor(context.Background(), &actor.Actor{UID: differentNonSiteAdminUID})
		result, err := (&schemaResolver{}).CreateAccessToken(ctx, &createAccessTokenInput{User: uid1GQLID, Note: "n"})
		if err == nil {
			t.Error("Expected error, but there was none")
		}
		if result != nil {
			t.Errorf("got result %v, want nil", result)
		}
	})
}

// 🚨 SECURITY: This tests that users can't delete tokens they shouldn't be allowed to delete.
func TestMutation_DeleteAccessToken(t *testing.T) {
	mockAccessTokens := func(t *testing.T) {
		db.Mocks.AccessTokens.DeleteByID = func(id int64, subjectUserID int32) error {
			if want := int64(1); id != want {
				t.Errorf("got %q, want %q", id, want)
			}
			if want := int32(2); subjectUserID != want {
				t.Errorf("got %v, want %v", subjectUserID, want)
			}
			return nil
		}
		db.Mocks.AccessTokens.GetByID = func(id int64) (*db.AccessToken, error) {
			if want := int64(1); id != want {
				t.Errorf("got %d, want %d", id, want)
			}
			return &db.AccessToken{ID: 1, SubjectUserID: 2}, nil
		}
	}

	token1GQLID := graphql.ID("QWNjZXNzVG9rZW46MQ==")

	t.Run("authenticated as user", func(t *testing.T) {
		resetMocks()
		mockAccessTokens(t)
		gqltesting.RunTests(t, []*gqltesting.Test{
			{
				Context: actor.WithActor(context.Background(), &actor.Actor{UID: 2}),
				Schema:  mustParseGraphQLSchema(t),
				Query: `
				mutation {
					deleteAccessToken(byID: "` + string(token1GQLID) + `") {
						alwaysNil
					}
				}
			`,
				ExpectedResult: `
				{
					"deleteAccessToken": {
						"alwaysNil": null
					}
				}
			`,
			},
		})
	})

	t.Run("authenticated as different user who is a site-admin", func(t *testing.T) {
		resetMocks()
		const differentSiteAdminUID = 234
		mockAccessTokens(t)
		db.Mocks.Users.GetByCurrentAuthUser = func(ctx context.Context) (*types.User, error) {
			return &types.User{ID: differentSiteAdminUID, SiteAdmin: true}, nil
		}
		defer func() { db.Mocks.Users.GetByCurrentAuthUser = nil }()

		gqltesting.RunTests(t, []*gqltesting.Test{
			{
				Context: actor.WithActor(context.Background(), &actor.Actor{UID: differentSiteAdminUID}),
				Schema:  mustParseGraphQLSchema(t),
				Query: `
				mutation {
					deleteAccessToken(byID: "` + string(token1GQLID) + `") {
						alwaysNil
					}
				}
			`,
				ExpectedResult: `
				{
					"deleteAccessToken": {
						"alwaysNil": null
					}
				}
			`,
			},
		})
	})

	t.Run("unauthenticated", func(t *testing.T) {
		resetMocks()
		mockAccessTokens(t)
		db.Mocks.Users.GetByCurrentAuthUser = func(ctx context.Context) (*types.User, error) { return nil, db.ErrNoCurrentUser }
		defer func() { db.Mocks.Users.GetByCurrentAuthUser = nil }()
		db.Mocks.Users.GetByID = func(_ context.Context, userID int32) (*types.User, error) {
			return &types.User{Username: "username"}, nil
		}
		defer func() { db.Mocks.Users.GetByID = nil }()

		ctx := actor.WithActor(context.Background(), nil)
		result, err := (&schemaResolver{}).DeleteAccessToken(ctx, &deleteAccessTokenInput{ByID: &token1GQLID})
		if err == nil {
			t.Error("Expected error, but there was none")
		}
		if result != nil {
			t.Errorf("got result %v, want nil", result)
		}
	})

	t.Run("authenticated as different non-site-admin user", func(t *testing.T) {
		resetMocks()
		const differentNonSiteAdminUID = 456
		mockAccessTokens(t)
		db.Mocks.Users.GetByCurrentAuthUser = func(ctx context.Context) (*types.User, error) { return &types.User{ID: differentNonSiteAdminUID}, nil }
		defer func() { db.Mocks.Users.GetByCurrentAuthUser = nil }()
		db.Mocks.Users.GetByID = func(_ context.Context, userID int32) (*types.User, error) {
			return &types.User{Username: "username"}, nil
		}
		defer func() { db.Mocks.Users.GetByID = nil }()

		ctx := actor.WithActor(context.Background(), &actor.Actor{UID: differentNonSiteAdminUID})
		result, err := (&schemaResolver{}).DeleteAccessToken(ctx, &deleteAccessTokenInput{ByID: &token1GQLID})
		if err == nil {
			t.Error("Expected error, but there was none")
		}
		if result != nil {
			t.Errorf("got result %v, want nil", result)
		}
	})
}
