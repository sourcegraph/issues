package graphqlbackend

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/cockroachdb/errors"
	otlog "github.com/opentracing/opentracing-go/log"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
	"github.com/sourcegraph/sourcegraph/internal/endpoint"
	"github.com/sourcegraph/sourcegraph/internal/search"
	searchbackend "github.com/sourcegraph/sourcegraph/internal/search/backend"
	"github.com/sourcegraph/sourcegraph/internal/search/query"
	searchrepos "github.com/sourcegraph/sourcegraph/internal/search/repos"
	"github.com/sourcegraph/sourcegraph/internal/search/run"
	"github.com/sourcegraph/sourcegraph/internal/search/streaming"
	"github.com/sourcegraph/sourcegraph/internal/search/unindexed"
	"github.com/sourcegraph/sourcegraph/internal/trace"
	"github.com/sourcegraph/sourcegraph/schema"
)

// This file contains the root resolver for search. It currently has a lot of
// logic that spans out into all the other search_* files.
var mockResolveRepositories func() (resolved searchrepos.Resolved, err error)

type SearchArgs struct {
	Version        string
	PatternType    *string
	Query          string
	VersionContext *string

	// Stream if non-nil will stream all SearchEvents.
	//
	// This is how our streaming and our batch interface co-exist. When this
	// is set, it exposes a way to stream out results as we collect them.
	//
	// TODO(keegan) This is not our final design. For example this doesn't
	// allow us to stream out things like dynamic filters or take into account
	// AND/OR. However, streaming is behind a feature flag for now, so this is
	// to make it visible in the browser.
	Stream streaming.Sender

	// For tests
	Settings *schema.Settings
}

type SearchImplementer interface {
	Results(context.Context) (*SearchResultsResolver, error)
	Suggestions(context.Context, *searchSuggestionsArgs) ([]SearchSuggestionResolver, error)
	//lint:ignore U1000 is used by graphql via reflection
	Stats(context.Context) (*searchResultsStats, error)

	Inputs() run.SearchInputs
}

// NewSearchImplementer returns a SearchImplementer that provides search results and suggestions.
func NewSearchImplementer(ctx context.Context, db dbutil.DB, args *SearchArgs) (_ SearchImplementer, err error) {
	tr, ctx := trace.New(ctx, "NewSearchImplementer", args.Query)
	defer func() {
		tr.SetError(err)
		tr.Finish()
	}()

	settings := args.Settings
	if settings == nil {
		var err error
		settings, err = decodedViewerFinalSettings(ctx, db)
		if err != nil {
			return nil, err
		}
	}

	searchType, err := detectSearchType(args.Version, args.PatternType)
	if err != nil {
		return nil, err
	}
	searchType = overrideSearchType(args.Query, searchType)

	if searchType == query.SearchTypeStructural && !conf.StructuralSearchEnabled() {
		return nil, errors.New("Structural search is disabled in the site configuration.")
	}

	var plan query.Plan
	globbing := getBoolPtr(settings.SearchGlobbing, false)
	tr.LogFields(otlog.Bool("globbing", globbing))
	plan, err = query.Pipeline(
		query.Init(args.Query, searchType),
		query.With(globbing, query.Globbing),
	)
	if err != nil {
		return alertForQuery(args.Query, err).wrapSearchImplementer(db), nil
	}
	tr.LazyPrintf("parsing done")

	defaultLimit := defaultMaxSearchResults
	if args.Stream != nil {
		defaultLimit = defaultMaxSearchResultsStreaming
	}
	if searchType == query.SearchTypeStructural {
		// Set a lower max result count until structural search supports true streaming.
		defaultLimit = defaultMaxSearchResults
	}

	return &searchResolver{
		db: db,
		SearchInputs: &run.SearchInputs{
			Plan:           plan,
			Query:          plan.ToParseTree(),
			OriginalQuery:  args.Query,
			VersionContext: args.VersionContext,
			UserSettings:   settings,
			PatternType:    searchType,
			DefaultLimit:   defaultLimit,
		},

		stream: args.Stream,

		zoekt:        search.Indexed(),
		searcherURLs: search.SearcherURLs(),
		reposMu:      &sync.Mutex{},
		resolved:     &searchrepos.Resolved{},
	}, nil
}

func (r *schemaResolver) Search(ctx context.Context, args *SearchArgs) (SearchImplementer, error) {
	return NewSearchImplementer(ctx, r.db, args)
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
			return -1, errors.Errorf("unrecognized patternType: %v", patternType)
		}
	} else {
		switch version {
		case "V1":
			searchType = query.SearchTypeRegex
		case "V2":
			searchType = query.SearchTypeLiteral
		default:
			return -1, errors.Errorf("unrecognized version want \"V1\" or \"V2\": %v", version)
		}
	}
	return searchType, nil
}

func overrideSearchType(input string, searchType query.SearchType) query.SearchType {
	q, err := query.Parse(input, query.SearchTypeLiteral)
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
	return searchType
}

func getBoolPtr(b *bool, def bool) bool {
	if b == nil {
		return def
	}
	return *b
}

