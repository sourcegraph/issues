package enforcement

import (
	"context"
	"fmt"
	"testing"

	"github.com/sourcegraph/sourcegraph/enterprise/internal/license"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/licensing"
)

func TestEnforcementPreCreateUser(t *testing.T) {
	tests := []struct {
		license         *license.Info
		activeUserCount int
		wantErr         bool
	}{
		// See the impl for why we treat UserCount == 0 as unlimited.
		{
			license:         &license.Info{UserCount: 0},
			activeUserCount: 5,
			wantErr:         false,
		},

		// Non-true-up licenses.
		{
			license:         &license.Info{UserCount: 10},
			activeUserCount: 0,
			wantErr:         false,
		},
		{
			license:         &license.Info{UserCount: 10},
			activeUserCount: 5,
			wantErr:         false,
		},
		{
			license:         &license.Info{UserCount: 10},
			activeUserCount: 9,
			wantErr:         false,
		},
		{
			license:         &license.Info{UserCount: 10},
			activeUserCount: 10,
			wantErr:         true,
		},
		{
			license:         &license.Info{UserCount: 10},
			activeUserCount: 11,
			wantErr:         true,
		},
		{
			license:         &license.Info{UserCount: 10},
			activeUserCount: 12,
			wantErr:         true,
		},

		// True-up licenses.
		{
			license:         &license.Info{Tags: []string{licensing.TrueUpUserCountTag}, UserCount: 10},
			activeUserCount: 5,
			wantErr:         false,
		},
		{
			license:         &license.Info{Tags: []string{licensing.TrueUpUserCountTag}, UserCount: 10},
			activeUserCount: 15,
			wantErr:         false,
		},
	}
	for _, test := range tests {
		t.Run(fmt.Sprintf("license %s with %d active users", test.license, test.activeUserCount), func(t *testing.T) {
			licensing.MockGetConfiguredProductLicenseInfo = func() (*license.Info, string, error) {
				return test.license, "test-signature", nil
			}
			defer func() { licensing.MockGetConfiguredProductLicenseInfo = nil }()
			store := fakeStore{count: test.activeUserCount}
			err := NewPreCreateUserHook(store)(context.Background())
			if gotErr := err != nil; gotErr != test.wantErr {
				t.Errorf("got error %v, want %v", gotErr, test.wantErr)
			}
		})
	}
}

type fakeStore struct{ count int }

func (s fakeStore) Count(context.Context) (int, error) {
	return s.count, nil
}
