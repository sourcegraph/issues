package graphqlbackend

import (
	"context"

	"github.com/graph-gophers/graphql-go"
	"github.com/inconshreveable/log15"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/internal/authz"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/db"
)

func (r *UserResolver) Emails(ctx context.Context) ([]*userEmailResolver, error) {
	// 🚨 SECURITY: Only the self user and site admins can fetch a user's emails.
	if err := backend.CheckSiteAdminOrSameUser(ctx, r.user.ID); err != nil {
		return nil, err
	}

	userEmails, err := db.UserEmails.ListByUser(ctx, db.UserEmailsListOptions{
		UserID: r.user.ID,
	})
	if err != nil {
		return nil, err
	}

	rs := make([]*userEmailResolver, len(userEmails))
	for i, userEmail := range userEmails {
		rs[i] = &userEmailResolver{
			userEmail: *userEmail,
			user:      r,
		}
	}
	return rs, nil
}

type userEmailResolver struct {
	userEmail db.UserEmail
	user      *UserResolver
}

func (r *userEmailResolver) Email() string { return r.userEmail.Email }

func (r *userEmailResolver) IsPrimary(ctx context.Context) (bool, error) {
	email, _, err := db.UserEmails.GetPrimaryEmail(ctx, r.user.user.ID)
	if err != nil {
		return false, err
	}
	return email == r.userEmail.Email, nil
}

func (r *userEmailResolver) Verified() bool { return r.userEmail.VerifiedAt != nil }
func (r *userEmailResolver) VerificationPending() bool {
	return !r.Verified() && conf.EmailVerificationRequired()
}
func (r *userEmailResolver) User() *UserResolver { return r.user }

func (r *userEmailResolver) ViewerCanManuallyVerify(ctx context.Context) (bool, error) {
	if err := backend.CheckCurrentUserIsSiteAdmin(ctx); err == backend.ErrNotAuthenticated || err == backend.ErrMustBeSiteAdmin {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

func (r *schemaResolver) AddUserEmail(ctx context.Context, args *struct {
	User  graphql.ID
	Email string
}) (*EmptyResponse, error) {
	userID, err := UnmarshalUserID(args.User)
	if err != nil {
		return nil, err
	}
	if err := backend.UserEmails.Add(ctx, userID, args.Email); err != nil {
		return nil, err
	}

	if conf.CanSendEmail() {
		if err := backend.UserEmails.SendUserEmailOnFieldUpdate(ctx, userID, "added an email"); err != nil {
			log15.Warn("Failed to send email to inform user of email addition", "error", err)
		}
	}

	return &EmptyResponse{}, nil
}

func (r *schemaResolver) RemoveUserEmail(ctx context.Context, args *struct {
	User  graphql.ID
	Email string
}) (*EmptyResponse, error) {
	userID, err := UnmarshalUserID(args.User)
	if err != nil {
		return nil, err
	}

	// 🚨 SECURITY: Only the user and site admins can remove an email address from a user.
	if err := backend.CheckSiteAdminOrSameUser(ctx, userID); err != nil {
		return nil, err
	}

	if err := db.UserEmails.Remove(ctx, userID, args.Email); err != nil {
		return nil, err
	}

	// 🚨 SECURITY: If an email is removed, invalidate any existing password reset tokens that may have been sent to that email.
	if err := db.Users.DeletePasswordResetCode(ctx, userID); err != nil {
		return nil, err
	}

	if conf.CanSendEmail() {
		if err := backend.UserEmails.SendUserEmailOnFieldUpdate(ctx, userID, "removed an email"); err != nil {
			log15.Warn("Failed to send email to inform user of email removal", "error", err)
		}
	}

	return &EmptyResponse{}, nil
}

func (r *schemaResolver) SetUserEmailVerified(ctx context.Context, args *struct {
	User     graphql.ID
	Email    string
	Verified bool
}) (*EmptyResponse, error) {
	// 🚨 SECURITY: Only site admins (NOT users themselves) can manually set email verification
	// status. Users themselves must go through the normal email verification process.
	if err := backend.CheckCurrentUserIsSiteAdmin(ctx); err != nil {
		return nil, err
	}

	userID, err := UnmarshalUserID(args.User)
	if err != nil {
		return nil, err
	}
	if err := db.UserEmails.SetVerified(ctx, userID, args.Email, args.Verified); err != nil {
		return nil, err
	}

	// Avoid unnecessary calls if the email is set to unverified.
	if args.Verified {
		if err = db.Authz.GrantPendingPermissions(ctx, &db.GrantPendingPermissionsArgs{
			UserID: userID,
			Perm:   authz.Read,
			Type:   authz.PermRepos,
		}); err != nil {
			log15.Error("Failed to grant user pending permissions", "userID", userID, "error", err)
		}
	}

	return &EmptyResponse{}, nil
}
