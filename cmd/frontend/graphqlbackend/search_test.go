package graphqlbackend

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/zoekt"
	"github.com/graph-gophers/graphql-go"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/db"
	"github.com/sourcegraph/sourcegraph/internal/gitserver"
	"github.com/sourcegraph/sourcegraph/internal/search"
	searchbackend "github.com/sourcegraph/sourcegraph/internal/search/backend"
	"github.com/sourcegraph/sourcegraph/internal/search/query"
	querytypes "github.com/sourcegraph/sourcegraph/internal/search/query/types"
	"github.com/sourcegraph/sourcegraph/internal/vcs/git"
	"github.com/sourcegraph/sourcegraph/schema"
)

func TestSearch(t *testing.T) {
	type Results struct {
		Results     []interface{}
		ResultCount int
	}
	tcs := []struct {
		name                         string
		searchQuery                  string
		searchVersion                string
		reposListMock                func(v0 context.Context, v1 db.ReposListOptions) ([]*types.Repo, error)
		repoRevsMock                 func(spec string, opt git.ResolveRevisionOptions) (api.CommitID, error)
		externalServicesListMock     func(opt db.ExternalServicesListOptions) ([]*types.ExternalService, error)
		phabricatorGetRepoByNameMock func(repo api.RepoName) (*types.PhabricatorRepo, error)
		wantResults                  Results
	}{
		{
			name:        "empty query against no repos gets no results",
			searchQuery: "",
			reposListMock: func(v0 context.Context, v1 db.ReposListOptions) ([]*types.Repo, error) {
				return nil, nil
			},
			repoRevsMock: func(spec string, opt git.ResolveRevisionOptions) (api.CommitID, error) {
				return "", nil
			},
			externalServicesListMock: func(opt db.ExternalServicesListOptions) ([]*types.ExternalService, error) {
				return nil, nil
			},
			phabricatorGetRepoByNameMock: func(repo api.RepoName) (*types.PhabricatorRepo, error) {
				return nil, nil
			},
			wantResults: Results{
				Results:     nil,
				ResultCount: 0,
			},
			searchVersion: "V1",
		},
		{
			name:        "empty query against empty repo gets no results",
			searchQuery: "",
			reposListMock: func(v0 context.Context, v1 db.ReposListOptions) ([]*types.Repo, error) {
				return []*types.Repo{{Name: "test"}},

					nil
			},
			repoRevsMock: func(spec string, opt git.ResolveRevisionOptions) (api.CommitID, error) {
				return "", nil
			},
			externalServicesListMock: func(opt db.ExternalServicesListOptions) ([]*types.ExternalService, error) {
				return nil, nil
			},
			phabricatorGetRepoByNameMock: func(repo api.RepoName) (*types.PhabricatorRepo, error) {
				return nil, nil
			},
			wantResults: Results{
				Results:     nil,
				ResultCount: 0,
			},
			searchVersion: "V1",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			conf.Mock(&conf.Unified{})
			defer conf.Mock(nil)
			vars := map[string]interface{}{"query": tc.searchQuery, "version": tc.searchVersion}

			mockDecodedViewerFinalSettings = &schema.Settings{}
			defer func() { mockDecodedViewerFinalSettings = nil }()

			db.Mocks.Repos.List = tc.reposListMock
			sr := &schemaResolver{}
			schema, err := graphql.ParseSchema(Schema, sr, graphql.Tracer(prometheusTracer{}))
			if err != nil {
				t.Fatal(err)
			}
			db.Mocks.ExternalServices.List = tc.externalServicesListMock
			db.Mocks.Phabricator.GetByName = tc.phabricatorGetRepoByNameMock
			git.Mocks.ResolveRevision = tc.repoRevsMock
			result := schema.Exec(context.Background(), testSearchGQLQuery, "", vars)
			if len(result.Errors) > 0 {
				t.Fatalf("graphQL query returned errors: %+v", result.Errors)
			}
			var search struct {
				Results Results
			}
			if err := json.Unmarshal(result.Data, &search); err != nil {
				t.Fatalf("parsing JSON response: %v", err)
			}
			gotResults := search.Results
			if !reflect.DeepEqual(gotResults, tc.wantResults) {
				t.Fatalf("results = %+v, want %+v", gotResults, tc.wantResults)
			}
		})
	}
}

