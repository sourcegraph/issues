package search

import (
	"archive/tar"
	"bytes"
	"context"
	"strings"

	"github.com/google/zoekt/ignore"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/gitserver"
	"github.com/sourcegraph/sourcegraph/internal/store"
	"github.com/sourcegraph/sourcegraph/internal/vcs/git"
)

// NewFilter is a wrapper around newIgnoreMatcher.
func NewFilter(ctx context.Context, repo gitserver.Repo, commit api.CommitID) (store.FilterFunc, error) {
	ig, err := newIgnoreMatcher(ctx, repo, commit)
	if err != nil {
		return nil, err
	}
	return func(header *tar.Header) bool {
		return ig.Match(header.Name)
	}, nil
}

// newIgnoreMatcher calls gitserver to retrieve the ignore-file.
// If the file doesn't exist we return an empty ignore.Matcher.
func newIgnoreMatcher(ctx context.Context, repo gitserver.Repo, commit api.CommitID) (*ignore.Matcher, error) {
	ignoreFile, err := git.ReadFile(ctx, repo, commit, ignore.IgnoreFile, 0)
	if err != nil {
		if strings.Contains(err.Error(), "file does not exist") {
			return &ignore.Matcher{}, nil
		}
		return nil, err
	}
	return ignore.ParseIgnoreFile(bytes.NewReader(ignoreFile))
}
