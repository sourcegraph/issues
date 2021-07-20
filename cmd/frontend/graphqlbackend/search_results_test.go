package graphqlbackend

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/zoekt"
	"go.uber.org/atomic"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/envvar"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/database/dbtest"
	"github.com/sourcegraph/sourcegraph/internal/database/dbtesting"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/search"
	searchbackend "github.com/sourcegraph/sourcegraph/internal/search/backend"
	"github.com/sourcegraph/sourcegraph/internal/search/query"
	searchrepos "github.com/sourcegraph/sourcegraph/internal/search/repos"
	"github.com/sourcegraph/sourcegraph/internal/search/result"
	"github.com/sourcegraph/sourcegraph/internal/search/run"
	"github.com/sourcegraph/sourcegraph/internal/search/streaming"
	"github.com/sourcegraph/sourcegraph/internal/search/symbol"
	"github.com/sourcegraph/sourcegraph/internal/search/unindexed"
	"github.com/sourcegraph/sourcegraph/internal/types"
	"github.com/sourcegraph/sourcegraph/schema"
)

var mockCount = func(_ context.Context, options database.ReposListOptions) (int, error) { return 0, nil }

func assertEqual(t *testing.T, got, want interface{}) {
	t.Helper()

	if diff := cmp.Diff(got, want); diff != "" {
		t.Fatalf("(-want +got):\n%s", diff)
	}
}