var testSearchGQLQuery = `
		fragment FileMatchFields on FileMatch {
			repository {
				name
				url
			}
			file {
				name
				path
				url
				commit {
					oid
				}
			}
			lineMatches {
				preview
				lineNumber
				offsetAndLengths
				limitHit
			}
		}

		fragment CommitSearchResultFields on CommitSearchResult {
			messagePreview {
				value
				highlights{
					line
					character
					length
				}
			}
			diffPreview {
				value
				highlights {
					line
					character
					length
				}
			}
			label {
				html
			}
			url
			matches {
				url
				body {
					html
					text
				}
				highlights {
					character
					line
					length
				}
			}
			commit {
				repository {
					name
				}
				oid
				url
				subject
				author {
					date
					person {
						displayName
					}
				}
			}
		}

		fragment RepositoryFields on Repository {
			name
			url
			externalURLs {
				serviceType
				url
			}
			label {
				html
			}
		}

		query ($query: String!, $version: SearchVersion!, $patternType: SearchPatternType) {
			site {
				buildVersion
			}
			search(query: $query, version: $version, patternType: $patternType) {
				results {
					results{
						__typename
						... on FileMatch {
						...FileMatchFields
					}
						... on CommitSearchResult {
						...CommitSearchResultFields
					}
						... on Repository {
						...RepositoryFields
					}
					}
					limitHit
					cloning {
						name
					}
					missing {
						name
					}
					timedout {
						name
					}
					resultCount
					elapsedMilliseconds
				}
			}
		}
`

func testStringResult(result *searchSuggestionResolver) string {
	var name string
	switch r := result.result.(type) {
	case *RepositoryResolver:
		name = "repo:" + string(r.repo.Name)
	case *GitTreeEntryResolver:
		name = "file:" + r.Path()
	case *languageResolver:
		name = "lang:" + r.name
	default:
		panic("never here")
	}
	if result.score == 0 {
		return "<removed>"
	}
	return name
}

func TestDefaultRepositories(t *testing.T) {
	tcs := []struct {
		name             string
		defaultsInDb     []string
		indexedRepoNames map[string]bool
		want             []string
		excludePatterns  []string
	}{
		{
			name:             "none in db => none returned",
			defaultsInDb:     nil,
			indexedRepoNames: nil,
			want:             nil,
		},
		{
			name:             "two in db, one indexed => indexed repo returned",
			defaultsInDb:     []string{"unindexedrepo", "indexedrepo"},
			indexedRepoNames: map[string]bool{"indexedrepo": true},
			want:             []string{"indexedrepo"},
		},
		{
			name:             "should not return excluded repo",
			defaultsInDb:     []string{"unindexedrepo1", "indexedrepo1", "indexedrepo2", "indexedrepo3"},
			indexedRepoNames: map[string]bool{"indexedrepo1": true, "indexedrepo2": true, "indexedrepo3": true},
			excludePatterns:  []string{"indexedrepo3"},
			want:             []string{"indexedrepo1", "indexedrepo2"},
		},
		{
			name:             "should not return excluded repo (case insensitive)",
			defaultsInDb:     []string{"unindexedrepo1", "indexedrepo1", "indexedrepo2", "Indexedrepo3"},
			indexedRepoNames: map[string]bool{"indexedrepo1": true, "indexedrepo2": true, "Indexedrepo3": true},
			excludePatterns:  []string{"indexedrepo3"},
			want:             []string{"indexedrepo1", "indexedrepo2"},
		},
		{
			name:             "should not return excluded repos ending in `test`",
			defaultsInDb:     []string{"repo1", "repo2", "repo-test", "repoTEST"},
			indexedRepoNames: map[string]bool{"repo1": true, "repo2": true, "repo-test": true, "repoTEST": true},
			excludePatterns:  []string{"test$"},
			want:             []string{"repo1", "repo2"},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {

			var drs []*types.Repo
			for i, name := range tc.defaultsInDb {
				r := &types.Repo{
					ID:   api.RepoID(i),
					Name: api.RepoName(name),
				}
				drs = append(drs, r)
			}
			getRawDefaultRepos := func(ctx context.Context) ([]*types.Repo, error) {
				return drs, nil
			}

			var indexed []*zoekt.RepoListEntry
			for name := range tc.indexedRepoNames {
				indexed = append(indexed, &zoekt.RepoListEntry{Repository: zoekt.Repository{Name: name}})
			}
			z := &searchbackend.Zoekt{
				Client:       &fakeSearcher{repos: indexed},
				DisableCache: true,
			}

			ctx := context.Background()
			drs, err := defaultRepositories(ctx, getRawDefaultRepos, z, tc.excludePatterns)
			if err != nil {
				t.Fatal(err)
			}
			var drNames []string
			for _, dr := range drs {
				drNames = append(drNames, string(dr.Name))
			}
			if !reflect.DeepEqual(drNames, tc.want) {
				t.Errorf("names of default repos = %v, want %v", drNames, tc.want)
			}
		})
	}
}

