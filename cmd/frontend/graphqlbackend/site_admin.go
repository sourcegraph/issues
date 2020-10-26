package graphqlbackend

import (
	"context"

	"github.com/graph-gophers/graphql-go"
	"github.com/pkg/errors"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/external/session"
	"github.com/sourcegraph/sourcegraph/internal/authz"
	"github.com/sourcegraph/sourcegraph/internal/db"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
)

func (*schemaResolver) DeleteUser(ctx context.Context, args *struct {
	User graphql.ID
	Hard *bool
}) (*EmptyResponse, error) {
	// 🚨 SECURITY: Only site admins can delete users.
	if err := backend.CheckCurrentUserIsSiteAdmin(ctx); err != nil {
		return nil, err
	}

	userID, err := UnmarshalUserID(args.User)
	if err != nil {
		return nil, err
	}

	currentUser, err := CurrentUser(ctx)
	if err != nil {
		return nil, err
	}
	if currentUser.ID() == args.User {
		return nil, errors.New("unable to delete current user")
	}

	// Collect username, verified email addresses, and external accounts to be used
	// for revoking user permissions later, otherwise they will be removed from database
	// if it's a hard delete.
	user, err := db.Users.GetByID(ctx, userID)
	if err != nil {
		return nil, errors.Wrap(err, "get user by ID")
	}

	var accounts []*extsvc.Accounts

	extAccounts, err := db.ExternalAccounts.List(ctx, db.ExternalAccountsListOptions{UserID: userID})
	if err != nil {
		return nil, errors.Wrap(err, "list external accounts")
	}
	for _, acct := range extAccounts {
		accounts = append(accounts, &extsvc.Accounts{
			ServiceType: acct.ServiceType,
			ServiceID:   acct.ServiceID,
			AccountIDs:  []string{acct.AccountID},
		})
	}

	verifiedEmails, err := db.UserEmails.ListByUser(ctx, db.UserEmailsListOptions{
		UserID:       user.ID,
		OnlyVerified: true,
	})
	if err != nil {
		return nil, err
	}
	emailStrs := make([]string, len(verifiedEmails))
	for i := range verifiedEmails {
		emailStrs[i] = verifiedEmails[i].Email
	}
	accounts = append(accounts, &extsvc.Accounts{
		ServiceType: authz.SourcegraphServiceType,
		ServiceID:   authz.SourcegraphServiceID,
		AccountIDs:  append(emailStrs, user.Username),
	})

	if args.Hard != nil && *args.Hard {
		if err := db.Users.HardDelete(ctx, user.ID); err != nil {
			return nil, err
		}
	} else {
		if err := db.Users.Delete(ctx, user.ID); err != nil {
			return nil, err
		}
	}

	// NOTE: Practically, we don't reuse the ID for any new users, and the situation of left-over pending permissions
	// is possible but highly unlikely. Therefore, there is no need to roll back user deletion even if this step failed.
	// This call is purely for the purpose of cleanup.
	if err := db.Authz.RevokeUserPermissions(ctx, &db.RevokeUserPermissionsArgs{
		UserID:   user.ID,
		Accounts: accounts,
	}); err != nil {
		return nil, err
	}

	return &EmptyResponse{}, nil
}

func (*schemaResolver) DeleteOrganization(ctx context.Context, args *struct {
	Organization graphql.ID
}) (*EmptyResponse, error) {
	// 🚨 SECURITY: Only site admins can delete orgs.
	if err := backend.CheckCurrentUserIsSiteAdmin(ctx); err != nil {
		return nil, err
	}

	orgID, err := UnmarshalOrgID(args.Organization)
	if err != nil {
		return nil, err
	}

	if err := db.Orgs.Delete(ctx, orgID); err != nil {
		return nil, err
	}
	return &EmptyResponse{}, nil
}

func (*schemaResolver) SetUserIsSiteAdmin(ctx context.Context, args *struct {
	UserID    graphql.ID
	SiteAdmin bool
}) (*EmptyResponse, error) {
	// 🚨 SECURITY: Only site admins can promote other users to site admin (or demote from site
	// admin).
	if err := backend.CheckCurrentUserIsSiteAdmin(ctx); err != nil {
		return nil, err
	}

	user, err := CurrentUser(ctx)
	if err != nil {
		return nil, err
	}
	if user.ID() == args.UserID {
		return nil, errors.New("refusing to set current user site admin status")
	}

	userID, err := UnmarshalUserID(args.UserID)
	if err != nil {
		return nil, err
	}

	if err := db.Users.SetIsSiteAdmin(ctx, userID, args.SiteAdmin); err != nil {
		return nil, err
	}
	return &EmptyResponse{}, nil
}

func (r *schemaResolver) InvalidateSessionsByID(ctx context.Context, args *struct {
	UserID graphql.ID
}) (*EmptyResponse, error) {
	// 🚨 SECURITY: Only the site admin can invalidate the sessions of a user
	if err := backend.CheckCurrentUserIsSiteAdmin(ctx); err != nil {
		return nil, err
	}
	userID, err := UnmarshalUserID(args.UserID)
	if err != nil {
		return nil, err
	}
	if err := session.InvalidateSessionsByID(ctx, userID); err != nil {
		return nil, err
	}
	return &EmptyResponse{}, nil

}
