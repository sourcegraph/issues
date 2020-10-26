package repos

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/dnaeon/go-vcr/cassette"
	"github.com/dnaeon/go-vcr/recorder"
	"github.com/google/go-cmp/cmp"
	"github.com/inconshreveable/log15"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/conf/conftypes"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/phabricator"
	"github.com/sourcegraph/sourcegraph/internal/httpcli"
	"github.com/sourcegraph/sourcegraph/internal/httptestutil"
	"github.com/sourcegraph/sourcegraph/schema"
)

func TestNewSourcer(t *testing.T) {
	now := time.Now()

	github := ExternalService{
		Kind:        extsvc.KindGitHub,
		DisplayName: "Github - Test",
		Config:      `{"url": "https://github.com"}`,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	gitlab := ExternalService{
		Kind:        extsvc.KindGitHub,
		DisplayName: "Github - Test",
		Config:      `{"url": "https://github.com"}`,
		CreatedAt:   now,
		UpdatedAt:   now,
		DeletedAt:   now,
	}

	sources := func(es ...*ExternalService) (srcs []Source) {
		t.Helper()

		for _, e := range es {
			src, err := NewSource(e, nil)
			if err != nil {
				t.Fatal(err)
			}
			srcs = append(srcs, src)
		}

		return srcs
	}

	for _, tc := range []struct {
		name string
		svcs ExternalServices
		srcs Sources
		err  string
	}{
		{
			name: "deleted external services are excluded",
			svcs: ExternalServices{&github, &gitlab},
			srcs: sources(&github),
			err:  "<nil>",
		},
	} {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			srcs, err := NewSourcer(nil)(tc.svcs...)
			if have, want := fmt.Sprint(err), tc.err; have != want {
				t.Errorf("error:\nhave: %q\nwant: %q", have, want)
			}

			have := srcs.ExternalServices()
			want := tc.srcs.ExternalServices()

			if !reflect.DeepEqual(have, want) {
				t.Errorf("sources:\n%s", cmp.Diff(have, want))
			}
		})
	}
}

