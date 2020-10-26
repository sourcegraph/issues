package graphqlbackend

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"path"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/inconshreveable/log15"
	"github.com/neelance/parallel"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	otlog "github.com/opentracing/opentracing-go/log"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/envvar"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/internal/inventory"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/internal/usagestats"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/internal/actor"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/authz"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/gitserver"
	"github.com/sourcegraph/sourcegraph/internal/goroutine"
	"github.com/sourcegraph/sourcegraph/internal/honey"
	"github.com/sourcegraph/sourcegraph/internal/lazyregexp"
	"github.com/sourcegraph/sourcegraph/internal/rcache"
	"github.com/sourcegraph/sourcegraph/internal/search"
	"github.com/sourcegraph/sourcegraph/internal/search/query"
	"github.com/sourcegraph/sourcegraph/internal/trace"
	"github.com/sourcegraph/sourcegraph/internal/trace/ot"
	"github.com/sourcegraph/sourcegraph/internal/vcs/git"
	"github.com/sourcegraph/sourcegraph/schema"
	"github.com/src-d/enry/v2"
)

// searchResultsCommon contains fields that should be returned by all funcs
// that contribute to the overall search result set.
type searchResultsCommon struct {
	limitHit bool // whether the limit on results was hit

	repos    []*types.Repo             // repos that were matched by the repo-related filters
	searched []*types.Repo             // repos that were searched
	indexed  []*types.Repo             // repos that were searched using an index
	cloning  []*types.Repo             // repos that could not be searched because they were still being cloned
	missing  []*types.Repo             // repos that could not be searched because they do not exist
	excluded excludedRepos             // repo counts of excluded repos because the search query doesn't apply to them, but that we want to know about (forks, archives)
	partial  map[api.RepoName]struct{} // repos that were searched, but have results that were not returned due to exceeded limits

	maxResultsCount, resultCount int32

	// timedout usually contains repos that haven't finished being fetched yet.
	// This should only happen for large repos and the searcher caches are
	// purged.
	timedout []*types.Repo

	indexUnavailable bool // True if indexed search is enabled but was not available during this search.
}

func (c *searchResultsCommon) LimitHit() bool {
	return c.limitHit || c.resultCount > c.maxResultsCount
}

func (c *searchResultsCommon) Repositories() []*RepositoryResolver {
	return RepositoryResolvers(c.repos)
}

func (c *searchResultsCommon) RepositoriesCount() int32 {
	return int32(len(c.repos))
}

func (c *searchResultsCommon) RepositoriesSearched() []*RepositoryResolver {
	return RepositoryResolvers(c.searched)
}

func (c *searchResultsCommon) IndexedRepositoriesSearched() []*RepositoryResolver {
	return RepositoryResolvers(c.indexed)
}

func (c *searchResultsCommon) Cloning() []*RepositoryResolver {
	return RepositoryResolvers(c.cloning)
}

func (c *searchResultsCommon) Missing() []*RepositoryResolver {
	return RepositoryResolvers(c.missing)
}

func (c *searchResultsCommon) Timedout() []*RepositoryResolver {
	return RepositoryResolvers(c.timedout)
}

func (c *searchResultsCommon) IndexUnavailable() bool {
	return c.indexUnavailable
}

func (c *searchResultsCommon) Equal(other *searchResultsCommon) bool {
	return reflect.DeepEqual(c, other)
}

func RepositoryResolvers(repos types.Repos) []*RepositoryResolver {
	dedupSort(&repos)
	return toRepositoryResolvers(repos)
}

// update updates c with the other data, deduping as necessary. It modifies c but
// does not modify other.
func (c *searchResultsCommon) update(other searchResultsCommon) {
	c.limitHit = c.limitHit || other.limitHit
	c.indexUnavailable = c.indexUnavailable || other.indexUnavailable

	c.repos = append(c.repos, other.repos...)
	c.searched = append(c.searched, other.searched...)
	c.indexed = append(c.indexed, other.indexed...)
	c.cloning = append(c.cloning, other.cloning...)
	c.missing = append(c.missing, other.missing...)
	c.excluded.forks = c.excluded.forks + other.excluded.forks
	c.excluded.archived = c.excluded.archived + other.excluded.archived
	c.timedout = append(c.timedout, other.timedout...)
	c.resultCount += other.resultCount

	if c.partial == nil {
		c.partial = make(map[api.RepoName]struct{})
	}

	for repo := range other.partial {
		c.partial[repo] = struct{}{}
	}
}

// dedupSort sorts (by ID in ascending order) and deduplicates
// the given repos in-place.
func dedupSort(repos *types.Repos) {
	if len(*repos) == 0 {
		return
	}

	sort.Sort(*repos)

	j := 0
	for i := 1; i < len(*repos); i++ {
		if (*repos)[j].ID != (*repos)[i].ID {
			j++
			(*repos)[j] = (*repos)[i]
		}
	}

	*repos = (*repos)[:j+1]
}

// SearchResultsResolver is a resolver for the GraphQL type `SearchResults`
type SearchResultsResolver struct {
	SearchResults []SearchResultResolver
	searchResultsCommon
	alert *searchAlert
	start time.Time // when the results started being computed

	// cursor to return for paginated search requests, or nil if the request
	// wasn't paginated.
	cursor *searchCursor

	// cache for user settings. Ideally this should be set just once in the code path
	// by an upstream resolver
	userSettings *schema.Settings
}

func (sr *SearchResultsResolver) Results() []SearchResultResolver {
	return sr.SearchResults
}

func (sr *SearchResultsResolver) MatchCount() int32 {
	var totalResults int32
	for _, result := range sr.SearchResults {
		totalResults += result.resultCount()
	}
	return totalResults
}

// Deprecated. Prefer MatchCount.
func (sr *SearchResultsResolver) ResultCount() int32 { return sr.MatchCount() }

func (sr *SearchResultsResolver) ApproximateResultCount() string {
	count := sr.MatchCount()
	if sr.LimitHit() || len(sr.cloning) > 0 || len(sr.timedout) > 0 {
		return fmt.Sprintf("%d+", count)
	}
	return strconv.Itoa(int(count))
}

func (sr *SearchResultsResolver) Alert() *searchAlert { return sr.alert }

func (sr *SearchResultsResolver) ElapsedMilliseconds() int32 {
	return int32(time.Since(sr.start).Nanoseconds() / int64(time.Millisecond))
}

