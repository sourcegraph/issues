package auth

import (
	"context"
	"fmt"

	"github.com/inconshreveable/log15"
	"github.com/sourcegraph/sourcegraph/internal/actor"
	"github.com/sourcegraph/sourcegraph/internal/authz"
	"github.com/sourcegraph/sourcegraph/internal/db"
	"github.com/sourcegraph/sourcegraph/internal/errcode"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
)

var MockGetAndSaveUser func(ctx context.Context, op GetAndSaveUserOp) (userID int32, safeErrMsg string, err error)

type GetAndSaveUserOp struct {
	UserProps           db.NewUser
	ExternalAccount     extsvc.AccountSpec
	ExternalAccountData extsvc.AccountData
	CreateIfNotExist    bool
	LookUpByUsername    bool
}

// GetAndSaveUser accepts authentication information associated with a given user, validates and applies
// the necessary updates to the DB, and returns the user ID after the updates have been applied.
//
// At a high level, it does the following:
// 1. Determine the identity of the user by applying the following rules in order:
//    a. If ctx contains an authenticated Actor, the Actor's identity is the user identity.
//    b. Look up the user by external account ID.
//    c. If the email specified in op.UserProps is verified, Look up the user by verified email.
//       If op.LookUpByUsername is true, look up by username instead of verified email.
//       (Note: most clients should look up by email, as username is typically insecure.)
//    d. If op.CreateIfNotExist is true, attempt to create a new user with the properties
//       specified in op.UserProps. This may fail if the desired username is already taken.
//    e. If a new user is successfully created, attempt to grant pending permissions.
// 2. Ensure that the user is associated with the external account information. This means
//    creating the external account if it does not already exist or updating it if it
//    already does.
// 3. Update any user props that have changed.
// 4. Return the user ID.
//
// 🚨 SECURITY: It is the caller's responsibility to ensure the veracity of the information that
// op contains (e.g., by receiving it from the appropriate authentication mechanism). It must
// also ensure that the user identity implied by op is consistent. Specifically, the values used
// in step 1 above must be consistent:
// * The authenticated Actor, if it exists
// * op.ExternalAccount
// * op.UserProps, especially op.UserProps.Email
//
// 🚨 SECURITY: The safeErrMsg is an error message that can be shown to unauthenticated users to
// describe the problem. The err may contain sensitive information and should only be written to the
// server error logs, not to the HTTP response to shown to unauthenticated users.
func GetAndSaveUser(ctx context.Context, op GetAndSaveUserOp) (userID int32, safeErrMsg string, err error) {
	if MockGetAndSaveUser != nil {
		return MockGetAndSaveUser(ctx, op)
	}
	userID, userSaved, extAcctSaved, safeErrMsg, err := func() (int32, bool, bool, string, error) {
		if actor := actor.FromContext(ctx); actor.IsAuthenticated() {
			return actor.UID, false, false, "", nil
		}

		uid, lookupByExternalErr := db.ExternalAccounts.LookupUserAndSave(ctx, op.ExternalAccount, op.ExternalAccountData)
		if lookupByExternalErr == nil {
			return uid, false, true, "", nil
		}
		if !errcode.IsNotFound(lookupByExternalErr) {
			return 0, false, false, "Unexpected error looking up the Sourcegraph user account associated with the external account. Ask a site admin for help.", lookupByExternalErr
		}

		if op.LookUpByUsername {
			user, getByUsernameErr := db.Users.GetByUsername(ctx, op.UserProps.Username)
			if getByUsernameErr == nil {
				return user.ID, false, false, "", nil
			}
			if !errcode.IsNotFound(getByUsernameErr) {
				return 0, false, false, "Unexpected error looking up the Sourcegraph user by username. Ask a site admin for help.", getByUsernameErr
			}
			if !op.CreateIfNotExist {
				return 0, false, false, fmt.Sprintf("User account with username %q does not exist. Ask a site admin to create your account.", op.UserProps.Username), getByUsernameErr
			}
		} else if op.UserProps.EmailIsVerified {
			user, getByVerifiedEmailErr := db.Users.GetByVerifiedEmail(ctx, op.UserProps.Email)
			if getByVerifiedEmailErr == nil {
				return user.ID, false, false, "", nil
			}
			if !errcode.IsNotFound(getByVerifiedEmailErr) {
				return 0, false, false, "Unexpected error looking up the Sourcegraph user by verified email. Ask a site admin for help.", getByVerifiedEmailErr
			}
			if !op.CreateIfNotExist {
				return 0, false, false, fmt.Sprintf("User account with verified email %q does not exist. Ask a site admin to create your account and then verify your email.", op.UserProps.Email), getByVerifiedEmailErr
			}
		}
		if !op.CreateIfNotExist {
			return 0, false, false, "It looks like this is your first time signing in with this external identity. Sourcegraph couldn't link it to an existing user, because no verified email was provided. Ask your site admin to configure the auth provider to include the user's verified email on sign-in.", lookupByExternalErr
		}

		// If CreateIfNotExist is true, create the new user, regardless of whether the email was verified or not.
		userID, err := db.ExternalAccounts.CreateUserAndSave(ctx, op.UserProps, op.ExternalAccount, op.ExternalAccountData)
		switch {
		case db.IsUsernameExists(err):
			return 0, false, false, fmt.Sprintf("Username %q already exists, but no verified email matched %q", op.UserProps.Username, op.UserProps.Email), err
		case errcode.PresentationMessage(err) != "":
			return 0, false, false, errcode.PresentationMessage(err), err
		case err != nil:
			return 0, false, false, "Unable to create a new user account due to a unexpected error. Ask a site admin for help.", err
		}

		if err = db.Authz.GrantPendingPermissions(ctx, &db.GrantPendingPermissionsArgs{
			UserID: userID,
			Perm:   authz.Read,
			Type:   authz.PermRepos,
		}); err != nil {
			log15.Error("Failed to grant user pending permissions", "userID", userID, "error", err)
		}

		return userID, true, true, "", nil
	}()
	if err != nil {
		return 0, safeErrMsg, err
	}

	// Update user properties, if they've changed
	if !userSaved {
		// Update user in our DB if their profile info changed on the issuer. (Except username and
		// email, which the user is somewhat likely to want to control separately on Sourcegraph.)
		user, err := db.Users.GetByID(ctx, userID)
		if err != nil {
			return 0, "Unexpected error getting the Sourcegraph user account. Ask a site admin for help.", err
		}
		var userUpdate db.UserUpdate
		if user.DisplayName != op.UserProps.DisplayName {
			userUpdate.DisplayName = &op.UserProps.DisplayName
		}
		if user.AvatarURL != op.UserProps.AvatarURL {
			userUpdate.AvatarURL = &op.UserProps.AvatarURL
		}
		if userUpdate != (db.UserUpdate{}) {
			if err := db.Users.Update(ctx, user.ID, userUpdate); err != nil {
				return 0, "Unexpected error updating the Sourcegraph user account with new user profile information from the external account. Ask a site admin for help.", err
			}
		}
	}

	// Create/update the external account and ensure it's associated with the user ID
	if !extAcctSaved {
		err := db.ExternalAccounts.AssociateUserAndSave(ctx, userID, op.ExternalAccount, op.ExternalAccountData)
		if err != nil {
			return 0, "Unexpected error associating the external account with your Sourcegraph user. The most likely cause for this problem is that another Sourcegraph user is already linked with this external account. A site admin or the other user can unlink the account to fix this problem.", err
		}

		if err = db.Authz.GrantPendingPermissions(ctx, &db.GrantPendingPermissionsArgs{
			UserID: userID,
			Perm:   authz.Read,
			Type:   authz.PermRepos,
		}); err != nil {
			log15.Error("Failed to grant user pending permissions", "userID", userID, "error", err)
		}
	}

	return userID, "", nil
}