func TestSources_ListRepos(t *testing.T) {
	conf.Mock(&conf.Unified{
		ServiceConnections: conftypes.ServiceConnections{
			GitServers: []string{"127.0.0.1:3178"},
		},
	})
	defer conf.Mock(nil)

	type testCase struct {
		name   string
		ctx    context.Context
		svcs   ExternalServices
		assert func(*ExternalService) ReposAssertion
		err    string
	}

	var testCases []testCase

	{
		svcs := ExternalServices{
			{
				Kind: extsvc.KindGitHub,
				Config: marshalJSON(t, &schema.GitHubConnection{
					Url:   "https://github.com",
					Token: os.Getenv("GITHUB_ACCESS_TOKEN"),
					RepositoryQuery: []string{
						"user:tsenart in:name patrol",
					},
					Repos: []string{
						"sourcegraph/Sourcegraph",
						"keegancsmith/sqlf",
						"tsenart/VEGETA",
						"tsenart/go-tsz", // fork
					},
					Exclude: []*schema.ExcludedGitHubRepo{
						{Name: "tsenart/Vegeta"},
						{Id: "MDEwOlJlcG9zaXRvcnkxNTM2NTcyNDU="}, // tsenart/patrol ID
						{Pattern: "^keegancsmith/.*"},
						{Forks: true},
					},
				}),
			},
			{
				Kind: extsvc.KindGitLab,
				Config: marshalJSON(t, &schema.GitLabConnection{
					Url:   "https://gitlab.com",
					Token: os.Getenv("GITLAB_ACCESS_TOKEN"),
					ProjectQuery: []string{
						"?search=gokulkarthick",
						"?search=dotfiles-vegetableman",
					},
					Exclude: []*schema.ExcludedGitLabProject{
						{Name: "gokulkarthick/gokulkarthick"},
						{Id: 7789240},
					},
				}),
			},
			{
				Kind: extsvc.KindBitbucketServer,
				Config: marshalJSON(t, &schema.BitbucketServerConnection{
					Url:   "https://bitbucket.sgdev.org",
					Token: os.Getenv("BITBUCKET_SERVER_TOKEN"),
					Repos: []string{
						"SOUR/vegeta",
						"sour/sourcegraph",
					},
					RepositoryQuery: []string{
						"?visibility=private",
					},
					Exclude: []*schema.ExcludedBitbucketServerRepo{
						{Name: "SOUR/Vegeta"},      // test case insensitivity
						{Id: 10067},                // sourcegraph repo id
						{Pattern: ".*/automation"}, // only matches automation-testing repo
					},
				}),
			},
			{
				Kind: extsvc.KindAWSCodeCommit,
				Config: marshalJSON(t, &schema.AWSCodeCommitConnection{
					AccessKeyID:     getAWSEnv("AWS_ACCESS_KEY_ID"),
					SecretAccessKey: getAWSEnv("AWS_SECRET_ACCESS_KEY"),
					Region:          "us-west-1",
					GitCredentials: schema.AWSCodeCommitGitCredentials{
						Username: "git-username",
						Password: "git-password",
					},
					Exclude: []*schema.ExcludedAWSCodeCommitRepo{
						{Name: "stRIPE-gO"},
						{Id: "020a4751-0f46-4e19-82bf-07d0989b67dd"},                // ID of `test`
						{Name: "test2", Id: "2686d63d-bff4-4a3e-a94f-3e6df904238d"}, // ID of `test2`
					},
				}),
			},
			{
				Kind: extsvc.KindGitolite,
				Config: marshalJSON(t, &schema.GitoliteConnection{
					Prefix: "gitolite.mycorp.com/",
					Host:   "ssh://git@127.0.0.1:2222",
					Exclude: []*schema.ExcludedGitoliteRepo{
						{Name: "bar"},
					},
				}),
			},
		}

		testCases = append(testCases, testCase{
			name: "excluded repos are never yielded",
			svcs: svcs,
			assert: func(s *ExternalService) ReposAssertion {
				return func(t testing.TB, rs Repos) {
					t.Helper()

					set := make(map[string]bool)
					var patterns []*regexp.Regexp

					c, err := s.Configuration()
					if err != nil {
						t.Fatal(err)
					}

					type excluded struct {
						name, id, pattern string
					}

					var ex []excluded
					switch cfg := c.(type) {
					case *schema.GitHubConnection:
						for _, e := range cfg.Exclude {
							ex = append(ex, excluded{name: e.Name, id: e.Id, pattern: e.Pattern})
						}
					case *schema.GitLabConnection:
						for _, e := range cfg.Exclude {
							ex = append(ex, excluded{name: e.Name, id: strconv.Itoa(e.Id)})
						}
					case *schema.BitbucketServerConnection:
						for _, e := range cfg.Exclude {
							ex = append(ex, excluded{name: e.Name, id: strconv.Itoa(e.Id), pattern: e.Pattern})
						}
					case *schema.AWSCodeCommitConnection:
						for _, e := range cfg.Exclude {
							ex = append(ex, excluded{name: e.Name, id: e.Id})
						}
					case *schema.GitoliteConnection:
						for _, e := range cfg.Exclude {
							ex = append(ex, excluded{name: e.Name, pattern: e.Pattern})
						}
					}

					if len(ex) == 0 {
						t.Fatal("exclude list must not be empty")
					}

					for _, e := range ex {
						name := e.name
						switch s.Kind {
						case extsvc.KindGitHub, extsvc.KindBitbucketServer:
							name = strings.ToLower(name)
						}
						set[name], set[e.id] = true, true
						if e.pattern != "" {
							re, err := regexp.Compile(e.pattern)
							if err != nil {
								t.Fatal(err)
							}
							patterns = append(patterns, re)
						}
					}

					for _, r := range rs {
						if r.Fork {
							t.Errorf("excluded fork was yielded: %s", r.Name)
						}

						if set[r.Name] || set[r.ExternalRepo.ID] {
							t.Errorf("excluded repo{name=%s, id=%s} was yielded", r.Name, r.ExternalRepo.ID)
						}

						for _, re := range patterns {
							if re.MatchString(r.Name) {
								t.Errorf("excluded repo{name=%s} matching %q was yielded", r.Name, re.String())
							}
						}
					}
				}
			},
			err: "<nil>",
		})
	}

	{
		svcs := ExternalServices{
			{
				Kind: extsvc.KindGitHub,
				Config: marshalJSON(t, &schema.GitHubConnection{
					Url:   "https://github.com",
					Token: os.Getenv("GITHUB_ACCESS_TOKEN"),
					Repos: []string{
						"sourcegraph/Sourcegraph",
						"tsenart/Vegeta",
						"tsenart/vegeta-missing",
					},
				}),
			},
			{
				Kind: extsvc.KindGitLab,
				Config: marshalJSON(t, &schema.GitLabConnection{
					Url:          "https://gitlab.com",
					Token:        os.Getenv("GITLAB_ACCESS_TOKEN"),
					ProjectQuery: []string{"none"},
					Projects: []*schema.GitLabProject{
						{Name: "gnachman/iterm2"},
						{Name: "gnachman/iterm2-missing"},
						{Id: 13083}, // https://gitlab.com/gitlab-org/gitlab-ce
					},
				}),
			},
			{
				Kind: extsvc.KindBitbucketServer,
				Config: marshalJSON(t, &schema.BitbucketServerConnection{
					Url:             "https://bitbucket.sgdev.org",
					Token:           os.Getenv("BITBUCKET_SERVER_TOKEN"),
					RepositoryQuery: []string{"none"},
					Repos: []string{
						"Sour/vegetA",
						"sour/sourcegraph",
					},
				}),
			},
			{
				Kind: extsvc.KindOther,
				Config: marshalJSON(t, &schema.OtherExternalServiceConnection{
					Url: "https://github.com",
					Repos: []string{
						"google/go-cmp",
					},
				}),
			},
			{
				Kind: extsvc.KindAWSCodeCommit,
				Config: marshalJSON(t, &schema.AWSCodeCommitConnection{
					AccessKeyID:     getAWSEnv("AWS_ACCESS_KEY_ID"),
					SecretAccessKey: getAWSEnv("AWS_SECRET_ACCESS_KEY"),
					Region:          "us-west-1",
					GitCredentials: schema.AWSCodeCommitGitCredentials{
						Username: "git-username",
						Password: "git-password",
					},
				}),
			},
		}

		testCases = append(testCases, testCase{
			name: "included repos that exist are yielded",
			svcs: svcs,
			assert: func(s *ExternalService) ReposAssertion {
				return func(t testing.TB, rs Repos) {
					t.Helper()

					have := rs.Names()
					sort.Strings(have)

					var want []string
					switch s.Kind {
					case extsvc.KindGitHub:
						want = []string{
							"github.com/sourcegraph/sourcegraph",
							"github.com/tsenart/vegeta",
						}
					case extsvc.KindBitbucketServer:
						want = []string{
							"bitbucket.sgdev.org/SOUR/sourcegraph",
							"bitbucket.sgdev.org/SOUR/vegeta",
						}
					case extsvc.KindGitLab:
						want = []string{
							"gitlab.com/gitlab-org/gitlab-ce",
							"gitlab.com/gnachman/iterm2",
						}
					case extsvc.KindAWSCodeCommit:
						want = []string{
							"__WARNING_DO_NOT_PUT_ANY_PRIVATE_CODE_IN_HERE",
							"empty-repo",
							"stripe-go",
							"test",
							"test2",
						}
					case extsvc.KindOther:
						want = []string{
							"github.com/google/go-cmp",
						}
					}

					if !reflect.DeepEqual(have, want) {
						t.Error(cmp.Diff(have, want))
					}
				}
			},
			err: "<nil>",
		})
	}

	{
		svcs := ExternalServices{
			{
				Kind: extsvc.KindGitHub,
				Config: marshalJSON(t, &schema.GitHubConnection{
					Url:                   "https://github.com",
					Token:                 os.Getenv("GITHUB_ACCESS_TOKEN"),
					RepositoryPathPattern: "{host}/a/b/c/{nameWithOwner}",
					RepositoryQuery:       []string{"none"},
					Repos:                 []string{"tsenart/vegeta"},
				}),
			},
			{
				Kind: extsvc.KindGitLab,
				Config: marshalJSON(t, &schema.GitLabConnection{
					Url:                   "https://gitlab.com",
					Token:                 os.Getenv("GITLAB_ACCESS_TOKEN"),
					RepositoryPathPattern: "{host}/a/b/c/{pathWithNamespace}",
					ProjectQuery:          []string{"none"},
					Projects: []*schema.GitLabProject{
						{Name: "gnachman/iterm2"},
					},
				}),
			},
			{
				Kind: extsvc.KindBitbucketServer,
				Config: marshalJSON(t, &schema.BitbucketServerConnection{
					Url:                   "https://bitbucket.sgdev.org",
					Token:                 os.Getenv("BITBUCKET_SERVER_TOKEN"),
					RepositoryPathPattern: "{host}/a/b/c/{projectKey}/{repositorySlug}",
					RepositoryQuery:       []string{"none"},
					Repos:                 []string{"sour/vegeta"},
				}),
			},
			{
				Kind: extsvc.KindAWSCodeCommit,
				Config: marshalJSON(t, &schema.AWSCodeCommitConnection{
					AccessKeyID:     getAWSEnv("AWS_ACCESS_KEY_ID"),
					SecretAccessKey: getAWSEnv("AWS_SECRET_ACCESS_KEY"),
					Region:          "us-west-1",
					GitCredentials: schema.AWSCodeCommitGitCredentials{
						Username: "git-username",
						Password: "git-password",
					},
					RepositoryPathPattern: "a/b/c/{name}",
				}),
			},
			{
				Kind: extsvc.KindGitolite,
				Config: marshalJSON(t, &schema.GitoliteConnection{
					// Prefix serves as a sort of repositoryPathPattern for Gitolite
					Prefix: "gitolite.mycorp.com/",
					Host:   "ssh://git@127.0.0.1:2222",
				}),
			},
		}

		testCases = append(testCases, testCase{
			name: "repositoryPathPattern determines the repo name",
			svcs: svcs,
			assert: func(s *ExternalService) ReposAssertion {
				return func(t testing.TB, rs Repos) {
					t.Helper()

					haveNames := rs.Names()
					var haveURIs []string
					for _, r := range rs {
						haveURIs = append(haveURIs, r.URI)
					}

					var wantNames, wantURIs []string
					switch s.Kind {
					case extsvc.KindGitHub:
						wantNames = []string{
							"github.com/a/b/c/tsenart/vegeta",
						}
						wantURIs = []string{
							"github.com/tsenart/vegeta",
						}
					case extsvc.KindGitLab:
						wantNames = []string{
							"gitlab.com/a/b/c/gnachman/iterm2",
						}
						wantURIs = []string{
							"gitlab.com/gnachman/iterm2",
						}
					case extsvc.KindBitbucketServer:
						wantNames = []string{
							"bitbucket.sgdev.org/a/b/c/SOUR/vegeta",
						}
						wantURIs = []string{
							"bitbucket.sgdev.org/SOUR/vegeta",
						}
					case extsvc.KindAWSCodeCommit:
						wantNames = []string{
							"a/b/c/empty-repo",
							"a/b/c/stripe-go",
							"a/b/c/test2",
							"a/b/c/__WARNING_DO_NOT_PUT_ANY_PRIVATE_CODE_IN_HERE",
							"a/b/c/test",
						}
						wantURIs = []string{
							"empty-repo",
							"stripe-go",
							"test2",
							"__WARNING_DO_NOT_PUT_ANY_PRIVATE_CODE_IN_HERE",
							"test",
						}
					case extsvc.KindGitolite:
						wantNames = []string{
							"gitolite.mycorp.com/bar",
							"gitolite.mycorp.com/baz",
							"gitolite.mycorp.com/foo",
							"gitolite.mycorp.com/gitolite-admin",
							"gitolite.mycorp.com/testing",
						}
						wantURIs = wantNames
					}

					if !reflect.DeepEqual(haveNames, wantNames) {
						t.Error(cmp.Diff(haveNames, wantNames))
					}
					if !reflect.DeepEqual(haveURIs, wantURIs) {
						t.Error(cmp.Diff(haveURIs, wantURIs))
					}
				}
			},
			err: "<nil>",
		})
	}

	{
		svcs := ExternalServices{
			{
				Kind: extsvc.KindGitLab,
				Config: marshalJSON(t, &schema.GitLabConnection{
					Url:                   "https://gitlab.com",
					Token:                 os.Getenv("GITLAB_ACCESS_TOKEN"),
					RepositoryPathPattern: "{host}/{pathWithNamespace}",
					ProjectQuery:          []string{"none"},
					Projects: []*schema.GitLabProject{
						{Name: "sg-test.d/repo-git"},
						{Name: "sg-test.d/repo-gitrepo"},
					},
					NameTransformations: []*schema.GitLabNameTransformation{
						{
							Regex:       "\\.d/",
							Replacement: "/",
						},
						{
							Regex:       "-git$",
							Replacement: "",
						},
					},
				}),
			},
		}

		testCases = append(testCases, testCase{
			name: "nameTransformations updates the repo name",
			svcs: svcs,
			assert: func(s *ExternalService) ReposAssertion {
				return func(t testing.TB, rs Repos) {
					t.Helper()

					have := rs.Names()
					sort.Strings(have)

					var want []string
					switch s.Kind {
					case extsvc.KindGitLab:
						want = []string{
							"gitlab.com/sg-test/repo",
							"gitlab.com/sg-test/repo-gitrepo",
						}
					}

					if !reflect.DeepEqual(have, want) {
						t.Error(cmp.Diff(have, want))
					}
				}
			},
			err: "<nil>",
		})
	}

	{
		svcs := ExternalServices{
			{
				Kind: extsvc.KindAWSCodeCommit,
				Config: marshalJSON(t, &schema.AWSCodeCommitConnection{
					AccessKeyID:     getAWSEnv("AWS_ACCESS_KEY_ID"),
					SecretAccessKey: getAWSEnv("AWS_SECRET_ACCESS_KEY"),
					Region:          "us-west-1",
					GitCredentials: schema.AWSCodeCommitGitCredentials{
						Username: "git-username",
						Password: "git-password",
					},
				}),
			},
		}

		testCases = append(testCases, testCase{
			name: "yielded repos have authenticated CloneURLs",
			svcs: svcs,
			assert: func(s *ExternalService) ReposAssertion {
				return func(t testing.TB, rs Repos) {
					t.Helper()

					urls := []string{}
					for _, r := range rs {
						urls = append(urls, r.CloneURLs()...)
					}

					switch s.Kind {
					case extsvc.KindAWSCodeCommit:
						want := []string{
							"https://git-username:git-password@git-codecommit.us-west-1.amazonaws.com/v1/repos/empty-repo",
							"https://git-username:git-password@git-codecommit.us-west-1.amazonaws.com/v1/repos/stripe-go",
							"https://git-username:git-password@git-codecommit.us-west-1.amazonaws.com/v1/repos/test2",
							"https://git-username:git-password@git-codecommit.us-west-1.amazonaws.com/v1/repos/__WARNING_DO_NOT_PUT_ANY_PRIVATE_CODE_IN_HERE",
							"https://git-username:git-password@git-codecommit.us-west-1.amazonaws.com/v1/repos/test",
						}

						if have := urls; !reflect.DeepEqual(have, want) {
							t.Error(cmp.Diff(have, want))
						}
					}
				}
			},
			err: "<nil>",
		})
	}

	{
		svcs := ExternalServices{
			{
				Kind: extsvc.KindPhabricator,
				Config: marshalJSON(t, &schema.PhabricatorConnection{
					Url:   "https://secure.phabricator.com",
					Token: os.Getenv("PHABRICATOR_TOKEN"),
				}),
			},
		}

		testCases = append(testCases, testCase{
			name: "phabricator",
			svcs: svcs,
			assert: func(*ExternalService) ReposAssertion {
				return func(t testing.TB, rs Repos) {
					t.Helper()

					if len(rs) == 0 {
						t.Fatalf("no repos yielded")
					}

					for _, r := range rs {
						repo := r.Metadata.(*phabricator.Repo)
						if repo.VCS != "git" {
							t.Fatalf("non git repo yielded: %+v", repo)
						}

						if repo.Status == "inactive" {
							t.Fatalf("inactive repo yielded: %+v", repo)
						}

						if repo.Name == "" {
							t.Fatalf("empty repo name: %+v", repo)
						}

						ext := api.ExternalRepoSpec{
							ID:          repo.PHID,
							ServiceType: extsvc.TypePhabricator,
							ServiceID:   "https://secure.phabricator.com",
						}

						if have, want := r.ExternalRepo, ext; have != want {
							t.Fatal(cmp.Diff(have, want))
						}
					}
				}
			},
			err: "<nil>",
		})
	}

	{
		svcs := ExternalServices{
			{
				Kind: extsvc.KindBitbucketServer,
				Config: marshalJSON(t, &schema.BitbucketServerConnection{
					Url:                   "https://bitbucket.sgdev.org",
					Token:                 os.Getenv("BITBUCKET_SERVER_TOKEN"),
					RepositoryPathPattern: "{repositorySlug}",
					RepositoryQuery:       []string{"none"},
					Repos:                 []string{"sour/vegeta", "PUBLIC/archived-repo"},
				}),
			},
		}

		testCases = append(testCases, testCase{
			name: "bitbucketserver archived",
			svcs: svcs,
			assert: func(s *ExternalService) ReposAssertion {
				return func(t testing.TB, rs Repos) {
					t.Helper()

					want := map[string]bool{
						"vegeta":        false,
						"archived-repo": true,
					}
					got := map[string]bool{}
					for _, r := range rs {
						got[r.Name] = r.Archived
					}

					if !reflect.DeepEqual(got, want) {
						t.Error("mismatch archived state (-want +got):\n", cmp.Diff(want, got))
					}
				}
			},
			err: "<nil>",
		})
	}

	for _, tc := range testCases {
		tc := tc
		for _, svc := range tc.svcs {
			name := svc.Kind + "/" + tc.name
			t.Run(name, func(t *testing.T) {
				cf, save := newClientFactory(t, name)
				defer save(t)

				lg := log15.New()
				lg.SetHandler(log15.DiscardHandler())

				obs := ObservedSource(lg, NewSourceMetrics())
				srcs, err := NewSourcer(cf, obs)(svc)
				if err != nil {
					t.Fatal(err)
				}

				ctx := tc.ctx
				if ctx == nil {
					ctx = context.Background()
				}

				repos, err := listAll(ctx, srcs)
				if have, want := fmt.Sprint(err), tc.err; have != want {
					t.Errorf("error:\nhave: %q\nwant: %q", have, want)
				}

				if tc.assert != nil {
					tc.assert(svc)(t, repos)
				}
			})
		}
	}
}

