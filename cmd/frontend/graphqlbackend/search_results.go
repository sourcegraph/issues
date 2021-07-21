package graphqlbackend

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"path"
	"regexp"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/honeycombio/libhoney-go"
	"github.com/inconshreveable/log15"
	"github.com/neelance/parallel"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	otlog "github.com/opentracing/opentracing-go/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/envvar"
	searchlogs "github.com/sourcegraph/sourcegraph/cmd/frontend/internal/search/logs"
	"github.com/sourcegraph/sourcegraph/internal/actor"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
	"github.com/sourcegraph/sourcegraph/internal/featureflag"
	"github.com/sourcegraph/sourcegraph/internal/goroutine"
	"github.com/sourcegraph/sourcegraph/internal/honey"
	"github.com/sourcegraph/sourcegraph/internal/rcache"
	"github.com/sourcegraph/sourcegraph/internal/search"
	"github.com/sourcegraph/sourcegraph/internal/search/filter"
	"github.com/sourcegraph/sourcegraph/internal/search/query"
	searchrepos "github.com/sourcegraph/sourcegraph/internal/search/repos"
	"github.com/sourcegraph/sourcegraph/internal/search/result"
	"github.com/sourcegraph/sourcegraph/internal/search/run"
	"github.com/sourcegraph/sourcegraph/internal/search/searchcontexts"
	"github.com/sourcegraph/sourcegraph/internal/search/streaming"
	"github.com/sourcegraph/sourcegraph/internal/trace"
	"github.com/sourcegraph/sourcegraph/internal/trace/ot"
	"github.com/sourcegraph/sourcegraph/internal/types"
	"github.com/sourcegraph/sourcegraph/internal/usagestats"
	"github.com/sourcegraph/sourcegraph/internal/vcs/git"
	"github.com/sourcegraph/sourcegraph/schema"
)

func (c *SearchResultsResolver) LimitHit() bool {
	return c.Stats.IsLimitHit || (c.limit > 0 && len(c.Matches) > c.limit)
}

func (c *SearchResultsResolver) Repositories() []*RepositoryResolver {
	repos := c.Stats.Repos
	resolvers := make([]*RepositoryResolver, 0, len(repos))
	for _, r := range repos {
		resolvers = append(resolvers, NewRepositoryResolver(c.db, r.ToRepo()))
	}
	sort.Slice(resolvers, func(a, b int) bool {
		return resolvers[a].ID() < resolvers[b].ID()
	})
	return resolvers
}

func (c *SearchResultsResolver) RepositoriesCount() int32 {
	return int32(len(c.Stats.Repos))
}

func (c *SearchResultsResolver) repositoryResolvers(mask search.RepoStatus) []*RepositoryResolver {
	var resolvers []*RepositoryResolver
	c.Stats.Status.Filter(mask, func(id api.RepoID) {
		if r, ok := c.Stats.Repos[id]; ok {
			resolvers = append(resolvers, NewRepositoryResolver(c.db, r.ToRepo()))
		}
	})
	sort.Slice(resolvers, func(a, b int) bool {
		return resolvers[a].ID() < resolvers[b].ID()
	})
	return resolvers
}

func (c *SearchResultsResolver) Cloning() []*RepositoryResolver {
	return c.repositoryResolvers(search.RepoStatusCloning)
}

func (c *SearchResultsResolver) Missing() []*RepositoryResolver {
	return c.repositoryResolvers(search.RepoStatusMissing)
}

func (c *SearchResultsResolver) Timedout() []*RepositoryResolver {
	return c.repositoryResolvers(search.RepoStatusTimedout)
}

func (c *SearchResultsResolver) IndexUnavailable() bool {
	return c.Stats.IsIndexUnavailable
}

// SearchResultsResolver is a resolver for the GraphQL type `SearchResults`
type SearchResultsResolver struct {
	db dbutil.DB
	*SearchResults

	// limit is the maximum number of SearchResults to send back to the user.
	limit int

	// The time it took to compute all results.
	elapsed time.Duration

	// cache for user settings. Ideally this should be set just once in the code path
	// by an upstream resolver
	UserSettings *schema.Settings
}

type SearchResults struct {
	Matches []result.Match
	Stats   streaming.Stats
	Alert   *searchAlert
}

// Results are the results found by the search. It respects the limits set. To
// access all results directly access the SearchResults field.
func (sr *SearchResultsResolver) Results() []SearchResultResolver {
	limited := sr.Matches
	if sr.limit > 0 && sr.limit < len(sr.Matches) {
		limited = sr.Matches[:sr.limit]
	}

	return matchesToResolvers(sr.db, limited)
}

func matchesToResolvers(db dbutil.DB, matches []result.Match) []SearchResultResolver {
	type repoKey struct {
		Name types.RepoName
		Rev  string
	}
	repoResolvers := make(map[repoKey]*RepositoryResolver, 10)
	getRepoResolver := func(repoName types.RepoName, rev string) *RepositoryResolver {
		if existing, ok := repoResolvers[repoKey{repoName, rev}]; ok {
			return existing
		}
		resolver := NewRepositoryResolver(db, repoName.ToRepo())
		resolver.RepoMatch.Rev = rev
		repoResolvers[repoKey{repoName, rev}] = resolver
		return resolver
	}

	resolvers := make([]SearchResultResolver, 0, len(matches))
	for _, match := range matches {
		switch v := match.(type) {
		case *result.FileMatch:
			resolvers = append(resolvers, &FileMatchResolver{
				db:           db,
				FileMatch:    *v,
				RepoResolver: getRepoResolver(v.Repo, ""),
			})
		case *result.RepoMatch:
			resolvers = append(resolvers, getRepoResolver(v.RepoName(), v.Rev))
		case *result.CommitMatch:
			resolvers = append(resolvers, &CommitSearchResultResolver{
				db:          db,
				CommitMatch: *v,
			})
		}
	}
	return resolvers
}

func (sr *SearchResultsResolver) MatchCount() int32 {
	var totalResults int
	for _, result := range sr.Matches {
		totalResults += result.ResultCount()
	}
	return int32(totalResults)
}

// Deprecated. Prefer MatchCount.
func (sr *SearchResultsResolver) ResultCount() int32 { return sr.MatchCount() }

func (sr *SearchResultsResolver) ApproximateResultCount() string {
	count := sr.MatchCount()
	if sr.LimitHit() || sr.Stats.Status.Any(search.RepoStatusCloning|search.RepoStatusTimedout) {
		return fmt.Sprintf("%d+", count)
	}
	return strconv.Itoa(int(count))
}

func (sr *SearchResultsResolver) Alert() *searchAlert { return sr.SearchResults.Alert }

func (sr *SearchResultsResolver) ElapsedMilliseconds() int32 {
	return int32(sr.elapsed.Milliseconds())
}

