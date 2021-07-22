package repos

import (
	"context"
	"encoding/json"
	"strconv"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/time/rate"

	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/github"
	"github.com/sourcegraph/sourcegraph/internal/ratelimit"
	"github.com/sourcegraph/sourcegraph/internal/rcache"
	"github.com/sourcegraph/sourcegraph/internal/types"
	"github.com/sourcegraph/sourcegraph/schema"
)

func TestReposNamesSummary(t *testing.T) {
	var rps types.Repos

	eid := func(id int) api.ExternalRepoSpec {
		return api.ExternalRepoSpec{
			ID:          strconv.Itoa(id),
			ServiceType: "fake",
			ServiceID:   "https://fake.com",
		}
	}

	for i := 0; i < 5; i++ {
		rps = append(rps, &types.Repo{Name: "bar", ExternalRepo: eid(i)})
	}

	expected := "bar bar bar bar bar"
	ns := rps.NamesSummary()
	if ns != expected {
		t.Errorf("expected %s, got %s", expected, ns)
	}

	rps = nil

	for i := 0; i < 22; i++ {
		rps = append(rps, &types.Repo{Name: "b", ExternalRepo: eid(i)})
	}

	expected = "b b b b b b b b b b b b b b b b b b b b..."
	ns = rps.NamesSummary()
	if ns != expected {
		t.Errorf("expected %s, got %s", expected, ns)
	}
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
		services := make([]*types.ExternalService, 0, len(options))
		for i, o := range options {
			svc := &types.ExternalService{
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
			list: func(ctx context.Context, args database.ExternalServicesListOptions) ([]*types.ExternalService, error) {
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
	list func(context.Context, database.ExternalServicesListOptions) ([]*types.ExternalService, error)
}

func (m MockExternalServicesLister) List(ctx context.Context, args database.ExternalServicesListOptions) ([]*types.ExternalService, error) {
	return m.list(ctx, args)
}

func TestGrantedScopes(t *testing.T) {
	rcache.SetupForTest(t)
	cache := rcache.New("TestGrantedScopes")
	ctx := context.Background()

	want := []string{"repo"}
	github.MockGetAuthenticatedUserOAuthScopes = func(ctx context.Context) ([]string, error) {
		return want, nil
	}

	svc := &types.ExternalService{Kind: extsvc.KindGitHub, Config: `{"token": "abc"}`, NamespaceUserID: 123}
	// Run twice to use cache
	for i := 0; i < 2; i++ {
		have, err := GrantedScopes(ctx, cache, svc)
		if err != nil {
			t.Fatal(i, err)
		}
		if diff := cmp.Diff(want, have); diff != "" {
			t.Fatal(i, diff)
		}
	}
}

func TestHashToken(t *testing.T) {
	// Sanity check output of hash function
	h, err := hashToken("token")
	if err != nil {
		t.Fatal(err)
	}
	want := "47a1037c"
	if want != h {
		t.Fatalf("Want %q, got %q", want, h)
	}
}
