package graphqlbackend

import (
	"context"

	"github.com/inconshreveable/log15"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/globals"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/internal/auth/userpasswd"
	"github.com/sourcegraph/sourcegraph/internal/authz"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
	"github.com/sourcegraph/sourcegraph/internal/types"
)

func (r *schemaResolver) CreateUser(ctx context.Context, args *struct {
	Username string
	Email    *string
}) (*createUserResult, error) {
	// 🚨 SECURITY: Only site admins can create user accounts.
	if err := backend.CheckCurrentUserIsSiteAdmin(ctx, r.db); err != nil {
		return nil, err
	}

	var email string
	if args.Email != nil {
		email = *args.Email
	}

	// The new user will be created with a verified email address.
	user, err := database.Users(r.db).Create(ctx, database.NewUser{
		Username:        args.Username,
		Email:           email,
		EmailIsVerified: true,
		Password:        backend.MakeRandomHardToGuessPassword(),
	})
	if err != nil {
		return nil, err
	}

	if err = database.GlobalAuthz.GrantPendingPermissions(ctx, &database.GrantPendingPermissionsArgs{
		UserID: user.ID,
		Perm:   authz.Read,
		Type:   authz.PermRepos,
	}); err != nil {
		log15.Error("Failed to grant user pending permissions", "userID", user.ID, "error", err)
	}

	return &createUserResult{db: r.db, user: user}, nil
}

// createUserResult is the result of Mutation.createUser.
//
// 🚨 SECURITY: Only site admins should be able to instantiate this value.
type createUserResult struct {
	db   dbutil.DB
	user *types.User
}

func (r *createUserResult) User() *UserResolver { return NewUserResolver(r.db, r.user) }

func (r *createUserResult) ResetPasswordURL(ctx context.Context) (*string, error) {
	if !userpasswd.ResetPasswordEnabled() {
		return nil, nil
	}

	var ru string
	if conf.CanSendEmail() {
		ru, err := userpasswd.HandleSetPasswordEmail(ctx, r.db, r.user.ID)
		if err == nil {
			return &ru, nil
		}
		log15.Error("failed to send email", "error", err)
	}

	// This method modifies the DB, which is somewhat counterintuitive for a "value" type from an
	// implementation POV. Its behavior is justified because it is convenient and intuitive from the
	// POV of the API consumer.
	resetURL, err := backend.MakePasswordResetURL(ctx, r.db, r.user.ID)
	if err != nil {
		return nil, err
	}
	ru = globals.ExternalURL().ResolveReference(resetURL).String()
	return &ru, nil
}