func (sr *SearchResultsResolver) DynamicFilters(ctx context.Context) []*searchFilterResolver {
	tr, ctx := trace.New(ctx, "DynamicFilters", "", trace.Tag{Key: "resolver", Value: "SearchResultsResolver"})
	defer func() {
		tr.Finish()
	}()

	globbing := false
	// For search, sr.userSettings is set in (r *searchResolver) Results(ctx
	// context.Context). However we might regress on that or call DynamicFilters from
	// other code paths. Hence we fallback to accessing the user settings directly.
	if sr.UserSettings != nil {
		globbing = getBoolPtr(sr.UserSettings.SearchGlobbing, false)
	} else {
		settings, err := decodedViewerFinalSettings(ctx, sr.db)
		if err != nil {
			log15.Warn("DynamicFilters: could not get user settings from database")
		} else {
			globbing = getBoolPtr(settings.SearchGlobbing, false)
		}
	}
	tr.LogFields(otlog.Bool("globbing", globbing))

	filters := streaming.SearchFilters{
		Globbing: globbing,
	}
	filters.Update(streaming.SearchEvent{
		Results: sr.Matches,
		Stats:   sr.Stats,
	})

	var resolvers []*searchFilterResolver
	for _, f := range filters.Compute() {
		resolvers = append(resolvers, &searchFilterResolver{filter: *f})
	}
	return resolvers
}

type searchFilterResolver struct {
	filter streaming.Filter
}

func (sf *searchFilterResolver) Value() string {
	return sf.filter.Value
}

func (sf *searchFilterResolver) Label() string {
	return sf.filter.Label
}

func (sf *searchFilterResolver) Count() int32 {
	return int32(sf.filter.Count)
}

func (sf *searchFilterResolver) LimitHit() bool {
	return sf.filter.IsLimitHit
}

func (sf *searchFilterResolver) Kind() string {
	return sf.filter.Kind
}

// blameFileMatch blames the specified file match to produce the time at which
// the first line match inside of it was authored.
func (sr *SearchResultsResolver) blameFileMatch(ctx context.Context, fm *result.FileMatch) (t time.Time, err error) {
	span, ctx := ot.StartSpanFromContext(ctx, "blameFileMatch")
	defer func() {
		if err != nil {
			ext.Error.Set(span, true)
			span.SetTag("err", err.Error())
		}
		span.Finish()
	}()

	// Blame the first line match.
	if len(fm.LineMatches) == 0 {
		// No line match
		return time.Time{}, nil
	}
	lm := fm.LineMatches[0]
	hunks, err := git.BlameFile(ctx, fm.Repo.Name, fm.Path, &git.BlameOptions{
		NewestCommit: fm.CommitID,
		StartLine:    int(lm.LineNumber),
		EndLine:      int(lm.LineNumber),
	})
	if err != nil {
		return time.Time{}, err
	}

	return hunks[0].Author.Date, nil
}

func (sr *SearchResultsResolver) Sparkline(ctx context.Context) (sparkline []int32, err error) {
	var (
		days     = 30                 // number of days the sparkline represents
		maxBlame = 100                // maximum number of file results to blame for date/time information.
		run      = parallel.NewRun(8) // number of concurrent blame ops
	)

	var (
		sparklineMu sync.Mutex
		blameOps    = 0
	)
	sparkline = make([]int32, days)
	addPoint := func(t time.Time) {
		// Check if the author date of the search result is inside of our sparkline
		// timerange.
		now := time.Now()
		if t.Before(now.Add(-time.Duration(len(sparkline)) * 24 * time.Hour)) {
			// Outside the range of the sparkline.
			return
		}
		sparklineMu.Lock()
		defer sparklineMu.Unlock()
		for n := range sparkline {
			d1 := now.Add(-time.Duration(n) * 24 * time.Hour)
			d2 := now.Add(-time.Duration(n-1) * 24 * time.Hour)
			if t.After(d1) && t.Before(d2) {
				sparkline[n]++ // on the nth day
			}
		}
	}

	// Consider all of our search results as a potential data point in our
	// sparkline.
loop:
	for _, r := range sr.Matches {
		r := r // shadow so it doesn't change in the goroutine
		switch m := r.(type) {
		case *result.RepoMatch:
			// We don't care about repo results here.
			continue
		case *result.CommitMatch:
			// Diff searches are cheap, because we implicitly have author date info.
			addPoint(m.Commit.Author.Date)
		case *result.FileMatch:
			// File match searches are more expensive, because we must blame the
			// (first) line in order to know its placement in our sparkline.
			blameOps++
			if blameOps > maxBlame {
				// We have exceeded our budget of blame operations for
				// calculating this sparkline, so don't do any more file match
				// blaming.
				continue loop
			}

			run.Acquire()
			goroutine.Go(func() {
				defer run.Release()

				// Blame the file match in order to retrieve date informatino.
				var err error
				t, err := sr.blameFileMatch(ctx, m)
				if err != nil {
					log15.Warn("failed to blame fileMatch during sparkline generation", "error", err)
					return
				}
				addPoint(t)
			})
		default:
			panic("SearchResults.Sparkline unexpected union type state")
		}
	}
	span := opentracing.SpanFromContext(ctx)
	span.SetTag("blame_ops", blameOps)
	return sparkline, nil
}

var (
	searchResponseCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "src_graphql_search_response",
		Help: "Number of searches that have ended in the given status (success, error, timeout, partial_timeout).",
	}, []string{"status", "alert_type", "source", "request_name"})

	searchLatencyHistogram = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "src_search_response_latency_seconds",
		Help:    "Search response latencies in seconds that have ended in the given status (success, error, timeout, partial_timeout).",
		Buckets: []float64{0.01, 0.02, 0.05, 0.1, 0.2, 0.5, 1, 2, 5, 10, 30},
	}, []string{"status", "alert_type", "source", "request_name"})
)

// LogSearchLatency records search durations in the event database. This
// function may only be called after a search result is performed, because it
// relies on the invariant that query and pattern error checking has already
// been performed.
func LogSearchLatency(ctx context.Context, db dbutil.DB, si *run.SearchInputs, durationMs int32) {
	tr, ctx := trace.New(ctx, "LogSearchLatency", "")
	defer func() {
		tr.Finish()
	}()
	var types []string
	resultTypes, _ := si.Query.StringValues(query.FieldType)
	for _, typ := range resultTypes {
		switch typ {
		case "repo", "symbol", "diff", "commit":
			types = append(types, typ)
		case "path":
			// Map type:path to file
			types = append(types, "file")
		case "file":
			switch {
			case si.PatternType == query.SearchTypeStructural:
				types = append(types, "structural")
			case si.PatternType == query.SearchTypeLiteral:
				types = append(types, "literal")
			case si.PatternType == query.SearchTypeRegex:
				types = append(types, "regexp")
			}
		}
	}

	// Don't record composite searches that specify more than one type:
	// because we can't break down the search timings into multiple
	// categories.
	if len(types) > 1 {
		return
	}

	q, err := query.ToBasicQuery(si.Query)
	if err != nil {
		// Can't convert to a basic query, can't guarantee accurate reporting.
		return
	}
	if !query.IsPatternAtom(q) {
		// Not an atomic pattern, can't guarantee accurate reporting.
		return
	}

	// If no type: was explicitly specified, infer the result type.
	if len(types) == 0 {
		// If a pattern was specified, a content search happened.
		if q.IsLiteral() {
			types = append(types, "literal")
		} else if q.IsRegexp() {
			types = append(types, "regexp")
		} else if q.IsStructural() {
			types = append(types, "structural")
		} else if len(si.Query.Fields()["file"]) > 0 {
			// No search pattern specified and file: is specified.
			types = append(types, "file")
		} else {
			// No search pattern or file: is specified, assume repo.
			// This includes accounting for searches of fields that
			// specify repohasfile: and repohascommitafter:.
			types = append(types, "repo")
		}
	}

	// Only log the time if we successfully resolved one search type.
	if len(types) == 1 {
		a := actor.FromContext(ctx)
		if a.IsAuthenticated() {
			value := fmt.Sprintf(`{"durationMs": %d}`, durationMs)
			eventName := fmt.Sprintf("search.latencies.%s", types[0])
			featureFlags := featureflag.FromContext(ctx)
			go func() {
				err := usagestats.LogBackendEvent(db, a.UID, eventName, json.RawMessage(value), featureFlags, nil)
				if err != nil {
					log15.Warn("Could not log search latency", "err", err)
				}
			}()
		}
	}
}

