package graphqlbackend

import (
	"context"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"

	"github.com/google/zoekt"
	zoektquery "github.com/google/zoekt/query"
	"github.com/pkg/errors"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/internal/pkg/search"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/internal/pkg/search/query"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/pkg/api"
	"github.com/sourcegraph/sourcegraph/pkg/errcode"
	"github.com/sourcegraph/sourcegraph/pkg/gitserver"
	"github.com/sourcegraph/sourcegraph/pkg/vcs"
	"github.com/sourcegraph/sourcegraph/pkg/vcs/git"
)

func TestQueryToZoektQuery(t *testing.T) {
	cases := []struct {
		Name    string
		Pattern *search.PatternInfo
		Query   string
	}{
		{
			Name: "substr",
			Pattern: &search.PatternInfo{
				IsRegExp:                     true,
				IsCaseSensitive:              false,
				Pattern:                      "foo",
				IncludePatterns:              nil,
				ExcludePattern:               "",
				PathPatternsAreRegExps:       true,
				PathPatternsAreCaseSensitive: false,
			},
			Query: "foo case:no",
		},
		{
			Name: "regex",
			Pattern: &search.PatternInfo{
				IsRegExp:                     true,
				IsCaseSensitive:              false,
				Pattern:                      "(foo).*?(bar)",
				IncludePatterns:              nil,
				ExcludePattern:               "",
				PathPatternsAreRegExps:       true,
				PathPatternsAreCaseSensitive: false,
			},
			Query: "(foo).*?(bar) case:no",
		},
		{
			Name: "path",
			Pattern: &search.PatternInfo{
				IsRegExp:                     true,
				IsCaseSensitive:              false,
				Pattern:                      "foo",
				IncludePatterns:              []string{`\.go$`, `\.yaml$`},
				ExcludePattern:               `\bvendor\b`,
				PathPatternsAreRegExps:       true,
				PathPatternsAreCaseSensitive: false,
			},
			Query: `foo case:no f:\.go$ f:\.yaml$ -f:\bvendor\b`,
		},
		{
			Name: "case",
			Pattern: &search.PatternInfo{
				IsRegExp:                     true,
				IsCaseSensitive:              true,
				Pattern:                      "foo",
				IncludePatterns:              []string{`\.go$`, `yaml`},
				ExcludePattern:               "",
				PathPatternsAreRegExps:       true,
				PathPatternsAreCaseSensitive: true,
			},
			Query: `foo case:yes f:\.go$ f:yaml`,
		},
		{
			Name: "casepath",
			Pattern: &search.PatternInfo{
				IsRegExp:                     true,
				IsCaseSensitive:              true,
				Pattern:                      "foo",
				IncludePatterns:              []string{`\.go$`, `\.yaml$`},
				ExcludePattern:               `\bvendor\b`,
				PathPatternsAreRegExps:       true,
				PathPatternsAreCaseSensitive: true,
			},
			Query: `foo case:yes f:\.go$ f:\.yaml$ -f:\bvendor\b`,
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			q, err := zoektquery.Parse(tt.Query)
			if err != nil {
				t.Fatalf("failed to parse %q: %v", tt.Query, err)
			}
			got, err := queryToZoektQuery(tt.Pattern)
			if err != nil {
				t.Fatal("queryToZoektQuery failed:", err)
			}
			if !queryEqual(got, q) {
				t.Fatalf("mismatched queries\ngot  %s\nwant %s", got.String(), q.String())
			}
		})
	}
}

