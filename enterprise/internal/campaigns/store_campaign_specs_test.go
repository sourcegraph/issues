package campaigns

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/sourcegraph/sourcegraph/cmd/repo-updater/repos"
	"github.com/sourcegraph/sourcegraph/internal/campaigns"
	cmpgn "github.com/sourcegraph/sourcegraph/internal/campaigns"
)

func testStoreCampaignSpecs(t *testing.T, ctx context.Context, s *Store, _ repos.Store, clock clock) {
	campaignSpecs := make([]*cmpgn.CampaignSpec, 0, 3)

	t.Run("Create", func(t *testing.T) {
		for i := 0; i < cap(campaignSpecs); i++ {
			c := &cmpgn.CampaignSpec{
				RawSpec: `{"name": "Foobar", "description": "My description"}`,
				Spec: cmpgn.CampaignSpecFields{
					Name:        "Foobar",
					Description: "My description",
					ChangesetTemplate: cmpgn.ChangesetTemplate{
						Title:  "Hello there",
						Body:   "This is the body",
						Branch: "my-branch",
						Commit: campaigns.CommitTemplate{
							Message: "commit message",
						},
						Published: false,
					},
				},
				UserID: int32(i + 1234),
			}

			if i%2 == 0 {
				c.NamespaceOrgID = 23
			} else {
				c.NamespaceUserID = c.UserID
			}

			want := c.Clone()
			have := c

			err := s.CreateCampaignSpec(ctx, have)
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

			campaignSpecs = append(campaignSpecs, c)
		}
	})

	if len(campaignSpecs) != cap(campaignSpecs) {
		t.Fatalf("campaignSpecs is empty. creation failed")
	}

	t.Run("Count", func(t *testing.T) {
		count, err := s.CountCampaignSpecs(ctx)
		if err != nil {
			t.Fatal(err)
		}

		if have, want := count, int64(len(campaignSpecs)); have != want {
			t.Fatalf("have count: %d, want: %d", have, want)
		}
	})

	t.Run("List", func(t *testing.T) {
		t.Run("NoLimit", func(t *testing.T) {
			opts := []ListCampaignSpecsOpts{
				{},          // Empty limit should return default limit
				{Limit: -1}, // -1 should return all entries
			}

			for _, o := range opts {
				ts, next, err := s.ListCampaignSpecs(ctx, o)
				if err != nil {
					t.Fatal(err)
				}

				if have, want := next, int64(0); have != want {
					t.Fatalf("opts: %+v: have next %v, want %v", opts, have, want)
				}

				have, want := ts, campaignSpecs
				if len(have) != len(want) {
					t.Fatalf("listed %d campaignSpecs, want: %d", len(have), len(want))
				}

				if diff := cmp.Diff(have, want); diff != "" {
					t.Fatalf("opts: %+v, diff: %s", opts, diff)
				}
			}

		})

		t.Run("WithLimit", func(t *testing.T) {
			for i := 1; i <= len(campaignSpecs); i++ {
				cs, next, err := s.ListCampaignSpecs(ctx, ListCampaignSpecsOpts{Limit: i})
				if err != nil {
					t.Fatal(err)
				}

				{
					have, want := next, int64(0)
					if i < len(campaignSpecs) {
						want = campaignSpecs[i].ID
					}

					if have != want {
						t.Fatalf("limit: %v: have next %v, want %v", i, have, want)
					}
				}

				{
					have, want := cs, campaignSpecs[:i]
					if len(have) != len(want) {
						t.Fatalf("listed %d campaignSpecs, want: %d", len(have), len(want))
					}

					if diff := cmp.Diff(have, want); diff != "" {
						t.Fatal(diff)
					}
				}
			}

		})

		t.Run("WithLimitAndCursor", func(t *testing.T) {
			var cursor int64
			for i := 1; i <= len(campaignSpecs); i++ {
				opts := ListCampaignSpecsOpts{Cursor: cursor, Limit: 1}
				have, next, err := s.ListCampaignSpecs(ctx, opts)
				if err != nil {
					t.Fatal(err)
				}

				want := campaignSpecs[i-1 : i]
				if diff := cmp.Diff(have, want); diff != "" {
					t.Fatalf("opts: %+v, diff: %s", opts, diff)
				}

				cursor = next
			}
		})
	})

	t.Run("Update", func(t *testing.T) {
		for _, c := range campaignSpecs {
			c.UserID += 1234

			clock.add(1 * time.Second)

			want := c
			want.UpdatedAt = clock.now()

			have := c.Clone()
			if err := s.UpdateCampaignSpec(ctx, have); err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(have, want); diff != "" {
				t.Fatal(diff)
			}
		}
	})

	t.Run("Get", func(t *testing.T) {
		want := campaignSpecs[1]
		tests := map[string]GetCampaignSpecOpts{
			"ByID":          {ID: want.ID},
			"ByRandID":      {RandID: want.RandID},
			"ByIDAndRandID": {ID: want.ID, RandID: want.RandID},
		}

		for name, opts := range tests {
			t.Run(name, func(t *testing.T) {
				have, err := s.GetCampaignSpec(ctx, opts)
				if err != nil {
					t.Fatal(err)
				}

				if diff := cmp.Diff(have, want); diff != "" {
					t.Fatal(diff)
				}
			})
		}

		t.Run("NoResults", func(t *testing.T) {
			opts := GetCampaignSpecOpts{ID: 0xdeadbeef}

			_, have := s.GetCampaignSpec(ctx, opts)
			want := ErrNoResults

			if have != want {
				t.Fatalf("have err %v, want %v", have, want)
			}
		})
	})

	t.Run("Delete", func(t *testing.T) {
		for i := range campaignSpecs {
			err := s.DeleteCampaignSpec(ctx, campaignSpecs[i].ID)
			if err != nil {
				t.Fatal(err)
			}

			count, err := s.CountCampaignSpecs(ctx)
			if err != nil {
				t.Fatal(err)
			}

			if have, want := count, int64(len(campaignSpecs)-(i+1)); have != want {
				t.Fatalf("have count: %d, want: %d", have, want)
			}
		}
	})
}
