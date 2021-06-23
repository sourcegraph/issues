package graphqlbackend

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"
	"github.com/inconshreveable/log15"
	otlog "github.com/opentracing/opentracing-go/log"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend/graphqlutil"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
	"github.com/sourcegraph/sourcegraph/internal/search"
	"github.com/sourcegraph/sourcegraph/internal/search/query"
	"github.com/sourcegraph/sourcegraph/internal/search/result"
	searchresult "github.com/sourcegraph/sourcegraph/internal/search/result"
	"github.com/sourcegraph/sourcegraph/internal/search/run"
	"github.com/sourcegraph/sourcegraph/internal/search/streaming"
	"github.com/sourcegraph/sourcegraph/internal/trace"
	"github.com/sourcegraph/sourcegraph/internal/types"
)

const searchCursorKind = "SearchCursor"

// marshalSearchCursor marshals a search pagination cursor.
func marshalSearchCursor(c *run.SearchCursor) string {
	return string(relay.MarshalID(searchCursorKind, c))
}

// unmarshalSearchCursor unmarshals a search pagination cursor.
func unmarshalSearchCursor(cursor *string) (*run.SearchCursor, error) {
	if cursor == nil {
		return nil, nil
	}
	if kind := relay.UnmarshalKind(graphql.ID(*cursor)); kind != searchCursorKind {
		return nil, fmt.Errorf("cannot unmarshal search cursor type: %q", kind)
	}
	var spec *run.SearchCursor
	if err := relay.UnmarshalSpec(graphql.ID(*cursor), &spec); err != nil {
		return nil, err
	}
	return spec, nil
}

func (r *SearchResultsResolver) PageInfo() *graphqlutil.PageInfo {
	if r.Cursor == nil || r.Cursor.Finished {
		return graphqlutil.HasNextPage(false)
	}
	return graphqlutil.NextPageCursor(marshalSearchCursor(r.Cursor))
}