func queryEqual(a zoektquery.Q, b zoektquery.Q) bool {
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

func TestQueryToZoektFileOnlyQuery(t *testing.T) {
	cases := []struct {
		Name    string
		Pattern *search.PatternInfo
		Query   string
		// This should be the same value passed in to either FilePatternsReposMustInclude or FilePatternsReposMustExclude
		ListOfFilePaths []string
	}{
		{
			Name: "single repohasfile filter",
			Pattern: &search.PatternInfo{
				IsRegExp:                     true,
				IsCaseSensitive:              false,
				Pattern:                      "foo",
				IncludePatterns:              nil,
				ExcludePattern:               "",
				FilePatternsReposMustInclude: []string{"test.md"},
				PathPatternsAreRegExps:       true,
				PathPatternsAreCaseSensitive: false,
			},
			Query:           `f:"test.md"`,
			ListOfFilePaths: []string{"test.md"},
		},
		{
			Name: "multiple repohasfile filters",
			Pattern: &search.PatternInfo{
				IsRegExp:                     true,
				IsCaseSensitive:              false,
				Pattern:                      "foo",
				IncludePatterns:              nil,
				ExcludePattern:               "",
				FilePatternsReposMustInclude: []string{"t", "d"},
				PathPatternsAreRegExps:       true,
				PathPatternsAreCaseSensitive: false,
			},
			Query:           `f:"t" f:"d"`,
			ListOfFilePaths: []string{"t", "d"},
		},
		{
			Name: "single negated repohasfile filter",
			Pattern: &search.PatternInfo{
				IsRegExp:                     true,
				IsCaseSensitive:              false,
				Pattern:                      "foo",
				IncludePatterns:              nil,
				ExcludePattern:               "",
				FilePatternsReposMustExclude: []string{"test.md"},
				PathPatternsAreRegExps:       true,
				PathPatternsAreCaseSensitive: false,
			},
			Query:           `f:"test.md"`,
			ListOfFilePaths: []string{"test.md"},
		},
		{
			Name: "multiple negated repohasfile filter",
			Pattern: &search.PatternInfo{
				IsRegExp:                     true,
				IsCaseSensitive:              false,
				Pattern:                      "foo",
				IncludePatterns:              nil,
				ExcludePattern:               "",
				FilePatternsReposMustExclude: []string{"t", "d"},
				PathPatternsAreRegExps:       true,
				PathPatternsAreCaseSensitive: false,
			},
			Query:           `f:"t" f:"d"`,
			ListOfFilePaths: []string{"t", "d"},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			q, err := zoektquery.Parse(tt.Query)
			if err != nil {
				t.Fatalf("failed to parse %q: %v", tt.Query, err)
			}
			got, err := queryToZoektFileOnlyQuery(tt.Pattern, tt.ListOfFilePaths)
			if err != nil {
				t.Fatal("queryToZoektQuery failed:", err)
			}
			if !queryEqual(got, q) {
				t.Fatalf("mismatched queries\ngot  %s\nwant %s", got.String(), q.String())
			}
		})
	}
}

func TestSearchFilesInRepos(t *testing.T) {
	mockSearchFilesInRepo = func(ctx context.Context, repo *types.Repo, gitserverRepo gitserver.Repo, rev string, info *search.PatternInfo, fetchTimeout time.Duration) (matches []*fileMatchResolver, limitHit bool, err error) {
		repoName := repo.Name
		switch repoName {
		case "foo/one":
			return []*fileMatchResolver{
				{
					uri: "git://" + string(repoName) + "?" + rev + "#" + "main.go",
				},
			}, false, nil
		case "foo/two":
			return []*fileMatchResolver{
				{
					uri: "git://" + string(repoName) + "?" + rev + "#" + "main.go",
				},
			}, false, nil
		case "foo/empty":
			return nil, false, nil
		case "foo/cloning":
			return nil, false, &vcs.RepoNotExistError{Repo: repoName, CloneInProgress: true}
		case "foo/missing":
			return nil, false, &vcs.RepoNotExistError{Repo: repoName}
		case "foo/missing-db":
			return nil, false, &errcode.Mock{Message: "repo not found: foo/missing-db", IsNotFound: true}
		case "foo/timedout":
			return nil, false, context.DeadlineExceeded
		case "foo/no-rev":
			return nil, false, &git.RevisionNotFoundError{Repo: repoName, Spec: "missing"}
		default:
			return nil, false, errors.New("Unexpected repo")
		}
	}
	defer func() { mockSearchFilesInRepo = nil }()

	q, err := query.ParseAndCheck("foo")
	if err != nil {
		t.Fatal(err)
	}
	args := &search.Args{
		Pattern: &search.PatternInfo{
			FileMatchLimit: defaultMaxSearchResults,
			Pattern:        "foo",
		},
		Repos: makeRepositoryRevisions("foo/one", "foo/two", "foo/empty", "foo/cloning", "foo/missing", "foo/missing-db", "foo/timedout", "foo/no-rev"),
		Query: q,
	}
	results, common, err := searchFilesInRepos(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Errorf("expected two results, got %d", len(results))
	}
	if v := toRepoNames(common.cloning); !reflect.DeepEqual(v, []api.RepoName{"foo/cloning"}) {
		t.Errorf("unexpected cloning: %v", v)
	}
	sort.Slice(common.missing, func(i, j int) bool { return common.missing[i].Name < common.missing[j].Name }) // to make deterministic
	if v := toRepoNames(common.missing); !reflect.DeepEqual(v, []api.RepoName{"foo/missing", "foo/missing-db"}) {
		t.Errorf("unexpected missing: %v", v)
	}
	if v := toRepoNames(common.timedout); !reflect.DeepEqual(v, []api.RepoName{"foo/timedout"}) {
		t.Errorf("unexpected timedout: %v", v)
	}

	// If we specify a rev and it isn't found, we fail the whole search since
	// that should be checked earlier.
	args = &search.Args{
		Pattern: &search.PatternInfo{
			FileMatchLimit: defaultMaxSearchResults,
			Pattern:        "foo",
		},
		Repos: makeRepositoryRevisions("foo/no-rev@dev"),
		Query: q,
	}
	_, _, err = searchFilesInRepos(context.Background(), args)
	if !git.IsRevisionNotFound(errors.Cause(err)) {
		t.Fatalf("searching non-existent rev expected to fail with RevisionNotFoundError got: %v", err)
	}
}

