package graphqlbackend

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/zoekt"
	zoektquery "github.com/google/zoekt/query"
	zoektrpc "github.com/google/zoekt/rpc"
	"github.com/keegancsmith/sqlf"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/db"
	"github.com/sourcegraph/sourcegraph/internal/db/dbconn"
	"github.com/sourcegraph/sourcegraph/internal/db/dbtesting"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/search"
	searchbackend "github.com/sourcegraph/sourcegraph/internal/search/backend"
	"github.com/sourcegraph/sourcegraph/internal/search/query"
	"github.com/sourcegraph/sourcegraph/internal/symbols/protocol"
	"github.com/sourcegraph/sourcegraph/internal/trace"
	"github.com/sourcegraph/sourcegraph/schema"
)

// fakeSearcher is a zoekt.Searcher that returns a predefined search result.
type fakeSearcher struct {
	result *zoekt.SearchResult

	repos []*zoekt.RepoListEntry

	// Default all unimplemented zoekt.Searcher methods to panic.
	zoekt.Searcher
}

func (ss *fakeSearcher) Search(ctx context.Context, q zoektquery.Q, opts *zoekt.SearchOptions) (*zoekt.SearchResult, error) {
	if ss.result == nil {
		return &zoekt.SearchResult{}, nil
	}
	return ss.result, nil
}

func (ss *fakeSearcher) List(ctx context.Context, q zoektquery.Q) (*zoekt.RepoList, error) {
	return &zoekt.RepoList{Repos: ss.repos}, nil
}

func (ss *fakeSearcher) String() string {
	return fmt.Sprintf("fakeSearcher(result = %v, repos = %v)", ss.result, ss.repos)
}

type errorSearcher struct {
	err error

	// Default all unimplemented zoekt.Searcher methods to panic.
	zoekt.Searcher
}

func (es *errorSearcher) Search(ctx context.Context, q zoektquery.Q, opts *zoekt.SearchOptions) (*zoekt.SearchResult, error) {
	return nil, es.err
}

