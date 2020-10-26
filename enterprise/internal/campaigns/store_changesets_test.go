package campaigns

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/keegancsmith/sqlf"
	"github.com/sourcegraph/sourcegraph/cmd/repo-updater/repos"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/campaigns"
	"github.com/sourcegraph/sourcegraph/internal/db/basestore"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/bitbucketserver"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/github"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/gitlab"

	cmpgn "github.com/sourcegraph/sourcegraph/internal/campaigns"
)

func testStoreChangesets(t *testing.T, ctx context.Context, s *Store, reposStore repos.Store, clock clock) {
	githubActor := github.Actor{
		AvatarURL: "https://avatars2.githubusercontent.com/u/1185253",
		Login:     "mrnugget",
		URL:       "https://github.com/mrnugget",
	}
	githubPR := &github.PullRequest{
		ID:           "FOOBARID",
		Title:        "Fix a bunch of bugs",
		Body:         "This fixes a bunch of bugs",
		URL:          "https://github.com/sourcegraph/sourcegraph/pull/12345",
		Number:       12345,
		Author:       githubActor,
		Participants: []github.Actor{githubActor},
		CreatedAt:    clock.now(),
		UpdatedAt:    clock.now(),
		HeadRefName:  "campaigns/test",
	}

	repo := testRepo(t, reposStore, extsvc.KindGitHub)
	otherRepo := testRepo(t, reposStore, extsvc.KindGitHub)

	if err := reposStore.InsertRepos(ctx, repo, otherRepo); err != nil {
		t.Fatal(err)
	}
	deletedRepo := otherRepo.With(repos.Opt.RepoDeletedAt(clock.now()))
	if err := reposStore.DeleteRepos(ctx, deletedRepo.ID); err != nil {
		t.Fatal(err)
	}

	changesets := make(cmpgn.Changesets, 0, 3)

	deletedRepoChangeset := &cmpgn.Changeset{
		RepoID:              deletedRepo.ID,
		ExternalID:          fmt.Sprintf("foobar-%d", cap(changesets)),
		ExternalServiceType: extsvc.TypeGitHub,
	}

	var (
		added   int32 = 77
		deleted int32 = 88
		changed int32 = 99
	)

	t.Run("Create", func(t *testing.T) {
		var i int
		for i = 0; i < cap(changesets); i++ {
			failureMessage := fmt.Sprintf("failure-%d", i)
			th := &cmpgn.Changeset{
				RepoID:              repo.ID,
				CreatedAt:           clock.now(),
				UpdatedAt:           clock.now(),
				Metadata:            githubPR,
				CampaignIDs:         []int64{int64(i) + 1},
				ExternalID:          fmt.Sprintf("foobar-%d", i),
				ExternalServiceType: extsvc.TypeGitHub,
				ExternalBranch:      fmt.Sprintf("campaigns/test/%d", i),
				ExternalUpdatedAt:   clock.now(),
				ExternalState:       cmpgn.ChangesetExternalStateOpen,
				ExternalReviewState: cmpgn.ChangesetReviewStateApproved,
				ExternalCheckState:  cmpgn.ChangesetCheckStatePassed,

				CurrentSpecID:     int64(i) + 1,
				PreviousSpecID:    int64(i) + 1,
				OwnedByCampaignID: int64(i) + 1,
				PublicationState:  cmpgn.ChangesetPublicationStatePublished,

				ReconcilerState: cmpgn.ReconcilerStateCompleted,
				FailureMessage:  &failureMessage,
				NumResets:       18,
				NumFailures:     25,

				Unsynced: i != 0,
				Closing:  true,
			}

			// Only set these fields on a subset to make sure that
			// we handle nil pointers correctly
			if i != cap(changesets)-1 {
				th.DiffStatAdded = &added
				th.DiffStatChanged = &changed
				th.DiffStatDeleted = &deleted

				th.StartedAt = clock.now()
				th.FinishedAt = clock.now()
				th.ProcessAfter = clock.now()
			}

			if err := s.CreateChangeset(ctx, th); err != nil {
				t.Fatal(err)
			}

			changesets = append(changesets, th)
		}

		if err := s.CreateChangeset(ctx, deletedRepoChangeset); err != nil {
			t.Fatal(err)
		}

		for _, have := range changesets {
			if have.ID == 0 {
				t.Fatal("id should not be zero")
			}

			if have.IsDeleted() {
				t.Fatal("changeset is deleted")
			}

			if !have.ReconcilerState.Valid() {
				t.Fatalf("reconciler state is invalid: %s", have.ReconcilerState)
			}

			want := have.Clone()

			want.ID = have.ID
			want.CreatedAt = clock.now()
			want.UpdatedAt = clock.now()

			if diff := cmp.Diff(have, want); diff != "" {
				t.Fatal(diff)
			}
		}
	})

	t.Run("ReconcilerState database representation", func(t *testing.T) {
		// campaigns.ReconcilerStates are defined as "enum" string constants.
		// The string values are uppercase, because that way they can easily be
		// serialized/deserialized in the GraphQL resolvers, since GraphQL
		// expects the `ChangesetReconcilerState` values to be uppercase.
		//
		// But workerutil.Worker expects those values to be lowercase.
		//
		// So, what we do is to lowercase the Changeset.ReconcilerState value
		// before it enters the database and uppercase it when it leaves the
		// DB.
		//
		// If workerutil.Worker supports custom mappings for the state-machine
		// states, we can remove this.

		// This test ensures that the database representation is lowercase.

		queryRawReconcilerState := func(ch *cmpgn.Changeset) (string, error) {
			q := sqlf.Sprintf("SELECT reconciler_state FROM changesets WHERE id = %s", ch.ID)
			rawState, ok, err := basestore.ScanFirstString(s.Query(ctx, q))
			if err != nil || !ok {
				return rawState, err
			}
			return rawState, nil
		}

		for _, ch := range changesets {
			have, err := queryRawReconcilerState(ch)
			if err != nil {
				t.Fatal(err)
			}

			want := strings.ToLower(string(ch.ReconcilerState))
			if have != want {
				t.Fatalf("wrong database representation. want=%q, have=%q", want, have)
			}
		}
	})

	t.Run("GetChangesetExternalIDs", func(t *testing.T) {
		refs := make([]string, len(changesets))
		for i, c := range changesets {
			refs[i] = c.ExternalBranch
		}
		have, err := s.GetChangesetExternalIDs(ctx, repo.ExternalRepo, refs)
		if err != nil {
			t.Fatal(err)
		}
		want := []string{"foobar-0", "foobar-1", "foobar-2"}
		if diff := cmp.Diff(want, have); diff != "" {
			t.Fatal(diff)
		}
	})

	t.Run("GetChangesetExternalIDs no branch", func(t *testing.T) {
		spec := api.ExternalRepoSpec{
			ID:          "external-id",
			ServiceType: extsvc.TypeGitHub,
			ServiceID:   "https://github.com/",
		}
		have, err := s.GetChangesetExternalIDs(ctx, spec, []string{"foo"})
		if err != nil {
			t.Fatal(err)
		}
		var want []string
		if diff := cmp.Diff(want, have); diff != "" {
			t.Fatal(diff)
		}
	})

	t.Run("GetChangesetExternalIDs invalid external-id", func(t *testing.T) {
		spec := api.ExternalRepoSpec{
			ID:          "invalid",
			ServiceType: extsvc.TypeGitHub,
			ServiceID:   "https://github.com/",
		}
		have, err := s.GetChangesetExternalIDs(ctx, spec, []string{"campaigns/test"})
		if err != nil {
			t.Fatal(err)
		}
		var want []string
		if diff := cmp.Diff(want, have); diff != "" {
			t.Fatal(diff)
		}
	})

	t.Run("GetChangesetExternalIDs invalid external service id", func(t *testing.T) {
		spec := api.ExternalRepoSpec{
			ID:          "external-id",
			ServiceType: extsvc.TypeGitHub,
			ServiceID:   "invalid",
		}
		have, err := s.GetChangesetExternalIDs(ctx, spec, []string{"campaigns/test"})
		if err != nil {
			t.Fatal(err)
		}
		var want []string
		if diff := cmp.Diff(want, have); diff != "" {
			t.Fatal(diff)
		}
	})

	t.Run("Count", func(t *testing.T) {
		t.Run("No options", func(t *testing.T) {
			count, err := s.CountChangesets(ctx, CountChangesetsOpts{})
			if err != nil {
				t.Fatal(err)
			}

			if have, want := count, len(changesets); have != want {
				t.Fatalf("have count: %d, want: %d", have, want)
			}

		})

		t.Run("CampaignID", func(t *testing.T) {
			count, err := s.CountChangesets(ctx, CountChangesetsOpts{CampaignID: 1})
			if err != nil {
				t.Fatal(err)
			}

			if have, want := count, 1; have != want {
				t.Fatalf("have count: %d, want: %d", have, want)
			}
		})

		t.Run("ReconcilerState", func(t *testing.T) {
			completed := campaigns.ReconcilerStateCompleted
			countCompleted, err := s.CountChangesets(ctx, CountChangesetsOpts{ReconcilerStates: []campaigns.ReconcilerState{completed}})
			if err != nil {
				t.Fatal(err)
			}

			if have, want := countCompleted, len(changesets); have != want {
				t.Fatalf("have countCompleted: %d, want: %d", have, want)
			}

			processing := campaigns.ReconcilerStateProcessing
			countProcessing, err := s.CountChangesets(ctx, CountChangesetsOpts{ReconcilerStates: []campaigns.ReconcilerState{processing}})
			if err != nil {
				t.Fatal(err)
			}

			if have, want := countProcessing, 0; have != want {
				t.Fatalf("have countProcessing: %d, want: %d", have, want)
			}
		})

		t.Run("OwnedByCampaignID", func(t *testing.T) {
			count, err := s.CountChangesets(ctx, CountChangesetsOpts{OwnedByCampaignID: int64(1)})
			if err != nil {
				t.Fatal(err)
			}

			if have, want := count, 1; have != want {
				t.Fatalf("have count: %d, want: %d", have, want)
			}
		})
	})

	t.Run("List", func(t *testing.T) {
		for i := 1; i <= len(changesets); i++ {
			opts := ListChangesetsOpts{CampaignID: int64(i)}

			ts, next, err := s.ListChangesets(ctx, opts)
			if err != nil {
				t.Fatal(err)
			}

			if have, want := next, int64(0); have != want {
				t.Fatalf("opts: %+v: have next %v, want %v", opts, have, want)
			}

			have, want := ts, changesets[i-1:i]
			if len(have) != len(want) {
				t.Fatalf("listed %d changesets, want: %d", len(have), len(want))
			}

			if diff := cmp.Diff(have, want); diff != "" {
				t.Fatalf("opts: %+v, diff: %s", opts, diff)
			}
		}

		for i := 1; i <= len(changesets); i++ {
			ts, next, err := s.ListChangesets(ctx, ListChangesetsOpts{LimitOpts: LimitOpts{Limit: i}})
			if err != nil {
				t.Fatal(err)
			}

			{
				have, want := next, int64(0)
				if i < len(changesets) {
					want = changesets[i].ID
				}

				if have != want {
					t.Fatalf("limit: %v: have next %v, want %v", i, have, want)
				}
			}

			{
				have, want := ts, changesets[:i]
				if len(have) != len(want) {
					t.Fatalf("listed %d changesets, want: %d", len(have), len(want))
				}

				if diff := cmp.Diff(have, want); diff != "" {
					t.Fatal(diff)
				}
			}
		}

		{
			have, _, err := s.ListChangesets(ctx, ListChangesetsOpts{IDs: changesets.IDs()})
			if err != nil {
				t.Fatal(err)
			}

			want := changesets
			if diff := cmp.Diff(have, want); diff != "" {
				t.Fatal(diff)
			}
		}

		{
			var cursor int64
			for i := 1; i <= len(changesets); i++ {
				opts := ListChangesetsOpts{Cursor: cursor, LimitOpts: LimitOpts{Limit: 1}}
				have, next, err := s.ListChangesets(ctx, opts)
				if err != nil {
					t.Fatal(err)
				}

				want := changesets[i-1 : i]
				if diff := cmp.Diff(have, want); diff != "" {
					t.Fatalf("opts: %+v, diff: %s", opts, diff)
				}

				cursor = next
			}
		}

		{
			have, _, err := s.ListChangesets(ctx, ListChangesetsOpts{WithoutDeleted: true})
			if err != nil {
				t.Fatal(err)
			}

			if len(have) != len(changesets) {
				t.Fatalf("have 0 changesets. want %d", len(changesets))
			}

			for _, c := range changesets {
				c.SetDeleted()
				c.UpdatedAt = clock.now()

				if err := s.UpdateChangeset(ctx, c); err != nil {
					t.Fatal(err)
				}
			}

			have, _, err = s.ListChangesets(ctx, ListChangesetsOpts{WithoutDeleted: true})
			if err != nil {
				t.Fatal(err)
			}

			if len(have) != 0 {
				t.Fatalf("have %d changesets. want 0", len(changesets))
			}
		}

		{
			have, _, err := s.ListChangesets(ctx, ListChangesetsOpts{OnlyWithoutDiffStats: true})
			if err != nil {
				t.Fatal(err)
			}

			want := 1
			if len(have) != want {
				t.Fatalf("have %d changesets; want %d", len(have), want)
			}

			if have[0].ID != changesets[cap(changesets)-1].ID {
				t.Fatalf("unexpected changeset: have %+v; want %+v", have[0], changesets[cap(changesets)-1])
			}
		}

		// No Limit should return all Changesets
		{
			have, _, err := s.ListChangesets(ctx, ListChangesetsOpts{})
			if err != nil {
				t.Fatal(err)
			}

			if len(have) != 3 {
				t.Fatalf("have %d changesets. want 3", len(have))
			}
		}

		statePublished := cmpgn.ChangesetPublicationStatePublished
		stateUnpublished := cmpgn.ChangesetPublicationStateUnpublished
		stateQueued := cmpgn.ReconcilerStateQueued
		stateCompleted := cmpgn.ReconcilerStateCompleted
		stateOpen := cmpgn.ChangesetExternalStateOpen
		stateClosed := cmpgn.ChangesetExternalStateClosed
		stateApproved := cmpgn.ChangesetReviewStateApproved
		stateChangesRequested := cmpgn.ChangesetReviewStateChangesRequested
		statePassed := cmpgn.ChangesetCheckStatePassed
		stateFailed := cmpgn.ChangesetCheckStateFailed

		filterCases := []struct {
			opts      ListChangesetsOpts
			wantCount int
		}{
			{
				opts: ListChangesetsOpts{
					PublicationState: &statePublished,
				},
				wantCount: 3,
			},
			{
				opts: ListChangesetsOpts{
					PublicationState: &stateUnpublished,
				},
				wantCount: 0,
			},
			{
				opts: ListChangesetsOpts{
					ReconcilerStates: []campaigns.ReconcilerState{stateQueued},
				},
				wantCount: 0,
			},
			{
				opts: ListChangesetsOpts{
					ReconcilerStates: []campaigns.ReconcilerState{stateCompleted},
				},
				wantCount: 3,
			},
			{
				opts: ListChangesetsOpts{
					ExternalState: &stateOpen,
				},
				wantCount: 3,
			},
			{
				opts: ListChangesetsOpts{
					ExternalState: &stateClosed,
				},
				wantCount: 0,
			},
			{
				opts: ListChangesetsOpts{
					ExternalReviewState: &stateApproved,
				},
				wantCount: 3,
			},
			{
				opts: ListChangesetsOpts{
					ExternalReviewState: &stateChangesRequested,
				},
				wantCount: 0,
			},
			{
				opts: ListChangesetsOpts{
					ExternalCheckState: &statePassed,
				},
				wantCount: 3,
			},
			{
				opts: ListChangesetsOpts{
					ExternalCheckState: &stateFailed,
				},
				wantCount: 0,
			},
			{
				opts: ListChangesetsOpts{
					ExternalState:      &stateOpen,
					ExternalCheckState: &stateFailed,
				},
				wantCount: 0,
			},
			{
				opts: ListChangesetsOpts{
					ExternalState:       &stateOpen,
					ExternalReviewState: &stateChangesRequested,
				},
				wantCount: 0,
			},
			{
				opts: ListChangesetsOpts{
					OwnedByCampaignID: int64(1),
				},
				wantCount: 1,
			},
			{
				opts: ListChangesetsOpts{
					OnlySynced: true,
				},
				wantCount: 1,
			},
		}

		for i, tc := range filterCases {
			t.Run(strconv.Itoa(i), func(t *testing.T) {
				have, _, err := s.ListChangesets(ctx, tc.opts)
				if err != nil {
					t.Fatal(err)
				}
				if len(have) != tc.wantCount {
					t.Fatalf("opts: %+v. have %d changesets. want %d", tc.opts, len(have), tc.wantCount)
				}
			})
		}
	})

	t.Run("Null changeset external state", func(t *testing.T) {
		cs := &cmpgn.Changeset{
			RepoID:              repo.ID,
			Metadata:            githubPR,
			CampaignIDs:         []int64{1},
			ExternalID:          fmt.Sprintf("foobar-%d", 42),
			ExternalServiceType: extsvc.TypeGitHub,
			ExternalBranch:      "campaigns/test",
			ExternalUpdatedAt:   clock.now(),
			ExternalState:       "",
			ExternalReviewState: "",
			ExternalCheckState:  "",
		}

		err := s.CreateChangeset(ctx, cs)
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			err := s.DeleteChangeset(ctx, cs.ID)
			if err != nil {
				t.Fatal(err)
			}
		}()

		fromDB, err := s.GetChangeset(ctx, GetChangesetOpts{
			ID: cs.ID,
		})
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(cs.ExternalState, fromDB.ExternalState); diff != "" {
			t.Error(diff)
		}
		if diff := cmp.Diff(cs.ExternalReviewState, fromDB.ExternalReviewState); diff != "" {
			t.Error(diff)
		}
		if diff := cmp.Diff(cs.ExternalCheckState, fromDB.ExternalCheckState); diff != "" {
			t.Error(diff)
		}
	})

	t.Run("Get", func(t *testing.T) {
		t.Run("ByID", func(t *testing.T) {
			want := changesets[0]
			opts := GetChangesetOpts{ID: want.ID}

			have, err := s.GetChangeset(ctx, opts)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(have, want); diff != "" {
				t.Fatal(diff)
			}
		})

		t.Run("ByExternalID", func(t *testing.T) {
			want := changesets[0]
			opts := GetChangesetOpts{
				ExternalID:          want.ExternalID,
				ExternalServiceType: want.ExternalServiceType,
			}

			have, err := s.GetChangeset(ctx, opts)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(have, want); diff != "" {
				t.Fatal(diff)
			}
		})

		t.Run("ByRepoID", func(t *testing.T) {
			want := changesets[0]
			opts := GetChangesetOpts{
				RepoID: want.RepoID,
			}

			have, err := s.GetChangeset(ctx, opts)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(have, want); diff != "" {
				t.Fatal(diff)
			}
		})

		t.Run("NoResults", func(t *testing.T) {
			opts := GetChangesetOpts{ID: 0xdeadbeef}

			_, have := s.GetChangeset(ctx, opts)
			want := ErrNoResults

			if have != want {
				t.Fatalf("have err %v, want %v", have, want)
			}
		})

		t.Run("RepoDeleted", func(t *testing.T) {
			opts := GetChangesetOpts{ID: deletedRepoChangeset.ID}

			_, have := s.GetChangeset(ctx, opts)
			want := ErrNoResults

			if have != want {
				t.Fatalf("have err %v, want %v", have, want)
			}
		})

		t.Run("ExternalBranch", func(t *testing.T) {
			for _, c := range changesets {
				opts := GetChangesetOpts{ExternalBranch: c.ExternalBranch}

				have, err := s.GetChangeset(ctx, opts)
				if err != nil {
					t.Fatal(err)
				}
				want := c

				if diff := cmp.Diff(have, want); diff != "" {
					t.Fatal(diff)
				}
			}
		})
	})

	t.Run("Update", func(t *testing.T) {
		want := make([]*cmpgn.Changeset, 0, len(changesets))
		have := make([]*cmpgn.Changeset, 0, len(changesets))

		clock.add(1 * time.Second)
		for _, c := range changesets {
			c.Metadata = &bitbucketserver.PullRequest{ID: 1234}
			c.ExternalServiceType = extsvc.TypeBitbucketServer

			c.CurrentSpecID = c.CurrentSpecID + 1
			c.PreviousSpecID = c.PreviousSpecID + 1
			c.OwnedByCampaignID = c.OwnedByCampaignID + 1

			c.PublicationState = cmpgn.ChangesetPublicationStatePublished
			c.ReconcilerState = cmpgn.ReconcilerStateErrored
			c.FailureMessage = nil
			c.StartedAt = clock.now()
			c.FinishedAt = clock.now()
			c.ProcessAfter = clock.now()
			c.NumResets = 987
			c.NumFailures = 789

			clone := c.Clone()
			have = append(have, clone)

			c.UpdatedAt = clock.now()
			want = append(want, c)

			if err := s.UpdateChangeset(ctx, clone); err != nil {
				t.Fatal(err)
			}
		}

		if diff := cmp.Diff(have, want); diff != "" {
			t.Fatal(diff)
		}

		for i := range have {
			// Test that duplicates are not introduced.
			have[i].CampaignIDs = append(have[i].CampaignIDs, have[i].CampaignIDs...)

			if err := s.UpdateChangeset(ctx, have[i]); err != nil {
				t.Fatal(err)
			}

		}

		if diff := cmp.Diff(have, want); diff != "" {
			t.Fatal(diff)
		}

		for i := range have {
			// Test we can add to the set.
			have[i].CampaignIDs = append(have[i].CampaignIDs, 42)
			want[i].CampaignIDs = append(want[i].CampaignIDs, 42)

			if err := s.UpdateChangeset(ctx, have[i]); err != nil {
				t.Fatal(err)
			}

		}

		for i := range have {
			sort.Slice(have[i].CampaignIDs, func(a, b int) bool {
				return have[i].CampaignIDs[a] < have[i].CampaignIDs[b]
			})

			if diff := cmp.Diff(have[i], want[i]); diff != "" {
				t.Fatal(diff)
			}
		}

		for i := range have {
			// Test we can remove from the set.
			have[i].CampaignIDs = have[i].CampaignIDs[:0]
			want[i].CampaignIDs = want[i].CampaignIDs[:0]

			if err := s.UpdateChangeset(ctx, have[i]); err != nil {
				t.Fatal(err)
			}
		}

		if diff := cmp.Diff(have, want); diff != "" {
			t.Fatal(diff)
		}

		clock.add(1 * time.Second)
		want = want[0:0]
		have = have[0:0]
		for _, c := range changesets {
			c.Metadata = &gitlab.MergeRequest{ID: 1234, IID: 123}
			c.ExternalServiceType = extsvc.TypeGitLab

			clone := c.Clone()
			have = append(have, clone)

			c.UpdatedAt = clock.now()
			want = append(want, c)

			if err := s.UpdateChangeset(ctx, clone); err != nil {
				t.Fatal(err)
			}

		}

		if diff := cmp.Diff(have, want); diff != "" {
			t.Fatal(diff)
		}
	})

	t.Run("CancelQueuedCampaignChangesets", func(t *testing.T) {
		var campaignID int64 = 99999

		c1 := createChangeset(t, ctx, s, testChangesetOpts{
			repo:            repo.ID,
			campaign:        campaignID,
			ownedByCampaign: campaignID,
			reconcilerState: cmpgn.ReconcilerStateQueued,
		})

		c2 := createChangeset(t, ctx, s, testChangesetOpts{
			repo:            repo.ID,
			campaign:        campaignID,
			ownedByCampaign: campaignID,
			reconcilerState: cmpgn.ReconcilerStateErrored,
			numFailures:     reconcilerMaxNumRetries - 1,
		})

		c3 := createChangeset(t, ctx, s, testChangesetOpts{
			repo:            repo.ID,
			campaign:        campaignID,
			ownedByCampaign: campaignID,
			reconcilerState: cmpgn.ReconcilerStateCompleted,
		})

		c4 := createChangeset(t, ctx, s, testChangesetOpts{
			repo:            repo.ID,
			campaign:        campaignID,
			ownedByCampaign: 0,
			unsynced:        true,
			reconcilerState: cmpgn.ReconcilerStateQueued,
		})

		c5 := createChangeset(t, ctx, s, testChangesetOpts{
			repo:            repo.ID,
			campaign:        campaignID,
			ownedByCampaign: campaignID,
			reconcilerState: cmpgn.ReconcilerStateProcessing,
		})

		if err := s.CancelQueuedCampaignChangesets(ctx, campaignID); err != nil {
			t.Fatal(err)
		}

		reloadAndAssertChangeset(t, ctx, s, c1, changesetAssertions{
			repo:            repo.ID,
			reconcilerState: cmpgn.ReconcilerStateErrored,
			ownedByCampaign: campaignID,
			failureMessage:  &canceledChangesetFailureMessage,
			numFailures:     reconcilerMaxNumRetries,
		})

		reloadAndAssertChangeset(t, ctx, s, c2, changesetAssertions{
			repo:            repo.ID,
			reconcilerState: cmpgn.ReconcilerStateErrored,
			ownedByCampaign: campaignID,
			failureMessage:  &canceledChangesetFailureMessage,
			numFailures:     reconcilerMaxNumRetries,
		})

		reloadAndAssertChangeset(t, ctx, s, c3, changesetAssertions{
			repo:            repo.ID,
			reconcilerState: cmpgn.ReconcilerStateCompleted,
			ownedByCampaign: campaignID,
		})

		reloadAndAssertChangeset(t, ctx, s, c4, changesetAssertions{
			repo:            repo.ID,
			reconcilerState: cmpgn.ReconcilerStateQueued,
			unsynced:        true,
		})

		reloadAndAssertChangeset(t, ctx, s, c5, changesetAssertions{
			repo:            repo.ID,
			reconcilerState: cmpgn.ReconcilerStateErrored,
			failureMessage:  &canceledChangesetFailureMessage,
			ownedByCampaign: campaignID,
			numFailures:     reconcilerMaxNumRetries,
		})
	})

	t.Run("EnqueueChangesetsToClose", func(t *testing.T) {
		var campaignID int64 = 99999

		wantEnqueued := changesetAssertions{
			repo:            repo.ID,
			ownedByCampaign: campaignID,
			reconcilerState: campaigns.ReconcilerStateQueued,
			numFailures:     0,
			failureMessage:  nil,
			closing:         true,
		}

		tests := []struct {
			have testChangesetOpts
			want changesetAssertions
		}{
			{
				have: testChangesetOpts{reconcilerState: cmpgn.ReconcilerStateQueued},
				want: wantEnqueued,
			},
			{
				have: testChangesetOpts{reconcilerState: cmpgn.ReconcilerStateProcessing},
				want: wantEnqueued,
			},
			{
				have: testChangesetOpts{
					reconcilerState: cmpgn.ReconcilerStateErrored,
					failureMessage:  "failed",
					numFailures:     reconcilerMaxNumRetries - 1,
				},
				want: wantEnqueued,
			},
			{
				have: testChangesetOpts{
					externalState:   campaigns.ChangesetExternalStateOpen,
					reconcilerState: cmpgn.ReconcilerStateCompleted,
				},
				want: changesetAssertions{
					reconcilerState: campaigns.ReconcilerStateQueued,
					closing:         true,
					externalState:   campaigns.ChangesetExternalStateOpen,
				},
			},
			{
				have: testChangesetOpts{
					externalState:   campaigns.ChangesetExternalStateClosed,
					reconcilerState: cmpgn.ReconcilerStateCompleted,
				},
				want: changesetAssertions{
					reconcilerState: campaigns.ReconcilerStateCompleted,
					externalState:   campaigns.ChangesetExternalStateClosed,
				},
			},
		}

		changesets := make(map[*campaigns.Changeset]changesetAssertions)
		for _, tc := range tests {
			opts := tc.have
			opts.repo = repo.ID
			opts.campaign = campaignID
			opts.ownedByCampaign = campaignID

			c := createChangeset(t, ctx, s, opts)
			changesets[c] = tc.want
		}

		if err := s.EnqueueChangesetsToClose(ctx, campaignID); err != nil {
			t.Fatal(err)
		}

		for changeset, want := range changesets {
			want.repo = repo.ID
			want.ownedByCampaign = campaignID
			reloadAndAssertChangeset(t, ctx, s, changeset, want)
		}
	})

	t.Run("ListChangesetsAttachedOrOwnedByCampaign", func(t *testing.T) {
		var campaignID int64 = 191918

		baseOpts := testChangesetOpts{repo: repo.ID}

		opts1 := baseOpts
		opts1.campaign = campaignID
		opts1.ownedByCampaign = campaignID
		c1 := createChangeset(t, ctx, s, opts1)

		opts2 := baseOpts
		opts2.campaign = campaignID
		opts2.ownedByCampaign = 0
		c2 := createChangeset(t, ctx, s, opts2)

		opts3 := baseOpts
		opts3.campaign = campaignID + 999
		opts3.ownedByCampaign = campaignID + 999
		createChangeset(t, ctx, s, opts3)

		opts4 := baseOpts
		opts4.repo = deletedRepo.ID
		opts4.campaign = campaignID
		opts4.ownedByCampaign = 0
		createChangeset(t, ctx, s, opts4)

		cs, err := s.ListChangesetsAttachedOrOwnedByCampaign(ctx, campaignID)
		if err != nil {
			t.Fatal(err)
		}

		wantIDs := []int64{c1.ID, c2.ID}
		haveIDs := cs.IDs()
		if diff := cmp.Diff(wantIDs, haveIDs); diff != "" {
			t.Fatalf("wrong changesets returned. diff=%s", diff)
		}
	})
}

