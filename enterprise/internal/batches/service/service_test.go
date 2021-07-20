package service

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/google/go-cmp/cmp"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/batches/sources"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/batches/store"
	ct "github.com/sourcegraph/sourcegraph/enterprise/internal/batches/testing"
	btypes "github.com/sourcegraph/sourcegraph/enterprise/internal/batches/types"
	"github.com/sourcegraph/sourcegraph/internal/actor"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/database/dbtest"
	"github.com/sourcegraph/sourcegraph/internal/errcode"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/auth"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/github"
	"github.com/sourcegraph/sourcegraph/internal/repoupdater"
	"github.com/sourcegraph/sourcegraph/internal/timeutil"
)

func TestServicePermissionLevels(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	ctx := backend.WithAuthzBypass(context.Background())
	db := dbtest.NewDB(t, "")

	s := store.New(db, nil)
	svc := New(s)

	admin := ct.CreateTestUser(t, db, true)
	user := ct.CreateTestUser(t, db, false)
	otherUser := ct.CreateTestUser(t, db, false)

	rs, _ := ct.CreateTestRepos(t, ctx, db, 1)

	createTestData := func(t *testing.T, s *store.Store, svc *Service, author int32) (*btypes.BatchChange, *btypes.Changeset, *btypes.BatchSpec) {
		spec := testBatchSpec(author)
		if err := s.CreateBatchSpec(ctx, spec); err != nil {
			t.Fatal(err)
		}

		batchChange := testBatchChange(author, spec)
		if err := s.CreateBatchChange(ctx, batchChange); err != nil {
			t.Fatal(err)
		}

		changeset := testChangeset(rs[0].ID, batchChange.ID, btypes.ChangesetExternalStateOpen)
		if err := s.CreateChangeset(ctx, changeset); err != nil {
			t.Fatal(err)
		}

		return batchChange, changeset, spec
	}

	assertAuthError := func(t *testing.T, err error) {
		t.Helper()

		if err == nil {
			t.Fatalf("expected error. got none")
		}
		if err != nil {
			if !errors.HasType(err, &backend.InsufficientAuthorizationError{}) {
				t.Fatalf("wrong error: %s (%T)", err, err)
			}
		}
	}

	assertNoAuthError := func(t *testing.T, err error) {
		t.Helper()

		// Ignore other errors, we only want to check whether it's an auth error
		if errors.HasType(err, &backend.InsufficientAuthorizationError{}) {
			t.Fatalf("got auth error")
		}
	}

	tests := []struct {
		name              string
		batchChangeAuthor int32
		currentUser       int32
		assertFunc        func(t *testing.T, err error)
	}{
		{
			name:              "unauthorized user",
			batchChangeAuthor: user.ID,
			currentUser:       otherUser.ID,
			assertFunc:        assertAuthError,
		},
		{
			name:              "batch change author",
			batchChangeAuthor: user.ID,
			currentUser:       user.ID,
			assertFunc:        assertNoAuthError,
		},

		{
			name:              "site-admin",
			batchChangeAuthor: user.ID,
			currentUser:       admin.ID,
			assertFunc:        assertNoAuthError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			batchChange, changeset, batchSpec := createTestData(t, s, svc, tc.batchChangeAuthor)
			// Fresh context.Background() because the previous one is wrapped in AuthzBypas
			currentUserCtx := actor.WithActor(context.Background(), actor.FromUser(tc.currentUser))

			t.Run("EnqueueChangesetSync", func(t *testing.T) {
				// The cases that don't result in auth errors will fall through
				// to call repoupdater.EnqueueChangesetSync, so we need to
				// ensure we mock that call to avoid unexpected network calls.
				repoupdater.MockEnqueueChangesetSync = func(ctx context.Context, ids []int64) error {
					return nil
				}
				t.Cleanup(func() { repoupdater.MockEnqueueChangesetSync = nil })

				err := svc.EnqueueChangesetSync(currentUserCtx, changeset.ID)
				tc.assertFunc(t, err)
			})

			t.Run("ReenqueueChangeset", func(t *testing.T) {
				_, _, err := svc.ReenqueueChangeset(currentUserCtx, changeset.ID)
				tc.assertFunc(t, err)
			})

			t.Run("CloseBatchChange", func(t *testing.T) {
				_, err := svc.CloseBatchChange(currentUserCtx, batchChange.ID, false)
				tc.assertFunc(t, err)
			})

			t.Run("DeleteBatchChange", func(t *testing.T) {
				err := svc.DeleteBatchChange(currentUserCtx, batchChange.ID)
				tc.assertFunc(t, err)
			})

			t.Run("MoveBatchChange", func(t *testing.T) {
				_, err := svc.MoveBatchChange(currentUserCtx, MoveBatchChangeOpts{
					BatchChangeID: batchChange.ID,
					NewName:       "foobar2",
				})
				tc.assertFunc(t, err)
			})

			t.Run("ApplyBatchChange", func(t *testing.T) {
				_, err := svc.ApplyBatchChange(currentUserCtx, ApplyBatchChangeOpts{
					BatchSpecRandID: batchSpec.RandID,
				})
				tc.assertFunc(t, err)
			})

			t.Run("CreateChangesetJobs", func(t *testing.T) {
				_, err := svc.CreateChangesetJobs(currentUserCtx, batchChange.ID, []int64{changeset.ID}, btypes.ChangesetJobTypeComment, btypes.ChangesetJobCommentPayload{Message: "test"}, store.ListChangesetsOpts{})
				tc.assertFunc(t, err)
			})
		})
	}
}