// paginatedResults handles serving paginated search queries. It's logic does
// not live alongside the non-paginated doResults because:
//
// 1. It would introduce many `if r.pagination != nil` conditionals which would
//    make that code harder to reason about.
// 2. That method is already very large and brittle, common logic can be
//    refactored out instead.
// 3. The way that method operates (mixing in search result types depending on
//    a timeout, searcing result types in parallel) is fundamentally incompatible
//    with the absolute ordering we do here for pagination.
//
func (r *searchResolver) paginatedResults(ctx context.Context) (result *SearchResults, err error) {
	if r.Pagination == nil {
		panic("never here: this method should never be called in this state")
	}

	tr, ctx := trace.New(ctx, "graphql.SearchResults.paginatedResults", r.rawQuery())
	if r.Pagination.Cursor != nil {
		tr.LogFields(
			otlog.Int("Cursor.RepositoryOffset", int(r.Pagination.Cursor.RepositoryOffset)),
			otlog.Int("Cursor.ResultOffset", int(r.Pagination.Cursor.ResultOffset)),
			otlog.Bool("Cursor.Finished", r.Pagination.Cursor.Finished),
		)
		log15.Info("paginated search continue request",
			"query", fmt.Sprintf("%q", r.rawQuery()),
			"RepositoryOffset", int(r.Pagination.Cursor.RepositoryOffset),
			"ResultOffset", int(r.Pagination.Cursor.ResultOffset),
			"Finished", r.Pagination.Cursor.Finished,
		)
	} else {
		tr.LogFields(otlog.String("Cursor", "nil"))
		log15.Info("paginated search begin request", "query", fmt.Sprintf("%q", r.rawQuery()))
	}
	tr.LogFields(otlog.Int("Limit", int(r.Pagination.Limit)))
	defer func() {
		tr.SetError(err)
		tr.Finish()
	}()

	// All paginated search requests must complete within this timeframe.
	//
	// In an ideal world, all paginated search requests would complete in under
	// just a few seconds and we could safely have e.g. a 10s hard limit in
	// place here. But in practice, some queries can take longer:
	//
	// 	`repo:^github\.com/kubernetes/kubernetes$ .` (so many results it often can't complete in under 1m)
	// 	`repo:^github\.com/torvalds/linux$ type:diff error index:no ` (times out after 1m)
	//
	// These cases would make a UI paginated search experience bad, but from an
	// API use case you are very willing to wait as long as you get accurate
	// results so for now we use a 2m hard timeout (based on the non-paginated
	// search upper bound, it should not be increased further).
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	resolved, err := r.determineRepos(ctx, tr)
	if err != nil {
		if alert, err := errorToAlert(err); alert != nil {
			return alert.wrapResults(), err
		}
		return nil, err
	}

	q, err := query.ToBasicQuery(r.Query)
	if err != nil {
		return nil, err
	}
	p := search.ToTextPatternInfo(q, search.Pagination, query.Identity)
	args := search.TextParameters{
		PatternInfo:     p,
		RepoPromise:     (&search.Promise{}).Resolve(resolved.RepoRevs),
		Query:           r.Query,
		UseFullDeadline: false,
		Zoekt:           r.zoekt,
		SearcherURLs:    r.searcherURLs,
	}
	if err := args.PatternInfo.Validate(); err != nil {
		return nil, &badRequestError{err}
	}

	resultTypes := r.determineResultTypes(args, searchresult.TypeEmpty)
	tr.LazyPrintf("resultTypes: %v", resultTypes)

	if resultTypes != searchresult.TypeFile {
		return nil, fmt.Errorf("experimental paginated search currently only supports 'file' (text match) result types. Found %q", resultTypes)
	}

	// Since we're searching a subset of the repositories this query would
	// search overall, we must sort the repositories deterministically.
	for _, repoRev := range resolved.RepoRevs {
		sort.Slice(repoRev.Revs, func(i, j int) bool {
			return repoRev.Revs[i].Less(repoRev.Revs[j])
		})
	}
	sort.Slice(resolved.RepoRevs, func(i, j int) bool {
		return repoIsLess(resolved.RepoRevs[i].Repo, resolved.RepoRevs[j].Repo)
	})

	common := streaming.Stats{}
	cursor, results, fileCommon, err := paginatedSearchFilesInRepos(ctx, r.db, &args, r.Pagination)
	if err != nil {
		return nil, err
	}
	common.Update(fileCommon)

	tr.LazyPrintf("results=%d %s", len(results), &common)

	// Alert is a potential alert shown to the user.
	var alert *searchAlert

	if len(resolved.MissingRepoRevs) > 0 {
		alert = alertForMissingRepoRevs(resolved.MissingRepoRevs)
	}

	log15.Info("next cursor for paginated search request",
		"query", fmt.Sprintf("%q", r.rawQuery()),
		"RepositoryOffset", int(cursor.RepositoryOffset),
		"ResultOffset", int(cursor.ResultOffset),
		"Finished", cursor.Finished,
	)

	return &SearchResults{
		Matches: results,
		Stats:   common,
		Cursor:  cursor,
		Alert:   alert,
	}, nil
}

// repoIsLess sorts repositories first by name then by ID, suitable for use
// with sort.Slice.
func repoIsLess(i, j types.RepoName) bool {
	if i.Name != j.Name {
		return i.Name < j.Name
	}
	return i.ID < j.ID
}

// paginatedSearchFilesInRepos implements result-level pagination by calling
// searchFilesInRepos to search over subsets (batches) of the total list of
// repositories that may have results for this request (args.Repos). It does
// this by picking some tradeoffs to balance some conflicting facts:
//
// 1. Paginated text searches must currently ask Zoekt AND non-indexed search
//    to produce the entire result set for a repository. This is like querying
//    for `repo:^exact-repo$ count:1000000` in a non-paginated query, and is
//    more costly and slower than the default `count:30` used in non-paginated
//    requests (search for FileMatchLimit) which allows Zoekt/non-indexed
//    search to stop searching after finding enough results. Another reason for
//    needing to produce the entire result set for a repository is because
//    Zoekt does not today produce a stable order of results.
//
// 2. With NITH (needle-in-the-haystack) queries, if we don't search enough
//    repositories in parallel we would substantially harm the performance of
//    these queries. For example, if we were to search 100 repositories at a
//    time and there were 1000 repositories to search and only the last 100
//    repositories had results for you, you need to wait for the first 9
//    batched searches to complete making your results 10x slower to fetch on
//    top of the penalty we incur from the larger `count:` mentioned in point
//    2 above (in the worst case scenario).
//
func paginatedSearchFilesInRepos(ctx context.Context, db dbutil.DB, args *search.TextParameters, pagination *run.SearchPaginationInfo) (*run.SearchCursor, []result.Match, *streaming.Stats, error) {
	repos, err := getRepos(ctx, args.RepoPromise)
	if err != nil {
		return nil, nil, nil, err
	}

	plan := &repoPaginationPlan{
		pagination:          pagination,
		repositories:        repos,
		searchBucketDivisor: 8,
		searchBucketMin:     10,
		searchBucketMax:     1000,
	}
	return plan.execute(ctx, database.Repos(db), func(batch []*search.RepositoryRevisions) ([]result.Match, *streaming.Stats, error) {
		batchArgs := *args
		batchArgs.RepoPromise = (&search.Promise{}).Resolve(batch)
		fileMatches, fileCommon, err := run.SearchFilesInReposBatch(ctx, &batchArgs)
		// Timeouts are reported through Stats so don't report an error for them
		if err != nil && !(err == context.DeadlineExceeded || err == context.Canceled) {
			return nil, nil, err
		}

		matches := fileMatchesToMatches(fileMatches)
		sort.Sort(result.Matches(matches))
		return matches, &fileCommon, nil
	})
}

