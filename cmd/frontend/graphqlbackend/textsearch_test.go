package graphqlbackend

import (
	"context"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/cmd/searcher/protocol"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/endpoint"
	"github.com/sourcegraph/sourcegraph/internal/errcode"
	"github.com/sourcegraph/sourcegraph/internal/gitserver"
	"github.com/sourcegraph/sourcegraph/internal/search"
	searchbackend "github.com/sourcegraph/sourcegraph/internal/search/backend"
	"github.com/sourcegraph/sourcegraph/internal/search/query"
	"github.com/sourcegraph/sourcegraph/internal/search/searcher"
	"github.com/sourcegraph/sourcegraph/internal/vcs"
	"github.com/sourcegraph/sourcegraph/internal/vcs/git"
	"github.com/sourcegraph/sourcegraph/schema"
)

func TestSearchFilesInRepos(t *testing.T) {
	mockSearchFilesInRepo = func(ctx context.Context, repo *types.Repo, gitserverRepo gitserver.Repo, rev string, info *search.TextPatternInfo, fetchTimeout time.Duration) (matches []*FileMatchResolver, limitHit bool, err error) {
		repoName := repo.Name
		switch repoName {
		case "foo/one":
			return []*FileMatchResolver{
				{
					uri: "git://" + string(repoName) + "?" + rev + "#" + "main.go",
				},
			}, false, nil
		case "foo/two":
			return []*FileMatchResolver{
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
			return nil, false, &gitserver.RevisionNotFoundError{Repo: repoName, Spec: "missing"}
		default:
			return nil, false, errors.New("Unexpected repo")
		}
	}
	defer func() { mockSearchFilesInRepo = nil }()

	zoekt := &searchbackend.Zoekt{Client: &fakeSearcher{}}

	q, err := query.ParseAndCheck("foo")
	if err != nil {
		t.Fatal(err)
	}
	args := &search.TextParameters{
		PatternInfo: &search.TextPatternInfo{
			FileMatchLimit: defaultMaxSearchResults,
			Pattern:        "foo",
		},
		RepoPromise:  (&search.Promise{}).Resolve(makeRepositoryRevisions("foo/one", "foo/two", "foo/empty", "foo/cloning", "foo/missing", "foo/missing-db", "foo/timedout", "foo/no-rev")),
		Query:        q,
		Zoekt:        zoekt,
		SearcherURLs: endpoint.Static("test"),
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
	args = &search.TextParameters{
		PatternInfo: &search.TextPatternInfo{
			FileMatchLimit: defaultMaxSearchResults,
			Pattern:        "foo",
		},
		RepoPromise:  (&search.Promise{}).Resolve(makeRepositoryRevisions("foo/no-rev@dev")),
		Query:        q,
		Zoekt:        zoekt,
		SearcherURLs: endpoint.Static("test"),
	}

	_, _, err = searchFilesInRepos(context.Background(), args)
	if !gitserver.IsRevisionNotFound(errors.Cause(err)) {
		t.Fatalf("searching non-existent rev expected to fail with RevisionNotFoundError got: %v", err)
	}
}

func TestSearchFilesInRepos_multipleRevsPerRepo(t *testing.T) {
	mockSearchFilesInRepo = func(ctx context.Context, repo *types.Repo, gitserverRepo gitserver.Repo, rev string, info *search.TextPatternInfo, fetchTimeout time.Duration) (matches []*FileMatchResolver, limitHit bool, err error) {
		repoName := repo.Name
		switch repoName {
		case "foo":
			return []*FileMatchResolver{
				{
					uri: "git://" + string(repoName) + "?" + rev + "#" + "main.go",
				},
			}, false, nil
		default:
			panic("unexpected repo")
		}
	}
	defer func() { mockSearchFilesInRepo = nil }()

	trueVal := true
	conf.Mock(&conf.Unified{SiteConfiguration: schema.SiteConfiguration{
		ExperimentalFeatures: &schema.ExperimentalFeatures{SearchMultipleRevisionsPerRepository: &trueVal},
	}})
	defer conf.Mock(nil)

	zoekt := &searchbackend.Zoekt{Client: &fakeSearcher{}}

	q, err := query.ParseAndCheck("foo")
	if err != nil {
		t.Fatal(err)
	}
	args := &search.TextParameters{
		PatternInfo: &search.TextPatternInfo{
			FileMatchLimit: defaultMaxSearchResults,
			Pattern:        "foo",
		},
		RepoPromise:  (&search.Promise{}).Resolve(makeRepositoryRevisions("foo@master:mybranch:*refs/heads/")),
		Query:        q,
		Zoekt:        zoekt,
		SearcherURLs: endpoint.Static("test"),
	}
	repos, _ := getRepos(context.Background(), args.RepoPromise)
	repos[0].ListRefs = func(context.Context, gitserver.Repo) ([]git.Ref, error) {
		return []git.Ref{{Name: "refs/heads/branch3"}, {Name: "refs/heads/branch4"}}, nil
	}
	results, _, err := searchFilesInRepos(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}

	resultURIs := make([]string, len(results))
	for i, result := range results {
		resultURIs[i] = result.uri
	}
	sort.Strings(resultURIs)

	wantResultURIs := []string{
		"git://foo?branch3#main.go",
		"git://foo?branch4#main.go",
		"git://foo?master#main.go",
		"git://foo?mybranch#main.go",
	}
	if !reflect.DeepEqual(resultURIs, wantResultURIs) {
		t.Errorf("got %v, want %v", resultURIs, wantResultURIs)
	}
}

func TestRepoShouldBeSearched(t *testing.T) {
	searcher.MockSearch = func(ctx context.Context, repo gitserver.Repo, commit api.CommitID, p *search.TextPatternInfo, fetchTimeout time.Duration) (matches []*protocol.FileMatch, limitHit bool, err error) {
		repoName := repo.Name
		switch repoName {
		case "foo/one":
			return []*protocol.FileMatch{{Path: "main.go"}}, false, nil
		case "foo/no-filematch":
			return []*protocol.FileMatch{}, false, nil
		default:
			return nil, false, errors.New("Unexpected repo")
		}
	}
	defer func() { searcher.MockSearch = nil }()
	info := &search.TextPatternInfo{
		FileMatchLimit:               defaultMaxSearchResults,
		Pattern:                      "foo",
		FilePatternsReposMustInclude: []string{"main"},
	}

	shouldBeSearched, err := repoShouldBeSearched(context.Background(), nil, info, gitserver.Repo{Name: "foo/one", URL: "http://example.com/foo/one"}, "1a2b3c", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if !shouldBeSearched {
		t.Errorf("expected repo to be searched, got shouldn't be searched")
	}

	shouldBeSearched, err = repoShouldBeSearched(context.Background(), nil, info, gitserver.Repo{Name: "foo/no-filematch", URL: "http://example.com/foo/no-filematch"}, "1a2b3c", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if shouldBeSearched {
		t.Errorf("expected repo to not be searched, got should be searched")
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
		r[i] = &search.RepositoryRevisions{Repo: &types.Repo{Name: api.RepoName(repoName)}, Revs: revs}
	}
	return r
}

func TestLimitSearcherRepos(t *testing.T) {
	repos := func(names ...string) []*types.Repo {
		var repos []*types.Repo
		for _, name := range names {
			repos = append(repos, &types.Repo{Name: api.RepoName(name)})
		}
		return repos
	}

	repoRevs := func(repoRevs ...string) []*search.RepositoryRevisions {
		var result []*search.RepositoryRevisions
		for _, repoRev := range repoRevs {
			split := strings.Split(repoRev, "@")
			repo, rev := split[0], split[1]

			found := false
			for _, existing := range result {
				if string(existing.Repo.Name) == repo {
					existing.Revs = append(existing.Revs, search.RevisionSpecifier{RevSpec: rev})
					found = true
					break
				}
			}
			if found {
				continue
			}
			result = append(result, &search.RepositoryRevisions{
				Repo: &types.Repo{Name: api.RepoName(repo)},
				Revs: []search.RevisionSpecifier{{RevSpec: rev}},
			})
		}
		return result
	}

	tests := []struct {
		name        string
		limit       int
		input       []*search.RepositoryRevisions
		want        []*search.RepositoryRevisions
		wantLimited []*types.Repo
	}{
		{
			name:        "non_limited",
			limit:       5,
			input:       repoRevs("a@1", "a@2", "b@1", "c@1"),
			want:        repoRevs("a@1", "a@2", "b@1", "c@1"),
			wantLimited: nil,
		},
		{
			name:        "limited",
			limit:       5,
			input:       repoRevs("a@1", "b@1", "c@1", "d@1", "e@1", "f@1", "g@1"),
			want:        repoRevs("a@1", "b@1", "c@1", "d@1", "e@1"),
			wantLimited: repos("f", "g"),
		},
		{
			name:        "rev_limited",
			limit:       6,
			input:       repoRevs("a@1", "a@2", "b@1", "c@1", "d@1", "e@1", "f@1", "g@1"),
			want:        repoRevs("a@1", "a@2", "b@1", "c@1", "d@1", "e@1"),
			wantLimited: repos("f", "g"),
		},
		{
			name:        "rev_limited_duplication",
			limit:       6,
			input:       repoRevs("a@1", "a@2", "b@1", "c@1", "d@1", "e@1", "f@1", "f@2", "g@1"),
			want:        repoRevs("a@1", "a@2", "b@1", "c@1", "d@1", "e@1"),
			wantLimited: repos("f", "g"),
		},
	}
	for _, tst := range tests {
		t.Run(tst.name, func(t *testing.T) {
			got, gotLimited := limitSearcherRepos(tst.input, tst.limit)
			if !reflect.DeepEqual(got, tst.want) {
				t.Errorf("got %+v want %+v", got, tst.limit)
			}
			if !reflect.DeepEqual(gotLimited, tst.wantLimited) {
				t.Errorf("got limited %+v want %+v", gotLimited, tst.wantLimited)
			}
		})
	}
}