func TestSearchResults(t *testing.T) {
	db := new(dbtesting.MockDB)

	limitOffset := &database.LimitOffset{Limit: search.SearchLimits(conf.Get()).MaxRepos + 1}

	getResults := func(t *testing.T, query, version string) []string {
		r, err := (&schemaResolver{db: db}).Search(context.Background(), &SearchArgs{Query: query, Version: version})
		if err != nil {
			t.Fatal("Search:", err)
		}
		results, err := r.Results(context.Background())
		if err != nil {
			t.Fatal("Results:", err)
		}
		resultDescriptions := make([]string, len(results.Matches))
		for i, match := range results.Matches {
			// NOTE: Only supports one match per line. If we need to test other cases,
			// just remove that assumption in the following line of code.
			switch m := match.(type) {
			case *result.RepoMatch:
				resultDescriptions[i] = fmt.Sprintf("repo:%s", m.Name)
			case *result.FileMatch:
				resultDescriptions[i] = fmt.Sprintf("%s:%d", m.Path, m.LineMatches[0].LineNumber)
			default:
				t.Fatal("unexpected result type", match)
			}
		}
		// dedup results since we expect our clients to do dedupping
		if len(resultDescriptions) > 1 {
			sort.Strings(resultDescriptions)
			dedup := resultDescriptions[:1]
			for _, s := range resultDescriptions[1:] {
				if s != dedup[len(dedup)-1] {
					dedup = append(dedup, s)
				}
			}
			resultDescriptions = dedup
		}
		return resultDescriptions
	}
	testCallResults := func(t *testing.T, query, version string, want []string) {
		t.Helper()
		results := getResults(t, query, version)
		if d := cmp.Diff(want, results); d != "" {
			t.Errorf("unexpected results (-want, +got):\n%s", d)
		}
	}

	searchVersions := []string{"V1", "V2"}

	t.Run("repo: only", func(t *testing.T) {
		mockDecodedViewerFinalSettings = &schema.Settings{}
		defer func() { mockDecodedViewerFinalSettings = nil }()

		var calledReposListRepoNames bool
		database.Mocks.Repos.ListRepoNames = func(_ context.Context, op database.ReposListOptions) ([]types.RepoName, error) {
			calledReposListRepoNames = true

			// Validate that the following options are invariant
			// when calling the DB through Repos.ListRepoNames, no matter how
			// many times it is called for a single Search(...) operation.
			assertEqual(t, op.LimitOffset, limitOffset)
			assertEqual(t, op.IncludePatterns, []string{"r", "p"})

			return []types.RepoName{{ID: 1, Name: "repo"}}, nil
		}
		database.Mocks.Repos.MockGetByName(t, "repo", 1)
		database.Mocks.Repos.MockGet(t, 1)
		database.Mocks.Repos.Count = mockCount

		unindexed.MockSearchFilesInRepos = func(args *search.TextParameters) ([]result.Match, *streaming.Stats, error) {
			return nil, &streaming.Stats{}, nil
		}
		defer func() { unindexed.MockSearchFilesInRepos = nil }()

		for _, v := range searchVersions {
			testCallResults(t, `repo:r repo:p`, v, []string{"repo:repo"})
			if !calledReposListRepoNames {
				t.Error("!calledReposListRepoNames")
			}
		}

	})

	t.Run("multiple terms regexp", func(t *testing.T) {
		mockDecodedViewerFinalSettings = &schema.Settings{}
		defer func() { mockDecodedViewerFinalSettings = nil }()

		var calledReposListRepoNames bool
		database.Mocks.Repos.ListRepoNames = func(_ context.Context, op database.ReposListOptions) ([]types.RepoName, error) {
			calledReposListRepoNames = true

			// Validate that the following options are invariant
			// when calling the DB through Repos.List, no matter how
			// many times it is called for a single Search(...) operation.
			assertEqual(t, op.LimitOffset, limitOffset)

			return []types.RepoName{{ID: 1, Name: "repo"}}, nil
		}
		defer func() { database.Mocks = database.MockStores{} }()
		database.Mocks.Repos.MockGetByName(t, "repo", 1)
		database.Mocks.Repos.MockGet(t, 1)
		database.Mocks.Repos.Count = mockCount

		calledSearchRepositories := false
		run.MockSearchRepositories = func(args *search.TextParameters) ([]result.Match, *streaming.Stats, error) {
			calledSearchRepositories = true
			return nil, &streaming.Stats{}, nil
		}
		defer func() { run.MockSearchRepositories = nil }()

		calledSearchSymbols := false
		symbol.MockSearchSymbols = func(ctx context.Context, args *search.TextParameters, limit int) (res []result.Match, common *streaming.Stats, err error) {
			calledSearchSymbols = true
			if want := `(foo\d).*?(bar\*)`; args.PatternInfo.Pattern != want {
				t.Errorf("got %q, want %q", args.PatternInfo.Pattern, want)
			}
			// TODO return mock results here and assert that they are output as results
			return nil, nil, nil
		}
		defer func() { symbol.MockSearchSymbols = nil }()

		calledSearchFilesInRepos := atomic.NewBool(false)
		unindexed.MockSearchFilesInRepos = func(args *search.TextParameters) ([]result.Match, *streaming.Stats, error) {
			calledSearchFilesInRepos.Store(true)
			if want := `(foo\d).*?(bar\*)`; args.PatternInfo.Pattern != want {
				t.Errorf("got %q, want %q", args.PatternInfo.Pattern, want)
			}
			repo := types.RepoName{ID: 1, Name: "repo"}
			fm := mkFileMatch(repo, "dir/file", 123)
			return []result.Match{fm}, &streaming.Stats{}, nil
		}
		defer func() { unindexed.MockSearchFilesInRepos = nil }()

		testCallResults(t, `foo\d "bar*"`, "V1", []string{"dir/file:123"})
		if !calledReposListRepoNames {
			t.Error("!calledReposListRepoNames")
		}
		if !calledSearchRepositories {
			t.Error("!calledSearchRepositories")
		}
		if !calledSearchFilesInRepos.Load() {
			t.Error("!calledSearchFilesInRepos")
		}
		if calledSearchSymbols {
			t.Error("calledSearchSymbols")
		}
	})

	t.Run("multiple terms literal", func(t *testing.T) {
		mockDecodedViewerFinalSettings = &schema.Settings{}
		defer func() { mockDecodedViewerFinalSettings = nil }()

		var calledReposListRepoNames bool
		database.Mocks.Repos.ListRepoNames = func(_ context.Context, op database.ReposListOptions) ([]types.RepoName, error) {
			calledReposListRepoNames = true

			// Validate that the following options are invariant
			// when calling the DB through Repos.List, no matter how
			// many times it is called for a single Search(...) operation.
			assertEqual(t, op.LimitOffset, limitOffset)

			return []types.RepoName{{ID: 1, Name: "repo"}}, nil
		}
		defer func() { database.Mocks = database.MockStores{} }()
		database.Mocks.Repos.MockGetByName(t, "repo", 1)
		database.Mocks.Repos.MockGet(t, 1)
		database.Mocks.Repos.Count = mockCount

		calledSearchRepositories := false
		run.MockSearchRepositories = func(args *search.TextParameters) ([]result.Match, *streaming.Stats, error) {
			calledSearchRepositories = true
			return nil, &streaming.Stats{}, nil
		}
		defer func() { run.MockSearchRepositories = nil }()

		calledSearchSymbols := false
		symbol.MockSearchSymbols = func(ctx context.Context, args *search.TextParameters, limit int) (res []result.Match, common *streaming.Stats, err error) {
			calledSearchSymbols = true
			if want := `"foo\\d \"bar*\""`; args.PatternInfo.Pattern != want {
				t.Errorf("got %q, want %q", args.PatternInfo.Pattern, want)
			}
			// TODO return mock results here and assert that they are output as results
			return nil, nil, nil
		}
		defer func() { symbol.MockSearchSymbols = nil }()

		calledSearchFilesInRepos := atomic.NewBool(false)
		unindexed.MockSearchFilesInRepos = func(args *search.TextParameters) ([]result.Match, *streaming.Stats, error) {
			calledSearchFilesInRepos.Store(true)
			if want := `foo\\d "bar\*"`; args.PatternInfo.Pattern != want {
				t.Errorf("got %q, want %q", args.PatternInfo.Pattern, want)
			}
			repo := types.RepoName{ID: 1, Name: "repo"}
			fm := mkFileMatch(repo, "dir/file", 123)
			return []result.Match{fm}, &streaming.Stats{}, nil
		}
		defer func() { unindexed.MockSearchFilesInRepos = nil }()

		testCallResults(t, `foo\d "bar*"`, "V2", []string{"dir/file:123"})
		if !calledReposListRepoNames {
			t.Error("!calledReposListRepoNames")
		}
		if !calledSearchRepositories {
			t.Error("!calledSearchRepositories")
		}
		if !calledSearchFilesInRepos.Load() {
			t.Error("!calledSearchFilesInRepos")
		}
		if calledSearchSymbols {
			t.Error("calledSearchSymbols")
		}
	})
}

