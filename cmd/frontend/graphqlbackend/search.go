package graphqlbackend

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	regexpsyntax "regexp/syntax"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/inconshreveable/log15"
	otlog "github.com/opentracing/opentracing-go/log"

	"github.com/neelance/parallel"
	"github.com/pkg/errors"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/envvar"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/db"
	"github.com/sourcegraph/sourcegraph/internal/endpoint"
	"github.com/sourcegraph/sourcegraph/internal/errcode"
	"github.com/sourcegraph/sourcegraph/internal/gitserver"
	"github.com/sourcegraph/sourcegraph/internal/goroutine"
	"github.com/sourcegraph/sourcegraph/internal/lazyregexp"
	"github.com/sourcegraph/sourcegraph/internal/search"
	searchbackend "github.com/sourcegraph/sourcegraph/internal/search/backend"
	"github.com/sourcegraph/sourcegraph/internal/search/query"
	querytypes "github.com/sourcegraph/sourcegraph/internal/search/query/types"
	"github.com/sourcegraph/sourcegraph/internal/trace"
	"github.com/sourcegraph/sourcegraph/internal/vcs"
	"github.com/sourcegraph/sourcegraph/internal/vcs/git"
	"github.com/sourcegraph/sourcegraph/schema"
)

// This file contains the root resolver for search. It currently has a lot of
// logic that spans out into all the other search_* files.
var mockResolveRepositories func(effectiveRepoFieldValues []string) (resolved resolvedRepositories, err error)

type SearchArgs struct {
	Version        string
	PatternType    *string
	Query          string
	After          *string
	First          *int32
	VersionContext *string
}

type SearchImplementer interface {
	Results(context.Context) (*SearchResultsResolver, error)
	Suggestions(context.Context, *searchSuggestionsArgs) ([]*searchSuggestionResolver, error)
	//lint:ignore U1000 is used by graphql via reflection
	Stats(context.Context) (*searchResultsStats, error)
}

// NewSearchImplementer returns a SearchImplementer that provides search results and suggestions.
func NewSearchImplementer(ctx context.Context, args *SearchArgs) (_ SearchImplementer, err error) {
	tr, ctx := trace.New(ctx, "NewSearchImplementer", args.Query)
	defer func() {
		tr.SetError(err)
		tr.Finish()
	}()
	settings, err := decodedViewerFinalSettings(ctx)
	if err != nil {
		return nil, err
	}

	useNewParser := getBoolPtr(settings.SearchMigrateParser, true)
	tr.LogFields(otlog.Bool("useNewParser", useNewParser))

	searchType, err := detectSearchType(args.Version, args.PatternType)
	if err != nil {
		return nil, err
	}
	searchType = overrideSearchType(args.Query, searchType, useNewParser)

	if searchType == query.SearchTypeStructural && !conf.StructuralSearchEnabled() {
		return nil, errors.New("Structural search is disabled in the site configuration.")
	}

	var queryInfo query.QueryInfo
	if useNewParser {
		globbing := getBoolPtr(settings.SearchGlobbing, false)
		tr.LogFields(otlog.Bool("globbing", globbing))
		queryInfo, err = query.ProcessAndOr(args.Query, query.ParserOptions{SearchType: searchType, Globbing: globbing})
		if err != nil {
			return alertForQuery(args.Query, err), nil
		}
		if getBoolPtr(settings.SearchUppercase, false) {
			q := queryInfo.(*query.AndOrQuery)
			q.Query = query.SearchUppercase(q.Query)
		}
	} else {
		var queryString string
		if searchType == query.SearchTypeLiteral {
			queryString = query.ConvertToLiteral(args.Query)
		} else {
			queryString = args.Query
		}
		queryInfo, err = query.Process(queryString, searchType)
		if err != nil {
			return alertForQuery(queryString, err), nil
		}
	}
	tr.LazyPrintf("parsing done")

	// If stable:truthy is specified, make the query return a stable result ordering.
	if queryInfo.BoolValue(query.FieldStable) {
		args, queryInfo, err = queryForStableResults(args, queryInfo)
		if err != nil {
			return alertForQuery(args.Query, err), nil
		}
	}

	// If the request is a paginated one, decode those arguments now.
	var pagination *searchPaginationInfo
	if args.First != nil {
		pagination, err = processPaginationRequest(args, queryInfo)
		if err != nil {
			return nil, err
		}
	}

	return &searchResolver{
		query:          queryInfo,
		originalQuery:  args.Query,
		versionContext: args.VersionContext,
		userSettings:   settings,
		pagination:     pagination,
		patternType:    searchType,
		zoekt:          search.Indexed(),
		searcherURLs:   search.SearcherURLs(),
	}, nil
}

func (r *schemaResolver) Search(ctx context.Context, args *SearchArgs) (SearchImplementer, error) {
	return NewSearchImplementer(ctx, args)
}

// queryForStableResults transforms a query that returns a stable result
// ordering. The transformed query uses pagination underneath the hood.
func queryForStableResults(args *SearchArgs, queryInfo query.QueryInfo) (*SearchArgs, query.QueryInfo, error) {
	if queryInfo.BoolValue(query.FieldStable) {
		var stableResultCount int32
		if _, countPresent := queryInfo.Fields()["count"]; countPresent {
			count, _ := queryInfo.StringValue(query.FieldCount)
			count64, err := strconv.ParseInt(count, 10, 32)
			if err != nil {
				return nil, nil, err
			}
			stableResultCount = int32(count64)
			if stableResultCount > maxSearchResultsPerPaginatedRequest {
				return nil, nil, fmt.Errorf("Stable searches are limited to at max count:%d results. Consider removing 'stable:', narrowing the search with 'repo:', or using the paginated search API.", maxSearchResultsPerPaginatedRequest)
			}
		} else {
			stableResultCount = defaultMaxSearchResults
		}
		args.First = &stableResultCount
		fileValue := "file"
		// Pagination only works for file content searches, and will
		// raise an error otherwise. If stable is explicitly set, this
		// is implied. So, force this query to only return file content
		// results.
		queryInfo.Fields()["type"] = []*querytypes.Value{{String: &fileValue}}
	}
	return args, queryInfo, nil
}

