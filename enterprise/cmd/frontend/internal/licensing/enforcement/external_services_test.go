package enforcement

import (
	"context"
	"testing"

	"github.com/sourcegraph/sourcegraph/enterprise/internal/licensing"
	"github.com/sourcegraph/sourcegraph/internal/db"
)

func TestNewPreCreateExternalServiceHook(t *testing.T) {
	ctx := context.Background()
	if !licensing.EnforceTiers {
		licensing.EnforceTiers = true
		defer func() { licensing.EnforceTiers = false }()
	}

	t.Run("when external services can be added", func(t *testing.T) {
		mockGetExternalServicesLimit = func(_ context.Context) int {
			return 20
		}
		defer func() { mockGetExternalServicesLimit = nil }()

		hook := NewPreCreateExternalServiceHook(&mockExternalServicesStore{extSvcCount: 10})
		if got := hook(ctx); got != nil {
			t.Fatalf("got %v, want nil", got)
		}
	})

	t.Run("when the external service limit has been reached", func(t *testing.T) {
		mockGetExternalServicesLimit = func(_ context.Context) int {
			return 5
		}
		defer func() { mockGetExternalServicesLimit = nil }()

		hook := NewPreCreateExternalServiceHook(&mockExternalServicesStore{extSvcCount: 10})
		if hook(ctx) == nil {
			t.Fatalf("got nil, want error")
		}
	})
}

// A mockExternalServicesStore implements the ExternalServicesStore interface for test purposes.
type mockExternalServicesStore struct {
	extSvcCount int
	err         error
}

// Count returns the number of external services currently configured.
func (m *mockExternalServicesStore) Count(_ context.Context, _ db.ExternalServicesListOptions) (int, error) {
	return m.extSvcCount, m.err
}
