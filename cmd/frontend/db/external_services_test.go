package db

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/keegancsmith/sqlf"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
)

func TestExternalServicesListOptions_sqlConditions(t *testing.T) {
	tests := []struct {
		name      string
		kinds     []string
		wantQuery string
		wantArgs  []interface{}
	}{
		{
			name:      "only one kind: GitHub",
			kinds:     []string{"GITHUB"},
			wantQuery: "deleted_at IS NULL AND kind IN ($1)",
			wantArgs:  []interface{}{"GITHUB"},
		},
		{
			name:      "two kinds: GitHub and GitLab",
			kinds:     []string{"GITHUB", "GITLAB"},
			wantQuery: "deleted_at IS NULL AND kind IN ($1 , $2)",
			wantArgs:  []interface{}{"GITHUB", "GITLAB"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			opts := ExternalServicesListOptions{
				Kinds: test.kinds,
			}
			q := sqlf.Join(opts.sqlConditions(), "AND")
			if diff := cmp.Diff(test.wantQuery, q.Query(sqlf.PostgresBindVar)); diff != "" {
				t.Fatalf("query mismatch (-want +got):\n%s", diff)
			} else if diff = cmp.Diff(test.wantArgs, q.Args()); diff != "" {
				t.Fatalf("args mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestExternalServicesStore_ValidateConfig(t *testing.T) {
	tests := map[string]struct {
		kind, config string
		setup        func(t *testing.T)
		teardown     func()
		wantErr      string
	}{
		"0 errors": {
			kind:    "GITHUB",
			config:  `{"url": "https://github.com", "repositoryQuery": ["none"], "token": "abc"}`,
			wantErr: "",
		},
		"1 error": {
			kind:    "GITHUB",
			config:  `{"url": "https://github.com", "repositoryQuery": ["none"], "token": ""}`,
			wantErr: "1 error occurred:\n\t* token: String length must be greater than or equal to 1\n\n",
		},
		"2 errors": {
			kind:    "GITHUB",
			config:  `{"url": "https://github.com", "repositoryQuery": ["none"], "token": "", "x": 123}`,
			wantErr: "2 errors occurred:\n\t* Additional property x is not allowed\n\t* token: String length must be greater than or equal to 1\n\n",
		},
		"no conflicting rate limit": {
			kind:   "GITHUB",
			config: `{"url": "https://github.com", "repositoryQuery": ["none"], "token": "abc", "rateLimit": {"enabled": true, "requestsPerHour": 5000}}`,
			setup: func(t *testing.T) {
				Mocks.ExternalServices.List = func(opt ExternalServicesListOptions) ([]*types.ExternalService, error) {
					return nil, nil
				}
			},
			wantErr: "",
		},
		"conflicting rate limit": {
			kind:   "GITHUB",
			config: `{"url": "https://github.com", "repositoryQuery": ["none"], "token": "abc", "rateLimit": {"enabled": true, "requestsPerHour": 5000}}`,
			setup: func(t *testing.T) {
				t.Cleanup(func() {
					Mocks.ExternalServices.List = nil
				})
				Mocks.ExternalServices.List = func(opt ExternalServicesListOptions) ([]*types.ExternalService, error) {
					return []*types.ExternalService{
						{
							ID:          1,
							Kind:        "GITHUB",
							DisplayName: "GITHUB 1",
							Config:      `{"url": "https://github.com", "repositoryQuery": ["none"], "token": "abc", "rateLimit": {"enabled": true, "requestsPerHour": 5000}}`,
						},
					}, nil
				}
			},
			wantErr: "1 error occurred:\n\t* existing external service, \"GITHUB 1\", already has a rate limit set\n\n",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if test.setup != nil {
				test.setup(t)
			}

			err := (&ExternalServicesStore{}).ValidateConfig(context.Background(), 0, test.kind, test.config, nil)
			var errStr string
			if err != nil {
				errStr = err.Error()
			}
			if errStr != test.wantErr {
				t.Errorf("got error %q, want %q", errStr, test.wantErr)
			}
		})
	}
}