// commonFileFilters are common filters used. It is used by DynamicFilters to
// propose them if they match shown results.
var commonFileFilters = []struct {
	regexp      *lazyregexp.Regexp
	regexFilter string
	globFilter  string
}{
	// Exclude go tests
	{
		regexp:      lazyregexp.New(`_test\.go$`),
		regexFilter: `-file:_test\.go$`,
		globFilter:  `-file:**_test.go`,
	},
	// Exclude go vendor
	{
		regexp:      lazyregexp.New(`(^|/)vendor/`),
		regexFilter: `-file:(^|/)vendor/`,
		globFilter:  `-file:vendor/** -file:**/vendor/**`,
	},
	// Exclude node_modules
	{
		regexp:      lazyregexp.New(`(^|/)node_modules/`),
		regexFilter: `-file:(^|/)node_modules/`,
		globFilter:  `-file:node_modules/** -file:**/node_modules/**`,
	},
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
	if sr.userSettings != nil {
		globbing = getBoolPtr(sr.userSettings.SearchGlobbing, false)
	} else {
		settings, err := decodedViewerFinalSettings(ctx)
		if err != nil {
			log15.Warn("DynamicFilters: could not get user settings from db")
		} else {
			globbing = getBoolPtr(settings.SearchGlobbing, false)
		}
	}
	tr.LogFields(otlog.Bool("globbing", globbing))

	filters := map[string]*searchFilterResolver{}
	repoToMatchCount := make(map[string]int32)
	add := func(value string, label string, count int32, limitHit bool, kind string, score score) {
		sf, ok := filters[value]
		if !ok {
			sf = &searchFilterResolver{
				value:    value,
				label:    label,
				count:    count,
				limitHit: limitHit,
				kind:     kind,
			}
			filters[value] = sf
		} else {
			sf.count = count
		}

		sf.score = score
	}

	addRepoFilter := func(uri string, rev string, lineMatchCount int32) {
		var filter string
		if globbing {
			filter = fmt.Sprintf(`repo:%s`, uri)
		} else {
			filter = fmt.Sprintf(`repo:^%s$`, regexp.QuoteMeta(uri))
		}

		if rev != "" {
			// We don't need to quote rev. The only special characters we interpret
			// are @ and :, both of which are disallowed in git refs
			filter = filter + fmt.Sprintf(`@%s`, rev)
		}
		_, limitHit := sr.searchResultsCommon.partial[api.RepoName(uri)]
		// Increment number of matches per repo. Add will override previous entry for uri
		repoToMatchCount[uri] += lineMatchCount
		add(filter, uri, repoToMatchCount[uri], limitHit, "repo", scoreDefault)
	}

	addFileFilter := func(fileMatchPath string, lineMatchCount int32, limitHit bool) {
		for _, ff := range commonFileFilters {
			// use regexp to match file paths unconditionally, whether globbing is enabled or not,
			// since we have no native library call to match `**` for globs.
			if ff.regexp.MatchString(fileMatchPath) {
				if globbing {
					add(ff.globFilter, ff.globFilter, lineMatchCount, limitHit, "file", scoreDefault)
				} else {
					add(ff.regexFilter, ff.regexFilter, lineMatchCount, limitHit, "file", scoreDefault)
				}
			}
		}
	}

	addLangFilter := func(fileMatchPath string, lineMatchCount int32, limitHit bool) {
		extensionToLanguageLookup := func(path string) string {
			language, _ := inventory.GetLanguageByFilename(path)
			return strings.ToLower(language)
		}
		if ext := path.Ext(fileMatchPath); ext != "" {
			language := extensionToLanguageLookup(fileMatchPath)
			if language != "" {
				if strings.Contains(language, " ") {
					language = strconv.Quote(language)
				}
				value := fmt.Sprintf(`lang:%s`, language)
				add(value, value, lineMatchCount, limitHit, "lang", scoreDefault)
			}
		}
	}

	if sr.searchResultsCommon.excluded.forks > 0 {
		add("fork:yes", "fork:yes", int32(sr.searchResultsCommon.excluded.forks), sr.limitHit, "repo", scoreImportant)
	}
	if sr.searchResultsCommon.excluded.archived > 0 {
		add("archived:yes", "archived:yes", int32(sr.searchResultsCommon.excluded.archived), sr.limitHit, "repo", scoreImportant)
	}
	for _, result := range sr.SearchResults {
		if fm, ok := result.ToFileMatch(); ok {
			rev := ""
			if fm.InputRev != nil {
				rev = *fm.InputRev
			}
			lines := fm.resultCount()
			addRepoFilter(fm.Repo.Name(), rev, lines)
			addLangFilter(fm.path(), lines, fm.LimitHit())
			addFileFilter(fm.path(), lines, fm.LimitHit())

			if len(fm.symbols) > 0 {
				add("type:symbol", "type:symbol", 1, fm.LimitHit(), "symbol", scoreDefault)
			}
		} else if r, ok := result.ToRepository(); ok {
			// It should be fine to leave this blank since revision specifiers
			// can only be used with the 'repo:' scope. In that case,
			// we shouldn't be getting any repositoy name matches back.
			addRepoFilter(r.Name(), "", 1)
		}
	}

	filterSlice := make([]*searchFilterResolver, 0, len(filters))
	repoFilterSlice := make([]*searchFilterResolver, 0, len(filters)/2) // heuristic - half of all filters are repo filters.
	for _, f := range filters {
		if f.kind == "repo" {
			repoFilterSlice = append(repoFilterSlice, f)
		} else {
			filterSlice = append(filterSlice, f)
		}
	}
	sort.Slice(filterSlice, func(i, j int) bool {
		return filterSlice[j].score < filterSlice[i].score
	})
	// limit amount of non-repo filters to be rendered arbitrarily to 12
	if len(filterSlice) > 12 {
		filterSlice = filterSlice[:12]
	}

	allFilters := append(filterSlice, repoFilterSlice...)
	sort.Slice(allFilters, func(i, j int) bool {
		left := allFilters[i]
		right := allFilters[j]
		if left.score == right.score {
			// Order alphabetically for equal scores.
			return strings.Compare(left.value, right.value) < 0
		}
		return left.score < right.score
	})

	return allFilters
}

type searchFilterResolver struct {
	value string

	// the string to be displayed in the UI
	label string

	// the number of matches in a particular repository. Only used for `repo:` filters.
	count int32

	// whether the results returned for a repository are incomplete
	limitHit bool

	// the kind of filter. Should be "repo", "file", or "lang".
	kind string

	// score is used to prioritize the order that filters appear in
	score score
}

type score int

const (
	scoreImportant score = iota
	scoreDefault
)

func (sf *searchFilterResolver) Value() string {
	return sf.value
}

func (sf *searchFilterResolver) Label() string {
	return sf.label
}

func (sf *searchFilterResolver) Count() int32 {
	return sf.count
}

func (sf *searchFilterResolver) LimitHit() bool {
	return sf.limitHit
}

func (sf *searchFilterResolver) Kind() string {
	return sf.kind
}

