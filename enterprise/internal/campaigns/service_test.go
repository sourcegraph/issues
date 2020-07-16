package campaigns

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/authz"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/cmd/repo-updater/repos"
	ct "github.com/sourcegraph/sourcegraph/enterprise/internal/campaigns/testing"
	"github.com/sourcegraph/sourcegraph/internal/actor"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/campaigns"
	"github.com/sourcegraph/sourcegraph/internal/db"
	"github.com/sourcegraph/sourcegraph/internal/db/dbconn"
	"github.com/sourcegraph/sourcegraph/internal/db/dbtesting"
	"github.com/sourcegraph/sourcegraph/internal/errcode"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/github"
	"github.com/sourcegraph/sourcegraph/internal/httpcli"
	"github.com/sourcegraph/sourcegraph/internal/repoupdater"
	"github.com/sourcegraph/sourcegraph/schema"
)

func init() {
	dbtesting.DBNameSuffix = "campaignsenterpriserdb"
}

func TestServicePermissionLevels(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	ctx := backend.WithAuthzBypass(context.Background())
	dbtesting.SetupGlobalTestDB(t)

	store := NewStore(dbconn.Global)
	svc := NewService(store, nil)

	admin := createTestUser(ctx, t)
	if !admin.SiteAdmin {
		t.Fatalf("admin is not site admin")
	}

	user := createTestUser(ctx, t)
	if user.SiteAdmin {
		t.Fatalf("user cannot be site admin")
	}

	otherUser := createTestUser(ctx, t)
	if otherUser.SiteAdmin {
		t.Fatalf("user cannot be site admin")
	}

	var rs []*repos.Repo
	for i := 0; i < 4; i++ {
		rs = append(rs, testRepo(i, extsvc.TypeGitHub))
	}

	reposStore := repos.NewDBStore(dbconn.Global, sql.TxOptions{})
	err := reposStore.UpsertRepos(ctx, rs...)
	if err != nil {
		t.Fatal(err)
	}

	createTestData := func(t *testing.T, s *Store, svc *Service, author int32) (*campaigns.Campaign, *campaigns.Changeset) {
		campaign := testCampaign(author, 0)
		if err = s.CreateCampaign(ctx, campaign); err != nil {
			t.Fatal(err)
		}

		changeset := testChangeset(rs[0].ID, campaign.ID, campaign.ID, campaigns.ChangesetStateOpen)
		if err = s.CreateChangesets(ctx, changeset); err != nil {
			t.Fatal(err)
		}

		campaign.ChangesetIDs = append(campaign.ChangesetIDs, changeset.ID)
		if err := s.UpdateCampaign(ctx, campaign); err != nil {
			t.Fatal(err)
		}

		return campaign, changeset
	}

	assertAuthError := func(t *testing.T, err error) {
		t.Helper()

		if err == nil {
			t.Fatalf("expected error. got none")
		}
		if err != nil {
			if _, ok := err.(*backend.InsufficientAuthorizationError); !ok {
				t.Fatalf("wrong error: %s (%T)", err, err)
			}
		}
	}

	assertNoAuthError := func(t *testing.T, err error) {
		t.Helper()

		// Ignore other errors, we only want to check whether it's an auth error
		if _, ok := err.(*backend.InsufficientAuthorizationError); ok {
			t.Fatalf("got auth error")
		}
	}

	tests := []struct {
		name           string
		campaignAuthor int32
		currentUser    int32
		assertFunc     func(t *testing.T, err error)
	}{
		{
			name:           "unauthorized user",
			campaignAuthor: user.ID,
			currentUser:    otherUser.ID,
			assertFunc:     assertAuthError,
		},
		{
			name:           "campaign author",
			campaignAuthor: user.ID,
			currentUser:    user.ID,
			assertFunc:     assertNoAuthError,
		},

		{
			name:           "site-admin",
			campaignAuthor: user.ID,
			currentUser:    admin.ID,
			assertFunc:     assertNoAuthError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			campaign, changeset := createTestData(t, store, svc, tc.campaignAuthor)
			// Fresh context.Background() because the previous one is wrapped in AuthzBypas
			currentUserCtx := actor.WithActor(context.Background(), actor.FromUser(tc.currentUser))

			t.Run("EnqueueChangesetSync", func(t *testing.T) {
				err = svc.EnqueueChangesetSync(currentUserCtx, changeset.ID)
				tc.assertFunc(t, err)
			})

			t.Run("CloseCampaign", func(t *testing.T) {
				_, err = svc.CloseCampaign(currentUserCtx, campaign.ID, false)
				tc.assertFunc(t, err)
			})

			t.Run("DeleteCampaign", func(t *testing.T) {
				err = svc.DeleteCampaign(currentUserCtx, campaign.ID)
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
	dbtesting.SetupGlobalTestDB(t)

	now := time.Now().UTC().Truncate(time.Microsecond)
	clock := func() time.Time {
		return now.UTC().Truncate(time.Microsecond)
	}
	cf := httpcli.NewExternalHTTPClientFactory()

	user := createTestUser(ctx, t)

	store := NewStoreWithClock(dbconn.Global, clock)

	reposStore := repos.NewDBStore(dbconn.Global, sql.TxOptions{})

	ext := &repos.ExternalService{
		Kind:        extsvc.TypeGitHub,
		DisplayName: "GitHub",
		Config: marshalJSON(t, &schema.GitHubConnection{
			Url:   "https://github.com",
			Token: "SECRETTOKEN",
		}),
	}
	if err := reposStore.UpsertExternalServices(ctx, ext); err != nil {
		t.Fatal(err)
	}

	var rs []*repos.Repo
	for i := 0; i < 4; i++ {
		r := testRepo(i, extsvc.TypeGitHub)
		r.Sources = map[string]*repos.SourceInfo{ext.URN(): {ID: ext.URN()}}

		rs = append(rs, r)
	}

	awsCodeCommitRepoID := 4
	{
		r := testRepo(awsCodeCommitRepoID, extsvc.TypeAWSCodeCommit)
		r.Sources = map[string]*repos.SourceInfo{ext.URN(): {ID: ext.URN()}}
		rs = append(rs, r)
	}

	err := reposStore.UpsertRepos(ctx, rs...)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("CreateCampaign", func(t *testing.T) {
		patchSet := &campaigns.PatchSet{UserID: user.ID}
		err = store.CreatePatchSet(ctx, patchSet)
		if err != nil {
			t.Fatal(err)
		}

		campaign := testCampaign(user.ID, patchSet.ID)
		svc := NewServiceWithClock(store, cf, clock)

		// Without Patches it should fail
		err = svc.CreateCampaign(ctx, campaign)
		if err != ErrNoPatches {
			t.Fatal("CreateCampaign did not produce expected error")
		}

		for _, repo := range rs {
			patch := testPatch(patchSet.ID, repo.ID, now)
			err := store.CreatePatch(ctx, patch)
			if err != nil {
				t.Fatal(err)
			}
		}

		// With Patches it should succeed
		err = svc.CreateCampaign(ctx, campaign)
		if err != nil {
			t.Fatal(err)
		}

		_, err = store.GetCampaign(ctx, GetCampaignOpts{ID: campaign.ID})
		if err != nil {
			t.Fatal(err)
		}

		haveJobs, _, err := store.ListChangesetJobs(ctx, ListChangesetJobsOpts{
			CampaignID: campaign.ID,
		})
		if err != nil {
			t.Fatal(err)
		}

		// Validate no changeset jobs have been created yet.
		if len(haveJobs) != 0 {
			t.Errorf("wrong number of ChangesetJobs: %d. want=%d", len(haveJobs), 0)
		}
	})

	t.Run("DeleteCampaign", func(t *testing.T) {
		patchSet := &campaigns.PatchSet{UserID: user.ID}
		if err = store.CreatePatchSet(ctx, patchSet); err != nil {
			t.Fatal(err)
		}

		patch := testPatch(patchSet.ID, rs[0].ID, now)
		if err := store.CreatePatch(ctx, patch); err != nil {
			t.Fatal(err)
		}

		campaign := testCampaign(user.ID, patchSet.ID)

		svc := NewServiceWithClock(store, cf, clock)

		if err = svc.CreateCampaign(ctx, campaign); err != nil {
			t.Fatal(err)
		}

		// TODO: Fix this as soon as we have "processing" campaigns.
		//
		// // Create a processing changeset job.
		// err = svc.EnqueueChangesetJobForPatch(ctx, patch.ID)
		// if err != nil {
		// 	t.Fatalf("Failed to create ChangesetJob: %s", err)
		// }
		//
		// if err := svc.DeleteCampaign(ctx, campaign.ID, true); err != ErrDeleteProcessingCampaign {
		// 	t.Fatalf("DeleteCampaign returned unexpected error: %s", err)
		// }
		//
		// jobs, _, err := store.ListChangesetJobs(ctx, ListChangesetJobsOpts{
		// 	CampaignID: campaign.ID,
		// 	Limit:      -1,
		// })
		// if err != nil {
		// 	t.Fatal(err)
		// }
		//
		// if len(jobs) != 1 {
		// 	t.Fatalf("wrong number of changeset jobs: %d", len(jobs))
		// }
		//
		// for _, j := range jobs {
		// 	j.Error = "failed"
		// 	j.FinishedAt = clock()
		// 	if err := store.UpdateChangesetJob(ctx, j); err != nil {
		// 		t.Fatalf("updating changeset job failed: %s\n", err)
		// 	}
		// }

		// Now it should work, since the jobs failed to execute and campaign is
		// no longer processing.
		if err := svc.DeleteCampaign(ctx, campaign.ID); err != nil {
			t.Fatalf("campaign not deleted: %s", err)
		}
	})

	t.Run("CloseCampaign", func(t *testing.T) {
		patchSet := &campaigns.PatchSet{UserID: user.ID}
		if err = store.CreatePatchSet(ctx, patchSet); err != nil {
			t.Fatal(err)
		}

		patch := testPatch(patchSet.ID, rs[0].ID, now)
		if err := store.CreatePatch(ctx, patch); err != nil {
			t.Fatal(err)
		}

		campaign := testCampaign(user.ID, patchSet.ID)

		svc := NewServiceWithClock(store, cf, clock)

		if err = svc.CreateCampaign(ctx, campaign); err != nil {
			t.Fatal(err)
		}

		// TODO: Fix this as soon as we have "processing" campaigns.
		//
		// // Create a processing changeset job.
		// err = svc.EnqueueChangesetJobForPatch(ctx, patch.ID)
		// if err != nil {
		// 	t.Fatalf("Failed to create ChangesetJob: %s", err)
		// }
		//
		// if _, err = svc.CloseCampaign(ctx, campaign.ID, true); err != ErrCloseProcessingCampaign {
		// 	t.Fatalf("CloseCampaign returned unexpected error: %s", err)
		// }
		//
		// jobs, _, err := store.ListChangesetJobs(ctx, ListChangesetJobsOpts{
		// 	CampaignID: campaign.ID,
		// 	Limit:      -1,
		// })
		// if err != nil {
		// 	t.Fatal(err)
		// }
		//
		// if len(jobs) != 1 {
		// 	t.Fatalf("wrong number of changeset jobs: %d", len(jobs))
		// }
		//
		// for _, j := range jobs {
		// 	j.Error = "failed"
		// 	j.FinishedAt = clock()
		// 	if err := store.UpdateChangesetJob(ctx, j); err != nil {
		// 		t.Fatalf("updating changeset job failed: %s\n", err)
		// 	}
		// }

		// Now it should work, since the jobs failed to execute and campaign is
		// no longer processing.
		campaign, err = svc.CloseCampaign(ctx, campaign.ID, true)
		if err != nil {
			t.Fatalf("campaign not closed: %s", err)
		}
		if campaign.ClosedAt.IsZero() {
			t.Fatalf("campaign ClosedAt is zero")
		}
	})

	t.Run("CreateCampaignWithPatchSetAttachedToOtherCampaign", func(t *testing.T) {
		patchSet := &campaigns.PatchSet{UserID: user.ID}
		err = store.CreatePatchSet(ctx, patchSet)
		if err != nil {
			t.Fatal(err)
		}

		for _, repo := range rs {
			err := store.CreatePatch(ctx, testPatch(patchSet.ID, repo.ID, now))
			if err != nil {
				t.Fatal(err)
			}
		}

		campaign := testCampaign(user.ID, patchSet.ID)
		svc := NewServiceWithClock(store, cf, clock)

		err = svc.CreateCampaign(ctx, campaign)
		if err != nil {
			t.Fatal(err)
		}

		otherCampaign := testCampaign(user.ID, patchSet.ID)
		err = svc.CreateCampaign(ctx, otherCampaign)
		if err != ErrPatchSetDuplicate {
			t.Fatal("no error even though another campaign has same patch set")
		}
	})

	t.Run("GetCampaignStatus", func(t *testing.T) {
		// Make sure that user is an admin
		if !user.SiteAdmin {
			t.Fatalf("user is not an admin")
		}

		otherUser := createTestUser(ctx, t)
		if otherUser.SiteAdmin {
			t.Fatalf("otherUser is admin")
		}

		patchSet := &campaigns.PatchSet{UserID: otherUser.ID}
		if err = store.CreatePatchSet(ctx, patchSet); err != nil {
			t.Fatal(err)
		}

		patches := make([]*campaigns.Patch, 0, len(rs))
		for _, repo := range rs {
			patch := testPatch(patchSet.ID, repo.ID, now)
			if err := store.CreatePatch(ctx, patch); err != nil {
				t.Fatal(err)
			}
			patches = append(patches, patch)
		}

		campaign := testCampaign(otherUser.ID, patchSet.ID)
		if err = store.CreateCampaign(ctx, campaign); err != nil {
			t.Fatal(err)
		}

		changesetJobs := make([]*campaigns.ChangesetJob, 0, len(patches))
		for _, p := range patches {
			job := &campaigns.ChangesetJob{
				CampaignID: campaign.ID,
				PatchID:    p.ID,
				StartedAt:  clock(),
				FinishedAt: clock(),
				Error:      "error",
			}
			if err = store.CreateChangesetJob(ctx, job); err != nil {
				t.Fatal(err)
			}
			changesetJobs = append(changesetJobs, job)
		}

		// As site-admin
		userCtx := actor.WithActor(context.Background(), actor.FromUser(user.ID))
		svc := NewService(store, cf)
		status, err := svc.GetCampaignStatus(userCtx, campaign)
		if err != nil {
			t.Fatal(err)
		}

		if have, want := len(status.ProcessErrors), len(changesetJobs); have != want {
			t.Fatalf("wrong number of errors returned. want=%d, have=%d", want, have)
		}

		// As author of campaign and non-site-admin
		otherUserCtx := actor.WithActor(context.Background(), actor.FromUser(otherUser.ID))
		status, err = svc.GetCampaignStatus(otherUserCtx, campaign)
		if err != nil {
			t.Fatal(err)
		}

		if have, want := len(status.ProcessErrors), len(changesetJobs); have != want {
			t.Fatalf("wrong number of errors returned. want=%d, have=%d", want, have)
		}

		// As author of campaign and non-site-admin with filtered out repositories
		authzFilterRepo(t, patches[0].RepoID)

		status, err = svc.GetCampaignStatus(otherUserCtx, campaign)
		if err != nil {
			t.Fatal(err)
		}

		// One less error
		if have, want := len(status.ProcessErrors), len(changesetJobs)-1; have != want {
			t.Fatalf("wrong number of errors returned. want=%d, have=%d", want, have)
		}

		// Change author of campaign to site-admin
		campaign.AuthorID = user.ID
		if err = store.UpdateCampaign(ctx, campaign); err != nil {
			t.Fatal(err)
		}

		// As non-author and non-site-admin
		status, err = svc.GetCampaignStatus(otherUserCtx, campaign)
		if err != nil {
			t.Fatal(err)
		}

		if have, want := len(status.ProcessErrors), 0; have != want {
			t.Fatalf("wrong number of errors returned. want=%d, have=%d", want, have)
		}
	})

	t.Run("EnqueueChangesetSync", func(t *testing.T) {
		svc := NewServiceWithClock(store, cf, clock)

		campaign := testCampaign(user.ID, 0)
		if err = store.CreateCampaign(ctx, campaign); err != nil {
			t.Fatal(err)
		}

		changeset := testChangeset(rs[0].ID, campaign.ID, 0, campaigns.ChangesetStateOpen)
		if err = store.CreateChangesets(ctx, changeset); err != nil {
			t.Fatal(err)
		}

		campaign.ChangesetIDs = []int64{changeset.ID}
		if err = store.UpdateCampaign(ctx, campaign); err != nil {
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

		// Repo filtered out by authzFilter
		authzFilterRepo(t, rs[0].ID)

		// should result in a not found error
		if err := svc.EnqueueChangesetSync(ctx, changeset.ID); !errcode.IsNotFound(err) {
			t.Fatalf("expected not-found error but got %s", err)
		}
	})

	t.Run("CloseOpenChangesets", func(t *testing.T) {
		changeset1 := testChangeset(rs[0].ID, 0, 121314, campaigns.ChangesetStateOpen)
		changeset2 := testChangeset(rs[1].ID, 0, 141516, campaigns.ChangesetStateOpen)
		if err = store.CreateChangesets(ctx, changeset1, changeset2); err != nil {
			t.Fatal(err)
		}

		// Repo of changeset2 filtered out by authzFilter
		authzFilterRepo(t, changeset2.RepoID)

		fakeSource := &ct.FakeChangesetSource{Err: nil}
		sourcer := repos.NewFakeSourcer(nil, fakeSource)

		svc := NewServiceWithClock(store, cf, clock)
		svc.sourcer = sourcer

		// Try to close open changesets
		err := svc.CloseOpenChangesets(ctx, []*campaigns.Changeset{changeset1, changeset2})
		if err != nil {
			t.Fatal(err)
		}

		// Only changeset1 should be closed
		if have, want := len(fakeSource.ClosedChangesets), 1; have != want {
			t.Fatalf("ClosedChangesets has wrong length. want=%d, have=%d", want, have)
		}

		if have, want := fakeSource.ClosedChangesets[0].RepoID, changeset1.RepoID; have != want {
			t.Fatalf("wrong changesets closed. want=%d, have=%d", want, have)
		}
	})

	t.Run("CreateCampaignSpec", func(t *testing.T) {
		svc := NewServiceWithClock(store, cf, clock)

		changesetSpecs := make([]*campaigns.ChangesetSpec, 0, len(rs))
		changesetSpecRandIDs := make([]string, 0, len(rs))
		for _, r := range rs {
			cs := &campaigns.ChangesetSpec{RepoID: r.ID, UserID: user.ID}
			if err := store.CreateChangesetSpec(ctx, cs); err != nil {
				t.Fatal(err)
			}
			changesetSpecs = append(changesetSpecs, cs)
			changesetSpecRandIDs = append(changesetSpecRandIDs, cs.RandID)
		}

		t.Run("success", func(t *testing.T) {
			spec := &campaigns.CampaignSpec{
				UserID:          user.ID,
				NamespaceUserID: user.ID,
				RawSpec:         ct.TestRawCampaignSpec,
			}

			if err := svc.CreateCampaignSpec(ctx, spec, changesetSpecRandIDs); err != nil {
				t.Fatal(err)
			}

			if spec.ID == 0 {
				t.Fatalf("CampaignSpec ID is 0")
			}

			var wantFields campaigns.CampaignSpecFields
			if err := json.Unmarshal([]byte(spec.RawSpec), &wantFields); err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(wantFields, spec.Spec); diff != "" {
				t.Fatalf("wrong spec fields (-want +got):\n%s", diff)
			}

			for _, cs := range changesetSpecs {
				cs2, err := store.GetChangesetSpec(ctx, GetChangesetSpecOpts{ID: cs.ID})
				if err != nil {
					t.Fatal(err)
				}

				if have, want := cs2.CampaignSpecID, spec.ID; have != want {
					t.Fatalf("changesetSpec has wrong CampaignSpecID. want=%d, have=%d", want, have)
				}
			}
		})

		t.Run("missing repository permissions", func(t *testing.T) {
			// Single repository filtered out by authzFilter
			authzFilterRepo(t, changesetSpecs[0].RepoID)

			spec := &campaigns.CampaignSpec{
				UserID:          user.ID,
				NamespaceUserID: user.ID,
				RawSpec:         ct.TestRawCampaignSpec,
			}

			if err := svc.CreateCampaignSpec(ctx, spec, changesetSpecRandIDs); !errcode.IsNotFound(err) {
				t.Fatalf("expected not-found error but got %s", err)
			}

		})

		t.Run("invalid changesetspec id", func(t *testing.T) {
			spec := &campaigns.CampaignSpec{
				UserID:          user.ID,
				NamespaceUserID: user.ID,
				RawSpec:         ct.TestRawCampaignSpec,
			}

			containsInvalidID := []string{changesetSpecRandIDs[0], "foobar"}

			if err := svc.CreateCampaignSpec(ctx, spec, containsInvalidID); !errcode.IsNotFound(err) {
				t.Fatalf("expected not-found error but got %s", err)
			}
		})
	})

	t.Run("CreateChangesetSpec", func(t *testing.T) {
		svc := NewServiceWithClock(store, cf, clock)

		repo := rs[0]
		rawSpec := ct.NewRawChangesetSpec(graphqlbackend.MarshalRepositoryID(repo.ID))

		t.Run("success", func(t *testing.T) {
			spec := &campaigns.ChangesetSpec{UserID: user.ID, RawSpec: rawSpec}
			if err := svc.CreateChangesetSpec(ctx, spec); err != nil {
				t.Fatal(err)
			}

			if spec.ID == 0 {
				t.Fatalf("ChangesetSpec ID is 0")
			}

			var wantFields campaigns.ChangesetSpecFields
			if err := json.Unmarshal([]byte(spec.RawSpec), &wantFields); err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(wantFields, spec.Spec); diff != "" {
				t.Fatalf("wrong spec fields (-want +got):\n%s", diff)
			}
		})

		t.Run("missing repository permissions", func(t *testing.T) {
			// Single repository filtered out by authzFilter
			authzFilterRepo(t, repo.ID)

			spec := &campaigns.ChangesetSpec{UserID: user.ID, RawSpec: rawSpec}
			if err := svc.CreateChangesetSpec(ctx, spec); !errcode.IsNotFound(err) {
				t.Fatalf("expected not-found error but got %s", err)
			}
		})
	})

	t.Run("ApplyCampaign", func(t *testing.T) {
		svc := NewServiceWithClock(store, cf, clock)

		createCampaignSpec := func(t *testing.T, name string, userID int32) *campaigns.CampaignSpec {
			t.Helper()

			s := &campaigns.CampaignSpec{
				UserID:          userID,
				NamespaceUserID: userID,
				Spec: campaigns.CampaignSpecFields{
					Name:        name,
					Description: "the description",
					ChangesetTemplate: campaigns.ChangesetTemplate{
						Branch: "branch-name",
					},
				},
			}

			if err := store.CreateCampaignSpec(ctx, s); err != nil {
				t.Fatal(err)
			}

			return s
		}

		t.Run("new campaign", func(t *testing.T) {
			campaignSpec := createCampaignSpec(t, "campaign-name", user.ID)
			campaign, err := svc.ApplyCampaign(ctx, ApplyCampaignOpts{
				NamespaceUserID:    user.ID,
				CampaignSpecRandID: campaignSpec.RandID,
			})
			if err != nil {
				t.Fatal(err)
			}

			if campaign.ID == 0 {
				t.Fatalf("campaign ID is 0")
			}

			want := &campaigns.Campaign{
				Name:            campaignSpec.Spec.Name,
				Description:     campaignSpec.Spec.Description,
				Branch:          campaignSpec.Spec.ChangesetTemplate.Branch,
				AuthorID:        campaignSpec.UserID,
				ChangesetIDs:    []int64{},
				NamespaceUserID: campaignSpec.NamespaceUserID,
				CampaignSpecID:  campaignSpec.ID,

				// Ignore these fields
				ID:        campaign.ID,
				UpdatedAt: campaign.UpdatedAt,
				CreatedAt: campaign.CreatedAt,
			}

			if diff := cmp.Diff(want, campaign); diff != "" {
				t.Fatalf("wrong spec fields (-want +got):\n%s", diff)
			}
		})

		t.Run("existing campaign", func(t *testing.T) {
			campaignSpec := createCampaignSpec(t, "campaign-name", user.ID)
			campaign, err := svc.ApplyCampaign(ctx, ApplyCampaignOpts{
				NamespaceUserID:    user.ID,
				CampaignSpecRandID: campaignSpec.RandID,
			})
			if err != nil {
				t.Fatal(err)
			}

			if campaign.ID == 0 {
				t.Fatalf("campaign ID is 0")
			}

			t.Run("apply same campaignSpec", func(t *testing.T) {
				campaign2, err := svc.ApplyCampaign(ctx, ApplyCampaignOpts{
					NamespaceUserID:    user.ID,
					CampaignSpecRandID: campaignSpec.RandID,
				})
				if err != nil {
					t.Fatal(err)
				}

				if have, want := campaign2.ID, campaign.ID; have != want {
					t.Fatalf("campaign ID is wrong. want=%d, have=%d", want, have)
				}
			})

			t.Run("apply campaign spec with same name", func(t *testing.T) {
				campaignSpec2 := createCampaignSpec(t, "campaign-name", user.ID)
				campaign2, err := svc.ApplyCampaign(ctx, ApplyCampaignOpts{
					NamespaceUserID:    user.ID,
					CampaignSpecRandID: campaignSpec2.RandID,
				})
				if err != nil {
					t.Fatal(err)
				}

				if have, want := campaign2.ID, campaign.ID; have != want {
					t.Fatalf("campaign ID is wrong. want=%d, have=%d", want, have)
				}
			})

			t.Run("apply campaign spec with same name but different namespace", func(t *testing.T) {
				user2 := createTestUser(ctx, t)
				campaignSpec2 := createCampaignSpec(t, "campaign-name", user2.ID)

				campaign2, err := svc.ApplyCampaign(ctx, ApplyCampaignOpts{
					NamespaceUserID:    user2.ID,
					CampaignSpecRandID: campaignSpec2.RandID,
				})
				if err != nil {
					t.Fatal(err)
				}

				if campaign2.ID == 0 {
					t.Fatalf("campaign2 ID is 0")
				}

				if campaign2.ID == campaign.ID {
					t.Fatalf("campaign IDs are the same, but want different")
				}
			})

			t.Run("campaign spec with same name and same ensureCampaignID", func(t *testing.T) {
				campaignSpec2 := createCampaignSpec(t, "campaign-name", user.ID)

				campaign2, err := svc.ApplyCampaign(ctx, ApplyCampaignOpts{
					NamespaceUserID:    user.ID,
					CampaignSpecRandID: campaignSpec2.RandID,
					EnsureCampaignID:   campaign.ID,
				})
				if err != nil {
					t.Fatal(err)
				}
				if have, want := campaign2.ID, campaign.ID; have != want {
					t.Fatalf("campaign has wrong ID. want=%d, have=%d", want, have)
				}
			})

			t.Run("campaign spec with same name but different ensureCampaignID", func(t *testing.T) {
				campaignSpec2 := createCampaignSpec(t, "campaign-name", user.ID)

				_, err := svc.ApplyCampaign(ctx, ApplyCampaignOpts{
					NamespaceUserID:    user.ID,
					CampaignSpecRandID: campaignSpec2.RandID,
					EnsureCampaignID:   campaign.ID + 999,
				})
				if err != ErrEnsureCampaignFailed {
					t.Fatalf("wrong error: %s", err)
				}
			})
		})
	})
}

var testUser = db.NewUser{
	Email:                 "thorsten@sourcegraph.com",
	Username:              "thorsten",
	DisplayName:           "thorsten",
	Password:              "1234",
	EmailVerificationCode: "foobar",
}

var createTestUser = func() func(context.Context, *testing.T) *types.User {
	count := 0

	return func(ctx context.Context, t *testing.T) *types.User {
		t.Helper()

		u := testUser
		u.Username = fmt.Sprintf("%s-%d", u.Username, count)

		user, err := db.Users.Create(ctx, u)
		if err != nil {
			t.Fatal(err)
		}

		count += 1

		return user
	}
}()

func testPatch(patchSet int64, repo api.RepoID, t time.Time) *campaigns.Patch {
	return &campaigns.Patch{
		PatchSetID: patchSet,
		RepoID:     repo,
		Rev:        "deadbeef",
		BaseRef:    "refs/heads/master",
		Diff:       "cool diff",
	}
}

func testCampaign(user int32, patchSet int64) *campaigns.Campaign {
	c := &campaigns.Campaign{
		Name:            "Testing Campaign",
		Description:     "Testing Campaign",
		AuthorID:        user,
		NamespaceUserID: user,
		PatchSetID:      patchSet,
	}

	if patchSet != 0 {
		c.Branch = "test-branch"
	}

	return c
}

func testChangeset(repoID api.RepoID, campaign int64, changesetJob int64, state campaigns.ChangesetState) *campaigns.Changeset {
	changeset := &campaigns.Changeset{
		RepoID:              repoID,
		ExternalServiceType: extsvc.TypeGitHub,
		ExternalID:          fmt.Sprintf("ext-id-%d", changesetJob),
		Metadata:            &github.PullRequest{State: string(state)},
		ExternalState:       state,
	}

	if campaign != 0 {
		changeset.CampaignIDs = []int64{campaign}
	}

	return changeset
}

func authzFilterRepo(t *testing.T, id api.RepoID) {
	db.MockAuthzFilter = func(ctx context.Context, repos []*types.Repo, p authz.Perms) ([]*types.Repo, error) {
		var filtered []*types.Repo
		for _, r := range repos {
			if r.ID == id {
				continue
			}
			filtered = append(filtered, r)
		}
		return filtered, nil
	}
	t.Cleanup(func() { db.MockAuthzFilter = nil })
}