func processPaginationRequest(args *SearchArgs, queryInfo query.QueryInfo) (*searchPaginationInfo, error) {
	var pagination *searchPaginationInfo
	if args.First != nil {
		cursor, err := unmarshalSearchCursor(args.After)
		if err != nil {
			return nil, err
		}
		if *args.First < 0 || *args.First > maxSearchResultsPerPaginatedRequest {
			return nil, fmt.Errorf("search: requested pagination 'first' value outside allowed range (0 - %d)", maxSearchResultsPerPaginatedRequest)
		}
		pagination = &searchPaginationInfo{
			cursor: cursor,
			limit:  *args.First,
		}
	} else if args.After != nil {
		return nil, errors.New("search: paginated requests providing an 'after' cursor but no 'first' value is forbidden")
	}
	return pagination, nil
}

// detectSearchType returns the search type to perfrom ("regexp", or
// "literal"). The search type derives from three sources: the version and
// patternType parameters passed to the search endpoint (literal search is the
// default in V2), and the `patternType:` filter in the input query string which
// overrides the searchType, if present.
func detectSearchType(version string, patternType *string) (query.SearchType, error) {
	var searchType query.SearchType
	if patternType != nil {
		switch *patternType {
		case "literal":
			searchType = query.SearchTypeLiteral
		case "regexp":
			searchType = query.SearchTypeRegex
		case "structural":
			searchType = query.SearchTypeStructural
		default:
			return -1, fmt.Errorf("unrecognized patternType: %v", patternType)
		}
	} else {
		switch version {
		case "V1":
			searchType = query.SearchTypeRegex
		case "V2":
			searchType = query.SearchTypeLiteral
		default:
			return -1, fmt.Errorf("unrecognized version: %v", version)
		}
	}
	return searchType, nil
}

var patternTypeRegex = lazyregexp.New(`(?i)patterntype:([a-zA-Z"']+)`)

func overrideSearchType(input string, searchType query.SearchType, useNewParser bool) query.SearchType {
	if useNewParser {
		q, err := query.ParseAndOr(input, query.SearchTypeLiteral)
		q = query.LowercaseFieldNames(q)
		if err != nil {
			// If parsing fails, return the default search type. Any actual
			// parse errors will be raised by subsequent parser calls.
			return searchType
		}
		query.VisitField(q, "patterntype", func(value string, _ bool, _ query.Annotation) {
			switch value {
			case "regex", "regexp":
				searchType = query.SearchTypeRegex
			case "literal":
				searchType = query.SearchTypeLiteral
			case "structural":
				searchType = query.SearchTypeStructural
			}
		})
	} else {
		// The patterntype field is Singular, but not enforced since we do not
		// properly parse the input. The regex extraction, takes the left-most
		// "patterntype:value" match.
		patternFromField := patternTypeRegex.FindStringSubmatch(input)
		if len(patternFromField) > 1 {
			extracted := patternFromField[1]
			if match, _ := regexp.MatchString("regex", extracted); match {
				searchType = query.SearchTypeRegex
			} else if match, _ := regexp.MatchString("literal", extracted); match {
				searchType = query.SearchTypeLiteral

			} else if match, _ := regexp.MatchString("structural", extracted); match {
				searchType = query.SearchTypeStructural
			}
		}
	}
	return searchType
}

func getBoolPtr(b *bool, def bool) bool {
	if b == nil {
		return def
	}
	return *b
}

type resolvedRepositories struct {
	repoRevs        []*search.RepositoryRevisions
	missingRepoRevs []*search.RepositoryRevisions
	excludedRepos   excludedRepos
	overLimit       bool
}

// searchResolver is a resolver for the GraphQL type `Search`
type searchResolver struct {
	query               query.QueryInfo       // the query, either containing and/or expressions or otherwise ordinary
	originalQuery       string                // the raw string of the original search query
	pagination          *searchPaginationInfo // pagination information, or nil if the request is not paginated.
	patternType         query.SearchType
	versionContext      *string
	userSettings        *schema.Settings
	invalidateRepoCache bool // if true, invalidates the repo cache when evaluating search subexpressions.

	// Cached resolveRepositories results.
	reposMu  sync.Mutex
	resolved resolvedRepositories
	repoErr  error

	zoekt        *searchbackend.Zoekt
	searcherURLs *endpoint.Map
}

// rawQuery returns the original query string input.
func (r *searchResolver) rawQuery() string {
	return r.originalQuery
}

func (r *searchResolver) countIsSet() bool {
	count, _ := r.query.StringValues(query.FieldCount)
	max, _ := r.query.StringValues(query.FieldMax)
	return len(count) > 0 || len(max) > 0
}

const defaultMaxSearchResults = 30
const maxSearchResultsPerPaginatedRequest = 5000

func (r *searchResolver) maxResults() int32 {
	if r.pagination != nil {
		// Paginated search requests always consume an entire result set for a
		// given repository, so we do not want any limit here. See
		// search_pagination.go for details on why this is necessary .
		return math.MaxInt32
	}
	count, _ := r.query.StringValues(query.FieldCount)
	if len(count) > 0 {
		n, _ := strconv.Atoi(count[0])
		if n > 0 {
			return int32(n)
		}
	}
	max, _ := r.query.StringValues(query.FieldMax)
	if len(max) > 0 {
		n, _ := strconv.Atoi(max[0])
		if n > 0 {
			return int32(n)
		}
	}
	return defaultMaxSearchResults
}