func (r *searchResolver) toRepoOptions(q query.Q, opts resolveRepositoriesOpts) search.RepoOptions {
	repoFilters, minusRepoFilters := q.Repositories()
	if opts.effectiveRepoFieldValues != nil {
		repoFilters = opts.effectiveRepoFieldValues

	}
	repoGroupFilters, _ := q.StringValues(query.FieldRepoGroup)

	var settingForks, settingArchived bool
	if v := r.UserSettings.SearchIncludeForks; v != nil {
		settingForks = *v
	}
	if v := r.UserSettings.SearchIncludeArchived; v != nil {
		settingArchived = *v
	}

	fork := query.No
	if searchrepos.ExactlyOneRepo(repoFilters) || settingForks {
		// fork defaults to No unless either of:
		// (1) exactly one repo is being searched, or
		// (2) user/org/global setting includes forks
		fork = query.Yes
	}
	if setFork := q.Fork(); setFork != nil {
		fork = *setFork
	}

	archived := query.No
	if searchrepos.ExactlyOneRepo(repoFilters) || settingArchived {
		// archived defaults to No unless either of:
		// (1) exactly one repo is being searched, or
		// (2) user/org/global setting includes archives in all searches
		archived = query.Yes
	}
	if setArchived := q.Archived(); setArchived != nil {
		archived = *setArchived
	}

	visibilityStr, _ := q.StringValue(query.FieldVisibility)
	visibility := query.ParseVisibility(visibilityStr)

	commitAfter, _ := q.StringValue(query.FieldRepoHasCommitAfter)
	searchContextSpec, _ := q.StringValue(query.FieldContext)

	var versionContextName string
	if r.VersionContext != nil {
		versionContextName = *r.VersionContext
	}

	var CacheLookup bool
	if len(opts.effectiveRepoFieldValues) == 0 && opts.limit == 0 {
		// indicates resolving repositories should cache DB lookups
		CacheLookup = true
	}

	return search.RepoOptions{
		RepoFilters:        repoFilters,
		MinusRepoFilters:   minusRepoFilters,
		RepoGroupFilters:   repoGroupFilters,
		VersionContextName: versionContextName,
		SearchContextSpec:  searchContextSpec,
		UserSettings:       r.UserSettings,
		OnlyForks:          fork == query.Only,
		NoForks:            fork == query.No,
		OnlyArchived:       archived == query.Only,
		NoArchived:         archived == query.No,
		OnlyPrivate:        visibility == query.Private,
		OnlyPublic:         visibility == query.Public,
		CommitAfter:        commitAfter,
		Query:              q,
		Ranked:             true,
		Limit:              opts.limit,
		CacheLookup:        CacheLookup,
	}
}

func withMode(args search.TextParameters, st query.SearchType, versionContext *string) search.TextParameters {
	isGlobalSearch := func() bool {
		if st == query.SearchTypeStructural {
			return false
		}
		if versionContext != nil && *versionContext != "" {
			return false
		}
		querySearchContextSpec, _ := args.Query.StringValue(query.FieldContext)
		if !searchcontexts.IsGlobalSearchContextSpec(querySearchContextSpec) {
			return false
		}
		return len(args.Query.Values(query.FieldRepo)) == 0 && len(args.Query.Values(query.FieldRepoGroup)) == 0 && len(args.Query.Values(query.FieldRepoHasFile)) == 0
	}

	isFileOrPath := args.ResultTypes.Has(result.TypeFile) || args.ResultTypes.Has(result.TypePath)
	isIndexedSearch := args.PatternInfo.Index != query.No
	isEmpty := args.PatternInfo.Pattern == "" && args.PatternInfo.ExcludePattern == "" && len(args.PatternInfo.IncludePatterns) == 0
	if isGlobalSearch() && isIndexedSearch && isFileOrPath && !isEmpty {
		args.Mode = search.ZoektGlobalSearch
	}
	if isEmpty {
		args.Mode = search.SkipContentAndPathSearch
	}
	return args
}

func (r *searchResolver) toTextParameters(q query.Q) (*search.TextParameters, error) {
	forceResultTypes := result.TypeEmpty
	if r.PatternType == query.SearchTypeStructural {
		forceResultTypes = result.TypeFile
	}

	b, err := query.ToBasicQuery(q)
	if err != nil {
		return nil, err
	}
	p := search.ToTextPatternInfo(b, r.protocol(), query.Identity)

	// Fallback to literal search for searching repos and files if
	// the structural search pattern is empty.
	if r.PatternType == query.SearchTypeStructural && p.Pattern == "" {
		r.PatternType = query.SearchTypeLiteral
		p.IsStructuralPat = false
		forceResultTypes = result.Types(0)
	}

	args := search.TextParameters{
		PatternInfo: p,
		Query:       q,
		Timeout:     search.TimeoutDuration(b),

		// UseFullDeadline if timeout: set or we are streaming.
		UseFullDeadline: q.Timeout() != nil || q.Count() != nil || r.stream != nil,

		Zoekt:        r.zoekt,
		SearcherURLs: r.searcherURLs,
		RepoPromise:  &search.RepoPromise{},
	}
	args = withResultTypes(args, forceResultTypes)
	args = withMode(args, r.PatternType, r.VersionContext)
	return &args, nil
}

// evaluateLeaf performs a single search operation and corresponds to the
// evaluation of leaf expression in a query.
func (r *searchResolver) evaluateLeaf(ctx context.Context, args *search.TextParameters) (_ *SearchResults, err error) {
	tr, ctx := trace.New(ctx, "evaluateLeaf", "")
	defer func() {
		tr.SetError(err)
		tr.Finish()
	}()

	return r.resultsWithTimeoutSuggestion(ctx, args)
}

// union returns the union of two sets of search results and merges common search data.
func union(left, right *SearchResults) *SearchResults {
	if right == nil {
		return left
	}
	if left == nil {
		return right
	}

	if left.Matches != nil && right.Matches != nil {
		left.Matches = result.Union(left.Matches, right.Matches)
		left.Stats.Update(&right.Stats)
		return left
	} else if right.Matches != nil {
		return right
	}
	return left
}

