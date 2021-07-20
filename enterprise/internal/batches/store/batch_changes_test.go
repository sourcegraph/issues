package store

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/sourcegraph/go-diff/diff"

	ct "github.com/sourcegraph/sourcegraph/enterprise/internal/batches/testing"
	btypes "github.com/sourcegraph/sourcegraph/enterprise/internal/batches/types"
	"github.com/sourcegraph/sourcegraph/internal/actor"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
)

func testStoreBatchChanges(t *testing.T, ctx context.Context, s *Store, clock ct.Clock) {
	cs := make([]*btypes.BatchChange, 0, 3)

	t.Run("Create", func(t *testing.T) {
		for i := 0; i < cap(cs); i++ {
			c := &btypes.BatchChange{
				Name:        fmt.Sprintf("test-batch-change-%d", i),
				Description: "All the Javascripts are belong to us",

				InitialApplierID: int32(i) + 50,
				LastAppliedAt:    clock.Now(),
				LastApplierID:    int32(i) + 99,

				BatchSpecID: 1742 + int64(i),
				ClosedAt:    clock.Now(),
			}

			if i == 0 {
				// Check for nullability of fields by not setting them
				c.ClosedAt = time.Time{}
			}

			if i%2 == 0 {
				c.NamespaceOrgID = int32(i) + 23
			} else {
				c.NamespaceUserID = c.InitialApplierID
			}

			want := c.Clone()
			have := c

			err := s.CreateBatchChange(ctx, have)
			if err != nil {
				t.Fatal(err)
			}

			if have.ID == 0 {
				t.Fatal("ID should not be zero")
			}

			want.ID = have.ID
			want.CreatedAt = clock.Now()
			want.UpdatedAt = clock.Now()

			if diff := cmp.Diff(have, want); diff != "" {
				t.Fatal(diff)
			}

			cs = append(cs, c)
		}
	})

	t.Run("Count", func(t *testing.T) {
		count, err := s.CountBatchChanges(ctx, CountBatchChangesOpts{})
		if err != nil {
			t.Fatal(err)
		}

		if have, want := count, len(cs); have != want {
			t.Fatalf("have count: %d, want: %d", have, want)
		}

		t.Run("Global", func(t *testing.T) {
			count, err = s.CountBatchChanges(ctx, CountBatchChangesOpts{})
			if err != nil {
				t.Fatal(err)
			}

			if have, want := count, len(cs); have != want {
				t.Fatalf("have count: %d, want: %d", have, want)
			}
		})

		t.Run("ChangesetID", func(t *testing.T) {
			changeset := ct.CreateChangeset(t, ctx, s, ct.TestChangesetOpts{
				BatchChanges: []btypes.BatchChangeAssoc{{BatchChangeID: cs[0].ID}},
			})

			count, err = s.CountBatchChanges(ctx, CountBatchChangesOpts{ChangesetID: changeset.ID})
			if err != nil {
				t.Fatal(err)
			}

			if have, want := count, 1; have != want {
				t.Fatalf("have count: %d, want: %d", have, want)
			}
		})

		t.Run("OnlyForAuthor set", func(t *testing.T) {
			for _, c := range cs {
				count, err = s.CountBatchChanges(ctx, CountBatchChangesOpts{InitialApplierID: c.InitialApplierID})
				if err != nil {
					t.Fatal(err)
				}
				if have, want := count, 1; have != want {
					t.Fatalf("Incorrect number of batch changes counted, want=%d have=%d", want, have)
				}
			}
		})

		t.Run("NamespaceUserID", func(t *testing.T) {
			wantCounts := map[int32]int{}
			for _, c := range cs {
				if c.NamespaceUserID == 0 {
					continue
				}
				wantCounts[c.NamespaceUserID] += 1
			}
			if len(wantCounts) == 0 {
				t.Fatalf("No batch changes with NamespaceUserID")
			}

			for userID, want := range wantCounts {
				have, err := s.CountBatchChanges(ctx, CountBatchChangesOpts{NamespaceUserID: userID})
				if err != nil {
					t.Fatal(err)
				}

				if have != want {
					t.Fatalf("batch changes count for NamespaceUserID=%d wrong. want=%d, have=%d", userID, want, have)
				}
			}
		})

		t.Run("NamespaceOrgID", func(t *testing.T) {
			wantCounts := map[int32]int{}
			for _, c := range cs {
				if c.NamespaceOrgID == 0 {
					continue
				}
				wantCounts[c.NamespaceOrgID] += 1
			}
			if len(wantCounts) == 0 {
				t.Fatalf("No batch changes with NamespaceOrgID")
			}

			for orgID, want := range wantCounts {
				have, err := s.CountBatchChanges(ctx, CountBatchChangesOpts{NamespaceOrgID: orgID})
				if err != nil {
					t.Fatal(err)
				}

				if have != want {
					t.Fatalf("batch changes count for NamespaceOrgID=%d wrong. want=%d, have=%d", orgID, want, have)
				}
			}
		})
	})

	t.Run("List", func(t *testing.T) {
		t.Run("By ChangesetID", func(t *testing.T) {
			for i := 1; i <= len(cs); i++ {
				changeset := ct.CreateChangeset(t, ctx, s, ct.TestChangesetOpts{
					BatchChanges: []btypes.BatchChangeAssoc{{BatchChangeID: cs[i-1].ID}},
				})
				opts := ListBatchChangesOpts{ChangesetID: changeset.ID}

				ts, next, err := s.ListBatchChanges(ctx, opts)
				if err != nil {
					t.Fatal(err)
				}

				if have, want := next, int64(0); have != want {
					t.Fatalf("opts: %+v: have next %v, want %v", opts, have, want)
				}

				have, want := ts, cs[i-1:i]
				if len(have) != len(want) {
					t.Fatalf("listed %d batch changes, want: %d", len(have), len(want))
				}

				if diff := cmp.Diff(have, want); diff != "" {
					t.Fatalf("opts: %+v, diff: %s", opts, diff)
				}
			}
		})

		t.Run("By RepoID", func(t *testing.T) {
			repoStore := database.ReposWith(s)
			esStore := database.ExternalServicesWith(s)

			repo1 := ct.TestRepo(t, esStore, extsvc.KindGitHub)
			repo2 := ct.TestRepo(t, esStore, extsvc.KindGitHub)
			repo3 := ct.TestRepo(t, esStore, extsvc.KindGitHub)
			if err := repoStore.Create(ctx, repo1, repo2, repo3); err != nil {
				t.Fatal(err)
			}

			// 1 batch change + changeset is associated with the first repo
			ct.CreateChangeset(t, ctx, s, ct.TestChangesetOpts{
				Repo:         repo1.ID,
				BatchChanges: []btypes.BatchChangeAssoc{{BatchChangeID: cs[0].ID}},
			})

			// 2 batch changes, each with 1 changeset, are associated with the second repo
			ct.CreateChangeset(t, ctx, s, ct.TestChangesetOpts{
				Repo:         repo2.ID,
				BatchChanges: []btypes.BatchChangeAssoc{{BatchChangeID: cs[0].ID}},
			})
			ct.CreateChangeset(t, ctx, s, ct.TestChangesetOpts{
				Repo:         repo2.ID,
				BatchChanges: []btypes.BatchChangeAssoc{{BatchChangeID: cs[1].ID}},
			})

			// no batch changes are associated with the third repo

			{
				tcs := []struct {
					repoID        api.RepoID
					listLen       int
					batchChangeID *int64
				}{
					{
						repoID:        repo1.ID,
						listLen:       1,
						batchChangeID: &cs[0].ID,
					},
					{
						repoID:        repo2.ID,
						listLen:       2,
						batchChangeID: &cs[1].ID,
					},
					{
						repoID:        repo3.ID,
						listLen:       0,
						batchChangeID: nil,
					},
				}

				for i, tc := range tcs {
					t.Run(strconv.Itoa(i), func(t *testing.T) {
						opts := ListBatchChangesOpts{RepoID: tc.repoID}

						ts, next, err := s.ListBatchChanges(ctx, opts)
						if err != nil {
							t.Fatal(err)
						}

						if have, want := next, int64(0); have != want {
							t.Fatalf("opts: %+v: have next %v, want %v", opts, have, want)
						}

						if len(ts) != tc.listLen {
							t.Fatalf("listed the wrong number of batch changes: have %v, want %v", len(ts), tc.listLen)
						}

						if len(ts) > 0 {
							have, want := ts[0].ID, *tc.batchChangeID
							if have != want {
								t.Fatalf("listed batch change with id %d, wanted %d", have, want)
							}
						}
					})
				}
			}
		})

		// The batch changes store returns the batch changes in reversed order.
		reversedBatchChanges := make([]*btypes.BatchChange, len(cs))
		for i, c := range cs {
			reversedBatchChanges[len(cs)-i-1] = c
		}

		t.Run("With Limit", func(t *testing.T) {
			for i := 1; i <= len(reversedBatchChanges); i++ {
				cs, next, err := s.ListBatchChanges(ctx, ListBatchChangesOpts{LimitOpts: LimitOpts{Limit: i}})
				if err != nil {
					t.Fatal(err)
				}

				{
					have, want := next, int64(0)
					if i < len(reversedBatchChanges) {
						want = reversedBatchChanges[i].ID
					}

					if have != want {
						t.Fatalf("limit: %v: have next %v, want %v", i, have, want)
					}
				}

				{
					have, want := cs, reversedBatchChanges[:i]
					if len(have) != len(want) {
						t.Fatalf("listed %d batch changes, want: %d", len(have), len(want))
					}

					if diff := cmp.Diff(have, want); diff != "" {
						t.Fatal(diff)
					}
				}
			}
		})

		t.Run("With Cursor", func(t *testing.T) {
			var cursor int64
			for i := 1; i <= len(reversedBatchChanges); i++ {
				opts := ListBatchChangesOpts{Cursor: cursor, LimitOpts: LimitOpts{Limit: 1}}
				have, next, err := s.ListBatchChanges(ctx, opts)
				if err != nil {
					t.Fatal(err)
				}

				want := reversedBatchChanges[i-1 : i]
				if diff := cmp.Diff(have, want); diff != "" {
					t.Fatalf("opts: %+v, diff: %s", opts, diff)
				}

				cursor = next
			}
		})

		filterTests := []struct {
			name  string
			state btypes.BatchChangeState
			want  []*btypes.BatchChange
		}{
			{
				name:  "Any",
				state: btypes.BatchChangeStateAny,
				want:  reversedBatchChanges,
			},
			{
				name:  "Closed",
				state: btypes.BatchChangeStateClosed,
				want:  reversedBatchChanges[:len(reversedBatchChanges)-1],
			},
			{
				name:  "Open",
				state: btypes.BatchChangeStateOpen,
				want:  cs[0:1],
			},
		}

		for _, tc := range filterTests {
			t.Run("ListBatchChanges State "+tc.name, func(t *testing.T) {
				have, _, err := s.ListBatchChanges(ctx, ListBatchChangesOpts{State: tc.state})
				if err != nil {
					t.Fatal(err)
				}
				if diff := cmp.Diff(have, tc.want); diff != "" {
					t.Fatal(diff)
				}
			})
		}

		t.Run("ListBatchChanges OnlyForAuthor set", func(t *testing.T) {
			for _, c := range cs {
				have, next, err := s.ListBatchChanges(ctx, ListBatchChangesOpts{InitialApplierID: c.InitialApplierID})
				if err != nil {
					t.Fatal(err)
				}
				if next != 0 {
					t.Fatal("Next value was true, but false expected")
				}
				if have, want := len(have), 1; have != want {
					t.Fatalf("Incorrect number of batch changes returned, want=%d have=%d", want, have)
				}
				if diff := cmp.Diff(have[0], c); diff != "" {
					t.Fatal(diff)
				}
			}
		})

		t.Run("ListBatchChanges by NamespaceUserID", func(t *testing.T) {
			for _, c := range cs {
				if c.NamespaceUserID == 0 {
					continue
				}
				opts := ListBatchChangesOpts{NamespaceUserID: c.NamespaceUserID}
				have, _, err := s.ListBatchChanges(ctx, opts)
				if err != nil {
					t.Fatal(err)
				}

				for _, haveBatchChange := range have {
					if have, want := haveBatchChange.NamespaceUserID, opts.NamespaceUserID; have != want {
						t.Fatalf("batch change has wrong NamespaceUserID. want=%d, have=%d", want, have)
					}
				}
			}
		})

		t.Run("ListBatchChanges by NamespaceOrgID", func(t *testing.T) {
			for _, c := range cs {
				if c.NamespaceOrgID == 0 {
					continue
				}
				opts := ListBatchChangesOpts{NamespaceOrgID: c.NamespaceOrgID}
				have, _, err := s.ListBatchChanges(ctx, opts)
				if err != nil {
					t.Fatal(err)
				}

				for _, haveBatchChange := range have {
					if have, want := haveBatchChange.NamespaceOrgID, opts.NamespaceOrgID; have != want {
						t.Fatalf("batch change has wrong NamespaceOrgID. want=%d, have=%d", want, have)
					}
				}
			}
		})
	})

	t.Run("Update", func(t *testing.T) {
		for _, c := range cs {
			c.Name += "-updated"
			c.Description += "-updated"
			c.InitialApplierID++
			c.ClosedAt = c.ClosedAt.Add(5 * time.Second)

			if c.NamespaceUserID != 0 {
				c.NamespaceUserID++
			}

			if c.NamespaceOrgID != 0 {
				c.NamespaceOrgID++
			}

			clock.Add(1 * time.Second)

			want := c
			want.UpdatedAt = clock.Now()

			have := c.Clone()
			if err := s.UpdateBatchChange(ctx, have); err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(have, want); diff != "" {
				t.Fatal(diff)
			}
		}
	})

	t.Run("Get", func(t *testing.T) {
		t.Run("ByID", func(t *testing.T) {
			want := cs[0]
			opts := GetBatchChangeOpts{ID: want.ID}

			have, err := s.GetBatchChange(ctx, opts)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(have, want); diff != "" {
				t.Fatal(diff)
			}
		})

		t.Run("ByBatchSpecID", func(t *testing.T) {
			want := cs[0]
			opts := GetBatchChangeOpts{BatchSpecID: want.BatchSpecID}

			have, err := s.GetBatchChange(ctx, opts)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(have, want); diff != "" {
				t.Fatal(diff)
			}
		})

		t.Run("ByName", func(t *testing.T) {
			want := cs[0]

			have, err := s.GetBatchChange(ctx, GetBatchChangeOpts{Name: want.Name})
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(have, want); diff != "" {
				t.Fatal(diff)
			}
		})

		t.Run("ByNamespaceUserID", func(t *testing.T) {
			for _, c := range cs {
				if c.NamespaceUserID == 0 {
					continue
				}

				want := c
				opts := GetBatchChangeOpts{NamespaceUserID: c.NamespaceUserID}

				have, err := s.GetBatchChange(ctx, opts)
				if err != nil {
					t.Fatal(err)
				}

				if diff := cmp.Diff(have, want); diff != "" {
					t.Fatal(diff)
				}
			}
		})

		t.Run("ByNamespaceOrgID", func(t *testing.T) {
			for _, c := range cs {
				if c.NamespaceOrgID == 0 {
					continue
				}

				want := c
				opts := GetBatchChangeOpts{NamespaceOrgID: c.NamespaceOrgID}

				have, err := s.GetBatchChange(ctx, opts)
				if err != nil {
					t.Fatal(err)
				}

				if diff := cmp.Diff(have, want); diff != "" {
					t.Fatal(diff)
				}
			}
		})

		t.Run("NoResults", func(t *testing.T) {
			opts := GetBatchChangeOpts{ID: 0xdeadbeef}

			_, have := s.GetBatchChange(ctx, opts)
			want := ErrNoResults

			if have != want {
				t.Fatalf("have err %v, want %v", have, want)
			}
		})
	})

	t.Run("GetBatchChangeDiffStat", func(t *testing.T) {
		userID := ct.CreateTestUser(t, s.DB(), false).ID
		userCtx := actor.WithActor(ctx, actor.FromUser(userID))
		repoStore := database.ReposWith(s)
		esStore := database.ExternalServicesWith(s)
		repo := ct.TestRepo(t, esStore, extsvc.KindGitHub)
		repo.Private = true
		if err := repoStore.Create(ctx, repo); err != nil {
			t.Fatal(err)
		}

		batchChangeID := cs[0].ID
		var testDiffStatCount int32 = 10
		ct.CreateChangeset(t, ctx, s, ct.TestChangesetOpts{
			Repo:            repo.ID,
			BatchChanges:    []btypes.BatchChangeAssoc{{BatchChangeID: batchChangeID}},
			DiffStatAdded:   testDiffStatCount,
			DiffStatChanged: testDiffStatCount,
			DiffStatDeleted: testDiffStatCount,
		})

		{
			want := &diff.Stat{
				Added:   testDiffStatCount,
				Changed: testDiffStatCount,
				Deleted: testDiffStatCount,
			}
			opts := GetBatchChangeDiffStatOpts{BatchChangeID: batchChangeID}
			have, err := s.GetBatchChangeDiffStat(userCtx, opts)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(have, want); diff != "" {
				t.Fatal(diff)
			}
		}

		// Now revoke repo access, and check that we don't see it in the diff stat anymore.
		ct.MockRepoPermissions(t, s.DB(), 0, repo.ID)
		{
			want := &diff.Stat{
				Added:   0,
				Changed: 0,
				Deleted: 0,
			}
			opts := GetBatchChangeDiffStatOpts{BatchChangeID: batchChangeID}
			have, err := s.GetBatchChangeDiffStat(userCtx, opts)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(have, want); diff != "" {
				t.Fatal(diff)
			}
		}
	})

	t.Run("GetRepoDiffStat", func(t *testing.T) {
		userID := ct.CreateTestUser(t, s.DB(), false).ID
		userCtx := actor.WithActor(ctx, actor.FromUser(userID))
		repoStore := database.ReposWith(s)
		esStore := database.ExternalServicesWith(s)
		repo1 := ct.TestRepo(t, esStore, extsvc.KindGitHub)
		repo2 := ct.TestRepo(t, esStore, extsvc.KindGitHub)
		repo3 := ct.TestRepo(t, esStore, extsvc.KindGitHub)
		if err := repoStore.Create(ctx, repo1, repo2, repo3); err != nil {
			t.Fatal(err)
		}

		batchChangeID := cs[0].ID
		var testDiffStatCount1 int32 = 10
		var testDiffStatCount2 int32 = 20

		// two changesets on the first repo
		ct.CreateChangeset(t, ctx, s, ct.TestChangesetOpts{
			Repo:            repo1.ID,
			BatchChange:     batchChangeID,
			DiffStatAdded:   testDiffStatCount1,
			DiffStatChanged: testDiffStatCount1,
			DiffStatDeleted: testDiffStatCount1,
		})
		ct.CreateChangeset(t, ctx, s, ct.TestChangesetOpts{
			Repo:            repo1.ID,
			BatchChange:     batchChangeID,
			DiffStatAdded:   testDiffStatCount2,
			DiffStatChanged: 0,
			DiffStatDeleted: testDiffStatCount2,
		})

		// one changeset on the second repo
		ct.CreateChangeset(t, ctx, s, ct.TestChangesetOpts{
			Repo:            repo2.ID,
			BatchChange:     batchChangeID,
			DiffStatAdded:   testDiffStatCount2,
			DiffStatChanged: testDiffStatCount2,
			DiffStatDeleted: testDiffStatCount2,
		})

		// no changesets on the third repo

		{
			tcs := []struct {
				repoID api.RepoID
				want   *diff.Stat
			}{
				{
					repoID: repo1.ID,
					want: &diff.Stat{
						Added:   testDiffStatCount1 + testDiffStatCount2,
						Changed: testDiffStatCount1,
						Deleted: testDiffStatCount1 + testDiffStatCount2,
					},
				},
				{
					repoID: repo2.ID,
					want: &diff.Stat{
						Added:   testDiffStatCount2,
						Changed: testDiffStatCount2,
						Deleted: testDiffStatCount2,
					},
				},
				{
					repoID: repo3.ID,
					want: &diff.Stat{
						Added:   0,
						Changed: 0,
						Deleted: 0,
					},
				},
			}

			for i, tc := range tcs {
				t.Run(strconv.Itoa(i), func(t *testing.T) {
					have, err := s.GetRepoDiffStat(userCtx, tc.repoID)
					if err != nil {
						t.Fatal(err)
					}

					if diff := cmp.Diff(have, tc.want); diff != "" {
						t.Errorf("wrong diff returned. have=%+v want=%+v", have, tc.want)
					}
				})
			}

		}

		// Now revoke repo1 access, and check that we don't get a diff stat for it anymore.
		ct.MockRepoPermissions(t, s.DB(), 0, repo1.ID)
		{
			want := &diff.Stat{
				Added:   0,
				Changed: 0,
				Deleted: 0,
			}
			have, err := s.GetRepoDiffStat(userCtx, repo1.ID)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(have, want); diff != "" {
				t.Fatal(diff)
			}
		}
	})

	t.Run("Delete", func(t *testing.T) {
		for i := range cs {
			err := s.DeleteBatchChange(ctx, cs[i].ID)
			if err != nil {
				t.Fatal(err)
			}

			count, err := s.CountBatchChanges(ctx, CountBatchChangesOpts{})
			if err != nil {
				t.Fatal(err)
			}

			if have, want := count, len(cs)-(i+1); have != want {
				t.Fatalf("have count: %d, want: %d", have, want)
			}
		}
	})
}