func TestIndexedSearch(t *testing.T) {
	zeroTimeoutCtx, cancel := context.WithTimeout(context.Background(), 0)
	defer cancel()
	type args struct {
		ctx             context.Context
		query           string
		patternInfo     *search.TextPatternInfo
		repos           []*search.RepositoryRevisions
		useFullDeadline bool
		results         []zoekt.FileMatch
		since           func(time.Time) time.Duration
	}

	reposHEAD := makeRepositoryRevisions("foo/bar", "foo/foobar")
	zoektRepos := []*zoekt.RepoListEntry{{
		Repository: zoekt.Repository{
			Name:     "foo/bar",
			Branches: []zoekt.RepositoryBranch{{Name: "HEAD", Version: "barHEADSHA"}, {Name: "dev", Version: "bardevSHA"}, {Name: "main", Version: "barmainSHA"}},
		},
	}, {
		Repository: zoekt.Repository{
			Name:     "foo/foobar",
			Branches: []zoekt.RepositoryBranch{{Name: "HEAD", Version: "foobarHEADSHA"}},
		},
	}}

	tests := []struct {
		name               string
		args               args
		wantMatchCount     int
		wantMatchURLs      []string
		wantMatchInputRevs []string
		wantUnindexed      []*search.RepositoryRevisions
		wantLimitHit       bool
		wantReposLimitHit  map[string]struct{}
		wantErr            bool
	}{
		{
			name: "no matches",
			args: args{
				ctx:             context.Background(),
				patternInfo:     &search.TextPatternInfo{},
				repos:           reposHEAD,
				useFullDeadline: false,
				since:           func(time.Time) time.Duration { return time.Second - time.Millisecond },
			},
			wantLimitHit:      false,
			wantReposLimitHit: nil,
			wantErr:           false,
		},
		{
			name: "no matches timeout",
			args: args{
				ctx:             context.Background(),
				patternInfo:     &search.TextPatternInfo{},
				repos:           reposHEAD,
				useFullDeadline: false,
				since:           func(time.Time) time.Duration { return time.Minute },
			},
			wantLimitHit:      false,
			wantReposLimitHit: nil,
			wantErr:           true,
		},
		{
			name: "context timeout",
			args: args{
				ctx:             zeroTimeoutCtx,
				patternInfo:     &search.TextPatternInfo{},
				repos:           reposHEAD,
				useFullDeadline: true,
				since:           func(time.Time) time.Duration { return 0 },
			},
			wantLimitHit:      false,
			wantReposLimitHit: nil,
			wantErr:           true,
		},
		{
			name: "results",
			args: args{
				ctx:             context.Background(),
				patternInfo:     &search.TextPatternInfo{FileMatchLimit: 100},
				repos:           makeRepositoryRevisions("foo/bar", "foo/foobar"),
				useFullDeadline: false,
				results: []zoekt.FileMatch{
					{
						Repository: "foo/bar",
						Branches:   []string{"HEAD"},
						FileName:   "baz.go",
						LineMatches: []zoekt.LineMatch{
							{
								Line: []byte("I'm like 1.5+ hours into writing this test :'("),
								LineFragments: []zoekt.LineFragmentMatch{
									{LineOffset: 0, MatchLength: 5},
								},
							},
							{
								Line: []byte("I'm ready for the rain to stop."),
								LineFragments: []zoekt.LineFragmentMatch{
									{LineOffset: 0, MatchLength: 5},
									{LineOffset: 5, MatchLength: 10},
								},
							},
						},
					},
					{
						Repository: "foo/foobar",
						Branches:   []string{"HEAD"},
						FileName:   "baz.go",
						LineMatches: []zoekt.LineMatch{
							{
								Line: []byte("s/rain/pain"),
								LineFragments: []zoekt.LineFragmentMatch{
									{LineOffset: 0, MatchLength: 5},
									{LineOffset: 5, MatchLength: 2},
								},
							},
						},
					},
				},
				since: func(time.Time) time.Duration { return 0 },
			},
			wantLimitHit:      false,
			wantReposLimitHit: map[string]struct{}{},
			wantMatchCount:    5,
			wantMatchURLs: []string{
				"git://foo/bar#baz.go",
				"git://foo/foobar#baz.go",
			},
			wantMatchInputRevs: []string{
				"",
				"",
			},
			wantErr: false,
		},
		{
			name: "results multi-branch",
			args: args{
				ctx:             context.Background(),
				patternInfo:     &search.TextPatternInfo{FileMatchLimit: 100},
				repos:           makeRepositoryRevisions("foo/bar@HEAD:dev:main"),
				useFullDeadline: false,
				results: []zoekt.FileMatch{
					{
						Repository: "foo/bar",
						// baz.go is the same in HEAD and dev
						Branches: []string{"HEAD", "dev"},
						FileName: "baz.go",
					},
					{
						Repository: "foo/bar",
						Branches:   []string{"dev"},
						FileName:   "bam.go",
					},
				},
				since: func(time.Time) time.Duration { return 0 },
			},
			wantLimitHit:      false,
			wantReposLimitHit: map[string]struct{}{},
			wantMatchURLs: []string{
				"git://foo/bar?HEAD#baz.go",
				"git://foo/bar?dev#baz.go",
				"git://foo/bar?dev#bam.go",
			},
			wantMatchInputRevs: []string{
				"HEAD",
				"dev",
				"dev",
			},
			wantErr: false,
		},
		{
			// if we search a branch that is indexed and unindexed, we should
			// split the repository revision into the indexed and unindexed
			// parts.
			name: "split branch",
			args: args{
				ctx:             context.Background(),
				patternInfo:     &search.TextPatternInfo{FileMatchLimit: 100},
				repos:           makeRepositoryRevisions("foo/bar@HEAD:unindexed"),
				useFullDeadline: false,
				results: []zoekt.FileMatch{
					{
						Repository: "foo/bar",
						Branches:   []string{"HEAD"},
						FileName:   "baz.go",
					},
				},
			},
			wantUnindexed: makeRepositoryRevisions("foo/bar@unindexed"),
			wantMatchURLs: []string{
				"git://foo/bar?HEAD#baz.go",
			},
			wantMatchInputRevs: []string{"HEAD"},
		},
		{
			// Fallback to unindexed search if the query contains ref-globs.
			name: "ref-glob with explicit /*",
			args: args{
				ctx:             context.Background(),
				query:           "repo:foo/bar@*refs/heads/*",
				patternInfo:     &search.TextPatternInfo{FileMatchLimit: 100},
				repos:           makeRepositoryRevisions("foo/bar@HEAD"),
				useFullDeadline: false,
				results:         []zoekt.FileMatch{},
			},
			wantUnindexed:      makeRepositoryRevisions("foo/bar@HEAD"),
			wantMatchURLs:      nil,
			wantMatchInputRevs: nil,
			wantLimitHit:       false,
		},
		{
			name: "ref-glob with implicit /*",
			args: args{
				ctx:             context.Background(),
				query:           "repo:foo/bar@*refs/tags",
				patternInfo:     &search.TextPatternInfo{FileMatchLimit: 100},
				repos:           makeRepositoryRevisions("foo/bar@HEAD"),
				useFullDeadline: false,
				results:         []zoekt.FileMatch{},
			},
			wantUnindexed:      makeRepositoryRevisions("foo/bar@HEAD"),
			wantMatchURLs:      nil,
			wantMatchInputRevs: nil,
			wantLimitHit:       false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := query.ParseAndCheck(tt.args.query)
			if err != nil {
				t.Fatal(err)
			}

			args := &search.TextParameters{
				Query:           q,
				PatternInfo:     tt.args.patternInfo,
				RepoPromise:     (&search.Promise{}).Resolve(tt.args.repos),
				UseFullDeadline: tt.args.useFullDeadline,
				Zoekt: &searchbackend.Zoekt{
					Client: &fakeSearcher{
						result: &zoekt.SearchResult{Files: tt.args.results},
						repos:  zoektRepos,
					},
					DisableCache: true,
				},
			}

			indexed, err := newIndexedSearchRequest(context.Background(), args, textRequest)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(tt.wantUnindexed, indexed.Unindexed, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("unindexed mismatch (-want +got):\n%s", diff)
			}

			indexed.since = tt.args.since

			gotFm, gotLimitHit, gotReposLimitHit, err := indexed.Search(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("zoektSearchHEAD() error = %v, wantErr = %v", err, tt.wantErr)
				return
			}
			if gotLimitHit != tt.wantLimitHit {
				t.Errorf("zoektSearchHEAD() gotLimitHit = %v, want %v", gotLimitHit, tt.wantLimitHit)
			}
			if diff := cmp.Diff(tt.wantReposLimitHit, gotReposLimitHit, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("reposLimitHit mismatch (-want +got):\n%s", diff)
			}

			var gotMatchCount int
			var gotMatchURLs []string
			var gotMatchInputRevs []string
			for _, m := range gotFm {
				gotMatchCount += m.MatchCount
				gotMatchURLs = append(gotMatchURLs, m.Resource())
				if m.InputRev != nil {
					gotMatchInputRevs = append(gotMatchInputRevs, *m.InputRev)
				}
			}
			if diff := cmp.Diff(tt.wantMatchURLs, gotMatchURLs); diff != "" {
				t.Errorf("match URLs mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.wantMatchInputRevs, gotMatchInputRevs); diff != "" {
				t.Errorf("match InputRevs mismatch (-want +got):\n%s", diff)
			}
			if gotMatchCount != tt.wantMatchCount {
				t.Errorf("gotMatchCount = %v, want %v", gotMatchCount, tt.wantMatchCount)
			}
		})
	}
}

