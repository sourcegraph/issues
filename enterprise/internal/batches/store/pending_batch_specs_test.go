package store

import (
	"context"
	"testing"

	"github.com/pkg/errors"

	ct "github.com/sourcegraph/sourcegraph/enterprise/internal/batches/testing"
)

func testStorePendingBatchSpecs(t *testing.T, ctx context.Context, s *Store, clock ct.Clock) {
	t.Run("Create", func(t *testing.T) {
		t.Run("invalid user ID", func(t *testing.T) {
			tx, err := s.Transact(ctx)
			if err != nil {
				t.Fatal(tx)
			}
			t.Cleanup(func() { tx.Done(errors.New("always rollback")) })

			if _, err := tx.CreatePendingBatchSpec(ctx, "", -1); err == nil {
				t.Error("unexpected nil error")
			}
		})

		t.Run("valid user ID", func(t *testing.T) {
			spec := "spec"
			user := ct.CreateTestUser(t, s.DB(), false)

			pbs, err := s.CreatePendingBatchSpec(ctx, spec, user.ID)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if pbs.ID == 0 {
				t.Error("unexpected zero ID")
			}
			if pbs.CreatedAt.IsZero() {
				t.Error("unexpected zero CreatedAt")
			}
			if pbs.UpdatedAt.IsZero() {
				t.Error("unexpected zero UpdatedAt")
			}
			if have, want := pbs.CreatorUserID, user.ID; have != want {
				t.Errorf("unexpected user ID: have=%v want=%v", have, want)
			}
			if have, want := pbs.Spec, spec; have != want {
				t.Errorf("unexpected spec: have=%q want=%q", have, want)
			}
			if have, want := pbs.State, "queued"; have != want {
				t.Errorf("unexpected state: have=%q want=%q", have, want)
			}
		})
	})
}