func makeRepositoryRevisions(repos ...string) []*search.RepositoryRevisions {
	r := make([]*search.RepositoryRevisions, len(repos))
	for i, repospec := range repos {
		repoName, revs := search.ParseRepositoryRevisions(repospec)
		if len(revs) == 0 {
			// treat empty list as preferring master
			revs = []search.RevisionSpecifier{{RevSpec: ""}}
		}
		r[i] = &search.RepositoryRevisions{Repo: &types.Repo{Name: repoName}, Revs: revs}
	}
	return r
}

// fakeSearcher is a zoekt.Searcher that returns a predefined search result.
type fakeSearcher struct {
	result *zoekt.SearchResult

	// Default all unimplemented zoekt.Searcher methods to panic.
	zoekt.Searcher
}

func (ss *fakeSearcher) Search(ctx context.Context, q zoektquery.Q, opts *zoekt.SearchOptions) (*zoekt.SearchResult, error) {
	return ss.result, nil
}

type errorSearcher struct {
	err error

	// Default all unimplemented zoekt.Searcher methods to panic.
	zoekt.Searcher
}

func (es *errorSearcher) Search(ctx context.Context, q zoektquery.Q, opts *zoekt.SearchOptions) (*zoekt.SearchResult, error) {
	return nil, es.err
}