var mockDecodedViewerFinalSettings *schema.Settings

func decodedViewerFinalSettings(ctx context.Context) (_ *schema.Settings, err error) {
	tr, ctx := trace.New(ctx, "decodedViewerFinalSettings", "")
	defer func() {
		tr.SetError(err)
		tr.Finish()
	}()
	if mockDecodedViewerFinalSettings != nil {
		return mockDecodedViewerFinalSettings, nil
	}
	merged, err := viewerFinalSettings(ctx)
	if err != nil {
		return nil, err
	}
	var settings schema.Settings
	if err := json.Unmarshal([]byte(merged.Contents()), &settings); err != nil {
		return nil, err
	}
	return &settings, nil
}

// A repogroup value is either a exact repo path RepoPath, or a regular
// expression pattern RepoRegexpPattern.
type RepoGroupValue interface {
	value()
	String() string
}

type RepoPath string
type RepoRegexpPattern string

func (RepoPath) value() {}
func (r RepoPath) String() string {
	return string(r)
}

func (RepoRegexpPattern) value() {}
func (r RepoRegexpPattern) String() string {
	return string(r)
}

var mockResolveRepoGroups func() (map[string][]RepoGroupValue, error)

func resolveRepoGroups(settings *schema.Settings) (groups map[string][]RepoGroupValue, err error) {
	if mockResolveRepoGroups != nil {
		return mockResolveRepoGroups()
	}
	groups = map[string][]RepoGroupValue{}

	for name, values := range settings.SearchRepositoryGroups {
		repos := make([]RepoGroupValue, 0, len(values))

		for _, value := range values {
			switch path := value.(type) {
			case string:
				repos = append(repos, RepoPath(path))
			case map[string]interface{}:
				if stringRegex, ok := path["regex"].(string); ok {
					repos = append(repos, RepoRegexpPattern(stringRegex))
				} else {
					log15.Warn("ignoring repo group value because regex not specfied", "regex-string", path["regex"])
				}
			default:
				log15.Warn("ignoring repo group value of unrecognized type", "value", value, "type", fmt.Sprintf("%T", value))
			}
		}
		groups[name] = repos
	}
	return groups, nil
}

// repoGroupValuesToRegexp does a lookup of all repo groups by name and converts
// their values to a list of regular expressions to search.
func repoGroupValuesToRegexp(groupNames []string, groups map[string][]RepoGroupValue) []string {
	var patterns []string
	for _, groupName := range groupNames {
		for _, value := range groups[groupName] {
			switch v := value.(type) {
			case RepoPath:
				patterns = append(patterns, "^"+regexp.QuoteMeta(v.String())+"$")
			case RepoRegexpPattern:
				patterns = append(patterns, v.String())
			default:
				panic("unreachable")
			}
		}
	}
	return patterns
}

// NOTE: This function is not called if the version context is not used
func resolveVersionContext(versionContext string) (*schema.VersionContext, error) {
	for _, vc := range conf.Get().ExperimentalFeatures.VersionContexts {
		if vc.Name == versionContext {
			return vc, nil
		}
	}

	return nil, errors.New("version context not found")
}

// Cf. golang/go/src/regexp/syntax/parse.go.
const regexpFlags = regexpsyntax.ClassNL | regexpsyntax.PerlX | regexpsyntax.UnicodeGroups

// exactlyOneRepo returns whether exactly one repo: literal field is specified and
// delineated by regex anchors ^ and $. This function helps determine whether we
// should return results for a single repo regardless of whether it is a fork or
// archive.
func exactlyOneRepo(repoFilters []string) bool {
	if len(repoFilters) == 1 {
		filter, _ := search.ParseRepositoryRevisions(repoFilters[0])
		if strings.HasPrefix(filter, "^") && strings.HasSuffix(filter, "$") {
			filter := strings.TrimSuffix(strings.TrimPrefix(filter, "^"), "$")
			r, err := regexpsyntax.Parse(filter, regexpFlags)
			if err != nil {
				return false
			}
			return r.Op == regexpsyntax.OpLiteral
		}
	}
	return false
}

// A type that counts how many repos with a certain label were excluded from search results.
type excludedRepos struct {
	forks    int
	archived int
}

// computeExcludedRepositories returns a list of excluded repositories (forks or
// archives) based on the search query.
func computeExcludedRepositories(ctx context.Context, q query.QueryInfo, op db.ReposListOptions) (excluded excludedRepos) {
	if q == nil {
		return excludedRepos{}
	}

	// PERF: We query concurrently since each count call can be slow on
	// Sourcegraph.com (100ms+).
	var wg sync.WaitGroup
	var numExcludedForks, numExcludedArchived int

	forkStr, _ := q.StringValue(query.FieldFork)
	fork := parseYesNoOnly(forkStr)
	if fork == Invalid && !exactlyOneRepo(op.IncludePatterns) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// 'fork:...' was not specified and forks are excluded, find out
			// which repos are excluded.
			selectForks := op
			selectForks.OnlyForks = true
			selectForks.NoForks = false
			var err error
			numExcludedForks, err = db.Repos.Count(ctx, selectForks)
			if err != nil {
				log15.Warn("repo count for excluded fork", "err", err)
			}
		}()
	}

	archivedStr, _ := q.StringValue(query.FieldArchived)
	archived := parseYesNoOnly(archivedStr)
	if archived == Invalid && !exactlyOneRepo(op.IncludePatterns) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// archived...: was not specified and archives are excluded,
			// find out which repos are excluded.
			selectArchived := op
			selectArchived.OnlyArchived = true
			selectArchived.NoArchived = false
			var err error
			numExcludedArchived, err = db.Repos.Count(ctx, selectArchived)
			if err != nil {
				log15.Warn("repo count for excluded archive", "err", err)
			}
		}()
	}

	wg.Wait()

	return excludedRepos{forks: numExcludedForks, archived: numExcludedArchived}
}