// intersect returns the intersection of two sets of search result content
// matches, based on whether a single file path contains content matches in both sets.
func intersect(left, right *SearchResults) *SearchResults {
	if left == nil || right == nil {
		return nil
	}
	left.Matches = result.Intersect(left.Matches, right.Matches)
	left.Stats.Update(&right.Stats)
	return left
}

// evaluateAnd performs set intersection on result sets. It collects results for
// all expressions that are ANDed together by searching for each subexpression
// and then intersects those results that are in the same repo/file path. To
// collect N results for count:N, we need to opportunistically ask for more than
// N results for each subexpression (since intersect can never yield more than N,
// and likely yields fewer than N results). If the intersection does not yield N
// results, and is not exhaustive for every expression, we rerun the search by
// doubling count again.
func (r *searchResolver) evaluateAnd(ctx context.Context, q query.Basic) (*SearchResults, error) {
	start := time.Now()

	// Invariant: this function is only reachable from callers that
	// guarantee a root node with one or more operands.
	operands := q.Pattern.(query.Operator).Operands

	var (
		err        error
		result     *SearchResults
		termResult *SearchResults
	)

	// The number of results we want. Note that for intersect, this number
	// corresponds to documents, not line matches. By default, we ask for at
	// least 5 documents to fill the result page.
	want := 5
	// The fraction of file matches two terms share on average
	averageIntersection := 0.05
	// When we retry, cap the max search results we request for each expression
	// if search continues to not be exhaustive. Alert if exceeded.
	maxTryCount := 40000

	// Set an overall timeout in addition to the timeouts that are set for leaf-requests.
	ctx, cancel := context.WithTimeout(ctx, search.TimeoutDuration(q))
	defer cancel()

	if count := q.GetCount(); count != "" {
		want, _ = strconv.Atoi(count) // Invariant: count is validated.
	} else {
		q = q.AddCount(want)
	}

	// tryCount starts small but grows exponentially with the number of operands. It is capped at maxTryCount.
	tryCount := int(math.Floor(float64(want) / math.Pow(averageIntersection, float64(len(operands)-1))))
	if tryCount > maxTryCount {
		tryCount = maxTryCount
	}

	var exhausted bool
	for {
		q = q.MapCount(tryCount)
		result, err = r.evaluatePatternExpression(ctx, q.MapPattern(operands[0]))
		if err != nil {
			return nil, err
		}
		if result == nil {
			return &SearchResults{}, nil
		}
		if len(result.Matches) == 0 {
			// result might contain an alert.
			return result, nil
		}
		exhausted = !result.Stats.IsLimitHit
		for _, term := range operands[1:] {
			// check if we exceed the overall time limit before running the next query.
			select {
			case <-ctx.Done():
				usedTime := time.Since(start)
				suggestTime := longer(2, usedTime)
				return alertForTimeout(usedTime, suggestTime, r).wrapResults(), nil
			default:
			}

			termResult, err = r.evaluatePatternExpression(ctx, q.MapPattern(term))
			if err != nil {
				return nil, err
			}
			if termResult == nil {
				return &SearchResults{}, nil
			}
			if len(termResult.Matches) == 0 {
				// termResult might contain an alert.
				return termResult, nil
			}
			exhausted = exhausted && !termResult.Stats.IsLimitHit
			result = intersect(result, termResult)
		}
		if exhausted {
			break
		}
		if len(result.Matches) >= want {
			break
		}
		// If the result size set is not big enough, and we haven't
		// exhausted search on all expressions, double the tryCount and search more.
		tryCount *= 2
		if tryCount > maxTryCount {
			// We've capped out what we're willing to do, throw alert.
			return alertForCappedAndExpression().wrapResults(), nil
		}
	}
	result.Stats.IsLimitHit = !exhausted
	return result, nil
}

// evaluateOr performs set union on result sets. It collects results for all
// expressions that are ORed together by searching for each subexpression. If
// the maximum number of results are reached after evaluating a subexpression,
// we shortcircuit and return results immediately.
func (r *searchResolver) evaluateOr(ctx context.Context, q query.Basic) (*SearchResults, error) {
	// Invariant: this function is only reachable from callers that
	// guarantee a root node with one or more operands.
	operands := q.Pattern.(query.Operator).Operands

	wantCount := defaultMaxSearchResults
	if count := q.GetCount(); count != "" {
		wantCount, _ = strconv.Atoi(count) // Invariant: count is already validated
	}

	result := &SearchResults{}
	for _, term := range operands {
		new, err := r.evaluatePatternExpression(ctx, q.MapPattern(term))
		if err != nil {
			return nil, err
		}
		if new != nil {
			result = union(result, new)
			// Do not rely on result.Stats.resultCount because it may
			// count non-content matches and there's no easy way to know.
			if len(result.Matches) > wantCount {
				result.Matches = result.Matches[:wantCount]
				return result, nil
			}
		}
	}
	return result, nil
}

// invalidateCache invalidates the repo cache if we are preparing to evaluate
// subexpressions that require resolving potentially disjoint repository data.
func (r *searchResolver) invalidateCache() {
	if r.invalidateRepoCache {
		r.resolved.RepoRevs = nil
		r.resolved.MissingRepoRevs = nil
		r.repoErr = nil
	}
}

// evaluatePatternExpression evaluates a search pattern containing and/or expressions.
func (r *searchResolver) evaluatePatternExpression(ctx context.Context, q query.Basic) (*SearchResults, error) {
	switch term := q.Pattern.(type) {
	case query.Operator:
		if len(term.Operands) == 0 {
			return &SearchResults{}, nil
		}

		switch term.Kind {
		case query.And:
			return r.evaluateAnd(ctx, q)
		case query.Or:
			return r.evaluateOr(ctx, q)
		case query.Concat:
			r.invalidateCache()
			args, err := r.toTextParameters(q.ToParseTree())
			if err != nil {
				return &SearchResults{}, err
			}
			return r.evaluateLeaf(ctx, args)
		}
	case query.Pattern:
		r.invalidateCache()
		args, err := r.toTextParameters(q.ToParseTree())
		if err != nil {
			return &SearchResults{}, err
		}
		return r.evaluateLeaf(ctx, args)
	case query.Parameter:
		// evaluatePatternExpression does not process Parameter nodes.
		return &SearchResults{}, nil
	}
	// Unreachable.
	return nil, errors.Errorf("unrecognized type %T in evaluatePatternExpression", q.Pattern)
}

// evaluate evaluates all expressions of a search query.
func (r *searchResolver) evaluate(ctx context.Context, q query.Basic) (*SearchResults, error) {
	if q.Pattern == nil {
		r.invalidateCache()
		args, err := r.toTextParameters(query.ToNodes(q.Parameters))
		if err != nil {
			return &SearchResults{}, err
		}
		return r.evaluateLeaf(ctx, args)
	}
	return r.evaluatePatternExpression(ctx, q)
}

