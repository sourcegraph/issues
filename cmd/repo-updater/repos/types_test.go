package repos

import (
	"context"
	"encoding/json"
	"strconv"
	"testing"
	"time"

	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/awscodecommit"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/bitbucketserver"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/github"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/gitlab"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/gitolite"
	"github.com/sourcegraph/sourcegraph/internal/jsonc"
	"github.com/sourcegraph/sourcegraph/internal/ratelimit"
	"github.com/sourcegraph/sourcegraph/schema"
	"golang.org/x/time/rate"
)

func TestExternalService_Exclude(t *testing.T) {
	now := time.Now()

	type testCase struct {
		name   string
		svcs   ExternalServices
		repos  Repos
		assert ExternalServicesAssertion
	}

	githubService := ExternalService{
		Kind:        extsvc.KindGitHub,
		DisplayName: "Github",
		Config: `{
			// Some comment
			"url": "https://github.com",
			"token": "secret",
			"repositoryQuery": ["none"]
		}`,
		CreatedAt: now,
		UpdatedAt: now,
	}

	gitlabService := ExternalService{
		Kind:        extsvc.KindGitLab,
		DisplayName: "GitLab",
		Config: `{
			// Some comment
			"url": "https://gitlab.com",
			"token": "secret",
			"projectQuery": ["none"]
		}`,
		CreatedAt: now,
		UpdatedAt: now,
	}

	bitbucketServerService := ExternalService{
		Kind:        extsvc.KindBitbucketServer,
		DisplayName: "Bitbucket Server",
		Config: `{
			// Some comment
			"url": "https://bitbucketserver.mycorp.com",
			"username: "admin",
			"token": "secret",
			"repositoryQuery": ["none"]
		}`,
		CreatedAt: now,
		UpdatedAt: now,
	}

	awsCodeCommitService := ExternalService{
		ID:          9,
		Kind:        extsvc.KindAWSCodeCommit,
		DisplayName: "AWS CodeCommit",
		Config: `{
			"region": "us-west-1",
			"accessKeyID": "secret-accessKeyID",
			"secretAccessKey": "secret-secretAccessKey",
			"gitCredentials": {"username": "user", "password": "pw"},
		}`,
		CreatedAt: now,
		UpdatedAt: now,
	}

	gitoliteService := ExternalService{
		Kind:        extsvc.KindGitolite,
		DisplayName: "Gitolite",
		Config: `{
			// Some comment
			"host": "git@gitolite.mycorp.com",
			"prefix": "gitolite.mycorp.com/"
		}`,
		CreatedAt: now,
		UpdatedAt: now,
	}

	otherService := ExternalService{
		Kind:        extsvc.KindOther,
		DisplayName: "Other code hosts",
		Config: formatJSON(t, `{
			"url": "https://git-host.mycorp.com",
			"repos": []
		}`),
		CreatedAt: now,
		UpdatedAt: now,
	}

	repos := Repos{
		{
			Metadata: &github.Repository{
				ID:            "foo",
				NameWithOwner: "org/foo",
			},
		},
		{
			Metadata: &gitlab.Project{
				ProjectCommon: gitlab.ProjectCommon{
					ID:                1,
					PathWithNamespace: "org/foo",
				},
			},
		},
		{
			Metadata: &github.Repository{
				NameWithOwner: "org/baz",
			},
		},
		{
			Metadata: &gitlab.Project{
				ProjectCommon: gitlab.ProjectCommon{
					PathWithNamespace: "org/baz",
				},
			},
		},
		{
			Metadata: &bitbucketserver.Repo{
				ID:   1,
				Slug: "foo",
				Project: &bitbucketserver.Project{
					Key: "org",
				},
			},
		},
		{
			Metadata: &bitbucketserver.Repo{
				Slug: "baz",
				Project: &bitbucketserver.Project{
					Key: "org",
				},
			},
		},
		{
			Metadata: &awscodecommit.Repository{
				ID:   "f001337a-3450-46fd-b7d2-650c0EXAMPLE",
				Name: "foo",
			},
		},
		{
			Metadata: &awscodecommit.Repository{
				ID:   "b4455554-4444-5555-b7d2-888c9EXAMPLE",
				Name: "baz",
			},
		},
		{
			Name: "git-host.mycorp.com/org/foo",
			ExternalRepo: api.ExternalRepoSpec{
				ID:          "1",
				ServiceType: extsvc.TypeOther,
				ServiceID:   "https://git-host.mycorp.com/",
			},
		},
		{
			Name: "git-host.mycorp.com/org/baz",
			ExternalRepo: api.ExternalRepoSpec{
				ServiceType: extsvc.TypeOther,
				ServiceID:   "https://git-host.mycorp.com/",
			},
		},
		{
			Metadata: &gitolite.Repo{Name: "foo"},
		},
	}

	var testCases []testCase
	{
		svcs := ExternalServices{
			githubService.With(func(e *ExternalService) {
				e.Config = formatJSON(t, `
				{
					// Some comment
					"url": "https://github.com",
					"token": "secret",
					"repositoryQuery": ["none"],
					"exclude": [
						{"id": "foo"},
						{"name": "org/BAZ"}
					]
				}`)
			}),
			gitlabService.With(func(e *ExternalService) {
				e.Config = formatJSON(t, `
				{
					// Some comment
					"url": "https://gitlab.com",
					"token": "secret",
					"projectQuery": ["none"],
					"exclude": [
						{"id": 1},
						{"name": "org/baz"}
					]
				}`)
			}),
			bitbucketServerService.With(func(e *ExternalService) {
				e.Config = formatJSON(t, `
				{
					// Some comment
					"url": "https://bitbucketserver.mycorp.com",
					"username": "admin",
					"token": "secret",
					"repositoryQuery": ["none"],
					"exclude": [
						{"id": 1},
						{"name": "org/baz"}
					]
				}`)
			}),
			awsCodeCommitService.With(func(e *ExternalService) {
				e.Config = formatJSON(t, `
				{
					// Some comment
					"region": "us-west-1",
					"accessKeyID": "secret-accessKeyID",
					"secretAccessKey": "secret-secretAccessKey",
					"gitCredentials": {"username": "user", "password": "pw"},
					"exclude": [
						{"id": "f001337a-3450-46fd-b7d2-650c0EXAMPLE"},
						{"name": "baz"}
					]
				}`)
			}),
			gitoliteService.With(func(e *ExternalService) {
				e.Config = formatJSON(t, `
				{
					// Some comment
					"host": "git@gitolite.mycorp.com",
					"prefix": "gitolite.mycorp.com/",
					"exclude": [
						{"name": "foo"}
					]
				}`)
			}),
			&otherService,
		}

		testCases = append(testCases, testCase{
			name:   "already excluded repos are ignored",
			svcs:   svcs,
			repos:  repos,
			assert: Assert.ExternalServicesEqual(svcs...),
		})
	}
	{
		svcs := ExternalServices{
			githubService.With(func(e *ExternalService) {
				e.Config = formatJSON(t, `
				{
					// Some comment
					"url": "https://github.com",
					"token": "secret",
					"repositoryQuery": ["none"],
					"exclude": [
						{"name": "org/boo"},
					]
				}`)
			}),
			gitlabService.With(func(e *ExternalService) {
				e.Config = formatJSON(t, `
				{
					// Some comment
					"url": "https://gitlab.com",
					"token": "secret",
					"projectQuery": ["none"],
					"exclude": [
						{"name": "org/boo"},
					]
				}`)
			}),
			bitbucketServerService.With(func(e *ExternalService) {
				e.Config = formatJSON(t, `
				{
					// Some comment
					"url": "https://gitlab.com",
					"username": "admin",
					"token": "secret",
					"repositoryQuery": ["none"],
					"exclude": [
						{"name": "org/boo"},
					]
				}`)
			}),
			awsCodeCommitService.With(func(e *ExternalService) {
				e.Config = formatJSON(t, `
				{
					// Some comment
					"region": "us-west-1",
					"accessKeyID": "secret-accessKeyID",
					"secretAccessKey": "secret-secretAccessKey",
					"gitCredentials": {"username": "user", "password": "pw"},
					"exclude": [
						{"name": "boo"}
					]
				}`)
			}),
			gitoliteService.With(func(e *ExternalService) {
				e.Config = formatJSON(t, `
				{
					// Some comment
					"host": "git@gitolite.mycorp.com",
					"prefix": "gitolite.mycorp.com/",
					"exclude": [
						{"name": "boo"}
					]
				}`)
			}),
			otherService.With(func(e *ExternalService) {
				e.Config = formatJSON(t, `
				{
					"url": "https://git-host.mycorp.com",
					"repos": [
						"org/foo",
						"org/boo",
						"org/baz"
					]
				}`)
			}),
		}

		testCases = append(testCases, testCase{
			name:  "repos are excluded",
			svcs:  svcs,
			repos: repos,
			assert: Assert.ExternalServicesEqual(
				githubService.With(func(e *ExternalService) {
					e.Config = formatJSON(t, `
					{
						// Some comment
						"url": "https://github.com",
						"token": "secret",
						"repositoryQuery": ["none"],
						"exclude": [
							{"name": "org/boo"},
							{"id": "foo", "name": "org/foo"},
							{"name": "org/baz"}
						]
					}`)
				}),
				gitlabService.With(func(e *ExternalService) {
					e.Config = formatJSON(t, `
					{
						// Some comment
						"url": "https://gitlab.com",
						"token": "secret",
						"projectQuery": ["none"],
						"exclude": [
							{"name": "org/boo"},
							{"id": 1, "name": "org/foo"},
							{"name": "org/baz"}
						]
					}`)
				}),
				bitbucketServerService.With(func(e *ExternalService) {
					e.Config = formatJSON(t, `
					{
						// Some comment
						"url": "https://gitlab.com",
						"username": "admin",
						"token": "secret",
						"repositoryQuery": ["none"],
						"exclude": [
							{"name": "org/boo"},
							{"id": 1, "name": "org/foo"},
							{"name": "org/baz"}
						]
					}`)
				}),
				awsCodeCommitService.With(func(e *ExternalService) {
					e.Config = formatJSON(t, `
					{
						// Some comment
						"region": "us-west-1",
						"accessKeyID": "secret-accessKeyID",
						"secretAccessKey": "secret-secretAccessKey",
						"gitCredentials": {"username": "user", "password": "pw"},
						"exclude": [
							{"name": "boo"},
							{"id": "f001337a-3450-46fd-b7d2-650c0EXAMPLE", "name": "foo"},
							{"id": "b4455554-4444-5555-b7d2-888c9EXAMPLE", "name": "baz"}
						]
					}`)
				}),
				gitoliteService.With(func(e *ExternalService) {
					e.Config = formatJSON(t, `
					{
						// Some comment
						"host": "git@gitolite.mycorp.com",
						"prefix": "gitolite.mycorp.com/",
						"exclude": [
							{"name": "boo"},
							{"name": "foo"}
						]
					}`)
				}),
				otherService.With(func(e *ExternalService) {
					e.Config = formatJSON(t, `
					{
						"url": "https://git-host.mycorp.com",
						"repos": [
							"org/boo"
						]
					}`)
				}),
			),
		})
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			svcs, repos := tc.svcs.Clone(), tc.repos.Clone()

			var err error
			for _, svc := range svcs {
				if err = svc.Exclude(repos...); err != nil {
					t.Fatal(err)
				}
			}

			if tc.assert != nil {
				tc.assert(t, svcs)
			}
		})
	}
}

