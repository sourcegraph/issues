package app

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/gorilla/mux"

	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/database/dbtesting"
	srcprometheus "github.com/sourcegraph/sourcegraph/internal/src-prometheus"
	"github.com/sourcegraph/sourcegraph/internal/types"
	"github.com/sourcegraph/sourcegraph/schema"
)

func Test_prometheusValidator(t *testing.T) {
	type args struct {
		prometheusURL string
		config        conf.Unified
	}
	tests := []struct {
		name                 string
		args                 args
		wantProblemSubstring string
	}{
		{
			name: "no problem if prometheus not set",
			args: args{
				prometheusURL: "",
			},
			wantProblemSubstring: "",
		},
		{
			name: "no problem if no alerts set",
			args: args{
				prometheusURL: "http://prometheus:9090",
				config:        conf.Unified{},
			},
			wantProblemSubstring: "",
		},
		{
			name: "url and alerts set, but malformed prometheus URL",
			args: args{
				prometheusURL: " http://prometheus:9090",
				config: conf.Unified{
					SiteConfiguration: schema.SiteConfiguration{
						ObservabilityAlerts: []*schema.ObservabilityAlerts{{
							Level: "critical",
						}},
					},
				},
			},
			wantProblemSubstring: "misconfigured",
		},
		{
			name: "prometheus not found (with only observability.alerts configured)",
			args: args{
				prometheusURL: "http://no-prometheus:9090",
				config: conf.Unified{
					SiteConfiguration: schema.SiteConfiguration{
						ObservabilityAlerts: []*schema.ObservabilityAlerts{{
							Level: "critical",
						}},
					},
				},
			},
			wantProblemSubstring: "failed to fetch alerting configuration",
		},
		{
			name: "prometheus not found (with only observability.silenceAlerts configured)",
			args: args{
				prometheusURL: "http://no-prometheus:9090",
				config: conf.Unified{
					SiteConfiguration: schema.SiteConfiguration{
						ObservabilitySilenceAlerts: []string{"warning_gitserver_disk_space_remaining"},
					},
				},
			},
			wantProblemSubstring: "failed to fetch alerting configuration",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validate := newPrometheusValidator(srcprometheus.NewClient(tt.args.prometheusURL))
			problems := validate(tt.args.config)
			if tt.wantProblemSubstring == "" {
				if len(problems) > 0 {
					t.Errorf("expected no problems, got %+v", problems)
				}
			} else {
				found := false
				for _, p := range problems {
					if strings.Contains(p.String(), tt.wantProblemSubstring) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected problem '%s', got %+v", tt.wantProblemSubstring, problems)
				}
			}
		})
	}
}

func TestGrafanaLicensing(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	t.Run("licensed requests succeed", func(t *testing.T) {
		database.Mocks.Users.GetByCurrentAuthUser = func(ctx context.Context) (*types.User, error) {
			return &types.User{ID: 1, SiteAdmin: true}, nil
		}
		defer func() { database.Mocks.Users.GetByCurrentAuthUser = nil }()

		PreMountGrafanaHook = func() error { return nil }
		defer func() { PreMountGrafanaHook = nil }()

		router := mux.NewRouter()
		addGrafana(router, &dbtesting.MockDB{})
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest("GET", "/grafana", nil))

		if got, want := rec.Code, http.StatusOK; got != want {
			t.Fatalf("status code: got %d, want %d", got, want)
		}
	})

	t.Run("non-licensed requests fail", func(t *testing.T) {
		database.Mocks.Users.GetByCurrentAuthUser = func(ctx context.Context) (*types.User, error) {
			return &types.User{ID: 1, SiteAdmin: true}, nil
		}
		defer func() { database.Mocks.Users.GetByCurrentAuthUser = nil }()

		PreMountGrafanaHook = func() error { return errors.New("test fail") }
		defer func() { PreMountGrafanaHook = nil }()

		router := mux.NewRouter()
		addGrafana(router, &dbtesting.MockDB{})
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest("GET", "/grafana", nil))

		if got, want := rec.Code, http.StatusUnauthorized; got != want {
			t.Fatalf("status code: got %d, want %d", got, want)
		}
		// http.Error appends a trailing newline that won't be present in
		// the error message itself, so we need to remove it.
		if diff := cmp.Diff(strings.TrimSuffix(rec.Body.String(), "\n"), errMonitoringNotLicensed); diff != "" {
			t.Fatal(diff)
		}
	})
}