// shouldInvalidateRepoCache returns whether resolved repos should be invalidated when
// evaluating subexpressions. If a query contains more than one repo, revision,
// or repogroup field, we should invalidate resolved repos, since multiple
// repos, revisions, or repogroups imply that different repos may need to be
// resolved.
func shouldInvalidateRepoCache(plan query.Plan) bool {
	var seenRepo, seenRevision, seenRepoGroup, seenContext int
	query.VisitParameter(plan.ToParseTree(), func(field, _ string, _ bool, _ query.Annotation) {
		switch field {
		case query.FieldRepo:
			seenRepo += 1
		case query.FieldRev:
			seenRevision += 1
		case query.FieldRepoGroup:
			seenRepoGroup += 1
		case query.FieldContext:
			seenContext += 1
		}
	})
	return seenRepo+seenRepoGroup > 1 || seenRevision > 1 || seenContext > 1
}

func logPrometheusBatch(status, alertType, requestSource, requestName string, elapsed time.Duration) {
	searchResponseCounter.WithLabelValues(
		status,
		alertType,
		requestSource,
		requestName,
	).Inc()

	searchLatencyHistogram.WithLabelValues(
		status,
		alertType,
		requestSource,
		requestName,
	).Observe(elapsed.Seconds())
}

func newHoneyEvent(ctx context.Context, status, alertType, requestSource, requestName, query string, elapsed time.Duration, srr *SearchResultsResolver) *libhoney.Event {
	var n int
	if srr != nil {
		n = len(srr.Matches)
	}
	return honey.SearchEvent(ctx, honey.SearchEventArgs{
		OriginalQuery: query,
		Typ:           requestName,
		Source:        requestSource,
		Status:        status,
		AlertType:     alertType,
		DurationMs:    elapsed.Milliseconds(),
		ResultSize:    n,
	})
}

func logHoneyBatch(ctx context.Context, status, alertType, requestSource, requestName string, elapsed time.Duration, query string, start time.Time, srr *SearchResultsResolver) {
	var ev *libhoney.Event
	isSlow := time.Since(start) > searchlogs.LogSlowSearchesThreshold()

	if honey.Enabled() || isSlow {
		ev = newHoneyEvent(ctx, status, alertType, requestSource, requestName, query, elapsed, srr)
	}

	if honey.Enabled() && ev != nil {
		_ = ev.Send()
	}

	if isSlow && ev != nil {
		log15.Warn("slow search request", searchlogs.MapToLog15Ctx(ev.Fields())...)
	}
}

func (r *searchResolver) logBatch(ctx context.Context, srr *SearchResultsResolver, start time.Time, err error) {
	elapsed := time.Since(start)
	if srr != nil {
		srr.elapsed = elapsed
		LogSearchLatency(ctx, r.db, r.SearchInputs, srr.ElapsedMilliseconds())
	}

	var status, alertType string
	status = DetermineStatusForLogs(srr, err)
	if srr != nil && srr.SearchResults.Alert != nil {
		alertType = srr.SearchResults.Alert.PrometheusType()
	}
	requestSource := string(trace.RequestSource(ctx))
	requestName := trace.GraphQLRequestName(ctx)
	logPrometheusBatch(status, alertType, requestSource, requestName, elapsed)
	logHoneyBatch(ctx, status, alertType, requestSource, requestName, elapsed, r.rawQuery(), start, srr)
}

func (r *searchResolver) resultsBatch(ctx context.Context) (*SearchResultsResolver, error) {
	start := time.Now()
	sr, err := r.resultsRecursive(ctx, r.Plan)
	srr := r.resultsToResolver(sr)
	r.logBatch(ctx, srr, start, err)
	return srr, err
}

func (r *searchResolver) resultsStreaming(ctx context.Context) (*SearchResultsResolver, error) {
	if !query.IsStreamingCompatible(r.Plan) {
		// The query is not streaming compatible, but we still want to
		// use the streaming endpoint. Run a batch search then send the
		// results back on the stream.
		endpoint := r.stream
		r.stream = nil // Disables streaming: backends may not use the endpoint.
		srr, err := r.resultsBatch(ctx)
		if srr != nil {
			endpoint.Send(streaming.SearchEvent{
				Results: srr.Matches,
				Stats:   srr.Stats,
			})
		}
		return srr, err
	}
	if sp, _ := r.Plan.ToParseTree().StringValue(query.FieldSelect); sp != "" {
		// Ensure downstream events sent on the stream are processed by `select:`.
		selectPath, _ := filter.SelectPathFromString(sp) // Invariant: error already checked
		r.stream = streaming.WithSelect(r.stream, selectPath)
	}
	sr, err := r.resultsRecursive(ctx, r.Plan)
	srr := r.resultsToResolver(sr)
	return srr, err
}

func (r *searchResolver) resultsToResolver(results *SearchResults) *SearchResultsResolver {
	if results == nil {
		results = &SearchResults{}
	}
	return &SearchResultsResolver{
		SearchResults: results,
		limit:         r.MaxResults(),
		db:            r.db,
		UserSettings:  r.UserSettings,
	}
}

func (r *searchResolver) Results(ctx context.Context) (*SearchResultsResolver, error) {
	if r.stream == nil {
		return r.resultsBatch(ctx)
	}
	return r.resultsStreaming(ctx)
}

// DetermineStatusForLogs determines the final status of a search for logging
// purposes.
func DetermineStatusForLogs(srr *SearchResultsResolver, err error) string {
	switch {
	case err == context.DeadlineExceeded:
		return "timeout"
	case err != nil:
		return "error"
	case srr.Stats.AllReposTimedOut():
		return "timeout"
	case srr.Stats.Status.Any(search.RepoStatusTimedout):
		return "partial_timeout"
	case srr.SearchResults.Alert != nil:
		return "alert"
	default:
		return "success"
	}
}

func (r *searchResolver) resultsRecursive(ctx context.Context, plan query.Plan) (sr *SearchResults, err error) {
	tr, ctx := trace.New(ctx, "Results", "")
	defer func() {
		tr.SetError(err)
		tr.Finish()
	}()

	if shouldInvalidateRepoCache(plan) {
		r.invalidateRepoCache = true
	}

	wantCount := defaultMaxSearchResults
	if count := r.Query.Count(); count != nil {
		wantCount = *count
	}

	for _, q := range plan {
		predicatePlan, err := substitutePredicates(q, func(pred query.Predicate) (*SearchResults, error) {
			// Disable streaming for subqueries so we can use
			// the results rather than sending them back to the caller
			orig := r.stream
			r.stream = nil
			defer func() { r.stream = orig }()

			r.invalidateRepoCache = true
			plan, err := pred.Plan(q)
			if err != nil {
				return nil, err
			}
			return r.resultsRecursive(ctx, plan)
		})
		if errors.Is(err, ErrPredicateNoResults) {
			continue
		}
		if err != nil {
			// Fail if predicate processing fails.
			return nil, err
		}
		if predicatePlan != nil {
			// If a predicate filter generated a new plan, evaluate that plan.
			return r.resultsRecursive(ctx, predicatePlan)
		}

		newResult, err := r.evaluate(ctx, q)
		if err != nil {
			// Fail if any subexpression fails.
			return nil, err
		}

		if newResult != nil {
			newResult.Matches = result.Select(newResult.Matches, q)
			sr = union(sr, newResult)
			if len(sr.Matches) > wantCount {
				sr.Matches = sr.Matches[:wantCount]
				break
			}
		}
	}

	if sr != nil {
		r.sortResults(sr.Matches)
	}
	return sr, err
}