func TestDetectSearchType(t *testing.T) {
	typeRegexp := "regexp"
	typeLiteral := "literal"
	testCases := []struct {
		name        string
		version     string
		patternType *string
		input       string
		want        query.SearchType
	}{
		{"V1, no pattern type", "V1", nil, "", query.SearchTypeRegex},
		{"V2, no pattern type", "V2", nil, "", query.SearchTypeLiteral},
		{"V2, no pattern type, input does not produce parse error", "V2", nil, "/-/godoc", query.SearchTypeLiteral},
		{"V1, regexp pattern type", "V1", &typeRegexp, "", query.SearchTypeRegex},
		{"V2, regexp pattern type", "V2", &typeRegexp, "", query.SearchTypeRegex},
		{"V1, literal pattern type", "V1", &typeLiteral, "", query.SearchTypeLiteral},
		{"V2, override regexp pattern type", "V2", &typeLiteral, "patterntype:regexp", query.SearchTypeRegex},
		{"V2, override regex variant pattern type", "V2", &typeLiteral, "patterntype:regex", query.SearchTypeRegex},
		{"V2, override regex variant pattern type with double quotes", "V2", &typeLiteral, `patterntype:"regex"`, query.SearchTypeRegex},
		{"V2, override regex variant pattern type with single quotes", "V2", &typeLiteral, `patterntype:'regex'`, query.SearchTypeRegex},
		{"V1, override literal pattern type", "V1", &typeRegexp, "patterntype:literal", query.SearchTypeLiteral},
		{"V1, override literal pattern type, with case-insensitive query", "V1", &typeRegexp, "pAtTErNTypE:literal", query.SearchTypeLiteral},
	}

	for _, test := range testCases {
		t.Run(test.name, func(*testing.T) {
			got, err := detectSearchType(test.version, test.patternType)
			useNewParser := []bool{true, false}
			for _, parserOpt := range useNewParser {
				got = overrideSearchType(test.input, got, parserOpt)
				if err != nil {
					t.Fatal(err)
				}
				if got != test.want {
					t.Errorf("failed %v, got %v, expected %v", test.name, got, test.want)
				}
			}
		})
	}
}

func TestExactlyOneRepo(t *testing.T) {
	cases := []struct {
		repoFilters []string
		want        bool
	}{
		{
			repoFilters: []string{`^github\.com/sourcegraph/zoekt$`},
			want:        true,
		},
		{
			repoFilters: []string{`^github\.com/sourcegraph/zoekt$@ef3ec23`},
			want:        true,
		},
		{
			repoFilters: []string{`^github\.com/sourcegraph/zoekt$@ef3ec23:deadbeef`},
			want:        true,
		},
		{
			repoFilters: []string{`^.*$`},
			want:        false,
		},

		{
			repoFilters: []string{`^github\.com/sourcegraph/zoekt`},
			want:        false,
		},
		{
			repoFilters: []string{`^github\.com/sourcegraph/zoekt$`, `github\.com/sourcegraph/sourcegraph`},
			want:        false,
		},
	}
	for _, c := range cases {
		t.Run("exactly one repo", func(t *testing.T) {
			if got := exactlyOneRepo(c.repoFilters); got != c.want {
				t.Errorf("got %t, want %t", got, c.want)
			}
		})
	}
}