func TestReposNamesSummary(t *testing.T) {
	var rps Repos

	eid := func(id int) api.ExternalRepoSpec {
		return api.ExternalRepoSpec{
			ID:          strconv.Itoa(id),
			ServiceType: "fake",
			ServiceID:   "https://fake.com",
		}
	}

	for i := 0; i < 5; i++ {
		rps = append(rps, &Repo{Name: "bar", ExternalRepo: eid(i)})
	}

	expected := "bar bar bar bar bar"
	ns := rps.NamesSummary()
	if ns != expected {
		t.Errorf("expected %s, got %s", expected, ns)
	}

	rps = nil

	for i := 0; i < 22; i++ {
		rps = append(rps, &Repo{Name: "b", ExternalRepo: eid(i)})
	}

	expected = "b b b b b b b b b b b b b b b b b b b b..."
	ns = rps.NamesSummary()
	if ns != expected {
		t.Errorf("expected %s, got %s", expected, ns)
	}
}

// Our uses of pick happen from iterating through a map. So we can't guarantee
// that we test both pick(a, b) and pick(b, a) without writing this specific
// test.
func TestPick(t *testing.T) {
	eid := func(id string) api.ExternalRepoSpec {
		return api.ExternalRepoSpec{
			ID:          id,
			ServiceType: "fake",
			ServiceID:   "https://fake.com",
		}
	}
	a := &Repo{Name: "bar", ExternalRepo: eid("1")}
	b := &Repo{Name: "bar", ExternalRepo: eid("2")}

	for _, args := range [][2]*Repo{{a, b}, {b, a}} {
		keep, discard := pick(args[0], args[1])
		if keep != a || discard != b {
			t.Errorf("unexpected pick(%v, %v)", args[0], args[1])
		}
	}
}