// searchResultsToRepoNodes converts a set of search results into repository nodes
// such that they can be used to replace a repository predicate
func searchResultsToRepoNodes(matches []result.Match) ([]query.Node, error) {
	nodes := make([]query.Node, 0, len(matches))
	for _, match := range matches {
		repoMatch, ok := match.(*result.RepoMatch)
		if !ok {
			return nil, errors.Errorf("expected type %T, but got %T", &result.RepoMatch{}, match)
		}

		nodes = append(nodes, query.Parameter{
			Field: query.FieldRepo,
			Value: "^" + regexp.QuoteMeta(string(repoMatch.Name)) + "$",
		})
	}

	return nodes, nil
}

// searchResultsToFileNodes converts a set of search results into repo/file nodes so that they
// can replace a file predicate
func searchResultsToFileNodes(matches []result.Match) ([]query.Node, error) {
	nodes := make([]query.Node, 0, len(matches))
	for _, match := range matches {
		fileMatch, ok := match.(*result.FileMatch)
		if !ok {
			return nil, errors.Errorf("expected type %T, but got %T", &result.FileMatch{}, match)
		}

		// We create AND nodes to match both the repo and the file at the same time so
		// we don't get files of the same name from different repositories.
		nodes = append(nodes, query.Operator{
			Kind: query.And,
			Operands: []query.Node{
				query.Parameter{
					Field: query.FieldRepo,
					Value: "^" + regexp.QuoteMeta(string(fileMatch.Repo.Name)) + "$",
				},
				query.Parameter{
					Field: query.FieldFile,
					Value: "^" + regexp.QuoteMeta(fileMatch.Path) + "$",
				},
			},
		})
	}

	return nodes, nil
}

// resultsWithTimeoutSuggestion calls doResults, and in case of deadline
// exceeded returns a search alert with a did-you-mean link for the same
// query with a longer timeout.
func (r *searchResolver) resultsWithTimeoutSuggestion(ctx context.Context, args *search.TextParameters) (*SearchResults, error) {
	start := time.Now()
	rr, err := r.doResults(ctx, args)

	// If we encountered a context timeout, it indicates one of the many result
	// type searchers (file, diff, symbol, etc) completely timed out and could not
	// produce even partial results. Other searcher types may have produced results.
	//
	// In this case, or if we got a partial timeout where ALL repositories timed out,
	// we do not return partial results and instead display a timeout alert.
	shouldShowAlert := err == context.DeadlineExceeded
	if err == nil && rr.Stats.AllReposTimedOut() {
		shouldShowAlert = true
	}
	if shouldShowAlert {
		usedTime := time.Since(start)
		suggestTime := longer(2, usedTime)
		return alertForTimeout(usedTime, suggestTime, r).wrapResults(), nil
	}
	return rr, err
}

// substitutePredicates replaces all the predicates in a query with their expanded form. The predicates
// are expanded using the doExpand function.
func substitutePredicates(q query.Basic, evaluate func(query.Predicate) (*SearchResults, error)) (query.Plan, error) {
	var topErr error
	success := false
	newQ := query.MapParameter(q.ToParseTree(), func(field, value string, neg bool, ann query.Annotation) query.Node {
		orig := query.Parameter{
			Field:      field,
			Value:      value,
			Negated:    neg,
			Annotation: ann,
		}

		if !ann.Labels.IsSet(query.IsPredicate) {
			return orig
		}

		if topErr != nil {
			return orig
		}

		name, params := query.ParseAsPredicate(value)
		predicate := query.DefaultPredicateRegistry.Get(field, name)
		predicate.ParseParams(params)
		srr, err := evaluate(predicate)
		if err != nil {
			topErr = err
			return nil
		}

		var nodes []query.Node
		switch predicate.Field() {
		case query.FieldRepo:
			nodes, err = searchResultsToRepoNodes(srr.Matches)
			if err != nil {
				topErr = err
				return nil
			}
		case query.FieldFile:
			nodes, err = searchResultsToFileNodes(srr.Matches)
			if err != nil {
				topErr = err
				return nil
			}
		default:
			topErr = errors.Errorf("unsupported predicate result type %q", predicate.Field())
			return nil
		}

		// If no results are returned, we need to return a sentinel error rather
		// than an empty expansion because an empty expansion means "everything"
		// rather than "nothing".
		if len(nodes) == 0 {
			topErr = ErrPredicateNoResults
			return nil
		}

		// A predicate was successfully evaluated and has results.
		success = true

		// No need to return an operator for only one result
		if len(nodes) == 1 {
			return nodes[0]
		}

		return query.Operator{
			Kind:     query.Or,
			Operands: nodes,
		}
	})

	if topErr != nil || !success {
		return nil, topErr
	}
	plan, err := query.ToPlan(query.Dnf(newQ))
	if err != nil {
		return nil, err
	}
	return plan, nil
}

var ErrPredicateNoResults = errors.New("no results returned for predicate")

// longer returns a suggested longer time to wait if the given duration wasn't long enough.
func longer(n int, dt time.Duration) time.Duration {
	dt2 := func() time.Duration {
		Ndt := time.Duration(n) * dt
		dceil := func(x float64) time.Duration {
			return time.Duration(math.Ceil(x))
		}
		switch {
		case math.Floor(Ndt.Hours()) > 0:
			return dceil(Ndt.Hours()) * time.Hour
		case math.Floor(Ndt.Minutes()) > 0:
			return dceil(Ndt.Minutes()) * time.Minute
		case math.Floor(Ndt.Seconds()) > 0:
			return dceil(Ndt.Seconds()) * time.Second
		default:
			return 0
		}
	}()
	lowest := 2 * time.Second
	if dt2 < lowest {
		return lowest
	}
	return dt2
}

type searchResultsStats struct {
	JApproximateResultCount string
	JSparkline              []int32

	sr *searchResolver

	once   sync.Once
	srs    *SearchResultsResolver
	srsErr error
}

func (srs *searchResultsStats) ApproximateResultCount() string { return srs.JApproximateResultCount }
func (srs *searchResultsStats) Sparkline() []int32             { return srs.JSparkline }

var (
	searchResultsStatsCache   = rcache.NewWithTTL("search_results_stats", 3600) // 1h
	searchResultsStatsCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "src_graphql_search_results_stats_cache_hit",
		Help: "Counts cache hits and misses for search results stats (e.g. sparklines).",
	}, []string{"type"})
)