func TestZoektIndexedRepos(t *testing.T) {
	repos := makeRepositoryRevisions(
		"foo/indexed-one@",
		"foo/indexed-two@",
		"foo/indexed-three@",
		"foo/unindexed-one",
		"foo/unindexed-two",
		"foo/multi-rev@a:b",
	)

	zoektRepos := map[string]*zoekt.Repository{}
	for _, r := range []*zoekt.Repository{{
		Name:     "foo/indexed-one",
		Branches: []zoekt.RepositoryBranch{{Name: "HEAD", Version: "deadbeef"}},
	}, {
		Name:     "foo/indexed-two",
		Branches: []zoekt.RepositoryBranch{{Name: "HEAD", Version: "deadbeef"}},
	}, {
		Name: "foo/indexed-three",
		Branches: []zoekt.RepositoryBranch{
			{Name: "HEAD", Version: "deadbeef"},
			{Name: "foobar", Version: "deadcow"},
		},
	}} {
		zoektRepos[r.Name] = r
	}

	makeIndexed := func(repos []*search.RepositoryRevisions) []*search.RepositoryRevisions {
		var indexed []*search.RepositoryRevisions
		for _, r := range repos {
			rev := &search.RepositoryRevisions{
				Repo: r.Repo,
				Revs: r.Revs,
			}
			indexed = append(indexed, rev)
		}
		return indexed
	}

	cases := []struct {
		name      string
		repos     []*search.RepositoryRevisions
		indexed   []*search.RepositoryRevisions
		unindexed []*search.RepositoryRevisions
	}{{
		name:      "all",
		repos:     repos,
		indexed:   makeIndexed(repos[:3]),
		unindexed: repos[3:],
	}, {
		name:      "one unindexed",
		repos:     repos[3:4],
		indexed:   repos[:0],
		unindexed: repos[3:4],
	}, {
		name:      "one indexed",
		repos:     repos[:1],
		indexed:   makeIndexed(repos[:1]),
		unindexed: repos[:0],
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			indexed, unindexed := zoektIndexedRepos(zoektRepos, tc.repos, nil)

			if diff := cmp.Diff(repoRevsSliceToMap(tc.indexed), indexed.repoRevs); diff != "" {
				t.Error("unexpected indexed:", diff)
			}
			if diff := cmp.Diff(tc.unindexed, unindexed); diff != "" {
				t.Error("unexpected unindexed:", diff)
			}
		})
	}
}

