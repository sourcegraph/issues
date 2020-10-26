package httpapi

import (
	"net/http"

	"github.com/inconshreveable/log15"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/internal/actor"
	"github.com/sourcegraph/sourcegraph/internal/authz"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/db"
	"github.com/sourcegraph/sourcegraph/internal/errcode"
)

// AccessTokenAuthMiddleware authenticates the user based on the
// token query parameter or the "Authorization" header.
func AccessTokenAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Authorization")

		var sudoUser string
		token := r.URL.Query().Get("token")

		if token == "" {
			// Handle token passed via basic auth (https://<token>@sourcegraph.com/foobar).
			basicAuthUsername, _, _ := r.BasicAuth()
			if basicAuthUsername != "" {
				token = basicAuthUsername
			}
		}

		if headerValue := r.Header.Get("Authorization"); headerValue != "" && token == "" {
			// Handle Authorization header
			var err error
			token, sudoUser, err = authz.ParseAuthorizationHeader(headerValue)
			if err != nil {
				if authz.IsUnrecognizedScheme(err) {
					// Ignore Authorization headers that we don't handle.
					log15.Warn("Ignoring unrecognized Authorization header.", "err", err, "value", headerValue)
					next.ServeHTTP(w, r)
					return
				}

				// Report errors on malformed Authorization headers for schemes we do handle, to
				// make it clear to the client that their request is not proceeding with their
				// supplied credentials.
				log15.Error("Invalid Authorization header.", "err", err)
				http.Error(w, "Invalid Authorization header.", http.StatusUnauthorized)
				return
			}
		}

		if token != "" {
			if !(conf.AccessTokensAllow() == conf.AccessTokensAll || conf.AccessTokensAllow() == conf.AccessTokensAdmin) {
				// if conf.AccessTokensAllow() == conf.AccessTokensNone {
				http.Error(w, "Access token authorization is disabled.", http.StatusUnauthorized)
				return
			}

			// Validate access token.
			//
			// 🚨 SECURITY: It's important we check for the correct scopes to know what this token
			// is allowed to do.
			var requiredScope string
			if sudoUser == "" {
				requiredScope = authz.ScopeUserAll
			} else {
				requiredScope = authz.ScopeSiteAdminSudo
			}
			subjectUserID, err := db.AccessTokens.Lookup(r.Context(), token, requiredScope)
			if err != nil {
				log15.Error("Invalid access token.", "token", token, "err", err)
				http.Error(w, "Invalid access token.", http.StatusUnauthorized)
				return
			}

			// Determine the actor's user ID.
			var actorUserID int32
			if sudoUser == "" {
				actorUserID = subjectUserID
			} else {
				// 🚨 SECURITY: Confirm that the sudo token's subject is still a site admin, to
				// prevent users from retaining site admin privileges after being demoted.
				if err := backend.CheckUserIsSiteAdmin(r.Context(), subjectUserID); err != nil {
					log15.Error("Sudo access token's subject is not a site admin.", "subjectUserID", subjectUserID, "err", err)
					http.Error(w, "The subject user of a sudo access token must be a site admin.", http.StatusForbidden)
					return
				}

				// Sudo to the other user if this is a sudo token. We already checked that the token has
				// the necessary scope in the Lookup call above.
				user, err := db.Users.GetByUsername(r.Context(), sudoUser)
				if err != nil {
					log15.Error("Invalid username used with sudo access token.", "sudoUser", sudoUser, "err", err)
					var message string
					if errcode.IsNotFound(err) {
						message = "Unable to sudo to nonexistent user."
					} else {
						message = "Unable to sudo to the specified user due to an unexpected error."
					}
					http.Error(w, message, http.StatusForbidden)
					return
				}
				actorUserID = user.ID
				log15.Debug("HTTP request used sudo token.", "requestURI", r.URL.RequestURI(), "tokenSubjectUserID", subjectUserID, "actorUserID", actorUserID, "actorUsername", user.Username)
			}

			r = r.WithContext(actor.WithActor(r.Context(), &actor.Actor{UID: actorUserID}))
		}

		next.ServeHTTP(w, r)
	})
}
