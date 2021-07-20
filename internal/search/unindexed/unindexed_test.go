package unindexed

import (
	"context"
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"
	"testing/quick"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/google/go-cmp/cmp"

	"github.com/sourcegraph/sourcegraph/cmd/searcher/protocol"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/endpoint"
	"github.com/sourcegraph/sourcegraph/internal/errcode"
	"github.com/sourcegraph/sourcegraph/internal/gitserver"
	"github.com/sourcegraph/sourcegraph/internal/search"
	searchbackend "github.com/sourcegraph/sourcegraph/internal/search/backend"
	"github.com/sourcegraph/sourcegraph/internal/search/query"
	"github.com/sourcegraph/sourcegraph/internal/search/result"
	"github.com/sourcegraph/sourcegraph/internal/search/searcher"
	"github.com/sourcegraph/sourcegraph/internal/types"
	"github.com/sourcegraph/sourcegraph/internal/vcs"
	"github.com/sourcegraph/sourcegraph/internal/vcs/git"
	"github.com/sourcegraph/sourcegraph/schema"
)

func TestSearchFilesInRepos(t *testing.T) {
	mockSearchFilesInRepo = func(ctx context.Context, repo types.RepoName, gitserverRepo api.RepoName, rev string, info *search.TextPatternInfo, fetchTimeout time.Duration) (matches []result.Match, limitHit bool, err error) {
		repoName := repo.Name
		switch repoName {
		case "foo/one":
			return []result.Match{&result.FileMatch{
				File: result.File{
					Repo:     repo,
					InputRev: &rev,
					Path:     "main.go",
				},
			}}, false, nil
		case "foo/two":
			return []result.Match{&result.FileMatch{
				File: result.File{
					Repo:     repo,
					InputRev: &rev,
					Path:     "main.go",
				},
			}}, false, nil
		case "foo/empty":
			return nil, false, nil
		case "foo/cloning":
			return nil, false, &vcs.RepoNotExistError{Repo: repoName, CloneInProgress: true}
		case "foo/missing":
			return nil, false, &vcs.RepoNotExistError{Repo: repoName}
		case "foo/missing-database":
			return nil, false, &errcode.Mock{Message: "repo not found: foo/missing-database", IsNotFound: true}
		case "foo/timedout":
			return nil, false, context.DeadlineExceeded
		case "foo/no-rev":
			// TODO we do not specify a rev when searching "foo/no-rev", so it
			// is treated as an empty repository. We need to test the fatal
			// case of trying to search a revision which doesn't exist.
			return nil, false, &gitserver.RevisionNotFoundError{Repo: repoName, Spec: "missing"}
		default:
			return nil, false, errors.New("Unexpected repo")
		}
	}
	defer func() { mockSearchFilesInRepo = nil }()

	zoekt := &searchbackend.Zoekt{Client: &searchbackend.FakeSearcher{}}

	q, err := query.ParseLiteral("foo")
	if err != nil {
		t.Fatal(err)
	}
	repoRevs := makeRepositoryRevisions("foo/one", "foo/two", "foo/empty", "foo/cloning", "foo/missing", "foo/missing-database", "foo/timedout", "foo/no-rev")
	args := &search.TextParameters{
		PatternInfo: &search.TextPatternInfo{
			FileMatchLimit: search.DefaultMaxSearchResults,
			Pattern:        "foo",
		},
		RepoPromise:  (&search.RepoPromise{}).Resolve(repoRevs),
		Query:        q,
		Zoekt:        zoekt,
		SearcherURLs: endpoint.Static("test"),
	}
	matches, common, err := SearchFilesInReposBatch(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 2 {
		t.Errorf("expected two results, got %d", len(matches))
	}
	repoNames := map[api.RepoID]string{}
	for _, rr := range repoRevs {
		repoNames[rr.Repo.ID] = string(rr.Repo.Name)
	}
	assertReposStatus(t, repoNames, common.Status, map[string]search.RepoStatus{
		"foo/cloning":          search.RepoStatusCloning,
		"foo/missing":          search.RepoStatusMissing,
		"foo/missing-database": search.RepoStatusMissing,
		"foo/timedout":         search.RepoStatusTimedout,
	})

	// If we specify a rev and it isn't found, we fail the whole search since
	// that should be checked earlier.
	args = &search.TextParameters{
		PatternInfo: &search.TextPatternInfo{
			FileMatchLimit: search.DefaultMaxSearchResults,
			Pattern:        "foo",
		},
		RepoPromise:  (&search.RepoPromise{}).Resolve(makeRepositoryRevisions("foo/no-rev@dev")),
		Query:        q,
		Zoekt:        zoekt,
		SearcherURLs: endpoint.Static("test"),
	}

	_, _, err = SearchFilesInReposBatch(context.Background(), args)
	if !errors.HasType(err, &gitserver.RevisionNotFoundError{}) {
		t.Fatalf("searching non-existent rev expected to fail with RevisionNotFoundError got: %v", err)
	}
}

func TestSearchFilesInReposStream(t *testing.T) {
	mockSearchFilesInRepo = func(ctx context.Context, repo types.RepoName, gitserverRepo api.RepoName, rev string, info *search.TextPatternInfo, fetchTimeout time.Duration) (matches []result.Match, limitHit bool, err error) {
		repoName := repo.Name
		switch repoName {
		case "foo/one":
			return []result.Match{&result.FileMatch{
				File: result.File{
					Repo:     repo,
					InputRev: &rev,
					Path:     "main.go",
				},
			}}, false, nil
		case "foo/two":
			return []result.Match{&result.FileMatch{
				File: result.File{
					Repo:     repo,
					InputRev: &rev,
					Path:     "main.go",
				},
			}}, false, nil
		case "foo/three":
			return []result.Match{&result.FileMatch{
				File: result.File{
					Repo:     repo,
					InputRev: &rev,
					Path:     "main.go",
				},
			}}, false, nil
		default:
			return nil, false, errors.New("Unexpected repo")
		}
	}
	defer func() { mockSearchFilesInRepo = nil }()

	zoekt := &searchbackend.Zoekt{Client: &searchbackend.FakeSearcher{}}

	q, err := query.ParseLiteral("foo")
	if err != nil {
		t.Fatal(err)
	}
	args := &search.TextParameters{
		PatternInfo: &search.TextPatternInfo{
			FileMatchLimit: search.DefaultMaxSearchResults,
			Pattern:        "foo",
		},
		RepoPromise:  (&search.RepoPromise{}).Resolve(makeRepositoryRevisions("foo/one", "foo/two", "foo/three")),
		Query:        q,
		Zoekt:        zoekt,
		SearcherURLs: endpoint.Static("test"),
	}

	matches, _, err := SearchFilesInReposBatch(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}

	if len(matches) != 3 {
		t.Errorf("expected three results, got %d", len(matches))
	}
}

func assertReposStatus(t *testing.T, repoNames map[api.RepoID]string, got search.RepoStatusMap, want map[string]search.RepoStatus) {
	t.Helper()
	gotM := map[string]search.RepoStatus{}
	got.Iterate(func(id api.RepoID, mask search.RepoStatus) {
		name := repoNames[id]
		if name == "" {
			name = fmt.Sprintf("UNKNOWNREPO{ID=%d}", id)
		}
		gotM[name] = mask
	})
	if diff := cmp.Diff(want, gotM); diff != "" {
		t.Errorf("RepoStatusMap mismatch (-want +got):\n%s", diff)
	}
}

func TestSearchFilesInRepos_multipleRevsPerRepo(t *testing.T) {
	mockSearchFilesInRepo = func(ctx context.Context, repo types.RepoName, gitserverRepo api.RepoName, rev string, info *search.TextPatternInfo, fetchTimeout time.Duration) (matches []result.Match, limitHit bool, err error) {
		repoName := repo.Name
		switch repoName {
		case "foo":
			return []result.Match{&result.FileMatch{
				File: result.File{
					Repo:     repo,
					CommitID: api.CommitID(rev),
					Path:     "main.go",
				},
			}}, false, nil
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

	zoekt := &searchbackend.Zoekt{Client: &searchbackend.FakeSearcher{}}

	q, err := query.ParseLiteral("foo")
	if err != nil {
		t.Fatal(err)
	}
	args := &search.TextParameters{
		PatternInfo: &search.TextPatternInfo{
			FileMatchLimit: search.DefaultMaxSearchResults,
			Pattern:        "foo",
		},
		RepoPromise:  (&search.RepoPromise{}).Resolve(makeRepositoryRevisions("foo@master:mybranch:*refs/heads/")),
		Query:        q,
		Zoekt:        zoekt,
		SearcherURLs: endpoint.Static("test"),
	}
	repos, _ := args.RepoPromise.Get(context.Background())
	repos[0].ListRefs = func(context.Context, api.RepoName) ([]git.Ref, error) {
		return []git.Ref{{Name: "refs/heads/branch3"}, {Name: "refs/heads/branch4"}}, nil
	}
	matches, _, err := SearchFilesInReposBatch(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}

	matchKeys := make([]result.Key, len(matches))
	for i, match := range matches {
		matchKeys[i] = match.Key()
	}
	sort.Slice(matchKeys, func(i, j int) bool { return matchKeys[i].Less(matchKeys[j]) })

	wantResultKeys := []result.Key{
		{Repo: "foo", Commit: "branch3", Path: "main.go"},
		{Repo: "foo", Commit: "branch4", Path: "main.go"},
		{Repo: "foo", Commit: "master", Path: "main.go"},
		{Repo: "foo", Commit: "mybranch", Path: "main.go"},
	}
	if !reflect.DeepEqual(matchKeys, wantResultKeys) {
		t.Errorf("got %v, want %v", matchKeys, wantResultKeys)
	}
}

func TestRepoShouldBeSearched(t *testing.T) {
	searcher.MockSearch = func(ctx context.Context, repo api.RepoName, commit api.CommitID, p *search.TextPatternInfo, fetchTimeout time.Duration) (matches []*protocol.FileMatch, limitHit bool, err error) {
		repoName := repo
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
		FileMatchLimit:               search.DefaultMaxSearchResults,
		Pattern:                      "foo",
		FilePatternsReposMustInclude: []string{"main"},
	}

	shouldBeSearched, err := repoShouldBeSearched(context.Background(), nil, info, "foo/one", "1a2b3c", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if !shouldBeSearched {
		t.Errorf("expected repo to be searched, got shouldn't be searched")
	}

	shouldBeSearched, err = repoShouldBeSearched(context.Background(), nil, info, "foo/no-filematch", "1a2b3c", time.Minute)
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
		r[i] = &search.RepositoryRevisions{Repo: mkRepos(repoName)[0], Revs: revs}
	}
	return r
}

func mkRepos(names ...string) []types.RepoName {
	var repos []types.RepoName
	for _, name := range names {
		sum := md5.Sum([]byte(name))
		id := api.RepoID(binary.BigEndian.Uint64(sum[:]))
		if id < 0 {
			id = -(id / 2)
		}
		if id == 0 {
			id++
		}
		repos = append(repos, types.RepoName{ID: id, Name: api.RepoName(name)})
	}
	return repos
}

func TestFileMatch_Limit(t *testing.T) {
	desc := func(fm *result.FileMatch) string {
		parts := []string{fmt.Sprintf("symbols=%d", len(fm.Symbols))}
		for _, lm := range fm.LineMatches {
			parts = append(parts, fmt.Sprintf("lm=%d", len(lm.OffsetAndLengths)))
		}
		return strings.Join(parts, " ")
	}

	f := func(lineMatches []result.LineMatch, symbols []int, limitInput uint32) bool {
		fm := &result.FileMatch{
			// SearchSymbolResult fails to generate due to private fields. So
			// we just generate a slice of ints and use its length. This is
			// fine for limit which only looks at the slice and not in it.
			Symbols: make([]*result.SymbolMatch, len(symbols)),
		}
		// We don't use *LineMatch as args since quick can generate nil.
		for _, lm := range lineMatches {
			lm := lm
			fm.LineMatches = append(fm.LineMatches, &lm)
		}
		beforeDesc := desc(fm)

		// It isn't interesting to test limit > ResultCount, so we bound it to
		// [1, ResultCount]
		count := fm.ResultCount()
		limit := (int(limitInput) % count) + 1

		after := fm.Limit(limit)
		newCount := fm.ResultCount()

		if after == 0 && newCount == limit {
			return true
		}

		afterDesc := desc(fm)
		t.Logf("failed limit=%d count=%d => after=%d newCount=%d:\nbeforeDesc: %s\nafterDesc:  %s", limit, count, after, newCount, beforeDesc, afterDesc)
		return false
	}
	t.Run("quick", func(t *testing.T) {
		if err := quick.Check(f, nil); err != nil {
			t.Error("quick check failed")
		}
	})

	cases := []struct {
		Name        string
		LineMatches []result.LineMatch
		Symbols     int
		Limit       int
	}{{
		Name: "1 line match",
		LineMatches: []result.LineMatch{{
			OffsetAndLengths: [][2]int32{{1, 1}},
		}},
		Limit: 1,
	}, {
		Name:  "file path match",
		Limit: 1,
	}, {
		Name:  "file path match 2",
		Limit: 2,
	}}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			if !f(c.LineMatches, make([]int, c.Symbols), uint32(c.Limit)) {
				t.Error("failed")
			}
		})
	}
}