func TestQuoteSuggestions(t *testing.T) {
	t.Run("regex error", func(t *testing.T) {
		raw := "*"
		_, err := query.Process(raw, query.SearchTypeRegex)
		if err == nil {
			t.Fatalf("error returned from query.Process(%q) is nil", raw)
		}
		alert := alertForQuery(raw, err)
		if !strings.Contains(strings.ToLower(alert.title), "regexp") {
			t.Errorf("title is '%s', want it to contain 'regexp'", alert.title)
		}
		if !strings.Contains(alert.description, "regular expression") {
			t.Errorf("description is '%s', want it to contain 'regular expression'", alert.description)
		}
	})

	t.Run("type error that is not a regex error should show a suggestion", func(t *testing.T) {
		raw := "-foobar"
		_, alert := query.Process(raw, query.SearchTypeRegex)
		if alert == nil {
			t.Fatalf("alert returned from query.Process(%q) is nil", raw)
		}
	})

	t.Run("query parse error", func(t *testing.T) {
		raw := ":"
		_, err := query.Process(raw, query.SearchTypeRegex)
		if err == nil {
			t.Fatalf("error returned from query.Process(%q) is nil", raw)
		}
		alert := alertForQuery(raw, err)
		if strings.Contains(strings.ToLower(alert.title), "regexp") {
			t.Errorf("title is '%s', want it not to contain 'regexp'", alert.title)
		}
		if strings.Contains(alert.description, "regular expression") {
			t.Errorf("description is '%s', want it not to contain 'regular expression'", alert.description)
		}
	})

	t.Run("negated file field with an invalid regex", func(t *testing.T) {
		raw := "-f:(a"
		_, err := query.Process(raw, query.SearchTypeRegex)
		if err == nil {
			t.Fatal("query.Process failed to detect the invalid regex in the f: field")
		}
		alert := alertForQuery(raw, err)
		if len(alert.proposedQueries) != 1 {
			t.Fatalf("got %d proposed queries (%v), want exactly 1", len(alert.proposedQueries), alert.proposedQueries)
		}
	})
}

func TestEueryForStableResults(t *testing.T) {
	cases := []struct {
		query           string
		wantStableCount int32
		wantError       error
	}{
		{
			query:           "foo stable:yes",
			wantStableCount: 30,
		},
		{
			query:           "foo stable:yes count:1000",
			wantStableCount: 1000,
		},
		{
			query:     "foo stable:yes count:5001",
			wantError: fmt.Errorf("Stable searches are limited to at max count:%d results. Consider removing 'stable:', narrowing the search with 'repo:', or using the paginated search API.", maxSearchResultsPerPaginatedRequest),
		},
	}
	for _, c := range cases {
		t.Run("query for stable results", func(t *testing.T) {
			queryInfo, _ := query.Process(c.query, query.SearchTypeLiteral)
			args, queryInfo, err := queryForStableResults(&SearchArgs{}, queryInfo)
			if err != nil {
				if !reflect.DeepEqual(err, c.wantError) {
					t.Errorf("Got error %v, want %v", err, c.wantError)
				}
				return
			}
			if diff := cmp.Diff(*args.First, c.wantStableCount); diff != "" {
				t.Error(diff)
			}
			// Ensure type:file is set.
			fileValue := "file"
			wantTypeValue := querytypes.Value{String: &fileValue}
			gotTypeValues := queryInfo.Fields()["type"]
			if len(gotTypeValues) != 1 && *gotTypeValues[0] != wantTypeValue {
				t.Errorf("Query %s sets stable:yes but is not transformed with type:file.", c.query)
			}
		})
	}
}