func Test_zoektSearchHEAD(t *testing.T) {
	zeroTimeoutCtx, cancel := context.WithTimeout(context.Background(), 0)
	defer cancel()
	type args struct {
		ctx              context.Context
		query            *search.PatternInfo
		indexedRevisions map[*search.RepositoryRevisions]string
		repos            []*search.RepositoryRevisions
		useFullDeadline  bool
		searcher         zoekt.Searcher
		opts             zoekt.SearchOptions
		since            func(time.Time) time.Duration
	}

	singleRepositoryRevisions := []*search.RepositoryRevisions{
		{Repo: &types.Repo{}},
	}
	singleIndexedRevisions := map[*search.RepositoryRevisions]string{
		singleRepositoryRevisions[0]: "abc",
	}

	tests := []struct {
		name              string
		args              args
		wantFm            []*fileMatchResolver
		wantLimitHit      bool
		wantReposLimitHit map[string]struct{}
		wantErr           bool
	}{
		{
			name: "returns no error if search completed with no matches before timeout",
			args: args{
				ctx:              context.Background(),
				query:            &search.PatternInfo{PathPatternsAreRegExps: true},
				indexedRevisions: singleIndexedRevisions,
				repos:            singleRepositoryRevisions,
				useFullDeadline:  false,
				searcher:         &fakeSearcher{result: &zoekt.SearchResult{}},
				opts:             zoekt.SearchOptions{MaxWallTime: time.Second},
				since:            func(time.Time) time.Duration { return time.Second - time.Millisecond },
			},
			wantFm:            nil,
			wantLimitHit:      false,
			wantReposLimitHit: nil,
			wantErr:           false,
		},
		{
			name: "returns error if max wall time is exceeded but no matches have been found yet",
			args: args{
				ctx:              context.Background(),
				query:            &search.PatternInfo{PathPatternsAreRegExps: true},
				indexedRevisions: singleIndexedRevisions,
				repos:            singleRepositoryRevisions,
				useFullDeadline:  false,
				searcher:         &fakeSearcher{result: &zoekt.SearchResult{}},
				opts:             zoekt.SearchOptions{MaxWallTime: time.Second},
				since:            func(time.Time) time.Duration { return time.Second },
			},
			wantFm:            nil,
			wantLimitHit:      false,
			wantReposLimitHit: nil,
			wantErr:           true,
		},
		{
			name: "returns error if context timeout already passed",
			args: args{
				ctx:              zeroTimeoutCtx,
				query:            &search.PatternInfo{PathPatternsAreRegExps: true},
				indexedRevisions: singleIndexedRevisions,
				repos:            singleRepositoryRevisions,
				useFullDeadline:  true,
				searcher:         &fakeSearcher{result: &zoekt.SearchResult{}},
				opts:             zoekt.SearchOptions{},
				since:            func(time.Time) time.Duration { return 0 },
			},
			wantFm:            nil,
			wantLimitHit:      false,
			wantReposLimitHit: nil,
			wantErr:           true,
		},
		{
			name: "returns error if searcher returns an error",
			args: args{
				ctx:              context.Background(),
				query:            &search.PatternInfo{PathPatternsAreRegExps: true},
				indexedRevisions: singleIndexedRevisions,
				repos:            singleRepositoryRevisions,
				useFullDeadline:  true,
				searcher:         &errorSearcher{err: errors.New("womp womp")},
				opts:             zoekt.SearchOptions{},
				since:            func(time.Time) time.Duration { return 0 },
			},
			wantFm:            nil,
			wantLimitHit:      false,
			wantReposLimitHit: nil,
			wantErr:           true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFm, gotLimitHit, gotReposLimitHit, err := zoektSearchHEAD(tt.args.ctx, tt.args.query, tt.args.repos, tt.args.indexedRevisions, tt.args.useFullDeadline, tt.args.searcher, tt.args.opts, tt.args.since)
			if (err != nil) != tt.wantErr {
				t.Errorf("zoektSearchHEAD() error = %v, wantErr = %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotFm, tt.wantFm) {
				t.Errorf("zoektSearchHEAD() gotFm = %v, want %v", gotFm, tt.wantFm)
			}
			if gotLimitHit != tt.wantLimitHit {
				t.Errorf("zoektSearchHEAD() gotLimitHit = %v, want %v", gotLimitHit, tt.wantLimitHit)
			}
			if !reflect.DeepEqual(gotReposLimitHit, tt.wantReposLimitHit) {
				t.Errorf("zoektSearchHEAD() gotReposLimitHit = %v, want %v", gotReposLimitHit, tt.wantReposLimitHit)
			}
		})
	}
}