func TestSearchResolver_DynamicFilters(t *testing.T) {
	db := new(dbtesting.MockDB)

	repo := types.RepoName{Name: "testRepo"}
	repoMatch := &result.RepoMatch{Name: "testRepo"}
	fileMatch := func(path string) *result.FileMatch {
		return mkFileMatch(repo, path)
	}

	rev := "develop3.0"
	fileMatchRev := fileMatch("/testFile.md")
	fileMatchRev.InputRev = &rev

	type testCase struct {
		descr                             string
		searchResults                     []result.Match
		expectedDynamicFilterStrsRegexp   map[string]int
		expectedDynamicFilterStrsGlobbing map[string]int
	}

	tests := []testCase{

		{
			descr:         "single repo match",
			searchResults: []result.Match{repoMatch},
			expectedDynamicFilterStrsRegexp: map[string]int{
				`repo:^testRepo$`: 1,
			},
			expectedDynamicFilterStrsGlobbing: map[string]int{
				`repo:testRepo`: 1,
			},
		},

		{
			descr:         "single file match without revision in query",
			searchResults: []result.Match{fileMatch("/testFile.md")},
			expectedDynamicFilterStrsRegexp: map[string]int{
				`repo:^testRepo$`: 1,
				`lang:markdown`:   1,
			},
			expectedDynamicFilterStrsGlobbing: map[string]int{
				`repo:testRepo`: 1,
				`lang:markdown`: 1,
			},
		},

		{
			descr:         "single file match with specified revision",
			searchResults: []result.Match{fileMatchRev},
			expectedDynamicFilterStrsRegexp: map[string]int{
				`repo:^testRepo$@develop3.0`: 1,
				`lang:markdown`:              1,
			},
			expectedDynamicFilterStrsGlobbing: map[string]int{
				`repo:testRepo@develop3.0`: 1,
				`lang:markdown`:            1,
			},
		},
		{
			descr:         "file match from a language with two file extensions, using first extension",
			searchResults: []result.Match{fileMatch("/testFile.ts")},
			expectedDynamicFilterStrsRegexp: map[string]int{
				`repo:^testRepo$`: 1,
				`lang:typescript`: 1,
			},
			expectedDynamicFilterStrsGlobbing: map[string]int{
				`repo:testRepo`:   1,
				`lang:typescript`: 1,
			},
		},
		{
			descr:         "file match from a language with two file extensions, using second extension",
			searchResults: []result.Match{fileMatch("/testFile.tsx")},
			expectedDynamicFilterStrsRegexp: map[string]int{
				`repo:^testRepo$`: 1,
				`lang:typescript`: 1,
			},
			expectedDynamicFilterStrsGlobbing: map[string]int{
				`repo:testRepo`:   1,
				`lang:typescript`: 1,
			},
		},
		{
			descr:         "file match which matches one of the common file filters",
			searchResults: []result.Match{fileMatch("/anything/node_modules/testFile.md")},
			expectedDynamicFilterStrsRegexp: map[string]int{
				`repo:^testRepo$`:          1,
				`-file:(^|/)node_modules/`: 1,
				`lang:markdown`:            1,
			},
			expectedDynamicFilterStrsGlobbing: map[string]int{
				`repo:testRepo`: 1,
				`-file:node_modules/** -file:**/node_modules/**`: 1,
				`lang:markdown`: 1,
			},
		},
		{
			descr:         "file match which matches one of the common file filters",
			searchResults: []result.Match{fileMatch("/node_modules/testFile.md")},
			expectedDynamicFilterStrsRegexp: map[string]int{
				`repo:^testRepo$`:          1,
				`-file:(^|/)node_modules/`: 1,
				`lang:markdown`:            1,
			},
			expectedDynamicFilterStrsGlobbing: map[string]int{
				`repo:testRepo`: 1,
				`-file:node_modules/** -file:**/node_modules/**`: 1,
				`lang:markdown`: 1,
			},
		},
		{
			descr: "file match which matches one of the common file filters",
			searchResults: []result.Match{
				fileMatch("/foo_test.go"),
				fileMatch("/foo.go"),
			},
			expectedDynamicFilterStrsRegexp: map[string]int{
				`repo:^testRepo$`:  2,
				`-file:_test\.go$`: 1,
				`lang:go`:          2,
			},
			expectedDynamicFilterStrsGlobbing: map[string]int{
				`repo:testRepo`:    2,
				`-file:**_test.go`: 1,
				`lang:go`:          2,
			},
		},

		{
			descr: "prefer rust to renderscript",
			searchResults: []result.Match{
				fileMatch("/channel.rs"),
			},
			expectedDynamicFilterStrsRegexp: map[string]int{
				`repo:^testRepo$`: 1,
				`lang:rust`:       1,
			},
			expectedDynamicFilterStrsGlobbing: map[string]int{
				`repo:testRepo`: 1,
				`lang:rust`:     1,
			},
		},

		{
			descr: "javascript filters",
			searchResults: []result.Match{
				fileMatch("/jsrender.min.js.map"),
				fileMatch("playground/react/lib/app.js.map"),
				fileMatch("assets/javascripts/bootstrap.min.js"),
			},
			expectedDynamicFilterStrsRegexp: map[string]int{
				`repo:^testRepo$`:  3,
				`-file:\.min\.js$`: 1,
				`-file:\.js\.map$`: 2,
				`lang:javascript`:  1,
			},
			expectedDynamicFilterStrsGlobbing: map[string]int{
				`repo:testRepo`:   3,
				`-file:**.min.js`: 1,
				`-file:**.js.map`: 2,
				`lang:javascript`: 1,
			},
		},

		// If there are no search results, no filters should be displayed.
		{
			descr:                             "no results",
			searchResults:                     []result.Match{},
			expectedDynamicFilterStrsRegexp:   map[string]int{},
			expectedDynamicFilterStrsGlobbing: map[string]int{},
		},
		{
			descr:         "values containing spaces are quoted",
			searchResults: []result.Match{fileMatch("/.gitignore")},
			expectedDynamicFilterStrsRegexp: map[string]int{
				`repo:^testRepo$`:    1,
				`lang:"ignore list"`: 1,
			},
			expectedDynamicFilterStrsGlobbing: map[string]int{
				`repo:testRepo`:      1,
				`lang:"ignore list"`: 1,
			},
		},
	}

	mockDecodedViewerFinalSettings = &schema.Settings{}
	defer func() { mockDecodedViewerFinalSettings = nil }()

	var expectedDynamicFilterStrs map[string]int
	for _, test := range tests {
		t.Run(test.descr, func(t *testing.T) {
			for _, globbing := range []bool{true, false} {
				mockDecodedViewerFinalSettings.SearchGlobbing = &globbing
				actualDynamicFilters := (&SearchResultsResolver{db: db, SearchResults: &SearchResults{Matches: test.searchResults}}).DynamicFilters(context.Background())
				actualDynamicFilterStrs := make(map[string]int)

				for _, filter := range actualDynamicFilters {
					actualDynamicFilterStrs[filter.Value()] = int(filter.Count())
				}

				if globbing {
					expectedDynamicFilterStrs = test.expectedDynamicFilterStrsGlobbing
				} else {
					expectedDynamicFilterStrs = test.expectedDynamicFilterStrsRegexp
				}

				if diff := cmp.Diff(expectedDynamicFilterStrs, actualDynamicFilterStrs); diff != "" {
					t.Errorf("mismatch (-want, +got):\n%s", diff)
				}
			}
		})
	}
}

