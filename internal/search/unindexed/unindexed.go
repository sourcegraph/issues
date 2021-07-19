package unindexed

import (
	"context"
	"fmt"
	"time"

	"github.com/cockroachdb/errors"
	"golang.org/x/sync/errgroup"

	"github.com/sourcegraph/sourcegraph/internal/search/streaming"
	"github.com/sourcegraph/sourcegraph/internal/trace"

	otlog "github.com/opentracing/opentracing-go/log"

	"github.com/inconshreveable/log15"

	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/endpoint"
	"github.com/sourcegraph/sourcegraph/internal/errcode"
	"github.com/sourcegraph/sourcegraph/internal/mutablelimiter"
	"github.com/sourcegraph/sourcegraph/internal/search"
	"github.com/sourcegraph/sourcegraph/internal/search/repos"
	"github.com/sourcegraph/sourcegraph/internal/search/result"
	"github.com/sourcegraph/sourcegraph/internal/search/searcher"
	zoektutil "github.com/sourcegraph/sourcegraph/internal/search/zoekt"
	"github.com/sourcegraph/sourcegraph/internal/types"
	"github.com/sourcegraph/sourcegraph/internal/vcs/git"
)

// A global limiter on number of concurrent searcher searches.
var textSearchLimiter = mutablelimiter.New(32)

var MockSearchFilesInRepos func(args *search.TextParameters) ([]result.Match, *streaming.Stats, error)

// SearchFilesInRepos searches a set of repos for a pattern.
func SearchFilesInRepos(ctx context.Context, args *search.TextParameters, stream streaming.Sender) (err error) {
	if MockSearchFilesInRepos != nil {
		matches, mockStats, err := MockSearchFilesInRepos(args)
		stream.Send(streaming.SearchEvent{
			Results: matches,
			Stats:   mockStats.Deref(),
		})
		return err
	}

	ctx, stream, cleanup := streaming.WithLimit(ctx, stream, int(args.PatternInfo.FileMatchLimit))
	defer cleanup()

	tr, ctx := trace.New(ctx, "searchFilesInRepos", fmt.Sprintf("query: %s", args.PatternInfo.Pattern))
	defer func() {
		tr.SetErrorIfNotContext(err)
		tr.Finish()
	}()
	tr.LogFields(
		trace.Stringer("query", args.Query),
		trace.Stringer("info", args.PatternInfo),
		trace.Stringer("global_search_mode", args.Mode),
	)

	// performance: for global searches, we avoid calling NewIndexedSearchRequest
	// because zoekt will anyway have to search all its shards.
	var indexed *zoektutil.IndexedSearchRequest
	if args.Mode == search.ZoektGlobalSearch {
		indexed = &zoektutil.IndexedSearchRequest{
			Args:     args,
			Typ:      zoektutil.TextRequest,
			RepoRevs: &zoektutil.IndexedRepoRevs{},
		}
	} else {
		indexed, err = zoektutil.NewIndexedSearchRequest(ctx, args, zoektutil.TextRequest, stream)
		if err != nil {
			return err
		}
	}

	g, ctx := errgroup.WithContext(ctx)

	if args.Mode != search.SearcherOnly {
		// Run searches on indexed repositories.

		if !args.PatternInfo.IsStructuralPat {
			// Run literal and regexp searches.
			g.Go(func() error {
				return indexed.Search(ctx, stream)
			})
		} else {
			// Run structural search (fulfilled via searcher).
			g.Go(func() error {
				repos := make([]*search.RepositoryRevisions, 0, len(indexed.Repos()))
				for _, repo := range indexed.Repos() {
					repos = append(repos, repo)
				}
				return callSearcherOverRepos(ctx, args, stream, repos, true)
			})
		}
	}

	// Concurrently run searcher for all unindexed repos regardless whether text, regexp, or structural search.
	g.Go(func() error {
		return callSearcherOverRepos(ctx, args, stream, indexed.Unindexed, false)
	})

	return g.Wait()
}

// SearchFilesInRepoBatch is a convenience function around searchFilesInRepos
// which collects the results from the stream.
func SearchFilesInReposBatch(ctx context.Context, args *search.TextParameters) ([]*result.FileMatch, streaming.Stats, error) {
	matches, stats, err := streaming.CollectStream(func(stream streaming.Sender) error {
		return SearchFilesInRepos(ctx, args, stream)
	})

	fms, fmErr := matchesToFileMatches(matches)
	if fmErr != nil && err == nil {
		err = errors.Wrap(fmErr, "searchFilesInReposBatch failed to convert results")
	}
	return fms, stats, err
}

