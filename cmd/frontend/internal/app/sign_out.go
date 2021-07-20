package app

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/inconshreveable/log15"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/internal/session"
	"github.com/sourcegraph/sourcegraph/internal/actor"
	"github.com/sourcegraph/sourcegraph/internal/cookie"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
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

func serveSignOutHandler(db dbutil.DB) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		logSignOutEvent(r, db, database.SecurityEventNameSignOutAttempted, nil)

		// Invalidate all user sessions first
		// This way, any other signout failures should not leave a valid session
		var err error
		if err = session.InvalidateSessionCurrentUser(w, r); err != nil {
			logSignOutEvent(r, db, database.SecurityEventNameSignOutFailed, err)
			log15.Error("serveSignOutHandler", "err", err)
		}

		if err = session.SetActor(w, r, nil, 0, time.Time{}); err != nil {
			logSignOutEvent(r, db, database.SecurityEventNameSignOutFailed, err)
			log15.Error("serveSignOutHandler", "err", err)
		}

		if ssoSignOutHandler != nil {
			ssoSignOutHandler(w, r)
		}

		if err == nil {
			logSignOutEvent(r, db, database.SecurityEventNameSignOutSucceeded, nil)
		}

		http.Redirect(w, r, "/search", http.StatusSeeOther)
	}
}

// logSignOutEvent records an event into the security event log.
func logSignOutEvent(r *http.Request, db dbutil.DB, name database.SecurityEventName, err error) {
	ctx := r.Context()
	a := actor.FromContext(ctx)

	arg := struct {
		Error string `json:"error"`
	}{}
	if err != nil {
		arg.Error = err.Error()
	}

	marshalled, _ := json.Marshal(arg)

	event := &database.SecurityEvent{
		Name:      name,
		URL:       r.URL.Path,
		UserID:    uint32(a.UID),
		Argument:  marshalled,
		Source:    "BACKEND",
		Timestamp: time.Now(),
	}

	// Safe to ignore this error
	event.AnonymousUserID, _ = cookie.AnonymousUID(r)

	database.SecurityEventLogs(db).LogEvent(ctx, event)
}
