package graphqlbackend

import (
	"context"

	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/internal/db"
)

// accessTokenResolver resolves an access token.
//
// Access tokens provide scoped access to a user account (not just the API).
// This is different than other services such as GitHub, where access tokens
// only provide access to the API. This is OK for us because our general UI is
// completely implemented via our API, so access token authentication with our
// UI does not provide any additional functionality. In contrast, GitHub and
// other services likely allow user accounts to do more than what access tokens
// alone can via the API.
type accessTokenResolver struct {
	accessToken db.AccessToken
}

func accessTokenByID(ctx context.Context, id graphql.ID) (*accessTokenResolver, error) {
	accessTokenID, err := unmarshalAccessTokenID(id)
	if err != nil {
		return nil, err
	}
	accessToken, err := db.AccessTokens.GetByID(ctx, accessTokenID)
	if err != nil {
		return nil, err
	}
	// 🚨 SECURITY: Only the user (token owner) and site admins may retrieve the token.
	if err := backend.CheckSiteAdminOrSameUser(ctx, accessToken.SubjectUserID); err != nil {
		return nil, err
	}
	return &accessTokenResolver{accessToken: *accessToken}, nil
}

func marshalAccessTokenID(id int64) graphql.ID { return relay.MarshalID("AccessToken", id) }

func unmarshalAccessTokenID(id graphql.ID) (accessTokenID int64, err error) {
	err = relay.UnmarshalSpec(id, &accessTokenID)
	return
}

func (r *accessTokenResolver) ID() graphql.ID { return marshalAccessTokenID(r.accessToken.ID) }

func (r *accessTokenResolver) Subject(ctx context.Context) (*UserResolver, error) {
	return UserByIDInt32(ctx, r.accessToken.SubjectUserID)
}

func (r *accessTokenResolver) Scopes() []string { return r.accessToken.Scopes }

func (r *accessTokenResolver) Note() string { return r.accessToken.Note }

func (r *accessTokenResolver) Creator(ctx context.Context) (*UserResolver, error) {
	return UserByIDInt32(ctx, r.accessToken.CreatorUserID)
}

func (r *accessTokenResolver) CreatedAt() DateTime { return DateTime{Time: r.accessToken.CreatedAt} }

func (r *accessTokenResolver) LastUsedAt() *DateTime {
	return DateTimeOrNil(r.accessToken.LastUsedAt)
}
