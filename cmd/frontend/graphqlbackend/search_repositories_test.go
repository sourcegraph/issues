package graphqlbackend

import (
	"context"
	"errors"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/search"
	searchbackend "github.com/sourcegraph/sourcegraph/internal/search/backend"
	"github.com/sourcegraph/sourcegraph/internal/search/query"
	"github.com/sourcegraph/sourcegraph/internal/search/result"
	"github.com/sourcegraph/sourcegraph/internal/search/streaming"
	"github.com/sourcegraph/sourcegraph/internal/types"
)

func TestSearchRepositories(t *testing.T) {
	repositories := []*search.RepositoryRevisions{
		{Repo: types.RepoName{ID: 123, Name: "foo/one"}, Revs: []search.RevisionSpecifier{{RevSpec: ""}}},
		{Repo: types.RepoName{ID: 456, Name: "foo/no-match"}, Revs: []search.RevisionSpecifier{{RevSpec: ""}}},
		{Repo: types.RepoName{ID: 789, Name: "bar/one"}, Revs: []search.RevisionSpecifier{{RevSpec: ""}}},
	}

	zoekt := &searchbackend.Zoekt{Client: &searchbackend.FakeSearcher{}}

	mockSearchFilesInRepos = func(args *search.TextParameters) (matches []*result.FileMatch, common *streaming.Stats, err error) {
		repos, err := getRepos(context.Background(), args.RepoPromise)
		if err != nil {
			return nil, nil, err
		}
		repoName := repos[0].Repo.Name
		rev := "1a2b3c"
		switch repoName {
		case "foo/one":
			return []*result.FileMatch{{
				File: result.File{
					Repo:     types.RepoName{ID: 123, Name: repoName},
					InputRev: &rev,
					Path:     "f.go",
				},
			}}, &streaming.Stats{}, nil
		case "bar/one":
			return []*result.FileMatch{{
				File: result.File{
					Repo:     types.RepoName{ID: 789, Name: repoName},
					InputRev: &rev,
					Path:     "f.go",
				},
			}}, &streaming.Stats{}, nil
		case "foo/no-match":
			return []*result.FileMatch{}, &streaming.Stats{}, nil
		default:
			return nil, &streaming.Stats{}, errors.New("Unexpected repo")
		}
	}

	cases := []struct {
		name string
		q    string
		want []string
	}{{
		name: "all",
		q:    "type:repo",
		want: []string{"bar/one", "foo/no-match", "foo/one"},
	}, {
		name: "pattern filter",
		q:    "type:repo foo/one",
		want: []string{"foo/one"},
	}, {
		name: "repohasfile",
		q:    "foo type:repo repohasfile:f.go",
		want: []string{"foo/one"},
	}, {
		name: "case yes match",
		q:    "foo case:yes",
		want: []string{"foo/no-match", "foo/one"},
	}, {
		name: "case no match",
		q:    "Foo case:no",
		want: []string{"foo/no-match", "foo/one"},
	}, {
		name: "case exclude all",
		q:    "Foo case:yes",
		want: []string{},
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			q, _ := query.ParseLiteral(tc.q)
			b, _ := query.ToBasicQuery(q)
			pattern := search.ToTextPatternInfo(b, search.Batch, query.Identity)
			matches, _, err := searchRepositoriesBatch(context.Background(), &search.TextParameters{
				PatternInfo: pattern,
				RepoPromise: (&search.Promise{}).Resolve(repositories),
				Query:       q,
				Zoekt:       zoekt,
			}, int32(100))
			if err != nil {
				t.Fatal(err)
			}

			var got []string
			for _, res := range matches {
				r := res.(*result.RepoMatch)
				got = append(got, string(r.Name))
			}
			sort.Strings(got)

			if !cmp.Equal(tc.want, got, cmpopts.EquateEmpty()) {
				t.Errorf("mismatch (-want +got):\n%s", cmp.Diff(tc.want, got))
			}
		})
	}
}

func searchRepositoriesBatch(ctx context.Context, args *search.TextParameters, limit int32) ([]result.Match, streaming.Stats, error) {
	return streaming.CollectStream(func(stream streaming.Sender) error {
		return searchRepositories(ctx, args, limit, stream)
	})
}