func TestVersionContext(t *testing.T) {
	conf.Mock(&conf.Unified{
		SiteConfiguration: schema.SiteConfiguration{
			ExperimentalFeatures: &schema.ExperimentalFeatures{
				VersionContexts: []*schema.VersionContext{
					{
						Name: "ctx-1",
						Revisions: []*schema.VersionContextRevision{
							{Repo: "github.com/sourcegraph/foo", Rev: "some-branch"},
							{Repo: "github.com/sourcegraph/foobar", Rev: "v1.0.0"},
							{Repo: "github.com/sourcegraph/bar", Rev: "e62b6218f61cc1564d6ebcae19f9dafdf1357567"},
						},
					}, {
						Name: "multiple-revs",
						Revisions: []*schema.VersionContextRevision{
							{Repo: "github.com/sourcegraph/foobar", Rev: "v1.0.0"},
							{Repo: "github.com/sourcegraph/foobar", Rev: "v1.1.0"},
							{Repo: "github.com/sourcegraph/bar", Rev: "e62b6218f61cc1564d6ebcae19f9dafdf1357567"},
						},
					},
				},
			},
		},
	})
	defer conf.Mock(nil)

	tcs := []struct {
		name           string
		searchQuery    string
		versionContext string
		// db.ReposListOptions.Names
		wantReposListOptionsNames []string
		reposGetListNames         []string
		wantResults               []string
	}{{
		name:           "query with version context should return the right repositories",
		searchQuery:    "foo",
		versionContext: "ctx-1",
		wantReposListOptionsNames: []string{
			"github.com/sourcegraph/foo",
			"github.com/sourcegraph/foobar",
			"github.com/sourcegraph/bar",
		},
		reposGetListNames: []string{
			"github.com/sourcegraph/foo",
			"github.com/sourcegraph/foobar",
			"github.com/sourcegraph/bar",
		},
		wantResults: []string{
			"github.com/sourcegraph/foo@some-branch",
			"github.com/sourcegraph/foobar@v1.0.0",
			"github.com/sourcegraph/bar@e62b6218f61cc1564d6ebcae19f9dafdf1357567",
		},
	}, {
		name:           "query with version context and subset of repos",
		searchQuery:    "repo:github.com/sourcegraph/foo.*",
		versionContext: "ctx-1",
		wantReposListOptionsNames: []string{
			"github.com/sourcegraph/foo",
			"github.com/sourcegraph/foobar",
			"github.com/sourcegraph/bar",
		},
		reposGetListNames: []string{
			"github.com/sourcegraph/foo",
			"github.com/sourcegraph/foobar",
		},
		wantResults: []string{
			"github.com/sourcegraph/foo@some-branch",
			"github.com/sourcegraph/foobar@v1.0.0",
		},
	}, {
		name:           "query with version context and non-exact search",
		searchQuery:    "repo:github.com/sourcegraph/notincontext",
		versionContext: "ctx-1",
		wantReposListOptionsNames: []string{
			"github.com/sourcegraph/foo",
			"github.com/sourcegraph/foobar",
			"github.com/sourcegraph/bar",
		},
		reposGetListNames: []string{},
		wantResults:       []string{},
	}, {
		name:                      "query with version context and exact repo search",
		searchQuery:               "repo:github.com/sourcegraph/notincontext@v1.0.0",
		versionContext:            "ctx-1",
		wantReposListOptionsNames: []string{},
		reposGetListNames:         []string{"github.com/sourcegraph/notincontext"},
		wantResults:               []string{"github.com/sourcegraph/notincontext@v1.0.0"},
	}, {
		name:           "multiple revs",
		searchQuery:    "foo",
		versionContext: "multiple-revs",
		wantReposListOptionsNames: []string{
			"github.com/sourcegraph/foobar",
			"github.com/sourcegraph/foobar", // we don't mind listing repos twice
			"github.com/sourcegraph/bar",
		},
		reposGetListNames: []string{
			"github.com/sourcegraph/foobar",
			"github.com/sourcegraph/bar",
		},
		wantResults: []string{
			"github.com/sourcegraph/foobar@v1.0.0:v1.1.0",
			"github.com/sourcegraph/bar@e62b6218f61cc1564d6ebcae19f9dafdf1357567",
		},
	}}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			qinfo, err := query.ParseAndCheck(tc.searchQuery)
			if err != nil {
				t.Fatal(err)
			}

			resolver := searchResolver{
				query:          qinfo,
				versionContext: &tc.versionContext,
				userSettings:   &schema.Settings{},
			}

			db.Mocks.Repos.List = func(ctx context.Context, opts db.ReposListOptions) ([]*types.Repo, error) {
				if diff := cmp.Diff(tc.wantReposListOptionsNames, opts.Names, cmpopts.EquateEmpty()); diff != "" {
					t.Fatalf("db.RepostListOptions.Names mismatch (-want, +got):\n%s", diff)
				}
				var repos []*types.Repo
				for _, name := range tc.reposGetListNames {
					repos = append(repos, &types.Repo{Name: api.RepoName(name)})
				}
				return repos, nil
			}

			gotResult, err := resolver.resolveRepositories(context.Background(), nil)
			if err != nil {
				t.Fatal(err)
			}
			var got []string
			for _, repoRev := range gotResult.repoRevs {
				got = append(got, string(repoRev.Repo.Name)+"@"+strings.Join(repoRev.RevSpecs(), ":"))
			}

			if diff := cmp.Diff(tc.wantResults, got, cmpopts.EquateEmpty()); diff != "" {
				t.Fatalf("mismatch (-want, +got):\n%s", diff)
			}
		})
	}
}

