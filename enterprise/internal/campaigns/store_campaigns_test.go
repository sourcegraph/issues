package campaigns

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/keegancsmith/sqlf"
	"github.com/sourcegraph/sourcegraph/cmd/repo-updater/repos"
	"github.com/sourcegraph/sourcegraph/internal/campaigns"
	cmpgn "github.com/sourcegraph/sourcegraph/internal/campaigns"
	"github.com/sourcegraph/sourcegraph/internal/db/dbtest"
)

func testStoreCampaigns(t *testing.T, ctx context.Context, s *Store, _ repos.Store, clock clock) {
	campaigns := make([]*cmpgn.Campaign, 0, 3)

	t.Run("Create", func(t *testing.T) {
		for i := 0; i < cap(campaigns); i++ {
			c := &cmpgn.Campaign{
				Name:        fmt.Sprintf("test-campaign-%d", i),
				Description: "All the Javascripts are belong to us",

				InitialApplierID: int32(i) + 50,
				LastAppliedAt:    clock.now(),
				LastApplierID:    int32(i) + 99,

				ChangesetIDs:   []int64{int64(i) + 1},
				CampaignSpecID: 1742 + int64(i),
				ClosedAt:       clock.now(),
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

			err := s.CreateCampaign(ctx, have)
			if err != nil {
				t.Fatal(err)
			}

			if have.ID == 0 {
				t.Fatal("ID should not be zero")
			}

			want.ID = have.ID
			want.CreatedAt = clock.now()
			want.UpdatedAt = clock.now()

			if diff := cmp.Diff(have, want); diff != "" {
				t.Fatal(diff)
			}

			campaigns = append(campaigns, c)
		}
	})

	t.Run("Count", func(t *testing.T) {
		count, err := s.CountCampaigns(ctx, CountCampaignsOpts{})
		if err != nil {
			t.Fatal(err)
		}

		if have, want := count, len(campaigns); have != want {
			t.Fatalf("have count: %d, want: %d", have, want)
		}

		count, err = s.CountCampaigns(ctx, CountCampaignsOpts{ChangesetID: 1})
		if err != nil {
			t.Fatal(err)
		}

		if have, want := count, 1; have != want {
			t.Fatalf("have count: %d, want: %d", have, want)
		}

		t.Run("OnlyForAuthor set", func(t *testing.T) {
			for _, c := range campaigns {
				count, err = s.CountCampaigns(ctx, CountCampaignsOpts{InitialApplierID: c.InitialApplierID})
				if err != nil {
					t.Fatal(err)
				}
				if have, want := count, 1; have != want {
					t.Fatalf("Incorrect number of campaigns counted, want=%d have=%d", want, have)
				}
			}
		})

		t.Run("NamespaceUserID", func(t *testing.T) {
			wantCounts := map[int32]int{}
			for _, c := range campaigns {
				if c.NamespaceUserID == 0 {
					continue
				}
				wantCounts[c.NamespaceUserID] += 1
			}
			if len(wantCounts) == 0 {
				t.Fatalf("No campaigns with NamespaceUserID")
			}

			for userID, want := range wantCounts {
				have, err := s.CountCampaigns(ctx, CountCampaignsOpts{NamespaceUserID: userID})
				if err != nil {
					t.Fatal(err)
				}

				if have != want {
					t.Fatalf("campaigns count for NamespaceUserID=%d wrong. want=%d, have=%d", userID, want, have)
				}
			}
		})

		t.Run("NamespaceOrgID", func(t *testing.T) {
			wantCounts := map[int32]int{}
			for _, c := range campaigns {
				if c.NamespaceOrgID == 0 {
					continue
				}
				wantCounts[c.NamespaceOrgID] += 1
			}
			if len(wantCounts) == 0 {
				t.Fatalf("No campaigns with NamespaceOrgID")
			}

			for orgID, want := range wantCounts {
				have, err := s.CountCampaigns(ctx, CountCampaignsOpts{NamespaceOrgID: orgID})
				if err != nil {
					t.Fatal(err)
				}

				if have != want {
					t.Fatalf("campaigns count for NamespaceOrgID=%d wrong. want=%d, have=%d", orgID, want, have)
				}
			}
		})
	})

	t.Run("List", func(t *testing.T) {
		t.Run("By ChangesetID", func(t *testing.T) {
			for i := 1; i <= len(campaigns); i++ {
				opts := ListCampaignsOpts{ChangesetID: int64(i)}

				ts, next, err := s.ListCampaigns(ctx, opts)
				if err != nil {
					t.Fatal(err)
				}

				if have, want := next, int64(0); have != want {
					t.Fatalf("opts: %+v: have next %v, want %v", opts, have, want)
				}

				have, want := ts, campaigns[i-1:i]
				if len(have) != len(want) {
					t.Fatalf("listed %d campaigns, want: %d", len(have), len(want))
				}

				if diff := cmp.Diff(have, want); diff != "" {
					t.Fatalf("opts: %+v, diff: %s", opts, diff)
				}
			}
		})

		// The campaigns store returns the campaigns in reversed order.
		reversedCampaigns := make([]*cmpgn.Campaign, len(campaigns))
		for i, c := range campaigns {
			reversedCampaigns[len(campaigns)-i-1] = c
		}

		t.Run("With Limit", func(t *testing.T) {
			for i := 1; i <= len(reversedCampaigns); i++ {
				cs, next, err := s.ListCampaigns(ctx, ListCampaignsOpts{LimitOpts: LimitOpts{Limit: i}})
				if err != nil {
					t.Fatal(err)
				}

				{
					have, want := next, int64(0)
					if i < len(reversedCampaigns) {
						want = reversedCampaigns[i].ID
					}

					if have != want {
						t.Fatalf("limit: %v: have next %v, want %v", i, have, want)
					}
				}

				{
					have, want := cs, reversedCampaigns[:i]
					if len(have) != len(want) {
						t.Fatalf("listed %d campaigns, want: %d", len(have), len(want))
					}

					if diff := cmp.Diff(have, want); diff != "" {
						t.Fatal(diff)
					}
				}
			}
		})

		t.Run("With Cursor", func(t *testing.T) {
			var cursor int64
			for i := 1; i <= len(reversedCampaigns); i++ {
				opts := ListCampaignsOpts{Cursor: cursor, LimitOpts: LimitOpts{Limit: 1}}
				have, next, err := s.ListCampaigns(ctx, opts)
				if err != nil {
					t.Fatal(err)
				}

				want := reversedCampaigns[i-1 : i]
				if diff := cmp.Diff(have, want); diff != "" {
					t.Fatalf("opts: %+v, diff: %s", opts, diff)
				}

				cursor = next
			}
		})

		filterTests := []struct {
			name  string
			state cmpgn.CampaignState
			want  []*cmpgn.Campaign
		}{
			{
				name:  "Any",
				state: cmpgn.CampaignStateAny,
				want:  reversedCampaigns,
			},
			{
				name:  "Closed",
				state: cmpgn.CampaignStateClosed,
				want:  reversedCampaigns[:len(reversedCampaigns)-1],
			},
			{
				name:  "Open",
				state: cmpgn.CampaignStateOpen,
				want:  campaigns[0:1],
			},
		}

		for _, tc := range filterTests {
			t.Run("ListCampaigns State "+tc.name, func(t *testing.T) {
				have, _, err := s.ListCampaigns(ctx, ListCampaignsOpts{State: tc.state})
				if err != nil {
					t.Fatal(err)
				}
				if diff := cmp.Diff(have, tc.want); diff != "" {
					t.Fatal(diff)
				}
			})
		}

		t.Run("ListCampaigns OnlyForAuthor set", func(t *testing.T) {
			for _, c := range campaigns {
				have, next, err := s.ListCampaigns(ctx, ListCampaignsOpts{InitialApplierID: c.InitialApplierID})
				if err != nil {
					t.Fatal(err)
				}
				if next != 0 {
					t.Fatal("Next value was true, but false expected")
				}
				if have, want := len(have), 1; have != want {
					t.Fatalf("Incorrect number of campaigns returned, want=%d have=%d", want, have)
				}
				if diff := cmp.Diff(have[0], c); diff != "" {
					t.Fatal(diff)
				}
			}
		})

		t.Run("ListCampaigns by NamespaceUserID", func(t *testing.T) {
			for _, c := range campaigns {
				if c.NamespaceUserID == 0 {
					continue
				}
				opts := ListCampaignsOpts{NamespaceUserID: c.NamespaceUserID}
				have, _, err := s.ListCampaigns(ctx, opts)
				if err != nil {
					t.Fatal(err)
				}

				for _, haveCampaign := range have {
					if have, want := haveCampaign.NamespaceUserID, opts.NamespaceUserID; have != want {
						t.Fatalf("campaign has wrong NamespaceUserID. want=%d, have=%d", want, have)
					}
				}
			}
		})

		t.Run("ListCampaigns by NamespaceOrgID", func(t *testing.T) {
			for _, c := range campaigns {
				if c.NamespaceOrgID == 0 {
					continue
				}
				opts := ListCampaignsOpts{NamespaceOrgID: c.NamespaceOrgID}
				have, _, err := s.ListCampaigns(ctx, opts)
				if err != nil {
					t.Fatal(err)
				}

				for _, haveCampaign := range have {
					if have, want := haveCampaign.NamespaceOrgID, opts.NamespaceOrgID; have != want {
						t.Fatalf("campaign has wrong NamespaceOrgID. want=%d, have=%d", want, have)
					}
				}
			}
		})
	})

	t.Run("Update", func(t *testing.T) {
		for _, c := range campaigns {
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

			clock.add(1 * time.Second)

			want := c
			want.UpdatedAt = clock.now()

			have := c.Clone()
			if err := s.UpdateCampaign(ctx, have); err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(have, want); diff != "" {
				t.Fatal(diff)
			}

			// Test that duplicates are not introduced.
			have.ChangesetIDs = append(have.ChangesetIDs, have.ChangesetIDs...)
			if err := s.UpdateCampaign(ctx, have); err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(have, want); diff != "" {
				t.Fatal(diff)
			}

			// Test we can add to the set.
			have.ChangesetIDs = append(have.ChangesetIDs, 42)
			want.ChangesetIDs = append(want.ChangesetIDs, 42)

			if err := s.UpdateCampaign(ctx, have); err != nil {
				t.Fatal(err)
			}

			sort.Slice(have.ChangesetIDs, func(a, b int) bool {
				return have.ChangesetIDs[a] < have.ChangesetIDs[b]
			})

			if diff := cmp.Diff(have, want); diff != "" {
				t.Fatal(diff)
			}

			// Test we can remove from the set.
			have.ChangesetIDs = have.ChangesetIDs[:0]
			want.ChangesetIDs = want.ChangesetIDs[:0]

			if err := s.UpdateCampaign(ctx, have); err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(have, want); diff != "" {
				t.Fatal(diff)
			}
		}
	})

	t.Run("Get", func(t *testing.T) {
		t.Run("ByID", func(t *testing.T) {
			want := campaigns[0]
			opts := GetCampaignOpts{ID: want.ID}

			have, err := s.GetCampaign(ctx, opts)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(have, want); diff != "" {
				t.Fatal(diff)
			}
		})

		t.Run("ByCampaignSpecID", func(t *testing.T) {
			want := campaigns[0]
			opts := GetCampaignOpts{CampaignSpecID: want.CampaignSpecID}

			have, err := s.GetCampaign(ctx, opts)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(have, want); diff != "" {
				t.Fatal(diff)
			}
		})

		t.Run("ByName", func(t *testing.T) {
			want := campaigns[0]

			have, err := s.GetCampaign(ctx, GetCampaignOpts{Name: want.Name})
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(have, want); diff != "" {
				t.Fatal(diff)
			}
		})

		t.Run("ByNamespaceUserID", func(t *testing.T) {
			for _, c := range campaigns {
				if c.NamespaceUserID == 0 {
					continue
				}

				want := c
				opts := GetCampaignOpts{NamespaceUserID: c.NamespaceUserID}

				have, err := s.GetCampaign(ctx, opts)
				if err != nil {
					t.Fatal(err)
				}

				if diff := cmp.Diff(have, want); diff != "" {
					t.Fatal(diff)
				}
			}
		})

		t.Run("ByNamespaceOrgID", func(t *testing.T) {
			for _, c := range campaigns {
				if c.NamespaceOrgID == 0 {
					continue
				}

				want := c
				opts := GetCampaignOpts{NamespaceOrgID: c.NamespaceOrgID}

				have, err := s.GetCampaign(ctx, opts)
				if err != nil {
					t.Fatal(err)
				}

				if diff := cmp.Diff(have, want); diff != "" {
					t.Fatal(diff)
				}
			}
		})

		t.Run("NoResults", func(t *testing.T) {
			opts := GetCampaignOpts{ID: 0xdeadbeef}

			_, have := s.GetCampaign(ctx, opts)
			want := ErrNoResults

			if have != want {
				t.Fatalf("have err %v, want %v", have, want)
			}
		})
	})

	t.Run("Delete", func(t *testing.T) {
		for i := range campaigns {
			err := s.DeleteCampaign(ctx, campaigns[i].ID)
			if err != nil {
				t.Fatal(err)
			}

			count, err := s.CountCampaigns(ctx, CountCampaignsOpts{})
			if err != nil {
				t.Fatal(err)
			}

			if have, want := count, len(campaigns)-(i+1); have != want {
				t.Fatalf("have count: %d, want: %d", have, want)
			}
		}
	})
}

func TestUserDeleteCascades(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	db := dbtest.NewDB(t, *dsn)
	orgID := insertTestOrg(t, db)
	userID := insertTestUser(t, db)

	t.Run("user delete", storeTest(db, func(t *testing.T, ctx context.Context, store *Store, rs repos.Store, clock clock) {
		// Set up two campaigns and specs: one in the user's namespace (which
		// should be deleted when the user is hard deleted), and one that is
		// merely created by the user (which should remain).
		ownedSpec := &campaigns.CampaignSpec{
			NamespaceUserID: userID,
			UserID:          userID,
		}
		if err := store.CreateCampaignSpec(ctx, ownedSpec); err != nil {
			t.Fatal(err)
		}

		unownedSpec := &campaigns.CampaignSpec{
			NamespaceOrgID: orgID,
			UserID:         userID,
		}
		if err := store.CreateCampaignSpec(ctx, unownedSpec); err != nil {
			t.Fatal(err)
		}

		ownedCampaign := &campaigns.Campaign{
			Name:             "owned",
			NamespaceUserID:  userID,
			InitialApplierID: userID,
			LastApplierID:    userID,
			LastAppliedAt:    clock.now(),
			CampaignSpecID:   ownedSpec.ID,
		}
		if err := store.CreateCampaign(ctx, ownedCampaign); err != nil {
			t.Fatal(err)
		}

		unownedCampaign := &campaigns.Campaign{
			Name:             "unowned",
			NamespaceOrgID:   orgID,
			InitialApplierID: userID,
			LastApplierID:    userID,
			LastAppliedAt:    clock.now(),
			CampaignSpecID:   ownedSpec.ID,
		}
		if err := store.CreateCampaign(ctx, unownedCampaign); err != nil {
			t.Fatal(err)
		}

		// Now we'll try actually deleting the user.
		if err := store.Store.Exec(ctx, sqlf.Sprintf(
			"DELETE FROM users WHERE id = %s",
			userID,
		)); err != nil {
			t.Fatal(err)
		}

		// We should now have the unowned campaign still be valid, but the
		// owned campaign should have gone away.
		cs, _, err := store.ListCampaigns(ctx, ListCampaignsOpts{})
		if err != nil {
			t.Fatal(err)
		}
		if len(cs) != 1 {
			t.Errorf("unexpected number of campaigns: have %d; want %d", len(cs), 1)
		}
		if cs[0].ID != unownedCampaign.ID {
			t.Errorf("unexpected campaign: %+v", cs[0])
		}

		// Both campaign specs should still be in place, at least until we add
		// a foreign key constraint to campaign_specs.namespace_user_id.
		specs, _, err := store.ListCampaignSpecs(ctx, ListCampaignSpecsOpts{})
		if err != nil {
			t.Fatal(err)
		}
		if len(specs) != 2 {
			t.Errorf("unexpected number of campaign specs: have %d; want %d", len(specs), 2)
		}
	}))
}
