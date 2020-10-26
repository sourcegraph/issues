package repoupdater

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/inconshreveable/log15"
	"github.com/opentracing/opentracing-go"

	"github.com/sourcegraph/sourcegraph/cmd/repo-updater/repos"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/db/dbtest"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/awscodecommit"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/bitbucketserver"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/github"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/gitlab"
	"github.com/sourcegraph/sourcegraph/internal/gitserver"
	"github.com/sourcegraph/sourcegraph/internal/jsonc"
	"github.com/sourcegraph/sourcegraph/internal/repoupdater"
	"github.com/sourcegraph/sourcegraph/internal/repoupdater/protocol"
	"github.com/sourcegraph/sourcegraph/internal/trace"
)

var dsn = flag.String("dsn", "", "Database connection string to use in integration tests")

func TestIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	t.Parallel()

	db := dbtest.NewDB(t, *dsn)

	dbstore := repos.NewDBStore(db, sql.TxOptions{
		Isolation: sql.LevelSerializable,
	})

	lg := log15.New()
	lg.SetHandler(log15.DiscardHandler())

	store := repos.NewObservedStore(
		dbstore,
		lg,
		repos.NewStoreMetrics(),
		trace.Tracer{Tracer: opentracing.GlobalTracer()},
	)

	for _, tc := range []struct {
		name string
		test func(*testing.T, repos.Store) func(*testing.T)
	}{
		{"Server/SetRepoEnabled", testServerSetRepoEnabled},
		{"Server/EnqueueRepoUpdate", testServerEnqueueRepoUpdate},
		{"Server/RepoExternalServices", testServerRepoExternalServices},
		{"Server/StatusMessages", testServerStatusMessages},
		{"Server/RepoLookup", testRepoLookup(db)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Cleanup(func() {
				if _, err := db.Exec(`DELETE FROM external_service_sync_jobs; DELETE FROM external_service_repos; DELETE FROM external_services; DELETE FROM repo;`); err != nil {
					t.Fatalf("cleaning up external services failed: %v", err)
				}
			})

			tc.test(t, store)(t)
		})
	}
}