func TestComputeExcludedRepositories(t *testing.T) {
	cases := []struct {
		Name              string
		Query             string
		Repos             []types.Repo
		WantExcludedRepos excludedRepos
	}{
		{
			Name:  "filter out forks and archived repos",
			Query: "repo:repo",
			Repos: []types.Repo{
				{
					Name:       "repo-ordinary",
					RepoFields: &types.RepoFields{},
				},
				{
					Name:       "repo-forked",
					RepoFields: &types.RepoFields{Fork: true},
				},
				{
					Name:       "repo-forked-2",
					RepoFields: &types.RepoFields{Fork: true},
				},
				{
					Name:       "repo-archived",
					RepoFields: &types.RepoFields{Archived: true},
				},
			},
			WantExcludedRepos: excludedRepos{forks: 2, archived: 1},
		},
		{
			Name:  "exact repo match does not exclude fork",
			Query: "repo:^repo-forked$",
			Repos: []types.Repo{
				{
					Name:       "repo-forked",
					RepoFields: &types.RepoFields{Fork: true},
				},
			},
			WantExcludedRepos: excludedRepos{forks: 0, archived: 0},
		},
		{
			Name:  "when fork is set don't populate exclude",
			Query: "repo:repo fork:no",
			Repos: []types.Repo{
				{
					Name:       "repo",
					RepoFields: &types.RepoFields{},
				},
				{
					Name:       "repo-forked",
					RepoFields: &types.RepoFields{Fork: true},
				},
			},
			WantExcludedRepos: excludedRepos{forks: 0, archived: 0},
		},
	}

	for _, c := range cases {
		// Setup: parse the query, extract its repo filters, and use
		// those to populate the resolve repo options to pass to the
		// function under test.
		q, err := query.ParseAndCheck(c.Query)
		if err != nil {
			t.Fatal(err)
		}
		r := searchResolver{query: q}
		includePatterns, _ := r.query.RegexpPatterns(query.FieldRepo)
		options := db.ReposListOptions{IncludePatterns: includePatterns}

		// Setup: the mock DB lookup returns forked repo count if OnlyForks is set,
		// and archived repo count if OnlyArchived is set.
		db.Mocks.Repos.Count = func(_ context.Context, options db.ReposListOptions) (int, error) {
			var count int
			if options.OnlyForks {
				for _, repo := range c.Repos {
					if repo.Fork {
						count += 1
					}
				}
			}
			if options.OnlyArchived {
				for _, repo := range c.Repos {
					if repo.Archived {
						count += 1
					}
				}
			}
			return count, nil
		}

		t.Run("exclude repo", func(t *testing.T) {
			got := computeExcludedRepositories(context.Background(), q, options)
			if !reflect.DeepEqual(got, c.WantExcludedRepos) {
				t.Fatalf("results = %+v, want %+v", got, c.WantExcludedRepos)
			}
		})
	}
}