func TestLonger(t *testing.T) {
	N := 2
	noise := time.Nanosecond
	for dt := time.Millisecond + noise; dt < time.Hour; dt += time.Millisecond {
		dt2 := longer(N, dt)
		if dt2 < time.Duration(N)*dt {
			t.Fatalf("longer(%v)=%v < 2*%v, want more", dt, dt2, dt)
		}
		if strings.Contains(dt2.String(), ".") {
			t.Fatalf("longer(%v).String() = %q contains an unwanted decimal point, want a nice round duration", dt, dt2)
		}
		lowest := 2 * time.Second
		if dt2 < lowest {
			t.Fatalf("longer(%v) = %v < %s, too short", dt, dt2, lowest)
		}
	}
}

func TestSearchResultsHydration(t *testing.T) {
	db := new(dbtesting.MockDB)

	id := 42
	repoName := "reponame-foobar"
	fileName := "foobar.go"

	repoWithIDs := &types.Repo{

		ID:   api.RepoID(id),
		Name: api.RepoName(repoName),
		ExternalRepo: api.ExternalRepoSpec{
			ID:          repoName,
			ServiceType: extsvc.TypeGitHub,
			ServiceID:   "https://github.com",
		}}

	hydratedRepo := &types.Repo{

		ID:           repoWithIDs.ID,
		ExternalRepo: repoWithIDs.ExternalRepo,
		Name:         repoWithIDs.Name,
		URI:          fmt.Sprintf("github.com/my-org/%s", repoWithIDs.Name),
		Description:  "This is a description of a repository",
		Fork:         false,
	}

	database.Mocks.Repos.Get = func(ctx context.Context, id api.RepoID) (*types.Repo, error) {
		return hydratedRepo, nil
	}

	database.Mocks.Repos.ListRepoNames = func(_ context.Context, op database.ReposListOptions) ([]types.RepoName, error) {
		return []types.RepoName{{ID: repoWithIDs.ID, Name: repoWithIDs.Name}}, nil
	}
	database.Mocks.Repos.Count = mockCount

	defer func() { database.Mocks = database.MockStores{} }()

	zoektRepo := &zoekt.RepoListEntry{
		Repository: zoekt.Repository{
			ID:       uint32(repoWithIDs.ID),
			Name:     string(repoWithIDs.Name),
			Branches: []zoekt.RepositoryBranch{{Name: "HEAD", Version: "deadbeef"}},
		},
	}

	zoektFileMatches := []zoekt.FileMatch{{
		Score:        5.0,
		FileName:     fileName,
		RepositoryID: uint32(repoWithIDs.ID),
		Repository:   string(repoWithIDs.Name), // Important: this needs to match a name in `repos`
		Branches:     []string{"master"},
		LineMatches: []zoekt.LineMatch{
			{
				Line: nil,
			},
		},
		Checksum: []byte{0, 1, 2},
	}}

	z := &searchbackend.Zoekt{
		Client: &searchbackend.FakeSearcher{
			Repos:  []*zoekt.RepoListEntry{zoektRepo},
			Result: &zoekt.SearchResult{Files: zoektFileMatches},
		},
		DisableCache: true,
	}

	ctx := context.Background()

	p, err := query.Pipeline(query.InitLiteral(`foobar index:only count:350`))
	if err != nil {
		t.Fatal(err)
	}
	resolver := &searchResolver{
		db: db,
		SearchInputs: &run.SearchInputs{
			Plan:         p,
			Query:        p.ToParseTree(),
			UserSettings: &schema.Settings{},
		},
		zoekt:    z,
		reposMu:  &sync.Mutex{},
		resolved: &searchrepos.Resolved{},
	}
	results, err := resolver.Results(ctx)
	if err != nil {
		t.Fatal("Results:", err)
	}
	// We want one file match and one repository match
	wantMatchCount := 2
	if int(results.MatchCount()) != wantMatchCount {
		t.Fatalf("wrong results length. want=%d, have=%d\n", wantMatchCount, results.MatchCount())
	}

	for _, r := range results.Results() {
		switch r := r.(type) {
		case *FileMatchResolver:
			assertRepoResolverHydrated(ctx, t, r.Repository(), hydratedRepo)

		case *RepositoryResolver:
			assertRepoResolverHydrated(ctx, t, r, hydratedRepo)
		}
	}
}

