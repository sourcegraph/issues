package store

import (
	"context"
	"strconv"
	"testing"

	"github.com/google/go-cmp/cmp"

	ct "github.com/sourcegraph/sourcegraph/enterprise/internal/batches/testing"
	btypes "github.com/sourcegraph/sourcegraph/enterprise/internal/batches/types"
)

func testStoreChangesetSpecExecutions(t *testing.T, ctx context.Context, s *Store, clock ct.Clock) {
	testBatchSpec := `theSpec: yeah`

	execs := make([]*btypes.BatchSpecExecution, 0, 2)
	for i := 0; i < cap(execs); i++ {
		c := &btypes.BatchSpecExecution{
			State:           btypes.BatchSpecExecutionStateQueued,
			BatchSpec:       testBatchSpec,
			UserID:          int32(i + 123),
			NamespaceUserID: int32(i + 345),
		}

		execs = append(execs, c)
	}

	t.Run("Create", func(t *testing.T) {
		for i, exec := range execs {
			if err := s.CreateBatchSpecExecution(ctx, exec); err != nil {
				t.Fatal(err)
			}

			have := exec
			want := &btypes.BatchSpecExecution{
				ID:              have.ID,
				RandID:          have.RandID,
				CreatedAt:       clock.Now(),
				UpdatedAt:       clock.Now(),
				State:           btypes.BatchSpecExecutionStateQueued,
				BatchSpec:       testBatchSpec,
				UserID:          int32(i + 123),
				NamespaceUserID: int32(i + 345),
			}

			if have.ID == 0 {
				t.Fatal("ID should not be zero")
			}

			if have.RandID == "" {
				t.Fatal("RandID should not be empty")
			}

			if diff := cmp.Diff(have, want); diff != "" {
				t.Fatal(diff)
			}
		}
	})

	t.Run("Get", func(t *testing.T) {
		t.Run("GetByID", func(t *testing.T) {
			for i, exec := range execs {
				t.Run(strconv.Itoa(i), func(t *testing.T) {
					have, err := s.GetBatchSpecExecution(ctx, GetBatchSpecExecutionOpts{ID: exec.ID})
					if err != nil {
						t.Fatal(err)
					}

					if diff := cmp.Diff(have, exec); diff != "" {
						t.Fatal(diff)
					}
				})
			}
		})

		t.Run("GetByRandID", func(t *testing.T) {
			for i, exec := range execs {
				t.Run(strconv.Itoa(i), func(t *testing.T) {
					have, err := s.GetBatchSpecExecution(ctx, GetBatchSpecExecutionOpts{RandID: exec.RandID})
					if err != nil {
						t.Fatal(err)
					}

					if diff := cmp.Diff(have, exec); diff != "" {
						t.Fatal(diff)
					}
				})
			}
		})

		t.Run("NoResults", func(t *testing.T) {
			opts := GetBatchSpecExecutionOpts{ID: 0xdeadbeef}

			_, have := s.GetBatchSpecExecution(ctx, opts)
			want := ErrNoResults

			if have != want {
				t.Fatalf("have err %v, want %v", have, want)
			}
		})
	})
}