func mkFileMatch(repo *types.Repo, path string, lineNumbers ...int32) *FileMatchResolver {
	if repo == nil {
		repo = &types.Repo{
			ID:   1,
			Name: "repo",
		}
	}
	var lines []*lineMatch
	for _, n := range lineNumbers {
		lines = append(lines, &lineMatch{JLineNumber: n})
	}
	return &FileMatchResolver{
		uri:          fileMatchURI(repo.Name, "", path),
		JPath:        path,
		JLineMatches: lines,
		Repo:         &RepositoryResolver{repo: repo},
	}
}

func TestRevisionValidation(t *testing.T) {

	// mocks a repo repoFoo with revisions revBar and revBas
	git.Mocks.ResolveRevision = func(spec string, opt git.ResolveRevisionOptions) (api.CommitID, error) {
		// trigger errors
		if spec == "bad_commit" {
			return "", git.BadCommitError{}
		}
		if spec == "deadline_exceeded" {
			return "", context.DeadlineExceeded
		}

		// known revisions
		m := map[string]struct{}{
			"revBar": {},
			"revBas": {},
		}
		if _, ok := m[spec]; ok {
			return "", nil
		}
		return "", &gitserver.RevisionNotFoundError{Repo: "repoFoo", Spec: spec}
	}
	defer func() { git.Mocks.ResolveRevision = nil }()

	db.Mocks.Repos.List = func(ctx context.Context, opts db.ReposListOptions) ([]*types.Repo, error) {
		return []*types.Repo{{Name: "repoFoo"}}, nil
	}
	defer func() { db.Mocks.Repos.List = nil }()

	tests := []struct {
		repoFilters              []string
		wantRepoRevs             []*search.RepositoryRevisions
		wantMissingRepoRevisions []*search.RepositoryRevisions
		wantErr                  error
	}{
		{
			repoFilters: []string{"repoFoo@revBar:^revBas"},
			wantRepoRevs: []*search.RepositoryRevisions{{
				Repo: &types.Repo{Name: "repoFoo"},
				Revs: []search.RevisionSpecifier{
					{
						RevSpec:        "revBar",
						RefGlob:        "",
						ExcludeRefGlob: "",
					},
					{
						RevSpec:        "^revBas",
						RefGlob:        "",
						ExcludeRefGlob: "",
					},
				},
			}},
			wantMissingRepoRevisions: nil,
		},
		{
			repoFilters: []string{"repoFoo@*revBar:*!revBas"},
			wantRepoRevs: []*search.RepositoryRevisions{{
				Repo: &types.Repo{Name: "repoFoo"},
				Revs: []search.RevisionSpecifier{
					{
						RevSpec:        "",
						RefGlob:        "revBar",
						ExcludeRefGlob: "",
					},
					{
						RevSpec:        "",
						RefGlob:        "",
						ExcludeRefGlob: "revBas",
					},
				},
			}},
			wantMissingRepoRevisions: nil,
		},
		{
			repoFilters: []string{"repoFoo@revBar:^revQux"},
			wantRepoRevs: []*search.RepositoryRevisions{{
				Repo: &types.Repo{Name: "repoFoo"},
				Revs: []search.RevisionSpecifier{
					{
						RevSpec:        "revBar",
						RefGlob:        "",
						ExcludeRefGlob: "",
					},
				},
				ListRefs: nil,
			}},
			wantMissingRepoRevisions: []*search.RepositoryRevisions{{
				Repo: &types.Repo{Name: "repoFoo"},
				Revs: []search.RevisionSpecifier{
					{
						RevSpec:        "^revQux",
						RefGlob:        "",
						ExcludeRefGlob: "",
					},
				},
			}},
		},
		{
			repoFilters:              []string{"repoFoo@revBar:bad_commit"},
			wantRepoRevs:             nil,
			wantMissingRepoRevisions: nil,
			wantErr:                  git.BadCommitError{},
		},
		{
			repoFilters:              []string{"repoFoo@revBar:^bad_commit"},
			wantRepoRevs:             nil,
			wantMissingRepoRevisions: nil,
			wantErr:                  git.BadCommitError{},
		},
		{
			repoFilters:              []string{"repoFoo@revBar:deadline_exceeded"},
			wantRepoRevs:             nil,
			wantMissingRepoRevisions: nil,
			wantErr:                  context.DeadlineExceeded,
		},
		{
			repoFilters: []string{"repoFoo"},
			wantRepoRevs: []*search.RepositoryRevisions{{
				Repo: &types.Repo{Name: "repoFoo"},
				Revs: []search.RevisionSpecifier{
					{
						RevSpec:        "",
						RefGlob:        "",
						ExcludeRefGlob: "",
					},
				},
			}},
			wantMissingRepoRevisions: nil,
			wantErr:                  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.repoFilters[0], func(t *testing.T) {

			op := resolveRepoOp{repoFilters: tt.repoFilters}
			resolved, err := resolveRepositories(context.Background(), op)

			if diff := cmp.Diff(tt.wantRepoRevs, resolved.repoRevs); diff != "" {
				t.Error(diff)
			}
			if diff := cmp.Diff(tt.wantMissingRepoRevisions, resolved.missingRepoRevs); diff != "" {
				t.Error(diff)
			}
			if tt.wantErr != err {
				t.Errorf("got: %v, expected: %v", err, tt.wantErr)
			}
		})
	}
}

