package auth

import (
	"net/http"

	"github.com/inconshreveable/log15"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/auth"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/internal/session"
	"github.com/sourcegraph/sourcegraph/internal/actor"
	"github.com/sourcegraph/sourcegraph/internal/db"
	"github.com/sourcegraph/sourcegraph/internal/env"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
)

const (
	overrideSecretHeader   = "X-Override-Auth-Secret"
	overrideUsernameHeader = "X-Override-Auth-Username"

	defaultUsername = "override-auth-user"
)

// OverrideAuthMiddleware is middleware that causes a new authenticated session (associated with a
// new user named "anon-user") to be started if the client provides a secret header value specified
// in site config.
//
// It is used to enable our e2e tests to authenticate to https://sourcegraph.sgdev.org without
// needing to give them Google Workspace access.
func OverrideAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		secret := envOverrideAuthSecret
		// Accept both old header (X-Oidc-Override, deprecated) and new overrideSecretHeader for now.
		if secret != "" && (r.Header.Get("X-Oidc-Override") == secret || r.Header.Get(overrideSecretHeader) == secret) {
			username := r.Header.Get(overrideUsernameHeader)
			if username == "" {
				username = defaultUsername
			}

			userID, safeErrMsg, err := auth.GetAndSaveUser(r.Context(), auth.GetAndSaveUserOp{
				UserProps: db.NewUser{
					Username:        username,
					Email:           username + "+override@example.com",
					EmailIsVerified: true,
				},
				ExternalAccount: extsvc.AccountSpec{
					ServiceType: "override",
					AccountID:   username,
				},
				CreateIfNotExist: true,
			})
			if err != nil {
				log15.Error("Error getting/creating auth-override user.", "error", err, "userErr", safeErrMsg)
				http.Error(w, safeErrMsg, http.StatusInternalServerError)
				return
			}

			// Make the user a site admin because that is more useful for e2e tests and local dev
			// scripting (which are the use cases of this override auth provider).
			if err := db.Users.SetIsSiteAdmin(r.Context(), userID, true); err != nil {
				log15.Error("Error setting auth-override user as site admin.", "error", err)
				http.Error(w, "", http.StatusInternalServerError)
				return
			}

			a := actor.FromUser(userID)
			if err := session.SetActor(w, r, a, 0); err != nil {
				log15.Error("Error starting auth-override session.", "error", err)
				http.Error(w, "error starting auth-override session", http.StatusInternalServerError)
				return
			}

			r = r.WithContext(actor.WithActor(r.Context(), a))
		}

		next.ServeHTTP(w, r)
	})
}

// envOverrideAuthSecret (the env var OVERRIDE_AUTH_SECRET) is the preferred source of the secret
// for overriding auth.
var envOverrideAuthSecret = env.Get("OVERRIDE_AUTH_SECRET", "", "X-Override-Auth-Secret HTTP request header value used to authenticate site-admin-authed sessions (use X-Override-Auth-Username header to set username)")