func Test_createNewRepoSetWithRepoHasFileInputs(t *testing.T) {
	type args struct {
		ctx                             context.Context
		queryPatternInfo                *search.PatternInfo
		searcher                        zoekt.Searcher
		repoSet                         zoektquery.RepoSet
		repoHasFileFlagIsInQuery        bool
		negatedRepoHasFileFlagIsInQuery bool
	}

	tests := []struct {
		name        string
		args        args
		wantRepoSet *zoektquery.RepoSet
	}{
		{
			name: "returns filtered repoSet when repoHasFileFlag is in query",
			args: args{
				queryPatternInfo: &search.PatternInfo{FilePatternsReposMustInclude: []string{"1"}, PathPatternsAreRegExps: true},
				searcher: &fakeSearcher{result: &zoekt.SearchResult{
					Files: []zoekt.FileMatch{{
						FileName:   "1.md",
						Repository: "github.com/test/1",
						LineMatches: []zoekt.LineMatch{{
							FileName: true,
						}}},
					},
					RepoURLs: map[string]string{"github.com/test/1": "github.com/test/1"}}},
				repoSet:                         zoektquery.RepoSet{Set: map[string]bool{"github.com/test/1": true, "github.com/test/2": true}},
				repoHasFileFlagIsInQuery:        true,
				negatedRepoHasFileFlagIsInQuery: false,
			},
			wantRepoSet: &zoektquery.RepoSet{Set: map[string]bool{"github.com/test/1": true}},
		},
		{
			name: "returns filtered repoSet when negated repoHasFileFlag is in query",
			args: args{
				queryPatternInfo: &search.PatternInfo{FilePatternsReposMustExclude: []string{"1"}, PathPatternsAreRegExps: true},
				searcher: &fakeSearcher{result: &zoekt.SearchResult{
					Files: []zoekt.FileMatch{{
						FileName:   "1.md",
						Repository: "github.com/test/1",
						LineMatches: []zoekt.LineMatch{{
							FileName: true,
						}}},
					},
					RepoURLs: map[string]string{"github.com/test/1": "github.com/test/1"}}},
				repoSet:                         zoektquery.RepoSet{Set: map[string]bool{"github.com/test/1": true, "github.com/test/2": true}},
				repoHasFileFlagIsInQuery:        false,
				negatedRepoHasFileFlagIsInQuery: true,
			},
			wantRepoSet: &zoektquery.RepoSet{Set: map[string]bool{"github.com/test/2": true}},
		}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRepoSet, err := createNewRepoSetWithRepoHasFileInputs(tt.args.ctx, tt.args.queryPatternInfo, tt.args.searcher, tt.args.repoSet)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(gotRepoSet, tt.wantRepoSet) {
				t.Errorf("createNewRepoSetWithRepoHasFileInputs() gotRepoSet = %v, want %v", gotRepoSet, tt.wantRepoSet)
			}
		})
	}
}

func init() {
	// Set both URLs to something that will fail in tests. We shouldn't be
	// contacting them in tests.
	zoektAddr = "127.0.0.1:101010"
	searcherURL = "http://127.0.0.1:101010"
}