func (r *searchResolver) Stats(ctx context.Context) (stats *searchResultsStats, err error) {
	// Override user context to ensure that stats for this query are cached
	// regardless of the user context's cancellation. For example, if
	// stats/sparklines are slow to load on the homepage and all users navigate
	// away from that page before they load, no user would ever see them and we
	// would never cache them. This fixes that by ensuring the first request
	// 'kicks off loading' and places the result into cache regardless of
	// whether or not the original querier of this information still wants it.
	originalCtx := ctx
	ctx = context.Background()
	ctx = opentracing.ContextWithSpan(ctx, opentracing.SpanFromContext(originalCtx))

	cacheKey := r.rawQuery()
	// Check if value is in the cache.
	jsonRes, ok := searchResultsStatsCache.Get(cacheKey)
	if ok {
		searchResultsStatsCounter.WithLabelValues("hit").Inc()
		if err := json.Unmarshal(jsonRes, &stats); err != nil {
			return nil, err
		}
		stats.sr = r
		return stats, nil
	}

	// Calculate value from scratch.
	searchResultsStatsCounter.WithLabelValues("miss").Inc()
	attempts := 0
	var v *SearchResultsResolver
	for {
		// Query search results.
		var err error
		args, err := r.toTextParameters(r.Query)
		if err != nil {
			return nil, err
		}
		results, err := r.doResults(ctx, args)
		if err != nil {
			return nil, err // do not cache errors.
		}
		v = r.resultsToResolver(results)
		if v.MatchCount() > 0 {
			break
		}

		cloning := len(v.Cloning())
		timedout := len(v.Timedout())
		if cloning == 0 && timedout == 0 {
			break // zero results, but no cloning or timed out repos. No point in retrying.
		}

		if attempts > 5 {
			log15.Error("failed to generate sparkline due to cloning or timed out repos", "cloning", len(v.Cloning()), "timedout", len(v.Timedout()))
			return nil, errors.Errorf("failed to generate sparkline due to %d cloning %d timedout repos", len(v.Cloning()), len(v.Timedout()))
		}

		// We didn't find any search results. Some repos are cloning or timed
		// out, so try again in a few seconds.
		attempts++
		log15.Warn("sparkline generation found 0 search results due to cloning or timed out repos (retrying in 5s)", "cloning", len(v.Cloning()), "timedout", len(v.Timedout()))
		time.Sleep(5 * time.Second)
	}

	sparkline, err := v.Sparkline(ctx)
	if err != nil {
		return nil, err // sparkline generation failed, so don't cache.
	}
	stats = &searchResultsStats{
		JApproximateResultCount: v.ApproximateResultCount(),
		JSparkline:              sparkline,
		sr:                      r,
	}

	// Store in the cache if we got non-zero results. If we got zero results,
	// it should be quick and caching is not desired because e.g. it could be
	// a query for a repo that has not been added by the user yet.
	if v.ResultCount() > 0 {
		jsonRes, err = json.Marshal(stats)
		if err != nil {
			return nil, err
		}
		searchResultsStatsCache.Set(cacheKey, jsonRes)
	}
	return stats, nil
}

// withResultTypes populates the ResultTypes field of args, which drives the kind
// of search to run (e.g., text search, symbol search).
func withResultTypes(args search.TextParameters, forceTypes result.Types) search.TextParameters {
	var rts result.Types
	if forceTypes != 0 {
		rts = forceTypes
	} else {
		stringTypes, _ := args.Query.StringValues(query.FieldType)
		if len(stringTypes) == 0 {
			rts = result.TypeFile | result.TypePath | result.TypeRepo
		} else {
			for _, stringType := range stringTypes {
				rts = rts.With(result.TypeFromString[stringType])
			}
		}
	}

	if rts.Has(result.TypeFile) {
		args.PatternInfo.PatternMatchesContent = true
	}

	if rts.Has(result.TypePath) {
		args.PatternInfo.PatternMatchesPath = true
	}
	args.ResultTypes = rts
	return args
}

// doResults is one of the highest level search functions that handles finding results.
//
// If forceOnlyResultType is specified, only results of the given type are returned,
// regardless of what `type:` is specified in the query string.
//
// Partial results AND an error may be returned.
func (r *searchResolver) doResults(ctx context.Context, args *search.TextParameters) (_ *SearchResults, err error) {
	tr, ctx := trace.New(ctx, "doResults", r.rawQuery())
	defer func() {
		tr.SetError(err)
		tr.Finish()
	}()

	start := time.Now()

	ctx, cancel := context.WithTimeout(ctx, args.Timeout)
	defer cancel()

	limit := r.MaxResults()
	tr.LazyPrintf("resultTypes: %s", args.ResultTypes)
	var (
		requiredWg sync.WaitGroup
		optionalWg sync.WaitGroup
	)

	waitGroup := func(required bool) *sync.WaitGroup {
		if args.UseFullDeadline {
			// When a custom timeout is specified, all searches are required and get the full timeout.
			return &requiredWg
		}
		if required {
			return &requiredWg
		}
		return &optionalWg
	}

	// For streaming search we want to limit based on all results, not just
	// per backend. This works better than batch based since we have higher
	// defaults.
	stream := r.stream
	if stream != nil {
		var cancelOnLimit context.CancelFunc
		ctx, stream, cancelOnLimit = streaming.WithLimit(ctx, stream, limit)
		defer cancelOnLimit()
	}

	agg := run.NewAggregator(r.db, stream)

	// This ensures we properly cleanup in the case of an early return. In
	// particular we want to cancel global searches before returning early.
	hasStartedAllBackends := false
	defer func() {
		if hasStartedAllBackends {
			return
		}
		cancel()
		requiredWg.Wait()
		optionalWg.Wait()
		_, _, _ = agg.Get()
	}()

	args.RepoOptions = r.toRepoOptions(args.Query, resolveRepositoriesOpts{})

	// performance optimization: call zoekt early, resolve repos concurrently, filter
	// search results with resolved repos.
	if args.Mode == search.ZoektGlobalSearch {
		argsIndexed := *args
		wg := waitGroup(true)
		wg.Add(1)
		goroutine.Go(func() {
			defer wg.Done()
			_ = agg.DoFilePathSearch(ctx, &argsIndexed)
		})
		// On sourcegraph.com and for unscoped queries, determineRepos returns the subset
		// of indexed default searchrepos. No need to call searcher, because
		// len(searcherRepos) will always be 0.
		if envvar.SourcegraphDotComMode() {
			args.Mode = search.SkipContentAndPathSearch
		} else {
			args.Mode = search.SearcherOnly
		}
	}

	resolved, err := r.resolveRepositories(ctx, args.RepoOptions)
	if err != nil {
		if alert, err := errorToAlert(err); alert != nil {
			return alert.wrapResults(), err
		}
		return nil, err
	}

	tr.LazyPrintf("searching %d repos, %d missing", len(resolved.RepoRevs), len(resolved.MissingRepoRevs))
	if len(resolved.RepoRevs) == 0 {
		return r.alertForNoResolvedRepos(ctx, args.Query).wrapResults(), nil
	}

	if len(resolved.MissingRepoRevs) > 0 {
		agg.Error(&missingRepoRevsError{Missing: resolved.MissingRepoRevs})
	}

	// Send down our first bit of progress.
	{
		repos := make(map[api.RepoID]types.RepoName, len(resolved.RepoRevs))
		for _, repoRev := range resolved.RepoRevs {
			repos[repoRev.Repo.ID] = repoRev.Repo
		}

		agg.Send(streaming.SearchEvent{
			Stats: streaming.Stats{
				Repos:            repos,
				ExcludedForks:    resolved.ExcludedRepos.Forks,
				ExcludedArchived: resolved.ExcludedRepos.Archived,
			},
		})
	}

	// Resolve repo promise so searches waiting on it can proceed. We do this
	// after reporting the above progress to ensure we don't get search
	// results before the above reporting.
	args.RepoPromise.Resolve(resolved.RepoRevs)

	if args.ResultTypes.Has(result.TypeRepo) {
		wg := waitGroup(true)
		wg.Add(1)
		goroutine.Go(func() {
			defer wg.Done()
			_ = agg.DoRepoSearch(ctx, args, int32(limit))
		})

	}

	if args.ResultTypes.Has(result.TypeSymbol) {
		wg := waitGroup(args.ResultTypes.Without(result.TypeSymbol) == 0)
		wg.Add(1)
		goroutine.Go(func() {
			defer wg.Done()
			_ = agg.DoSymbolSearch(ctx, args, limit)
		})
	}

	if args.ResultTypes.Has(result.TypeFile | result.TypePath) {
		if args.Mode != search.SkipContentAndPathSearch {
			wg := waitGroup(true)
			wg.Add(1)
			goroutine.Go(func() {
				defer wg.Done()
				_ = agg.DoFilePathSearch(ctx, args)
			})
		}
	}

	if args.ResultTypes.Has(result.TypeDiff) {
		wg := waitGroup(args.ResultTypes.Without(result.TypeDiff) == 0)
		wg.Add(1)
		goroutine.Go(func() {
			defer wg.Done()
			_ = agg.DoDiffSearch(ctx, args)
		})
	}

	if args.ResultTypes.Has(result.TypeCommit) {
		wg := waitGroup(args.ResultTypes.Without(result.TypeCommit) == 0)
		wg.Add(1)
		goroutine.Go(func() {
			defer wg.Done()
			_ = agg.DoCommitSearch(ctx, args)
		})

	}

	hasStartedAllBackends = true

	// Wait for required searches.
	requiredWg.Wait()

	// Give optional searches some minimum budget in case required searches return quickly.
	// Cancel all remaining searches after this minimum budget.
	budget := 100 * time.Millisecond
	elapsed := time.Since(start)
	timer := time.AfterFunc(budget-elapsed, cancel)

	// Wait for remaining optional searches to finish or get cancelled.
	optionalWg.Wait()

	timer.Stop()

	// We have to call get once all waitgroups are done since it relies on
	// collecting from the streams.
	matches, common, aggErrs := agg.Get()

	ao := alertObserver{
		Inputs:     r.SearchInputs,
		hasResults: len(matches) > 0,
	}
	for _, err := range aggErrs.Errors {
		ao.Error(ctx, err)
	}
	alert, err := ao.Done(&common)

	tr.LazyPrintf("matches=%d %s", len(matches), &common)

	r.sortResults(matches)

	return &SearchResults{
		Matches: matches,
		Stats:   common,
		Alert:   alert,
	}, err
}