func Benchmark_zoektIndexedRepos(b *testing.B) {
	repoNames := []string{}
	zoektRepos := map[string]*zoekt.Repository{}

	for i := 0; i < 200000; i++ {
		indexedName := fmt.Sprintf("foo/indexed-%d@", i)
		unindexedName := fmt.Sprintf("foo/unindexed-%d@", i)

		repoNames = append(repoNames, indexedName, unindexedName)

		zoektName := strings.TrimSuffix(indexedName, "@")
		zoektRepos[zoektName] = &zoekt.Repository{
			Name:     zoektName,
			Branches: []zoekt.RepositoryBranch{{Name: "HEAD", Version: "deadbeef"}},
		}
	}

	repos := makeRepositoryRevisions(repoNames...)

	b.ResetTimer()
	b.ReportAllocs()

	for n := 0; n < b.N; n++ {
		_, _ = zoektIndexedRepos(zoektRepos, repos, nil)
	}
}

func TestZoektResultCountFactor(t *testing.T) {
	cases := []struct {
		name         string
		numRepos     int
		globalSearch bool
		pattern      *search.TextPatternInfo
		want         int
	}{
		{
			name:     "One repo implies max scaling factor",
			numRepos: 1,
			pattern:  &search.TextPatternInfo{},
			want:     100,
		},
		{
			name:     "Eleven repos implies a scaling factor between min and max",
			numRepos: 11,
			pattern:  &search.TextPatternInfo{},
			want:     8,
		},
		{
			name:     "More than 500 repos implies a min scaling factor",
			numRepos: 501,
			pattern:  &search.TextPatternInfo{},
			want:     1,
		},
		{
			name:     "Setting a count greater than defautl max results (30) adapts scaling factor",
			numRepos: 501,
			pattern:  &search.TextPatternInfo{FileMatchLimit: 100},
			want:     10,
		},
		{
			name:         "for global searches, k should be 1",
			numRepos:     0,
			globalSearch: true,
			pattern:      &search.TextPatternInfo{},
			want:         1,
		},
		{
			name:         "for global searches, k should be 1, adjusted by the FileMatchLimit",
			numRepos:     0,
			globalSearch: true,
			pattern:      &search.TextPatternInfo{FileMatchLimit: 100},
			want:         10,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := zoektResultCountFactor(tt.numRepos, tt.pattern.FileMatchLimit, tt.globalSearch)
			if tt.want != got {
				t.Fatalf("Want scaling factor %d but got %d", tt.want, got)
			}
		})
	}
}

