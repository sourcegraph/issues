package conf

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/sourcegraph/sourcegraph/internal/conf/confdefaults"
	"github.com/sourcegraph/sourcegraph/internal/conf/conftypes"

	"github.com/sourcegraph/sourcegraph/schema"
)

func TestSearchIndexEnabled(t *testing.T) {
	tests := []struct {
		name string
		sc   *Unified
		env  []string
		want interface{}
	}{{
		name: "SearchIndex defaults to false in docker",
		sc:   &Unified{},
		env:  []string{"DEPLOY_TYPE=docker-container"},
		want: false,
	}, {
		name: "SearchIndex defaults to true in k8s",
		sc:   &Unified{},
		env:  []string{"DEPLOY_TYPE=k8s"},
		want: true,
	}, {
		name: "SearchIndex enabled",
		sc:   &Unified{SiteConfiguration: schema.SiteConfiguration{SearchIndexEnabled: boolPtr(true)}},
		env:  []string{"DEPLOY_TYPE=docker-container"},
		want: true,
	}, {
		name: "SearchIndex disabled",
		sc:   &Unified{SiteConfiguration: schema.SiteConfiguration{SearchIndexEnabled: boolPtr(false)}},
		env:  []string{"DEPLOY_TYPE=docker-container"},
		want: false,
	}, {
		name: "SearchIndex INDEXED_SEARCH=f",
		sc:   &Unified{},
		env:  []string{"DEPLOY_TYPE=docker-container", "INDEXED_SEARCH=f"},
		want: false,
	}, {
		name: "SearchIndex INDEXED_SEARCH=t",
		sc:   &Unified{},
		env:  []string{"DEPLOY_TYPE=docker-container", "INDEXED_SEARCH=t"},
		want: true,
	}}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for _, e := range test.env {
				cleanup := setenv(t, e)
				defer cleanup()
			}
			Mock(test.sc)
			got := SearchIndexEnabled()
			if got != test.want {
				t.Fatalf("SearchIndexEnabled() = %v, want %v", got, test.want)
			}
		})
	}

	defaults := map[string]conftypes.RawUnified{
		"Kubernetes":      confdefaults.KubernetesOrDockerComposeOrPureDocker,
		"Default":         confdefaults.Default,
		"DevAndTesting":   confdefaults.DevAndTesting,
		"DockerContainer": confdefaults.DockerContainer,
	}
	for dStr, d := range defaults {
		test := fmt.Sprintf("for %s defaults", dStr)
		t.Run(test, func(t *testing.T) {
			cfg, err := ParseConfig(d)
			if err != nil {
				t.Fatal(err)
			}
			Mock(cfg)
			defer Mock(nil)
			if !SearchIndexEnabled() {
				t.Errorf("search indexing should be enabled by default for Docker deployments")
			}
		})
	}
}

func TestAuthPasswordResetLinkDuration(t *testing.T) {
	tests := []struct {
		name string
		sc   *Unified
		want int
	}{{
		name: "password link expiry has a default value if null",
		sc:   &Unified{},
		want: defaultPasswordLinkExpiry,
	}, {
		name: "password link expiry has a default value if blank",
		sc:   &Unified{SiteConfiguration: schema.SiteConfiguration{AuthPasswordResetLinkExpiry: 0}},
		want: defaultPasswordLinkExpiry,
	}, {
		name: "password link expiry can be customized",
		sc:   &Unified{SiteConfiguration: schema.SiteConfiguration{AuthPasswordResetLinkExpiry: 60}},
		want: 60,
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			Mock(test.sc)
			if got, want := AuthPasswordResetLinkExpiry(), test.want; got != want {
				t.Fatalf("AuthPasswordResetLinkExpiry() = %v, want %v", got, want)
			}
		})
	}
}

func TestGitMaxCodehostRequestsPerSecond(t *testing.T) {
	tests := []struct {
		name string
		sc   *Unified
		want int
	}{
		{
			name: "not set should return default",
			sc:   &Unified{SiteConfiguration: schema.SiteConfiguration{}},
			want: -1,
		},
		{
			name: "bad value should return default",
			sc:   &Unified{SiteConfiguration: schema.SiteConfiguration{GitMaxCodehostRequestsPerSecond: intPtr(-100)}},
			want: -1,
		},
		{
			name: "set 0 should return 0",
			sc:   &Unified{SiteConfiguration: schema.SiteConfiguration{GitMaxCodehostRequestsPerSecond: intPtr(0)}},
			want: 0,
		},
		{
			name: "set non-0 should return non-0",
			sc:   &Unified{SiteConfiguration: schema.SiteConfiguration{GitMaxCodehostRequestsPerSecond: intPtr(100)}},
			want: 100,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			Mock(test.sc)
			if got, want := GitMaxCodehostRequestsPerSecond(), test.want; got != want {
				t.Fatalf("GitMaxCodehostRequestsPerSecond() = %v, want %v", got, want)
			}
		})
	}
}

func TestGitMaxFlowRateMbps(t *testing.T) {
	tests := []struct {
		name string
		sc   *Unified
		want int
	}{
		{
			name: "a negative value returns the default",
			sc:   &Unified{SiteConfiguration: schema.SiteConfiguration{GitMaxFlowRateMbps: -1}},
			want: 1000,
		},
		{
			name: "a zero value returns the default",
			sc:   &Unified{SiteConfiguration: schema.SiteConfiguration{GitMaxFlowRateMbps: 0}},
			want: 1000,
		},
		{
			name: "a positive value is respected",
			sc:   &Unified{SiteConfiguration: schema.SiteConfiguration{GitMaxFlowRateMbps: 1}},
			want: 1,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			Mock(test.sc)
			if got, want := GitMaxFlowRateMbps(), test.want; got != want {
				t.Fatalf("GitMaxFlowRateMbps() = %v, want %v", got, want)
			}
		})
	}
}

func setenv(t *testing.T, keyval string) func() {
	t.Helper()

	parts := strings.SplitN(keyval, "=", 2)
	key := parts[0]
	value := parts[1]

	orig, set := os.LookupEnv(key)
	if err := os.Setenv(key, value); err != nil {
		t.Fatal(err)
	}
	if set {
		return func() {
			if err := os.Setenv(key, orig); err != nil {
				t.Fatal(err)
			}
		}
	}
	return func() {
		if err := os.Unsetenv(key); err != nil {
			t.Fatal(err)
		}
	}
}

func boolPtr(b bool) *bool {
	return &b
}

func intPtr(i int) *int {
	return &i
}
