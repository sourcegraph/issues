package backend

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/url"

	"github.com/inconshreveable/log15"
	"github.com/pkg/errors"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/envvar"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/globals"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/internal/app/router"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/db"
	"github.com/sourcegraph/sourcegraph/internal/errcode"
	"github.com/sourcegraph/sourcegraph/internal/txemail"
	"github.com/sourcegraph/sourcegraph/internal/txemail/txtypes"
)

// UserEmails contains backend methods related to user email addresses.
var UserEmails = &userEmails{}

type userEmails struct{}

// checkEmailAbuse performs abuse prevention checks to prevent email abuse, i.e. users using emails
// of other people whom they want to annoy.
func checkEmailAbuse(ctx context.Context, userID int32) (abused bool, reason string, err error) {
	if conf.EmailVerificationRequired() {
		emails, err := db.UserEmails.ListByUser(ctx, db.UserEmailsListOptions{
			UserID: userID,
		})
		if err != nil {
			return false, "", err
		}

		var verifiedCount, unverifiedCount int
		for _, email := range emails {
			if email.VerifiedAt == nil {
				unverifiedCount++
			} else {
				verifiedCount++
			}
		}

		// Abuse prevention check 1: Require user to have at least one verified email address
		// before adding another.
		//
		// (We need to also allow users who have zero addresses to add one, or else they could
		// delete all emails and then get into an unrecoverable state.)
		//
		// TODO(sqs): prevent users from deleting their last email, when we have the true notion
		// of a "primary" email address.)
		if verifiedCount == 0 && len(emails) != 0 {
			return true, "a verified email is required before you can add additional email addressed to your account", nil
		}

		// Abuse prevention check 2: Forbid user from having many unverified emails to prevent attackers from using this to
		// send spam or a high volume of annoying emails.
		const maxUnverified = 3
		if unverifiedCount >= maxUnverified {
			return true, "too many existing unverified email addresses", nil
		}
	}
	if envvar.SourcegraphDotComMode() {
		// Abuse prevention check 3: Set a quota on Sourcegraph.com users to prevent abuse.
		//
		// There is no quota for on-prem instances because we assume they can trust their users
		// to not abuse adding emails.
		//
		// TODO(sqs): This reuses the "invite quota", which is really just a number that counts
		// down (not specific to invites). Generalize this to just "quota" (remove "invite" from
		// the name).
		if ok, err := db.Users.CheckAndDecrementInviteQuota(ctx, userID); err != nil {
			return false, "", err
		} else if !ok {
			return true, "email address quota exceeded (contact support to increase the quota)", nil
		}
	}
	return false, "", nil
}

// Add adds an email address to a user. If email verification is required, it sends an email
// verification email.
func (userEmails) Add(ctx context.Context, userID int32, email string) error {
	// 🚨 SECURITY: Only the user and site admins can add an email address to a user.
	if err := CheckSiteAdminOrSameUser(ctx, userID); err != nil {
		return err
	}

	// Prevent abuse (users adding emails of other people whom they want to annoy) with the
	// following abuse prevention checks.
	if isSiteAdmin := CheckCurrentUserIsSiteAdmin(ctx) == nil; !isSiteAdmin {
		abused, reason, err := checkEmailAbuse(ctx, userID)
		if err != nil {
			return err
		} else if abused {
			return fmt.Errorf("refusing to add email address because %s", reason)
		}
	}

	var code *string
	if conf.EmailVerificationRequired() {
		tmp, err := MakeEmailVerificationCode()
		if err != nil {
			return err
		}
		code = &tmp
	}

	// Another user may have already verified this email address. If so, do not send another
	// verification email (it would be pointless and also be an abuse vector). Do not tell the
	// user that another user has already verified it, to avoid needlessly leaking the existence
	// of emails.
	var emailAlreadyExistsAndIsVerified bool
	if _, err := db.Users.GetByVerifiedEmail(ctx, email); err != nil && !errcode.IsNotFound(err) {
		return err
	} else if err == nil {
		emailAlreadyExistsAndIsVerified = true
	}

	if err := db.UserEmails.Add(ctx, userID, email, code); err != nil {
		return err
	}

	if conf.EmailVerificationRequired() && !emailAlreadyExistsAndIsVerified {
		// Send email verification email.
		if err := SendUserEmailVerificationEmail(ctx, email, *code); err != nil {
			return errors.Wrap(err, "SendUserEmailVerificationEmail")
		} else if err = db.UserEmails.SetLastVerificationSentAt(ctx, userID, email); err != nil {
			return errors.Wrap(err, "SetLastVerificationSentAt")
		}
	}
	return nil
}

// MakeEmailVerificationCode returns a random string that can be used as an email verification
// code. If there is not enough entropy to create a random string, it returns a non-nil error.
func MakeEmailVerificationCode() (string, error) {
	emailCodeBytes := make([]byte, 20)
	if _, err := rand.Read(emailCodeBytes); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(emailCodeBytes), nil
}

// SendUserEmailVerificationEmail sends an email to the user to verify the email address. The code
// is the verification code that the user must provide to verify their access to the email address.
func SendUserEmailVerificationEmail(ctx context.Context, email, code string) error {
	q := make(url.Values)
	q.Set("code", code)
	q.Set("email", email)
	verifyEmailPath, _ := router.Router().Get(router.VerifyEmail).URLPath()
	return txemail.Send(ctx, txemail.Message{
		To:       []string{email},
		Template: verifyEmailTemplates,
		Data: struct {
			Email string
			URL   string
		}{
			Email: email,
			URL: globals.ExternalURL().ResolveReference(&url.URL{
				Path:     verifyEmailPath.Path,
				RawQuery: q.Encode(),
			}).String(),
		},
	})
}

var verifyEmailTemplates = txemail.MustValidate(txtypes.Templates{
	Subject: `Verify your email on Sourcegraph`,
	Text: `
Verify your email address {{printf "%q" .Email}} on Sourcegraph by following this link:

  {{.URL}}
`,
	HTML: `
<p>Verify your email address {{printf "%q" .Email}} on Sourcegraph by following this link:</p>

<p><strong><a href="{{.URL}}">Verify email address</a></p>
`,
})

// SendUserEmailOnFieldUpdate sends the user an email that important account information has changed.
// The change is the information we want to provide the user about the change
func (userEmails) SendUserEmailOnFieldUpdate(ctx context.Context, id int32, change string) error {
	email, _, err := db.UserEmails.GetPrimaryEmail(ctx, id)
	if err != nil {
		log15.Warn("Failed to get user email", "error", err)
		return err
	}
	usr, err := db.Users.GetByID(ctx, id)
	if err != nil {
		log15.Warn("Failed to get user from database", "error", err)
		return err
	}

	return txemail.Send(ctx, txemail.Message{
		To:       []string{email},
		Template: updateAccountEmailTemplate,
		Data: struct {
			Email    string
			Change   string
			Username string
		}{
			Email:    email,
			Change:   change,
			Username: usr.Username,
		},
	})
}

var updateAccountEmailTemplate = txemail.MustValidate(txtypes.Templates{
	Subject: `Update to your Sourcegraph account`,
	Text: `
Somebody (likely you) {{.Change}} for the user {{.Username}} on Sourcegraph.

If this was not you please change your password immediately.
`,
	HTML: `
<p>
Somebody (likely you) <strong>{{.Change}}</strong> for the user <strong>{{.Username}}</strong> on Sourcegraph.
</p>

<p><strong>If this was not you please change your password immediately.</strong></p>
`,
})