func TestQueryToZoektQuery(t *testing.T) {
	cases := []struct {
		Name    string
		Type    indexedRequestType
		Pattern *search.TextPatternInfo
		Query   string
	}{
		{
			Name: "substr",
			Type: textRequest,
			Pattern: &search.TextPatternInfo{
				IsRegExp:                     true,
				IsCaseSensitive:              false,
				Pattern:                      "foo",
				IncludePatterns:              nil,
				ExcludePattern:               "",
				PathPatternsAreCaseSensitive: false,
			},
			Query: "foo case:no",
		},
		{
			Name: "symbol substr",
			Type: symbolRequest,
			Pattern: &search.TextPatternInfo{
				IsRegExp:                     true,
				IsCaseSensitive:              false,
				Pattern:                      "foo",
				IncludePatterns:              nil,
				ExcludePattern:               "",
				PathPatternsAreCaseSensitive: false,
			},
			Query: "sym:foo case:no",
		},
		{
			Name: "regex",
			Type: textRequest,
			Pattern: &search.TextPatternInfo{
				IsRegExp:                     true,
				IsCaseSensitive:              false,
				Pattern:                      "(foo).*?(bar)",
				IncludePatterns:              nil,
				ExcludePattern:               "",
				PathPatternsAreCaseSensitive: false,
			},
			Query: "(foo).*?(bar) case:no",
		},
		{
			Name: "path",
			Type: textRequest,
			Pattern: &search.TextPatternInfo{
				IsRegExp:                     true,
				IsCaseSensitive:              false,
				Pattern:                      "foo",
				IncludePatterns:              []string{`\.go$`, `\.yaml$`},
				ExcludePattern:               `\bvendor\b`,
				PathPatternsAreCaseSensitive: false,
			},
			Query: `foo case:no f:\.go$ f:\.yaml$ -f:\bvendor\b`,
		},
		{
			Name: "case",
			Type: textRequest,
			Pattern: &search.TextPatternInfo{
				IsRegExp:                     true,
				IsCaseSensitive:              true,
				Pattern:                      "foo",
				IncludePatterns:              []string{`\.go$`, `yaml`},
				ExcludePattern:               "",
				PathPatternsAreCaseSensitive: true,
			},
			Query: `foo case:yes f:\.go$ f:yaml`,
		},
		{
			Name: "casepath",
			Type: textRequest,
			Pattern: &search.TextPatternInfo{
				IsRegExp:                     true,
				IsCaseSensitive:              true,
				Pattern:                      "foo",
				IncludePatterns:              []string{`\.go$`, `\.yaml$`},
				ExcludePattern:               `\bvendor\b`,
				PathPatternsAreCaseSensitive: true,
			},
			Query: `foo case:yes f:\.go$ f:\.yaml$ -f:\bvendor\b`,
		},
		{
			Name: "path matches only",
			Type: textRequest,
			Pattern: &search.TextPatternInfo{
				IsRegExp:                     true,
				IsCaseSensitive:              false,
				Pattern:                      "test",
				IncludePatterns:              []string{},
				ExcludePattern:               ``,
				PathPatternsAreCaseSensitive: true,
				PatternMatchesContent:        false,
				PatternMatchesPath:           true,
			},
			Query: `f:test`,
		},
		{
			Name: "repos must include",
			Type: textRequest,
			Pattern: &search.TextPatternInfo{
				IsRegExp:                     true,
				Pattern:                      "foo",
				FilePatternsReposMustInclude: []string{`\.go$`, `\.yaml$`},
				FilePatternsReposMustExclude: []string{`\.java$`, `\.xml$`},
			},
			Query: `foo (type:repo file:\.go$) (type:repo file:\.yaml$) -(type:repo file:\.java$) -(type:repo file:\.xml$)`,
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			q, err := zoektquery.Parse(tt.Query)
			if err != nil {
				t.Fatalf("failed to parse %q: %v", tt.Query, err)
			}
			got, err := queryToZoektQuery(tt.Pattern, tt.Type)
			if err != nil {
				t.Fatal("queryToZoektQuery failed:", err)
			}
			if !queryEqual(got, q) {
				t.Fatalf("mismatched queries\ngot  %s\nwant %s", got.String(), q.String())
			}
		})
	}
}

func queryEqual(a, b zoektquery.Q) bool {
	sortChildren := func(q zoektquery.Q) zoektquery.Q {
		switch s := q.(type) {
		case *zoektquery.And:
			sort.Slice(s.Children, func(i, j int) bool {
				return s.Children[i].String() < s.Children[j].String()
			})
		case *zoektquery.Or:
			sort.Slice(s.Children, func(i, j int) bool {
				return s.Children[i].String() < s.Children[j].String()
			})
		}
		return q
	}
	return zoektquery.Map(a, sortChildren).String() == zoektquery.Map(b, sortChildren).String()
}

func BenchmarkSearchResults(b *testing.B) {
	minimalRepos, _, zoektRepos := generateRepos(5000)
	zoektFileMatches := generateZoektMatches(50)

	z := &searchbackend.Zoekt{
		Client: &fakeSearcher{
			repos:  zoektRepos,
			result: &zoekt.SearchResult{Files: zoektFileMatches},
		},
		DisableCache: true,
	}

	ctx := context.Background()

	db.Mocks.Repos.List = func(_ context.Context, op db.ReposListOptions) ([]*types.Repo, error) {
		return minimalRepos, nil
	}
	db.Mocks.Repos.Count = func(ctx context.Context, opt db.ReposListOptions) (int, error) {
		return len(minimalRepos), nil
	}
	defer func() { db.Mocks = db.MockStores{} }()

	b.ResetTimer()
	b.ReportAllocs()

	for n := 0; n < b.N; n++ {
		q, err := query.ProcessAndOr(`print index:only count:350`, query.ParserOptions{SearchType: query.SearchTypeLiteral})
		if err != nil {
			b.Fatal(err)
		}
		resolver := &searchResolver{query: q, zoekt: z, userSettings: &schema.Settings{}}
		results, err := resolver.Results(ctx)
		if err != nil {
			b.Fatal("Results:", err)
		}
		if int(results.MatchCount()) != len(zoektFileMatches) {
			b.Fatalf("wrong results length. want=%d, have=%d\n", len(zoektFileMatches), results.MatchCount())
		}
	}
}

