package graphqlbackend

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/go-cmp/cmp"
	"github.com/sergi/go-diff/diffmatchpatch"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/search"
)

func TestSearchPagination_unmarshalSearchCursor(t *testing.T) {
	got, err := unmarshalSearchCursor(nil)
	if got != nil || err != nil {
		t.Fatal("expected got == nil && err == nil for nil input")
	}

	want := &searchCursor{
		RepositoryOffset: 1,
		ResultOffset:     2,
	}
	enc := marshalSearchCursor(want)
	if enc == "" {
		t.Fatal("expected encoded string")
	}
	got, err = unmarshalSearchCursor(&enc)
	if err != nil {
		t.Fatal("unexpected error", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatal("expected got == want")
	}
}

func TestSearchPagination_sliceSearchResults(t *testing.T) {
	repo := func(name string) *types.Repo {
		return &types.Repo{Name: api.RepoName(name)}
	}
	result := mkFileMatch
	format := func(r slicedSearchResults) string {
		var b bytes.Buffer
		fmt.Fprintln(&b, "results:")
		for i, result := range r.results {
			fm, _ := result.ToFileMatch()
			fmt.Fprintf(&b, "	[%d] %s %s\n", i, fm.Repo.repo.Name, fm.JPath)
		}
		fmt.Fprintln(&b, "common.repos:")
		for i, r := range r.common.repos {
			fmt.Fprintf(&b, "	[%d] %s\n", i, r.Name)
		}
		fmt.Fprintf(&b, "common.resultCount: %v\n", r.common.resultCount)
		fmt.Fprintf(&b, "resultOffset: %d\n", r.resultOffset)
		fmt.Fprintf(&b, "limitHit: %v\n", r.limitHit)
		return b.String()
	}
	sharedResult := []SearchResultResolver{
		result(repo("org/repo1"), "a.go"),
		result(repo("org/repo1"), "b.go"),
		result(repo("org/repo1"), "c.go"),
		result(repo("org/repo2"), "a.go"),
		result(repo("org/repo2"), "b.go"),
		result(repo("org/repo3"), "a.go"),
		result(repo("org/repo4"), "a.go"),
		result(repo("org/repo4"), "b.go"),
		result(repo("org/repo4"), "c.go"),
		result(repo("org/repo5"), "a.go"),
		result(repo("org/repo5"), "b.go"),
		result(repo("org/repo5"), "c.go"),
		result(repo("org/repo5"), "d.go"),
		result(repo("org/repo5"), "e.go"),
	}
	sharedCommon := &searchResultsCommon{
		// Note: this is an intentionally unordered list to ensure we do not
		// rely on the order of lists in common (which is not guaranteed by
		// tests).
		repos: []*types.Repo{repo("org/repo1"), repo("org/repo3"), repo("org/repo2")},
	}
	tests := []struct {
		name          string
		results       []SearchResultResolver
		common        *searchResultsCommon
		offset, limit int
		want          slicedSearchResults
	}{
		{
			name:    "empty result set",
			results: []SearchResultResolver{},
			common:  &searchResultsCommon{},
			offset:  0,
			limit:   3,
			want: slicedSearchResults{
				results: []SearchResultResolver{},
				common: &searchResultsCommon{
					resultCount: 0,
					repos:       nil,
					partial:     make(map[api.RepoName]struct{}),
				},
				resultOffset: 0,
				limitHit:     false,
			},
		},
		{
			name:    "limit repo boundary",
			results: sharedResult,
			common:  sharedCommon,
			offset:  0,
			limit:   3,
			want: slicedSearchResults{
				results: []SearchResultResolver{
					result(repo("org/repo1"), "a.go"),
					result(repo("org/repo1"), "b.go"),
					result(repo("org/repo1"), "c.go"),
				},
				common: &searchResultsCommon{
					resultCount: 3,
					repos:       []*types.Repo{repo("org/repo1")},
					partial:     make(map[api.RepoName]struct{}),
				},
				resultOffset: 0,
				limitHit:     true,
			},
		},
		{
			name:    "limit non repo boundary",
			results: sharedResult,
			common:  sharedCommon,
			offset:  0,
			limit:   2,
			want: slicedSearchResults{
				results: []SearchResultResolver{
					result(repo("org/repo1"), "a.go"),
					result(repo("org/repo1"), "b.go"),
				},
				common: &searchResultsCommon{
					resultCount: 2,
					repos:       []*types.Repo{repo("org/repo1")},
					partial:     make(map[api.RepoName]struct{}),
				},
				resultOffset: 2,
				limitHit:     true,
			},
		},
		{
			name:    "offset repo boundary",
			results: sharedResult,
			common:  sharedCommon,
			offset:  3,
			limit:   3,
			want: slicedSearchResults{
				results: []SearchResultResolver{
					result(repo("org/repo2"), "a.go"),
					result(repo("org/repo2"), "b.go"),
					result(repo("org/repo3"), "a.go"),
				},
				common: &searchResultsCommon{
					resultCount: 3,
					repos:       []*types.Repo{repo("org/repo2"), repo("org/repo3")},
					partial:     make(map[api.RepoName]struct{}),
				},
				resultOffset: 0,
				limitHit:     true,
			},
		},
		{
			name:    "offset non-repo boundary",
			results: sharedResult,
			common:  sharedCommon,
			offset:  2,
			limit:   3,
			want: slicedSearchResults{
				results: []SearchResultResolver{
					result(repo("org/repo1"), "c.go"),
					result(repo("org/repo2"), "a.go"),
					result(repo("org/repo2"), "b.go"),
				},
				common: &searchResultsCommon{
					resultCount: 3,
					repos:       []*types.Repo{repo("org/repo1"), repo("org/repo2")},
					partial:     make(map[api.RepoName]struct{}),
				},
				resultOffset: 0,
				limitHit:     true,
			},
		},
		{
			name: "offset repo boundary fully consumed",
			results: []SearchResultResolver{
				result(repo("org/repo1"), "a.go"),
				result(repo("org/repo1"), "b.go"),
				result(repo("org/repo1"), "c.go"),
				result(repo("org/repo2"), "a.go"),
				result(repo("org/repo2"), "b.go"),
				result(repo("org/repo2"), "c.go"),
			},
			common: &searchResultsCommon{
				repos:       []*types.Repo{repo("org/repo1"), repo("org/repo2")},
				resultCount: 3,
			},
			offset: 3,
			limit:  3,
			want: slicedSearchResults{
				results: []SearchResultResolver{
					result(repo("org/repo2"), "a.go"),
					result(repo("org/repo2"), "b.go"),
					result(repo("org/repo2"), "c.go"),
				},
				common: &searchResultsCommon{
					resultCount: 3,
					repos:       []*types.Repo{repo("org/repo2")},
					partial:     make(map[api.RepoName]struct{}),
				},
				resultOffset: 0,
				limitHit:     false,
			},
		},
		{
			name:    "limit non-repo boundary small",
			results: sharedResult,
			common:  sharedCommon,
			offset:  1,
			limit:   1,
			want: slicedSearchResults{
				results: []SearchResultResolver{
					result(repo("org/repo1"), "b.go"),
				},
				common: &searchResultsCommon{
					resultCount: 1,
					repos:       []*types.Repo{repo("org/repo1")},
					partial:     make(map[api.RepoName]struct{}),
				},
				resultOffset: 2,
				limitHit:     true,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := sliceSearchResults(test.results, test.common, test.offset, test.limit)
			if !reflect.DeepEqual(got, test.want) {
				t.Logf("got != want")
				gotFormatted := format(got)
				wantFormatted := format(test.want)
				t.Logf("got:\n%s\n", gotFormatted)
				t.Logf("want:\n%s\n", wantFormatted)
				dmp := diffmatchpatch.New()
				t.Error("diff(got, want):\n", dmp.DiffPrettyText(dmp.DiffMain(wantFormatted, gotFormatted, true)))

				if wantFormatted == gotFormatted {
					dmp = diffmatchpatch.New()
					t.Error("diff(got, want):\n", dmp.DiffPrettyText(dmp.DiffMain(spew.Sdump(test.want), spew.Sdump(got), true)))
				}
			}
		})
	}
}

func TestSearchPagination_repoPaginationPlan(t *testing.T) {
	revs := func(rev ...string) (revs []search.RevisionSpecifier) {
		for _, r := range rev {
			revs = append(revs, search.RevisionSpecifier{RevSpec: r})
		}
		return revs
	}
	repo := func(name string) *types.Repo {
		return &types.Repo{Name: api.RepoName(name)}
	}
	result := func(repo *types.Repo, path, rev string) *FileMatchResolver {
		fm := mkFileMatch(repo, path)
		fm.InputRev = &rev
		return fm
	}
	repoRevs := func(name string, rev ...string) *search.RepositoryRevisions {
		return &search.RepositoryRevisions{
			Repo: repo(name),
			Revs: revs(rev...),
		}
	}
	searchRepos := []*search.RepositoryRevisions{
		repoRevs("1", "master"),
		repoRevs("2", "master"),
		repoRevs("3", "master", "feature"),
		repoRevs("4", "master"),
		repoRevs("5", "master"),
	}
	var searchedBatches [][]*search.RepositoryRevisions
	resultsExecutor := func(batch []*search.RepositoryRevisions) (results []SearchResultResolver, common *searchResultsCommon, err error) {
		searchedBatches = append(searchedBatches, batch)
		common = &searchResultsCommon{}
		for _, repoRev := range batch {
			for _, rev := range repoRev.Revs {
				rev := rev.RevSpec
				for i := 0; i < 3; i++ {
					results = append(results, result(repoRev.Repo, fmt.Sprintf("some/file%d.go", i), rev))
				}
			}
			common.repos = append(common.repos, repoRev.Repo)
		}
		return
	}
	noResultsExecutor := func(batch []*search.RepositoryRevisions) (results []SearchResultResolver, common *searchResultsCommon, err error) {
		return nil, &searchResultsCommon{}, nil
	}
	ctx := context.Background()

	tests := []struct {
		name                string
		executor            executor
		request             *searchPaginationInfo
		wantSearchedBatches [][]*search.RepositoryRevisions
		wantCursor          *searchCursor
		wantResults         []SearchResultResolver
		wantCommon          *searchResultsCommon
		wantErr             error
	}{
		{
			name: "first request",
			request: &searchPaginationInfo{
				cursor: &searchCursor{},
				limit:  10,
			},
			wantSearchedBatches: [][]*search.RepositoryRevisions{
				{
					repoRevs("1", "master"),
					repoRevs("2", "master"),
					repoRevs("3", "master", "feature"),
					repoRevs("4", "master"),
				},
			},
			wantCursor: &searchCursor{RepositoryOffset: 2, ResultOffset: 4},
			wantResults: []SearchResultResolver{
				result(repo("1"), "some/file0.go", "master"),
				result(repo("1"), "some/file1.go", "master"),
				result(repo("1"), "some/file2.go", "master"),
				result(repo("2"), "some/file0.go", "master"),
				result(repo("2"), "some/file1.go", "master"),
				result(repo("2"), "some/file2.go", "master"),
				result(repo("3"), "some/file0.go", "master"),
				result(repo("3"), "some/file1.go", "master"),
				result(repo("3"), "some/file2.go", "master"),
				result(repo("3"), "some/file0.go", "feature"),
			},
			wantCommon: &searchResultsCommon{
				repos:       []*types.Repo{repo("1"), repo("2"), repo("3")},
				partial:     map[api.RepoName]struct{}{},
				resultCount: 10,
			},
		},
		{
			name: "second request",
			request: &searchPaginationInfo{
				cursor: &searchCursor{RepositoryOffset: 2, ResultOffset: 4},
				limit:  10,
			},
			wantSearchedBatches: [][]*search.RepositoryRevisions{
				{
					repoRevs("3", "master", "feature"),
					repoRevs("4", "master"),
					repoRevs("5", "master"),
				},
			},
			wantCursor: &searchCursor{RepositoryOffset: 5, ResultOffset: 0, Finished: true},
			wantResults: []SearchResultResolver{
				result(repo("3"), "some/file1.go", "feature"),
				result(repo("3"), "some/file2.go", "feature"),
				result(repo("4"), "some/file0.go", "master"),
				result(repo("4"), "some/file1.go", "master"),
				result(repo("4"), "some/file2.go", "master"),
				result(repo("5"), "some/file0.go", "master"),
				result(repo("5"), "some/file1.go", "master"),
				result(repo("5"), "some/file2.go", "master"),
			},
			wantCommon: &searchResultsCommon{
				repos:   []*types.Repo{repo("3"), repo("4"), repo("5")},
				partial: map[api.RepoName]struct{}{},
			},
		},
		{
			name: "small limit, first request",
			request: &searchPaginationInfo{
				cursor: &searchCursor{},
				limit:  1,
			},
			wantSearchedBatches: [][]*search.RepositoryRevisions{
				{
					repoRevs("1", "master"),
					repoRevs("2", "master"),
					repoRevs("3", "master", "feature"),
					repoRevs("4", "master"),
				},
			},
			wantCursor: &searchCursor{RepositoryOffset: 0, ResultOffset: 1},
			wantResults: []SearchResultResolver{
				result(repo("1"), "some/file0.go", "master"),
			},
			wantCommon: &searchResultsCommon{
				repos:       []*types.Repo{repo("1")},
				partial:     map[api.RepoName]struct{}{},
				resultCount: 1,
			},
		},
		{
			name: "small limit, second request",
			request: &searchPaginationInfo{
				cursor: &searchCursor{RepositoryOffset: 0, ResultOffset: 1},
				limit:  1,
			},
			wantSearchedBatches: [][]*search.RepositoryRevisions{
				{
					repoRevs("1", "master"),
					repoRevs("2", "master"),
					repoRevs("3", "master", "feature"),
					repoRevs("4", "master"),
				},
			},
			wantCursor: &searchCursor{RepositoryOffset: 0, ResultOffset: 2},
			wantResults: []SearchResultResolver{
				result(repo("1"), "some/file1.go", "master"),
			},
			wantCommon: &searchResultsCommon{
				repos:       []*types.Repo{repo("1")},
				partial:     map[api.RepoName]struct{}{},
				resultCount: 1,
			},
		},
		{
			name:     "no results",
			executor: noResultsExecutor,
			request: &searchPaginationInfo{
				cursor: &searchCursor{},
				limit:  1,
			},
			wantCursor: &searchCursor{RepositoryOffset: 1, ResultOffset: 0, Finished: true},
			wantCommon: &searchResultsCommon{
				partial: map[api.RepoName]struct{}{},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			searchedBatches = nil
			plan := &repoPaginationPlan{
				pagination:          test.request,
				repositories:        searchRepos,
				searchBucketDivisor: 8,
				searchBucketMin:     4,
				searchBucketMax:     10,
				mockNumTotalRepos:   func() int { return len(searchRepos) },
			}
			executor := resultsExecutor
			if test.executor != nil {
				executor = test.executor
			}
			cursor, results, common, err := plan.execute(ctx, executor)
			if !cmp.Equal(test.wantCursor, cursor) {
				t.Error("wantCursor != cursor", cmp.Diff(test.wantCursor, cursor))
			}
			if !cmp.Equal(test.wantResults, results) {
				t.Error("wantResults != results", cmp.Diff(test.wantResults, results))
			}
			if !cmp.Equal(test.wantCommon, common) {
				t.Error("wantCommon != common", cmp.Diff(test.wantCommon, common))
			}
			if !cmp.Equal(test.wantErr, err) {
				t.Error("wantErr != err", cmp.Diff(test.wantErr, err))
			}
			if !cmp.Equal(test.wantSearchedBatches, searchedBatches) {
				t.Error("wantSearchedBatches != searchedBatches", cmp.Diff(test.wantSearchedBatches, searchedBatches))
			}
		})
	}
}

func TestSearchPagination_issue_6287(t *testing.T) {
	revs := func(rev ...string) (revs []search.RevisionSpecifier) {
		for _, r := range rev {
			revs = append(revs, search.RevisionSpecifier{RevSpec: r})
		}
		return revs
	}
	repo := func(name string) *types.Repo {
		return &types.Repo{Name: api.RepoName(name)}
	}
	result := mkFileMatch
	repoRevs := func(name string, rev ...string) *search.RepositoryRevisions {
		return &search.RepositoryRevisions{
			Repo: repo(name),
			Revs: revs(rev...),
		}
	}
	repoResults := map[string][]SearchResultResolver{
		"1": {
			result(repo("1"), "a.go"),
			result(repo("1"), "b.go"),
		},
		"2": {
			result(repo("2"), "a.go"),
			result(repo("2"), "b.go"),
			result(repo("2"), "c.go"),
			result(repo("2"), "d.go"),
			result(repo("2"), "e.go"),
		},
	}
	searchRepos := []*search.RepositoryRevisions{
		repoRevs("1", "master"),
		repoRevs("2", "master"),
	}
	executor := func(batch []*search.RepositoryRevisions) (results []SearchResultResolver, common *searchResultsCommon, err error) {
		common = &searchResultsCommon{}
		for _, repoRev := range batch {
			results = append(results, repoResults[string(repoRev.Repo.Name)]...)
			common.repos = append(common.repos, repoRev.Repo)
		}
		return
	}
	ctx := context.Background()

	tests := []struct {
		name        string
		request     *searchPaginationInfo
		wantCursor  *searchCursor
		wantResults []SearchResultResolver
		wantErr     error
	}{
		{
			name: "request 1",
			request: &searchPaginationInfo{
				cursor: &searchCursor{},
				limit:  3,
			},
			wantCursor: &searchCursor{RepositoryOffset: 1, ResultOffset: 1},
			wantResults: []SearchResultResolver{
				result(repo("1"), "a.go"),
				result(repo("1"), "b.go"),
				result(repo("2"), "a.go"),
			},
		},
		{
			name: "request 2",
			request: &searchPaginationInfo{
				cursor: &searchCursor{RepositoryOffset: 1, ResultOffset: 1},
				limit:  3,
			},
			wantCursor: &searchCursor{RepositoryOffset: 1, ResultOffset: 4},
			wantResults: []SearchResultResolver{
				result(repo("2"), "b.go"),
				result(repo("2"), "c.go"),
				result(repo("2"), "d.go"),
			},
		},
		{
			name: "request 3",
			request: &searchPaginationInfo{
				cursor: &searchCursor{RepositoryOffset: 1, ResultOffset: 4},
				limit:  3,
			},
			wantCursor: &searchCursor{RepositoryOffset: 2, ResultOffset: 0, Finished: true},
			wantResults: []SearchResultResolver{
				result(repo("2"), "e.go"),
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			plan := &repoPaginationPlan{
				pagination:          test.request,
				repositories:        searchRepos,
				searchBucketDivisor: 8,
				searchBucketMin:     4,
				searchBucketMax:     10,
				mockNumTotalRepos:   func() int { return len(searchRepos) },
			}
			cursor, results, _, err := plan.execute(ctx, executor)
			if !cmp.Equal(test.wantCursor, cursor) {
				t.Error("wantCursor != cursor", cmp.Diff(test.wantCursor, cursor))
			}
			if !cmp.Equal(test.wantResults, results) {
				t.Error("wantResults != results", cmp.Diff(test.wantResults, results))
			}
			if !cmp.Equal(test.wantErr, err) {
				t.Error("wantErr != err", cmp.Diff(test.wantErr, err))
			}
		})
	}
}

// TestSearchPagination_cloning_missing is a joint test for both
// repoPaginationPlan and sliceSearchResults's handling of cloning and missing
// repositories.
func TestSearchPagination_cloning_missing(t *testing.T) {
	revs := func(rev ...string) (revs []search.RevisionSpecifier) {
		for _, r := range rev {
			revs = append(revs, search.RevisionSpecifier{RevSpec: r})
		}
		return revs
	}
	repo := func(name string) *types.Repo {
		return &types.Repo{Name: api.RepoName(name)}
	}
	result := mkFileMatch
	repoRevs := func(name string, rev ...string) *search.RepositoryRevisions {
		return &search.RepositoryRevisions{
			Repo: repo(name),
			Revs: revs(rev...),
		}
	}
	repoResults := map[string][]SearchResultResolver{
		"a": {
			result(repo("a"), "a.go"),
		},
		"c": {
			result(repo("c"), "a.go"),
		},
		"f": {
			result(repo("f"), "a.go"),
		},
	}
	repoMissing := map[string]*types.Repo{
		"b": repo("b"),
		"e": repo("e"),
	}
	repoCloning := map[string]*types.Repo{
		"d": repo("d"),
	}
	searchRepos := []*search.RepositoryRevisions{
		repoRevs("a", "master"),
		repoRevs("b", "master"),
		repoRevs("c", "master"),
		repoRevs("d", "master"),
		repoRevs("e", "master"),
		repoRevs("f", "master"),
	}
	executor := func(batch []*search.RepositoryRevisions) (results []SearchResultResolver, common *searchResultsCommon, err error) {
		common = &searchResultsCommon{}
		for _, repoRev := range batch {
			if res, ok := repoResults[string(repoRev.Repo.Name)]; ok {
				results = append(results, res...)
				common.repos = append(common.repos, repoRev.Repo)
			}
			if missing, ok := repoMissing[string(repoRev.Repo.Name)]; ok {
				common.missing = append(common.missing, missing)
			}
			if cloning, ok := repoCloning[string(repoRev.Repo.Name)]; ok {
				common.cloning = append(common.cloning, cloning)
			}
		}
		return
	}
	ctx := context.Background()

	tests := []struct {
		name        string
		request     *searchPaginationInfo
		searchRepos []*search.RepositoryRevisions
		wantCursor  *searchCursor
		wantResults []SearchResultResolver
		wantCommon  *searchResultsCommon
		wantErr     error
	}{
		{
			name: "repo a",
			request: &searchPaginationInfo{
				cursor: &searchCursor{},
				limit:  1,
			},
			wantCursor: &searchCursor{RepositoryOffset: 1, ResultOffset: 0},
			wantResults: []SearchResultResolver{
				result(repo("a"), "a.go"),
			},
			wantCommon: &searchResultsCommon{
				partial:     map[api.RepoName]struct{}{},
				repos:       []*types.Repo{repo("a")},
				resultCount: 1,
			},
		},
		{
			name: "missing repo b, repo c",
			request: &searchPaginationInfo{
				cursor: &searchCursor{RepositoryOffset: 1, ResultOffset: 0},
				limit:  1,
			},
			wantCursor: &searchCursor{RepositoryOffset: 3, ResultOffset: 0},
			wantResults: []SearchResultResolver{
				result(repo("c"), "a.go"),
			},
			wantCommon: &searchResultsCommon{
				partial: map[api.RepoName]struct{}{},
				repos:   []*types.Repo{repo("c")},
				missing: []*types.Repo{repo("b")},
			},
		},
		{
			name: "repo a, missing repo b, repo c",
			request: &searchPaginationInfo{
				cursor: &searchCursor{},
				limit:  2,
			},
			wantCursor: &searchCursor{RepositoryOffset: 3, ResultOffset: 0},
			wantResults: []SearchResultResolver{
				result(repo("a"), "a.go"),
				result(repo("c"), "a.go"),
			},
			wantCommon: &searchResultsCommon{
				partial: map[api.RepoName]struct{}{},
				repos:   []*types.Repo{repo("a"), repo("c")},
				missing: []*types.Repo{repo("b")},
			},
		},
		{
			name: "all",
			request: &searchPaginationInfo{
				cursor: &searchCursor{},
				limit:  3,
			},
			wantCursor: &searchCursor{RepositoryOffset: 6, ResultOffset: 0, Finished: true},
			wantResults: []SearchResultResolver{
				result(repo("a"), "a.go"),
				result(repo("c"), "a.go"),
				result(repo("f"), "a.go"),
			},
			wantCommon: &searchResultsCommon{
				partial: map[api.RepoName]struct{}{},
				repos:   []*types.Repo{repo("a"), repo("c"), repo("f")},
				cloning: []*types.Repo{repo("d")},
				missing: []*types.Repo{repo("b"), repo("e")},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			plan := &repoPaginationPlan{
				pagination:          test.request,
				repositories:        searchRepos,
				searchBucketDivisor: 8,
				searchBucketMin:     4,
				searchBucketMax:     10,
				mockNumTotalRepos:   func() int { return len(test.searchRepos) },
			}
			cursor, results, common, err := plan.execute(ctx, executor)
			if !cmp.Equal(test.wantCursor, cursor) {
				t.Error("wantCursor != cursor", cmp.Diff(test.wantCursor, cursor))
			}
			if !cmp.Equal(test.wantResults, results) {
				t.Error("wantResults != results", cmp.Diff(test.wantResults, results))
			}
			if !cmp.Equal(test.wantCommon, common) {
				t.Error("wantCommon != common", cmp.Diff(test.wantCommon, common))
			}
			if !cmp.Equal(test.wantErr, err) {
				t.Error("wantErr != err", cmp.Diff(test.wantErr, err))
			}
		})
	}
}