func TestServer_handleRepoLookup(t *testing.T) {
	s := &Server{}

	h := ObservedHandler(
		log15.Root(),
		NewHandlerMetrics(),
		opentracing.NoopTracer{},
	)(s.Handler())

	repoLookup := func(t *testing.T, repo api.RepoName) (resp *protocol.RepoLookupResult, statusCode int) {
		t.Helper()
		rr := httptest.NewRecorder()
		body, err := json.Marshal(protocol.RepoLookupArgs{Repo: repo})
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest("GET", "/repo-lookup", bytes.NewReader(body))
		h.ServeHTTP(rr, req)
		if rr.Code == http.StatusOK {
			if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
				t.Fatal(err)
			}
		}
		return resp, rr.Code
	}
	repoLookupResult := func(t *testing.T, repo api.RepoName) protocol.RepoLookupResult {
		t.Helper()
		resp, statusCode := repoLookup(t, repo)
		if statusCode != http.StatusOK {
			t.Fatalf("http non-200 status %d", statusCode)
		}
		return *resp
	}

	t.Run("args", func(t *testing.T) {
		called := false
		mockRepoLookup = func(args protocol.RepoLookupArgs) (*protocol.RepoLookupResult, error) {
			called = true
			if want := api.RepoName("github.com/a/b"); args.Repo != want {
				t.Errorf("got owner %q, want %q", args.Repo, want)
			}
			return &protocol.RepoLookupResult{Repo: nil}, nil
		}
		defer func() { mockRepoLookup = nil }()

		repoLookupResult(t, "github.com/a/b")
		if !called {
			t.Error("!called")
		}
	})

	t.Run("not found", func(t *testing.T) {
		mockRepoLookup = func(protocol.RepoLookupArgs) (*protocol.RepoLookupResult, error) {
			return &protocol.RepoLookupResult{Repo: nil}, nil
		}
		defer func() { mockRepoLookup = nil }()

		if got, want := repoLookupResult(t, "github.com/a/b"), (protocol.RepoLookupResult{}); !reflect.DeepEqual(got, want) {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})

	t.Run("unexpected error", func(t *testing.T) {
		mockRepoLookup = func(protocol.RepoLookupArgs) (*protocol.RepoLookupResult, error) {
			return nil, errors.New("x")
		}
		defer func() { mockRepoLookup = nil }()

		result, statusCode := repoLookup(t, "github.com/a/b")
		if result != nil {
			t.Errorf("got result %+v, want nil", result)
		}
		if want := http.StatusInternalServerError; statusCode != want {
			t.Errorf("got HTTP status code %d, want %d", statusCode, want)
		}
	})

	t.Run("found", func(t *testing.T) {
		want := protocol.RepoLookupResult{
			Repo: &protocol.RepoInfo{
				ExternalRepo: api.ExternalRepoSpec{
					ID:          "a",
					ServiceType: extsvc.TypeGitHub,
					ServiceID:   "https://github.com/",
				},
				Name:        "github.com/c/d",
				Description: "b",
				Fork:        true,
			},
		}
		mockRepoLookup = func(protocol.RepoLookupArgs) (*protocol.RepoLookupResult, error) {
			return &want, nil
		}
		defer func() { mockRepoLookup = nil }()
		if got := repoLookupResult(t, "github.com/c/d"); !reflect.DeepEqual(got, want) {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})
}

func testServerSetRepoEnabled(t *testing.T, store repos.Store) func(t *testing.T) {
	return func(t *testing.T) {
		githubService := &repos.ExternalService{
			ID:          1,
			Kind:        extsvc.KindGitHub,
			DisplayName: "github.com - test",
			Config: formatJSON(`
		{
			// Some comment
			"url": "https://github.com",
			"repositoryQuery": ["none"],
			"token": "secret"
		}`),
		}

		githubRepo := (&repos.Repo{
			Name: "github.com/foo/bar",
			ExternalRepo: api.ExternalRepoSpec{
				ID:          "bar",
				ServiceType: extsvc.TypeGitHub,
				ServiceID:   "http://github.com",
			},
			Sources: map[string]*repos.SourceInfo{},
			Metadata: &github.Repository{
				ID:            "bar",
				NameWithOwner: "foo/bar",
			},
		}).With(repos.Opt.RepoSources(githubService.URN()))

		gitlabService := &repos.ExternalService{
			ID:          1,
			Kind:        extsvc.KindGitLab,
			DisplayName: "gitlab.com - test",
			Config: formatJSON(`
		{
			// Some comment
			"url": "https://gitlab.com",
			"projectQuery": ["none"],
			"token": "secret"
		}`),
		}

		gitlabRepo := (&repos.Repo{
			Name: "gitlab.com/foo/bar",
			ExternalRepo: api.ExternalRepoSpec{
				ID:          "1",
				ServiceType: extsvc.TypeGitLab,
				ServiceID:   "http://gitlab.com",
			},
			Sources: map[string]*repos.SourceInfo{},
			Metadata: &gitlab.Project{
				ProjectCommon: gitlab.ProjectCommon{
					ID:                1,
					PathWithNamespace: "foo/bar",
				},
			},
		}).With(repos.Opt.RepoSources(gitlabService.URN()))

		bitbucketServerService := &repos.ExternalService{
			ID:          1,
			Kind:        extsvc.KindBitbucketServer,
			DisplayName: "Bitbucket Server - Test",
			Config: formatJSON(`
		{
			// Some comment
			"url": "https://bitbucketserver.mycorp.com",
			"token": "secret",
			"username": "alice",
			"repositoryQuery": ["none"]
		}`),
		}

		bitbucketServerRepo := (&repos.Repo{
			Name: "bitbucketserver.mycorp.com/foo/bar",
			ExternalRepo: api.ExternalRepoSpec{
				ID:          "1",
				ServiceType: "bitbucketServer",
				ServiceID:   "http://bitbucketserver.mycorp.com",
			},
			Sources: map[string]*repos.SourceInfo{},
			Metadata: &bitbucketserver.Repo{
				ID:   1,
				Slug: "bar",
				Project: &bitbucketserver.Project{
					Key: "foo",
				},
			},
		}).With(repos.Opt.RepoSources(bitbucketServerService.URN()))

		type testCase struct {
			name  string
			svcs  repos.ExternalServices // stored services
			repos repos.Repos            // stored repos
			kind  string
			res   *protocol.ExcludeRepoResponse
			err   string
		}

		var testCases []testCase

		for _, k := range []struct {
			svc  *repos.ExternalService
			repo *repos.Repo
		}{
			{githubService, githubRepo},
			{bitbucketServerService, bitbucketServerRepo},
			{gitlabService, gitlabRepo},
		} {
			svcs := repos.ExternalServices{
				k.svc,
				k.svc.With(func(e *repos.ExternalService) {
					e.ID++
					e.DisplayName += " - Duplicate"
				}),
			}

			testCases = append(testCases, testCase{
				name:  "excluded from every external service of the same kind/" + k.svc.Kind,
				svcs:  svcs,
				repos: repos.Repos{k.repo}.With(repos.Opt.RepoSources()),
				kind:  k.svc.Kind,
				res: &protocol.ExcludeRepoResponse{
					ExternalServices: apiExternalServices(svcs.With(func(e *repos.ExternalService) {
						if err := e.Exclude(k.repo); err != nil {
							panic(err)
						}
					})...),
				},
			})
		}

		for _, tc := range testCases {
			tc := tc
			t.Run(tc.name, func(t *testing.T) {
				ctx := context.Background()

				storedSvcs := tc.svcs.Clone()
				err := store.UpsertExternalServices(ctx, storedSvcs...)
				if err != nil {
					t.Fatalf("failed to prepare store: %v", err)
				}

				storedRepos := tc.repos.Clone()
				err = store.InsertRepos(ctx, storedRepos...)
				if err != nil {
					t.Fatalf("failed to prepare store: %v", err)
				}

				s := &Server{Store: store}
				srv := httptest.NewServer(s.Handler())
				defer srv.Close()
				cli := repoupdater.Client{URL: srv.URL}

				if tc.err == "" {
					tc.err = "<nil>"
				}

				exclude := storedRepos.Filter(func(r *repos.Repo) bool {
					return strings.EqualFold(r.ExternalRepo.ServiceType, tc.kind)
				})

				if len(exclude) != 1 {
					t.Fatalf("no stored repo of kind %q", tc.kind)
				}

				res, err := cli.ExcludeRepo(ctx, exclude[0].ID)
				if have, want := fmt.Sprint(err), tc.err; have != want {
					t.Errorf("have err: %q, want: %q", have, want)
				}

				if have, want := res, tc.res; !reflect.DeepEqual(have, want) {
					// t.Logf("have: %s\nwant: %s\n", pp.Sprint(have), pp.Sprint(want))
					t.Errorf("response:\n%s", cmp.Diff(have, want))
				}

				if res == nil || len(res.ExternalServices) == 0 {
					return
				}

				ids := make([]int64, 0, len(res.ExternalServices))
				for _, s := range res.ExternalServices {
					ids = append(ids, s.ID)
				}

				svcs, err := store.ListExternalServices(ctx, repos.StoreListExternalServicesArgs{
					IDs: ids,
				})
				if err != nil {
					t.Fatalf("failed to read from store: %v", err)
				}

				have, want := apiExternalServices(svcs...), res.ExternalServices
				if !reflect.DeepEqual(have, want) {
					t.Errorf("stored external services:\n%s", cmp.Diff(have, want))
				}
			})
		}
	}
}

func testServerEnqueueRepoUpdate(t *testing.T, store repos.Store) func(t *testing.T) {
	return func(t *testing.T) {
		ctx := context.Background()

		svc := repos.ExternalService{
			Kind: extsvc.KindGitHub,
			Config: `{
"URL": "https://github.com",
"Token": "secret-token"
}`,
		}

		if err := store.UpsertExternalServices(ctx, &svc); err != nil {
			t.Fatal(err)
		}

		repo := repos.Repo{
			Name: "github.com/foo/bar",
			ExternalRepo: api.ExternalRepoSpec{
				ID:          "bar",
				ServiceType: extsvc.TypeGitHub,
				ServiceID:   "http://github.com",
			},
			Metadata: new(github.Repository),
			Sources: map[string]*repos.SourceInfo{
				svc.URN(): {
					ID:       svc.URN(),
					CloneURL: "https://secret-token@github.com/foo/bar",
				},
			},
		}

		repoWithMissingCloneURL := repos.Repo{
			Name: "github.com/foo/baz",
			ExternalRepo: api.ExternalRepoSpec{
				ID:          "missingURL",
				ServiceType: extsvc.TypeGitHub,
				ServiceID:   "http://github.com",
			},
			Metadata: new(github.Repository),
			Sources: map[string]*repos.SourceInfo{
				svc.URN(): {
					ID: svc.URN(),
				},
			},
		}

		if err := store.InsertRepos(ctx, &repo, &repoWithMissingCloneURL); err != nil {
			t.Fatal(err)
		}

		type testCase struct {
			name  string
			store repos.Store
			repo  gitserver.Repo
			res   *protocol.RepoUpdateResponse
			err   string
		}

		var testCases []testCase
		testCases = append(testCases,
			func() testCase {
				err := errors.New("boom")
				return testCase{
					name: "returns an error on store failure",
					store: &storeWithErrors{
						Store:        store,
						ListReposErr: err,
					},
					err: `store.list-repos: boom`,
				}
			}(),
			testCase{
				name:  "missing repo",
				store: store, // empty store
				repo:  gitserver.Repo{Name: "foo"},
				err:   `repo "foo" not found in store`,
			},
			func() testCase {
				repo := repoWithMissingCloneURL.Clone()

				return testCase{
					name:  "missing clone URL",
					store: store,
					repo:  gitserver.Repo{Name: api.RepoName(repo.Name)},
					res: &protocol.RepoUpdateResponse{
						ID:   repo.ID,
						Name: repo.Name,
					},
				}
			}(),
			func() testCase {
				repo := repo.Clone()
				cloneURL := "https://user:password@github.com/foo/bar"
				return testCase{
					name:  "given clone URL is preferred",
					store: store,
					repo:  gitserver.Repo{Name: api.RepoName(repo.Name), URL: cloneURL},
					res: &protocol.RepoUpdateResponse{
						ID:   repo.ID,
						Name: repo.Name,
						URL:  cloneURL,
					},
				}
			}(),
			func() testCase {
				repo := repo.Clone()
				return testCase{
					name:  "if missing, clone URL is set when stored",
					store: store,
					repo:  gitserver.Repo{Name: api.RepoName(repo.Name)},
					res: &protocol.RepoUpdateResponse{
						ID:   repo.ID,
						Name: repo.Name,
						URL:  repo.CloneURLs()[0],
					},
				}
			}(),
		)

		for _, tc := range testCases {
			tc := tc
			ctx := context.Background()

			t.Run(tc.name, func(t *testing.T) {
				s := &Server{Store: tc.store, Scheduler: &fakeScheduler{}}
				srv := httptest.NewServer(s.Handler())
				defer srv.Close()
				cli := repoupdater.Client{URL: srv.URL}

				if tc.err == "" {
					tc.err = "<nil>"
				}

				res, err := cli.EnqueueRepoUpdate(ctx, tc.repo)
				if have, want := fmt.Sprint(err), tc.err; have != want {
					t.Errorf("have err: %q, want: %q", have, want)
				}

				if have, want := res, tc.res; !reflect.DeepEqual(have, want) {
					t.Errorf("response: %s", cmp.Diff(have, want))
				}
			})
		}
	}
}

func testServerRepoExternalServices(t *testing.T, store repos.Store) func(t *testing.T) {
	return func(t *testing.T) {

		service1 := &repos.ExternalService{
			Kind:        extsvc.KindGitHub,
			DisplayName: "github.com - test",
			Config: formatJSON(`
		{
			// Some comment
			"url": "https://github.com",
			"token": "secret"
		}`),
		}

		service2 := &repos.ExternalService{
			Kind:        extsvc.KindGitHub,
			DisplayName: "github.com - test2",
			Config: formatJSON(`
		{
			// Some comment
			"url": "https://github.com",
			"token": "secret"
		}`),
		}

		// We share the store across test cases. Initialize now so we have IDs
		// set for test cases.
		ctx := context.Background()

		if err := store.UpsertExternalServices(ctx, service1, service2); err != nil {
			t.Fatal(err)
		}

		// No sources are repos that are not managed by the syncer
		repoNoSources := &repos.Repo{
			Name: "gitolite.example.com/oldschool",
			ExternalRepo: api.ExternalRepoSpec{
				ID:          "nosources",
				ServiceType: extsvc.TypeGitolite,
				ServiceID:   "http://gitolite.my.corp",
			},
		}

		repoSources := (&repos.Repo{
			Name: "github.com/foo/sources",
			ExternalRepo: api.ExternalRepoSpec{
				ID:          "sources",
				ServiceType: extsvc.TypeGitHub,
				ServiceID:   "http://github.com",
			},
			Metadata: new(github.Repository),
		}).With(repos.Opt.RepoSources(service1.URN(), service2.URN()))

		if err := store.InsertRepos(ctx, repoNoSources, repoSources); err != nil {
			t.Fatal(err)
		}

		testCases := []struct {
			name   string
			repoID api.RepoID
			svcs   []api.ExternalService
			err    string
		}{{
			name:   "repo no sources",
			repoID: repoNoSources.ID,
			svcs:   nil,
			err:    "<nil>",
		}, {
			name:   "repo sources",
			repoID: repoSources.ID,
			svcs:   apiExternalServices(service1, service2),
			err:    "<nil>",
		}, {
			name:   "repo not in store",
			repoID: 42,
			svcs:   nil,
			err:    "repository with ID 42 does not exist",
		}}

		s := &Server{Store: store}
		srv := httptest.NewServer(s.Handler())
		defer srv.Close()
		cli := repoupdater.Client{URL: srv.URL}
		for _, tc := range testCases {
			tc := tc
			t.Run(tc.name, func(t *testing.T) {
				res, err := cli.RepoExternalServices(ctx, tc.repoID)
				if have, want := fmt.Sprint(err), tc.err; have != want {
					t.Errorf("have err: %q, want: %q", have, want)
				}

				if have, want := res, tc.svcs; !reflect.DeepEqual(have, want) {
					t.Errorf("response:\n%s", cmp.Diff(have, want))
				}
			})
		}
	}
}

func testServerStatusMessages(t *testing.T, store repos.Store) func(t *testing.T) {
	return func(t *testing.T) {
		ctx := context.Background()

		githubService := &repos.ExternalService{
			ID:          1,
			Config:      `{}`,
			Kind:        extsvc.KindGitHub,
			DisplayName: "github.com - test",
		}

		err := store.UpsertExternalServices(ctx, githubService)
		if err != nil {
			t.Fatal(err)
		}

		testCases := []struct {
			name            string
			stored          repos.Repos
			gitserverCloned []string
			sourcerErr      error
			listRepoErr     error
			res             *protocol.StatusMessagesResponse
			err             string
		}{
			{
				name:            "all cloned",
				gitserverCloned: []string{"foobar"},
				stored:          []*repos.Repo{{Name: "foobar", Cloned: true}},
				res: &protocol.StatusMessagesResponse{
					Messages: []protocol.StatusMessage{},
				},
			},
			{
				name:            "nothing cloned",
				stored:          []*repos.Repo{{Name: "foobar"}},
				gitserverCloned: []string{},
				res: &protocol.StatusMessagesResponse{
					Messages: []protocol.StatusMessage{
						{
							Cloning: &protocol.CloningProgress{
								Message: "1 repositories enqueued for cloning...",
							},
						},
					},
				},
			},
			{
				name:            "subset cloned",
				stored:          []*repos.Repo{{Name: "foobar", Cloned: true}, {Name: "barfoo"}},
				gitserverCloned: []string{"foobar"},
				res: &protocol.StatusMessagesResponse{
					Messages: []protocol.StatusMessage{
						{
							Cloning: &protocol.CloningProgress{
								Message: "1 repositories enqueued for cloning...",
							},
						},
					},
				},
			},
			{
				name:            "more cloned than stored",
				stored:          []*repos.Repo{{Name: "foobar", Cloned: true}},
				gitserverCloned: []string{"foobar", "barfoo"},
				res: &protocol.StatusMessagesResponse{
					Messages: []protocol.StatusMessage{},
				},
			},
			{
				name:            "cloned different than stored",
				stored:          []*repos.Repo{{Name: "foobar"}, {Name: "barfoo"}},
				gitserverCloned: []string{"one", "two", "three"},
				res: &protocol.StatusMessagesResponse{
					Messages: []protocol.StatusMessage{
						{
							Cloning: &protocol.CloningProgress{
								Message: "2 repositories enqueued for cloning...",
							},
						},
					},
				},
			},
			{
				name:            "case insensitivity",
				gitserverCloned: []string{"foobar"},
				stored:          []*repos.Repo{{Name: "FOOBar", Cloned: true}},
				res: &protocol.StatusMessagesResponse{
					Messages: []protocol.StatusMessage{},
				},
			},
			{
				name:            "case insensitivity to gitserver names",
				gitserverCloned: []string{"FOOBar"},
				stored:          []*repos.Repo{{Name: "FOOBar", Cloned: true}},
				res: &protocol.StatusMessagesResponse{
					Messages: []protocol.StatusMessage{},
				},
			},
			{
				name:       "one external service syncer err",
				sourcerErr: errors.New("github is down"),
				res: &protocol.StatusMessagesResponse{
					Messages: []protocol.StatusMessage{
						{
							ExternalServiceSyncError: &protocol.ExternalServiceSyncError{
								Message:           "github is down",
								ExternalServiceId: githubService.ID,
							},
						},
					},
				},
			},
			{
				name:        "one syncer err",
				listRepoErr: errors.New("could not connect to database"),
				res: &protocol.StatusMessagesResponse{
					Messages: []protocol.StatusMessage{
						{
							SyncError: &protocol.SyncError{
								Message: "syncer.sync.store.list-repos: could not connect to database",
							},
						},
					},
				},
			},
		}

		for _, tc := range testCases {
			tc := tc
			ctx := context.Background()

			t.Run(tc.name, func(t *testing.T) {
				gitserverClient := &fakeGitserverClient{listClonedResponse: tc.gitserverCloned}

				stored := tc.stored.Clone()
				var cloned []string
				for _, r := range stored {
					r.ExternalRepo = api.ExternalRepoSpec{
						ID:          uuid.New().String(),
						ServiceType: extsvc.TypeGitHub,
						ServiceID:   "https://github.com/",
					}
					if r.Cloned {
						cloned = append(cloned, r.Name)
					}
				}

				err := store.InsertRepos(ctx, stored...)
				if err != nil {
					t.Fatal(err)
				}

				t.Cleanup(func() {
					ids := make([]api.RepoID, 0, len(stored))
					for _, r := range stored {
						ids = append(ids, r.ID)
					}
					err := store.DeleteRepos(ctx, ids...)
					if err != nil {
						t.Fatal(err)
					}
				})

				err = store.SetClonedRepos(ctx, cloned...)
				if err != nil {
					t.Fatal(err)
				}

				clock := repos.NewFakeClock(time.Now(), 0)
				syncer := &repos.Syncer{
					Now: clock.Now,
				}

				if tc.sourcerErr != nil || tc.listRepoErr != nil {
					store = &storeWithErrors{
						Store:        store,
						ListReposErr: tc.listRepoErr,
					}
					sourcer := repos.NewFakeSourcer(tc.sourcerErr, repos.NewFakeSource(githubService, nil))
					// Run Sync so that possibly `LastSyncErrors` is set
					syncer.Sourcer = sourcer
					_ = syncer.SyncExternalService(ctx, store, githubService.ID, time.Millisecond)
				}

				s := &Server{
					Syncer:          syncer,
					Store:           store,
					GitserverClient: gitserverClient,
				}

				srv := httptest.NewServer(s.Handler())
				defer srv.Close()
				cli := repoupdater.Client{URL: srv.URL}

				if tc.err == "" {
					tc.err = "<nil>"
				}

				res, err := cli.StatusMessages(ctx)
				if have, want := fmt.Sprint(err), tc.err; have != want {
					t.Errorf("have err: %q, want: %q", have, want)
				}

				if have, want := res, tc.res; !reflect.DeepEqual(have, want) {
					t.Errorf("response: %s", cmp.Diff(have, want))
				}
			})
		}
	}
}

func apiExternalServices(es ...*repos.ExternalService) []api.ExternalService {
	if len(es) == 0 {
		return nil
	}

	svcs := make([]api.ExternalService, 0, len(es))
	for _, e := range es {
		svc := api.ExternalService{
			ID:          e.ID,
			Kind:        e.Kind,
			DisplayName: e.DisplayName,
			Config:      e.Config,
			CreatedAt:   e.CreatedAt,
			UpdatedAt:   e.UpdatedAt,
		}

		if e.IsDeleted() {
			svc.DeletedAt = &e.DeletedAt
		}

		svcs = append(svcs, svc)
	}

	return svcs
}

func testRepoLookup(db *sql.DB) func(t *testing.T, repoStore repos.Store) func(t *testing.T) {
	return func(t *testing.T, store repos.Store) func(t *testing.T) {
		return func(t *testing.T) {
			ctx := context.Background()
			clock := repos.NewFakeClock(time.Now(), 0)
			now := clock.Now()

			githubSource := repos.ExternalService{
				Kind:   extsvc.KindGitHub,
				Config: `{}`,
			}
			awsSource := repos.ExternalService{
				Kind:   extsvc.KindAWSCodeCommit,
				Config: `{}`,
			}
			gitlabSource := repos.ExternalService{
				Kind:   extsvc.KindGitLab,
				Config: `{}`,
			}

			if err := store.UpsertExternalServices(ctx, &githubSource, &awsSource, &gitlabSource); err != nil {
				t.Fatal(err)
			}

			githubRepository := &repos.Repo{
				Name:        "github.com/foo/bar",
				Description: "The description",
				Archived:    false,
				Fork:        false,
				CreatedAt:   now,
				UpdatedAt:   now,
				ExternalRepo: api.ExternalRepoSpec{
					ID:          "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==",
					ServiceType: extsvc.TypeGitHub,
					ServiceID:   "https://github.com/",
				},
				Sources: map[string]*repos.SourceInfo{
					githubSource.URN(): {
						ID:       githubSource.URN(),
						CloneURL: "git@github.com:foo/bar.git",
					},
				},
				Metadata: &github.Repository{
					ID:            "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==",
					URL:           "github.com/foo/bar",
					DatabaseID:    1234,
					Description:   "The description",
					NameWithOwner: "foo/bar",
				},
			}

			awsCodeCommitRepository := &repos.Repo{
				Name:        "git-codecommit.us-west-1.amazonaws.com/stripe-go",
				Description: "The stripe-go lib",
				Archived:    false,
				Fork:        false,
				CreatedAt:   now,
				ExternalRepo: api.ExternalRepoSpec{
					ID:          "f001337a-3450-46fd-b7d2-650c0EXAMPLE",
					ServiceType: extsvc.TypeAWSCodeCommit,
					ServiceID:   "arn:aws:codecommit:us-west-1:999999999999:",
				},
				Sources: map[string]*repos.SourceInfo{
					awsSource.URN(): {
						ID:       awsSource.URN(),
						CloneURL: "git@git-codecommit.us-west-1.amazonaws.com/v1/repos/stripe-go",
					},
				},
				Metadata: &awscodecommit.Repository{
					ARN:          "arn:aws:codecommit:us-west-1:999999999999:stripe-go",
					AccountID:    "999999999999",
					ID:           "f001337a-3450-46fd-b7d2-650c0EXAMPLE",
					Name:         "stripe-go",
					Description:  "The stripe-go lib",
					HTTPCloneURL: "https://git-codecommit.us-west-1.amazonaws.com/v1/repos/stripe-go",
					LastModified: &now,
				},
			}

			gitlabRepository := &repos.Repo{
				Name:        "gitlab.com/gitlab-org/gitaly",
				Description: "Gitaly is a Git RPC service for handling all the git calls made by GitLab",
				URI:         "gitlab.com/gitlab-org/gitaly",
				CreatedAt:   now,
				UpdatedAt:   now,
				ExternalRepo: api.ExternalRepoSpec{
					ID:          "2009901",
					ServiceType: extsvc.TypeGitLab,
					ServiceID:   "https://gitlab.com/",
				},
				Sources: map[string]*repos.SourceInfo{
					gitlabSource.URN(): {
						ID:       gitlabSource.URN(),
						CloneURL: "https://gitlab.com/gitlab-org/gitaly.git",
					},
				},
				Metadata: &gitlab.Project{
					ProjectCommon: gitlab.ProjectCommon{
						ID:                2009901,
						PathWithNamespace: "gitlab-org/gitaly",
						Description:       "Gitaly is a Git RPC service for handling all the git calls made by GitLab",
						WebURL:            "https://gitlab.com/gitlab-org/gitaly",
						HTTPURLToRepo:     "https://gitlab.com/gitlab-org/gitaly.git",
						SSHURLToRepo:      "git@gitlab.com:gitlab-org/gitaly.git",
					},
					Visibility: "",
					Archived:   false,
				},
			}

			testCases := []struct {
				name               string
				args               protocol.RepoLookupArgs
				stored             repos.Repos
				result             *protocol.RepoLookupResult
				githubDotComSource *fakeRepoSource
				gitlabDotComSource *fakeRepoSource
				assert             repos.ReposAssertion
				assertDelay        time.Duration
				err                string
			}{
				{
					name: "not found",
					args: protocol.RepoLookupArgs{
						Repo: api.RepoName("github.com/a/b"),
					},
					result: &protocol.RepoLookupResult{ErrorNotFound: true},
					err:    fmt.Sprintf("repository not found (name=%s notfound=%v)", api.RepoName("github.com/a/b"), true),
				},
				{
					name: "found - GitHub",
					args: protocol.RepoLookupArgs{
						Repo: api.RepoName("github.com/foo/bar"),
					},
					stored: []*repos.Repo{githubRepository},
					result: &protocol.RepoLookupResult{Repo: &protocol.RepoInfo{
						ExternalRepo: api.ExternalRepoSpec{
							ID:          "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==",
							ServiceType: extsvc.TypeGitHub,
							ServiceID:   "https://github.com/",
						},
						Name:        "github.com/foo/bar",
						Description: "The description",
						VCS:         protocol.VCSInfo{URL: "git@github.com:foo/bar.git"},
						Links: &protocol.RepoLinks{
							Root:   "github.com/foo/bar",
							Tree:   "github.com/foo/bar/tree/{rev}/{path}",
							Blob:   "github.com/foo/bar/blob/{rev}/{path}",
							Commit: "github.com/foo/bar/commit/{commit}",
						},
					}},
				},
				{
					name: "found - AWS CodeCommit",
					args: protocol.RepoLookupArgs{
						Repo: api.RepoName("git-codecommit.us-west-1.amazonaws.com/stripe-go"),
					},
					stored: []*repos.Repo{awsCodeCommitRepository},
					result: &protocol.RepoLookupResult{Repo: &protocol.RepoInfo{
						ExternalRepo: api.ExternalRepoSpec{
							ID:          "f001337a-3450-46fd-b7d2-650c0EXAMPLE",
							ServiceType: extsvc.TypeAWSCodeCommit,
							ServiceID:   "arn:aws:codecommit:us-west-1:999999999999:",
						},
						Name:        "git-codecommit.us-west-1.amazonaws.com/stripe-go",
						Description: "The stripe-go lib",
						VCS:         protocol.VCSInfo{URL: "git@git-codecommit.us-west-1.amazonaws.com/v1/repos/stripe-go"},
						Links: &protocol.RepoLinks{
							Root:   "https://us-west-1.console.aws.amazon.com/codesuite/codecommit/repositories/stripe-go/browse",
							Tree:   "https://us-west-1.console.aws.amazon.com/codesuite/codecommit/repositories/stripe-go/browse/{rev}/--/{path}",
							Blob:   "https://us-west-1.console.aws.amazon.com/codesuite/codecommit/repositories/stripe-go/browse/{rev}/--/{path}",
							Commit: "https://us-west-1.console.aws.amazon.com/codesuite/codecommit/repositories/stripe-go/commit/{commit}",
						},
					}},
				},
				{
					name: "found - GitHub.com on Sourcegraph.com",
					args: protocol.RepoLookupArgs{
						Repo: api.RepoName("github.com/foo/bar"),
					},
					stored: []*repos.Repo{},
					githubDotComSource: &fakeRepoSource{
						repo: githubRepository,
					},
					result: &protocol.RepoLookupResult{Repo: &protocol.RepoInfo{
						ExternalRepo: api.ExternalRepoSpec{
							ID:          "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==",
							ServiceType: extsvc.TypeGitHub,
							ServiceID:   "https://github.com/",
						},
						Name:        "github.com/foo/bar",
						Description: "The description",
						VCS:         protocol.VCSInfo{URL: "git@github.com:foo/bar.git"},
						Links: &protocol.RepoLinks{
							Root:   "github.com/foo/bar",
							Tree:   "github.com/foo/bar/tree/{rev}/{path}",
							Blob:   "github.com/foo/bar/blob/{rev}/{path}",
							Commit: "github.com/foo/bar/commit/{commit}",
						},
					}},
					assert: repos.Assert.ReposEqual(githubRepository),
				},
				{
					name: "found - GitHub.com on Sourcegraph.com already exists",
					args: protocol.RepoLookupArgs{
						Repo: api.RepoName("github.com/foo/bar"),
					},
					stored: []*repos.Repo{githubRepository},
					githubDotComSource: &fakeRepoSource{
						repo: githubRepository,
					},
					result: &protocol.RepoLookupResult{Repo: &protocol.RepoInfo{
						ExternalRepo: api.ExternalRepoSpec{
							ID:          "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==",
							ServiceType: extsvc.TypeGitHub,
							ServiceID:   "https://github.com/",
						},
						Name:        "github.com/foo/bar",
						Description: "The description",
						VCS:         protocol.VCSInfo{URL: "git@github.com:foo/bar.git"},
						Links: &protocol.RepoLinks{
							Root:   "github.com/foo/bar",
							Tree:   "github.com/foo/bar/tree/{rev}/{path}",
							Blob:   "github.com/foo/bar/blob/{rev}/{path}",
							Commit: "github.com/foo/bar/commit/{commit}",
						},
					}},
				},
				{
					name: "not found - GitHub.com on Sourcegraph.com",
					args: protocol.RepoLookupArgs{
						Repo: api.RepoName("github.com/foo/bar"),
					},
					githubDotComSource: &fakeRepoSource{
						err: github.ErrNotFound,
					},
					result: &protocol.RepoLookupResult{ErrorNotFound: true},
					err:    fmt.Sprintf("repository not found (name=%s notfound=%v)", api.RepoName("github.com/foo/bar"), true),
					assert: repos.Assert.ReposEqual(),
				},
				{
					name: "unauthorized - GitHub.com on Sourcegraph.com",
					args: protocol.RepoLookupArgs{
						Repo: api.RepoName("github.com/foo/bar"),
					},
					githubDotComSource: &fakeRepoSource{
						err: &github.APIError{Code: http.StatusUnauthorized},
					},
					result: &protocol.RepoLookupResult{ErrorUnauthorized: true},
					err:    fmt.Sprintf("not authorized (name=%s noauthz=%v)", api.RepoName("github.com/foo/bar"), true),
					assert: repos.Assert.ReposEqual(),
				},
				{
					name: "temporarily unavailable - GitHub.com on Sourcegraph.com",
					args: protocol.RepoLookupArgs{
						Repo: api.RepoName("github.com/foo/bar"),
					},
					githubDotComSource: &fakeRepoSource{
						err: &github.APIError{Message: "API rate limit exceeded"},
					},
					result: &protocol.RepoLookupResult{ErrorTemporarilyUnavailable: true},
					err:    fmt.Sprintf("repository temporarily unavailable (name=%s istemporary=%v)", api.RepoName("github.com/foo/bar"), true),
					assert: repos.Assert.ReposEqual(),
				},
				{
					name: "found - gitlab.com on Sourcegraph.com",
					args: protocol.RepoLookupArgs{
						Repo: api.RepoName("gitlab.com/foo/bar"),
					},
					stored: []*repos.Repo{},
					gitlabDotComSource: &fakeRepoSource{
						repo: gitlabRepository,
					},
					result: &protocol.RepoLookupResult{Repo: &protocol.RepoInfo{
						Name:        "gitlab.com/gitlab-org/gitaly",
						Description: "Gitaly is a Git RPC service for handling all the git calls made by GitLab",
						Fork:        false,
						Archived:    false,
						VCS: protocol.VCSInfo{
							URL: "https://gitlab.com/gitlab-org/gitaly.git",
						},
						Links: &protocol.RepoLinks{
							Root:   "https://gitlab.com/gitlab-org/gitaly",
							Tree:   "https://gitlab.com/gitlab-org/gitaly/tree/{rev}/{path}",
							Blob:   "https://gitlab.com/gitlab-org/gitaly/blob/{rev}/{path}",
							Commit: "https://gitlab.com/gitlab-org/gitaly/commit/{commit}",
						},
						ExternalRepo: gitlabRepository.ExternalRepo,
					}},
					assert: repos.Assert.ReposEqual(gitlabRepository),
				},
				{
					name: "found - gitlab.com on Sourcegraph.com already exists",
					args: protocol.RepoLookupArgs{
						Repo: api.RepoName("gitlab.com/foo/bar"),
					},
					stored: []*repos.Repo{gitlabRepository},
					gitlabDotComSource: &fakeRepoSource{
						repo: gitlabRepository,
					},
					result: &protocol.RepoLookupResult{Repo: &protocol.RepoInfo{
						Name:        "gitlab.com/gitlab-org/gitaly",
						Description: "Gitaly is a Git RPC service for handling all the git calls made by GitLab",
						Fork:        false,
						Archived:    false,
						VCS: protocol.VCSInfo{
							URL: "https://gitlab.com/gitlab-org/gitaly.git",
						},
						Links: &protocol.RepoLinks{
							Root:   "https://gitlab.com/gitlab-org/gitaly",
							Tree:   "https://gitlab.com/gitlab-org/gitaly/tree/{rev}/{path}",
							Blob:   "https://gitlab.com/gitlab-org/gitaly/blob/{rev}/{path}",
							Commit: "https://gitlab.com/gitlab-org/gitaly/commit/{commit}",
						},
						ExternalRepo: gitlabRepository.ExternalRepo,
					}},
				},
				{
					name: "GithubDotcomSource on Sourcegraph.com ignores non-Github.com repos",
					args: protocol.RepoLookupArgs{
						Repo: api.RepoName("git-codecommit.us-west-1.amazonaws.com/stripe-go"),
					},
					githubDotComSource: &fakeRepoSource{
						repo: githubRepository,
					},
					result: &protocol.RepoLookupResult{ErrorNotFound: true},
					err:    fmt.Sprintf("repository not found (name=%s notfound=%v)", api.RepoName("git-codecommit.us-west-1.amazonaws.com/stripe-go"), true),
				},
				{
					name: "Private repos are not supported on sourcegraph.com",
					args: protocol.RepoLookupArgs{
						Repo: api.RepoName(githubRepository.Name),
					},
					githubDotComSource: &fakeRepoSource{
						repo: githubRepository.With(func(r *repos.Repo) {
							r.Private = true
						}),
					},
					result: &protocol.RepoLookupResult{ErrorNotFound: true},
					err:    fmt.Sprintf("repository not found (name=%s notfound=%v)", api.RepoName(githubRepository.Name), true),
				},
				{
					name: "Private repos that used to be public should be removed asynchronously",
					args: protocol.RepoLookupArgs{
						Repo: api.RepoName(githubRepository.Name),
					},
					githubDotComSource: &fakeRepoSource{
						err: github.ErrNotFound,
					},
					stored: []*repos.Repo{githubRepository},
					result: &protocol.RepoLookupResult{Repo: &protocol.RepoInfo{
						ExternalRepo: api.ExternalRepoSpec{
							ID:          "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==",
							ServiceType: extsvc.TypeGitHub,
							ServiceID:   "https://github.com/",
						},
						Name:        "github.com/foo/bar",
						Description: "The description",
						VCS:         protocol.VCSInfo{URL: "git@github.com:foo/bar.git"},
						Links: &protocol.RepoLinks{
							Root:   "github.com/foo/bar",
							Tree:   "github.com/foo/bar/tree/{rev}/{path}",
							Blob:   "github.com/foo/bar/blob/{rev}/{path}",
							Commit: "github.com/foo/bar/commit/{commit}",
						},
					}},
					assertDelay: time.Second,
					assert:      repos.Assert.ReposEqual(),
				},
			}

			for _, tc := range testCases {
				tc := tc

				t.Run(tc.name, func(t *testing.T) {
					ctx := context.Background()

					rs := tc.stored.Clone()
					err := store.InsertRepos(ctx, rs...)
					if err != nil {
						t.Fatal(err)
					}
					t.Cleanup(func() {
						_, err := db.ExecContext(ctx, "DELETE FROM repo")
						if err != nil {
							t.Fatal(err)
						}
					})

					t.Cleanup(func() {
						ids := make([]api.RepoID, 0, len(tc.stored))
						for _, r := range tc.stored {
							ids = append(ids, r.ID)
						}
						err := store.DeleteRepos(ctx, ids...)
						if err != nil {
							t.Fatal(err)
						}
					})

					clock := clock
					syncer := &repos.Syncer{
						Now: clock.Now,
					}
					s := &Server{Syncer: syncer, Store: store}
					if tc.githubDotComSource != nil {
						s.SourcegraphDotComMode = true
						s.GithubDotComSource = tc.githubDotComSource
					}

					if tc.gitlabDotComSource != nil {
						s.SourcegraphDotComMode = true
						s.GitLabDotComSource = tc.gitlabDotComSource
					}

					srv := httptest.NewServer(s.Handler())
					defer srv.Close()

					cli := repoupdater.Client{URL: srv.URL}

					if tc.err == "" {
						tc.err = "<nil>"
					}

					res, err := cli.RepoLookup(ctx, tc.args)
					if have, want := fmt.Sprint(err), tc.err; have != want {
						t.Errorf("have err: %q, want: %q", have, want)
					}

					if have, want := res, tc.result; !reflect.DeepEqual(have, want) {
						t.Errorf("response: %s", cmp.Diff(have, want))
					}

					if diff := cmp.Diff(res, tc.result); diff != "" {
						t.Fatalf("RepoLookup:\n%s", diff)
					}

					if tc.assert != nil {
						if tc.assertDelay != 0 {
							time.Sleep(tc.assertDelay)
						}
						rs, err := store.ListRepos(ctx, repos.StoreListReposArgs{})
						if err != nil {
							t.Fatal(err)
						}
						tc.assert(t, rs)
					}
				})
			}
		}
	}
}

type fakeRepoSource struct {
	repo *repos.Repo
	err  error
}

func (s *fakeRepoSource) GetRepo(context.Context, string) (*repos.Repo, error) {
	return s.repo.Clone(), s.err
}

type fakeScheduler struct{}

func (s *fakeScheduler) UpdateOnce(_ api.RepoID, _ api.RepoName, _ string) {}
func (s *fakeScheduler) ScheduleInfo(id api.RepoID) *protocol.RepoUpdateSchedulerInfoResult {
	return &protocol.RepoUpdateSchedulerInfoResult{}
}

type fakeGitserverClient struct {
	listClonedResponse []string
}

func (g *fakeGitserverClient) ListCloned(ctx context.Context) ([]string, error) {
	return g.listClonedResponse, nil
}

type fakePermsSyncer struct{}

func (*fakePermsSyncer) ScheduleUsers(ctx context.Context, userIDs ...int32) {
}

func (*fakePermsSyncer) ScheduleRepos(ctx context.Context, repoIDs ...api.RepoID) {
}

func TestServer_handleSchedulePermsSync(t *testing.T) {
	tests := []struct {
		name           string
		permsSyncer    *fakePermsSyncer
		body           string
		wantStatusCode int
		wantBody       string
	}{
		{
			name:           "PermsSyncer not available",
			wantStatusCode: http.StatusForbidden,
			wantBody:       "null",
		},
		{
			name:           "bad JSON",
			permsSyncer:    &fakePermsSyncer{},
			body:           "{",
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "unexpected EOF",
		},
		{
			name:           "missing ids",
			permsSyncer:    &fakePermsSyncer{},
			body:           "{}",
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "neither user and repo ids provided",
		},

		{
			name:           "successful call with user IDs",
			permsSyncer:    &fakePermsSyncer{},
			body:           `{"user_ids": [1]}`,
			wantStatusCode: http.StatusOK,
			wantBody:       "null",
		},
		{
			name:           "successful call with repo IDs",
			permsSyncer:    &fakePermsSyncer{},
			body:           `{"repo_ids":[1]}`,
			wantStatusCode: http.StatusOK,
			wantBody:       "null",
		},
		{
			name:           "successful call with both IDs",
			permsSyncer:    &fakePermsSyncer{},
			body:           `{"user_ids": [1], "repo_ids":[1]}`,
			wantStatusCode: http.StatusOK,
			wantBody:       "null",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := httptest.NewRequest("POST", "/schedule-perms-sync", strings.NewReader(test.body))
			w := httptest.NewRecorder()

			s := &Server{}
			// NOTE: An interface has nil value is not a nil interface,
			// so should only assign to the interface when the value is not nil.
			if test.permsSyncer != nil {
				s.PermsSyncer = test.permsSyncer
			}
			s.handleSchedulePermsSync(w, r)

			if w.Code != test.wantStatusCode {
				t.Fatalf("Code: want %v but got %v", test.wantStatusCode, w.Code)
			} else if diff := cmp.Diff(test.wantBody, w.Body.String()); diff != "" {
				t.Fatalf("Body mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func formatJSON(s string) string {
	formatted, err := jsonc.Format(s, nil)
	if err != nil {
		panic(err)
	}
	return formatted
}

type storeWithErrors struct {
	repos.Store

	ListReposErr   error
	UpsertReposErr error
}

func (s *storeWithErrors) ListRepos(ctx context.Context, args repos.StoreListReposArgs) ([]*repos.Repo, error) {
	if s.ListReposErr != nil {
		return nil, s.ListReposErr
	}
	return s.Store.ListRepos(ctx, args)
}

func (s *storeWithErrors) UpsertRepos(ctx context.Context, repos ...*repos.Repo) error {
	if s.UpsertReposErr != nil {
		return s.UpsertReposErr
	}
	return s.Store.UpsertRepos(ctx, repos...)
}
