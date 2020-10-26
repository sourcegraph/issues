package graphqlbackend

import (
	"context"
	"reflect"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/internal/inventory"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/db"
	"github.com/sourcegraph/sourcegraph/internal/search"
	"github.com/sourcegraph/sourcegraph/internal/vcs/git"
	"github.com/sourcegraph/sourcegraph/schema"
	"go.uber.org/atomic"
)

func TestSearchSuggestions(t *testing.T) {
	limitOffset := &db.LimitOffset{Limit: searchLimits().MaxRepos + 1}

	getSuggestions := func(t *testing.T, query, version string) []string {
		t.Helper()
		r, err := (&schemaResolver{}).Search(context.Background(), &SearchArgs{Query: query, Version: version})
		if err != nil {
			t.Fatal("Search:", err)
		}
		results, err := r.Suggestions(context.Background(), &searchSuggestionsArgs{})
		if err != nil {
			t.Fatal("Suggestions:", err)
		}
		resultDescriptions := make([]string, len(results))
		for i, result := range results {
			resultDescriptions[i] = testStringResult(result)
		}
		return resultDescriptions
	}
	testSuggestions := func(t *testing.T, query, version string, want []string) {
		t.Helper()
		got := getSuggestions(t, query, version)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got != want\ngot:  %v\nwant: %v", got, want)
		}
	}

	mockSearchSymbols = func(ctx context.Context, args *search.TextParameters, limit int) (res []*FileMatchResolver, common *searchResultsCommon, err error) {
		// TODO test symbol suggestions
		return nil, nil, nil
	}
	defer func() { mockSearchSymbols = nil }()

	mockDecodedViewerFinalSettings = &schema.Settings{}
	defer func() { mockDecodedViewerFinalSettings = nil }()

	searchVersions := []string{"V1", "V2"}

	t.Run("empty", func(t *testing.T) {
		for _, v := range searchVersions {
			testSuggestions(t, "", v, []string{})
		}
	})

	t.Run("whitespace", func(t *testing.T) {
		for _, v := range searchVersions {
			testSuggestions(t, " ", v, []string{})
		}
	})

	t.Run("single term", func(t *testing.T) {
		mockDecodedViewerFinalSettings = &schema.Settings{}
		defer func() { mockDecodedViewerFinalSettings = nil }()

		var calledReposListAll, calledReposListFoo bool
		db.Mocks.Repos.List = func(_ context.Context, op db.ReposListOptions) ([]*types.Repo, error) {

			// Validate that the following options are invariant
			// when calling the DB through Repos.List, no matter how
			// many times it is called for a single Search(...) operation.
			assertEqual(t, op.OnlyRepoIDs, true)
			assertEqual(t, op.LimitOffset, limitOffset)

			if reflect.DeepEqual(op.IncludePatterns, []string{"foo"}) {
				// when treating term as repo: field
				calledReposListFoo = true
				return []*types.Repo{{Name: "foo-repo"}}, nil
			} else {
				// when treating term as text query
				calledReposListAll = true
				return []*types.Repo{{Name: "bar-repo"}}, nil
			}
			return nil, nil
		}
		db.Mocks.Repos.Count = mockCount
		db.Mocks.Repos.MockGetByName(t, "repo", 1)
		backend.Mocks.Repos.MockResolveRev_NoCheck(t, api.CommitID("deadbeef"))

		defer func() { db.Mocks = db.MockStores{} }()
		git.Mocks.ResolveRevision = func(rev string, opt git.ResolveRevisionOptions) (api.CommitID, error) {
			return api.CommitID("deadbeef"), nil
		}
		defer git.ResetMocks()

		calledSearchFilesInRepos := atomic.NewBool(false)
		mockSearchFilesInRepos = func(args *search.TextParameters) ([]*FileMatchResolver, *searchResultsCommon, error) {
			calledSearchFilesInRepos.Store(true)
			if want := "foo"; args.PatternInfo.Pattern != want {
				t.Errorf("got %q, want %q", args.PatternInfo.Pattern, want)
			}
			fm := mkFileMatch(&types.Repo{Name: "repo"}, "dir/file")
			fm.uri = "git://repo?rev#dir/file"
			fm.CommitID = "rev"
			return []*FileMatchResolver{fm}, &searchResultsCommon{}, nil
		}
		defer func() { mockSearchFilesInRepos = nil }()
		for _, v := range searchVersions {
			testSuggestions(t, "foo", v, []string{"repo:foo-repo", "file:dir/file"})
			if !calledReposListAll {
				t.Error("!calledReposListAll")
			}
			if !calledReposListFoo {
				t.Error("!calledReposListFoo")
			}
			if !calledSearchFilesInRepos.Load() {
				t.Error("!calledSearchFilesInRepos")
			}
		}
	})

	t.Run("repogroup: and single term", func(t *testing.T) {
		t.Skip("TODO(slimsag): this test is not reliable")
		var mu sync.Mutex

		mockDecodedViewerFinalSettings = &schema.Settings{}
		defer func() { mockDecodedViewerFinalSettings = nil }()

		var calledReposListReposInGroup, calledReposListFooRepo3 bool
		db.Mocks.Repos.List = func(_ context.Context, op db.ReposListOptions) ([]*types.Repo, error) {
			mu.Lock()
			defer mu.Unlock()
			wantReposInGroup := db.ReposListOptions{IncludePatterns: []string{`^foo-repo1$|^repo3$`}, LimitOffset: limitOffset}    // when treating term as repo: field
			wantFooRepo3 := db.ReposListOptions{IncludePatterns: []string{"foo", `^foo-repo1$|^repo3$`}, LimitOffset: limitOffset} // when treating term as repo: field
			if reflect.DeepEqual(op, wantReposInGroup) {
				calledReposListReposInGroup = true
				return []*types.Repo{
					{Name: "foo-repo1"},
					{Name: "repo3"},
				}, nil
			} else if reflect.DeepEqual(op, wantFooRepo3) {
				calledReposListFooRepo3 = true
				return []*types.Repo{{Name: "foo-repo1"}}, nil
			}
			t.Errorf("got %+v, want %+v or %+v", op, wantReposInGroup, wantFooRepo3)
			return nil, nil
		}
		db.Mocks.Repos.Count = mockCount
		defer func() { db.Mocks = db.MockStores{} }()
		db.Mocks.Repos.MockGetByName(t, "repo", 1)
		backend.Mocks.Repos.MockResolveRev_NoCheck(t, api.CommitID("deadbeef"))

		calledSearchFilesInRepos := atomic.NewBool(false)
		mockSearchFilesInRepos = func(args *search.TextParameters) ([]*FileMatchResolver, *searchResultsCommon, error) {
			mu.Lock()
			defer mu.Unlock()
			calledSearchFilesInRepos.Store(true)
			if args.PatternInfo.Pattern != "." && args.PatternInfo.Pattern != "foo" {
				t.Errorf("got %q, want %q", args.PatternInfo.Pattern, `"foo" or "."`)
			}
			mk := func(name api.RepoName, path string) *FileMatchResolver {
				fm := mkFileMatch(&types.Repo{Name: name}, path)
				fm.uri = fileMatchURI(name, "rev", path)
				fm.CommitID = "rev"
				return fm
			}
			return []*FileMatchResolver{
				mk("repo3", "dir/foo-repo3-file-name-match"),
				mk("repo1", "dir/foo-repo1-file-name-match"),
				mk("repo", "dir/file-content-match"),
			}, &searchResultsCommon{}, nil
		}
		defer func() { mockSearchFilesInRepos = nil }()

		calledResolveRepoGroups := false
		mockResolveRepoGroups = func() (map[string][]RepoGroupValue, error) {
			mu.Lock()
			defer mu.Unlock()
			calledResolveRepoGroups = true
			return map[string][]RepoGroupValue{
				"baz": {
					RepoPath("foo-repo1"),
					RepoPath("repo3"),
				},
			}, nil
		}
		defer func() { mockResolveRepoGroups = nil }()
		for _, v := range searchVersions {
			testSuggestions(t, "repogroup:baz foo", v, []string{"repo:foo-repo1", "file:dir/foo-repo3-file-name-match", "file:dir/foo-repo1-file-name-match", "file:dir/file-content-match"})
			if !calledReposListReposInGroup {
				t.Error("!calledReposListReposInGroup")
			}
			if !calledReposListFooRepo3 {
				t.Error("!calledReposListFooRepo3")
			}
			if !calledSearchFilesInRepos.Load() {
				t.Error("!calledSearchFilesInRepos")
			}
			if !calledResolveRepoGroups {
				t.Error("!calledResolveRepoGroups")
			}

		}
	})

	t.Run("repo: field", func(t *testing.T) {
		var mu sync.Mutex

		mockDecodedViewerFinalSettings = &schema.Settings{}
		defer func() { mockDecodedViewerFinalSettings = nil }()

		calledReposList := false
		db.Mocks.Repos.List = func(_ context.Context, op db.ReposListOptions) ([]*types.Repo, error) {
			mu.Lock()
			defer mu.Unlock()
			calledReposList = true

			// Validate that the following options are invariant
			// when calling the DB through Repos.List, no matter how
			// many times it is called for a single Search(...) operation.
			assertEqual(t, op.OnlyRepoIDs, true)
			assertEqual(t, op.LimitOffset, limitOffset)
			assertEqual(t, op.IncludePatterns, []string{"foo"})

			return []*types.Repo{{Name: "foo-repo"}}, nil
		}
		db.Mocks.Repos.Count = mockCount
		defer func() { db.Mocks.Repos.List = nil }()

		// Mock to bypass language suggestions.
		mockShowLangSuggestions = func() ([]*searchSuggestionResolver, error) { return nil, nil }
		defer func() { mockShowLangSuggestions = nil }()

		calledSearchFilesInRepos := atomic.NewBool(false)
		mockSearchFilesInRepos = func(args *search.TextParameters) ([]*FileMatchResolver, *searchResultsCommon, error) {
			mu.Lock()
			defer mu.Unlock()
			calledSearchFilesInRepos.Store(true)
			repos, err := getRepos(context.Background(), args.RepoPromise)
			if err != nil {
				t.Error(err)
			}
			if want := "foo-repo"; len(repos) != 1 || string(repos[0].Repo.Name) != want {
				t.Errorf("got %q, want %q", repos, want)
			}
			return []*FileMatchResolver{
				mkFileMatch(&types.Repo{Name: "foo-repo"}, "dir/file"),
			}, &searchResultsCommon{}, nil
		}
		defer func() { mockSearchFilesInRepos = nil }()

		for _, v := range searchVersions {
			testSuggestions(t, "repo:foo", v, []string{"repo:foo-repo", "file:dir/file"})
			if !calledReposList {
				t.Error("!calledReposList")
			}
			if !calledSearchFilesInRepos.Load() {
				t.Error("!calledSearchFilesInRepos")
			}
		}
	})

	t.Run("repo: field for language suggestions", func(t *testing.T) {
		mockDecodedViewerFinalSettings = &schema.Settings{}
		defer func() { mockDecodedViewerFinalSettings = nil }()

		db.Mocks.Repos.List = func(_ context.Context, have db.ReposListOptions) ([]*types.Repo, error) {
			want := db.ReposListOptions{
				IncludePatterns: []string{"foo"},
				OnlyRepoIDs:     true,
				LimitOffset: &db.LimitOffset{
					Limit: 1,
				},
			}
			if !reflect.DeepEqual(have, want) {
				t.Error(cmp.Diff(have, want))
			}
			return []*types.Repo{{Name: "foo-repo"}}, nil
		}
		db.Mocks.Repos.Count = mockCount
		defer func() { db.Mocks.Repos.List = nil }()
		git.Mocks.ResolveRevision = func(rev string, opt git.ResolveRevisionOptions) (api.CommitID, error) {
			return api.CommitID("deadbeef"), nil
		}
		defer git.ResetMocks()

		calledReposGetInventory := false
		backend.Mocks.Repos.GetInventory = func(_ context.Context, _ *types.Repo, _ api.CommitID) (*inventory.Inventory, error) {
			calledReposGetInventory = true
			return &inventory.Inventory{
				Languages: []inventory.Lang{
					{Name: "Go"},
					{Name: "TypeScript"},
					{Name: "Java"},
				},
			}, nil
		}
		defer func() { backend.Mocks.Repos.GetInventory = nil }()

		// Mock to bypass other suggestions.
		mockShowRepoSuggestions = func() ([]*searchSuggestionResolver, error) { return nil, nil }
		defer func() { mockShowRepoSuggestions = nil }()
		mockShowFileSuggestions = func() ([]*searchSuggestionResolver, error) { return nil, nil }
		defer func() { mockShowFileSuggestions = nil }()
		mockShowSymbolMatches = func() ([]*searchSuggestionResolver, error) { return nil, nil }
		defer func() { mockShowSymbolMatches = nil }()

		for _, v := range searchVersions {
			testSuggestions(t, "repo:foo@master", v, []string{"lang:go", "lang:java", "lang:typescript"})
			if !calledReposGetInventory {
				t.Error("!calledReposGetInventory")
			}
		}
	})

	t.Run("repo: and file: field", func(t *testing.T) {
		var mu sync.Mutex

		mockDecodedViewerFinalSettings = &schema.Settings{}
		defer func() { mockDecodedViewerFinalSettings = nil }()

		calledReposList := false
		db.Mocks.Repos.List = func(_ context.Context, op db.ReposListOptions) ([]*types.Repo, error) {
			mu.Lock()
			defer mu.Unlock()
			calledReposList = true

			// Validate that the following options are invariant
			// when calling the DB through Repos.List, no matter how
			// many times it is called for a single Search(...) operation.
			assertEqual(t, op.OnlyRepoIDs, true)
			assertEqual(t, op.LimitOffset, limitOffset)
			assertEqual(t, op.IncludePatterns, []string{"foo"})

			return []*types.Repo{{Name: "foo-repo"}}, nil
		}
		db.Mocks.Repos.Count = mockCount
		defer func() { db.Mocks.Repos.List = nil }()

		// Mock to bypass language suggestions.
		mockShowLangSuggestions = func() ([]*searchSuggestionResolver, error) { return nil, nil }
		defer func() { mockShowLangSuggestions = nil }()

		calledSearchFilesInRepos := atomic.NewBool(false)
		mockSearchFilesInRepos = func(args *search.TextParameters) ([]*FileMatchResolver, *searchResultsCommon, error) {
			mu.Lock()
			defer mu.Unlock()
			calledSearchFilesInRepos.Store(true)
			repos, err := getRepos(context.Background(), args.RepoPromise)
			if err != nil {
				t.Error(err)
			}
			if want := "foo-repo"; len(repos) != 1 || string(repos[0].Repo.Name) != want {
				t.Errorf("got %q, want %q", repos, want)
			}
			return []*FileMatchResolver{
				mkFileMatch(&types.Repo{Name: "foo-repo"}, "dir/bar-file"),
			}, &searchResultsCommon{}, nil
		}
		defer func() { mockSearchFilesInRepos = nil }()

		for _, v := range searchVersions {
			testSuggestions(t, "repo:foo file:bar", v, []string{"file:dir/bar-file"})
			if !calledReposList {
				t.Error("!calledReposList")
			}
			if !calledSearchFilesInRepos.Load() {
				t.Error("!calledSearchFilesInRepos")
			}
		}
	})
}