// resolveRepositories calls doResolveRepositories, caching the result for the common
// case where effectiveRepoFieldValues == nil.
func (r *searchResolver) resolveRepositories(ctx context.Context, effectiveRepoFieldValues []string) (resolvedRepositories, error) {
	var err error
	var repoRevs, missingRepoRevs []*search.RepositoryRevisions
	var overLimit bool
	if mockResolveRepositories != nil {
		return mockResolveRepositories(effectiveRepoFieldValues)
	}

	tr, ctx := trace.New(ctx, "graphql.resolveRepositories", fmt.Sprintf("effectiveRepoFieldValues: %v", effectiveRepoFieldValues))
	defer func() {
		if err != nil {
			tr.SetError(err)
		} else {
			tr.LazyPrintf("numRepoRevs: %d, numMissingRepoRevs: %d, overLimit: %v", len(repoRevs), len(missingRepoRevs), overLimit)
		}
		tr.Finish()
	}()
	if effectiveRepoFieldValues == nil {
		r.reposMu.Lock()
		defer r.reposMu.Unlock()
		if r.resolved.repoRevs != nil || r.resolved.missingRepoRevs != nil || r.repoErr != nil {
			tr.LazyPrintf("cached")
			return r.resolved, r.repoErr
		}
	}

	repoFilters, minusRepoFilters := r.query.RegexpPatterns(query.FieldRepo)
	if effectiveRepoFieldValues != nil {
		repoFilters = effectiveRepoFieldValues
	}
	repoGroupFilters, _ := r.query.StringValues(query.FieldRepoGroup)

	var settingForks, settingArchived bool
	if v := r.userSettings.SearchIncludeForks; v != nil {
		settingForks = *v
	}
	if v := r.userSettings.SearchIncludeArchived; v != nil {
		settingArchived = *v
	}

	forkStr, _ := r.query.StringValue(query.FieldFork)
	fork := parseYesNoOnly(forkStr)
	if fork == Invalid && !exactlyOneRepo(repoFilters) && !settingForks {
		// fork defaults to No unless either of:
		// (1) exactly one repo is being searched, or
		// (2) user/org/global setting includes forks
		fork = No
	}

	archivedStr, _ := r.query.StringValue(query.FieldArchived)
	archived := parseYesNoOnly(archivedStr)
	if archived == Invalid && !exactlyOneRepo(repoFilters) && !settingArchived {
		// archived defaults to No unless either of:
		// (1) exactly one repo is being searched, or
		// (2) user/org/global setting includes archives in all searches
		archived = No
	}

	visibilityStr, _ := r.query.StringValue(query.FieldVisibility)
	visibility := query.ParseVisibility(visibilityStr)

	commitAfter, _ := r.query.StringValue(query.FieldRepoHasCommitAfter)

	var versionContextName string
	if r.versionContext != nil {
		versionContextName = *r.versionContext
	}

	tr.LazyPrintf("resolveRepositories - start")
	options := resolveRepoOp{
		repoFilters:        repoFilters,
		minusRepoFilters:   minusRepoFilters,
		repoGroupFilters:   repoGroupFilters,
		versionContextName: versionContextName,
		userSettings:       r.userSettings,
		onlyForks:          fork == Only,
		noForks:            fork == No,
		onlyArchived:       archived == Only,
		noArchived:         archived == No,
		onlyPrivate:        visibility == query.Private,
		onlyPublic:         visibility == query.Public,
		commitAfter:        commitAfter,
		query:              r.query,
	}
	resolved, err := resolveRepositories(ctx, options)
	tr.LazyPrintf("resolveRepositories - done")
	if effectiveRepoFieldValues == nil {
		r.resolved = resolved
		r.repoErr = err
	}
	return resolved, err
}

// a patternRevspec maps an include pattern to a list of revisions
// for repos matching that pattern. "map" in this case does not mean
// an actual map, because we want regexp matches, not identity matches.
type patternRevspec struct {
	includePattern *regexp.Regexp
	revs           []search.RevisionSpecifier
}

// given a repo name, determine whether it matched any patterns for which we have
// revspecs (or ref globs), and if so, return the matching/allowed ones.
func getRevsForMatchedRepo(repo api.RepoName, pats []patternRevspec) (matched []search.RevisionSpecifier, clashing []search.RevisionSpecifier) {
	revLists := make([][]search.RevisionSpecifier, 0, len(pats))
	for _, rev := range pats {
		if rev.includePattern.MatchString(string(repo)) {
			revLists = append(revLists, rev.revs)
		}
	}
	// exactly one match: we accept that list
	if len(revLists) == 1 {
		matched = revLists[0]
		return
	}
	// no matches: we generate a dummy list containing only master
	if len(revLists) == 0 {
		matched = []search.RevisionSpecifier{{RevSpec: ""}}
		return
	}
	// if two repo specs match, and both provided non-empty rev lists,
	// we want their intersection
	allowedRevs := make(map[search.RevisionSpecifier]struct{}, len(revLists[0]))
	allRevs := make(map[search.RevisionSpecifier]struct{}, len(revLists[0]))
	// starting point: everything is "true" if it is currently allowed
	for _, rev := range revLists[0] {
		allowedRevs[rev] = struct{}{}
		allRevs[rev] = struct{}{}
	}
	// in theory, "master-by-default" entries won't even be participating
	// in this.
	for _, revList := range revLists[1:] {
		restrictedRevs := make(map[search.RevisionSpecifier]struct{}, len(revList))
		for _, rev := range revList {
			allRevs[rev] = struct{}{}
			if _, ok := allowedRevs[rev]; ok {
				restrictedRevs[rev] = struct{}{}
			}
		}
		allowedRevs = restrictedRevs
	}
	if len(allowedRevs) > 0 {
		matched = make([]search.RevisionSpecifier, 0, len(allowedRevs))
		for rev := range allowedRevs {
			matched = append(matched, rev)
		}
		sort.Slice(matched, func(i, j int) bool { return matched[i].Less(matched[j]) })
		return
	}
	// build a list of the revspecs which broke this, return it
	// as the "clashing" list.
	clashing = make([]search.RevisionSpecifier, 0, len(allRevs))
	for rev := range allRevs {
		clashing = append(clashing, rev)
	}
	// ensure that lists are always returned in sorted order.
	sort.Slice(clashing, func(i, j int) bool { return clashing[i].Less(clashing[j]) })
	return
}