var mockSearchFilesInRepo func(ctx context.Context, repo types.RepoName, gitserverRepo api.RepoName, rev string, info *search.TextPatternInfo, fetchTimeout time.Duration) (matches []result.Match, limitHit bool, err error)

func searchFilesInRepo(ctx context.Context, searcherURLs *endpoint.Map, repo types.RepoName, gitserverRepo api.RepoName, rev string, index bool, info *search.TextPatternInfo, fetchTimeout time.Duration) ([]result.Match, bool, error) {
	if mockSearchFilesInRepo != nil {
		return mockSearchFilesInRepo(ctx, repo, gitserverRepo, rev, info, fetchTimeout)
	}

	// Do not trigger a repo-updater lookup (e.g.,
	// backend.{GitRepo,Repos.ResolveRev}) because that would slow this operation
	// down by a lot (if we're looping over many repos). This means that it'll fail if a
	// repo is not on gitserver.
	commit, err := git.ResolveRevision(ctx, gitserverRepo, rev, git.ResolveRevisionOptions{NoEnsureRevision: true})
	if err != nil {
		return nil, false, err
	}

	shouldBeSearched, err := repoShouldBeSearched(ctx, searcherURLs, info, gitserverRepo, commit, fetchTimeout)
	if err != nil {
		return nil, false, err
	}
	if !shouldBeSearched {
		return nil, false, err
	}

	var indexerEndpoints []string
	if info.IsStructuralPat {
		endpoints, err := search.Indexers().Map.Endpoints()
		for key := range endpoints {
			indexerEndpoints = append(indexerEndpoints, key)
		}
		if err != nil {
			return nil, false, err
		}
	}
	searcherMatches, limitHit, err := searcher.Search(ctx, searcherURLs, gitserverRepo, rev, commit, index, info, fetchTimeout, indexerEndpoints)
	if err != nil {
		return nil, false, err
	}

	matches := make([]result.Match, 0, len(searcherMatches))
	for _, fm := range searcherMatches {
		lineMatches := make([]*result.LineMatch, 0, len(fm.LineMatches))
		for _, lm := range fm.LineMatches {
			ranges := make([][2]int32, 0, len(lm.OffsetAndLengths))
			for _, ol := range lm.OffsetAndLengths {
				ranges = append(ranges, [2]int32{int32(ol[0]), int32(ol[1])})
			}
			lineMatches = append(lineMatches, &result.LineMatch{
				Preview:          lm.Preview,
				OffsetAndLengths: ranges,
				LineNumber:       int32(lm.LineNumber),
			})
		}

		matches = append(matches, &result.FileMatch{
			File: result.File{
				Path:     fm.Path,
				Repo:     repo,
				CommitID: commit,
				InputRev: &rev,
			},
			LineMatches: lineMatches,
			LimitHit:    fm.LimitHit,
		})
	}

	return matches, limitHit, err
}

// repoShouldBeSearched determines whether a repository should be searched in, based on whether the repository
// fits in the subset of repositories specified in the query's `repohasfile` and `-repohasfile` flags if they exist.
func repoShouldBeSearched(ctx context.Context, searcherURLs *endpoint.Map, searchPattern *search.TextPatternInfo, gitserverRepo api.RepoName, commit api.CommitID, fetchTimeout time.Duration) (shouldBeSearched bool, err error) {
	shouldBeSearched = true
	flagInQuery := len(searchPattern.FilePatternsReposMustInclude) > 0
	if flagInQuery {
		shouldBeSearched, err = repoHasFilesWithNamesMatching(ctx, searcherURLs, true, searchPattern.FilePatternsReposMustInclude, gitserverRepo, commit, fetchTimeout)
		if err != nil {
			return shouldBeSearched, err
		}
	}
	negFlagInQuery := len(searchPattern.FilePatternsReposMustExclude) > 0
	if negFlagInQuery {
		shouldBeSearched, err = repoHasFilesWithNamesMatching(ctx, searcherURLs, false, searchPattern.FilePatternsReposMustExclude, gitserverRepo, commit, fetchTimeout)
		if err != nil {
			return shouldBeSearched, err
		}
	}
	return shouldBeSearched, nil
}