// isContextError returns true if ctx.Err() is not nil or if err
// is an error caused by context cancelation or timeout.
func isContextError(ctx context.Context, err error) bool {
	return ctx.Err() != nil || err == context.Canceled || err == context.DeadlineExceeded
}

// SearchResultResolver is a resolver for the GraphQL union type `SearchResult`.
//
// Supported types:
//
//   - *RepositoryResolver         // repo name match
//   - *fileMatchResolver          // text match
//   - *commitSearchResultResolver // diff or commit match
//
// Note: Any new result types added here also need to be handled properly in search_results.go:301 (sparklines)
type SearchResultResolver interface {
	ToRepository() (*RepositoryResolver, bool)
	ToFileMatch() (*FileMatchResolver, bool)
	ToCommitSearchResult() (*CommitSearchResultResolver, bool)

	ResultCount() int32
}

// compareFileLengths sorts file paths such that they appear earlier if they
// match file: patterns in the query exactly.
func compareFileLengths(left, right string, exactFilePatterns map[string]struct{}) bool {
	_, aMatch := exactFilePatterns[path.Base(left)]
	_, bMatch := exactFilePatterns[path.Base(right)]
	if aMatch || bMatch {
		if aMatch && bMatch {
			// Prefer shorter file names (ie root files come first)
			if len(left) != len(right) {
				return len(left) < len(right)
			}
			return left < right
		}
		// Prefer exact match
		return aMatch
	}
	return left < right
}

func compareDates(left, right *time.Time) bool {
	if left == nil || right == nil {
		return left != nil // Place the value that is defined first.
	}
	return left.After(*right)
}

// compareSearchResults sorts repository matches, file matches, and commits.
// Repositories and filenames are sorted alphabetically. As a refinement, if any
// filename matches a value in a non-empty set exactFilePatterns, then such
// filenames are listed earlier.
//
// Commits are sorted by date. Commits are not associated with searchrepos, and
// will always list after repository or file match results, if any.
func compareSearchResults(left, right result.Match, exactFilePatterns map[string]struct{}) bool {
	sortKeys := func(match result.Match) (string, string, *time.Time) {
		switch r := match.(type) {
		case *result.RepoMatch:
			return string(r.Name), "", nil
		case *result.FileMatch:
			return string(r.Repo.Name), r.Path, nil
		case *result.CommitMatch:
			// Commits are relatively sorted by date, and after repo
			// or path names. We use ~ as the key for repo and
			// paths,lexicographically last in ASCII.
			return "~", "~", &r.Commit.Author.Date
		}
		// Unreachable.
		panic("unreachable: compareSearchResults expects RepositoryResolver, FileMatchResolver, or CommitSearchResultResolver")
	}

	arepo, afile, adate := sortKeys(left)
	brepo, bfile, bdate := sortKeys(right)

	if arepo == brepo {
		if len(exactFilePatterns) == 0 {
			if afile != bfile {
				return afile < bfile
			}
			return compareDates(adate, bdate)
		}
		return compareFileLengths(afile, bfile, exactFilePatterns)
	}
	return arepo < brepo
}

func (r *searchResolver) sortResults(results []result.Match) {
	var exactPatterns map[string]struct{}
	if getBoolPtr(r.UserSettings.SearchGlobbing, false) {
		exactPatterns = r.getExactFilePatterns()
	}
	sort.Slice(results, func(i, j int) bool { return compareSearchResults(results[i], results[j], exactPatterns) })
}

// getExactFilePatterns returns the set of file patterns without glob syntax.
func (r *searchResolver) getExactFilePatterns() map[string]struct{} {
	m := map[string]struct{}{}
	query.VisitField(
		r.Query,
		query.FieldFile,
		func(value string, negated bool, annotation query.Annotation) {
			originalValue := r.OriginalQuery[annotation.Range.Start.Column+len(query.FieldFile)+1 : annotation.Range.End.Column]
			if !negated && query.ContainsNoGlobSyntax(originalValue) {
				m[originalValue] = struct{}{}
			}
		})
	return m
}