func fileMatchesToMatches(fms []*result.FileMatch) []result.Match {
	matches := make([]result.Match, 0, len(fms))
	for _, fm := range fms {
		newFm := fm
		matches = append(matches, newFm)
	}
	return matches
}

// repoPaginationPlan describes a plan for executing a search function that
// searches only over a set of repositories (i.e. the search function offers no
// pagination or result-level pagination capabilities) to provide result-level
// pagination. That is, if you have a function which can provide a complete
// list of results for a given repository, this planner can be used to
// implement result-level pagination on top of that function.
//
// It does this by searching over a globally-sorted list of repositories in
// batches.
type repoPaginationPlan struct {
	// pagination is the pagination request we're trying to fulfill.
	pagination *run.SearchPaginationInfo

	// repositories is the exhaustive and complete list of sorted repositories
	// to be searched over multiple requests.
	repositories []*search.RepositoryRevisions

	// parameters for controlling the size of batches that the executor is
	// called to search. The final batch size is calculated as:
	//
	// 	batchSize = numTotalReposOnSourcegraph() / searchBucketDivisor
	//
	// With the additional constraint that it must be at least min and no
	// larger than max.
	searchBucketDivisor              int
	searchBucketMin, searchBucketMax int

	mockNumTotalRepos func() int
}

// executor is a function which searches a batch of repositories.
//
// A non-nil Stats must always be returned, even if an error is
// returned.
type executor func(batch []*search.RepositoryRevisions) ([]result.Match, *streaming.Stats, error)

// repoOfMatch is a helper function to resolve the repo associated with a result type.
func repoOfMatch(match result.Match) string {
	switch r := match.(type) {
	case *result.RepoMatch:
		return string(r.Name)
	case *result.FileMatch:
		return string(r.Repo.Name)
	case *result.CommitMatch:
		// Pagination does not support commit searches at the
		// moment. Ideally we want to return the repo associated
		// with a commit, but the commit result type does not
		// clearly expose the repo it is associated with.
		return "~" // lexicographically last.
	}
	// Unreachable.
	panic("unreachable: repoOfResult expects RepositoryResolver, FileMatchResolver, or CommitSearchResultResolver")
}