func BenchmarkIntegrationSearchResults(b *testing.B) {
	dbtesting.SetupGlobalTestDB(b)

	ctx := context.Background()

	_, repos, zoektRepos := generateRepos(5000)
	zoektFileMatches := generateZoektMatches(50)

	zoektClient, cleanup := zoektRPC(&fakeSearcher{
		repos:  zoektRepos,
		result: &zoekt.SearchResult{Files: zoektFileMatches},
	})
	defer cleanup()
	z := &searchbackend.Zoekt{
		Client:       zoektClient,
		DisableCache: true,
	}

	rows := make([]*sqlf.Query, 0, len(repos))
	for _, r := range repos {
		rows = append(rows, sqlf.Sprintf(
			"(%s, %s, %s, %s, %s, %s, %s)",
			r.Name,
			r.Description,
			r.Fork,
			true,
			r.ExternalRepo.ServiceType,
			r.ExternalRepo.ServiceID,
			r.ExternalRepo.ID,
		))
	}

	q := sqlf.Sprintf(`
		INSERT INTO repo (
			name,
			description,
			fork,
			enabled,
			external_service_type,
			external_service_id,
			external_id
		)
		VALUES %s`,
		sqlf.Join(rows, ","),
	)

	_, err := dbconn.Global.ExecContext(ctx, q.Query(sqlf.PostgresBindVar), q.Args()...)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for n := 0; n < b.N; n++ {
		q, err := query.ParseAndCheck(`print index:only count:350`)
		if err != nil {
			b.Fatal(err)
		}
		resolver := &searchResolver{query: q, zoekt: z}
		results, err := resolver.Results(ctx)
		if err != nil {
			b.Fatal("Results:", err)
		}
		if int(results.MatchCount()) != len(zoektFileMatches) {
			b.Fatalf("wrong results length. want=%d, have=%d\n", len(zoektFileMatches), results.MatchCount())
		}
	}
}

func generateRepos(count int) ([]*types.Repo, []*types.Repo, []*zoekt.RepoListEntry) {
	var reposWithIDs []*types.Repo
	var repos []*types.Repo
	var zoektRepos []*zoekt.RepoListEntry

	for i := 1; i <= count; i++ {
		name := fmt.Sprintf("repo-%d", i)

		repoWithIDs := &types.Repo{
			ID:   api.RepoID(i),
			Name: api.RepoName(name),
			ExternalRepo: api.ExternalRepoSpec{
				ID:          name,
				ServiceType: extsvc.TypeGitHub,
				ServiceID:   "https://github.com",
			}}

		reposWithIDs = append(reposWithIDs, repoWithIDs)

		repos = append(repos, &types.Repo{

			ID:           repoWithIDs.ID,
			Name:         repoWithIDs.Name,
			ExternalRepo: repoWithIDs.ExternalRepo,

			RepoFields: &types.RepoFields{
				URI:         fmt.Sprintf("https://github.com/foobar/%s", repoWithIDs.Name),
				Description: "this repositoriy contains a side project that I haven't maintained in 2 years",
			}})

		zoektRepos = append(zoektRepos, &zoekt.RepoListEntry{
			Repository: zoekt.Repository{
				Name:     name,
				Branches: []zoekt.RepositoryBranch{{Name: "HEAD", Version: "deadbeef"}},
			},
		})
	}
	return reposWithIDs, repos, zoektRepos
}

func generateZoektMatches(count int) []zoekt.FileMatch {
	var zoektFileMatches []zoekt.FileMatch
	for i := 1; i <= count; i++ {
		repoName := fmt.Sprintf("repo-%d", i)
		fileName := fmt.Sprintf("foobar-%d.go", i)

		zoektFileMatches = append(zoektFileMatches, zoekt.FileMatch{
			Score:      5.0,
			FileName:   fileName,
			Repository: repoName, // Important: this needs to match a name in `repos`
			Branches:   []string{"master"},
			LineMatches: []zoekt.LineMatch{
				{
					Line: nil,
				},
			},
			Checksum: []byte{0, 1, 2},
		})
	}
	return zoektFileMatches
}

// zoektRPC starts zoekts rpc interface and returns a client to
// searcher. Useful for capturing CPU/memory usage when benchmarking the zoekt
// client.
func zoektRPC(s zoekt.Searcher) (zoekt.Searcher, func()) {
	mux := http.NewServeMux()
	mux.Handle(zoektrpc.DefaultRPCPath, zoektrpc.Server(s))
	ts := httptest.NewServer(mux)
	cl := zoektrpc.Client(strings.TrimPrefix(ts.URL, "http://"))
	return cl, func() {
		cl.Close()
		ts.Close()
	}
}