func newClientFactory(t testing.TB, name string, mws ...httpcli.Middleware) (*httpcli.Factory, func(testing.TB)) {
	cassete := filepath.Join("testdata", "sources", strings.Replace(name, " ", "-", -1))
	rec := newRecorder(t, cassete, update(name))
	mws = append(mws, httpcli.GitHubProxyRedirectMiddleware, gitserverRedirectMiddleware)
	mw := httpcli.NewMiddleware(mws...)
	return httpcli.NewFactory(mw, httptestutil.NewRecorderOpt(rec)),
		func(t testing.TB) { save(t, rec) }
}

func gitserverRedirectMiddleware(cli httpcli.Doer) httpcli.Doer {
	return httpcli.DoerFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Hostname() == "gitserver" {
			// Start local git server first
			req.URL.Host = "127.0.0.1:3178"
			req.URL.Scheme = "http"
		}
		return cli.Do(req)
	})
}

func newRecorder(t testing.TB, file string, record bool) *recorder.Recorder {
	rec, err := httptestutil.NewRecorder(file, record, func(i *cassette.Interaction) error {
		// The ratelimit.Monitor type resets its internal timestamp if it's
		// updated with a timestamp in the past. This makes tests ran with
		// recorded interations just wait for a very long time. Removing
		// these headers from the casseste effectively disables rate-limiting
		// in tests which replay HTTP interactions, which is desired behaviour.
		for _, name := range [...]string{
			"RateLimit-Limit",
			"RateLimit-Observed",
			"RateLimit-Remaining",
			"RateLimit-Reset",
			"RateLimit-Resettime",
			"X-RateLimit-Limit",
			"X-RateLimit-Remaining",
			"X-RateLimit-Reset",
		} {
			i.Response.Headers.Del(name)
		}

		// Phabricator requests include a token in the form and body.
		ua := i.Request.Headers.Get("User-Agent")
		if strings.Contains(strings.ToLower(ua), extsvc.TypePhabricator) {
			i.Request.Body = ""
			i.Request.Form = nil
		}

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	return rec
}

func save(t testing.TB, rec *recorder.Recorder) {
	if err := rec.Stop(); err != nil {
		t.Errorf("failed to update test data: %s", err)
	}
}

func marshalJSON(t testing.TB, v interface{}) string {
	t.Helper()

	bs, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}

	return string(bs)
}

func getAWSEnv(envVar string) string {
	s := os.Getenv(envVar)
	if s == "" {
		s = fmt.Sprintf("BOGUS-%s", envVar)
	}
	return s
}