// execute executes the repository pagination plan by invoking the executor to
// search batches of repositories.
//
// If the executor returns any error, the search will be cancelled and the error
// returned.
func (p *repoPaginationPlan) execute(ctx context.Context, repoStore *database.RepoStore, exec executor) (c *run.SearchCursor, results []result.Match, common *streaming.Stats, err error) {
	// Determine how large the batches of repositories we will search over will be.
	var totalRepos int
	if p.mockNumTotalRepos != nil {
		totalRepos = p.mockNumTotalRepos()
	} else {
		totalRepos = numTotalRepos.get(ctx, repoStore)
	}
	batchSize := clamp(totalRepos/p.searchBucketDivisor, p.searchBucketMin, p.searchBucketMax)

	// Determine where in the repositories list we will begin searching.
	var (
		repos                          = p.repositories
		repositoryOffset, resultOffset int
	)
	if cursor := p.pagination.Cursor; cursor != nil {
		resultOffset = int(cursor.ResultOffset)

		// Clamping is required here because the repositories the user has
		// access to could have changed if e.g. permissions for that user
		// were updated OR if this cursor was generated by a user with
		// different permissions.
		repositoryOffset = clamp(int(cursor.RepositoryOffset), 0, len(repos)-1)
		repos = repos[repositoryOffset:]
	}

	// Search backends don't populate Stats.repos, the
	// repository searcher does. We need to do that here.
	commonRepos := make(map[api.RepoID]types.RepoName, len(repos))
	for _, r := range repos {
		commonRepos[r.Repo.ID] = r.Repo
	}

	// Search over the repos list in batches.
	common = &streaming.Stats{
		Repos: commonRepos,
	}
	for start := 0; start <= len(repos); start += batchSize {
		if start > len(repos) {
			break
		}

		batch := repos[start:clamp(start+batchSize, 0, len(repos))]
		batchResults, batchCommon, err := exec(batch)
		if batchCommon == nil {
			panic("never here: repoPaginationPlan.executor illegally returned nil Stats structure")
		}
		if err != nil {
			return nil, nil, nil, err
		}

		// Accumulate the results and stop if we have enough for the user.
		results = append(results, batchResults...)
		common.Update(batchCommon)

		if len(results) >= resultOffset+int(p.pagination.Limit) {
			break
		}
	}
	// If we found more results than the user wanted, discard the remaining
	// ones.
	sliced := sliceSearchResults(results, common, resultOffset, int(p.pagination.Limit))
	nextCursor := &run.SearchCursor{ResultOffset: sliced.resultOffset}

	if len(sliced.results) > 0 {
		// First, identify what repository corresponds to the last result.
		lastRepoConsumedName := repoOfMatch(sliced.results[len(sliced.results)-1])
		var lastRepoConsumed types.RepoName
		for _, repo := range p.repositories {
			if string(repo.Repo.Name) == lastRepoConsumedName {
				lastRepoConsumed = repo.Repo
			}
		}

		// If any repositories were cloning or missing, then we need to skip
		// over them. We aren't sure though what position those errored
		// repositories have relative to lastRepoConsumed, though, so we figure
		// that out now. For example, a cloning repository could be last or
		// first in the results and we need to know the position for the cursor
		// RepositoryOffset.
		potentialLastRepos := []types.RepoName{lastRepoConsumed}
		sliced.common.Status.Filter(search.RepoStatusCloning|search.RepoStatusMissing, func(id api.RepoID) {
			potentialLastRepos = append(potentialLastRepos, sliced.common.Repos[id])
		})
		sort.Slice(potentialLastRepos, func(i, j int) bool {
			return repoIsLess(potentialLastRepos[i], potentialLastRepos[j])
		})
		lastRepoConsumed = potentialLastRepos[len(potentialLastRepos)-1]

		// Find the last repo's global offset.
		for globalOffset, repo := range p.repositories {
			if repo.Repo.Name == lastRepoConsumed.Name {
				nextCursor.RepositoryOffset = int32(globalOffset)
			}
		}
	}

	lastRepoConsumedPartially := sliced.resultOffset != 0
	if !lastRepoConsumedPartially {
		nextCursor.RepositoryOffset++
	}
	nextCursor.Finished = len(sliced.results) == 0 || !sliced.limitHit && int(nextCursor.RepositoryOffset) == len(p.repositories) // Finished if we searched the last repository
	return nextCursor, sliced.results, sliced.common, nil
}

type slicedSearchResults struct {
	// results is the new results, sliced.
	results []result.Match

	// common is the new common results structure, updated to reflect the sliced results only.
	common *streaming.Stats

	// resultOffset indicates where the search would continue within the last
	// repository whose results were consumed. For example:
	//
	// 	limit := 5
	// 	results := [a1, a2, a3, b1, b2, b3, c1, c2, c3]
	// 	sliceSearchResults(results, ..., limit).resultOffset = 2 // in repository B, resume at result offset 2 (b3)
	//
	resultOffset int32

	// limitHit indicates if the limit was hit and results were truncated.
	limitHit bool
}

