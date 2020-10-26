package campaigns

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
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

	rs, _ := createTestRepos(t, ctx, dbconn.Global, 1)

	createTestData := func(t *testing.T, s *Store, svc *Service, author int32) (*campaigns.Campaign, *campaigns.Changeset, *campaigns.CampaignSpec) {
		spec := testCampaignSpec(author)
		if err := s.CreateCampaignSpec(ctx, spec); err != nil {
			t.Fatal(err)
		}

		campaign := testCampaign(author, spec)
		if err := s.CreateCampaign(ctx, campaign); err != nil {
			t.Fatal(err)
		}

		changeset := testChangeset(rs[0].ID, campaign.ID, campaigns.ChangesetExternalStateOpen)
		if err := s.CreateChangeset(ctx, changeset); err != nil {
			t.Fatal(err)
		}

		campaign.ChangesetIDs = append(campaign.ChangesetIDs, changeset.ID)
		if err := s.UpdateCampaign(ctx, campaign); err != nil {
			t.Fatal(err)
		}

		return campaign, changeset, spec
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
			campaign, changeset, campaignSpec := createTestData(t, store, svc, tc.campaignAuthor)
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

			t.Run("CloseCampaign", func(t *testing.T) {
				_, err := svc.CloseCampaign(currentUserCtx, campaign.ID, false)
				tc.assertFunc(t, err)
			})

			t.Run("DeleteCampaign", func(t *testing.T) {
				err := svc.DeleteCampaign(currentUserCtx, campaign.ID)
				tc.assertFunc(t, err)
			})

			t.Run("MoveCampaign", func(t *testing.T) {
				_, err := svc.MoveCampaign(currentUserCtx, MoveCampaignOpts{
					CampaignID: campaign.ID,
					NewName:    "foobar2",
				})
				tc.assertFunc(t, err)
			})

			t.Run("ApplyCampaign", func(t *testing.T) {
				_, err := svc.ApplyCampaign(currentUserCtx, ApplyCampaignOpts{
					CampaignSpecRandID: campaignSpec.RandID,
				})
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

	admin := createTestUser(ctx, t)
	if !admin.SiteAdmin {
		t.Fatal("admin is not a site-admin")
	}

	user := createTestUser(ctx, t)
	if user.SiteAdmin {
		t.Fatal("user is admin, want non-admin")
	}

	now := time.Now().UTC().Truncate(time.Microsecond)
	clock := func() time.Time { return now }

	store := NewStoreWithClock(dbconn.Global, clock)
	rs, _ := createTestRepos(t, ctx, dbconn.Global, 4)

	fakeSource := &ct.FakeChangesetSource{}
	sourcer := repos.NewFakeSourcer(nil, fakeSource)

	svc := NewService(store, nil)
	svc.sourcer = sourcer

	t.Run("DeleteCampaign", func(t *testing.T) {
		spec := testCampaignSpec(admin.ID)
		if err := store.CreateCampaignSpec(ctx, spec); err != nil {
			t.Fatal(err)
		}

		campaign := testCampaign(admin.ID, spec)
		if err := store.CreateCampaign(ctx, campaign); err != nil {
			t.Fatal(err)
		}
		if err := svc.DeleteCampaign(ctx, campaign.ID); err != nil {
			t.Fatalf("campaign not deleted: %s", err)
		}

		_, err := store.GetCampaign(ctx, GetCampaignOpts{ID: campaign.ID})
		if err != nil && err != ErrNoResults {
			t.Fatalf("want campaign to be deleted, but was not: %e", err)
		}
	})

	t.Run("CloseCampaign", func(t *testing.T) {
		createCampaign := func(t *testing.T) *campaigns.Campaign {
			t.Helper()

			spec := testCampaignSpec(admin.ID)
			if err := store.CreateCampaignSpec(ctx, spec); err != nil {
				t.Fatal(err)
			}

			campaign := testCampaign(admin.ID, spec)
			if err := store.CreateCampaign(ctx, campaign); err != nil {
				t.Fatal(err)
			}
			return campaign
		}

		adminCtx := actor.WithActor(context.Background(), actor.FromUser(admin.ID))

		closeConfirm := func(t *testing.T, c *campaigns.Campaign, closeChangesets bool) {
			t.Helper()

			closedCampaign, err := svc.CloseCampaign(adminCtx, c.ID, closeChangesets)
			if err != nil {
				t.Fatalf("campaign not closed: %s", err)
			}
			if !closedCampaign.ClosedAt.Equal(now) {
				t.Fatalf("campaign ClosedAt is zero")
			}

			if !closeChangesets {
				return
			}

			cs, _, err := store.ListChangesets(ctx, ListChangesetsOpts{
				OwnedByCampaignID: c.ID,
			})
			if err != nil {
				t.Fatalf("listing changesets failed: %s", err)
			}
			for _, c := range cs {
				if !c.Closing {
					t.Errorf("changeset should be Closing, but is not")
				}

				if have, want := c.ReconcilerState, campaigns.ReconcilerStateQueued; have != want {
					t.Errorf("changeset ReconcilerState wrong. want=%s, have=%s", want, have)
				}
			}
		}

		t.Run("no changesets", func(t *testing.T) {
			campaign := createCampaign(t)
			closeConfirm(t, campaign, false)
		})

		t.Run("changesets", func(t *testing.T) {
			campaign := createCampaign(t)

			changeset1 := testChangeset(rs[0].ID, campaign.ID, campaigns.ChangesetExternalStateOpen)
			changeset1.ReconcilerState = campaigns.ReconcilerStateCompleted
			if err := store.CreateChangeset(ctx, changeset1); err != nil {
				t.Fatal(err)
			}

			changeset2 := testChangeset(rs[1].ID, campaign.ID, campaigns.ChangesetExternalStateOpen)
			changeset2.ReconcilerState = campaigns.ReconcilerStateCompleted
			if err := store.CreateChangeset(ctx, changeset2); err != nil {
				t.Fatal(err)
			}

			closeConfirm(t, campaign, true)
		})
	})

	t.Run("EnqueueChangesetSync", func(t *testing.T) {
		spec := testCampaignSpec(admin.ID)
		if err := store.CreateCampaignSpec(ctx, spec); err != nil {
			t.Fatal(err)
		}

		campaign := testCampaign(admin.ID, spec)
		if err := store.CreateCampaign(ctx, campaign); err != nil {
			t.Fatal(err)
		}

		changeset := testChangeset(rs[0].ID, campaign.ID, campaigns.ChangesetExternalStateOpen)
		if err := store.CreateChangeset(ctx, changeset); err != nil {
			t.Fatal(err)
		}

		campaign.ChangesetIDs = []int64{changeset.ID}
		if err := store.UpdateCampaign(ctx, campaign); err != nil {
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
		ct.AuthzFilterRepos(t, rs[0].ID)

		// should result in a not found error
		if err := svc.EnqueueChangesetSync(ctx, changeset.ID); !errcode.IsNotFound(err) {
			t.Fatalf("expected not-found error but got %s", err)
		}
	})

	t.Run("CreateCampaignSpec", func(t *testing.T) {
		changesetSpecs := make([]*campaigns.ChangesetSpec, 0, len(rs))
		changesetSpecRandIDs := make([]string, 0, len(rs))
		for _, r := range rs {
			cs := &campaigns.ChangesetSpec{RepoID: r.ID, UserID: admin.ID}
			if err := store.CreateChangesetSpec(ctx, cs); err != nil {
				t.Fatal(err)
			}
			changesetSpecs = append(changesetSpecs, cs)
			changesetSpecRandIDs = append(changesetSpecRandIDs, cs.RandID)
		}

		adminCtx := actor.WithActor(context.Background(), actor.FromUser(admin.ID))

		t.Run("success", func(t *testing.T) {
			opts := CreateCampaignSpecOpts{
				NamespaceUserID:      admin.ID,
				RawSpec:              ct.TestRawCampaignSpec,
				ChangesetSpecRandIDs: changesetSpecRandIDs,
			}

			spec, err := svc.CreateCampaignSpec(adminCtx, opts)
			if err != nil {
				t.Fatal(err)
			}

			if spec.ID == 0 {
				t.Fatalf("CampaignSpec ID is 0")
			}

			if have, want := spec.UserID, admin.ID; have != want {
				t.Fatalf("UserID is %d, want %d", have, want)
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

		t.Run("success with YAML raw spec", func(t *testing.T) {
			opts := CreateCampaignSpecOpts{
				NamespaceUserID: admin.ID,
				RawSpec:         ct.TestRawCampaignSpecYAML,
			}

			spec, err := svc.CreateCampaignSpec(adminCtx, opts)
			if err != nil {
				t.Fatal(err)
			}

			if spec.ID == 0 {
				t.Fatalf("CampaignSpec ID is 0")
			}

			var wantFields campaigns.CampaignSpecFields
			if err := json.Unmarshal([]byte(ct.TestRawCampaignSpec), &wantFields); err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(wantFields, spec.Spec); diff != "" {
				t.Fatalf("wrong spec fields (-want +got):\n%s", diff)
			}
		})

		t.Run("missing repository permissions", func(t *testing.T) {
			// Single repository filtered out by authzFilter
			ct.AuthzFilterRepos(t, changesetSpecs[0].RepoID)

			opts := CreateCampaignSpecOpts{
				NamespaceUserID:      admin.ID,
				RawSpec:              ct.TestRawCampaignSpec,
				ChangesetSpecRandIDs: changesetSpecRandIDs,
			}

			if _, err := svc.CreateCampaignSpec(adminCtx, opts); !errcode.IsNotFound(err) {
				t.Fatalf("expected not-found error but got %s", err)
			}
		})

		t.Run("invalid changesetspec id", func(t *testing.T) {
			containsInvalidID := []string{changesetSpecRandIDs[0], "foobar"}
			opts := CreateCampaignSpecOpts{
				NamespaceUserID:      admin.ID,
				RawSpec:              ct.TestRawCampaignSpec,
				ChangesetSpecRandIDs: containsInvalidID,
			}

			if _, err := svc.CreateCampaignSpec(adminCtx, opts); !errcode.IsNotFound(err) {
				t.Fatalf("expected not-found error but got %s", err)
			}
		})

		t.Run("namespace user is not admin and not creator", func(t *testing.T) {
			userCtx := actor.WithActor(context.Background(), actor.FromUser(user.ID))

			opts := CreateCampaignSpecOpts{
				NamespaceUserID: admin.ID,
				RawSpec:         ct.TestRawCampaignSpecYAML,
			}

			_, err := svc.CreateCampaignSpec(userCtx, opts)
			if !errcode.IsUnauthorized(err) {
				t.Fatalf("expected unauthorized error but got %s", err)
			}

			// Try again as admin
			adminCtx := actor.WithActor(context.Background(), actor.FromUser(admin.ID))

			opts.NamespaceUserID = user.ID

			_, err = svc.CreateCampaignSpec(adminCtx, opts)
			if err != nil {
				t.Fatalf("expected no error but got %s", err)
			}
		})

		t.Run("missing access to namespace org", func(t *testing.T) {
			org, err := db.Orgs.Create(ctx, "test-org", nil)
			if err != nil {
				t.Fatal(err)
			}

			opts := CreateCampaignSpecOpts{
				NamespaceOrgID:       org.ID,
				RawSpec:              ct.TestRawCampaignSpec,
				ChangesetSpecRandIDs: changesetSpecRandIDs,
			}

			userCtx := actor.WithActor(context.Background(), actor.FromUser(user.ID))

			_, err = svc.CreateCampaignSpec(userCtx, opts)
			if have, want := err, backend.ErrNotAnOrgMember; have != want {
				t.Fatalf("expected %s error but got %s", want, have)
			}

			// Create org membership and try again
			if _, err := db.OrgMembers.Create(ctx, org.ID, user.ID); err != nil {
				t.Fatal(err)
			}

			_, err = svc.CreateCampaignSpec(userCtx, opts)
			if err != nil {
				t.Fatalf("expected no error but got %s", err)
			}
		})

		t.Run("no side-effects if no changeset spec IDs are given", func(t *testing.T) {
			// We already have ChangesetSpecs in the database. Here we
			// want to make sure that the new CampaignSpec is created,
			// without accidently attaching the existing ChangesetSpecs.
			opts := CreateCampaignSpecOpts{
				NamespaceUserID:      admin.ID,
				RawSpec:              ct.TestRawCampaignSpec,
				ChangesetSpecRandIDs: []string{},
			}

			spec, err := svc.CreateCampaignSpec(adminCtx, opts)
			if err != nil {
				t.Fatal(err)
			}

			countOpts := CountChangesetSpecsOpts{CampaignSpecID: spec.ID}
			count, err := store.CountChangesetSpecs(adminCtx, countOpts)
			if err != nil {
				return
			}
			if count != 0 {
				t.Fatalf("want no changeset specs attached to campaign spec, but have %d", count)
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

			wantFields := &campaigns.ChangesetSpecDescription{}
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
			// Single repository filtered out by authzFilter
			ct.AuthzFilterRepos(t, repo.ID)

			_, err := svc.CreateChangesetSpec(ctx, rawSpec, admin.ID)
			if !errcode.IsNotFound(err) {
				t.Fatalf("expected not-found error but got %s", err)
			}
		})
	})

	t.Run("ApplyCampaign", func(t *testing.T) {
		// See TestServiceApplyCampaign
	})

	t.Run("MoveCampaign", func(t *testing.T) {
		createCampaign := func(t *testing.T, name string, authorID, userID, orgID int32) *campaigns.Campaign {
			t.Helper()

			spec := &campaigns.CampaignSpec{
				UserID:          authorID,
				NamespaceUserID: userID,
				NamespaceOrgID:  orgID,
			}

			if err := store.CreateCampaignSpec(ctx, spec); err != nil {
				t.Fatal(err)
			}

			c := &campaigns.Campaign{
				InitialApplierID: authorID,
				NamespaceUserID:  userID,
				NamespaceOrgID:   orgID,
				Name:             name,
				LastApplierID:    authorID,
				LastAppliedAt:    time.Now(),
				CampaignSpecID:   spec.ID,
			}

			if err := store.CreateCampaign(ctx, c); err != nil {
				t.Fatal(err)
			}

			return c
		}

		t.Run("new name", func(t *testing.T) {
			campaign := createCampaign(t, "old-name", admin.ID, admin.ID, 0)

			opts := MoveCampaignOpts{CampaignID: campaign.ID, NewName: "new-name"}
			moved, err := svc.MoveCampaign(ctx, opts)
			if err != nil {
				t.Fatal(err)
			}

			if have, want := moved.Name, opts.NewName; have != want {
				t.Fatalf("wrong name. want=%q, have=%q", want, have)
			}
		})

		t.Run("new user namespace", func(t *testing.T) {
			campaign := createCampaign(t, "old-name", admin.ID, admin.ID, 0)

			user2 := createTestUser(ctx, t)

			opts := MoveCampaignOpts{CampaignID: campaign.ID, NewNamespaceUserID: user2.ID}
			moved, err := svc.MoveCampaign(ctx, opts)
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
			campaign := createCampaign(t, "old-name", user.ID, user.ID, 0)

			user2 := createTestUser(ctx, t)

			opts := MoveCampaignOpts{CampaignID: campaign.ID, NewNamespaceUserID: user2.ID}

			userCtx := actor.WithActor(context.Background(), actor.FromUser(user.ID))
			_, err := svc.MoveCampaign(userCtx, opts)
			if !errcode.IsUnauthorized(err) {
				t.Fatalf("expected unauthorized error but got %s", err)
			}
		})

		t.Run("new org namespace", func(t *testing.T) {
			campaign := createCampaign(t, "old-name", admin.ID, admin.ID, 0)

			org, err := db.Orgs.Create(ctx, "org", nil)
			if err != nil {
				t.Fatal(err)
			}

			opts := MoveCampaignOpts{CampaignID: campaign.ID, NewNamespaceOrgID: org.ID}
			moved, err := svc.MoveCampaign(ctx, opts)
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
			campaign := createCampaign(t, "old-name", user.ID, user.ID, 0)

			org, err := db.Orgs.Create(ctx, "org-no-access", nil)
			if err != nil {
				t.Fatal(err)
			}

			opts := MoveCampaignOpts{CampaignID: campaign.ID, NewNamespaceOrgID: org.ID}

			userCtx := actor.WithActor(context.Background(), actor.FromUser(user.ID))
			_, err = svc.MoveCampaign(userCtx, opts)
			if have, want := err, backend.ErrNotAnOrgMember; have != want {
				t.Fatalf("expected %s error but got %s", want, have)
			}
		})
	})

	t.Run("GetCampaignMatchingCampaignSpec", func(t *testing.T) {
		campaignSpec := createCampaignSpec(t, ctx, store, "matching-campaign-spec", admin.ID)

		haveCampaign, err := svc.GetCampaignMatchingCampaignSpec(ctx, store, campaignSpec)
		if err != nil {
			t.Fatalf("unexpected error: %s\n", err)
		}
		if haveCampaign != nil {
			t.Fatalf("expected campaign to be nil, but is not: %+v\n", haveCampaign)
		}

		matchingCampaign := &campaigns.Campaign{
			Name:             campaignSpec.Spec.Name,
			Description:      campaignSpec.Spec.Description,
			InitialApplierID: admin.ID,
			NamespaceOrgID:   campaignSpec.NamespaceOrgID,
			NamespaceUserID:  campaignSpec.NamespaceUserID,
			CampaignSpecID:   campaignSpec.ID,
			LastApplierID:    admin.ID,
			LastAppliedAt:    time.Now(),
		}
		if err := store.CreateCampaign(ctx, matchingCampaign); err != nil {
			t.Fatalf("failed to create campaign: %s\n", err)
		}

		haveCampaign, err = svc.GetCampaignMatchingCampaignSpec(ctx, store, campaignSpec)
		if err != nil {
			t.Fatalf("unexpected error: %s\n", err)
		}
		if haveCampaign == nil {
			t.Fatalf("expected to have matching campaign, but got nil")
		}

		if diff := cmp.Diff(matchingCampaign, haveCampaign); diff != "" {
			t.Fatalf("wrong campaign was matched (-want +got):\n%s", diff)
		}
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

func testCampaign(user int32, spec *campaigns.CampaignSpec) *campaigns.Campaign {
	c := &campaigns.Campaign{
		Name:             "test-campaign",
		InitialApplierID: user,
		NamespaceUserID:  user,
		CampaignSpecID:   spec.ID,
		LastApplierID:    user,
		LastAppliedAt:    time.Now(),
	}

	return c
}

func testCampaignSpec(user int32) *campaigns.CampaignSpec {
	return &campaigns.CampaignSpec{
		UserID:          user,
		NamespaceUserID: user,
	}
}

func testChangeset(repoID api.RepoID, campaign int64, extState campaigns.ChangesetExternalState) *campaigns.Changeset {
	changeset := &campaigns.Changeset{
		RepoID:              repoID,
		ExternalServiceType: extsvc.TypeGitHub,
		ExternalID:          fmt.Sprintf("ext-id-%d", campaign),
		Metadata:            &github.PullRequest{State: string(extState), CreatedAt: time.Now()},
		ExternalState:       extState,
	}

	if campaign != 0 {
		changeset.CampaignIDs = []int64{campaign}
	}

	return changeset
}

func createTestRepos(t *testing.T, ctx context.Context, db *sql.DB, count int) ([]*repos.Repo, *repos.ExternalService) {
	t.Helper()

	rstore := repos.NewDBStore(db, sql.TxOptions{})

	ext := &repos.ExternalService{
		Kind:        extsvc.KindGitHub,
		DisplayName: "GitHub",
		Config: marshalJSON(t, &schema.GitHubConnection{
			Url:   "https://github.com",
			Token: "SECRETTOKEN",
		}),
	}
	if err := rstore.UpsertExternalServices(ctx, ext); err != nil {
		t.Fatal(err)
	}

	var rs []*repos.Repo
	for i := 0; i < count; i++ {
		r := testRepo(t, rstore, extsvc.KindGitHub)
		r.Sources = map[string]*repos.SourceInfo{ext.URN(): {ID: ext.URN()}}

		rs = append(rs, r)
	}

	err := rstore.InsertRepos(ctx, rs...)
	if err != nil {
		t.Fatal(err)
	}

	return rs, ext
}

func createBbsTestRepos(t *testing.T, ctx context.Context, db *sql.DB, count int) ([]*repos.Repo, *repos.ExternalService) {
	t.Helper()

	rstore := repos.NewDBStore(db, sql.TxOptions{})

	ext := &repos.ExternalService{
		Kind:        extsvc.KindBitbucketServer,
		DisplayName: "Bitbucket Server",
		Config: marshalJSON(t, &schema.BitbucketServerConnection{
			Url:   "https://bitbucket.sourcegraph.com",
			Token: "SECRETTOKEN",
		}),
	}
	if err := rstore.UpsertExternalServices(ctx, ext); err != nil {
		t.Fatal(err)
	}

	var rs []*repos.Repo
	for i := 0; i < count; i++ {
		r := testRepo(t, rstore, extsvc.KindBitbucketServer)
		r.Sources = map[string]*repos.SourceInfo{ext.URN(): {ID: ext.URN()}}

		rs = append(rs, r)
	}

	err := rstore.InsertRepos(ctx, rs...)
	if err != nil {
		t.Fatal(err)
	}

	return rs, ext
}
