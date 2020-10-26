package graphqlbackend

import (
	"context"

	"github.com/graph-gophers/graphql-go"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/globals"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/internal/auth/userpasswd"
	"github.com/sourcegraph/sourcegraph/internal/db"
)

type randomizeUserPasswordResult struct {
	userID int32
}

func (r *randomizeUserPasswordResult) ResetPasswordURL(ctx context.Context) (*string, error) {
	if !userpasswd.ResetPasswordEnabled() {
		return nil, nil
	}

	// This method modifies the DB, which is somewhat counterintuitive for a "value" type from an
	// implementation POV. Its behavior is justified because it is convenient and intuitive from the
	// POV of the API consumer.
	resetURL, err := backend.MakePasswordResetURL(ctx, r.userID)
	if err != nil {
		return nil, err
	}
	urlStr := globals.ExternalURL().ResolveReference(resetURL).String()
	return &urlStr, nil
}

func (*schemaResolver) RandomizeUserPassword(ctx context.Context, args *struct {
	User graphql.ID
}) (*randomizeUserPasswordResult, error) {
	// 🚨 SECURITY: Only site admins can randomize user passwords.
	if err := backend.CheckCurrentUserIsSiteAdmin(ctx); err != nil {
		return nil, err
	}

	userID, err := UnmarshalUserID(args.User)
	if err != nil {
		return nil, err
	}

	if err := db.Users.RandomizePasswordAndClearPasswordResetRateLimit(ctx, userID); err != nil {
		return nil, err
	}

	return &randomizeUserPasswordResult{userID: userID}, nil
}
