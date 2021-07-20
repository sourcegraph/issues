package background

import (
	"context"
	"testing"

	"github.com/sourcegraph/sourcegraph/enterprise/internal/batches/global"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/batches/sources"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/batches/store"
	ct "github.com/sourcegraph/sourcegraph/enterprise/internal/batches/testing"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/batches/types"
	btypes "github.com/sourcegraph/sourcegraph/enterprise/internal/batches/types"
	"github.com/sourcegraph/sourcegraph/internal/database/dbtest"
	"github.com/sourcegraph/sourcegraph/internal/errcode"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/github"
)

func TestBulkProcessor(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := dbtest.NewDB(t, "")
	tx := dbtest.NewTx(t, db)
	bstore := store.New(tx, nil)
	user := ct.CreateTestUser(t, db, true)
	repos, _ := ct.CreateTestRepos(t, ctx, db, 1)
	repo := repos[0]
	ct.CreateTestSiteCredential(t, bstore, repo)
	batchSpec := ct.CreateBatchSpec(t, ctx, bstore, "test-bulk", user.ID)
	batchChange := ct.CreateBatchChange(t, ctx, bstore, "test-bulk", user.ID, batchSpec.ID)
	changesetSpec := ct.CreateChangesetSpec(t, ctx, bstore, ct.TestSpecOpts{
		User:      user.ID,
		Repo:      repo.ID,
		BatchSpec: batchSpec.ID,
		HeadRef:   "main",
	})
	changeset := ct.CreateChangeset(t, ctx, bstore, ct.TestChangesetOpts{
		Repo:                repo.ID,
		BatchChanges:        []types.BatchChangeAssoc{{BatchChangeID: batchChange.ID}},
		Metadata:            &github.PullRequest{},
		ExternalServiceType: extsvc.TypeGitHub,
		CurrentSpec:         changesetSpec.ID,
	})

	t.Run("Unknown job type", func(t *testing.T) {
		fake := &sources.FakeChangesetSource{}
		bp := &bulkProcessor{
			tx:      bstore,
			sourcer: sources.NewFakeSourcer(nil, fake),
		}
		job := &types.ChangesetJob{
			JobType:     types.ChangesetJobTypeComment,
			ChangesetID: changeset.ID,
			UserID:      user.ID,
		}
		if err := bstore.CreateChangesetJob(ctx, job); err != nil {
			t.Fatal(err)
		}
		job.JobType = types.ChangesetJobType("UNKNOWN")
		err := bp.process(ctx, job)
		if err == nil || err.Error() != `invalid job type "UNKNOWN"` {
			t.Fatalf("unexpected error returned %s", err)
		}
	})

	t.Run("Comment job", func(t *testing.T) {
		fake := &sources.FakeChangesetSource{}
		bp := &bulkProcessor{
			tx:      bstore,
			sourcer: sources.NewFakeSourcer(nil, fake),
		}
		job := &types.ChangesetJob{
			JobType:     types.ChangesetJobTypeComment,
			ChangesetID: changeset.ID,
			UserID:      user.ID,
		}
		if err := bstore.CreateChangesetJob(ctx, job); err != nil {
			t.Fatal(err)
		}
		err := bp.process(ctx, job)
		if err != nil {
			t.Fatal(err)
		}
		if !fake.CreateCommentCalled {
			t.Fatal("expected CreateComment to be called but wasn't")
		}
	})

	t.Run("Detach job", func(t *testing.T) {
		fake := &sources.FakeChangesetSource{}
		bp := &bulkProcessor{
			tx:      bstore,
			sourcer: sources.NewFakeSourcer(nil, fake),
		}
		job := &types.ChangesetJob{
			JobType:       types.ChangesetJobTypeDetach,
			ChangesetID:   changeset.ID,
			UserID:        user.ID,
			BatchChangeID: batchChange.ID,
		}
		if err := bstore.CreateChangesetJob(ctx, job); err != nil {
			t.Fatal(err)
		}
		err := bp.process(ctx, job)
		if err != nil {
			t.Fatal(err)
		}
		ch, err := bstore.GetChangesetByID(ctx, changeset.ID)
		if err != nil {
			t.Fatal(err)
		}
		if len(ch.BatchChanges) != 1 {
			t.Fatalf("invalid batch changes associated, expected one, got=%+v", ch.BatchChanges)
		}
		if !ch.BatchChanges[0].Detach {
			t.Fatal("not marked as to be detached")
		}
		if ch.ReconcilerState != btypes.ReconcilerStateQueued {
			t.Fatalf("invalid reconciler state, got=%q", ch.ReconcilerState)
		}
	})

	t.Run("Reenqueue job", func(t *testing.T) {
		fake := &sources.FakeChangesetSource{}
		bp := &bulkProcessor{
			tx:      bstore,
			sourcer: sources.NewFakeSourcer(nil, fake),
		}
		job := &types.ChangesetJob{
			JobType:     types.ChangesetJobTypeReenqueue,
			ChangesetID: changeset.ID,
			UserID:      user.ID,
		}
		changeset.ReconcilerState = btypes.ReconcilerStateFailed
		if err := bstore.UpdateChangeset(ctx, changeset); err != nil {
			t.Fatal(err)
		}
		if err := bstore.CreateChangesetJob(ctx, job); err != nil {
			t.Fatal(err)
		}
		err := bp.process(ctx, job)
		if err != nil {
			t.Fatal(err)
		}
		changeset, err = bstore.GetChangesetByID(ctx, changeset.ID)
		if err != nil {
			t.Fatal(err)
		}
		if have, want := changeset.ReconcilerState, btypes.ReconcilerStateQueued; have != want {
			t.Fatalf("unexpected reconciler state, have=%q want=%q", have, want)
		}
	})

	t.Run("Merge job", func(t *testing.T) {
		fake := &sources.FakeChangesetSource{}
		bp := &bulkProcessor{
			tx:      bstore,
			sourcer: sources.NewFakeSourcer(nil, fake),
		}
		job := &types.ChangesetJob{
			JobType:     types.ChangesetJobTypeMerge,
			ChangesetID: changeset.ID,
			UserID:      user.ID,
		}
		if err := bstore.CreateChangesetJob(ctx, job); err != nil {
			t.Fatal(err)
		}
		err := bp.process(ctx, job)
		if err != nil {
			t.Fatal(err)
		}
		if !fake.MergeChangesetCalled {
			t.Fatal("expected MergeChangeset to be called but wasn't")
		}
	})

	t.Run("Close job", func(t *testing.T) {
		fake := &sources.FakeChangesetSource{FakeMetadata: &github.PullRequest{}}
		bp := &bulkProcessor{
			tx:      bstore,
			sourcer: sources.NewFakeSourcer(nil, fake),
		}
		job := &types.ChangesetJob{
			JobType:     types.ChangesetJobTypeClose,
			ChangesetID: changeset.ID,
			UserID:      user.ID,
		}
		if err := bstore.CreateChangesetJob(ctx, job); err != nil {
			t.Fatal(err)
		}
		err := bp.process(ctx, job)
		if err != nil {
			t.Fatal(err)
		}
		if !fake.CloseChangesetCalled {
			t.Fatal("expected CloseChangeset to be called but wasn't")
		}
	})

	t.Run("Publish job", func(t *testing.T) {
		fake := &sources.FakeChangesetSource{FakeMetadata: &github.PullRequest{}}
		bp := &bulkProcessor{
			tx:      bstore,
			sourcer: sources.NewFakeSourcer(nil, fake),
		}

		t.Run("errors", func(t *testing.T) {
			for name, tc := range map[string]struct {
				spec          *ct.TestSpecOpts
				changeset     ct.TestChangesetOpts
				wantRetryable bool
			}{
				"imported changeset": {
					spec: nil,
					changeset: ct.TestChangesetOpts{
						Repo:            repo.ID,
						BatchChange:     batchChange.ID,
						CurrentSpec:     0,
						ReconcilerState: btypes.ReconcilerStateCompleted,
					},
					wantRetryable: false,
				},
				"bogus changeset spec ID, dude": {
					spec: nil,
					changeset: ct.TestChangesetOpts{
						Repo:            repo.ID,
						BatchChange:     batchChange.ID,
						CurrentSpec:     -1,
						ReconcilerState: btypes.ReconcilerStateCompleted,
					},
					wantRetryable: false,
				},
				"publication state set": {
					spec: &ct.TestSpecOpts{
						User:      user.ID,
						Repo:      repo.ID,
						BatchSpec: batchSpec.ID,
						HeadRef:   "main",
						Published: false,
					},
					changeset: ct.TestChangesetOpts{
						Repo:            repo.ID,
						BatchChange:     batchChange.ID,
						ReconcilerState: btypes.ReconcilerStateCompleted,
					},
					wantRetryable: false,
				},
				"processing": {
					spec: &ct.TestSpecOpts{
						User:      user.ID,
						Repo:      repo.ID,
						BatchSpec: batchSpec.ID,
						HeadRef:   "main",
					},
					changeset: ct.TestChangesetOpts{
						Repo:            repo.ID,
						BatchChange:     batchChange.ID,
						ReconcilerState: btypes.ReconcilerStateProcessing,
					},
					wantRetryable: true,
				},
			} {
				t.Run(name, func(t *testing.T) {
					var changesetSpec *btypes.ChangesetSpec
					if tc.spec != nil {
						changesetSpec = ct.CreateChangesetSpec(t, ctx, bstore, *tc.spec)
					}

					if changesetSpec != nil {
						tc.changeset.CurrentSpec = changesetSpec.ID
					}
					changeset := ct.CreateChangeset(t, ctx, bstore, tc.changeset)

					job := &types.ChangesetJob{
						JobType:       types.ChangesetJobTypePublish,
						BatchChangeID: batchChange.ID,
						ChangesetID:   changeset.ID,
						UserID:        user.ID,
						Payload: &types.ChangesetJobPublishPayload{
							Draft: false,
						},
					}

					if err := bp.process(ctx, job); err == nil {
						t.Error("unexpected nil error")
					} else if tc.wantRetryable && errcode.IsNonRetryable(err) {
						t.Errorf("error is not retryable: %v", err)
					} else if !tc.wantRetryable && !errcode.IsNonRetryable(err) {
						t.Errorf("error is retryable: %v", err)
					}
				})
			}
		})

		t.Run("success", func(t *testing.T) {
			for _, reconcilerState := range []btypes.ReconcilerState{
				btypes.ReconcilerStateCompleted,
				btypes.ReconcilerStateErrored,
				btypes.ReconcilerStateFailed,
				btypes.ReconcilerStateQueued,
				btypes.ReconcilerStateScheduled,
			} {
				t.Run(string(reconcilerState), func(t *testing.T) {
					for name, draft := range map[string]bool{
						"draft":     true,
						"published": false,
					} {
						t.Run(name, func(t *testing.T) {
							changesetSpec := ct.CreateChangesetSpec(t, ctx, bstore, ct.TestSpecOpts{
								User:      user.ID,
								Repo:      repo.ID,
								BatchSpec: batchSpec.ID,
								HeadRef:   "main",
							})
							changeset := ct.CreateChangeset(t, ctx, bstore, ct.TestChangesetOpts{
								Repo:            repo.ID,
								BatchChange:     batchChange.ID,
								CurrentSpec:     changesetSpec.ID,
								ReconcilerState: reconcilerState,
							})

							job := &types.ChangesetJob{
								JobType:       types.ChangesetJobTypePublish,
								BatchChangeID: batchChange.ID,
								ChangesetID:   changeset.ID,
								UserID:        user.ID,
								Payload: &types.ChangesetJobPublishPayload{
									Draft: draft,
								},
							}

							if err := bp.process(ctx, job); err != nil {
								t.Errorf("unexpected error: %v", err)
							}

							changeset, err := bstore.GetChangesetByID(ctx, changeset.ID)
							if err != nil {
								t.Fatal(err)
							}

							var want btypes.ChangesetUiPublicationState
							if draft {
								want = btypes.ChangesetUiPublicationStateDraft
							} else {
								want = btypes.ChangesetUiPublicationStatePublished
							}
							if have := changeset.UiPublicationState; have == nil || *have != want {
								t.Fatalf("unexpected UI publication state: have=%v want=%q", have, want)
							}

							if have, want := changeset.ReconcilerState, global.DefaultReconcilerEnqueueState(); have != want {
								t.Fatalf("unexpected reconciler state, have=%q want=%q", have, want)
							}
						})
					}
				})
			}
		})
	})
}