// findPatternRevs mutates the given list of include patterns to
// be a raw list of the repository name patterns we want, separating
// out their revision specs, if any.
func findPatternRevs(includePatterns []string) (includePatternRevs []patternRevspec, err error) {
	includePatternRevs = make([]patternRevspec, 0, len(includePatterns))
	for i, includePattern := range includePatterns {
		repoPattern, revs := search.ParseRepositoryRevisions(includePattern)
		// Validate pattern now so the error message is more recognizable to the
		// user
		if _, err := regexp.Compile(repoPattern); err != nil {
			return nil, &badRequestError{err}
		}
		repoPattern = optimizeRepoPatternWithHeuristics(repoPattern)
		includePatterns[i] = repoPattern
		if len(revs) > 0 {
			p, err := regexp.Compile("(?i:" + includePatterns[i] + ")")
			if err != nil {
				return nil, &badRequestError{err}
			}
			patternRev := patternRevspec{includePattern: p, revs: revs}
			includePatternRevs = append(includePatternRevs, patternRev)
		}
	}
	return
}

type resolveRepoOp struct {
	repoFilters        []string
	minusRepoFilters   []string
	repoGroupFilters   []string
	versionContextName string
	userSettings       *schema.Settings
	noForks            bool
	onlyForks          bool
	noArchived         bool
	onlyArchived       bool
	commitAfter        string
	onlyPrivate        bool
	onlyPublic         bool
	query              query.QueryInfo
}

func (op *resolveRepoOp) String() string {
	var b strings.Builder
	if len(op.repoFilters) == 0 {
		b.WriteString("r=[]")
	}
	for i, r := range op.repoFilters {
		if i != 0 {
			b.WriteByte(' ')
		}
		b.WriteString(strconv.Quote(r))
	}

	if len(op.minusRepoFilters) > 0 {
		_, _ = fmt.Fprintf(&b, " -r=%v", op.minusRepoFilters)
	}
	if len(op.repoGroupFilters) > 0 {
		_, _ = fmt.Fprintf(&b, " groups=%v", op.repoGroupFilters)
	}
	if op.versionContextName != "" {
		_, _ = fmt.Fprintf(&b, " versionContext=%q", op.versionContextName)
	}
	if op.commitAfter != "" {
		_, _ = fmt.Fprintf(&b, " commitAfter=%q", op.commitAfter)
	}

	if op.noForks {
		b.WriteString(" noForks")
	}
	if op.onlyForks {
		b.WriteString(" onlyForks")
	}
	if op.noArchived {
		b.WriteString(" noArchived")
	}
	if op.onlyArchived {
		b.WriteString(" onlyArchived")
	}
	if op.onlyPrivate {
		b.WriteString(" onlyPrivate")
	}
	if op.onlyPublic {
		b.WriteString(" onlyPublic")
	}

	return b.String()
}

func searchLimits() schema.SearchLimits {
	// Our configuration reader does not set defaults from schema. So we rely
	// on Go default values to mean defaults.
	withDefault := func(x *int, def int) {
		if *x <= 0 {
			*x = def
		}
	}

	c := conf.Get()

	var limits schema.SearchLimits
	if c.SearchLimits != nil {
		limits = *c.SearchLimits
	}

	// If MaxRepos unset use deprecated value
	if limits.MaxRepos == 0 {
		limits.MaxRepos = c.MaxReposToSearch
	}

	withDefault(&limits.MaxRepos, math.MaxInt32>>1)
	withDefault(&limits.CommitDiffMaxRepos, 50)
	withDefault(&limits.CommitDiffWithTimeFilterMaxRepos, 10000)
	withDefault(&limits.MaxTimeoutSeconds, 60)

	return limits
}