func TestService(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	ctx := backend.WithAuthzBypass(context.Background())
	db := dbtest.NewDB(t, "")

	admin := ct.CreateTestUser(t, db, true)
	user := ct.CreateTestUser(t, db, false)
	adminCtx := actor.WithActor(context.Background(), actor.FromUser(admin.ID))
	userCtx := actor.WithActor(context.Background(), actor.FromUser(user.ID))

	now := timeutil.Now()
	clock := func() time.Time { return now }

	s := store.NewWithClock(db, nil, clock)
	rs, _ := ct.CreateTestRepos(t, ctx, db, 4)

	fakeSource := &sources.FakeChangesetSource{}
	sourcer := sources.NewFakeSourcer(nil, fakeSource)

	svc := New(s)
	svc.sourcer = sourcer

	t.Run("DeleteBatchChange", func(t *testing.T) {
		spec := testBatchSpec(admin.ID)
		if err := s.CreateBatchSpec(ctx, spec); err != nil {
			t.Fatal(err)
		}

		batchChange := testBatchChange(admin.ID, spec)
		if err := s.CreateBatchChange(ctx, batchChange); err != nil {
			t.Fatal(err)
		}
		if err := svc.DeleteBatchChange(ctx, batchChange.ID); err != nil {
			t.Fatalf("batch change not deleted: %s", err)
		}

		_, err := s.GetBatchChange(ctx, store.GetBatchChangeOpts{ID: batchChange.ID})
		if err != nil && err != store.ErrNoResults {
			t.Fatalf("want batch change to be deleted, but was not: %e", err)
		}
	})

	t.Run("CloseBatchChange", func(t *testing.T) {
		createBatchChange := func(t *testing.T) *btypes.BatchChange {
			t.Helper()

			spec := testBatchSpec(admin.ID)
			if err := s.CreateBatchSpec(ctx, spec); err != nil {
				t.Fatal(err)
			}

			batchChange := testBatchChange(admin.ID, spec)
			if err := s.CreateBatchChange(ctx, batchChange); err != nil {
				t.Fatal(err)
			}
			return batchChange
		}

		closeConfirm := func(t *testing.T, c *btypes.BatchChange, closeChangesets bool) {
			t.Helper()

			closedBatchChange, err := svc.CloseBatchChange(adminCtx, c.ID, closeChangesets)
			if err != nil {
				t.Fatalf("batch change not closed: %s", err)
			}
			if !closedBatchChange.ClosedAt.Equal(now) {
				t.Fatalf("batch change ClosedAt is zero")
			}

			if !closeChangesets {
				return
			}

			cs, _, err := s.ListChangesets(ctx, store.ListChangesetsOpts{
				OwnedByBatchChangeID: c.ID,
			})
			if err != nil {
				t.Fatalf("listing changesets failed: %s", err)
			}
			for _, c := range cs {
				if !c.Closing {
					t.Errorf("changeset should be Closing, but is not")
				}

				if have, want := c.ReconcilerState, btypes.ReconcilerStateQueued; have != want {
					t.Errorf("changeset ReconcilerState wrong. want=%s, have=%s", want, have)
				}
			}
		}

		t.Run("no changesets", func(t *testing.T) {
			batchChange := createBatchChange(t)
			closeConfirm(t, batchChange, false)
		})

		t.Run("changesets", func(t *testing.T) {
			batchChange := createBatchChange(t)

			changeset1 := testChangeset(rs[0].ID, batchChange.ID, btypes.ChangesetExternalStateOpen)
			changeset1.ReconcilerState = btypes.ReconcilerStateCompleted
			if err := s.CreateChangeset(ctx, changeset1); err != nil {
				t.Fatal(err)
			}

			changeset2 := testChangeset(rs[1].ID, batchChange.ID, btypes.ChangesetExternalStateOpen)
			changeset2.ReconcilerState = btypes.ReconcilerStateCompleted
			if err := s.CreateChangeset(ctx, changeset2); err != nil {
				t.Fatal(err)
			}

			closeConfirm(t, batchChange, true)
		})
	})

	t.Run("EnqueueChangesetSync", func(t *testing.T) {
		spec := testBatchSpec(admin.ID)
		if err := s.CreateBatchSpec(ctx, spec); err != nil {
			t.Fatal(err)
		}

		batchChange := testBatchChange(admin.ID, spec)
		if err := s.CreateBatchChange(ctx, batchChange); err != nil {
			t.Fatal(err)
		}

		changeset := testChangeset(rs[0].ID, batchChange.ID, btypes.ChangesetExternalStateOpen)
		if err := s.CreateChangeset(ctx, changeset); err != nil {
			t.Fatal(err)
		}

		called := false
		repoupdater.MockEnqueueChangesetSync = func(ctx context.Context, ids []int64) error {
			if len(ids) != 1 && ids[0] != changeset.ID {
				t.Fatalf("MockEnqueueChangesetSync received wrong ids: %+v", ids)
			}
			called = true
			return nil
		}
		t.Cleanup(func() { repoupdater.MockEnqueueChangesetSync = nil })

		if err := svc.EnqueueChangesetSync(ctx, changeset.ID); err != nil {
			t.Fatal(err)
		}

		if !called {
			t.Fatal("MockEnqueueChangesetSync not called")
		}

		// rs[0] is filtered out
		ct.MockRepoPermissions(t, db, user.ID, rs[1].ID, rs[2].ID, rs[3].ID)

		// should result in a not found error
		if err := svc.EnqueueChangesetSync(ctx, changeset.ID); !errcode.IsNotFound(err) {
			t.Fatalf("expected not-found error but got %v", err)
		}
	})

	t.Run("ReenqueueChangeset", func(t *testing.T) {
		spec := testBatchSpec(admin.ID)
		if err := s.CreateBatchSpec(ctx, spec); err != nil {
			t.Fatal(err)
		}

		batchChange := testBatchChange(admin.ID, spec)
		if err := s.CreateBatchChange(ctx, batchChange); err != nil {
			t.Fatal(err)
		}

		changeset := testChangeset(rs[0].ID, batchChange.ID, btypes.ChangesetExternalStateOpen)
		if err := s.CreateChangeset(ctx, changeset); err != nil {
			t.Fatal(err)
		}

		ct.SetChangesetFailed(t, ctx, s, changeset)

		if _, _, err := svc.ReenqueueChangeset(ctx, changeset.ID); err != nil {
			t.Fatal(err)
		}

		ct.ReloadAndAssertChangeset(t, ctx, s, changeset, ct.ChangesetAssertions{
			Repo:          rs[0].ID,
			ExternalState: btypes.ChangesetExternalStateOpen,
			ExternalID:    "ext-id-5",
			AttachedTo:    []int64{batchChange.ID},

			// The important fields:
			ReconcilerState: btypes.ReconcilerStateQueued,
			NumResets:       0,
			NumFailures:     0,
			FailureMessage:  nil,
		})

		// rs[0] is filtered out
		ct.MockRepoPermissions(t, db, user.ID, rs[1].ID, rs[2].ID, rs[3].ID)

		// should result in a not found error
		if _, _, err := svc.ReenqueueChangeset(ctx, changeset.ID); !errcode.IsNotFound(err) {
			t.Fatalf("expected not-found error but got %v", err)
		}
	})

	t.Run("CreateBatchSpec", func(t *testing.T) {
		changesetSpecs := make([]*btypes.ChangesetSpec, 0, len(rs))
		changesetSpecRandIDs := make([]string, 0, len(rs))
		for _, r := range rs {
			cs := &btypes.ChangesetSpec{RepoID: r.ID, UserID: admin.ID}
			if err := s.CreateChangesetSpec(ctx, cs); err != nil {
				t.Fatal(err)
			}
			changesetSpecs = append(changesetSpecs, cs)
			changesetSpecRandIDs = append(changesetSpecRandIDs, cs.RandID)
		}

		t.Run("success", func(t *testing.T) {
			opts := CreateBatchSpecOpts{
				NamespaceUserID:      admin.ID,
				RawSpec:              ct.TestRawBatchSpec,
				ChangesetSpecRandIDs: changesetSpecRandIDs,
			}

			spec, err := svc.CreateBatchSpec(adminCtx, opts)
			if err != nil {
				t.Fatal(err)
			}

			if spec.ID == 0 {
				t.Fatalf("BatchSpec ID is 0")
			}

			if have, want := spec.UserID, admin.ID; have != want {
				t.Fatalf("UserID is %d, want %d", have, want)
			}

			var wantFields btypes.BatchSpecFields
			if err := json.Unmarshal([]byte(spec.RawSpec), &wantFields); err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(wantFields, spec.Spec); diff != "" {
				t.Fatalf("wrong spec fields (-want +got):\n%s", diff)
			}

			for _, cs := range changesetSpecs {
				cs2, err := s.GetChangesetSpec(ctx, store.GetChangesetSpecOpts{ID: cs.ID})
				if err != nil {
					t.Fatal(err)
				}

				if have, want := cs2.BatchSpecID, spec.ID; have != want {
					t.Fatalf("changesetSpec has wrong BatchSpec. want=%d, have=%d", want, have)
				}
			}
		})

		t.Run("success with YAML raw spec", func(t *testing.T) {
			opts := CreateBatchSpecOpts{
				NamespaceUserID: admin.ID,
				RawSpec:         ct.TestRawBatchSpecYAML,
			}

			spec, err := svc.CreateBatchSpec(adminCtx, opts)
			if err != nil {
				t.Fatal(err)
			}

			if spec.ID == 0 {
				t.Fatalf("BatchSpec ID is 0")
			}

			var wantFields btypes.BatchSpecFields
			if err := json.Unmarshal([]byte(ct.TestRawBatchSpec), &wantFields); err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(wantFields, spec.Spec); diff != "" {
				t.Fatalf("wrong spec fields (-want +got):\n%s", diff)
			}
		})

		t.Run("missing repository permissions", func(t *testing.T) {
			ct.MockRepoPermissions(t, db, user.ID)

			opts := CreateBatchSpecOpts{
				NamespaceUserID:      user.ID,
				RawSpec:              ct.TestRawBatchSpec,
				ChangesetSpecRandIDs: changesetSpecRandIDs,
			}

			if _, err := svc.CreateBatchSpec(userCtx, opts); !errcode.IsNotFound(err) {
				t.Fatalf("expected not-found error but got %s", err)
			}
		})

		t.Run("invalid changesetspec id", func(t *testing.T) {
			containsInvalidID := []string{changesetSpecRandIDs[0], "foobar"}
			opts := CreateBatchSpecOpts{
				NamespaceUserID:      admin.ID,
				RawSpec:              ct.TestRawBatchSpec,
				ChangesetSpecRandIDs: containsInvalidID,
			}

			if _, err := svc.CreateBatchSpec(adminCtx, opts); !errcode.IsNotFound(err) {
				t.Fatalf("expected not-found error but got %s", err)
			}
		})

		t.Run("namespace user is not admin and not creator", func(t *testing.T) {
			opts := CreateBatchSpecOpts{
				NamespaceUserID: admin.ID,
				RawSpec:         ct.TestRawBatchSpecYAML,
			}

			_, err := svc.CreateBatchSpec(userCtx, opts)
			if !errcode.IsUnauthorized(err) {
				t.Fatalf("expected unauthorized error but got %s", err)
			}

			// Try again as admin
			opts.NamespaceUserID = user.ID

			_, err = svc.CreateBatchSpec(adminCtx, opts)
			if err != nil {
				t.Fatalf("expected no error but got %s", err)
			}
		})

		t.Run("missing access to namespace org", func(t *testing.T) {
			orgID := ct.InsertTestOrg(t, db, "test-org")

			opts := CreateBatchSpecOpts{
				NamespaceOrgID:       orgID,
				RawSpec:              ct.TestRawBatchSpec,
				ChangesetSpecRandIDs: changesetSpecRandIDs,
			}

			_, err := svc.CreateBatchSpec(userCtx, opts)
			if have, want := err, backend.ErrNotAnOrgMember; have != want {
				t.Fatalf("expected %s error but got %s", want, have)
			}

			// Create org membership and try again
			if _, err := database.OrgMembers(db).Create(ctx, orgID, user.ID); err != nil {
				t.Fatal(err)
			}

			_, err = svc.CreateBatchSpec(userCtx, opts)
			if err != nil {
				t.Fatalf("expected no error but got %s", err)
			}
		})

		t.Run("no side-effects if no changeset spec IDs are given", func(t *testing.T) {
			// We already have ChangesetSpecs in the database. Here we
			// want to make sure that the new BatchSpec is created,
			// without accidently attaching the existing ChangesetSpecs.
			opts := CreateBatchSpecOpts{
				NamespaceUserID:      admin.ID,
				RawSpec:              ct.TestRawBatchSpec,
				ChangesetSpecRandIDs: []string{},
			}

			spec, err := svc.CreateBatchSpec(adminCtx, opts)
			if err != nil {
				t.Fatal(err)
			}

			countOpts := store.CountChangesetSpecsOpts{BatchSpecID: spec.ID}
			count, err := s.CountChangesetSpecs(adminCtx, countOpts)
			if err != nil {
				return
			}
			if count != 0 {
				t.Fatalf("want no changeset specs attached to batch spec, but have %d", count)
			}
		})
	})

	t.Run("CreateChangesetSpec", func(t *testing.T) {
		repo := rs[0]
		rawSpec := ct.NewRawChangesetSpecGitBranch(graphqlbackend.MarshalRepositoryID(repo.ID), "d34db33f")

		t.Run("success", func(t *testing.T) {
			spec, err := svc.CreateChangesetSpec(ctx, rawSpec, admin.ID)
			if err != nil {
				t.Fatal(err)
			}

			if spec.ID == 0 {
				t.Fatalf("ChangesetSpec ID is 0")
			}

			wantFields := &btypes.ChangesetSpecDescription{}
			if err := json.Unmarshal([]byte(spec.RawSpec), wantFields); err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(wantFields, spec.Spec); diff != "" {
				t.Fatalf("wrong spec fields (-want +got):\n%s", diff)
			}

			wantDiffStat := *ct.ChangesetSpecDiffStat
			if diff := cmp.Diff(wantDiffStat, spec.DiffStat()); diff != "" {
				t.Fatalf("wrong diff stat (-want +got):\n%s", diff)
			}
		})

		t.Run("invalid raw spec", func(t *testing.T) {
			invalidRaw := `{"externalComputer": "beepboop"}`
			_, err := svc.CreateChangesetSpec(ctx, invalidRaw, admin.ID)
			if err == nil {
				t.Fatal("expected error but got nil")
			}

			haveErr := fmt.Sprintf("%v", err)
			wantErr := "4 errors occurred:\n\t* Must validate one and only one schema (oneOf)\n\t* baseRepository is required\n\t* externalID is required\n\t* Additional property externalComputer is not allowed\n\n"
			if diff := cmp.Diff(wantErr, haveErr); diff != "" {
				t.Fatalf("unexpected error (-want +got):\n%s", diff)
			}
		})

		t.Run("missing repository permissions", func(t *testing.T) {
			ct.MockRepoPermissions(t, db, user.ID, rs[1].ID, rs[2].ID, rs[3].ID)

			_, err := svc.CreateChangesetSpec(ctx, rawSpec, admin.ID)
			if !errcode.IsNotFound(err) {
				t.Fatalf("expected not-found error but got %v", err)
			}
		})
	})

	t.Run("ApplyBatchChange", func(t *testing.T) {
		// See TestServiceApplyBatchChange
	})

	t.Run("MoveBatchChange", func(t *testing.T) {
		createBatchChange := func(t *testing.T, name string, authorID, userID, orgID int32) *btypes.BatchChange {
			t.Helper()

			spec := &btypes.BatchSpec{
				UserID:          authorID,
				NamespaceUserID: userID,
				NamespaceOrgID:  orgID,
			}

			if err := s.CreateBatchSpec(ctx, spec); err != nil {
				t.Fatal(err)
			}

			c := &btypes.BatchChange{
				InitialApplierID: authorID,
				NamespaceUserID:  userID,
				NamespaceOrgID:   orgID,
				Name:             name,
				LastApplierID:    authorID,
				LastAppliedAt:    time.Now(),
				BatchSpecID:      spec.ID,
			}

			if err := s.CreateBatchChange(ctx, c); err != nil {
				t.Fatal(err)
			}

			return c
		}

		t.Run("new name", func(t *testing.T) {
			batchChange := createBatchChange(t, "old-name", admin.ID, admin.ID, 0)

			opts := MoveBatchChangeOpts{BatchChangeID: batchChange.ID, NewName: "new-name"}
			moved, err := svc.MoveBatchChange(ctx, opts)
			if err != nil {
				t.Fatal(err)
			}

			if have, want := moved.Name, opts.NewName; have != want {
				t.Fatalf("wrong name. want=%q, have=%q", want, have)
			}
		})

		t.Run("new user namespace", func(t *testing.T) {
			batchChange := createBatchChange(t, "old-name", admin.ID, admin.ID, 0)

			user2 := ct.CreateTestUser(t, db, false)

			opts := MoveBatchChangeOpts{BatchChangeID: batchChange.ID, NewNamespaceUserID: user2.ID}
			moved, err := svc.MoveBatchChange(ctx, opts)
			if err != nil {
				t.Fatal(err)
			}

			if have, want := moved.NamespaceUserID, opts.NewNamespaceUserID; have != want {
				t.Fatalf("wrong NamespaceUserID. want=%d, have=%d", want, have)
			}

			if have, want := moved.NamespaceOrgID, opts.NewNamespaceOrgID; have != want {
				t.Fatalf("wrong NamespaceOrgID. want=%d, have=%d", want, have)
			}
		})

		t.Run("new user namespace but current user is not admin", func(t *testing.T) {
			batchChange := createBatchChange(t, "old-name", user.ID, user.ID, 0)

			user2 := ct.CreateTestUser(t, db, false)

			opts := MoveBatchChangeOpts{BatchChangeID: batchChange.ID, NewNamespaceUserID: user2.ID}

			_, err := svc.MoveBatchChange(userCtx, opts)
			if !errcode.IsUnauthorized(err) {
				t.Fatalf("expected unauthorized error but got %s", err)
			}
		})

		t.Run("new org namespace", func(t *testing.T) {
			batchChange := createBatchChange(t, "old-name", admin.ID, admin.ID, 0)

			orgID := ct.InsertTestOrg(t, db, "org")

			opts := MoveBatchChangeOpts{BatchChangeID: batchChange.ID, NewNamespaceOrgID: orgID}
			moved, err := svc.MoveBatchChange(ctx, opts)
			if err != nil {
				t.Fatal(err)
			}

			if have, want := moved.NamespaceUserID, opts.NewNamespaceUserID; have != want {
				t.Fatalf("wrong NamespaceUserID. want=%d, have=%d", want, have)
			}

			if have, want := moved.NamespaceOrgID, opts.NewNamespaceOrgID; have != want {
				t.Fatalf("wrong NamespaceOrgID. want=%d, have=%d", want, have)
			}
		})

		t.Run("new org namespace but current user is missing access", func(t *testing.T) {
			batchChange := createBatchChange(t, "old-name", user.ID, user.ID, 0)

			orgID := ct.InsertTestOrg(t, db, "org-no-access")

			opts := MoveBatchChangeOpts{BatchChangeID: batchChange.ID, NewNamespaceOrgID: orgID}

			_, err := svc.MoveBatchChange(userCtx, opts)
			if have, want := err, backend.ErrNotAnOrgMember; have != want {
				t.Fatalf("expected %s error but got %s", want, have)
			}
		})
	})

	t.Run("GetBatchChangeMatchingBatchSpec", func(t *testing.T) {
		batchSpec := ct.CreateBatchSpec(t, ctx, s, "matching-batch-spec", admin.ID)

		haveBatchChange, err := svc.GetBatchChangeMatchingBatchSpec(ctx, batchSpec)
		if err != nil {
			t.Fatalf("unexpected error: %s\n", err)
		}
		if haveBatchChange != nil {
			t.Fatalf("expected batch change to be nil, but is not: %+v\n", haveBatchChange)
		}

		matchingBatchChange := &btypes.BatchChange{
			Name:             batchSpec.Spec.Name,
			Description:      batchSpec.Spec.Description,
			InitialApplierID: admin.ID,
			NamespaceOrgID:   batchSpec.NamespaceOrgID,
			NamespaceUserID:  batchSpec.NamespaceUserID,
			BatchSpecID:      batchSpec.ID,
			LastApplierID:    admin.ID,
			LastAppliedAt:    time.Now(),
		}
		if err := s.CreateBatchChange(ctx, matchingBatchChange); err != nil {
			t.Fatalf("failed to create batch change: %s\n", err)
		}

		haveBatchChange, err = svc.GetBatchChangeMatchingBatchSpec(ctx, batchSpec)
		if err != nil {
			t.Fatalf("unexpected error: %s\n", err)
		}
		if haveBatchChange == nil {
			t.Fatalf("expected to have matching batch change, but got nil")
		}

		if diff := cmp.Diff(matchingBatchChange, haveBatchChange); diff != "" {
			t.Fatalf("wrong batch change was matched (-want +got):\n%s", diff)
		}
	})

	t.Run("GetNewestBatchSpec", func(t *testing.T) {
		older := ct.CreateBatchSpec(t, ctx, s, "superseding", user.ID)
		newer := ct.CreateBatchSpec(t, ctx, s, "superseding", user.ID)

		for name, in := range map[string]*btypes.BatchSpec{
			"older": older,
			"newer": newer,
		} {
			t.Run(name, func(t *testing.T) {
				have, err := svc.GetNewestBatchSpec(ctx, s, in, user.ID)
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}

				if diff := cmp.Diff(newer, have); diff != "" {
					t.Errorf("unexpected newer batch spec (-want +have):\n%s", diff)
				}
			})
		}

		t.Run("different user", func(t *testing.T) {
			have, err := svc.GetNewestBatchSpec(ctx, s, older, admin.ID)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if have != nil {
				t.Errorf("unexpected non-nil batch spec: %+v", have)
			}
		})
	})

	t.Run("FetchUsernameForBitbucketServerToken", func(t *testing.T) {
		fakeSource := &sources.FakeChangesetSource{Username: "my-bbs-username"}
		sourcer := sources.NewFakeSourcer(nil, fakeSource)

		// Create a fresh service for this test as to not mess with state
		// possibly used by other tests.
		testSvc := New(s)
		testSvc.sourcer = sourcer

		rs, _ := ct.CreateBbsTestRepos(t, ctx, db, 1)
		repo := rs[0]

		url := repo.ExternalRepo.ServiceID
		extType := repo.ExternalRepo.ServiceType

		username, err := testSvc.FetchUsernameForBitbucketServerToken(ctx, url, extType, "my-token")
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}

		if !fakeSource.AuthenticatedUsernameCalled {
			t.Errorf("service didn't call AuthenticatedUsername")
		}

		if have, want := username, fakeSource.Username; have != want {
			t.Errorf("wrong username returned. want=%q, have=%q", want, have)
		}
	})

	t.Run("ValidateAuthenticator", func(t *testing.T) {
		t.Run("valid", func(t *testing.T) {
			fakeSource.AuthenticatorIsValid = true
			fakeSource.ValidateAuthenticatorCalled = false
			if err := svc.ValidateAuthenticator(
				ctx,
				"https://github.com/",
				extsvc.TypeGitHub,
				&auth.OAuthBearerToken{Token: "test123"},
			); err != nil {
				t.Fatal(err)
			}
			if !fakeSource.ValidateAuthenticatorCalled {
				t.Fatal("ValidateAuthenticator on Source not called")
			}
		})
		t.Run("invalid", func(t *testing.T) {
			fakeSource.AuthenticatorIsValid = false
			fakeSource.ValidateAuthenticatorCalled = false
			if err := svc.ValidateAuthenticator(
				ctx,
				"https://github.com/",
				extsvc.TypeGitHub,
				&auth.OAuthBearerToken{Token: "test123"},
			); err == nil {
				t.Fatal("unexpected nil-error returned from ValidateAuthenticator")
			}
			if !fakeSource.ValidateAuthenticatorCalled {
				t.Fatal("ValidateAuthenticator on Source not called")
			}
		})
	})

	t.Run("CreateChangesetJobs", func(t *testing.T) {
		spec := testBatchSpec(admin.ID)
		if err := s.CreateBatchSpec(ctx, spec); err != nil {
			t.Fatal(err)
		}

		batchChange := testBatchChange(admin.ID, spec)
		if err := s.CreateBatchChange(ctx, batchChange); err != nil {
			t.Fatal(err)
		}

		t.Run("creates jobs", func(t *testing.T) {
			changeset1 := ct.CreateChangeset(t, ctx, s, ct.TestChangesetOpts{
				Repo:             rs[0].ID,
				PublicationState: btypes.ChangesetPublicationStatePublished,
				BatchChange:      batchChange.ID,
			})
			changeset2 := ct.CreateChangeset(t, ctx, s, ct.TestChangesetOpts{
				Repo:             rs[1].ID,
				PublicationState: btypes.ChangesetPublicationStatePublished,
				BatchChange:      batchChange.ID,
			})
			bulkOperationID, err := svc.CreateChangesetJobs(
				adminCtx,
				batchChange.ID,
				[]int64{changeset1.ID, changeset2.ID},
				btypes.ChangesetJobTypeComment,
				btypes.ChangesetJobCommentPayload{Message: "test"},
				store.ListChangesetsOpts{},
			)
			if err != nil {
				t.Fatal(err)
			}
			// Validate the bulk operation exists.
			if _, err = s.GetBulkOperation(ctx, store.GetBulkOperationOpts{ID: bulkOperationID}); err != nil {
				t.Fatal(err)
			}
		})
		t.Run("changeset not found", func(t *testing.T) {
			changeset := ct.CreateChangeset(t, ctx, s, ct.TestChangesetOpts{
				Repo:        rs[0].ID,
				BatchChange: batchChange.ID,
			})
			_, err := svc.CreateChangesetJobs(
				adminCtx,
				batchChange.ID,
				[]int64{changeset.ID},
				btypes.ChangesetJobTypeComment,
				btypes.ChangesetJobCommentPayload{Message: "test"},
				store.ListChangesetsOpts{
					ReconcilerStates: []btypes.ReconcilerState{btypes.ReconcilerStateCompleted},
				},
			)
			if err != ErrChangesetsForJobNotFound {
				t.Fatalf("wrong error. want=%s, got=%s", ErrChangesetsForJobNotFound, err)
			}
		})
		t.Run("DetachChangesets", func(t *testing.T) {
			spec := testBatchSpec(admin.ID)
			if err := s.CreateBatchSpec(ctx, spec); err != nil {
				t.Fatal(err)
			}

			batchChange := testBatchChange(admin.ID, spec)
			if err := s.CreateBatchChange(ctx, batchChange); err != nil {
				t.Fatal(err)
			}
			t.Run("attached changeset", func(t *testing.T) {
				changeset := ct.CreateChangeset(t, ctx, s, ct.TestChangesetOpts{
					Repo:            rs[0].ID,
					ReconcilerState: btypes.ReconcilerStateCompleted,
					BatchChange:     batchChange.ID,
					IsArchived:      false,
				})
				_, err := svc.CreateChangesetJobs(ctx, batchChange.ID, []int64{changeset.ID}, btypes.ChangesetJobTypeDetach, btypes.ChangesetJobDetachPayload{}, store.ListChangesetsOpts{OnlyArchived: true})
				if err != ErrChangesetsForJobNotFound {
					t.Fatalf("wrong error. want=%s, got=%s", ErrChangesetsForJobNotFound, err)
				}
			})
			t.Run("detached changeset", func(t *testing.T) {
				detachedChangeset := ct.CreateChangeset(t, ctx, s, ct.TestChangesetOpts{
					Repo:            rs[2].ID,
					ReconcilerState: btypes.ReconcilerStateCompleted,
					BatchChanges:    []btypes.BatchChangeAssoc{},
				})
				_, err := svc.CreateChangesetJobs(ctx, batchChange.ID, []int64{detachedChangeset.ID}, btypes.ChangesetJobTypeDetach, btypes.ChangesetJobDetachPayload{}, store.ListChangesetsOpts{OnlyArchived: true})
				if err != ErrChangesetsForJobNotFound {
					t.Fatalf("wrong error. want=%s, got=%s", ErrChangesetsForJobNotFound, err)
				}
			})
		})
		t.Run("MergeChangesets", func(t *testing.T) {
			spec := testBatchSpec(admin.ID)
			if err := s.CreateBatchSpec(ctx, spec); err != nil {
				t.Fatal(err)
			}

			batchChange := testBatchChange(admin.ID, spec)
			if err := s.CreateBatchChange(ctx, batchChange); err != nil {
				t.Fatal(err)
			}
			published := btypes.ChangesetPublicationStatePublished
			openState := btypes.ChangesetExternalStateOpen
			t.Run("open changeset", func(t *testing.T) {
				changeset := ct.CreateChangeset(t, adminCtx, s, ct.TestChangesetOpts{
					Repo:             rs[0].ID,
					ReconcilerState:  btypes.ReconcilerStateCompleted,
					ExternalState:    btypes.ChangesetExternalStateOpen,
					PublicationState: btypes.ChangesetPublicationStatePublished,
					BatchChange:      batchChange.ID,
					IsArchived:       false,
				})
				_, err := svc.CreateChangesetJobs(
					adminCtx,
					batchChange.ID,
					[]int64{changeset.ID},
					btypes.ChangesetJobTypeMerge,
					btypes.ChangesetJobMergePayload{Squash: true},
					store.ListChangesetsOpts{
						PublicationState: &published,
						ReconcilerStates: []btypes.ReconcilerState{btypes.ReconcilerStateCompleted},
						ExternalStates:   []btypes.ChangesetExternalState{openState},
					},
				)
				if err != nil {
					t.Fatal(err)
				}
			})
			t.Run("closed changeset", func(t *testing.T) {
				closedChangeset := ct.CreateChangeset(t, adminCtx, s, ct.TestChangesetOpts{
					Repo:             rs[0].ID,
					ReconcilerState:  btypes.ReconcilerStateCompleted,
					ExternalState:    btypes.ChangesetExternalStateClosed,
					PublicationState: btypes.ChangesetPublicationStatePublished,
					BatchChange:      batchChange.ID,
				})
				_, err := svc.CreateChangesetJobs(
					adminCtx,
					batchChange.ID,
					[]int64{closedChangeset.ID},
					btypes.ChangesetJobTypeMerge,
					btypes.ChangesetJobMergePayload{},
					store.ListChangesetsOpts{
						PublicationState: &published,
						ReconcilerStates: []btypes.ReconcilerState{btypes.ReconcilerStateCompleted},
						ExternalStates:   []btypes.ChangesetExternalState{openState},
					},
				)
				if err != ErrChangesetsForJobNotFound {
					t.Fatalf("wrong error. want=%s, got=%s", ErrChangesetsForJobNotFound, err)
				}
			})
		})

		t.Run("CloseChangesets", func(t *testing.T) {
			spec := testBatchSpec(admin.ID)
			if err := s.CreateBatchSpec(ctx, spec); err != nil {
				t.Fatal(err)
			}

			batchChange := testBatchChange(admin.ID, spec)
			if err := s.CreateBatchChange(ctx, batchChange); err != nil {
				t.Fatal(err)
			}
			published := btypes.ChangesetPublicationStatePublished
			openState := btypes.ChangesetExternalStateOpen
			t.Run("open changeset", func(t *testing.T) {
				changeset := ct.CreateChangeset(t, adminCtx, s, ct.TestChangesetOpts{
					Repo:             rs[0].ID,
					ReconcilerState:  btypes.ReconcilerStateCompleted,
					ExternalState:    btypes.ChangesetExternalStateOpen,
					PublicationState: btypes.ChangesetPublicationStatePublished,
					BatchChange:      batchChange.ID,
					IsArchived:       false,
				})
				_, err := svc.CreateChangesetJobs(
					adminCtx,
					batchChange.ID,
					[]int64{changeset.ID},
					btypes.ChangesetJobTypeClose,
					btypes.ChangesetJobClosePayload{},
					store.ListChangesetsOpts{
						PublicationState: &published,
						ReconcilerStates: []btypes.ReconcilerState{btypes.ReconcilerStateCompleted},
						ExternalStates:   []btypes.ChangesetExternalState{openState},
					},
				)
				if err != nil {
					t.Fatal(err)
				}
			})
			t.Run("closed changeset", func(t *testing.T) {
				closedChangeset := ct.CreateChangeset(t, adminCtx, s, ct.TestChangesetOpts{
					Repo:             rs[0].ID,
					ReconcilerState:  btypes.ReconcilerStateCompleted,
					ExternalState:    btypes.ChangesetExternalStateClosed,
					PublicationState: btypes.ChangesetPublicationStatePublished,
					BatchChange:      batchChange.ID,
				})
				_, err := svc.CreateChangesetJobs(
					adminCtx,
					batchChange.ID,
					[]int64{closedChangeset.ID},
					btypes.ChangesetJobTypeClose,
					btypes.ChangesetJobClosePayload{},
					store.ListChangesetsOpts{
						PublicationState: &published,
						ReconcilerStates: []btypes.ReconcilerState{btypes.ReconcilerStateCompleted},
						ExternalStates:   []btypes.ChangesetExternalState{openState},
					},
				)
				if err != ErrChangesetsForJobNotFound {
					t.Fatalf("wrong error. want=%s, got=%s", ErrChangesetsForJobNotFound, err)
				}
			})
		})

		t.Run("PublishChangesets", func(t *testing.T) {
			spec := testBatchSpec(admin.ID)
			if err := s.CreateBatchSpec(ctx, spec); err != nil {
				t.Fatal(err)
			}

			batchChange := testBatchChange(admin.ID, spec)
			if err := s.CreateBatchChange(ctx, batchChange); err != nil {
				t.Fatal(err)
			}

			changeset := ct.CreateChangeset(t, adminCtx, s, ct.TestChangesetOpts{
				Repo:            rs[0].ID,
				ReconcilerState: btypes.ReconcilerStateCompleted,
				BatchChange:     batchChange.ID,
			})

			_, err := svc.CreateChangesetJobs(
				adminCtx,
				batchChange.ID,
				[]int64{changeset.ID},
				btypes.ChangesetJobTypePublish,
				btypes.ChangesetJobPublishPayload{Draft: true},
				store.ListChangesetsOpts{},
			)
			if err != nil {
				t.Fatal(err)
			}

		})
	})
}

func testBatchChange(user int32, spec *btypes.BatchSpec) *btypes.BatchChange {
	c := &btypes.BatchChange{
		Name:             "test-batch-change",
		InitialApplierID: user,
		NamespaceUserID:  user,
		BatchSpecID:      spec.ID,
		LastApplierID:    user,
		LastAppliedAt:    time.Now(),
	}

	return c
}

func testBatchSpec(user int32) *btypes.BatchSpec {
	return &btypes.BatchSpec{
		UserID:          user,
		NamespaceUserID: user,
	}
}

func testChangeset(repoID api.RepoID, batchChange int64, extState btypes.ChangesetExternalState) *btypes.Changeset {
	changeset := &btypes.Changeset{
		RepoID:              repoID,
		ExternalServiceType: extsvc.TypeGitHub,
		ExternalID:          fmt.Sprintf("ext-id-%d", batchChange),
		Metadata:            &github.PullRequest{State: string(extState), CreatedAt: time.Now()},
		ExternalState:       extState,
	}

	if batchChange != 0 {
		changeset.BatchChanges = []btypes.BatchChangeAssoc{{BatchChangeID: batchChange}}
	}

	return changeset
}
