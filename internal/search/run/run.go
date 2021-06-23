package run

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/sourcegraph/sourcegraph/internal/errcode"
	"github.com/sourcegraph/sourcegraph/internal/gitserver"
	"github.com/sourcegraph/sourcegraph/internal/search"
	"github.com/sourcegraph/sourcegraph/internal/search/query"
	searchrepos "github.com/sourcegraph/sourcegraph/internal/search/repos"
	"github.com/sourcegraph/sourcegraph/internal/search/streaming"
	"github.com/sourcegraph/sourcegraph/internal/vcs"
	"github.com/sourcegraph/sourcegraph/schema"
)

const defaultMaxSearchResults = 30

// SearchInputs contains fields we set before kicking off search.
type SearchInputs struct {
	Plan           query.Plan            // the comprehensive query plan
	Query          query.Q               // the current basic query being evaluated, one part of query.Plan
	OriginalQuery  string                // the raw string of the original search query
	Pagination     *SearchPaginationInfo // pagination information, or nil if the request is not paginated.
	PatternType    query.SearchType
	VersionContext *string
	UserSettings   *schema.Settings

	// DefaultLimit is the default limit to use if not specified in query.
	DefaultLimit int
}

// MaxResults computes the limit for the query.
func (inputs SearchInputs) MaxResults() int {
	if inputs.Pagination != nil {
		// Paginated search requests always consume an entire result set for a
		// given repository, so we do not want any limit here. See
		// search_pagination.go for details on why this is necessary .
		return math.MaxInt32
	}

	if inputs.Query == nil {
		return 0
	}

	if count := inputs.Query.Count(); count != nil {
		return *count
	}

	if inputs.DefaultLimit != 0 {
		return inputs.DefaultLimit
	}

	return defaultMaxSearchResults
}

// Timeout computes the timeout for the query.
func (inputs SearchInputs) Timeout() time.Duration {
	// The default timeout to use for queries.
	d := 20 * time.Second

	maxTimeout := time.Duration(searchrepos.SearchLimits().MaxTimeoutSeconds) * time.Second

	if timeout := inputs.Query.Timeout(); timeout != nil {
		d = *timeout
	} else if inputs.Query.Count() != nil {
		// If `count:` is set but `timeout:` is not explicitly set, use the max timeout
		d = maxTimeout
	}

	if d > maxTimeout {
		d = maxTimeout
	}

	return d
}

// SearchPaginationInfo describes information around a paginated search
// request.
type SearchPaginationInfo struct {
	// cursor indicates where to resume searching from (see docstrings on
	// SearchCursor) or nil when requesting the first page of results.
	Cursor *SearchCursor

	// limit indicates at max how many search results to return.
	Limit int32
}

// SearchCursor represents a decoded search pagination cursor. From an API
// consumer standpoint, it is an encoded opaque string.
type SearchCursor struct {
	// RepositoryOffset indicates how many repositories (which are globally
	// sorted and ordered) to offset by.
	RepositoryOffset int32

	// ResultOffset indicates how many results within the first repository we
	// would search in to further offset by. This is so that we can paginate
	// results within e.g. a single large repository.
	ResultOffset int32

	// Finished tells if there are more results for the query or if we've
	// consumed them all.
	Finished bool
}

// handleRepoSearchResult handles the limitHit and searchErr returned by a search function,
// returning common as to reflect that new information. If searchErr is a fatal error,
// it returns a non-nil error; otherwise, if searchErr == nil or a non-fatal error, it returns a
// nil error.
func handleRepoSearchResult(repoRev *search.RepositoryRevisions, limitHit, timedOut bool, searchErr error) (_ streaming.Stats, fatalErr error) {
	var status search.RepoStatus
	if limitHit {
		status |= search.RepoStatusLimitHit
	}

	if vcs.IsRepoNotExist(searchErr) {
		if vcs.IsCloneInProgress(searchErr) {
			status |= search.RepoStatusCloning
		} else {
			status |= search.RepoStatusMissing
		}
	} else if gitserver.IsRevisionNotFound(searchErr) {
		if len(repoRev.Revs) == 0 || len(repoRev.Revs) == 1 && repoRev.Revs[0].RevSpec == "" {
			// If we didn't specify an input revision, then the repo is empty and can be ignored.
		} else {
			fatalErr = searchErr
		}
	} else if errcode.IsNotFound(searchErr) {
		status |= search.RepoStatusMissing
	} else if errcode.IsTimeout(searchErr) || errcode.IsTemporary(searchErr) || timedOut {
		status |= search.RepoStatusTimedout
	} else if searchErr != nil {
		fatalErr = searchErr
	}
	return streaming.Stats{
		Status:     search.RepoStatusSingleton(repoRev.Repo.ID, status),
		IsLimitHit: limitHit,
	}, fatalErr
}

// getRepos is a wrapper around p.Get. It returns an error if the promise
// contains an underlying type other than []*search.RepositoryRevisions.
func getRepos(ctx context.Context, p *search.Promise) ([]*search.RepositoryRevisions, error) {
	v, err := p.Get(ctx)
	if err != nil {
		return nil, err
	}
	repoRevs, ok := v.([]*search.RepositoryRevisions)
	if !ok {
		return nil, fmt.Errorf("unexpected underlying type (%T) of promise", v)
	}
	return repoRevs, nil
}

func statsDeref(s *streaming.Stats) streaming.Stats {
	if s == nil {
		return streaming.Stats{}
	}
	return *s
}