func TestZoektIndexedRepos_single(t *testing.T) {
	repoRev := func(revSpec string) *search.RepositoryRevisions {
		return &search.RepositoryRevisions{
			Repo: &types.Repo{ID: api.RepoID(0), Name: "test/repo"},
			Revs: []search.RevisionSpecifier{
				{RevSpec: revSpec},
			},
		}
	}
	zoektRepos := map[string]*zoekt.Repository{
		"test/repo": {
			Name: "test/repo",
			Branches: []zoekt.RepositoryBranch{
				{
					Name:    "HEAD",
					Version: "df3f4e499698e48152b39cd655d8901eaf583fa5",
				},
				{
					Name:    "NOT-HEAD",
					Version: "8ec975423738fe7851676083ebf660a062ed1578",
				},
			},
		},
	}
	cases := []struct {
		rev           string
		wantIndexed   []*search.RepositoryRevisions
		wantUnindexed []*search.RepositoryRevisions
	}{
		{
			rev:           "",
			wantIndexed:   []*search.RepositoryRevisions{repoRev("")},
			wantUnindexed: []*search.RepositoryRevisions{},
		},
		{
			rev:           "HEAD",
			wantIndexed:   []*search.RepositoryRevisions{repoRev("HEAD")},
			wantUnindexed: []*search.RepositoryRevisions{},
		},
		{
			rev:           "df3f4e499698e48152b39cd655d8901eaf583fa5",
			wantIndexed:   []*search.RepositoryRevisions{repoRev("df3f4e499698e48152b39cd655d8901eaf583fa5")},
			wantUnindexed: []*search.RepositoryRevisions{},
		},
		{
			rev:           "df3f4e",
			wantIndexed:   []*search.RepositoryRevisions{repoRev("df3f4e")},
			wantUnindexed: []*search.RepositoryRevisions{},
		},
		{
			rev:           "d",
			wantIndexed:   []*search.RepositoryRevisions{},
			wantUnindexed: []*search.RepositoryRevisions{repoRev("d")},
		},
		{
			rev:           "HEAD^1",
			wantIndexed:   []*search.RepositoryRevisions{},
			wantUnindexed: []*search.RepositoryRevisions{repoRev("HEAD^1")},
		},
		{
			rev:           "8ec975423738fe7851676083ebf660a062ed1578",
			wantUnindexed: []*search.RepositoryRevisions{},
			wantIndexed:   []*search.RepositoryRevisions{repoRev("8ec975423738fe7851676083ebf660a062ed1578")},
		},
	}

	type ret struct {
		Indexed   map[string]*search.RepositoryRevisions
		Unindexed []*search.RepositoryRevisions
	}

	for _, tt := range cases {
		indexed, unindexed := zoektIndexedRepos(zoektRepos, []*search.RepositoryRevisions{repoRev(tt.rev)}, nil)
		got := ret{
			Indexed:   indexed.repoRevs,
			Unindexed: unindexed,
		}
		want := ret{
			Indexed:   repoRevsSliceToMap(tt.wantIndexed),
			Unindexed: tt.wantUnindexed,
		}
		if !cmp.Equal(want, got) {
			t.Errorf("%s mismatch (-want +got):\n%s", tt.rev, cmp.Diff(want, got))
		}
	}
}

