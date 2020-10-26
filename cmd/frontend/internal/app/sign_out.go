package app

import (
	"net/http"

	"github.com/inconshreveable/log15"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/internal/session"
)

type SignOutURL struct {
	ProviderDisplayName string
	ProviderServiceType string
	URL                 string
}

var ssoSignOutHandler func(w http.ResponseWriter, r *http.Request)

// RegisterSSOSignOutHandler registers a SSO sign-out handler that takes care of cleaning up SSO
// session state, both on Sourcegraph and on the SSO provider. This function should only be called
// once from an init function.
func RegisterSSOSignOutHandler(f func(w http.ResponseWriter, r *http.Request)) {
	if ssoSignOutHandler != nil {
		panic("RegisterSSOSignOutHandler already called")
	}
	ssoSignOutHandler = f
}

func serveSignOut(w http.ResponseWriter, r *http.Request) {
	// Invalidate all user sessions first
	// This way, any other signout failures should not leave a valid session
	if err := session.InvalidateSessionCurrentUser(r); err != nil {
		log15.Error("Error in signout.", "err", err)
	}
	if err := session.SetActor(w, r, nil, 0); err != nil {
		log15.Error("Error in signout.", "err", err)
	}
	if ssoSignOutHandler != nil {
		ssoSignOutHandler(w, r)
	}

	http.Redirect(w, r, "/search", http.StatusSeeOther)
}
