package api

import (
	"net/http"

	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
)

// HandleRegistry is called to handle HTTP requests for the extension registry. If there is no local
// extension registry, it returns an HTTP error response.
var HandleRegistry = func(db dbutil.DB) func(http.ResponseWriter, *http.Request) error {
	return func(w http.ResponseWriter, r *http.Request) error {
		http.Error(w, "no local extension registry exists", http.StatusNotFound)
		return nil
	}
}
