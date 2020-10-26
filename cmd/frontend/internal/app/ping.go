package app

import (
	"database/sql"
	"io"
	"net/http"

	"github.com/inconshreveable/log15"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/internal/db"
)

// latestPingHandler fetches the most recent ping data from the event log
// (if any is present) and returns it as JSON.
func latestPingHandler(w http.ResponseWriter, r *http.Request) {
	// 🚨SECURITY: Only site admins may access ping data.
	if err := backend.CheckCurrentUserIsSiteAdmin(r.Context()); err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	ping, err := db.EventLogs.LatestPing(r.Context())
	switch err {
	case sql.ErrNoRows:
		_, _ = io.WriteString(w, "{}")
	case nil:
		_, _ = io.WriteString(w, ping.Argument)
	default:
		log15.Error("pings.latest", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}