// sliceSearchResults effectively slices results[offset:offset+limit] and
// returns an updated Stats structure to reflect that, as well as
// information about the slicing that was performed.
func sliceSearchResults(results []result.Match, common *streaming.Stats, offset, limit int) (final slicedSearchResults) {
	firstRepo := ""
	if len(results[:offset]) > 0 {
		firstRepo = repoOfMatch(results[offset])
	}
	// First we handle the case of having few enough results that we do not
	// need to slice anything.
	if len(results[offset:]) <= limit {
		results = results[offset:]
		final.results = results
		if len(final.results) > 0 {
			lastResultRepo := repoOfMatch(final.results[len(final.results)-1])
			final.common = sliceSearchResultsCommon(common, firstRepo, lastResultRepo)
		} else {
			final.common = sliceSearchResultsCommon(common, firstRepo, "")
		}
		return
	}
	final.limitHit = true
	originalResults := results
	results = results[offset:]

	// Break results into repositories because for each result we need to add
	// the respective repository to the new common structure.
	reposByName := map[string]types.RepoName{}
	for _, r := range common.Repos {
		reposByName[string(r.Name)] = r
	}
	resultsByRepo := map[types.RepoName][]result.Match{}
	for _, r := range results[:limit] {
		repo := reposByName[repoOfMatch(r)]
		resultsByRepo[repo] = append(resultsByRepo[repo], r)
	}

	// Create the relative cursor.
	//
	// Above we may have sliced the results sorted by repo like so:
	//
	// 	results := [a1, a2, a3, b1, b2, b3, c1, c2, c3]
	// 	results = results[offset:offset+limit] // [a2, a3, b1, b2]
	//
	// Since it is within the boundary of B's results, the next paginated
	// request should use a Cursor.ResultOffset == 2 to indicate we should
	// resume fetching results starting at b3.
	var lastResultRepo string
	for _, r := range originalResults[:offset+limit] {
		repo := repoOfMatch(r)
		if repo != lastResultRepo {
			final.resultOffset = 0
		} else {
			final.resultOffset++
		}
		lastResultRepo = repo
	}
	nextRepo := repoOfMatch(results[limit])
	if nextRepo != lastResultRepo {
		final.resultOffset = 0
	} else {
		final.resultOffset++
	}

	// Construct the new Stats structure for just the results
	// we're returning.
	seenRepos := map[string]struct{}{}
	finalResults := make([]result.Match, 0, limit)
	for _, r := range results[:limit] {
		repoName := repoOfMatch(r)
		if _, ok := seenRepos[repoName]; ok {
			continue
		}
		seenRepos[repoName] = struct{}{}

		repo := reposByName[repoName]
		results := resultsByRepo[repo]

		// Include the results and copy over metadata from the common structure.
		finalResults = append(finalResults, results...)
	}
	final.common = sliceSearchResultsCommon(common, firstRepo, lastResultRepo)
	final.results = finalResults
	return
}

func sliceSearchResultsCommon(common *streaming.Stats, firstResultRepo, lastResultRepo string) *streaming.Stats {
	if common.Status.Any(search.RepoStatusLimitHit) {
		panic("never here: partial results should never be present in paginated search")
	}
	final := &streaming.Stats{
		IsLimitHit:         false, // irrelevant in paginated search
		IsIndexUnavailable: common.IsIndexUnavailable,
		Repos:              make(map[api.RepoID]types.RepoName),
	}

	for _, r := range common.Repos {
		if lastResultRepo == "" || string(r.Name) > lastResultRepo {
			continue
		}
		if firstResultRepo != "" && string(r.Name) < firstResultRepo {
			continue
		}
		final.Repos[r.ID] = r
	}

	common.Status.Iterate(func(id api.RepoID, status search.RepoStatus) {
		if _, ok := final.Repos[id]; ok {
			final.Status.Update(id, status)
		}
	})

	final.ExcludedForks = final.ExcludedForks + common.ExcludedForks
	final.ExcludedArchived = final.ExcludedArchived + common.ExcludedArchived

	return final
}

// clamp clamps x into the range of [min, max].
func clamp(x, min, max int) int {
	if x < min {
		return min
	}
	if x > max {
		return max
	}
	return x
}

// Since we will need to know the number of total repos on Sourcegraph for
// every paginated search request, but the exact number doesn't matter, we
// cache the result for a minute to avoid executing many DB count operations.
type numTotalReposCache struct {
	sync.RWMutex
	lastUpdate time.Time
	count      int
}

func (n *numTotalReposCache) get(ctx context.Context, repoStore *database.RepoStore) int {
	n.RLock()
	if !n.lastUpdate.IsZero() && time.Since(n.lastUpdate) < 1*time.Minute {
		defer n.RUnlock()
		return n.count
	}
	n.RUnlock()

	n.Lock()
	newCount, err := repoStore.Count(ctx, database.ReposListOptions{})
	if err != nil {
		defer n.Unlock()
		log15.Error("failed to determine numTotalRepos", "error", err)
		return n.count
	}
	n.count = newCount
	n.Unlock()
	return newCount
}

var numTotalRepos = &numTotalReposCache{}