func Test_splitMatchesOnNewlines(t *testing.T) {
	type args struct {
		fileMatches []zoekt.FileMatch
	}
	tests := []struct {
		name string
		args args
		want []zoekt.FileMatch
	}{
		{name: "empty", args: args{fileMatches: nil}, want: nil},
		{
			name: "no newlines",
			args: args{
				fileMatches: []zoekt.FileMatch{
					{
						LineMatches: []zoekt.LineMatch{
							{
								Line:       []byte("abc"),
								LineStart:  0,
								LineEnd:    len("abc"),
								LineNumber: len("abc"),
								LineFragments: []zoekt.LineFragmentMatch{
									{
										LineOffset:  0,
										Offset:      0,
										MatchLength: len("abc"),
									},
								},
							},
						},
					},
				},
			},
			// Same as the input
			want: []zoekt.FileMatch{
				{
					LineMatches: []zoekt.LineMatch{
						{
							Line:       []byte("abc"),
							LineStart:  0,
							LineEnd:    len("abc"),
							LineNumber: len("abc"),
							LineFragments: []zoekt.LineFragmentMatch{
								{
									LineOffset:  0,
									Offset:      0,
									MatchLength: len("abc"),
								},
							},
						},
					},
				},
			},
		},
		{
			name: "one newline",
			args: args{
				fileMatches: []zoekt.FileMatch{
					{
						LineMatches: []zoekt.LineMatch{
							{
								Line:       []byte("a\nb"),
								LineStart:  0,
								LineEnd:    len("a\nb"),
								LineNumber: len("a\nb"),
								LineFragments: []zoekt.LineFragmentMatch{
									{
										LineOffset:  0,
										Offset:      0,
										MatchLength: len("a\nb"),
									},
								},
							},
						},
					},
				},
			},
			want: []zoekt.FileMatch{
				{
					LineMatches: []zoekt.LineMatch{
						{
							Line:       []byte("a"),
							LineStart:  0,
							LineEnd:    len("a"),
							LineNumber: len("a"),
							LineFragments: []zoekt.LineFragmentMatch{
								{
									LineOffset:  0,
									Offset:      0,
									MatchLength: len("a"),
								},
							},
						},
						{
							Line:       []byte("b"),
							LineStart:  len("a\n"),
							LineEnd:    len("a\nb"),
							LineNumber: len("b"),
							LineFragments: []zoekt.LineFragmentMatch{
								{
									LineOffset:  0,
									Offset:      2,
									MatchLength: len("b"),
								},
							},
						},
					},
				},
			},
		},
		{
			name: "newline at end",
			args: args{
				fileMatches: []zoekt.FileMatch{
					{
						LineMatches: []zoekt.LineMatch{
							{
								Line:       []byte("a\n"),
								LineStart:  0,
								LineEnd:    len("a\n"),
								LineNumber: len("a\n"),
								LineFragments: []zoekt.LineFragmentMatch{
									{
										LineOffset:  0,
										Offset:      0,
										MatchLength: len("a\n"),
									},
								},
							},
						},
					},
				},
			},
			want: []zoekt.FileMatch{
				{
					LineMatches: []zoekt.LineMatch{
						{
							Line:       []byte("a"),
							LineStart:  0,
							LineEnd:    len("a"),
							LineNumber: len("a"),
							LineFragments: []zoekt.LineFragmentMatch{
								{
									LineOffset:  0,
									Offset:      0,
									MatchLength: len("a"),
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := splitMatchesOnNewlines(tt.args.fileMatches); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("splitMatchesOnNewlines() = \n%+v\nwant\n%+v", got, tt.want)
			}
		})
	}
}

func Test_fixupLineMatch(t *testing.T) {
	t.Run("line number stays the same", func(t *testing.T) {
		s := "b"
		lm := zoekt.LineMatch{
			Line:          []byte(s),
			LineStart:     1,
			LineEnd:       1+len(s),
			LineNumber:    1,
			LineFragments: []zoekt.LineFragmentMatch{ { MatchLength: 1 } },
		}
		lms := fixupLineMatch(lm)
		if len(lms) != 1 {
			t.Fatalf("got %d line matches, want exactly 1", len(lms))
		}
		lm2 := lms[0]
		if lm2.LineNumber != lm.LineNumber {
			t.Errorf("output LineNumber = %d, want %d", lm2.LineNumber, lm.LineNumber)
		}
	})
	t.Run("property tests", func(t *testing.T) {
		pp := gopter.DefaultTestParameters()
		props := gopter.NewProperties(pp)
		props.Property("number of results is number of newlines plus one", prop.ForAll(
			func(s string, start int) bool {
				lm := zoekt.LineMatch{
					Line:          []byte(s),
					LineStart:     start,
					LineEnd:       start + len(s),
					LineFragments: nil,
				}
				n := strings.Count(s, "\n")
				lms := fixupLineMatch(lm)
				return len(lms) == n+1
			},
			gen.RegexMatch(`^[abc123\n]*$`),
			gen.IntRange(0, 100),
		))
		props.Property("line number stays the same for line matches without newlines", prop.ForAll(
			func(s string, start, lnum int) bool {
				lm := zoekt.LineMatch{
					Line:          []byte(s),
					LineStart:     start,
					LineEnd:       start + len(s),
					LineNumber:    lnum,
					LineFragments: []zoekt.LineFragmentMatch{ { MatchLength: 1 } },
				}
				lms := fixupLineMatch(lm)
				return len(lms) == 1 && lms[0].LineNumber == lm.LineNumber
			},
			gen.RegexMatch(`^[abc123]+$`),
			gen.IntRange(0, 100),
			gen.IntRange(1, 100),
		))
		props.TestingRun(t)
	})
}