func resolveRepositories(ctx context.Context, op resolveRepoOp) (resolvedRepositories, error) {
	var err error
	tr, ctx := trace.New(ctx, "resolveRepositories", op.String())
	defer func() {
		tr.SetError(err)
		tr.Finish()
	}()

	includePatterns := op.repoFilters
	if includePatterns != nil {
		// Copy to avoid race condition.
		includePatterns = append([]string{}, includePatterns...)
	}

	excludePatterns := op.minusRepoFilters

	maxRepoListSize := searchLimits().MaxRepos

	// If any repo groups are specified, take the intersection of the repo
	// groups and the set of repos specified with repo:. (If none are specified
	// with repo:, then include all from the group.)
	if groupNames := op.repoGroupFilters; len(groupNames) > 0 {
		groups, err := resolveRepoGroups(op.userSettings)
		if err != nil {
			return resolvedRepositories{}, err
		}
		patterns := repoGroupValuesToRegexp(groupNames, groups)
		tr.LazyPrintf("repogroups: adding %d repos to include pattern", len(patterns))
		includePatterns = append(includePatterns, unionRegExps(patterns))

		// Ensure we don't omit any repos explicitly included via a repo group. (Each explicitly
		// listed repo generates at least one pattern.)
		if len(patterns) > maxRepoListSize {
			maxRepoListSize = len(patterns)
		}
	}

	// note that this mutates the strings in includePatterns, stripping their
	// revision specs, if they had any.
	includePatternRevs, err := findPatternRevs(includePatterns)
	if err != nil {
		return resolvedRepositories{}, err
	}

	// If a version context is specified, gather the list of repository names
	// to limit the results to these repositories.
	var versionContextRepositories []string
	var versionContext *schema.VersionContext
	// If a ref is specified we skip using version contexts.
	if len(includePatternRevs) == 0 && op.versionContextName != "" {
		versionContext, err = resolveVersionContext(op.versionContextName)
		if err != nil {
			return resolvedRepositories{}, err
		}

		for _, revision := range versionContext.Revisions {
			versionContextRepositories = append(versionContextRepositories, revision.Repo)
		}
	}

	var defaultRepos []*types.Repo
	if envvar.SourcegraphDotComMode() && len(includePatterns) == 0 {
		start := time.Now()
		defaultRepos, err = defaultRepositories(ctx, db.DefaultRepos.List, search.Indexed(), excludePatterns)
		if err != nil {
			return resolvedRepositories{}, errors.Wrap(err, "getting list of default repos")
		}
		tr.LazyPrintf("defaultrepos: took %s to add %d repos", time.Since(start), len(defaultRepos))

		// Search all default repos since indexed search is fast.
		if len(defaultRepos) > maxRepoListSize {
			maxRepoListSize = len(defaultRepos)
		}
	}

	var repos []*types.Repo
	var excluded excludedRepos
	if len(defaultRepos) > 0 {
		repos = defaultRepos
		if len(repos) > maxRepoListSize {
			repos = repos[:maxRepoListSize]
		}
	} else {
		tr.LazyPrintf("Repos.List - start")
		options := db.ReposListOptions{
			OnlyRepoIDs:     true,
			IncludePatterns: includePatterns,
			Names:           versionContextRepositories,
			ExcludePattern:  unionRegExps(excludePatterns),
			// List N+1 repos so we can see if there are repos omitted due to our repo limit.
			LimitOffset:  &db.LimitOffset{Limit: maxRepoListSize + 1},
			NoForks:      op.noForks,
			OnlyForks:    op.onlyForks,
			NoArchived:   op.noArchived,
			OnlyArchived: op.onlyArchived,
			NoPrivate:    op.onlyPublic,
			OnlyPrivate:  op.onlyPrivate,
		}

		// PERF: We query concurrently since Count and List call can be slow
		// on Sourcegraph.com (100ms+).
		excludedC := make(chan excludedRepos)
		go func() {
			excludedC <- computeExcludedRepositories(ctx, op.query, options)
		}()

		repos, err = db.Repos.List(ctx, options)
		tr.LazyPrintf("Repos.List - done")

		excluded = <-excludedC
		tr.LazyPrintf("excluded repos: %+v", excluded)

		if err != nil {
			return resolvedRepositories{}, err
		}
	}
	overLimit := len(repos) >= maxRepoListSize
	repoRevs := make([]*search.RepositoryRevisions, 0, len(repos))
	var missingRepoRevs []*search.RepositoryRevisions
	tr.LazyPrintf("Associate/validate revs - start")

	for _, repo := range repos {
		var repoRev search.RepositoryRevisions
		var revs []search.RevisionSpecifier
		// versionContext will be nil if the query contains revision specifiers
		if versionContext != nil {
			for _, vcRepoRev := range versionContext.Revisions {
				if vcRepoRev.Repo == string(repo.Name) {
					repoRev.Repo = repo
					revs = append(revs, search.RevisionSpecifier{RevSpec: vcRepoRev.Rev})
				}
			}
		} else {
			var clashingRevs []search.RevisionSpecifier
			revs, clashingRevs = getRevsForMatchedRepo(repo.Name, includePatternRevs)
			repoRev.Repo = repo
			// if multiple specified revisions clash, report this usefully:
			if len(revs) == 0 && clashingRevs != nil {
				missingRepoRevs = append(missingRepoRevs, &search.RepositoryRevisions{
					Repo: repo,
					Revs: clashingRevs,
				})
			}
		}

		// We do in place filtering to reduce allocations. Common path is no
		// filtering of revs.
		if len(revs) > 0 {
			repoRev.Revs = revs[:0]
		}

		// Check if the repository actually has the revisions that the user specified.
		for _, rev := range revs {
			if rev.RefGlob != "" || rev.ExcludeRefGlob != "" {
				// Do not validate ref patterns. A ref pattern matching 0 refs is not necessarily
				// invalid, so it's not clear what validation would even mean.
				repoRev.Revs = append(repoRev.Revs, rev)
				continue
			}
			if rev.RevSpec == "" { // skip default branch resolution to save time
				repoRev.Revs = append(repoRev.Revs, rev)
				continue
			}

			// Validate the revspec.
			// Do not trigger a repo-updater lookup (e.g.,
			// backend.{GitRepo,Repos.ResolveRev}) because that would slow this operation
			// down by a lot (if we're looping over many repos). This means that it'll fail if a
			// repo is not on gitserver.
			//
			// TODO(sqs): make this NOT send gitserver this revspec in EnsureRevision, to avoid
			// searches like "repo:@foobar" (where foobar is an invalid revspec on most repos)
			// taking a long time because they all ask gitserver to try to fetch from the remote
			// repo.
			trimmedRefSpec := strings.TrimPrefix(rev.RevSpec, "^") // handle negated revisions, such as ^<branch>, ^<tag>, or ^<commit>
			if _, err := git.ResolveRevision(ctx, repoRev.GitserverRepo(), nil, trimmedRefSpec, git.ResolveRevisionOptions{NoEnsureRevision: true}); err != nil {
				if errors.Is(err, context.DeadlineExceeded) {
					return resolvedRepositories{}, context.DeadlineExceeded
				}
				if errors.As(err, &git.BadCommitError{}) {
					return resolvedRepositories{}, err
				}
				if gitserver.IsRevisionNotFound(err) {
					// The revspec does not exist, so don't include it, and report that it's missing.
					if rev.RevSpec == "" {
						// Report as HEAD not "" (empty string) to avoid user confusion.
						rev.RevSpec = "HEAD"
					}
					missingRepoRevs = append(missingRepoRevs, &search.RepositoryRevisions{
						Repo: repo,
						Revs: []search.RevisionSpecifier{{RevSpec: rev.RevSpec}},
					})
				}
				// If err != nil and is not one of the err values checked for above, cloning and other errors will be handled later, so just ignore an error
				// if there is one.
				continue
			}
			repoRev.Revs = append(repoRev.Revs, rev)
		}
		repoRevs = append(repoRevs, &repoRev)
	}

	tr.LazyPrintf("Associate/validate revs - done")

	if op.commitAfter != "" {
		start := time.Now()
		before := len(repoRevs)
		repoRevs, err = filterRepoHasCommitAfter(ctx, repoRevs, op.commitAfter)
		tr.LazyPrintf("repohascommitafter removed %d repos in %s", before-len(repoRevs), time.Since(start))
	}

	return resolvedRepositories{
		repoRevs:        repoRevs,
		missingRepoRevs: missingRepoRevs,
		excludedRepos:   excluded,
		overLimit:       overLimit,
	}, err
}