func TestZoektFileMatchToSymbolResults(t *testing.T) {
	symbolInfo := func(sym string) *zoekt.Symbol {
		return &zoekt.Symbol{
			Sym:        sym,
			Kind:       "kind",
			Parent:     "parent",
			ParentKind: "parentkind",
		}
	}

	file := &zoekt.FileMatch{
		FileName:   "bar.go",
		Repository: "foo",
		Language:   "go",
		Version:    "deadbeef",
		LineMatches: []zoekt.LineMatch{{
			// Skips missing symbol info (shouldn't happen in practice).
			LineNumber:    5,
			LineFragments: []zoekt.LineFragmentMatch{{}},
		}, {
			LineNumber: 10,
			LineFragments: []zoekt.LineFragmentMatch{{
				SymbolInfo: symbolInfo("a"),
			}, {
				SymbolInfo: symbolInfo("b"),
			}},
		}, {
			LineNumber: 15,
			LineFragments: []zoekt.LineFragmentMatch{{
				SymbolInfo: symbolInfo("c"),
			}},
		}},
	}

	repo := &RepositoryResolver{repo: &types.Repo{Name: "foo"}}

	results := zoektFileMatchToSymbolResults(repo, "master", file)
	var symbols []protocol.Symbol
	for _, res := range results {
		// Check the fields which are not specific to the symbol
		if got, want := res.lang, "go"; got != want {
			t.Fatalf("lang: got %q want %q", got, want)
		}
		if got, want := res.baseURI.URL.String(), "git://foo?master"; got != want {
			t.Fatalf("baseURI: got %q want %q", got, want)
		}
		if got, want := string(res.commit.repoResolver.repo.Name), "foo"; got != want {
			t.Fatalf("reporesolver: got %q want %q", got, want)
		}
		if got, want := string(res.commit.oid), "deadbeef"; got != want {
			t.Fatalf("oid: got %q want %q", got, want)
		}
		if got, want := *res.commit.inputRev, "master"; got != want {
			t.Fatalf("inputRev: got %q want %q", got, want)
		}

		symbols = append(symbols, res.symbol)
	}

	want := []protocol.Symbol{{
		Name: "a",
		Line: 10,
	}, {
		Name: "b",
		Line: 10,
	}, {
		Name: "c",
		Line: 15,
	}}
	for i := range want {
		want[i].Kind = "kind"
		want[i].Parent = "parent"
		want[i].ParentKind = "parentkind"
		want[i].Path = "bar.go"
	}

	if diff := cmp.Diff(want, symbols); diff != "" {
		t.Fatalf("symbol mismatch (-want +got):\n%s", diff)
	}
}

func repoRevsSliceToMap(rs []*search.RepositoryRevisions) map[string]*search.RepositoryRevisions {
	m := map[string]*search.RepositoryRevisions{}
	for _, r := range rs {
		m[string(r.Repo.Name)] = r
	}
	return m
}

func TestContainsRefGlobs(t *testing.T) {
	tests := []struct {
		query    string
		want     bool
		globbing bool
	}{
		{
			query: "repo:foo",
			want:  false,
		},
		{
			query: "repo:foo@bar",
			want:  false,
		},
		{
			query: "repo:foo@*ref/tags",
			want:  true,
		},
		{
			query: "repo:foo@*!refs/tags",
			want:  true,
		},
		{
			query: "repo:foo@bar:*refs/heads",
			want:  true,
		},
		{
			query: "repo:foo@refs/tags/v3.14.3",
			want:  false,
		},
		{
			query: "repo:foo@*refs/tags/v3.14.?",
			want:  true,
		},
		{
			query:    "repo:*foo*@v3.14.3",
			globbing: true,
			want:     false,
		},
		{
			query: "repo:foo@v3.14.3 repo:foo@*refs/tags/v3.14.* bar",
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			qInfo, err := query.ProcessAndOr(tt.query, query.ParserOptions{SearchType: query.SearchTypeLiteral, Globbing: tt.globbing})
			if err != nil {
				t.Error(err)
			}
			got := containsRefGlobs(qInfo)
			if got != tt.want {
				t.Errorf("got %t, expected %t", got, tt.want)
			}
		})
	}
}

func TestContextWithoutDeadline(t *testing.T) {
	ctxWithDeadline, cancelWithDeadline := context.WithTimeout(context.Background(), time.Minute)
	defer cancelWithDeadline()

	tr, ctxWithDeadline := trace.New(ctxWithDeadline, "", "")

	if _, ok := ctxWithDeadline.Deadline(); !ok {
		t.Fatal("expected context to have deadline")
	}

	ctxNoDeadline, cancelNoDeadline := contextWithoutDeadline(ctxWithDeadline)
	defer cancelNoDeadline()

	if _, ok := ctxNoDeadline.Deadline(); ok {
		t.Fatal("expected context to not have deadline")
	}

	// We want to keep trace info
	if tr2 := trace.TraceFromContext(ctxNoDeadline); tr != tr2 {
		t.Error("trace information not propogated")
	}

	// Calling cancelWithDeadline should cancel ctxNoDeadline
	cancelWithDeadline()
	select {
	case <-ctxNoDeadline.Done():
	case <-time.After(10 * time.Second):
		t.Fatal("expected context to be done")
	}
}

func TestContextWithoutDeadline_cancel(t *testing.T) {
	ctxWithDeadline, cancelWithDeadline := context.WithTimeout(context.Background(), time.Minute)
	defer cancelWithDeadline()
	ctxNoDeadline, cancelNoDeadline := contextWithoutDeadline(ctxWithDeadline)

	cancelNoDeadline()
	select {
	case <-ctxNoDeadline.Done():
	case <-time.After(10 * time.Second):
		t.Fatal("expected context to be done")
	}
}
