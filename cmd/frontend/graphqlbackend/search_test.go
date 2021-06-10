package graphqlbackend

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/zoekt"
	zoektrpc "github.com/google/zoekt/rpc"
	"github.com/graph-gophers/graphql-go"
	"github.com/keegancsmith/sqlf"

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
	"github.com/sourcegraph/sourcegraph/internal/types"
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
		reposListMock                func(v0 context.Context, v1 database.ReposListOptions) ([]*types.Repo, error)
		repoRevsMock                 func(spec string, opt git.ResolveRevisionOptions) (api.CommitID, error)
		externalServicesListMock     func(opt database.ExternalServicesListOptions) ([]*types.ExternalService, error)
		phabricatorGetRepoByNameMock func(repo api.RepoName) (*types.PhabricatorRepo, error)
		wantResults                  Results
	}{
		{
			name:        "empty query against no repos gets no results",
			searchQuery: "",
			reposListMock: func(v0 context.Context, v1 database.ReposListOptions) ([]*types.Repo, error) {
				return nil, nil
			},
			repoRevsMock: func(spec string, opt git.ResolveRevisionOptions) (api.CommitID, error) {
				return "", nil
			},
			externalServicesListMock: func(opt database.ExternalServicesListOptions) ([]*types.ExternalService, error) {
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
			reposListMock: func(v0 context.Context, v1 database.ReposListOptions) ([]*types.Repo, error) {
				return []*types.Repo{{Name: "test"}},

					nil
			},
			repoRevsMock: func(spec string, opt git.ResolveRevisionOptions) (api.CommitID, error) {
				return "", nil
			},
			externalServicesListMock: func(opt database.ExternalServicesListOptions) ([]*types.ExternalService, error) {
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

			db := new(dbtesting.MockDB)
			database.Mocks.Repos.List = tc.reposListMock
			sr := &schemaResolver{db: db}
			schema, err := graphql.ParseSchema(mainSchema, sr, graphql.Tracer(&prometheusTracer{db: db}))
			if err != nil {
				t.Fatal(err)
			}
			database.Mocks.ExternalServices.List = tc.externalServicesListMock
			database.Mocks.Phabricator.GetByName = tc.phabricatorGetRepoByNameMock
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
				serviceKind
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

func testStringResult(result SearchSuggestionResolver) string {
	var name string
	switch r := result.(type) {
	case repositorySuggestionResolver:
		name = "repo:" + r.repo.Name()
	case gitTreeSuggestionResolver:
		name = "file:" + r.gitTreeEntry.Path()
	case languageSuggestionResolver:
		name = "lang:" + r.lang.name
	case symbolSuggestionResolver:
		name = "symbol:" + r.symbol.Symbol.Name
	default:
		panic("never here")
	}
	if result.Score() == 0 {
		return "<removed>"
	}
	return name
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
			got = overrideSearchType(test.input, got)
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Errorf("failed %v, got %v, expected %v", test.name, got, test.want)
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
			if got := searchrepos.ExactlyOneRepo(c.repoFilters); got != c.want {
				t.Errorf("got %t, want %t", got, c.want)
			}
		})
	}
}

func TestQuoteSuggestions(t *testing.T) {
	t.Run("regex error", func(t *testing.T) {
		raw := "*"
		_, err := query.Pipeline(query.InitRegexp(raw))
		if err == nil {
			t.Fatalf("error returned from query.ParseRegexp(%q) is nil", raw)
		}
		alert := alertForQuery(raw, err)
		if !strings.Contains(alert.description, "regexp") {
			t.Errorf("description is '%s', want it to contain 'regexp'", alert.description)
		}
	})
}

func TestVersionContext(t *testing.T) {
	db := new(dbtesting.MockDB)

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
		// database.ReposListOptions.Names
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
			plan, err := query.Pipeline(query.InitLiteral(tc.searchQuery))
			if err != nil {
				t.Fatal(err)
			}

			resolver := searchResolver{
				db: db,
				SearchInputs: &run.SearchInputs{
					Plan:           plan,
					Query:          plan.ToParseTree(),
					VersionContext: &tc.versionContext,
					UserSettings:   &schema.Settings{},
				},
				reposMu:  &sync.Mutex{},
				resolved: &searchrepos.Resolved{},
			}

			database.Mocks.Repos.ListRepoNames = func(ctx context.Context, opts database.ReposListOptions) ([]types.RepoName, error) {
				if diff := cmp.Diff(tc.wantReposListOptionsNames, opts.Names, cmpopts.EquateEmpty()); diff != "" {
					t.Fatalf("database.RepostListOptions.Names mismatch (-want, +got):\n%s", diff)
				}
				var repos []types.RepoName
				for _, name := range tc.reposGetListNames {
					repos = append(repos, types.RepoName{Name: api.RepoName(name)})
				}
				return repos, nil
			}
			database.Mocks.Repos.Count = func(context.Context, database.ReposListOptions) (int, error) { return len(tc.reposGetListNames), nil }
			defer func() {
				database.Mocks.Repos.ListRepoNames = nil
				database.Mocks.Repos.Count = nil
			}()
			git.Mocks.ResolveRevision = func(rev string, opt git.ResolveRevisionOptions) (api.CommitID, error) {
				return api.CommitID("deadbeef"), nil
			}
			defer git.ResetMocks()

			gotResult, err := resolver.resolveRepositories(context.Background(), resolveRepositoriesOpts{})
			if err != nil {
				t.Fatal(err)
			}
			var got []string
			for _, repoRev := range gotResult.RepoRevs {
				got = append(got, string(repoRev.Repo.Name)+"@"+strings.Join(repoRev.RevSpecs(), ":"))
			}

			if diff := cmp.Diff(tc.wantResults, got, cmpopts.EquateEmpty()); diff != "" {
				t.Fatalf("mismatch (-want, +got):\n%s", diff)
			}
		})
	}
}