type defaultReposFunc func(ctx context.Context) ([]*types.Repo, error)

func defaultRepositories(ctx context.Context, getRawDefaultRepos defaultReposFunc, z *searchbackend.Zoekt, excludePatterns []string) ([]*types.Repo, error) {
	// Get the list of default repos from the db.
	defaultRepos, err := getRawDefaultRepos(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "querying db for default repos")
	}

	// Remove excluded repos
	if len(excludePatterns) > 0 {
		patterns, _ := regexp.Compile(`(?i)` + unionRegExps(excludePatterns))
		filteredRepos := defaultRepos[:0]
		for _, repo := range defaultRepos {
			if matched := patterns.MatchString(string(repo.Name)); !matched {
				filteredRepos = append(filteredRepos, repo)
			}
		}
		defaultRepos = filteredRepos
	}

	// Ask Zoekt which repos it has indexed
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	set, err := z.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	// In place filtering of defaultRepos to only include names from set.
	repos := defaultRepos[:0]
	for _, r := range defaultRepos {
		if _, ok := set[string(r.Name)]; ok {
			repos = append(repos, r)
		}
	}

	return repos, nil
}

func filterRepoHasCommitAfter(ctx context.Context, revisions []*search.RepositoryRevisions, after string) ([]*search.RepositoryRevisions, error) {
	var (
		mut  sync.Mutex
		pass = []*search.RepositoryRevisions{}
		res  = make(chan *search.RepositoryRevisions, 100)
		run  = parallel.NewRun(128)
	)

	goroutine.Go(func() {
		for rev := range res {
			if len(rev.Revs) != 0 {
				mut.Lock()
				pass = append(pass, rev)
				mut.Unlock()
			}
			run.Release()
		}
	})

	for _, revs := range revisions {
		run.Acquire()

		revs := revs
		goroutine.Go(func() {
			var specifiers []search.RevisionSpecifier
			for _, rev := range revs.Revs {
				ok, err := git.HasCommitAfter(ctx, revs.GitserverRepo(), after, rev.RevSpec)
				if err != nil {
					if gitserver.IsRevisionNotFound(err) || vcs.IsRepoNotExist(err) {
						continue
					}

					run.Error(err)
					continue
				}
				if ok {
					specifiers = append(specifiers, rev)
				}
			}
			res <- &search.RepositoryRevisions{Repo: revs.Repo, Revs: specifiers}
		})
	}

	err := run.Wait()
	close(res)

	return pass, err
}

func optimizeRepoPatternWithHeuristics(repoPattern string) string {
	if envvar.SourcegraphDotComMode() && (strings.HasPrefix(string(repoPattern), "github.com") || strings.HasPrefix(string(repoPattern), `github\.com`)) {
		repoPattern = "^" + repoPattern
	}
	// Optimization: make the "." in "github.com" a literal dot
	// so that the regexp can be optimized more effectively.
	repoPattern = strings.Replace(string(repoPattern), "github.com", `github\.com`, -1)
	return repoPattern
}

func (r *searchResolver) suggestFilePaths(ctx context.Context, limit int) ([]*searchSuggestionResolver, error) {
	resolved, err := r.resolveRepositories(ctx, nil)
	if err != nil {
		return nil, err
	}

	if resolved.overLimit {
		// If we've exceeded the repo limit, then we may miss files from repos we care
		// about, so don't bother searching filenames at all.
		return nil, nil
	}

	p, err := r.getPatternInfo(&getPatternInfoOptions{forceFileSearch: true})
	if err != nil {
		return nil, err
	}

	args := search.TextParameters{
		PatternInfo:     p,
		RepoPromise:     (&search.Promise{}).Resolve(resolved.repoRevs),
		Query:           r.query,
		UseFullDeadline: r.searchTimeoutFieldSet(),
		Zoekt:           r.zoekt,
		SearcherURLs:    r.searcherURLs,
	}
	if err := args.PatternInfo.Validate(); err != nil {
		return nil, err
	}

	fileResults, _, err := searchFilesInRepos(ctx, &args)
	if err != nil {
		return nil, err
	}

	var suggestions []*searchSuggestionResolver
	for i, result := range fileResults {
		assumedScore := len(fileResults) - i // Greater score is first, so we inverse the index.
		suggestions = append(suggestions, newSearchSuggestionResolver(result.File(), assumedScore))
	}
	return suggestions, nil
}