// repoHasFilesWithNamesMatching searches in a repository for matches for the patterns in the `repohasfile` or `-repohasfile` flags, and returns
// whether or not the repoShouldBeSearched in or not, based on whether matches were returned.
func repoHasFilesWithNamesMatching(ctx context.Context, searcherURLs *endpoint.Map, include bool, repoHasFileFlag []string, gitserverRepo api.RepoName, commit api.CommitID, fetchTimeout time.Duration) (bool, error) {
	for _, pattern := range repoHasFileFlag {
		p := search.TextPatternInfo{IsRegExp: true, FileMatchLimit: 1, IncludePatterns: []string{pattern}, PathPatternsAreCaseSensitive: false, PatternMatchesContent: true, PatternMatchesPath: true}
		matches, _, err := searcher.Search(ctx, searcherURLs, gitserverRepo, "", commit, false, &p, fetchTimeout, []string{})
		if err != nil {
			return false, err
		}
		if include && len(matches) == 0 || !include && len(matches) > 0 {
			// repo shouldn't be searched if it does not have matches for the patterns in `repohasfile`
			// or if it has file matches for the patterns in `-repohasfile`.
			return false, nil
		}
	}

	return true, nil
}

func matchesToFileMatches(matches []result.Match) ([]*result.FileMatch, error) {
	fms := make([]*result.FileMatch, 0, len(matches))
	for _, match := range matches {
		fm, ok := match.(*result.FileMatch)
		if !ok {
			return nil, errors.Errorf("expected only file match results")
		}
		fms = append(fms, fm)
	}
	return fms, nil
}

// callSearcherOverRepos calls searcher on searcherRepos.
func callSearcherOverRepos(
	ctx context.Context,
	args *search.TextParameters,
	stream streaming.Sender,
	searcherRepos []*search.RepositoryRevisions,
	index bool,
) (err error) {
	tr, ctx := trace.New(ctx, "searcherOverRepos", fmt.Sprintf("query: %s", args.PatternInfo.Pattern))
	defer func() {
		tr.SetError(err)
		tr.Finish()
	}()

	var fetchTimeout time.Duration
	if len(searcherRepos) == 1 || args.UseFullDeadline {
		// When searching a single repo or when an explicit timeout was specified, give it the remaining deadline to fetch the archive.
		deadline, ok := ctx.Deadline()
		if ok {
			fetchTimeout = time.Until(deadline)
		} else {
			// In practice, this case should not happen because a deadline should always be set
			// but if it does happen just set a long but finite timeout.
			fetchTimeout = time.Minute
		}
	} else {
		// When searching many repos, don't wait long for any single repo to fetch.
		fetchTimeout = 500 * time.Millisecond
	}

	tr.LogFields(
		otlog.Int64("fetch_timeout_ms", fetchTimeout.Milliseconds()),
		otlog.Int64("repos_count", int64(len(searcherRepos))),
	)

	if len(searcherRepos) == 0 {
		return nil
	}

	// The number of searcher endpoints can change over time. Inform our
	// limiter of the new limit, which is a multiple of the number of
	// searchers.
	eps, err := args.SearcherURLs.Endpoints()
	if err != nil {
		return err
	}
	textSearchLimiter.SetLimit(len(eps) * 32)

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		for _, repoAllRevs := range searcherRepos {
			if len(repoAllRevs.Revs) == 0 {
				continue
			}

			revSpecs, err := repoAllRevs.ExpandedRevSpecs(ctx)
			if err != nil {
				return err
			}

			for _, rev := range revSpecs {
				limitCtx, limitDone, err := textSearchLimiter.Acquire(ctx)
				if err != nil {
					return err
				}

				// Make a new repoRev for just the operation of searching this revspec.
				repoRev := &search.RepositoryRevisions{Repo: repoAllRevs.Repo, Revs: []search.RevisionSpecifier{{RevSpec: rev}}}
				g.Go(func() error {
					ctx, done := limitCtx, limitDone
					defer done()

					matches, repoLimitHit, err := searchFilesInRepo(ctx, args.SearcherURLs, repoRev.Repo, repoRev.GitserverRepo(), repoRev.RevSpecs()[0], index, args.PatternInfo, fetchTimeout)
					if err != nil {
						tr.LogFields(otlog.String("repo", string(repoRev.Repo.Name)), otlog.Error(err), otlog.Bool("timeout", errcode.IsTimeout(err)), otlog.Bool("temporary", errcode.IsTemporary(err)))
						log15.Warn("searchFilesInRepo failed", "error", err, "repo", repoRev.Repo.Name)
					}
					// non-diff search reports timeout through err, so pass false for timedOut
					stats, err := repos.HandleRepoSearchResult(repoRev, repoLimitHit, false, err)
					stream.Send(streaming.SearchEvent{
						Results: matches,
						Stats:   stats,
					})
					return err
				})
			}
		}

		return nil
	})

	return g.Wait()
}