// blameFileMatch blames the specified file match to produce the time at which
// the first line match inside of it was authored.
func (sr *SearchResultsResolver) blameFileMatch(ctx context.Context, fm *FileMatchResolver) (t time.Time, err error) {
	span, ctx := ot.StartSpanFromContext(ctx, "blameFileMatch")
	defer func() {
		if err != nil {
			ext.Error.Set(span, true)
			span.SetTag("err", err.Error())
		}
		span.Finish()
	}()

	// Blame the first line match.
	lineMatches := fm.LineMatches()
	if len(lineMatches) == 0 {
		// No line match
		return time.Time{}, nil
	}
	lm := fm.LineMatches()[0]
	hunks, err := git.BlameFile(ctx, gitserver.Repo{Name: fm.Repo.repo.Name}, fm.path(), &git.BlameOptions{
		NewestCommit: fm.CommitID,
		StartLine:    int(lm.LineNumber()),
		EndLine:      int(lm.LineNumber()),
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
	for _, r := range sr.SearchResults {
		r := r // shadow so it doesn't change in the goroutine
		switch m := r.(type) {
		case *RepositoryResolver:
			// We don't care about repo results here.
			continue
		case *CommitSearchResultResolver:
			// Diff searches are cheap, because we implicitly have author date info.
			addPoint(m.commit.author.date)
		case *FileMatchResolver:
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

var searchResponseCounter = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "src_graphql_search_response",
	Help: "Number of searches that have ended in the given status (success, error, timeout, partial_timeout).",
}, []string{"status", "alert_type", "source", "request_name"})

// logSearchLatency records search durations in the event database. This
// function may only be called after a search result is performed, because it
// relies on the invariant that query and pattern error checking has already
// been performed.
func (r *searchResolver) logSearchLatency(ctx context.Context, durationMs int32) {
	tr, ctx := trace.New(ctx, "logSearchLatency", "")
	defer func() {
		tr.Finish()
	}()
	var types []string
	resultTypes, _ := r.query.StringValues(query.FieldType)
	for _, typ := range resultTypes {
		switch typ {
		case "repo", "symbol", "diff", "commit":
			types = append(types, typ)
		case "path":
			// Map type:path to file
			types = append(types, "file")
		case "file":
			switch {
			case r.patternType == query.SearchTypeStructural:
				types = append(types, "structural")
			case r.patternType == query.SearchTypeLiteral:
				types = append(types, "literal")
			case r.patternType == query.SearchTypeRegex:
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

	options := &getPatternInfoOptions{}
	if r.patternType == query.SearchTypeStructural {
		options = &getPatternInfoOptions{performStructuralSearch: true}
	}
	if r.patternType == query.SearchTypeLiteral {
		options = &getPatternInfoOptions{performLiteralSearch: true}
	}
	p, _ := r.getPatternInfo(options)

	// If no type: was explicitly specified, infer the result type.
	if len(types) == 0 {
		// If a pattern was specified, a content search happened.
		if p.Pattern != "" {
			switch {
			case r.patternType == query.SearchTypeStructural:
				types = append(types, "structural")
			case r.patternType == query.SearchTypeLiteral:
				types = append(types, "literal")
			case r.patternType == query.SearchTypeRegex:
				types = append(types, "regexp")
			}
		} else if len(r.query.Fields()["file"]) > 0 {
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
		actor := actor.FromContext(ctx)
		if actor.IsAuthenticated() {
			value := fmt.Sprintf(`{"durationMs": %d}`, durationMs)
			eventName := fmt.Sprintf("search.latencies.%s", types[0])
			go func() {
				err := usagestats.LogBackendEvent(actor.UID, eventName, json.RawMessage(value))
				if err != nil {
					log15.Warn("Could not log search latency", "err", err)
				}
			}()
		}
	}
}

// evaluateLeaf performs a single search operation and corresponds to the
// evaluation of leaf expression in a query.
func (r *searchResolver) evaluateLeaf(ctx context.Context) (_ *SearchResultsResolver, err error) {
	tr, ctx := trace.New(ctx, "evaluateLeaf", "")
	defer func() {
		tr.SetError(err)
		tr.Finish()
	}()
	start := time.Now()
	// If the request specifies stable:truthy, use pagination to return a stable ordering.
	if r.query.BoolValue("stable") {
		result, err := r.paginatedResults(ctx)
		if err != nil {
			return nil, err
		}
		if result == nil {
			// Panic if paginatedResults does not ensure a non-nil search result.
			panic("stable search: paginated search returned nil results")
		}
		if result.cursor == nil {
			// Perhaps an alert was raised.
			return result, err
		}
		if !result.cursor.Finished {
			// For stable result queries limitHit = true implies
			// there is a next cursor, and more results may exist.
			result.searchResultsCommon.limitHit = true
		}
		return result, err
	}

	// If the request is a paginated one, we handle it separately. See
	// paginatedResults for more details.
	if r.pagination != nil {
		return r.paginatedResults(ctx)
	}

	rr, err := r.resultsWithTimeoutSuggestion(ctx)
	if rr != nil {
		r.logSearchLatency(ctx, rr.ElapsedMilliseconds())
	}

	// Record what type of response we sent back via Prometheus.
	var status, alertType string
	switch {
	case err == context.DeadlineExceeded || (err == nil && len(rr.searchResultsCommon.timedout) > 0 && len(rr.searchResultsCommon.timedout) == len(rr.searchResultsCommon.repos)):
		status = "timeout"
	case err == nil && len(rr.searchResultsCommon.timedout) > 0:
		status = "partial_timeout"
	case err == nil && rr.alert != nil:
		status = "alert"
		alertType = rr.alert.prometheusType
	case err != nil:
		status = "error"
	case err == nil:
		status = "success"
	default:
		status = "unknown"
	}
	searchResponseCounter.WithLabelValues(
		status,
		alertType,
		string(trace.RequestSource(ctx)),
		trace.GraphQLRequestName(ctx),
	).Inc()

	isSlow := time.Since(start) > logSlowSearchesThreshold()
	if honey.Enabled() || isSlow {
		var act actor.Actor
		if a := actor.FromContext(ctx); a != nil {
			act = *a
		}

		ev := honey.Event("search")
		ev.AddField("query", r.rawQuery())
		ev.AddField("actor_uid", act.UID)
		ev.AddField("actor_internal", act.Internal)
		ev.AddField("type", trace.GraphQLRequestName(ctx))
		ev.AddField("source", string(trace.RequestSource(ctx)))
		ev.AddField("status", status)
		ev.AddField("alert_type", alertType)
		ev.AddField("duration_ms", time.Since(start).Milliseconds())
		if rr != nil {
			ev.AddField("result_size", len(rr.SearchResults))
		}

		if honey.Enabled() {
			_ = ev.Send()
		}

		if isSlow {
			log15.Warn("slow search request", mapToLog15Ctx(ev.Fields())...)
		}
	}

	return rr, err
}

// unionMerge performs a merge of file match results, merging line matches when
// they occur in the same file, and taking care to update match counts.
func unionMerge(left, right *SearchResultsResolver) *SearchResultsResolver {
	var count int // count non-overlapping files when we merge.
	var merged []SearchResultResolver
	rightFileMatches := make(map[string]*FileMatchResolver)
	rightRepoMatches := make(map[string]*RepositoryResolver)

	// accumulate matches for the right subexpression in a lookup.
	for _, r := range right.SearchResults {
		if fileMatch, ok := r.ToFileMatch(); ok {
			rightFileMatches[fileMatch.uri] = fileMatch
			continue
		}
		if repoMatch, ok := r.ToRepository(); ok {
			rightRepoMatches[string(repoMatch.URL())] = repoMatch
			continue
		}
		merged = append(merged, r)
	}

	for _, leftMatch := range left.SearchResults {
		if leftFileMatch, ok := leftMatch.ToFileMatch(); ok {
			rightFileMatch := rightFileMatches[leftFileMatch.uri]
			if rightFileMatch == nil {
				// no overlap with existing matches.
				merged = append(merged, leftMatch)
				count++
				continue
			}

			// merge line matches with a file match that already exists.
			rightFileMatch.appendMatches(leftFileMatch)
			rightFileMatches[leftFileMatch.uri] = rightFileMatch
			continue
		}

		if leftRepoMatch, ok := leftMatch.ToRepository(); ok {
			rightRepoMatch := rightRepoMatches[string(leftRepoMatch.URL())]
			if rightRepoMatch == nil {
				// no overlap with existing matches.
				merged = append(merged, leftMatch)
				count++
				continue
			}

			rightRepoMatches[string(leftRepoMatch.URL())] = rightRepoMatch
			continue
		}
		merged = append(merged, leftMatch)
	}

	for _, v := range rightFileMatches {
		merged = append(merged, v)
	}
	for _, v := range rightRepoMatches {
		merged = append(merged, v)
	}

	left.SearchResults = merged
	left.searchResultsCommon.update(right.searchResultsCommon)
	// set the count that tracks non-overlapping result count.
	left.searchResultsCommon.resultCount = int32(count)
	return left
}

// union returns the union of two sets of search results and merges common search data.
func union(left, right *SearchResultsResolver) *SearchResultsResolver {
	if right == nil {
		return left
	}
	if left == nil {
		return right
	}

	if left.SearchResults != nil && right.SearchResults != nil {
		return unionMerge(left, right)
	} else if right.SearchResults != nil {
		return right
	}
	return left
}

// intersectMerge performs a merge of file match results, merging line matches
// for files contained in both result sets, and updating counts.
func intersectMerge(left, right *SearchResultsResolver) *SearchResultsResolver {
	rightFileMatches := make(map[string]*FileMatchResolver)
	for _, r := range right.SearchResults {
		if fileMatch, ok := r.ToFileMatch(); ok {
			rightFileMatches[fileMatch.uri] = fileMatch
		}
	}

	var merged []SearchResultResolver
	for _, leftMatch := range left.SearchResults {
		leftFileMatch, ok := leftMatch.ToFileMatch()
		if !ok {
			continue
		}

		rightFileMatch := rightFileMatches[leftFileMatch.uri]
		if rightFileMatch == nil {
			continue
		}

		leftFileMatch.appendMatches(rightFileMatch)
		merged = append(merged, leftMatch)
	}
	left.SearchResults = merged
	left.searchResultsCommon.update(right.searchResultsCommon)
	// for intersect we want the newly computed intersection size.
	left.searchResultsCommon.resultCount = int32(len(merged))
	return left
}

// intersect returns the intersection of two sets of search result content
// matches, based on whether a single file path contains content matches in both sets.
func intersect(left, right *SearchResultsResolver) *SearchResultsResolver {
	if left == nil || right == nil {
		return nil
	}
	return intersectMerge(left, right)
}

// evaluateAnd performs set intersection on result sets. It collects results for
// all expressions that are ANDed together by searching for each subexpression
// and then intersects those results that are in the same repo/file path. To
// collect N results for count:N, we need to opportunistically ask for more than
// N results for each subexpression (since intersect can never yield more than N,
// and likely yields fewer than N results). If the intersection does not yield N
// results, and is not exhaustive for every expression, we rerun the search by
// doubling count again.
func (r *searchResolver) evaluateAnd(ctx context.Context, scopeParameters []query.Node, operands []query.Node) (*SearchResultsResolver, error) {
	start := time.Now()

	if len(operands) == 0 {
		return nil, nil
	}

	var (
		err        error
		result     *SearchResultsResolver
		termResult *SearchResultsResolver
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
	ctx, cancel, err := r.withTimeout(ctx)
	if err != nil {
		return nil, err
	}
	defer cancel()

	// Set count: if not specified.
	var countStr string
	query.VisitField(scopeParameters, "count", func(value string, _ bool, _ query.Annotation) {
		countStr = value
	})
	if countStr != "" {
		// Override "want" if count is specified.
		want, _ = strconv.Atoi(countStr) // Invariant: count is validated.
	} else {
		scopeParameters = append(scopeParameters, query.Parameter{
			Field: "count",
			Value: strconv.FormatInt(int64(want), 10),
		})
	}

	// tryCount starts small but grows exponentially with the number of operands. It is capped at maxTryCount.
	tryCount := int(math.Floor(float64(want) / math.Pow(averageIntersection, float64(len(operands)-1))))
	if tryCount > maxTryCount {
		tryCount = maxTryCount
	}

	var exhausted bool
	for {
		scopeParameters = query.MapParameter(scopeParameters, func(field, value string, negated bool, annotation query.Annotation) query.Node {
			if field == "count" {
				value = strconv.FormatInt(int64(tryCount), 10)
			}
			return query.Parameter{Field: field, Value: value, Negated: negated, Annotation: annotation}
		})

		result, err = r.evaluatePatternExpression(ctx, scopeParameters, operands[0])
		if err != nil {
			return nil, err
		}
		if result == nil {
			return nil, nil
		}
		exhausted = !result.limitHit
		for _, term := range operands[1:] {
			// check if we exceed the overall time limit before running the next query.
			select {
			case <-ctx.Done():
				usedTime := time.Since(start)
				suggestTime := longer(2, usedTime)
				return alertForTimeout(usedTime, suggestTime, r).wrap(), nil
			default:
			}

			termResult, err = r.evaluatePatternExpression(ctx, scopeParameters, term)
			if err != nil {
				return nil, err
			}
			if termResult != nil {
				exhausted = exhausted && !termResult.limitHit
				result = intersect(result, termResult)
			}
		}
		if exhausted {
			break
		}
		if result.searchResultsCommon.resultCount >= int32(want) {
			break
		}
		// If the result size set is not big enough, and we haven't
		// exhausted search on all expressions, double the tryCount and search more.
		tryCount *= 2
		if tryCount > maxTryCount {
			// We've capped out what we're willing to do, throw alert.
			return alertForCappedAndExpression().wrap(), nil
		}
	}
	result.limitHit = !exhausted
	return result, nil
}

// evaluateOr performs set union on result sets. It collects results for all
// expressions that are ORed together by searching for each subexpression. If
// the maximum number of results are reached after evaluating a subexpression,
// we shortcircuit and return results immediately.
func (r *searchResolver) evaluateOr(ctx context.Context, scopeParameters []query.Node, operands []query.Node) (*SearchResultsResolver, error) {
	if len(operands) == 0 {
		return nil, nil
	}

	var countStr string
	wantCount := defaultMaxSearchResults
	query.VisitField(scopeParameters, "count", func(value string, _ bool, _ query.Annotation) {
		countStr = value
	})
	if countStr != "" {
		wantCount, _ = strconv.Atoi(countStr) // Invariant: count is validated.
	}

	result, err := r.evaluatePatternExpression(ctx, scopeParameters, operands[0])
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	// Do not rely on result.searchResultsCommon.resultCount because it may
	// count non-content matches and there's no easy way to know.
	if len(result.SearchResults) > wantCount {
		result.SearchResults = result.SearchResults[:wantCount]
		result.searchResultsCommon.resultCount = int32(wantCount)
		return result, nil
	}
	var new *SearchResultsResolver
	for _, term := range operands[1:] {
		new, err = r.evaluatePatternExpression(ctx, scopeParameters, term)
		if err != nil {
			return nil, err
		}
		if new != nil {
			result = union(result, new)
			// Do not rely on result.searchResultsCommon.resultCount because it may
			// count non-content matches and there's no easy way to know.
			if len(result.SearchResults) > wantCount {
				result.SearchResults = result.SearchResults[:wantCount]
				result.searchResultsCommon.resultCount = int32(wantCount)
				return result, nil
			}
		}
	}
	return result, nil
}

func (r *searchResolver) evaluateOperator(ctx context.Context, scopeParameters []query.Node, operator query.Operator) (*SearchResultsResolver, error) {
	if len(operator.Operands) == 0 {
		return nil, nil
	}
	var result *SearchResultsResolver
	var err error
	if operator.Kind == query.And {
		result, err = r.evaluateAnd(ctx, scopeParameters, operator.Operands)
	} else {
		result, err = r.evaluateOr(ctx, scopeParameters, operator.Operands)
	}
	if err != nil {
		return nil, err
	}
	return result, nil
}

// setQuery sets a new query in the search resolver, for potentially repeated
// calls in the search pipeline. The important part is it takes care of
// invalidating cached repo info.
func (r *searchResolver) setQuery(q []query.Node) {
	if r.invalidateRepoCache {
		r.resolved.repoRevs = nil
		r.resolved.missingRepoRevs = nil
		r.repoErr = nil
	}
	r.query.(*query.AndOrQuery).Query = q
}

// evaluatePatternExpression evaluates a search pattern containing and/or expressions.
func (r *searchResolver) evaluatePatternExpression(ctx context.Context, scopeParameters []query.Node, node query.Node) (*SearchResultsResolver, error) {
	switch term := node.(type) {
	case query.Operator:
		if term.Kind == query.And || term.Kind == query.Or {
			return r.evaluateOperator(ctx, scopeParameters, term)
		} else if term.Kind == query.Concat {
			r.setQuery(append(scopeParameters, term))
			return r.evaluateLeaf(ctx)
		}
	case query.Pattern:
		r.setQuery(append(scopeParameters, term))
		return r.evaluateLeaf(ctx)
	case query.Parameter:
		// evaluatePatternExpression does not process Parameter nodes.
		return nil, nil
	}
	// Unreachable.
	return nil, fmt.Errorf("unrecognized type %s in evaluatePatternExpression", reflect.TypeOf(node).String())
}

// evaluate evaluates all expressions of a search query.
func (r *searchResolver) evaluate(ctx context.Context, q []query.Node) (*SearchResultsResolver, error) {
	scopeParameters, pattern, err := query.PartitionSearchPattern(q)
	if err != nil {
		return alertForQuery("", err).wrap(), nil
	}
	if pattern == nil {
		r.setQuery(scopeParameters)
		return r.evaluateLeaf(ctx)
	}
	result, err := r.evaluatePatternExpression(ctx, scopeParameters, pattern)
	if err != nil {
		return nil, err
	}
	r.sortResults(ctx, result.SearchResults)
	return result, nil
}

// setRepoCacheInvalidation sets whether resolved repos should be invalidated when
// evaluating subexpressions. If a query contains more than one repo or repogroup
// field, we should invalidate resolved repos, since multiple repo or repogroups
// imply that different repos may need to be resolved.
func (r *searchResolver) setRepoCacheInvalidation() {
	var seenRepo, seenRepoGroup int
	query.VisitField(r.query.(*query.AndOrQuery).Query, "repo", func(_ string, _ bool, _ query.Annotation) {
		seenRepo += 1
	})
	query.VisitField(r.query.(*query.AndOrQuery).Query, "repogroup", func(_ string, _ bool, _ query.Annotation) {
		seenRepoGroup += 1
	})
	if seenRepo+seenRepoGroup > 1 {
		r.invalidateRepoCache = true
	}
}

func (r *searchResolver) Results(ctx context.Context) (srr *SearchResultsResolver, err error) {
	tr, ctx := trace.New(ctx, "Results", "")
	defer func() {
		tr.SetError(err)
		tr.Finish()
	}()
	switch q := r.query.(type) {
	case *query.OrdinaryQuery:
		srr, err = r.evaluateLeaf(ctx)
	case *query.AndOrQuery:
		var countStr string
		wantCount := defaultMaxSearchResults
		query.VisitField(q.Query, "count", func(value string, _ bool, _ query.Annotation) {
			countStr = value
		})
		if countStr != "" {
			wantCount, _ = strconv.Atoi(countStr) // Invariant: count is validated.
		}

		for _, disjunct := range query.Dnf(q.Query) {
			disjunct = query.ConcatRevFilters(disjunct)
			newResult, err := r.evaluate(ctx, disjunct)
			if err != nil {
				// Fail if any subquery fails.
				return nil, err
			}
			if newResult != nil {
				srr = union(srr, newResult)
				if len(srr.SearchResults) > wantCount {
					srr.SearchResults = srr.SearchResults[:wantCount]
					srr.searchResultsCommon.resultCount = int32(wantCount)
					break
				}

			}
		}
		if srr != nil {
			r.sortResults(ctx, srr.SearchResults)
		}
	default:
		// Unreachable.
		return nil, fmt.Errorf("unrecognized type %s in searchResolver Results", reflect.TypeOf(r.query).String())
	}
	// copy userSettings from searchResolver to SearchResultsResolver
	if srr != nil {
		srr.userSettings = r.userSettings
	}
	if srr == nil {
		srr = &SearchResultsResolver{}
	}
	return srr, err
}

// resultsWithTimeoutSuggestion calls doResults, and in case of deadline
// exceeded returns a search alert with a did-you-mean link for the same
// query with a longer timeout.
func (r *searchResolver) resultsWithTimeoutSuggestion(ctx context.Context) (*SearchResultsResolver, error) {
	start := time.Now()
	rr, err := r.doResults(ctx, "")

	// If we encountered a context timeout, it indicates one of the many result
	// type searchers (file, diff, symbol, etc) completely timed out and could not
	// produce even partial results. Other searcher types may have produced results.
	//
	// In this case, or if we got a partial timeout where ALL repositories timed out,
	// we do not return partial results and instead display a timeout alert.
	shouldShowAlert := err == context.DeadlineExceeded
	if err == nil && len(rr.searchResultsCommon.timedout) > 0 && len(rr.searchResultsCommon.timedout) == len(rr.searchResultsCommon.repos) {
		shouldShowAlert = true
	}
	if shouldShowAlert {
		usedTime := time.Since(start)
		suggestTime := longer(2, usedTime)
		return alertForTimeout(usedTime, suggestTime, r).wrap(), nil
	}
	return rr, err
}

// longer returns a suggested longer time to wait if the given duration wasn't long enough.
func longer(N int, dt time.Duration) time.Duration {
	dt2 := func() time.Duration {
		Ndt := time.Duration(N) * dt
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

var decimalRx = lazyregexp.New(`\d+\.\d+`)

// roundStr rounds the first number containing a decimal within a string
func roundStr(s string) string {
	return decimalRx.ReplaceAllStringFunc(s, func(ns string) string {
		f, err := strconv.ParseFloat(ns, 64)
		if err != nil {
			return s
		}
		f = math.Round(f)
		return strconv.Itoa(int(f))
	})
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
	searchResultsStatsCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "src_graphql_search_results_stats_cache_hit",
		Help: "Counts cache hits and misses for search results stats (e.g. sparklines).",
	}, []string{"type"})
)

func init() {
	prometheus.MustRegister(searchResultsStatsCounter)
}

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
		v, err = r.doResults(ctx, "")
		if err != nil {
			return nil, err // do not cache errors.
		}
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
			return nil, fmt.Errorf("failed to generate sparkline due to %d cloning %d timedout repos", len(v.Cloning()), len(v.Timedout()))
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

type getPatternInfoOptions struct {
	// forceFileSearch, when true, specifies that the search query should be
	// treated as if every default term had `file:` before it. This can be used
	// to allow users to jump to files by just typing their name.
	forceFileSearch         bool
	performStructuralSearch bool
	performLiteralSearch    bool

	fileMatchLimit int32
}

// getPatternInfo gets the search pattern info for the query in the resolver.
func (r *searchResolver) getPatternInfo(opts *getPatternInfoOptions) (*search.TextPatternInfo, error) {
	if opts == nil {
		opts = &getPatternInfoOptions{}
	}

	if opts.fileMatchLimit == 0 {
		opts.fileMatchLimit = r.maxResults()
	}

	return getPatternInfo(r.query, opts)
}

func isPatternNegated(q []query.Node) bool {
	isNegated := false
	patternsFound := 0
	query.VisitPattern(q, func(_ string, negated bool, _ query.Annotation) {
		patternsFound++
		if patternsFound > 1 {
			return
		}
		isNegated = negated
	})

	// we only support negation for queries that contain exactly 1 pattern.
	if patternsFound > 1 {
		return false
	}
	return isNegated
}

// processSearchPattern processes the search pattern for a query. It handles the interpretation of search patterns
// as literal, regex, or structural patterns, and applies fuzzy regex matching if applicable.
func processSearchPattern(q query.QueryInfo, opts *getPatternInfoOptions) (string, bool, bool, bool) {
	var pattern string
	var pieces []string
	var contentFieldSet bool
	isRegExp := false
	isStructuralPat := false

	patternValues := q.Values(query.FieldDefault)

	isNegated := false
	if andOrQuery, ok := q.(*query.AndOrQuery); ok {
		isNegated = isPatternNegated(andOrQuery.Query)
	}

	if overridePattern := q.Values(query.FieldContent); len(overridePattern) > 0 {
		patternValues = overridePattern
		contentFieldSet = true
	}

	if opts.performStructuralSearch {
		isStructuralPat = true
		for _, v := range patternValues {
			if piece := v.ToString(); piece != "" {
				pieces = append(pieces, piece)
			}
		}
		pattern = strings.Join(pieces, " ")
	} else if !opts.forceFileSearch {
		isRegExp = true
		for _, v := range patternValues {
			var piece string
			switch {
			case v.String != nil:
				if contentFieldSet && !opts.performLiteralSearch {
					piece = *v.String
				} else {
					// Treat quoted strings as literal
					// strings to match, not regexps.
					piece = regexp.QuoteMeta(*v.String)
				}
			case v.Regexp != nil:
				piece = v.Regexp.String()
			}
			if piece == "" {
				continue
			}
			pieces = append(pieces, piece)
		}
		pattern = orderedFuzzyRegexp(pieces)
	} else {
		// TODO: We must have some pattern that always matches here, or else
		// cmd/searcher/search/matcher.go:97 would cause a nil regexp panic
		// when not using indexed search. I am unsure what the right solution
		// is here. Would this code path go away when we switch fully to
		// indexed search @keegan? This workaround is OK for now though.
		isRegExp = true
		pattern = "."
	}

	return pattern, isRegExp, isStructuralPat, isNegated
}

// getPatternInfo gets the search pattern info for q
func getPatternInfo(q query.QueryInfo, opts *getPatternInfoOptions) (*search.TextPatternInfo, error) {
	pattern, isRegExp, isStructuralPat, isNegated := processSearchPattern(q, opts)

	// Handle file: and -file: filters.
	includePatterns, excludePatterns := q.RegexpPatterns(query.FieldFile)
	filePatternsReposMustInclude, filePatternsReposMustExclude := q.RegexpPatterns(query.FieldRepoHasFile)

	if opts.forceFileSearch {
		for _, v := range q.Values(query.FieldDefault) {
			includePatterns = append(includePatterns, v.ToString())
		}
	}

	var combyRule []string
	for _, v := range q.Values(query.FieldCombyRule) {
		combyRule = append(combyRule, v.ToString())
	}

	// Handle lang: and -lang: filters.
	langIncludePatterns, langExcludePatterns, err := langIncludeExcludePatterns(q.StringValues(query.FieldLang))
	if err != nil {
		return nil, err
	}
	includePatterns = append(includePatterns, langIncludePatterns...)
	excludePatterns = append(excludePatterns, langExcludePatterns...)

	languages, _ := q.StringValues(query.FieldLang)

	patternInfo := &search.TextPatternInfo{
		IsRegExp:                     isRegExp,
		IsStructuralPat:              isStructuralPat,
		IsCaseSensitive:              q.IsCaseSensitive(),
		FileMatchLimit:               opts.fileMatchLimit,
		Pattern:                      pattern,
		IsNegated:                    isNegated,
		IncludePatterns:              includePatterns,
		FilePatternsReposMustInclude: filePatternsReposMustInclude,
		FilePatternsReposMustExclude: filePatternsReposMustExclude,
		Languages:                    languages,
		PathPatternsAreCaseSensitive: q.IsCaseSensitive(),
		CombyRule:                    strings.Join(combyRule, ""),
	}
	if len(excludePatterns) > 0 {
		patternInfo.ExcludePattern = unionRegExps(excludePatterns)
	}
	return patternInfo, nil
}

// langIncludeExcludePatterns returns regexps for the include/exclude path patterns given the lang:
// and -lang: filter values in a search query. For example, a query containing "lang:go" should
// include files whose paths match /\.go$/.
func langIncludeExcludePatterns(values, negatedValues []string) (includePatterns, excludePatterns []string, err error) {
	do := func(values []string, patterns *[]string) error {
		for _, value := range values {
			lang, ok := enry.GetLanguageByAlias(value)
			if !ok {
				return fmt.Errorf("unknown language: %q", value)
			}
			exts := enry.GetLanguageExtensions(lang)
			extPatterns := make([]string, len(exts))
			for i, ext := range exts {
				// Add `\.ext$` pattern to match files with the given extension.
				extPatterns[i] = regexp.QuoteMeta(ext) + "$"
			}
			*patterns = append(*patterns, unionRegExps(extPatterns))
		}
		return nil
	}

	if err := do(values, &includePatterns); err != nil {
		return nil, nil, err
	}
	if err := do(negatedValues, &excludePatterns); err != nil {
		return nil, nil, err
	}
	return includePatterns, excludePatterns, nil
}

var (
	// The default timeout to use for queries.
	defaultTimeout = 20 * time.Second
)

func (r *searchResolver) searchTimeoutFieldSet() bool {
	timeout, _ := r.query.StringValue(query.FieldTimeout)
	return timeout != "" || r.countIsSet()
}

func (r *searchResolver) withTimeout(ctx context.Context) (context.Context, context.CancelFunc, error) {
	d := defaultTimeout
	maxTimeout := time.Duration(searchLimits().MaxTimeoutSeconds) * time.Second
	timeout, _ := r.query.StringValue(query.FieldTimeout)
	if timeout != "" {
		var err error
		d, err = time.ParseDuration(timeout)
		if err != nil {
			return nil, nil, errors.WithMessage(err, `invalid "timeout:" value (examples: "timeout:2s", "timeout:200ms")`)
		}
	} else if r.countIsSet() {
		// If `count:` is set but `timeout:` is not explicitly set, use the max timeout
		d = maxTimeout
	}
	if d > maxTimeout {
		d = maxTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, d)
	return ctx, cancel, nil
}

func (r *searchResolver) determineResultTypes(args search.TextParameters, forceOnlyResultType string) (resultTypes []string) {
	// Determine which types of results to return.
	if forceOnlyResultType != "" {
		resultTypes = []string{forceOnlyResultType}
	} else {
		resultTypes, _ = r.query.StringValues(query.FieldType)
		if len(resultTypes) == 0 {
			resultTypes = []string{"file", "path", "repo"}
		}
	}
	for _, resultType := range resultTypes {
		if resultType == "file" {
			args.PatternInfo.PatternMatchesContent = true
		} else if resultType == "path" {
			args.PatternInfo.PatternMatchesPath = true
		}
	}
	return resultTypes
}

func (r *searchResolver) determineRepos(ctx context.Context, tr *trace.Trace, start time.Time) (resolved resolvedRepositories, res *SearchResultsResolver, err error) {
	resolved, err = r.resolveRepositories(ctx, nil)
	if err != nil {
		if errors.Is(err, authz.ErrStalePermissions{}) {
			log15.Debug("searchResolver.determineRepos", "err", err)
			alert := alertForStalePermissions()
			return resolvedRepositories{}, &SearchResultsResolver{alert: alert, start: start}, nil
		}
		e := git.BadCommitError{}
		if errors.As(err, &e) {
			alert := r.alertForInvalidRevision(e.Spec)
			return resolvedRepositories{}, &SearchResultsResolver{alert: alert, start: start}, nil
		}
		return resolvedRepositories{}, nil, err
	}

	tr.LazyPrintf("searching %d repos, %d missing", len(resolved.repoRevs), len(resolved.missingRepoRevs))
	if len(resolved.repoRevs) == 0 {
		alert := r.alertForNoResolvedRepos(ctx)
		return resolvedRepositories{}, &SearchResultsResolver{alert: alert, start: start}, nil
	}
	if resolved.overLimit {
		alert := r.alertForOverRepoLimit(ctx)
		return resolvedRepositories{}, &SearchResultsResolver{alert: alert, start: start}, nil
	}
	return resolved, nil, nil
}

// Surface an alert if a query exceeds limits that we place on search. Currently limits
// diff and commit searches where more than repoLimit repos need to be searched.
func alertOnSearchLimit(resultTypes []string, args *search.TextParameters) ([]string, *searchAlert) {
	limits := searchLimits()
	// we don't need to handle the error here, because we don't have context and the
	// promise can only return context errors
	repos, _ := getRepos(context.Background(), args.RepoPromise)

	for _, resultType := range resultTypes {
		if resultType != "commit" && resultType != "diff" {
			continue
		}

		hasTimeFilter := false
		if _, afterPresent := args.Query.Fields()["after"]; afterPresent {
			hasTimeFilter = true
		}
		if _, beforePresent := args.Query.Fields()["before"]; beforePresent {
			hasTimeFilter = true
		}

		if max := limits.CommitDiffMaxRepos; !hasTimeFilter && len(repos) > max {
			return []string{}, &searchAlert{
				prometheusType: "exceeded_diff_commit_search_limit",
				title:          fmt.Sprintf("Too many matching repositories for %s search to handle", resultType),
				description:    fmt.Sprintf(`%s search can currently only handle searching over %d repositories at a time. Try using the "repo:" filter to narrow down which repositories to search, or using 'after:"1 week ago"'. Tracking issue: https://github.com/sourcegraph/sourcegraph/issues/6826`, strings.Title(resultType), max),
			}
		}
		if max := limits.CommitDiffWithTimeFilterMaxRepos; hasTimeFilter && len(repos) > max {
			return []string{}, &searchAlert{
				prometheusType: "exceeded_diff_commit_with_time_search_limit",
				title:          fmt.Sprintf("Too many matching repositories for %s search to handle", resultType),
				description:    fmt.Sprintf(`%s search can currently only handle searching over %d repositories at a time. Try using the "repo:" filter to narrow down which repositories to search. Tracking issue: https://github.com/sourcegraph/sourcegraph/issues/6826`, strings.Title(resultType), max),
			}
		}
	}

	return resultTypes, nil
}

type aggregator struct {
	mu       sync.Mutex
	results  []SearchResultResolver
	common   searchResultsCommon
	multiErr *multierror.Error
	// fileMatches is a map from git:// URI of the file to FileMatch resolver
	// to merge multiple results of different types for the same file
	fileMatches map[string]*FileMatchResolver
}

func (a *aggregator) report(ctx context.Context, results []SearchResultResolver, common *searchResultsCommon, err error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err != nil && !isContextError(ctx, err) {
		a.multiErr = multierror.Append(a.multiErr, errors.Wrap(err, "repository search failed"))
	}
	if results != nil {
		a.results = append(a.results, results...)
	}
	if common != nil {
		a.common.update(*common)
	}
}

func (a *aggregator) doRepoSearch(ctx context.Context, args *search.TextParameters, limit int32) {
	tr, ctx := trace.New(ctx, "doRepoSearch", "")
	defer func() {
		tr.Finish()
	}()
	repoResults, repoCommon, err := searchRepositories(ctx, args, limit)
	a.report(ctx, repoResults, repoCommon, errors.Wrap(err, "repository search failed"))
}

func (a *aggregator) doSymbolSearch(ctx context.Context, args *search.TextParameters, limit int) {
	tr, ctx := trace.New(ctx, "doSymbolSearch", "")
	defer func() {
		tr.Finish()
	}()
	symbolFileMatches, symbolsCommon, err := searchSymbols(ctx, args, limit)
	a.mu.Lock()
	defer a.mu.Unlock()
	// Timeouts are reported through searchResultsCommon so don't report an error for them
	if err != nil && !isContextError(ctx, err) {
		a.multiErr = multierror.Append(a.multiErr, errors.Wrap(err, "symbol search failed"))
	}
	for _, symbolFileMatch := range symbolFileMatches {
		key := symbolFileMatch.uri
		if m, ok := a.fileMatches[key]; ok {
			m.symbols = symbolFileMatch.symbols
		} else {
			a.fileMatches[key] = symbolFileMatch
			a.results = append(a.results, symbolFileMatch)
		}
	}
	if symbolsCommon != nil {
		a.common.update(*symbolsCommon)
	}
}

func (a *aggregator) doFilePathSearch(ctx context.Context, args *search.TextParameters) {
	tr, ctx := trace.New(ctx, "doFilePathSearch", "")
	defer func() {
		tr.Finish()
	}()
	fileResults, fileCommon, err := searchFilesInRepos(ctx, args)
	a.mu.Lock()
	defer a.mu.Unlock()
	// Timeouts are reported through searchResultsCommon so don't report an error for them
	if err != nil && !isContextError(ctx, err) {
		a.multiErr = multierror.Append(a.multiErr, errors.Wrap(err, "text search failed"))
	}
	if args.PatternInfo.IsStructuralPat && args.PatternInfo.FileMatchLimit == defaultMaxSearchResults && len(fileResults) == 0 && err == nil {
		// No results for structural search? Automatically search again and force Zoekt
		// to resolve more potential file matches by setting a higher FileMatchLimit.
		args.PatternInfo.FileMatchLimit = 1000
		fileResults, fileCommon, err = searchFilesInRepos(ctx, args)
		if len(fileResults) == 0 {
			// Still no results? Give up.
			log15.Warn("Structural search gives up after more exhaustive attempt. Results may have been missed.")
			if fileCommon != nil {
				fileCommon.limitHit = false // Ensure we don't display "Show more".
			}
		}
	}
	for _, r := range fileResults {
		key := r.uri
		m, ok := a.fileMatches[key]
		if ok {
			// TODO(keegan) This looks broken? It isn't merging.
			// merge line match results with an existing symbol result
			m.JLimitHit = m.JLimitHit || r.JLimitHit
			m.JLineMatches = r.JLineMatches
		} else {
			a.fileMatches[key] = r
			a.results = append(a.results, r)
		}
	}
	if fileCommon != nil {
		a.common.update(*fileCommon)
	}
}

func (a *aggregator) doDiffSearch(ctx context.Context, tp *search.TextParameters) {
	tr, ctx := trace.New(ctx, "doDiffSearch", "")
	defer func() {
		tr.Finish()
	}()
	old := tp.PatternInfo
	patternInfo := &search.CommitPatternInfo{
		Pattern:                      old.Pattern,
		IsRegExp:                     old.IsRegExp,
		IsCaseSensitive:              old.IsCaseSensitive,
		FileMatchLimit:               old.FileMatchLimit,
		IncludePatterns:              old.IncludePatterns,
		ExcludePattern:               old.ExcludePattern,
		PathPatternsAreRegExps:       true,
		PathPatternsAreCaseSensitive: tp.PatternInfo.PathPatternsAreCaseSensitive,
	}
	repos, err := getRepos(ctx, tp.RepoPromise)
	if err != nil {
		log15.Warn("doDiffSearch: error while getting repos from promise:", err.Error())
		return
	}

	args := search.TextParametersForCommitParameters{
		PatternInfo: patternInfo,
		Repos:       repos,
		Query:       tp.Query,
	}
	diffResults, diffCommon, err := searchCommitDiffsInRepos(ctx, &args)
	a.report(ctx, diffResults, diffCommon, errors.Wrap(err, "diff search failed"))
}

func (a *aggregator) doCommitSearch(ctx context.Context, tp *search.TextParameters) {
	tr, ctx := trace.New(ctx, "doCommitSearch", "")
	defer func() {
		tr.Finish()
	}()
	old := tp.PatternInfo
	patternInfo := &search.CommitPatternInfo{
		Pattern:                      old.Pattern,
		IsRegExp:                     old.IsRegExp,
		IsCaseSensitive:              old.IsCaseSensitive,
		FileMatchLimit:               old.FileMatchLimit,
		IncludePatterns:              old.IncludePatterns,
		ExcludePattern:               old.ExcludePattern,
		PathPatternsAreRegExps:       true,
		PathPatternsAreCaseSensitive: old.PathPatternsAreCaseSensitive,
	}
	repos, err := getRepos(ctx, tp.RepoPromise)
	if err != nil {
		log15.Warn("doCommitSearch: error while getting repos from promise:", err.Error())
		return
	}

	args := search.TextParametersForCommitParameters{
		PatternInfo: patternInfo,
		Repos:       repos,
		Query:       tp.Query,
	}
	commitResults, commitCommon, err := searchCommitLogInRepos(ctx, &args)
	a.report(ctx, commitResults, commitCommon, errors.Wrap(err, "commit search failed"))
}

// isGlobalSearch returns true if the query contains the filters repo or
// repogroup. For structural queries and queries with version context
// isGlobalSearch always return false.
func (r *searchResolver) isGlobalSearch() bool {
	if r.patternType == query.SearchTypeStructural {
		return false
	}
	if r.versionContext != nil && *r.versionContext != "" {
		return false
	}
	return len(r.query.Values(query.FieldRepo)) == 0 && len(r.query.Values(query.FieldRepoGroup)) == 0
}

// doResults is one of the highest level search functions that handles finding results.
//
// If forceOnlyResultType is specified, only results of the given type are returned,
// regardless of what `type:` is specified in the query string.
//
// Partial results AND an error may be returned.
func (r *searchResolver) doResults(ctx context.Context, forceOnlyResultType string) (_ *SearchResultsResolver, err error) {
	tr, ctx := trace.New(ctx, "doResults", r.rawQuery())
	defer func() {
		tr.SetError(err)
		tr.Finish()
	}()

	start := time.Now()

	ctx, cancel, err := r.withTimeout(ctx)
	if err != nil {
		return nil, err
	}
	defer cancel()

	options := &getPatternInfoOptions{}
	if r.patternType == query.SearchTypeStructural {
		options = &getPatternInfoOptions{performStructuralSearch: true}
		forceOnlyResultType = "file"
	}
	if r.patternType == query.SearchTypeLiteral {
		options = &getPatternInfoOptions{performLiteralSearch: true}
	}
	p, err := r.getPatternInfo(options)
	if err != nil {
		return nil, err
	}

	// Fallback to literal search for searching repos and files if
	// the structural search pattern is empty.
	if r.patternType == query.SearchTypeStructural && p.Pattern == "" {
		r.patternType = query.SearchTypeLiteral
		p.IsStructuralPat = false
		forceOnlyResultType = ""
	}

	args := search.TextParameters{
		PatternInfo:     p,
		Query:           r.query,
		UseFullDeadline: r.searchTimeoutFieldSet(),
		Zoekt:           r.zoekt,
		SearcherURLs:    r.searcherURLs,
		RepoPromise:     &search.Promise{},
	}
	if err := args.PatternInfo.Validate(); err != nil {
		return nil, &badRequestError{err}
	}

	err = validateRepoHasFileUsage(r.query)
	if err != nil {
		return nil, err
	}

	resultTypes := r.determineResultTypes(args, forceOnlyResultType)
	tr.LazyPrintf("resultTypes: %v", resultTypes)
	var (
		requiredWg sync.WaitGroup
		optionalWg sync.WaitGroup
		// Alert is a potential alert shown to the user.
		alert           *searchAlert
		seenResultTypes = make(map[string]struct{})
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

	agg := aggregator{
		common:      searchResultsCommon{maxResultsCount: r.maxResults()},
		fileMatches: make(map[string]*FileMatchResolver),
	}

	isFileOrPath := func() bool {
		for _, rt := range resultTypes {
			if rt == "file" || rt == "path" {
				return true
			}
		}
		return false
	}

	// performance optimization: call zoekt early, resolve repos concurrently, filter
	// search results with resolved repos.
	if r.isGlobalSearch() && isFileOrPath() {
		// to protect us from regression, we explicitly create a child context which is
		// canceled if we return from doResults
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		argsIndexed := args
		argsIndexed.Mode = search.ZoektGlobalSearch
		wg := waitGroup(true)
		wg.Add(1)
		goroutine.Go(func() {
			defer wg.Done()
			agg.doFilePathSearch(ctx, &argsIndexed)
		})
		// On sourcegraph.com and for unscoped queries, determineRepos returns the subset
		// of indexed default repositories. No need to call searcher, because
		// len(searcherRepos) will always be 0.
		if envvar.SourcegraphDotComMode() {
			args.Mode = search.NoFilePath
		} else {
			args.Mode = search.SearcherOnly
		}
	}

	resolved, alertResult, err := r.determineRepos(ctx, tr, start)
	if err != nil {
		return nil, err
	}
	if alertResult != nil {
		return alertResult, nil
	}
	args.RepoPromise.Resolve(resolved.repoRevs)

	agg.mu.Lock()
	agg.common.excluded = resolved.excludedRepos
	agg.mu.Unlock()

	// Apply search limits and generate warnings before firing off workers.
	// This currently limits diff and commit search to a set number of
	// repos, and removes the diff and commit resultTypes if it is breached.
	resultTypes, alert = alertOnSearchLimit(resultTypes, &args)

	searchedFileContentsOrPaths := false
	for _, resultType := range resultTypes {
		resultType := resultType // shadow so it doesn't change in the goroutine
		if _, seen := seenResultTypes[resultType]; seen {
			continue
		}
		seenResultTypes[resultType] = struct{}{}
		switch resultType {
		case "repo":
			wg := waitGroup(true)
			wg.Add(1)
			goroutine.Go(func() {
				defer wg.Done()
				agg.doRepoSearch(ctx, &args, r.maxResults())
			})
		case "symbol":
			wg := waitGroup(len(resultTypes) == 1)
			wg.Add(1)
			goroutine.Go(func() {
				defer wg.Done()
				agg.doSymbolSearch(ctx, &args, int(r.maxResults()))
			})
		case "file", "path":
			if searchedFileContentsOrPaths || args.Mode == search.NoFilePath {
				// type:file and type:path use same searchFilesInRepos, so don't call 2x.
				continue
			}
			searchedFileContentsOrPaths = true
			wg := waitGroup(true)
			wg.Add(1)
			goroutine.Go(func() {
				defer wg.Done()
				agg.doFilePathSearch(ctx, &args)
			})
		case "diff":
			wg := waitGroup(len(resultTypes) == 1)
			wg.Add(1)
			goroutine.Go(func() {
				defer wg.Done()
				agg.doDiffSearch(ctx, &args)
			})
		case "commit":
			wg := waitGroup(len(resultTypes) == 1)
			wg.Add(1)
			goroutine.Go(func() {
				defer wg.Done()
				agg.doCommitSearch(ctx, &args)
			})
		}
	}

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

	tr.LazyPrintf("results=%d limitHit=%v cloning=%d missing=%d excludedFork=%d excludedArchived=%d timedout=%d",
		len(agg.results),
		agg.common.limitHit,
		len(agg.common.cloning),
		len(agg.common.missing),
		agg.common.excluded.forks,
		agg.common.excluded.archived,
		len(agg.common.timedout))

	multiErr, newAlert := alertForStructuralSearch(agg.multiErr)
	if newAlert != nil {
		alert = newAlert // takes higher precedence
	}

	if len(agg.results) == 0 && r.patternType != query.SearchTypeStructural && matchHoleRegexp.MatchString(r.originalQuery) {
		alert = alertForStructuralSearchNotSet(r.originalQuery)
	}

	if len(resolved.missingRepoRevs) > 0 {
		alert = alertForMissingRepoRevs(r.patternType, resolved.missingRepoRevs)
	}

	if len(agg.results) == 0 && strings.Contains(r.originalQuery, `"`) && r.patternType == query.SearchTypeLiteral {
		alert = alertForQuotesInQueryInLiteralMode(r.query.ParseTree())
	}

	// If we have some results, only log the error instead of returning it,
	// because otherwise the client would not receive the partial results
	if len(agg.results) > 0 && multiErr != nil {
		log15.Error("Errors during search", "error", multiErr)
		multiErr = nil
	}

	r.sortResults(ctx, agg.results)

	resultsResolver := SearchResultsResolver{
		start:               start,
		searchResultsCommon: agg.common,
		SearchResults:       agg.results,
		alert:               alert,
	}

	return &resultsResolver, multiErr.ErrorOrNil()
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
	searchResultURIGetter

	ToRepository() (*RepositoryResolver, bool)
	ToFileMatch() (*FileMatchResolver, bool)
	ToCommitSearchResult() (*CommitSearchResultResolver, bool)

	resultCount() int32
}

type searchResultURIGetter interface {
	// SearchResultURIs returns the repo name and file uri respectively
	searchResultURIs() (string, string)
}

// compareSearchResults sorts alphabetically unless one of the filenames is contained in exactFilePatterns,
// in which case exact matches are sorted by length of their file path and then alphabetically.
func compareSearchResults(a, b searchResultURIGetter, exactFilePatterns map[string]struct{}) bool {
	arepo, afile := a.searchResultURIs()
	brepo, bfile := b.searchResultURIs()

	if arepo == brepo {
		if len(exactFilePatterns) == 0 {
			return afile < bfile
		}
		_, aMatch := exactFilePatterns[path.Base(afile)]
		_, bMatch := exactFilePatterns[path.Base(bfile)]
		if aMatch || bMatch {
			if aMatch && bMatch {
				// Prefer shorter file names (ie root files come first)
				if len(afile) != len(bfile) {
					return len(afile) < len(bfile)
				}
				return afile < bfile
			}
			// prefer exact match
			return aMatch
		}
		return afile < bfile
	}
	return arepo < brepo
}

func (r *searchResolver) sortResults(ctx context.Context, results []SearchResultResolver) {
	var exactPatterns map[string]struct{}
	if getBoolPtr(r.userSettings.SearchGlobbing, false) {
		exactPatterns = r.getExactFilePatterns()
	}
	sort.Slice(results, func(i, j int) bool { return compareSearchResults(results[i], results[j], exactPatterns) })
}

// getExactFilePatterns returns the set of file patterns without glob syntax.
func (r *searchResolver) getExactFilePatterns() map[string]struct{} {
	m := map[string]struct{}{}
	query.VisitField(
		r.query.(*query.AndOrQuery).Query,
		query.FieldFile,
		func(value string, negated bool, annotation query.Annotation) {
			originalValue := r.originalQuery[annotation.Range.Start.Column+len(query.FieldFile)+1 : annotation.Range.End.Column]
			if !negated && query.ContainsNoGlobSyntax(originalValue) {
				m[originalValue] = struct{}{}
			}
		})
	return m
}

// orderedFuzzyRegexp interpolate a lazy 'match everything' regexp pattern
// to achieve an ordered fuzzy regexp match.
func orderedFuzzyRegexp(pieces []string) string {
	if len(pieces) == 0 {
		return ""
	}
	if len(pieces) == 1 {
		return pieces[0]
	}
	return "(" + strings.Join(pieces, ").*?(") + ")"
}

// Validates usage of the `repohasfile` filter
func validateRepoHasFileUsage(q query.QueryInfo) error {
	// Query only contains "repohasfile:" and "type:symbol"
	if len(q.Fields()) == 2 && q.Fields()["repohasfile"] != nil && q.Fields()["type"] != nil && len(q.Fields()["type"]) == 1 && q.Fields()["type"][0].Value() == "symbol" {
		return errors.New("repohasfile does not currently return symbol results. Support for symbol results is coming soon. Subscribe to https://github.com/sourcegraph/sourcegraph/issues/4610 for updates")
	}
	return nil
}

// logSlowSearchesThreshold returns the minimum duration configured in site
// settings for logging slow searches.
func logSlowSearchesThreshold() time.Duration {
	ms := conf.Get().ObservabilityLogSlowSearches
	if ms == 0 {
		return time.Duration(math.MaxInt64)
	}
	return time.Duration(ms) * time.Millisecond
}

// mapToLog15Ctx translates a map to log15 context fields.
func mapToLog15Ctx(m map[string]interface{}) []interface{} {
	// sort so its stable
	keys := make([]string, len(m))
	i := 0
	for k := range m {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	ctx := make([]interface{}, len(m)*2)
	for i, k := range keys {
		j := i * 2
		ctx[j] = k
		ctx[j+1] = m[k]
	}
	return ctx
}