func testStoreListChangesetSyncData(t *testing.T, ctx context.Context, s *Store, reposStore repos.Store, clock clock) {
	githubActor := github.Actor{
		AvatarURL: "https://avatars2.githubusercontent.com/u/1185253",
		Login:     "mrnugget",
		URL:       "https://github.com/mrnugget",
	}
	githubPR := &github.PullRequest{
		ID:           "FOOBARID",
		Title:        "Fix a bunch of bugs",
		Body:         "This fixes a bunch of bugs",
		URL:          "https://github.com/sourcegraph/sourcegraph/pull/12345",
		Number:       12345,
		Author:       githubActor,
		Participants: []github.Actor{githubActor},
		CreatedAt:    clock.now(),
		UpdatedAt:    clock.now(),
		HeadRefName:  "campaigns/test",
	}
	issueComment := &github.IssueComment{
		DatabaseID: 443827703,
		Author: github.Actor{
			AvatarURL: "https://avatars0.githubusercontent.com/u/1976?v=4",
			Login:     "sqs",
			URL:       "https://github.com/sqs",
		},
		Editor:              nil,
		AuthorAssociation:   "MEMBER",
		Body:                "> Just to be sure: you mean the \"searchFilters\" \"Filters\" should be lowercase, not the \"Search Filters\" from the description, right?\r\n\r\nNo, the prose “Search Filters” should have the F lowercased to fit with our style guide preference for sentence case over title case. (Can’t find this comment on the GitHub mobile interface anymore so quoting the email.)",
		URL:                 "https://github.com/sourcegraph/sourcegraph/pull/999#issuecomment-443827703",
		CreatedAt:           clock.now(),
		UpdatedAt:           clock.now(),
		IncludesCreatedEdit: false,
	}

	repo := testRepo(t, reposStore, extsvc.KindGitHub)
	if err := reposStore.InsertRepos(ctx, repo); err != nil {
		t.Fatal(err)
	}

	changesets := make(cmpgn.Changesets, 0, 3)
	events := make([]*cmpgn.ChangesetEvent, 0)

	for i := 0; i < cap(changesets); i++ {
		ch := &cmpgn.Changeset{
			RepoID:              repo.ID,
			CreatedAt:           clock.now(),
			UpdatedAt:           clock.now(),
			Metadata:            githubPR,
			CampaignIDs:         []int64{int64(i) + 1},
			ExternalID:          fmt.Sprintf("foobar-%d", i),
			ExternalServiceType: extsvc.TypeGitHub,
			ExternalBranch:      "campaigns/test",
			ExternalUpdatedAt:   clock.now(),
			ExternalState:       cmpgn.ChangesetExternalStateOpen,
			ExternalReviewState: cmpgn.ChangesetReviewStateApproved,
			ExternalCheckState:  cmpgn.ChangesetCheckStatePassed,
			PublicationState:    cmpgn.ChangesetPublicationStatePublished,
			ReconcilerState:     cmpgn.ReconcilerStateCompleted,
		}

		if err := s.CreateChangeset(ctx, ch); err != nil {
			t.Fatal(err)
		}

		changesets = append(changesets, ch)
	}

	// We need campaigns attached to each changeset
	for _, cs := range changesets {
		c := &cmpgn.Campaign{
			Name:           "ListChangesetSyncData test",
			ChangesetIDs:   []int64{cs.ID},
			NamespaceOrgID: 23,
			LastApplierID:  1,
			LastAppliedAt:  time.Now(),
			CampaignSpecID: 42,
		}
		err := s.CreateCampaign(ctx, c)
		if err != nil {
			t.Fatal(err)
		}
		cs.CampaignIDs = []int64{c.ID}

		if err := s.UpdateChangeset(ctx, cs); err != nil {
			t.Fatal(err)
		}
	}

	// The changesets, except one, get changeset events
	for _, cs := range changesets[:len(changesets)-1] {
		e := &cmpgn.ChangesetEvent{
			ChangesetID: cs.ID,
			Kind:        cmpgn.ChangesetEventKindGitHubCommented,
			Key:         issueComment.Key(),
			CreatedAt:   clock.now(),
			Metadata:    issueComment,
		}

		events = append(events, e)
	}
	if err := s.UpsertChangesetEvents(ctx, events...); err != nil {
		t.Fatal(err)
	}

	checkChangesetIDs := func(t *testing.T, hs []cmpgn.ChangesetSyncData, want []int64) {
		t.Helper()

		haveIDs := []int64{}
		for _, sd := range hs {
			haveIDs = append(haveIDs, sd.ChangesetID)
		}
		if diff := cmp.Diff(want, haveIDs); diff != "" {
			t.Fatalf("wrong changesetIDs in changeset sync data (-want +got):\n%s", diff)
		}
	}

	t.Run("success", func(t *testing.T) {
		hs, err := s.ListChangesetSyncData(ctx, ListChangesetSyncDataOpts{})
		if err != nil {
			t.Fatal(err)
		}
		want := []cmpgn.ChangesetSyncData{
			{
				ChangesetID:           changesets[0].ID,
				UpdatedAt:             clock.now(),
				LatestEvent:           clock.now(),
				ExternalUpdatedAt:     clock.now(),
				RepoExternalServiceID: "https://example.com/",
			},
			{
				ChangesetID:           changesets[1].ID,
				UpdatedAt:             clock.now(),
				LatestEvent:           clock.now(),
				ExternalUpdatedAt:     clock.now(),
				RepoExternalServiceID: "https://example.com/",
			},
			{
				// No events
				ChangesetID:           changesets[2].ID,
				UpdatedAt:             clock.now(),
				ExternalUpdatedAt:     clock.now(),
				RepoExternalServiceID: "https://example.com/",
			},
		}
		if diff := cmp.Diff(want, hs); diff != "" {
			t.Fatal(diff)
		}
	})

	t.Run("ignore closed campaign", func(t *testing.T) {
		closedCampaignID := changesets[0].CampaignIDs[0]
		c, err := s.GetCampaign(ctx, GetCampaignOpts{ID: closedCampaignID})
		if err != nil {
			t.Fatal(err)
		}
		c.ClosedAt = clock.now()
		err = s.UpdateCampaign(ctx, c)
		if err != nil {
			t.Fatal(err)
		}

		hs, err := s.ListChangesetSyncData(ctx, ListChangesetSyncDataOpts{})
		if err != nil {
			t.Fatal(err)
		}
		checkChangesetIDs(t, hs, changesets[1:].IDs())

		// If a changeset has ANY open campaigns we should list it
		// Attach cs1 to both an open and closed campaign
		openCampaignID := changesets[1].CampaignIDs[0]
		changesets[0].CampaignIDs = []int64{closedCampaignID, openCampaignID}
		err = s.UpdateChangeset(ctx, changesets[0])
		if err != nil {
			t.Fatal(err)
		}

		c1, err := s.GetCampaign(ctx, GetCampaignOpts{ID: openCampaignID})
		if err != nil {
			t.Fatal(err)
		}
		c1.ChangesetIDs = []int64{changesets[0].ID, changesets[1].ID}
		err = s.UpdateCampaign(ctx, c1)
		if err != nil {
			t.Fatal(err)
		}

		hs, err = s.ListChangesetSyncData(ctx, ListChangesetSyncDataOpts{})
		if err != nil {
			t.Fatal(err)
		}
		checkChangesetIDs(t, hs, changesets.IDs())
	})

	t.Run("ignore processing changesets", func(t *testing.T) {
		ch := changesets[0]
		ch.PublicationState = cmpgn.ChangesetPublicationStatePublished
		ch.ReconcilerState = cmpgn.ReconcilerStateProcessing
		if err := s.UpdateChangeset(ctx, ch); err != nil {
			t.Fatal(err)
		}

		hs, err := s.ListChangesetSyncData(ctx, ListChangesetSyncDataOpts{})
		if err != nil {
			t.Fatal(err)
		}
		checkChangesetIDs(t, hs, changesets[1:].IDs())
	})

	t.Run("ignore unpublished changesets", func(t *testing.T) {
		ch := changesets[0]
		ch.PublicationState = cmpgn.ChangesetPublicationStateUnpublished
		ch.ReconcilerState = cmpgn.ReconcilerStateCompleted
		if err := s.UpdateChangeset(ctx, ch); err != nil {
			t.Fatal(err)
		}

		hs, err := s.ListChangesetSyncData(ctx, ListChangesetSyncDataOpts{})
		if err != nil {
			t.Fatal(err)
		}
		checkChangesetIDs(t, hs, changesets[1:].IDs())
	})
}
