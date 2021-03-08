package app

import (
	"errors"
	"fmt"
	"net/http"
	"path"
	"strings"

	"github.com/sourcegraph/sourcegraph/internal/errcode"
)

// isGoRepoPath returns whether pkg is (likely to be) a Go stdlib
// package import path.
func isGoRepoPath(pkg string) bool {
	// If no path components have a ".", then guess that it's a Go
	// stdlib package.
	parts := strings.Split(pkg, "/")
	for _, p := range parts {
		if strings.Contains(p, ".") {
			return false
		}
	}
	return true
}

// serveGDDORefs handles requests referred from godoc.org refs links.
func serveGDDORefs(w http.ResponseWriter, r *http.Request) error {
	q := r.URL.Query()
	repo := q.Get("repo")
	pkg := q.Get("pkg")
	def := q.Get("def")

	if path.IsAbs(repo) {
		// Prevent open redirect.
		return &errcode.HTTPErr{Status: http.StatusBadRequest, Err: errors.New("repo path should not be absolute")}
	}

	// The Go standard library doesn't use package import comments (https://golang.org/cmd/go/#hdr-Import_path_checking)
	// and as such standard library packages like e.g. encoding/json are
	// unfortunately viewable at:
	//
	// 	https://godoc.org/github.com/golang/go/src/encoding/json#Marshal
	//
	// Instead of where they should be viewed:
	//
	// 	https://godoc.org/encoding/json#Marshal
	//
	// This is really a bug in godoc.org, but because it is easy for users to
	// end up on these links via Google or otherwise general confusion, etc. we
	// handle such links here. The package in this case will always start with
	// "github.com/golang/go/src/" and we simply want to remove that to end up
	// with the canonical package import path. In this case, the repo field is
	// correct. Example query to this endpoint:
	//
	// 	?def=Marshal&pkg=github.com%2Fgolang%2Fgo%2Fsrc%2Fencoding%2Fjson&repo=github.com%2Fgolang%2Fgo
	//
	pkg = strings.TrimPrefix(pkg, "github.com/golang/go/src/")

	if repo == "" && isGoRepoPath(pkg) {
		repo = "github.com/golang/go"
	}

	if repo == "" || pkg == "" || def == "" {
		return &errcode.HTTPErr{Status: http.StatusBadRequest, Err: errors.New("repo, pkg, and def must be specified in query string")}
	}

	if idx := strings.Index(repo, "@"); idx >= 0 {
		pkg += repo[idx:]
	}

	http.Redirect(w, r, fmt.Sprintf("/go/%s/-/%s", pkg, def), http.StatusMovedPermanently)
	return nil
}
