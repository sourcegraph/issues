package middleware

import (
	"html/template"
	"log"
	"net/http"
	"path"
	"strings"

	"github.com/cockroachdb/errors"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/globals"
	"github.com/sourcegraph/sourcegraph/internal/trace"
)

// goImportMetaTag represents a go-import meta tag.
type goImportMetaTag struct {
	// ImportPrefix is the import path corresponding to the repository root.
	// It must be a prefix or an exact match of the package being fetched with "go get".
	// If it's not an exact match, another http request is made at the prefix to verify
	// the <meta> tags match.
	ImportPrefix string

	// VCS is one of "git", "hg", "svn", etc.
	VCS string

	// RepoRoot is the root of the version control system containing a scheme and
	// not containing a .vcs qualifier.
	RepoRoot string
}

// goImportMetaTagTemplate is an HTML template for rendering a blank page with a go-import meta tag.
var goImportMetaTagTemplate = template.Must(template.New("").Parse(`<html><head><meta name="go-import" content="{{.ImportPrefix}} {{.VCS}} {{.RepoRoot}}"></head><body></body></html>`))

// SourcegraphComGoGetHandler is middleware for serving go-import meta tags for requests with ?go-get=1 query
// on sourcegraph.com.
//
// It implements the following mapping:
//
// 1. If the username (first path element) is "sourcegraph", consider it to be a vanity
//    import path pointing to github.com/sourcegraph/<repo> as the clone URL.
// 2. All other requests are served with 404 Not Found.
//
// 🚨 SECURITY: This handler is served to all clients, even on private servers to clients who have
// not authenticated. It must not reveal any sensitive information.
func SourcegraphComGoGetHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Query().Get("go-get") != "1" {
			next.ServeHTTP(w, req)
			return
		}

		trace.SetRouteName(req, "middleware.go-get")
		if !strings.HasPrefix(req.URL.Path, "/") {
			err := errors.Errorf("req.URL.Path doesn't have a leading /: %q", req.URL.Path)
			log.Println(err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Handle "go get sourcegraph.com/{sourcegraph,sqs}/*" for all non-hosted repositories.
		// It's a vanity import path that maps to "github.com/{sourcegraph,sqs}/*" clone URLs.
		pathElements := strings.Split(req.URL.Path[1:], "/")
		if len(pathElements) >= 2 && (pathElements[0] == "sourcegraph" || pathElements[0] == "sqs") {
			host := globals.ExternalURL().Host

			user := pathElements[0]
			repo := pathElements[1]

			err := goImportMetaTagTemplate.Execute(w, goImportMetaTag{
				ImportPrefix: path.Join(host, user, repo),
				VCS:          "git",
				RepoRoot:     "https://github.com/" + user + "/" + repo,
			})
			if err != nil {
				log.Println("goImportMetaTagTemplate.Execute:", err)
			}
			return
		}

		// If we get here, there isn't a Go package for this request.
		http.Error(w, "no such repository", http.StatusNotFound)
	})
}