func Test_SearchResultsResolver_ApproximateResultCount(t *testing.T) {
	db := new(dbtesting.MockDB)
	type fields struct {
		results             []result.Match
		searchResultsCommon streaming.Stats
		alert               *searchAlert
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name:   "empty",
			fields: fields{},
			want:   "0",
		},

		{
			name: "file matches",
			fields: fields{
				results: []result.Match{&result.FileMatch{}},
			},
			want: "1",
		},

		{
			name: "file matches limit hit",
			fields: fields{
				results:             []result.Match{&result.FileMatch{}},
				searchResultsCommon: streaming.Stats{IsLimitHit: true},
			},
			want: "1+",
		},

		{
			name: "symbol matches",
			fields: fields{
				results: []result.Match{
					&result.FileMatch{
						Symbols: []*result.SymbolMatch{
							// 1
							{},
							// 2
							{},
						},
					},
				},
			},
			want: "2",
		},

		{
			name: "symbol matches limit hit",
			fields: fields{
				results: []result.Match{
					&result.FileMatch{
						Symbols: []*result.SymbolMatch{
							// 1
							{},
							// 2
							{},
						},
					},
				},
				searchResultsCommon: streaming.Stats{IsLimitHit: true},
			},
			want: "2+",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sr := &SearchResultsResolver{
				db: db,
				SearchResults: &SearchResults{
					Stats:   tt.fields.searchResultsCommon,
					Matches: tt.fields.results,
					Alert:   tt.fields.alert,
				},
			}
			if got := sr.ApproximateResultCount(); got != tt.want {
				t.Errorf("searchResultsResolver.ApproximateResultCount() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetExactFilePatterns(t *testing.T) {
	tests := []struct {
		in   string
		want map[string]struct{}
	}{
		{
			in:   "file:foo.bar file:*.bas",
			want: map[string]struct{}{"foo.bar": {}},
		},
		{
			in:   "file:foo.bar file:foo.bas",
			want: map[string]struct{}{"foo.bar": {}, "foo.bas": {}},
		},
		{
			in:   "file:*.bar",
			want: map[string]struct{}{},
		},
		{
			in:   "repo:github.com/foo/bar file:foo.bar",
			want: map[string]struct{}{"foo.bar": {}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			plan, err := query.Pipeline(query.InitLiteral(tt.in), query.Globbing)
			if err != nil {
				t.Fatal(err)
			}
			r := searchResolver{
				SearchInputs: &run.SearchInputs{
					Plan:          plan,
					Query:         plan.ToParseTree(),
					OriginalQuery: tt.in,
				},
			}
			if got := r.getExactFilePatterns(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getExactFilePatterns() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCompareSearchResults(t *testing.T) {
	makeResult := func(repo, file string) *result.FileMatch {
		return &result.FileMatch{File: result.File{
			Repo: types.RepoName{Name: api.RepoName(repo)},
			Path: file,
		}}
	}

	tests := []struct {
		name              string
		a                 *result.FileMatch
		b                 *result.FileMatch
		exactFilePatterns map[string]struct{}
		aIsLess           bool
	}{
		{
			name:              "prefer exact match",
			a:                 makeResult("arepo", "afile"),
			b:                 makeResult("arepo", "file"),
			exactFilePatterns: map[string]struct{}{"file": {}},
			aIsLess:           false,
		},
		{
			name:              "reverse a and b",
			a:                 makeResult("arepo", "file"),
			b:                 makeResult("arepo", "afile"),
			exactFilePatterns: map[string]struct{}{"file": {}},
			aIsLess:           true,
		},
		{
			name:              "alphabetical order if exactFilePatterns is empty",
			a:                 makeResult("arepo", "afile"),
			b:                 makeResult("arepo", "file"),
			exactFilePatterns: map[string]struct{}{},
			aIsLess:           true,
		},
		{
			name:              "alphabetical order if exactFilePatterns is nil",
			a:                 makeResult("arepo", "afile"),
			b:                 makeResult("arepo", "bfile"),
			exactFilePatterns: nil,
			aIsLess:           true,
		},
		{
			name:              "same length, different files",
			a:                 makeResult("arepo", "bfile"),
			b:                 makeResult("arepo", "afile"),
			exactFilePatterns: nil,
			aIsLess:           false,
		},
		{
			name:              "exact matches with different length",
			a:                 makeResult("arepo", "adir1/file"),
			b:                 makeResult("arepo", "dir1/file"),
			exactFilePatterns: map[string]struct{}{"file": {}},
			aIsLess:           false,
		},
		{
			name:              "exact matches with same length",
			a:                 makeResult("arepo", "dir2/file"),
			b:                 makeResult("arepo", "dir1/file"),
			exactFilePatterns: map[string]struct{}{"file": {}},
			aIsLess:           false,
		},
		{
			name:              "no match",
			a:                 makeResult("arepo", "afile"),
			b:                 makeResult("arepo", "bfile"),
			exactFilePatterns: map[string]struct{}{"file": {}},
			aIsLess:           true,
		},
		{
			name:              "different repo, 1 exact match",
			a:                 makeResult("arepo", "file"),
			b:                 makeResult("brepo", "afile"),
			exactFilePatterns: map[string]struct{}{"file": {}},
			aIsLess:           true,
		},
		{
			name:              "different repo, no exact patterns",
			a:                 makeResult("arepo", "file"),
			b:                 makeResult("brepo", "afile"),
			exactFilePatterns: nil,
			aIsLess:           true,
		},
		{
			name:              "different repo, 2 exact matches",
			a:                 makeResult("arepo", "file"),
			b:                 makeResult("brepo", "file"),
			exactFilePatterns: map[string]struct{}{"file": {}},
			aIsLess:           true,
		},
		{
			name:              "repo matches only",
			a:                 makeResult("arepo", ""),
			b:                 makeResult("brepo", ""),
			exactFilePatterns: nil,
			aIsLess:           true,
		},
		{
			name:              "repo match and file match, same repo",
			a:                 makeResult("arepo", "file"),
			b:                 makeResult("arepo", ""),
			exactFilePatterns: nil,
			aIsLess:           false,
		},
		{
			name:              "repo match and file match, different repos",
			a:                 makeResult("arepo", ""),
			b:                 makeResult("brepo", "file"),
			exactFilePatterns: nil,
			aIsLess:           true,
		},
		{
			name:              "prefer repo matches",
			a:                 makeResult("arepo", ""),
			b:                 makeResult("brepo", "file"),
			exactFilePatterns: map[string]struct{}{"file": {}},
			aIsLess:           true,
		},
	}
	for _, tt := range tests {
		t.Run("test", func(t *testing.T) {
			if got := compareSearchResults(tt.a, tt.b, tt.exactFilePatterns); got != tt.aIsLess {
				t.Errorf("compareSearchResults() = %v, aIsLess %v", got, tt.aIsLess)
			}
		})
	}
}

func TestEvaluateAnd(t *testing.T) {
	db := new(dbtesting.MockDB)

	tests := []struct {
		name         string
		query        string
		zoektMatches int
		filesSkipped int
		wantAlert    bool
	}{
		{
			name:         "zoekt returns enough matches, exhausted",
			query:        "foo and bar index:only count:5",
			zoektMatches: 5,
			filesSkipped: 0,
			wantAlert:    false,
		},
		{
			name:         "zoekt does not return enough matches, not exhausted",
			query:        "foo and bar index:only count:50",
			zoektMatches: 10,
			filesSkipped: 1,
			wantAlert:    true,
		},
		{
			name:         "zoekt returns enough matches, not exhausted",
			query:        "foo and bar index:only count:50",
			zoektMatches: 50,
			filesSkipped: 1,
			wantAlert:    false,
		},
	}

	minimalRepos, _, zoektRepos := generateRepos(5000)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			zoektFileMatches := generateZoektMatches(tt.zoektMatches)
			z := &searchbackend.Zoekt{
				Client: &searchbackend.FakeSearcher{
					Repos:  zoektRepos,
					Result: &zoekt.SearchResult{Files: zoektFileMatches, Stats: zoekt.Stats{FilesSkipped: tt.filesSkipped}},
				},
				DisableCache: true,
			}

			ctx := context.Background()

			database.Mocks.Repos.ListRepoNames = func(_ context.Context, op database.ReposListOptions) ([]types.RepoName, error) {
				repoNames := make([]types.RepoName, len(minimalRepos))
				for i := range minimalRepos {
					repoNames[i] = types.RepoName{ID: minimalRepos[i].ID, Name: minimalRepos[i].Name}
				}
				return repoNames, nil
			}
			database.Mocks.Repos.Count = func(ctx context.Context, opt database.ReposListOptions) (int, error) {
				return len(minimalRepos), nil
			}
			defer func() { database.Mocks = database.MockStores{} }()

			p, err := query.Pipeline(query.InitLiteral(tt.query))
			if err != nil {
				t.Fatal(err)
			}
			resolver := &searchResolver{
				db: db,
				SearchInputs: &run.SearchInputs{
					Plan:         p,
					Query:        p.ToParseTree(),
					UserSettings: &schema.Settings{},
				},
				zoekt:    z,
				reposMu:  &sync.Mutex{},
				resolved: &searchrepos.Resolved{},
			}
			results, err := resolver.Results(ctx)
			if err != nil {
				t.Fatal("Results:", err)
			}
			if tt.wantAlert {
				if results.SearchResults.Alert == nil {
					t.Errorf("Expected results")
				}
			} else if int(results.MatchCount()) != len(zoektFileMatches) {
				t.Errorf("wrong results length. want=%d, have=%d\n", len(zoektFileMatches), results.MatchCount())
			}
		})
	}
}

func TestSearchContext(t *testing.T) {
	orig := envvar.SourcegraphDotComMode()
	envvar.MockSourcegraphDotComMode(true)
	defer envvar.MockSourcegraphDotComMode(orig)

	db := dbtest.NewDB(t, "")

	tts := []struct {
		name        string
		searchQuery string
		numContexts int
	}{
		{name: "single search context", searchQuery: "foo context:@userA", numContexts: 1},
		{name: "multiple search contexts", searchQuery: "foo (context:@userA or context:@userB)", numContexts: 2},
	}

	users := map[string]int32{
		"userA": 1,
		"userB": 2,
	}

	mockZoekt := &searchbackend.Zoekt{
		Client:       &searchbackend.FakeSearcher{Repos: []*zoekt.RepoListEntry{}},
		DisableCache: true,
	}

	for _, tt := range tts {
		t.Run(tt.name, func(t *testing.T) {
			p, err := query.Pipeline(query.InitLiteral(tt.searchQuery))
			if err != nil {
				t.Fatal(err)
			}

			resolver := searchResolver{
				SearchInputs: &run.SearchInputs{
					Plan:         p,
					Query:        p.ToParseTree(),
					UserSettings: &schema.Settings{},
				},
				reposMu:  &sync.Mutex{},
				resolved: &searchrepos.Resolved{},
				zoekt:    mockZoekt,
				db:       db,
			}

			numGetByNameCalls := 0
			database.Mocks.Repos.ListRepoNames = func(ctx context.Context, opts database.ReposListOptions) ([]types.RepoName, error) {
				return []types.RepoName{}, nil
			}
			database.Mocks.Repos.Count = func(ctx context.Context, op database.ReposListOptions) (int, error) { return 0, nil }
			database.Mocks.Namespaces.GetByName = func(ctx context.Context, name string) (*database.Namespace, error) {
				userID, ok := users[name]
				if !ok {
					t.Errorf("User with ID %d not found", userID)
				}
				numGetByNameCalls += 1
				return &database.Namespace{Name: name, User: userID}, nil
			}
			defer func() {
				database.Mocks.Repos.ListRepoNames = nil
				database.Mocks.Repos.Count = nil
				database.Mocks.Namespaces.GetByName = nil
			}()

			_, err = resolver.Results(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if numGetByNameCalls != tt.numContexts {
				t.Fatalf("got %d, want %d", numGetByNameCalls, tt.numContexts)
			}
		})
	}
}

func TestIsGlobalSearch(t *testing.T) {
	orig := envvar.SourcegraphDotComMode()
	envvar.MockSourcegraphDotComMode(true)
	defer envvar.MockSourcegraphDotComMode(orig)

	versionContext := "versionCtx"
	tts := []struct {
		name           string
		searchQuery    string
		versionContext *string
		patternType    query.SearchType
		mode           search.GlobalSearchMode
	}{
		{name: "user search context", searchQuery: "foo context:@userA", mode: search.DefaultMode},
		{name: "structural search", searchQuery: "foo", patternType: query.SearchTypeStructural, mode: search.DefaultMode},
		{name: "version context", searchQuery: "foo", versionContext: &versionContext, mode: search.DefaultMode},
		{name: "repo", searchQuery: "foo repo:sourcegraph/sourcegraph", versionContext: &versionContext, mode: search.DefaultMode},
		{name: "repogroup", searchQuery: "foo repogroup:grp", versionContext: &versionContext, mode: search.DefaultMode},
		{name: "repohasfile", searchQuery: "foo repohasfile:bar", versionContext: &versionContext, mode: search.DefaultMode},
		{name: "global search context", searchQuery: "foo context:global", mode: search.ZoektGlobalSearch},
		{name: "global search", searchQuery: "foo", mode: search.ZoektGlobalSearch},
	}

	for _, tt := range tts {
		t.Run(tt.name, func(t *testing.T) {
			qinfo, err := query.ParseLiteral(tt.searchQuery)
			if err != nil {
				t.Fatal(err)
			}

			resolver := searchResolver{
				SearchInputs: &run.SearchInputs{
					Query:          qinfo,
					UserSettings:   &schema.Settings{},
					PatternType:    tt.patternType,
					VersionContext: tt.versionContext,
				},
			}

			p, _ := resolver.toTextParameters(resolver.Query)
			if p.Mode != tt.mode {
				t.Fatalf("got %+v, want %+v", p.Mode, tt.mode)
			}
		})
	}

}

func TestZeroElapsedMilliseconds(t *testing.T) {
	r := &SearchResultsResolver{}
	if got := r.ElapsedMilliseconds(); got != 0 {
		t.Fatalf("got %d, want %d", got, 0)
	}
}