func unionRegExps(patterns []string) string {
	if len(patterns) == 0 {
		return ""
	}
	if len(patterns) == 1 {
		return patterns[0]
	}

	// We only need to wrap the pattern in parentheses if it contains a "|" because
	// "|" has the lowest precedence of any operator.
	patterns2 := make([]string, len(patterns))
	for i, p := range patterns {
		if strings.Contains(p, "|") {
			p = "(" + p + ")"
		}
		patterns2[i] = p
	}
	return strings.Join(patterns2, "|")
}

type badRequestError struct {
	err error
}

func (e *badRequestError) BadRequest() bool {
	return true
}

func (e *badRequestError) Error() string {
	return "bad request: " + e.err.Error()
}

func (e *badRequestError) Cause() error {
	return e.err
}

// searchSuggestionResolver is a resolver for the GraphQL union type `SearchSuggestion`
type searchSuggestionResolver struct {
	// result is either a RepositoryResolver or a GitTreeEntryResolver
	result interface{}
	// score defines how well this item matches the query for sorting purposes
	score int
	// length holds the length of the item name as a second sorting criterium
	length int
	// label to sort alphabetically by when all else is equal.
	label string
}

func (r *searchSuggestionResolver) ToRepository() (*RepositoryResolver, bool) {
	res, ok := r.result.(*RepositoryResolver)
	return res, ok
}

func (r *searchSuggestionResolver) ToFile() (*GitTreeEntryResolver, bool) {
	res, ok := r.result.(*GitTreeEntryResolver)
	return res, ok
}

func (r *searchSuggestionResolver) ToGitBlob() (*GitTreeEntryResolver, bool) {
	res, ok := r.result.(*GitTreeEntryResolver)
	return res, ok && res.stat.Mode().IsRegular()
}

func (r *searchSuggestionResolver) ToGitTree() (*GitTreeEntryResolver, bool) {
	res, ok := r.result.(*GitTreeEntryResolver)
	return res, ok && res.stat.Mode().IsDir()
}

func (r *searchSuggestionResolver) ToSymbol() (*symbolResolver, bool) {
	s, ok := r.result.(*searchSymbolResult)
	if !ok {
		return nil, false
	}
	return toSymbolResolver(s.symbol, s.baseURI, s.lang, s.commit), true
}

func (r *searchSuggestionResolver) ToLanguage() (*languageResolver, bool) {
	res, ok := r.result.(*languageResolver)
	return res, ok
}

// newSearchSuggestionResolver returns a new searchSuggestionResolver wrapping the
// given result.
//
// A panic occurs if the type of result is not a *RepositoryResolver, *GitTreeEntryResolver,
// *searchSymbolResult or *languageResolver.
func newSearchSuggestionResolver(result interface{}, score int) *searchSuggestionResolver {
	switch r := result.(type) {
	case *RepositoryResolver:
		return &searchSuggestionResolver{result: r, score: score, length: len(r.repo.Name), label: string(r.repo.Name)}

	case *GitTreeEntryResolver:
		return &searchSuggestionResolver{result: r, score: score, length: len(r.Path()), label: r.Path()}

	case *searchSymbolResult:
		return &searchSuggestionResolver{result: r, score: score, length: len(r.symbol.Name + " " + r.symbol.Parent), label: r.symbol.Name + " " + r.symbol.Parent}

	case *languageResolver:
		return &searchSuggestionResolver{result: r, score: score, length: len(r.Name()), label: r.Name()}

	default:
		panic("never here")
	}
}

func sortSearchSuggestions(s []*searchSuggestionResolver) {
	sort.Slice(s, func(i, j int) bool {
		// Sort by score
		a, b := s[i], s[j]
		if a.score != b.score {
			return a.score > b.score
		}
		// Prefer shorter strings for the same match score
		// E.g. prefer gorilla/mux over gorilla/muxy, Microsoft/vscode over g3ortega/vscode-crystal
		if a.length != b.length {
			return a.length < b.length
		}

		// All else equal, sort alphabetically.
		return a.label < b.label
	})
}

// handleRepoSearchResult handles the limitHit and searchErr returned by a search function,
// updating common as to reflect that new information. If searchErr is a fatal error,
// it returns a non-nil error; otherwise, if searchErr == nil or a non-fatal error, it returns a
// nil error.
func handleRepoSearchResult(common *searchResultsCommon, repoRev *search.RepositoryRevisions, limitHit, timedOut bool, searchErr error) (fatalErr error) {
	common.limitHit = common.limitHit || limitHit
	if vcs.IsRepoNotExist(searchErr) {
		if vcs.IsCloneInProgress(searchErr) {
			common.cloning = append(common.cloning, repoRev.Repo)
		} else {
			common.missing = append(common.missing, repoRev.Repo)
		}
	} else if gitserver.IsRevisionNotFound(searchErr) {
		if len(repoRev.Revs) == 0 || len(repoRev.Revs) == 1 && repoRev.Revs[0].RevSpec == "" {
			// If we didn't specify an input revision, then the repo is empty and can be ignored.
		} else {
			return searchErr
		}
	} else if errcode.IsNotFound(searchErr) {
		common.missing = append(common.missing, repoRev.Repo)
	} else if errcode.IsTimeout(searchErr) || errcode.IsTemporary(searchErr) || timedOut {
		common.timedout = append(common.timedout, repoRev.Repo)
	} else if searchErr != nil {
		return searchErr
	}
	return nil
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
