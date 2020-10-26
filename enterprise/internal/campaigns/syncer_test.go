package campaigns

import (
	"container/heap"
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	"github.com/sourcegraph/sourcegraph/cmd/repo-updater/repos"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/campaigns"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
)

func TestNextSync(t *testing.T) {
	clock := func() time.Time { return time.Date(2020, 01, 01, 01, 01, 01, 01, time.UTC) }
	tests := []struct {
		name string
		h    campaigns.ChangesetSyncData
		want time.Time
	}{
		{
			name: "No time passed",
			h: campaigns.ChangesetSyncData{
				UpdatedAt:         clock(),
				ExternalUpdatedAt: clock(),
			},
			want: clock().Add(minSyncDelay),
		},
		{
			name: "Linear backoff",
			h: campaigns.ChangesetSyncData{
				UpdatedAt:         clock(),
				ExternalUpdatedAt: clock().Add(-1 * time.Hour),
			},
			want: clock().Add(1 * time.Hour),
		},
		{
			name: "Use max of ExternalUpdateAt and LatestEvent",
			h: campaigns.ChangesetSyncData{
				UpdatedAt:         clock(),
				ExternalUpdatedAt: clock().Add(-2 * time.Hour),
				LatestEvent:       clock().Add(-1 * time.Hour),
			},
			want: clock().Add(1 * time.Hour),
		},
		{
			name: "Diff max is capped",
			h: campaigns.ChangesetSyncData{
				UpdatedAt:         clock(),
				ExternalUpdatedAt: clock().Add(-2 * maxSyncDelay),
			},
			want: clock().Add(maxSyncDelay),
		},
		{
			name: "Diff min is capped",
			h: campaigns.ChangesetSyncData{
				UpdatedAt:         clock(),
				ExternalUpdatedAt: clock().Add(-1 * minSyncDelay / 2),
			},
			want: clock().Add(minSyncDelay),
		},
		{
			name: "Event arrives after sync",
			h: campaigns.ChangesetSyncData{
				UpdatedAt:         clock(),
				ExternalUpdatedAt: clock().Add(-1 * maxSyncDelay / 2),
				LatestEvent:       clock().Add(10 * time.Minute),
			},
			want: clock().Add(10 * time.Minute).Add(minSyncDelay),
		},
		{
			name: "Never synced",
			h:    campaigns.ChangesetSyncData{},
			want: clock(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NextSync(clock, tt.h)
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestChangesetPriorityQueue(t *testing.T) {
	assertOrder := func(t *testing.T, q *changesetPriorityQueue, expected []int64) {
		t.Helper()
		ids := make([]int64, len(q.items))
		for i := range ids {
			ids[i] = q.items[i].changesetID
		}
		if diff := cmp.Diff(expected, ids); diff != "" {
			t.Fatal(diff)
		}
	}

	now := time.Now()
	q := newChangesetPriorityQueue()

	items := []scheduledSync{
		{
			changesetID: 1,
			nextSync:    now,
			priority:    priorityNormal,
		},
		{
			changesetID: 2,
			nextSync:    now,
			priority:    priorityHigh,
		},
		{
			changesetID: 3,
			nextSync:    now.Add(-1 * time.Minute),
			priority:    priorityNormal,
		},
		{
			changesetID: 4,
			nextSync:    now.Add(-2 * time.Hour),
			priority:    priorityNormal,
		},
		{
			changesetID: 5,
			nextSync:    now.Add(1 * time.Hour),
			priority:    priorityNormal,
		},
	}

	for i := range items {
		q.Upsert(items[i])
	}

	assertOrder(t, q, []int64{2, 4, 3, 1, 5})

	// Set item to high priority
	q.Upsert(scheduledSync{
		changesetID: 4,
		nextSync:    now.Add(-2 * time.Hour),
		priority:    priorityHigh,
	})

	assertOrder(t, q, []int64{4, 2, 3, 1, 5})

	// Can't reduce priority of existing item
	q.Upsert(scheduledSync{
		changesetID: 4,
		nextSync:    now.Add(-2 * time.Hour),
		priority:    priorityNormal,
	})

	if q.Len() != len(items) {
		t.Fatalf("Expected %d, got %d", q.Len(), len(items))
	}

	assertOrder(t, q, []int64{4, 2, 3, 1, 5})

	for i := 0; i < len(items); i++ {
		peeked, ok := q.Peek()
		if !ok {
			t.Fatalf("Queue should not be empty")
		}
		item := heap.Pop(q).(scheduledSync)
		if peeked.changesetID != item.changesetID {
			t.Fatalf("Peeked and Popped item should have the same id")
		}
	}

	// Len() should be zero after all items popped
	if q.Len() != 0 {
		t.Fatalf("Expected %d, got %d", q.Len(), 0)
	}
}

func TestPrioritizeChangesetsWithoutDiffStats(t *testing.T) {
	for name, tc := range map[string]struct {
		listChangesets func(context.Context, ListChangesetsOpts) (campaigns.Changesets, int64, error)
		wantError      bool
		wantIDs        []int64
	}{
		"ListChangesets error": {
			listChangesets: func(ctx context.Context, opts ListChangesetsOpts) (campaigns.Changesets, int64, error) {
				return nil, 0, errors.New("hello!")
			},
			wantError: true,
		},
		"empty list": {
			listChangesets: func(ctx context.Context, opts ListChangesetsOpts) (campaigns.Changesets, int64, error) {
				return []*campaigns.Changeset{}, 0, nil
			},
		},
		"non-empty list with published changesets": {
			listChangesets: func(ctx context.Context, opts ListChangesetsOpts) (campaigns.Changesets, int64, error) {
				return []*campaigns.Changeset{
					{
						ID:               1,
						ReconcilerState:  campaigns.ReconcilerStateCompleted,
						PublicationState: campaigns.ChangesetPublicationStatePublished,
					},
					{
						ID:               2,
						ReconcilerState:  campaigns.ReconcilerStateCompleted,
						PublicationState: campaigns.ChangesetPublicationStatePublished,
					},
				}, 0, nil
			},
			wantIDs: []int64{1, 2},
		},
		"non-empty list with unpublished changesets": {
			listChangesets: func(ctx context.Context, opts ListChangesetsOpts) (campaigns.Changesets, int64, error) {
				return []*campaigns.Changeset{
					{
						ID:               1,
						ReconcilerState:  campaigns.ReconcilerStateCompleted,
						PublicationState: campaigns.ChangesetPublicationStateUnpublished,
					},
					{
						ID:               2,
						ReconcilerState:  campaigns.ReconcilerStateCompleted,
						PublicationState: campaigns.ChangesetPublicationStateUnpublished,
					},
				}, 0, nil
			},
			wantIDs: []int64{},
		},
		"non-empty list with processing changesets": {
			listChangesets: func(ctx context.Context, opts ListChangesetsOpts) (campaigns.Changesets, int64, error) {
				return []*campaigns.Changeset{
					{
						ID:               1,
						ReconcilerState:  campaigns.ReconcilerStateProcessing,
						PublicationState: campaigns.ChangesetPublicationStatePublished,
					},
					{
						ID:               2,
						ReconcilerState:  campaigns.ReconcilerStateProcessing,
						PublicationState: campaigns.ChangesetPublicationStatePublished,
					},
				}, 0, nil
			},
			wantIDs: []int64{},
		},
	} {
		t.Run(name, func(t *testing.T) {
			var wg sync.WaitGroup

			syncer := &ChangesetSyncer{
				SyncStore:      MockSyncStore{listChangesets: tc.listChangesets},
				priorityNotify: make(chan []int64),
			}

			wg.Add(1)
			go func() {
				defer wg.Done()
				var (
					ids  []int64
					seen bool
				)

				for ids = range syncer.priorityNotify {
					if seen {
						t.Error("received more than one message on priorityNotify")
					}
					seen = true
				}

				if diff := cmp.Diff(ids, tc.wantIDs); diff != "" {
					t.Errorf("invalid IDs received on the priorityNotify channel: have %+v; want %+v", ids, tc.wantIDs)
				}
			}()

			err := syncer.prioritizeChangesetsWithoutDiffStats(context.Background())
			if tc.wantError && err == nil {
				t.Error("expected an error; got nil")
			} else if !tc.wantError && err != nil {
				t.Errorf("got unexpected error: %+v", err)
			}

			// We have to wait for the goroutine to exit lest we fail after this
			// function is complete.
			close(syncer.priorityNotify)
			wg.Wait()
		})
	}
}

func TestSyncerRun(t *testing.T) {
	t.Run("Sync due", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		now := time.Now()
		store := MockSyncStore{
			listChangesets: func(ctx context.Context, opts ListChangesetsOpts) (campaigns.Changesets, int64, error) {
				return nil, 0, nil
			},
			listChangesetSyncData: func(ctx context.Context, opts ListChangesetSyncDataOpts) ([]campaigns.ChangesetSyncData, error) {
				return []campaigns.ChangesetSyncData{
					{
						ChangesetID:       1,
						UpdatedAt:         now.Add(-2 * maxSyncDelay),
						LatestEvent:       now.Add(-2 * maxSyncDelay),
						ExternalUpdatedAt: now.Add(-2 * maxSyncDelay),
					},
				}, nil
			},
		}
		syncFunc := func(ctx context.Context, ids int64) error {
			cancel()
			return nil
		}
		syncer := &ChangesetSyncer{
			SyncStore:        store,
			scheduleInterval: 10 * time.Minute,
			syncFunc:         syncFunc,
		}
		go syncer.Run(ctx)
		select {
		case <-ctx.Done():
		case <-time.After(50 * time.Millisecond):
			t.Fatal("Sync should have been triggered")
		}
	})

	t.Run("Sync not due", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()
		now := time.Now()
		store := MockSyncStore{
			listChangesets: mockListChangesets,
			listChangesetSyncData: func(ctx context.Context, opts ListChangesetSyncDataOpts) ([]campaigns.ChangesetSyncData, error) {
				return []campaigns.ChangesetSyncData{
					{
						ChangesetID:       1,
						UpdatedAt:         now,
						LatestEvent:       now,
						ExternalUpdatedAt: now,
					},
				}, nil
			},
		}
		var syncCalled bool
		syncFunc := func(ctx context.Context, ids int64) error {
			syncCalled = true
			return nil
		}
		syncer := &ChangesetSyncer{
			SyncStore:        store,
			scheduleInterval: 10 * time.Minute,
			syncFunc:         syncFunc,
		}
		syncer.Run(ctx)
		if syncCalled {
			t.Fatal("Sync should not have been triggered")
		}
	})

	t.Run("Priority added", func(t *testing.T) {
		// Empty schedule but then we add an item
		ctx, cancel := context.WithCancel(context.Background())
		store := MockSyncStore{
			listChangesets: mockListChangesets,
			listChangesetSyncData: func(ctx context.Context, opts ListChangesetSyncDataOpts) ([]campaigns.ChangesetSyncData, error) {
				return []campaigns.ChangesetSyncData{}, nil
			},
		}
		syncFunc := func(ctx context.Context, ids int64) error {
			cancel()
			return nil
		}
		syncer := &ChangesetSyncer{
			SyncStore:        store,
			scheduleInterval: 10 * time.Minute,
			syncFunc:         syncFunc,
			priorityNotify:   make(chan []int64, 1),
		}
		syncer.priorityNotify <- []int64{1}
		go syncer.Run(ctx)
		select {
		case <-ctx.Done():
		case <-time.After(50 * time.Millisecond):
			t.Fatal("Sync not called")
		}
	})

}

func TestFilterSyncData(t *testing.T) {
	testCases := []struct {
		name        string
		codeHostURL string
		data        []campaigns.ChangesetSyncData
		want        []campaigns.ChangesetSyncData
	}{
		{
			name:        "Empty",
			codeHostURL: "https://example.com/",
			data:        []campaigns.ChangesetSyncData{},
			want:        []campaigns.ChangesetSyncData{},
		},
		{
			name:        "single item, should match",
			codeHostURL: "https://example.com/",
			data: []campaigns.ChangesetSyncData{
				{
					ChangesetID:           1,
					RepoExternalServiceID: "https://example.com/",
				},
			},
			want: []campaigns.ChangesetSyncData{
				{
					ChangesetID:           1,
					RepoExternalServiceID: "https://example.com/",
				},
			},
		},
		{
			name:        "single item, should not match",
			codeHostURL: "https://example.com/",
			data: []campaigns.ChangesetSyncData{
				{
					ChangesetID:           1,
					RepoExternalServiceID: "https://example2.com/",
				},
			},
			want: []campaigns.ChangesetSyncData{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data := filterSyncData(tc.codeHostURL, tc.data)
			if diff := cmp.Diff(tc.want, data); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestSyncRegistry(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	now := time.Now()

	repoStore := MockRepoStore{
		listExternalServices: func(ctx context.Context, args repos.StoreListExternalServicesArgs) (services []*repos.ExternalService, err error) {
			return []*repos.ExternalService{
				{
					ID:          1,
					Kind:        extsvc.KindGitHub,
					DisplayName: "",
					Config:      `{"url": "https://example.com/"}`,
					CreatedAt:   time.Time{},
					UpdatedAt:   time.Time{},
				},
			}, nil
		},
	}

	syncStore := MockSyncStore{
		listChangesets: mockListChangesets,
		listChangesetSyncData: func(ctx context.Context, opts ListChangesetSyncDataOpts) (data []campaigns.ChangesetSyncData, err error) {
			return []campaigns.ChangesetSyncData{
				{
					ChangesetID:           1,
					UpdatedAt:             now,
					RepoExternalServiceID: "https://example.com/",
				},
			}, nil
		},
	}

	r := NewSyncRegistry(ctx, syncStore, repoStore, nil)

	assertSyncerCount := func(want int) {
		r.mu.Lock()
		if len(r.syncers) != want {
			t.Fatalf("Expected %d syncer, got %d", want, len(r.syncers))
		}
		r.mu.Unlock()
	}

	assertSyncerCount(1)

	// Adding it again should have no effect
	r.Add(1)
	assertSyncerCount(1)

	// Simulate a service being removed
	r.HandleExternalServiceSync(api.ExternalService{
		ID:        1,
		Kind:      extsvc.KindGitHub,
		Config:    `{"url": "https://example.com/"}`,
		DeletedAt: &now,
	})
	assertSyncerCount(0)

	// And added again
	r.HandleExternalServiceSync(api.ExternalService{
		ID:        1,
		Kind:      extsvc.KindGitHub,
		DeletedAt: nil,
	})
	assertSyncerCount(1)

	syncChan := make(chan int64, 1)

	// In order to test that priority items are delivered we'll inject our own syncer
	// with a custom sync func
	syncer := &ChangesetSyncer{
		SyncStore:   syncStore,
		ReposStore:  repoStore,
		HTTPFactory: nil,
		codeHostURL: "https://example.com/",
		syncFunc: func(ctx context.Context, id int64) error {
			syncChan <- id
			return nil
		},
		priorityNotify: make(chan []int64, 1),
	}
	go syncer.Run(ctx)

	// Set the syncer
	r.mu.Lock()
	r.syncers["https://example.com/"] = syncer
	r.mu.Unlock()

	// Send priority items
	err := r.EnqueueChangesetSyncs(ctx, []int64{1, 2})
	if err != nil {
		t.Fatal(err)
	}

	select {
	case id := <-syncChan:
		if id != 1 {
			t.Fatalf("Expected 1, got %d", id)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timed out waiting for sync")
	}
}

type MockSyncStore struct {
	listChangesetSyncData func(context.Context, ListChangesetSyncDataOpts) ([]campaigns.ChangesetSyncData, error)
	getChangeset          func(context.Context, GetChangesetOpts) (*campaigns.Changeset, error)
	listChangesets        func(context.Context, ListChangesetsOpts) (campaigns.Changesets, int64, error)
	updateChangeset       func(context.Context, *campaigns.Changeset) error
	upsertChangesetEvents func(context.Context, ...*campaigns.ChangesetEvent) error
	transact              func(context.Context) (*Store, error)
}

func (m MockSyncStore) ListChangesetSyncData(ctx context.Context, opts ListChangesetSyncDataOpts) ([]campaigns.ChangesetSyncData, error) {
	return m.listChangesetSyncData(ctx, opts)
}

func (m MockSyncStore) GetChangeset(ctx context.Context, opts GetChangesetOpts) (*campaigns.Changeset, error) {
	return m.getChangeset(ctx, opts)
}

func (m MockSyncStore) ListChangesets(ctx context.Context, opts ListChangesetsOpts) (campaigns.Changesets, int64, error) {
	return m.listChangesets(ctx, opts)
}

func (m MockSyncStore) UpdateChangeset(ctx context.Context, c *campaigns.Changeset) error {
	return m.updateChangeset(ctx, c)
}

func (m MockSyncStore) UpsertChangesetEvents(ctx context.Context, cs ...*campaigns.ChangesetEvent) error {
	return m.upsertChangesetEvents(ctx, cs...)
}

func (m MockSyncStore) Transact(ctx context.Context) (*Store, error) {
	return m.transact(ctx)
}

type MockRepoStore struct {
	listExternalServices func(context.Context, repos.StoreListExternalServicesArgs) ([]*repos.ExternalService, error)
	listRepos            func(context.Context, repos.StoreListReposArgs) ([]*repos.Repo, error)
}

func (m MockRepoStore) UpsertExternalServices(ctx context.Context, svcs ...*repos.ExternalService) error {
	panic("implement me")
}

func (m MockRepoStore) UpsertRepos(ctx context.Context, repos ...*repos.Repo) error {
	panic("implement me")
}

func (m MockRepoStore) ListExternalServices(ctx context.Context, args repos.StoreListExternalServicesArgs) ([]*repos.ExternalService, error) {
	return m.listExternalServices(ctx, args)
}

func (m MockRepoStore) ListRepos(ctx context.Context, args repos.StoreListReposArgs) ([]*repos.Repo, error) {
	return m.listRepos(ctx, args)
}

func (m MockRepoStore) SetClonedRepos(ctx context.Context, repoNames ...string) error {
	panic("implement me")
}

func (m MockRepoStore) CountNotClonedRepos(ctx context.Context) (uint64, error) {
	panic("implement me")
}

func mockListChangesets(ctx context.Context, opts ListChangesetsOpts) (campaigns.Changesets, int64, error) {
	return nil, 0, nil
}