func TestRepoGroupValuesToRegexp(t *testing.T) {
	groups := map[string][]RepoGroupValue{
		"go": {
			RepoPath("github.com/saucegraph/saucegraph"),
			RepoRegexpPattern(`github\.com/golang/.*`),
		},
		"typescript": {
			RepoPath("github.com/eslint/eslint"),
		},
	}

	cases := []struct {
		LookupGroupNames []string
		Want             []string
	}{
		{
			LookupGroupNames: []string{"go"},
			Want: []string{
				`^github\.com/saucegraph/saucegraph$`,
				`github\.com/golang/.*`,
			},
		},
		{
			LookupGroupNames: []string{"go", "typescript"},
			Want: []string{
				`^github\.com/saucegraph/saucegraph$`,
				`github\.com/golang/.*`,
				`^github\.com/eslint/eslint$`,
			},
		},
	}

	for _, c := range cases {
		t.Run("repogroup values to regexp", func(t *testing.T) {
			got := repoGroupValuesToRegexp(c.LookupGroupNames, groups)
			if diff := cmp.Diff(c.Want, got); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func repoRev(revSpec string) *search.RepositoryRevisions {
	return &search.RepositoryRevisions{
		Repo: &types.Repo{ID: api.RepoID(0), Name: "test/repo"},
		Revs: []search.RevisionSpecifier{
			{RevSpec: revSpec},
		},
	}
}

func TestGetRepos(t *testing.T) {
	in := []*search.RepositoryRevisions{repoRev("HEAD")}
	rp := (&search.Promise{}).Resolve(in)
	out, err := getRepos(context.Background(), rp)
	if err != nil {
		t.Error(err)
	}
	if ok := reflect.DeepEqual(in, out); !ok {
		t.Errorf("got %+v, expected %+v", out, in)
	}
}

func TestGetReposWrongUnderlyingType(t *testing.T) {
	in := "anything"
	rp := (&search.Promise{}).Resolve(in)
	_, err := getRepos(context.Background(), rp)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}
