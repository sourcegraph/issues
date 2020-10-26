package userpasswd

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/inconshreveable/log15"
	"github.com/pkg/errors"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/globals"
	"github.com/sourcegraph/sourcegraph/internal/actor"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/db"
	"github.com/sourcegraph/sourcegraph/internal/errcode"
	"github.com/sourcegraph/sourcegraph/internal/txemail"
	"github.com/sourcegraph/sourcegraph/internal/txemail/txtypes"
)

// HandleResetPasswordInit initiates the builtin-auth password reset flow by sending a password-reset email.
func HandleResetPasswordInit(w http.ResponseWriter, r *http.Request) {
	if handleEnabledCheck(w) {
		return
	}
	if handleNotAuthenticatedCheck(w, r) {
		return
	}
	if !conf.CanSendEmail() {
		httpLogAndError(w, "Unable to reset password because email sending is not configured on this site", http.StatusNotFound)
		return
	}

	ctx := r.Context()
	var formData struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&formData); err != nil {
		httpLogAndError(w, "Could not decode password reset request body", http.StatusBadRequest, "err", err)
		return
	}

	if formData.Email == "" {
		httpLogAndError(w, "No email specified in password reset request", http.StatusBadRequest)
		return
	}

	usr, err := db.Users.GetByVerifiedEmail(ctx, formData.Email)
	if err != nil {
		// 🚨 SECURITY: We don't show an error message when the user is not found
		// as to not leak the existence of a given e-mail address in the database.
		if !errcode.IsNotFound(err) {
			httpLogAndError(w, "Failed to lookup user", http.StatusInternalServerError)
		}
		return
	}

	resetURL, err := backend.MakePasswordResetURL(ctx, usr.ID)
	if err == db.ErrPasswordResetRateLimit {
		httpLogAndError(w, "Too many password reset requests. Try again in a few minutes.", http.StatusTooManyRequests, "err", err)
		return
	} else if err != nil {
		httpLogAndError(w, "Could not reset password", http.StatusBadRequest, "err", err)
		return
	}

	if err := txemail.Send(r.Context(), txemail.Message{
		To:       []string{formData.Email},
		Template: resetPasswordEmailTemplates,
		Data: struct {
			Username string
			URL      string
		}{
			Username: usr.Username,
			URL:      globals.ExternalURL().ResolveReference(resetURL).String(),
		},
	}); err != nil {
		httpLogAndError(w, "Could not send reset password email", http.StatusInternalServerError, "err", err)
		return
	}
}

var setPasswordEmailTemplates = txemail.MustValidate(txtypes.Templates{
	Subject: `Set your Sourcegraph password`,
	Text: `
Your administrator created an account for you on Sourcegraph.

To set the password for {{.Username}} on Sourcegraph, follow this link:

  {{.URL}}
`,
	HTML: `
<p>
  Your administrator created an account for you on Sourcegraph.
</p>

<p><strong><a href="{{.URL}}">Set password for {{.Username}}</a></strong></p>
`,
})

// HandleSetPasswordEmail sends the password reset email directly to the user for users created by site admins.
func HandleSetPasswordEmail(ctx context.Context, id int32) (string, error) {
	e, _, err := db.UserEmails.GetPrimaryEmail(ctx, id)
	if err != nil {
		return "", errors.Wrap(err, "get user primary email")
	}

	usr, err := db.Users.GetByID(ctx, id)
	if err != nil {
		return "", errors.Wrap(err, "get user by ID")
	}

	ru, err := backend.MakePasswordResetURL(ctx, id)
	if err == db.ErrPasswordResetRateLimit {
		return "", err
	} else if err != nil {
		return "", errors.Wrap(err, "make password reset URL")
	}

	rus := globals.ExternalURL().ResolveReference(ru).String()
	if err := txemail.Send(ctx, txemail.Message{
		To:       []string{e},
		Template: setPasswordEmailTemplates,
		Data: struct {
			Username string
			URL      string
		}{
			Username: usr.Username,
			URL:      rus,
		},
	}); err != nil {
		return "", err
	}
	return rus, nil
}

var resetPasswordEmailTemplates = txemail.MustValidate(txtypes.Templates{
	Subject: `Reset your Sourcegraph password`,
	Text: `
Somebody (likely you) requested a password reset for the user {{.Username}} on Sourcegraph.

To reset the password for {{.Username}} on Sourcegraph, follow this link:

  {{.URL}}
`,
	HTML: `
<p>
  Somebody (likely you) requested a password reset for <strong>{{.Username}}</strong>
  on Sourcegraph.
</p>

<p><strong><a href="{{.URL}}">Reset password for {{.Username}}</a></strong></p>
`,
})

// HandleResetPasswordCode resets the password if the correct code is provided.
func HandleResetPasswordCode(w http.ResponseWriter, r *http.Request) {
	if handleEnabledCheck(w) {
		return
	}
	if handleNotAuthenticatedCheck(w, r) {
		return
	}

	ctx := r.Context()
	var params struct {
		UserID   int32  `json:"userID"`
		Code     string `json:"code"`
		Password string `json:"password"` // new password
	}
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		httpLogAndError(w, "Password reset with code: could not decode request body", http.StatusBadGateway, "err", err)
		return
	}

	if err := db.CheckPasswordLength(params.Password); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	success, err := db.Users.SetPassword(ctx, params.UserID, params.Code, params.Password)
	if err != nil {
		httpLogAndError(w, "Unexpected error", http.StatusInternalServerError, "err", err)
		return
	}

	if !success {
		http.Error(w, "Password reset code was invalid or expired.", http.StatusUnauthorized)
		return
	}

	if conf.CanSendEmail() {
		if err := backend.UserEmails.SendUserEmailOnFieldUpdate(ctx, params.UserID, "reset the password"); err != nil {
			log15.Warn("Failed to send email to inform user of password reset", "error", err)
		}
	}
}

func handleNotAuthenticatedCheck(w http.ResponseWriter, r *http.Request) (handled bool) {
	if actor.FromContext(r.Context()).IsAuthenticated() {
		http.Error(w, "Authenticated users may not perform password reset.", http.StatusInternalServerError)
		return true
	}
	return false
}
