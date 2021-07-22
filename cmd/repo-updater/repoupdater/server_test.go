package repoupdater

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/google/go-cmp/cmp"
	"github.com/inconshreveable/log15"
	"github.com/opentracing/opentracing-go"

	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/database/dbtest"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/auth"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/awscodecommit"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/github"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/gitlab"
	"github.com/sourcegraph/sourcegraph/internal/repos"
	"github.com/sourcegraph/sourcegraph/internal/repoupdater"
	"github.com/sourcegraph/sourcegraph/internal/repoupdater/protocol"
	"github.com/sourcegraph/sourcegraph/internal/timeutil"
	"github.com/sourcegraph/sourcegraph/internal/trace"
	"github.com/sourcegraph/sourcegraph/internal/types"
)

var dsn = flag.String("dsn", "", "Database connection string to use in integration tests")

func TestIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	t.Parallel()

	db := dbtest.NewDB(t, *dsn)

	store := repos.NewStore(db, sql.TxOptions{})

	lg := log15.New()
	lg.SetHandler(log15.DiscardHandler())
	store.Log = lg
	store.Metrics = repos.NewStoreMetrics()
	store.Tracer = trace.Tracer{Tracer: opentracing.GlobalTracer()}

	for _, tc := range []struct {
		name string
		test func(*testing.T, *repos.Store) func(*testing.T)
	}{
		{"Server/EnqueueRepoUpdate", testServerEnqueueRepoUpdate},
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

func testServerEnqueueRepoUpdate(t *testing.T, store *repos.Store) func(t *testing.T) {
	return func(t *testing.T) {
		ctx := context.Background()

		svc := types.ExternalService{
			Kind: extsvc.KindGitHub,
			Config: `{
"URL": "https://github.com",
"Token": "secret-token"
}`,
		}

		if err := store.ExternalServiceStore.Upsert(ctx, &svc); err != nil {
			t.Fatal(err)
		}

		repo := types.Repo{
			Name: "github.com/foo/bar",
			ExternalRepo: api.ExternalRepoSpec{
				ID:          "bar",
				ServiceType: extsvc.TypeGitHub,
				ServiceID:   "http://github.com",
			},
			Metadata: new(github.Repository),
		}

		if err := store.RepoStore.Create(ctx, &repo); err != nil {
			t.Fatal(err)
		}

		type testCase struct {
			name     string
			store    *repos.Store
			repo     api.RepoName
			res      *protocol.RepoUpdateResponse
			err      string
			teardown func()
		}

		var testCases []testCase
		testCases = append(testCases,
			func() testCase {
				database.Mocks.Repos.List = func(v0 context.Context, v1 database.ReposListOptions) ([]*types.Repo, error) {
					return nil, errors.New("boom")
				}
				return testCase{
					name:  "returns an error on store failure",
					store: store,
					err:   `store.list-repos: boom`,
					teardown: func() {
						database.Mocks.Repos = database.MockRepos{}
					},
				}
			}(),
			testCase{
				name:  "missing repo",
				store: store,
				repo:  "foo",
				err:   `repo foo not found with response: repo "foo" not found in store`,
			},
			func() testCase {
				repo := repo.Clone()
				return testCase{
					name:  "existing repo",
					store: store,
					repo:  repo.Name,
					res: &protocol.RepoUpdateResponse{
						ID:   repo.ID,
						Name: string(repo.Name),
					},
				}
			}(),
		)

		for _, tc := range testCases {
			tc := tc
			ctx := context.Background()

			t.Run(tc.name, func(t *testing.T) {
				if tc.teardown != nil {
					defer tc.teardown()
				}

				s := &Server{Store: tc.store, Scheduler: &fakeScheduler{}}
				srv := httptest.NewServer(s.Handler())
				defer srv.Close()
				cli := repoupdater.NewClient(srv.URL)

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

func testRepoLookup(db *sql.DB) func(t *testing.T, repoStore *repos.Store) func(t *testing.T) {
	return func(t *testing.T, store *repos.Store) func(t *testing.T) {
		return func(t *testing.T) {
			ctx := context.Background()
			clock := timeutil.NewFakeClock(time.Now(), 0)
			now := clock.Now()

			githubSource := types.ExternalService{
				Kind:         extsvc.KindGitHub,
				CloudDefault: true,
				Config:       `{}`,
			}
			awsSource := types.ExternalService{
				Kind:   extsvc.KindAWSCodeCommit,
				Config: `{}`,
			}
			gitlabSource := types.ExternalService{
				Kind:         extsvc.KindGitLab,
				CloudDefault: true,
				Config:       `{}`,
			}

			if err := store.ExternalServiceStore.Upsert(ctx, &githubSource, &awsSource, &gitlabSource); err != nil {
				t.Fatal(err)
			}

			githubRepository := &types.Repo{
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
				Sources: map[string]*types.SourceInfo{
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

			awsCodeCommitRepository := &types.Repo{
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
				Sources: map[string]*types.SourceInfo{
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

			gitlabRepository := &types.Repo{
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
				Sources: map[string]*types.SourceInfo{
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
				stored             types.Repos
				result             *protocol.RepoLookupResult
				githubDotComSource *fakeRepoSource
				gitlabDotComSource *fakeRepoSource
				assert             types.ReposAssertion
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
					stored: []*types.Repo{githubRepository},
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
					stored: []*types.Repo{awsCodeCommitRepository},
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
					stored: []*types.Repo{},
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
					assert: types.Assert.ReposEqual(githubRepository),
				},
				{
					name: "found - GitHub.com on Sourcegraph.com already exists",
					args: protocol.RepoLookupArgs{
						Repo: api.RepoName("github.com/foo/bar"),
					},
					stored: []*types.Repo{githubRepository},
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
						err: github.ErrRepoNotFound,
					},
					result: &protocol.RepoLookupResult{ErrorNotFound: true},
					err:    fmt.Sprintf("repository not found (name=%s notfound=%v)", api.RepoName("github.com/foo/bar"), true),
					assert: types.Assert.ReposEqual(),
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
					assert: types.Assert.ReposEqual(),
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
					assert: types.Assert.ReposEqual(),
				},
				{
					name: "found - gitlab.com on Sourcegraph.com",
					args: protocol.RepoLookupArgs{
						Repo: api.RepoName("gitlab.com/foo/bar"),
					},
					stored: []*types.Repo{},
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
					assert: types.Assert.ReposEqual(gitlabRepository),
				},
				{
					name: "found - gitlab.com on Sourcegraph.com already exists",
					args: protocol.RepoLookupArgs{
						Repo: api.RepoName("gitlab.com/foo/bar"),
					},
					stored: []*types.Repo{gitlabRepository},
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
						Repo: githubRepository.Name,
					},
					githubDotComSource: &fakeRepoSource{
						repo: githubRepository.With(func(r *types.Repo) {
							r.Private = true
						}),
					},
					result: &protocol.RepoLookupResult{ErrorNotFound: true},
					err:    fmt.Sprintf("repository not found (name=%s notfound=%v)", githubRepository.Name, true),
				},
				{
					name: "Private repos that used to be public should be removed asynchronously",
					args: protocol.RepoLookupArgs{
						Repo: githubRepository.Name,
					},
					githubDotComSource: &fakeRepoSource{
						err: github.ErrRepoNotFound,
					},
					stored: []*types.Repo{githubRepository},
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
					assert:      types.Assert.ReposEqual(),
				},
			}

			for _, tc := range testCases {
				tc := tc

				t.Run(tc.name, func(t *testing.T) {
					ctx := context.Background()

					rs := tc.stored.Clone()
					err := store.RepoStore.Create(ctx, rs...)
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
						err := store.RepoStore.Delete(ctx, ids...)
						if err != nil {
							t.Fatal(err)
						}
					})

					clock := clock
					syncer := &repos.Syncer{
						Now:   clock.Now,
						Store: store,
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

					cli := repoupdater.NewClient(srv.URL)

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
						rs, err := store.RepoStore.List(ctx, database.ReposListOptions{})
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
	repo *types.Repo
	err  error
}

func (s *fakeRepoSource) GetRepo(context.Context, string) (*types.Repo, error) {
	return s.repo.Clone(), s.err
}

type fakeScheduler struct{}

func (s *fakeScheduler) UpdateOnce(_ api.RepoID, _ api.RepoName) {}
func (s *fakeScheduler) ScheduleInfo(id api.RepoID) *protocol.RepoUpdateSchedulerInfoResult {
	return &protocol.RepoUpdateSchedulerInfoResult{}
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

func TestExternalServiceValidate_ValidatesToken(t *testing.T) {
	var (
		src    repos.Source
		called bool
		ctx    = context.Background()
	)
	src = testSource{
		fn: func() error {
			called = true
			return nil
		},
	}
	err := externalServiceValidate(ctx, protocol.ExternalServiceSyncRequest{}, src)
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
	if !called {
		t.Errorf("expected called, got not called")
	}
}

type testSource struct {
	fn func() error
}

var _ repos.Source = &testSource{}
var _ repos.UserSource = &testSource{}

func (t testSource) ListRepos(ctx context.Context, results chan repos.SourceResult) {
}

func (t testSource) ExternalServices() types.ExternalServices {
	return nil
}

func (t testSource) WithAuthenticator(a auth.Authenticator) (repos.Source, error) {
	return t, nil
}

func (t testSource) ValidateAuthenticator(ctx context.Context) error {
	return t.fn()
}