func formatJSON(t testing.TB, s string) string {
	formatted, err := jsonc.Format(s, nil)
	if err != nil {
		t.Fatal(err)
	}

	return formatted
}

func TestSyncRateLimiters(t *testing.T) {
	now := time.Now()
	ctx := context.Background()

	baseURL := "http://gitlab.com/"

	type limitOptions struct {
		includeLimit bool
		enabled      bool
		perHour      float64
	}

	makeLister := func(options ...limitOptions) *MockExternalServicesLister {
		services := make([]*ExternalService, 0, len(options))
		for i, o := range options {
			svc := &ExternalService{
				ID:          int64(i) + 1,
				Kind:        "GitLab",
				DisplayName: "GitLab",
				CreatedAt:   now,
				UpdatedAt:   now,
				DeletedAt:   time.Time{},
			}
			config := schema.GitLabConnection{
				Url: baseURL,
			}
			if o.includeLimit {
				config.RateLimit = &schema.GitLabRateLimit{
					RequestsPerHour: o.perHour,
					Enabled:         o.enabled,
				}
			}
			data, err := json.Marshal(config)
			if err != nil {
				t.Fatal(err)
			}
			svc.Config = string(data)
			services = append(services, svc)
		}
		return &MockExternalServicesLister{
			listExternalServices: func(ctx context.Context, args StoreListExternalServicesArgs) ([]*ExternalService, error) {
				return services, nil
			},
		}
	}

	for _, tc := range []struct {
		name    string
		options []limitOptions
		want    rate.Limit
	}{
		{
			name:    "No limiters defined",
			options: []limitOptions{},
			want:    rate.Inf,
		},
		{
			name: "One limit, enabled",
			options: []limitOptions{
				{
					includeLimit: true,
					enabled:      true,
					perHour:      3600,
				},
			},
			want: rate.Limit(1),
		},
		{
			name: "Two limits, enabled",
			options: []limitOptions{
				{
					includeLimit: true,
					enabled:      true,
					perHour:      3600,
				},
				{
					includeLimit: true,
					enabled:      true,
					perHour:      7200,
				},
			},
			want: rate.Limit(1),
		},
		{
			name: "One limit, disabled",
			options: []limitOptions{
				{
					includeLimit: true,
					enabled:      false,
					perHour:      3600,
				},
			},
			want: rate.Inf,
		},
		{
			name: "One limit, zero",
			options: []limitOptions{
				{
					includeLimit: true,
					enabled:      true,
					perHour:      0,
				},
			},
			want: rate.Limit(0),
		},
		{
			name: "No limit",
			options: []limitOptions{
				{
					includeLimit: false,
				},
			},
			want: rate.Limit(10),
		},
		{
			name: "Two limits, one default",
			options: []limitOptions{
				{
					includeLimit: true,
					enabled:      true,
					perHour:      3600,
				},
				{
					includeLimit: false,
				},
			},
			want: rate.Limit(1),
		},
		// Default for GitLab is 10 per second
		{
			name: "Default, Higher than default",
			options: []limitOptions{
				{
					includeLimit: true,
					enabled:      true,
					perHour:      20 * 3600,
				},
				{
					includeLimit: false,
				},
			},
			want: rate.Limit(20),
		},
		{
			name: "Higher than default, Default",
			options: []limitOptions{
				{
					includeLimit: false,
				},
				{
					includeLimit: true,
					enabled:      true,
					perHour:      20 * 3600,
				},
			},
			want: rate.Limit(20),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reg := ratelimit.NewRegistry()
			r := &RateLimitSyncer{
				registry:      reg,
				serviceLister: makeLister(tc.options...),
				limit:         10,
			}

			err := r.SyncRateLimiters(ctx)
			if err != nil {
				t.Fatal(err)
			}

			// We should have the lower limit
			l := reg.Get(baseURL)
			if l == nil {
				t.Fatalf("expected a limiter")
			}
			if l.Limit() != tc.want {
				t.Fatalf("Expected limit %f, got %f", tc.want, l.Limit())
			}
		})
	}
}

type MockExternalServicesLister struct {
	listExternalServices func(context.Context, StoreListExternalServicesArgs) ([]*ExternalService, error)
}

func (m MockExternalServicesLister) ListExternalServices(ctx context.Context, args StoreListExternalServicesArgs) ([]*ExternalService, error) {
	return m.listExternalServices(ctx, args)
}
