package ui

import (
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/envvar"
	"github.com/sourcegraph/sourcegraph/internal/version"
)

// serveDocumentation redirects to documentation pages on https://docs.sourcegraph.com for the current
// product version, i.e., /help/PATH -> https://docs.sourcegraph.com/@VERSION/PATH. In unreleased
// development builds (whose docs aren't necessarily available on https://docs.sourcegraph.com, it
// shows a message with instructions on how to see the docs.)
func serveDocumentation(w http.ResponseWriter, r *http.Request) {
	page := strings.TrimPrefix(r.URL.Path, "/documentation")
	versionStr := version.Version()

	// For release builds, use the version string. Otherwise, don't use any version string because:
	//
	// - For unreleased dev builds, we serve the contents from the working tree.
	// - Sourcegraph.com users probably want the latest docs on the default branch.
	var docRevPrefix string
	if !version.IsDev(versionStr) && !envvar.SourcegraphDotComMode() {
		docRevPrefix = "@v" + versionStr
	}

	// Note that the URI fragment (e.g., #some-section-in-doc) *should* be preserved by most user
	// agents even though the Location HTTP response header omits it. See
	// https://stackoverflow.com/a/2305927.
	dest := &url.URL{
		Path: path.Join("/", docRevPrefix, page),
	}
	if version.IsDev(versionStr) && !envvar.SourcegraphDotComMode() {
		dest.Scheme = "http"
		dest.Host = "localhost:5080" // local documentation server (defined in Procfile) -- CI:LOCALHOST_OK
	} else {
		dest.Scheme = "https"
		dest.Host = "docs.sourcegraph.com"

	}

	// Use temporary, not permanent, redirect, because the destination URL changes (depending on the
	// current product version).
	http.Redirect(w, r, dest.String(), http.StatusTemporaryRedirect)
}