func mkFileMatch(repo types.RepoName, path string, lineNumbers ...int32) *result.FileMatch {
	var lines []*result.LineMatch
	for _, n := range lineNumbers {
		lines = append(lines, &result.LineMatch{LineNumber: n})
	}
	return &result.FileMatch{
		File: result.File{
			Path: path,
			Repo: repo,
		},
		LineMatches: lines,
	}
}

func repoRev(revSpec string) *search.RepositoryRevisions {
	return &search.RepositoryRevisions{
		Repo: types.RepoName{ID: api.RepoID(0), Name: "test/repo"},
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

func BenchmarkSearchResults(b *testing.B) {
	db := new(dbtesting.MockDB)

	minimalRepos, _, zoektRepos := generateRepos(5000)
	zoektFileMatches := generateZoektMatches(50)

	z := &searchbackend.Zoekt{
		Client: &searchbackend.FakeSearcher{
			Repos:  zoektRepos,
			Result: &zoekt.SearchResult{Files: zoektFileMatches},
		},
		DisableCache: true,
	}

	ctx := context.Background()

	database.Mocks.Repos.List = func(_ context.Context, op database.ReposListOptions) ([]*types.Repo, error) {
		return minimalRepos, nil
	}
	database.Mocks.Repos.Count = func(ctx context.Context, opt database.ReposListOptions) (int, error) {
		return len(minimalRepos), nil
	}
	defer func() { database.Mocks = database.MockStores{} }()

	b.ResetTimer()
	b.ReportAllocs()

	for n := 0; n < b.N; n++ {
		q, err := query.ParseLiteral(`print index:only count:350`)
		if err != nil {
			b.Fatal(err)
		}
		resolver := &searchResolver{
			db: db,
			SearchInputs: &run.SearchInputs{
				Query:        q,
				UserSettings: &schema.Settings{},
			},
			zoekt:    z,
			reposMu:  &sync.Mutex{},
			resolved: &searchrepos.Resolved{},
		}
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
	db := dbtest.NewDB(b, "")

	ctx := context.Background()

	_, repos, zoektRepos := generateRepos(5000)
	zoektFileMatches := generateZoektMatches(50)

	zoektClient, cleanup := zoektRPC(&searchbackend.FakeSearcher{
		Repos:  zoektRepos,
		Result: &zoekt.SearchResult{Files: zoektFileMatches},
	})
	defer cleanup()
	z := &searchbackend.Zoekt{
		Client:       &searchbackend.StreamSearchAdapter{zoektClient},
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

	_, err := db.ExecContext(ctx, q.Query(sqlf.PostgresBindVar), q.Args()...)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for n := 0; n < b.N; n++ {
		q, err := query.ParseLiteral(`print index:only count:350`)
		if err != nil {
			b.Fatal(err)
		}
		resolver := &searchResolver{
			db: db,
			SearchInputs: &run.SearchInputs{
				Query: q,
			},
			zoekt:    z,
			reposMu:  &sync.Mutex{},
			resolved: &searchrepos.Resolved{},
		}
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
			URI:          fmt.Sprintf("https://github.com/foobar/%s", repoWithIDs.Name),
			Description:  "this repositoriy contains a side project that I haven't maintained in 2 years",
		})

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