// searchResolver is a resolver for the GraphQL type `Search`
type searchResolver struct {
	*run.SearchInputs
	db                  dbutil.DB
	invalidateRepoCache bool // if true, invalidates the repo cache when evaluating search subexpressions.

	// stream if non-nil will send all search events we receive down it.
	stream streaming.Sender

	// Cached resolveRepositories results. We use a pointer to the mutex so that we
	// can copy the resolver, while sharing the mutex. If we didn't use a pointer,
	// the mutex would lead to unexpected behaviour.
	reposMu  *sync.Mutex
	resolved *searchrepos.Resolved
	repoErr  error

	zoekt        *searchbackend.Zoekt
	searcherURLs *endpoint.Map
}

func (r *searchResolver) Inputs() run.SearchInputs {
	return *r.SearchInputs
}

// rawQuery returns the original query string input.
func (r *searchResolver) rawQuery() string {
	return r.OriginalQuery
}

// protocol returns what type of search we are doing (batch, stream,
// paginated).
func (r *searchResolver) protocol() search.Protocol {
	if r.stream != nil {
		return search.Streaming
	}
	return search.Batch
}

const defaultMaxSearchResults = 30
const defaultMaxSearchResultsStreaming = 500

var mockDecodedViewerFinalSettings *schema.Settings

func decodedViewerFinalSettings(ctx context.Context, db dbutil.DB) (_ *schema.Settings, err error) {
	tr, ctx := trace.New(ctx, "decodedViewerFinalSettings", "")
	defer func() {
		tr.SetError(err)
		tr.Finish()
	}()
	if mockDecodedViewerFinalSettings != nil {
		return mockDecodedViewerFinalSettings, nil
	}
	merged, err := viewerFinalSettings(ctx, db)
	if err != nil {
		return nil, err
	}
	var settings schema.Settings
	if err := json.Unmarshal([]byte(merged.Contents()), &settings); err != nil {
		return nil, err
	}
	return &settings, nil
}

type resolveRepositoriesOpts struct {
	effectiveRepoFieldValues []string

	limit int // Maximum repositories to return
}

// resolveRepositories calls ResolveRepositories, caching the result for the common case
// where opts.effectiveRepoFieldValues == nil.
func (r *searchResolver) resolveRepositories(ctx context.Context, options search.RepoOptions) (resolved searchrepos.Resolved, err error) {
	if mockResolveRepositories != nil {
		return mockResolveRepositories()
	}

	tr, ctx := trace.New(ctx, "graphql.resolveRepositories", fmt.Sprintf("options: %+v", options))
	defer func() {
		tr.SetError(err)
		tr.LazyPrintf("%s", resolved.String())
		tr.Finish()
	}()

	if options.CacheLookup {
		// Cache if opts are empty, so that multiple calls to resolveRepositories only
		// hit the database once.
		r.reposMu.Lock()
		defer r.reposMu.Unlock()
		if r.resolved.RepoRevs != nil || r.resolved.MissingRepoRevs != nil || r.repoErr != nil {
			tr.LazyPrintf("cached")
			return *r.resolved, r.repoErr
		}
		defer func() {
			r.resolved = &resolved
			r.repoErr = err
		}()
	}

	tr.LazyPrintf("resolveRepositories - start")
	defer tr.LazyPrintf("resolveRepositories - done")

	repositoryResolver := &searchrepos.Resolver{
		DB:                  r.db,
		Zoekt:               r.zoekt,
		SearchableReposFunc: backend.Repos.ListSearchable,
	}

	return repositoryResolver.Resolve(ctx, options)
}

func (r *searchResolver) suggestFilePaths(ctx context.Context, limit int) ([]SearchSuggestionResolver, error) {
	q, err := query.ToBasicQuery(r.Query)
	if err != nil {
		return nil, err
	}
	if !query.IsPatternAtom(q) {
		// Not an atomic pattern, can't guarantee it will behave well.
		return nil, nil
	}
	p := search.ToTextPatternInfo(q, r.protocol(), query.PatternToFile)

	args := search.TextParameters{
		PatternInfo:     p,
		RepoPromise:     &search.RepoPromise{}, // TODO(rvantonder) remove this field for this type.
		Query:           r.Query,
		UseFullDeadline: r.Query.Timeout() != nil || r.Query.Count() != nil,
		Zoekt:           r.zoekt,
		SearcherURLs:    r.searcherURLs,
	}
	repoOptions := r.toRepoOptions(args.Query, resolveRepositoriesOpts{})
	resolved, err := r.resolveRepositories(ctx, repoOptions)
	if err != nil {
		return nil, err
	}

	if resolved.OverLimit {
		// If we've exceeded the repo limit, then we may miss files from repos we care
		// about, so don't bother searching filenames at all.
		return nil, nil
	}

	args.RepoPromise = (&search.RepoPromise{}).Resolve(resolved.RepoRevs)

	fileMatches, _, err := unindexed.SearchFilesInReposBatch(ctx, &args)
	if err != nil {
		return nil, err
	}

	var suggestions []SearchSuggestionResolver
	for i, fm := range fileMatches {
		assumedScore := len(fileMatches) - i // Greater score is first, so we inverse the index.
		fmr := &FileMatchResolver{
			FileMatch:    *fm,
			db:           r.db,
			RepoResolver: NewRepositoryResolver(r.db, fm.Repo.ToRepo()),
		}
		suggestions = append(suggestions, gitTreeSuggestionResolver{
			gitTreeEntry: fmr.File(),
			score:        assumedScore,
		})
	}
	return suggestions, nil
}