func TestRepoShouldBeAdded(t *testing.T) {
	mockSearchFilesInRepos = func(args *search.TextParameters) (matches []*result.FileMatch, common *streaming.Stats, err error) {
		repos, err := getRepos(context.Background(), args.RepoPromise)
		if err != nil {
			return nil, nil, err
		}
		repoName := repos[0].Repo.Name
		rev := "1a2b3c"
		switch repoName {
		case "foo/one":
			return []*result.FileMatch{{
				File: result.File{
					Repo:     types.RepoName{ID: 123, Name: repoName},
					InputRev: &rev,
					Path:     "foo.go",
				},
			}}, &streaming.Stats{}, nil
		case "foo/no-match":
			return nil, &streaming.Stats{}, nil
		default:
			return nil, &streaming.Stats{}, errors.New("Unexpected repo")
		}
	}

	zoekt := &searchbackend.Zoekt{Client: &searchbackend.FakeSearcher{}}

	t.Run("repo should be included in results, query has repoHasFile filter", func(t *testing.T) {
		repo := &search.RepositoryRevisions{Repo: types.RepoName{ID: 123, Name: "foo/one"}, Revs: []search.RevisionSpecifier{{RevSpec: ""}}}
		mockSearchFilesInRepos = func(args *search.TextParameters) (matches []*result.FileMatch, common *streaming.Stats, err error) {
			rev := "1a2b3c"
			return []*result.FileMatch{{
				File: result.File{
					Repo:     types.RepoName{ID: 123, Name: repo.Repo.Name},
					InputRev: &rev,
					Path:     "foo.go",
				},
			}}, &streaming.Stats{}, nil
		}
		pat := &search.TextPatternInfo{Pattern: "", FilePatternsReposMustInclude: []string{"foo"}, IsRegExp: true, FileMatchLimit: 1, PathPatternsAreCaseSensitive: false, PatternMatchesContent: true, PatternMatchesPath: true}
		shouldBeAdded, err := repoShouldBeAdded(context.Background(), zoekt, repo, pat)
		if err != nil {
			t.Fatal(err)
		}
		if !shouldBeAdded {
			t.Errorf("Expected shouldBeAdded for repo %v to be true, but got false", repo)
		}
	})

	t.Run("repo shouldn't be included in results, query has repoHasFile filter ", func(t *testing.T) {
		repo := &search.RepositoryRevisions{Repo: types.RepoName{Name: "foo/no-match"}, Revs: []search.RevisionSpecifier{{RevSpec: ""}}}
		mockSearchFilesInRepos = func(args *search.TextParameters) (matches []*result.FileMatch, common *streaming.Stats, err error) {
			return nil, &streaming.Stats{}, nil
		}
		pat := &search.TextPatternInfo{Pattern: "", FilePatternsReposMustInclude: []string{"foo"}, IsRegExp: true, FileMatchLimit: 1, PathPatternsAreCaseSensitive: false, PatternMatchesContent: true, PatternMatchesPath: true}
		shouldBeAdded, err := repoShouldBeAdded(context.Background(), zoekt, repo, pat)
		if err != nil {
			t.Fatal(err)
		}
		if shouldBeAdded {
			t.Errorf("Expected shouldBeAdded for repo %v to be false, but got true", repo)
		}
	})

	t.Run("repo shouldn't be included in results, query has -repoHasFile filter", func(t *testing.T) {
		repo := &search.RepositoryRevisions{Repo: types.RepoName{ID: 123, Name: "foo/one"}, Revs: []search.RevisionSpecifier{{RevSpec: ""}}}
		mockSearchFilesInRepos = func(args *search.TextParameters) (matches []*result.FileMatch, common *streaming.Stats, err error) {
			rev := "1a2b3c"
			return []*result.FileMatch{{
				File: result.File{
					Repo:     types.RepoName{ID: 123, Name: repo.Repo.Name},
					InputRev: &rev,
					Path:     "foo.go",
				},
			}}, &streaming.Stats{}, nil
		}
		pat := &search.TextPatternInfo{Pattern: "", FilePatternsReposMustExclude: []string{"foo"}, IsRegExp: true, FileMatchLimit: 1, PathPatternsAreCaseSensitive: false, PatternMatchesContent: true, PatternMatchesPath: true}
		shouldBeAdded, err := repoShouldBeAdded(context.Background(), zoekt, repo, pat)
		if err != nil {
			t.Fatal(err)
		}
		if shouldBeAdded {
			t.Errorf("Expected shouldBeAdded for repo %v to be false, but got true", repo)
		}
	})

	t.Run("repo should be included in results, query has -repoHasFile filter", func(t *testing.T) {
		repo := &search.RepositoryRevisions{Repo: types.RepoName{Name: "foo/no-match"}, Revs: []search.RevisionSpecifier{{RevSpec: ""}}}
		mockSearchFilesInRepos = func(args *search.TextParameters) (matches []*result.FileMatch, common *streaming.Stats, err error) {
			return nil, &streaming.Stats{}, nil
		}
		pat := &search.TextPatternInfo{Pattern: "", FilePatternsReposMustExclude: []string{"foo"}, IsRegExp: true, FileMatchLimit: 1, PathPatternsAreCaseSensitive: false, PatternMatchesContent: true, PatternMatchesPath: true}
		shouldBeAdded, err := repoShouldBeAdded(context.Background(), zoekt, repo, pat)
		if err != nil {
			t.Fatal(err)
		}
		if !shouldBeAdded {
			t.Errorf("Expected shouldBeAdded for repo %v to be true, but got false", repo)
		}
	})
}