func testUserDeleteCascades(t *testing.T, ctx context.Context, s *Store, clock ct.Clock) {
	orgID := ct.InsertTestOrg(t, s.DB(), "user-delete-cascades")
	user := ct.CreateTestUser(t, s.DB(), false)

	t.Run("User delete", func(t *testing.T) {
		// Set up two batch changes and specs: one in the user's namespace (which
		// should be deleted when the user is hard deleted), and one that is
		// merely created by the user (which should remain).
		ownedSpec := &btypes.BatchSpec{
			NamespaceUserID: user.ID,
			UserID:          user.ID,
		}
		if err := s.CreateBatchSpec(ctx, ownedSpec); err != nil {
			t.Fatal(err)
		}

		unownedSpec := &btypes.BatchSpec{
			NamespaceOrgID: orgID,
			UserID:         user.ID,
		}
		if err := s.CreateBatchSpec(ctx, unownedSpec); err != nil {
			t.Fatal(err)
		}

		ownedBatchChange := &btypes.BatchChange{
			Name:             "owned",
			NamespaceUserID:  user.ID,
			InitialApplierID: user.ID,
			LastApplierID:    user.ID,
			LastAppliedAt:    clock.Now(),
			BatchSpecID:      ownedSpec.ID,
		}
		if err := s.CreateBatchChange(ctx, ownedBatchChange); err != nil {
			t.Fatal(err)
		}

		unownedBatchChange := &btypes.BatchChange{
			Name:             "unowned",
			NamespaceOrgID:   orgID,
			InitialApplierID: user.ID,
			LastApplierID:    user.ID,
			LastAppliedAt:    clock.Now(),
			BatchSpecID:      ownedSpec.ID,
		}
		if err := s.CreateBatchChange(ctx, unownedBatchChange); err != nil {
			t.Fatal(err)
		}

		// Now we soft-delete the user.
		if err := database.UsersWith(s).Delete(ctx, user.ID); err != nil {
			t.Fatal(err)
		}

		var testBatchChangeIsGone = func() {
			// We should now have the unowned batch change still be valid, but the
			// owned batch change should have gone away.
			cs, _, err := s.ListBatchChanges(ctx, ListBatchChangesOpts{})
			if err != nil {
				t.Fatal(err)
			}
			if len(cs) != 1 {
				t.Errorf("unexpected number of batch changes: have %d; want %d", len(cs), 1)
			}
			if cs[0].ID != unownedBatchChange.ID {
				t.Errorf("unexpected batch change: %+v", cs[0])
			}

			// The count of batch changes should also respect it.
			count, err := s.CountBatchChanges(ctx, CountBatchChangesOpts{})
			if err != nil {
				t.Fatal(err)
			}

			if have, want := count, len(cs); have != want {
				t.Fatalf("have count: %d, want: %d", have, want)
			}

			// And getting the batch change by its ID also shouldn't work.
			if _, err := s.GetBatchChange(ctx, GetBatchChangeOpts{ID: ownedBatchChange.ID}); err == nil || err != ErrNoResults {
				t.Fatalf("got invalid error, want=%+v have=%+v", ErrNoResults, err)
			}

			// Both batch specs should still be in place, at least until we add
			// a foreign key constraint to batch_specs.namespace_user_id.
			specs, _, err := s.ListBatchSpecs(ctx, ListBatchSpecsOpts{})
			if err != nil {
				t.Fatal(err)
			}
			if len(specs) != 2 {
				t.Errorf("unexpected number of batch specs: have %d; want %d", len(specs), 2)
			}
		}

		testBatchChangeIsGone()

		// Now we hard-delete the user.
		if err := database.UsersWith(s).HardDelete(ctx, user.ID); err != nil {
			t.Fatal(err)
		}

		testBatchChangeIsGone()
	})
}
