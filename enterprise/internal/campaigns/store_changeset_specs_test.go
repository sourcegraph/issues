package campaigns

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/sourcegraph/sourcegraph/cmd/repo-updater/repos"
	cmpgn "github.com/sourcegraph/sourcegraph/internal/campaigns"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
)

func testStoreChangesetSpecs(t *testing.T, ctx context.Context, s *Store, rs repos.Store, clock clock) {
	repo := testRepo(t, rs, extsvc.KindGitHub)
	deletedRepo := testRepo(t, rs, extsvc.KindGitHub).With(repos.Opt.RepoDeletedAt(clock.now()))

	if err := rs.InsertRepos(ctx, repo); err != nil {
		t.Fatal(err)
	}
	if err := rs.DeleteRepos(ctx, deletedRepo.ID); err != nil {
		t.Fatal(err)
	}

	changesetSpecs := make(cmpgn.ChangesetSpecs, 0, 3)
	for i := 0; i < cap(changesetSpecs); i++ {
		c := &cmpgn.ChangesetSpec{
			RawSpec: `{"externalID":"12345"}`,
			Spec: &cmpgn.ChangesetSpecDescription{
				ExternalID: "123456",
			},
			UserID:         int32(i + 1234),
			CampaignSpecID: int64(i + 910),
			RepoID:         repo.ID,

			DiffStatAdded:   123,
			DiffStatChanged: 456,
			DiffStatDeleted: 789,
		}

		if i == cap(changesetSpecs)-1 {
			c.CampaignSpecID = 0
		}
		changesetSpecs = append(changesetSpecs, c)
	}

	// We create this ChangesetSpec to make sure that it's not returned when
	// listing or getting ChangesetSpecs, since we don't want to load
	// ChangesetSpecs whose repository has been (soft-)deleted.
	changesetSpecDeletedRepo := &cmpgn.ChangesetSpec{
		UserID:         int32(424242),
		Spec:           &cmpgn.ChangesetSpecDescription{},
		CampaignSpecID: int64(424242),
		RawSpec:        `{}`,
		RepoID:         deletedRepo.ID,
	}

	t.Run("Create", func(t *testing.T) {
		toCreate := make(cmpgn.ChangesetSpecs, 0, len(changesetSpecs)+1)
		toCreate = append(toCreate, changesetSpecDeletedRepo)
		toCreate = append(toCreate, changesetSpecs...)

		for _, c := range toCreate {
			want := c.Clone()
			have := c

			err := s.CreateChangesetSpec(ctx, have)
			if err != nil {
				t.Fatal(err)
			}

			if have.ID == 0 {
				t.Fatal("ID should not be zero")
			}

			if have.RandID == "" {
				t.Fatal("RandID should not be empty")
			}

			want.ID = have.ID
			want.RandID = have.RandID
			want.CreatedAt = clock.now()
			want.UpdatedAt = clock.now()

			if diff := cmp.Diff(have, want); diff != "" {
				t.Fatal(diff)
			}
		}
	})

	t.Run("Count", func(t *testing.T) {
		count, err := s.CountChangesetSpecs(ctx, CountChangesetSpecsOpts{})
		if err != nil {
			t.Fatal(err)
		}

		if have, want := count, len(changesetSpecs); have != want {
			t.Fatalf("have count: %d, want: %d", have, want)
		}

		t.Run("WithCampaignSpecID", func(t *testing.T) {
			testsRan := false
			for _, c := range changesetSpecs {
				if c.CampaignSpecID == 0 {
					continue
				}

				opts := CountChangesetSpecsOpts{CampaignSpecID: c.CampaignSpecID}
				subCount, err := s.CountChangesetSpecs(ctx, opts)
				if err != nil {
					t.Fatal(err)
				}

				if have, want := subCount, 1; have != want {
					t.Fatalf("have count: %d, want: %d", have, want)
				}
				testsRan = true
			}

			if !testsRan {
				t.Fatal("no changesetSpec has a non-zero CampaignSpecID")
			}
		})
	})

	t.Run("List", func(t *testing.T) {
		t.Run("NoLimit", func(t *testing.T) {
			// Empty limit should return all entries.
			opts := ListChangesetSpecsOpts{}
			ts, next, err := s.ListChangesetSpecs(ctx, opts)
			if err != nil {
				t.Fatal(err)
			}

			if have, want := next, int64(0); have != want {
				t.Fatalf("opts: %+v: have next %v, want %v", opts, have, want)
			}

			have, want := ts, changesetSpecs
			if len(have) != len(want) {
				t.Fatalf("listed %d changesetSpecs, want: %d", len(have), len(want))
			}

			if diff := cmp.Diff(have, want); diff != "" {
				t.Fatalf("opts: %+v, diff: %s", opts, diff)
			}
		})

		t.Run("WithLimit", func(t *testing.T) {
			for i := 1; i <= len(changesetSpecs); i++ {
				cs, next, err := s.ListChangesetSpecs(ctx, ListChangesetSpecsOpts{LimitOpts: LimitOpts{Limit: i}})
				if err != nil {
					t.Fatal(err)
				}

				{
					have, want := next, int64(0)
					if i < len(changesetSpecs) {
						want = changesetSpecs[i].ID
					}

					if have != want {
						t.Fatalf("limit: %v: have next %v, want %v", i, have, want)
					}
				}

				{
					have, want := cs, changesetSpecs[:i]
					if len(have) != len(want) {
						t.Fatalf("listed %d changesetSpecs, want: %d", len(have), len(want))
					}

					if diff := cmp.Diff(have, want); diff != "" {
						t.Fatal(diff)
					}
				}
			}
		})

		t.Run("WithLimitAndCursor", func(t *testing.T) {
			var cursor int64
			for i := 1; i <= len(changesetSpecs); i++ {
				opts := ListChangesetSpecsOpts{Cursor: cursor, LimitOpts: LimitOpts{Limit: 1}}
				have, next, err := s.ListChangesetSpecs(ctx, opts)
				if err != nil {
					t.Fatal(err)
				}

				want := changesetSpecs[i-1 : i]
				if diff := cmp.Diff(have, want); diff != "" {
					t.Fatalf("opts: %+v, diff: %s", opts, diff)
				}

				cursor = next
			}
		})

		t.Run("WithCampaignSpecID", func(t *testing.T) {
			for _, c := range changesetSpecs {
				if c.CampaignSpecID == 0 {
					continue
				}
				opts := ListChangesetSpecsOpts{CampaignSpecID: c.CampaignSpecID}
				have, _, err := s.ListChangesetSpecs(ctx, opts)
				if err != nil {
					t.Fatal(err)
				}

				want := cmpgn.ChangesetSpecs{c}
				if diff := cmp.Diff(have, want); diff != "" {
					t.Fatalf("opts: %+v, diff: %s", opts, diff)
				}
			}
		})

		t.Run("WithRandIDs", func(t *testing.T) {
			for _, c := range changesetSpecs {
				opts := ListChangesetSpecsOpts{RandIDs: []string{c.RandID}}
				have, _, err := s.ListChangesetSpecs(ctx, opts)
				if err != nil {
					t.Fatal(err)
				}

				want := cmpgn.ChangesetSpecs{c}
				if diff := cmp.Diff(have, want); diff != "" {
					t.Fatalf("opts: %+v, diff: %s", opts, diff)
				}
			}

			opts := ListChangesetSpecsOpts{}
			for _, c := range changesetSpecs {
				opts.RandIDs = append(opts.RandIDs, c.RandID)
			}

			have, _, err := s.ListChangesetSpecs(ctx, opts)
			if err != nil {
				t.Fatal(err)
			}

			// ListChangesetSpecs should not return ChangesetSpecs whose
			// repository was (soft-)deleted.
			if diff := cmp.Diff(have, changesetSpecs); diff != "" {
				t.Fatalf("opts: %+v, diff: %s", opts, diff)
			}
		})
	})

	t.Run("Update", func(t *testing.T) {
		for _, c := range changesetSpecs {
			c.UserID += 1234
			c.DiffStatAdded += 1234
			c.DiffStatChanged += 1234
			c.DiffStatDeleted += 1234

			clock.add(1 * time.Second)

			want := c
			want.UpdatedAt = clock.now()

			have := c.Clone()
			if err := s.UpdateChangesetSpec(ctx, have); err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(have, want); diff != "" {
				t.Fatal(diff)
			}
		}
	})

	t.Run("Get", func(t *testing.T) {
		want := changesetSpecs[1]
		tests := map[string]GetChangesetSpecOpts{
			"ByID":          {ID: want.ID},
			"ByRandID":      {RandID: want.RandID},
			"ByIDAndRandID": {ID: want.ID, RandID: want.RandID},
		}

		for name, opts := range tests {
			t.Run(name, func(t *testing.T) {
				have, err := s.GetChangesetSpec(ctx, opts)
				if err != nil {
					t.Fatal(err)
				}

				if diff := cmp.Diff(have, want); diff != "" {
					t.Fatal(diff)
				}
			})
		}

		t.Run("NoResults", func(t *testing.T) {
			opts := GetChangesetSpecOpts{ID: 0xdeadbeef}

			_, have := s.GetChangesetSpec(ctx, opts)
			want := ErrNoResults

			if have != want {
				t.Fatalf("have err %v, want %v", have, want)
			}
		})
	})

	t.Run("Delete", func(t *testing.T) {
		for i := range changesetSpecs {
			err := s.DeleteChangesetSpec(ctx, changesetSpecs[i].ID)
			if err != nil {
				t.Fatal(err)
			}

			count, err := s.CountChangesetSpecs(ctx, CountChangesetSpecsOpts{})
			if err != nil {
				t.Fatal(err)
			}

			if have, want := count, len(changesetSpecs)-(i+1); have != want {
				t.Fatalf("have count: %d, want: %d", have, want)
			}
		}
	})

	t.Run("DeleteExpiredChangesetSpecs", func(t *testing.T) {
		underTTL := clock.now().Add(-cmpgn.ChangesetSpecTTL + 24*time.Hour)
		overTTL := clock.now().Add(-cmpgn.ChangesetSpecTTL - 24*time.Hour)

		type testCase struct {
			createdAt time.Time

			hasCampaignSpec     bool
			campaignSpecApplied bool

			isCurrentSpec  bool
			isPreviousSpec bool

			wantDeleted bool
		}

		printTestCase := func(tc testCase) string {
			var tooOld bool
			if tc.createdAt.Equal(overTTL) {
				tooOld = true
			}

			return fmt.Sprintf(
				"[tooOld=%t, hasCampaignSpec=%t, campaignSpecApplied=%t, isCurrentSpec=%t, isPreviousSpec=%t]",
				tooOld, tc.hasCampaignSpec, tc.campaignSpecApplied, tc.isCurrentSpec, tc.isPreviousSpec,
			)
		}

		tests := []testCase{
			// ChangesetSpec was created but never attached to a CampaignSpec
			{hasCampaignSpec: false, createdAt: underTTL, wantDeleted: false},
			{hasCampaignSpec: false, createdAt: overTTL, wantDeleted: true},

			// Attached to CampaignSpec that's applied to a Campaign
			{hasCampaignSpec: true, campaignSpecApplied: true, isCurrentSpec: true, createdAt: underTTL, wantDeleted: false},
			{hasCampaignSpec: true, campaignSpecApplied: true, isCurrentSpec: true, createdAt: overTTL, wantDeleted: false},

			// CampaignSpec is not applied to a Campaign anymore and the
			// ChangesetSpecs are now the PreviousSpec.
			{hasCampaignSpec: true, isPreviousSpec: true, createdAt: underTTL, wantDeleted: false},
			{hasCampaignSpec: true, isPreviousSpec: true, createdAt: overTTL, wantDeleted: false},

			// Has a CampaignSpec, but that CampaignSpec is not applied
			// anymore, and the ChangesetSpec is neither the current, nor the
			// previous spec.
			{hasCampaignSpec: true, createdAt: underTTL, wantDeleted: false},
			{hasCampaignSpec: true, createdAt: overTTL, wantDeleted: true},
		}

		for _, tc := range tests {
			campaignSpec := &cmpgn.CampaignSpec{UserID: 4567, NamespaceUserID: 4567}

			if tc.hasCampaignSpec {
				if err := s.CreateCampaignSpec(ctx, campaignSpec); err != nil {
					t.Fatal(err)
				}

				if tc.campaignSpecApplied {
					campaign := &cmpgn.Campaign{
						Name:             fmt.Sprintf("campaign for spec %d", campaignSpec.ID),
						CampaignSpecID:   campaignSpec.ID,
						InitialApplierID: campaignSpec.UserID,
						NamespaceUserID:  campaignSpec.NamespaceUserID,
						LastApplierID:    campaignSpec.UserID,
						LastAppliedAt:    time.Now(),
					}
					if err := s.CreateCampaign(ctx, campaign); err != nil {
						t.Fatal(err)
					}
				}
			}

			changesetSpec := &cmpgn.ChangesetSpec{
				CampaignSpecID: campaignSpec.ID,
				// Need to set a RepoID otherwise GetChangesetSpec filters it out.
				RepoID:    repo.ID,
				CreatedAt: tc.createdAt,
			}

			if err := s.CreateChangesetSpec(ctx, changesetSpec); err != nil {
				t.Fatal(err)
			}

			if tc.isCurrentSpec {
				changeset := &cmpgn.Changeset{
					ExternalServiceType: "github",
					RepoID:              1,
					CurrentSpecID:       changesetSpec.ID,
				}
				if err := s.CreateChangeset(ctx, changeset); err != nil {
					t.Fatal(err)
				}
			}

			if tc.isPreviousSpec {
				changeset := &cmpgn.Changeset{
					ExternalServiceType: "github",
					RepoID:              1,
					PreviousSpecID:      changesetSpec.ID,
				}
				if err := s.CreateChangeset(ctx, changeset); err != nil {
					t.Fatal(err)
				}
			}

			if err := s.DeleteExpiredChangesetSpecs(ctx); err != nil {
				t.Fatal(err)
			}

			_, err := s.GetChangesetSpec(ctx, GetChangesetSpecOpts{ID: changesetSpec.ID})
			if err != nil && err != ErrNoResults {
				t.Fatal(err)
			}

			if tc.wantDeleted && err == nil {
				t.Fatalf("tc=%s\n\t want changeset spec to be deleted, but was NOT", printTestCase(tc))
			}

			if !tc.wantDeleted && err == ErrNoResults {
				t.Fatalf("tc=%s\n\t want changeset spec NOT to be deleted, but got deleted", printTestCase(tc))
			}
		}
	})
}