// repoShouldBeAdded determines whether a repository should be included in the result set based on whether the repository fits in the subset
// of repostiories specified in the query's `repohasfile` and `-repohasfile` fields if they exist.
func repoShouldBeAdded(ctx context.Context, zoekt *searchbackend.Zoekt, repo *search.RepositoryRevisions, pattern *search.TextPatternInfo) (bool, error) {
	repos := []*search.RepositoryRevisions{repo}
	args := search.TextParameters{
		PatternInfo: pattern,
		Zoekt:       zoekt,
	}
	rsta, err := reposToAdd(ctx, &args, repos)
	if err != nil {
		return false, err
	}
	return len(rsta) == 1, nil
}

func TestMatchRepos(t *testing.T) {
	want := makeRepositoryRevisions("foo/bar", "abc/foo")
	in := append(want, makeRepositoryRevisions("beef/bam", "qux/bas")...)
	pattern := regexp.MustCompile("foo")

	results := make(chan []*search.RepositoryRevisions)
	go func() {
		defer close(results)
		matchRepos(pattern, in, results)
	}()
	var repos []*search.RepositoryRevisions
	for matched := range results {
		repos = append(repos, matched...)
	}

	// because of the concurrency we cannot rely on the order of "repos" to be the
	// same as "want". Hence we create map of repo names and compare those.
	toMap := func(reporevs []*search.RepositoryRevisions) map[string]struct{} {
		out := map[string]struct{}{}
		for _, r := range reporevs {
			out[string(r.Repo.Name)] = struct{}{}
		}
		return out
	}
	if !reflect.DeepEqual(toMap(repos), toMap(want)) {
		t.Fatalf("expected %v, got %v", want, repos)
	}
}

func BenchmarkSearchRepositories(b *testing.B) {
	n := 200 * 1000
	repos := make([]*search.RepositoryRevisions, n)
	for i := 0; i < n; i++ {
		repo := types.RepoName{Name: api.RepoName("github.com/org/repo" + strconv.Itoa(i))}
		repos[i] = &search.RepositoryRevisions{Repo: repo, Revs: []search.RevisionSpecifier{{}}}
	}
	q, _ := query.ParseLiteral("context.WithValue")
	bq, _ := query.ToBasicQuery(q)
	pattern := search.ToTextPatternInfo(bq, search.Batch, query.Identity)
	tp := search.TextParameters{
		PatternInfo: pattern,
		RepoPromise: (&search.Promise{}).Resolve(repos),
		Query:       q,
	}
	for i := 0; i < b.N; i++ {
		_, _, err := searchRepositoriesBatch(context.Background(), &tp, tp.PatternInfo.FileMatchLimit)
		if err != nil {
			b.Fatal(err)
		}
	}
}
