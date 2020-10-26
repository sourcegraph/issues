package store

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/sourcegraph/sourcegraph/internal/db/dbtesting"
)

func TestIndexableRepositories(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	store := testStore()

	t1 := time.Unix(1587396557, 0).UTC()
	t2 := t1.Add(-time.Hour)
	t3 := t2.Add(-time.Hour)

	updates := []UpdateableIndexableRepository{
		{RepositoryID: 1},
		{RepositoryID: 1, SearchCount: intptr(1)},
		{RepositoryID: 1, SearchCount: intptr(10)},
		{RepositoryID: 1, PreciseCount: intptr(15)},
		{RepositoryID: 2, SearchCount: intptr(20)},
		{RepositoryID: 2, PreciseCount: intptr(2)},
		{RepositoryID: 2, PreciseCount: intptr(25)},
		{RepositoryID: 3, SearchCount: intptr(10), PreciseCount: intptr(20), LastIndexEnqueuedAt: &t1},
		{RepositoryID: 3, SearchCount: intptr(30), PreciseCount: intptr(35)},
		{RepositoryID: 4, SearchCount: intptr(40), PreciseCount: intptr(45), LastIndexEnqueuedAt: &t2},
		{RepositoryID: 5, SearchCount: intptr(50), PreciseCount: intptr(55), LastIndexEnqueuedAt: &t3},
	}

	for _, update := range updates {
		if err := store.UpdateIndexableRepository(context.Background(), update, time.Now().UTC()); err != nil {
			t.Fatalf("unexpected error while updating indexable repository: %s", err)
		}
	}

	indexableRepositories, err := store.IndexableRepositories(context.Background(), IndexableRepositoryQueryOptions{
		Limit: 50,
	})
	if err != nil {
		t.Fatalf("unexpected error while fetching indexable repository: %s", err)
	}

	expectedIndexableRepositories := []IndexableRepository{
		{RepositoryID: 1, SearchCount: 10, PreciseCount: 15, LastIndexEnqueuedAt: nil},
		{RepositoryID: 2, SearchCount: 20, PreciseCount: 25, LastIndexEnqueuedAt: nil},
		{RepositoryID: 3, SearchCount: 30, PreciseCount: 35, LastIndexEnqueuedAt: &t1},
		{RepositoryID: 4, SearchCount: 40, PreciseCount: 45, LastIndexEnqueuedAt: &t2},
		{RepositoryID: 5, SearchCount: 50, PreciseCount: 55, LastIndexEnqueuedAt: &t3},
	}
	if diff := cmp.Diff(expectedIndexableRepositories, indexableRepositories); diff != "" {
		t.Errorf("unexpected ids (-want +got):\n%s", diff)
	}
}

func TestIndexableRepositoriesMinimumTimeSinceLastEnqueue(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	store := testStore()

	t1 := time.Unix(1587396557, 0).UTC()
	t2 := t1.Add(-time.Hour)
	t3 := t2.Add(-time.Hour)
	t4 := t3.Add(-time.Hour)

	updates := []UpdateableIndexableRepository{
		{RepositoryID: 1},
		{RepositoryID: 2, LastIndexEnqueuedAt: &t1},
		{RepositoryID: 3, LastIndexEnqueuedAt: &t2},
		{RepositoryID: 4, LastIndexEnqueuedAt: &t3},
		{RepositoryID: 5, LastIndexEnqueuedAt: &t4},
	}

	for _, update := range updates {
		if err := store.UpdateIndexableRepository(context.Background(), update, t1); err != nil {
			t.Fatalf("unexpected error while updating indexable repository: %s", err)
		}
	}

	indexableRepositories, err := store.IndexableRepositories(context.Background(), IndexableRepositoryQueryOptions{
		Limit:                       50,
		MinimumTimeSinceLastEnqueue: time.Hour + time.Minute*30,
		now:                         t1,
	})
	if err != nil {
		t.Fatalf("unexpected error while fetching indexable repository: %s", err)
	}

	expectedIndexableRepositories := []IndexableRepository{
		{RepositoryID: 1},
		{RepositoryID: 4, LastIndexEnqueuedAt: &t3},
		{RepositoryID: 5, LastIndexEnqueuedAt: &t4},
	}
	if diff := cmp.Diff(expectedIndexableRepositories, indexableRepositories); diff != "" {
		t.Errorf("unexpected ids (-want +got):\n%s", diff)
	}
}

func TestIndexableRepositoriesMinimumSearchAndPreciseCount(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	store := testStore()

	updates := []UpdateableIndexableRepository{
		{RepositoryID: 1, SearchCount: intptr(10), PreciseCount: intptr(30)},
		{RepositoryID: 2, SearchCount: intptr(10), PreciseCount: intptr(40)},
		{RepositoryID: 3, SearchCount: intptr(20), PreciseCount: intptr(10)},
		{RepositoryID: 4, SearchCount: intptr(30), PreciseCount: intptr(10)},
		{RepositoryID: 5, SearchCount: intptr(20), PreciseCount: intptr(30)},
		{RepositoryID: 6, SearchCount: intptr(30), PreciseCount: intptr(40)},
	}

	for _, update := range updates {
		if err := store.UpdateIndexableRepository(context.Background(), update, time.Now().UTC()); err != nil {
			t.Fatalf("unexpected error while updating indexable repository: %s", err)
		}
	}

	indexableRepositories, err := store.IndexableRepositories(context.Background(), IndexableRepositoryQueryOptions{
		Limit:               50,
		MinimumSearchCount:  25,
		MinimumPreciseCount: 35,
	})
	if err != nil {
		t.Fatalf("unexpected error while fetching indexable repository: %s", err)
	}

	expectedIndexableRepositories := []IndexableRepository{
		{RepositoryID: 2, SearchCount: 10, PreciseCount: 40},
		{RepositoryID: 4, SearchCount: 30, PreciseCount: 10},
		{RepositoryID: 6, SearchCount: 30, PreciseCount: 40},
	}
	if diff := cmp.Diff(expectedIndexableRepositories, indexableRepositories); diff != "" {
		t.Errorf("unexpected ids (-want +got):\n%s", diff)
	}
}

func TestIndexableRepositoriesMinimumSearchRatio(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	store := testStore()

	updates := []UpdateableIndexableRepository{
		{RepositoryID: 1, SearchCount: intptr(10)},                           // 100%
		{RepositoryID: 2, SearchCount: intptr(10), PreciseCount: intptr(10)}, // 50%
		{RepositoryID: 3, SearchCount: intptr(10), PreciseCount: intptr(20)}, // 30%
		{RepositoryID: 4, SearchCount: intptr(10), PreciseCount: intptr(30)}, // 25%
		{RepositoryID: 5, SearchCount: intptr(10), PreciseCount: intptr(90)}, // 10%
	}

	for _, update := range updates {
		if err := store.UpdateIndexableRepository(context.Background(), update, time.Now().UTC()); err != nil {
			t.Fatalf("unexpected error while updating indexable repository: %s", err)
		}
	}

	indexableRepositories, err := store.IndexableRepositories(context.Background(), IndexableRepositoryQueryOptions{
		Limit:              50,
		MinimumSearchRatio: 0.28,
	})
	if err != nil {
		t.Fatalf("unexpected error while fetching indexable repository: %s", err)
	}

	expectedIndexableRepositories := []IndexableRepository{
		{RepositoryID: 1, SearchCount: 10},
		{RepositoryID: 2, SearchCount: 10, PreciseCount: 10},
		{RepositoryID: 3, SearchCount: 10, PreciseCount: 20},
	}
	if diff := cmp.Diff(expectedIndexableRepositories, indexableRepositories); diff != "" {
		t.Errorf("unexpected ids (-want +got):\n%s", diff)
	}
}

func TestIndexableRepositoriesEnabled(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	store := testStore()

	updates := []UpdateableIndexableRepository{
		{RepositoryID: 1},
		{RepositoryID: 2, Enabled: boolptr(true)},
		{RepositoryID: 3, Enabled: boolptr(false)},
		{RepositoryID: 4, SearchCount: intptr(10)},                                                    // 100%
		{RepositoryID: 5, SearchCount: intptr(10), Enabled: boolptr(true)},                            // 100%
		{RepositoryID: 6, SearchCount: intptr(10), Enabled: boolptr(false)},                           // 100%
		{RepositoryID: 7, SearchCount: intptr(10), PreciseCount: intptr(30)},                          // 30%
		{RepositoryID: 8, SearchCount: intptr(10), PreciseCount: intptr(30), Enabled: boolptr(true)},  // 30%
		{RepositoryID: 9, SearchCount: intptr(10), PreciseCount: intptr(30), Enabled: boolptr(false)}, // 30%
	}

	for _, update := range updates {
		if err := store.UpdateIndexableRepository(context.Background(), update, time.Now().UTC()); err != nil {
			t.Fatalf("unexpected error while updating indexable repository: %s", err)
		}
	}

	indexableRepositories, err := store.IndexableRepositories(context.Background(), IndexableRepositoryQueryOptions{
		Limit:              50,
		MinimumSearchRatio: 0.50,
	})
	if err != nil {
		t.Fatalf("unexpected error while fetching indexable repository: %s", err)
	}

	expectedIndexableRepositories := []IndexableRepository{
		{RepositoryID: 2, Enabled: boolptr(true)},
		{RepositoryID: 4, SearchCount: 10},
		{RepositoryID: 5, SearchCount: 10, Enabled: boolptr(true)},
		{RepositoryID: 8, SearchCount: 10, PreciseCount: 30, Enabled: boolptr(true)},
	}
	if diff := cmp.Diff(expectedIndexableRepositories, indexableRepositories); diff != "" {
		t.Errorf("unexpected ids (-want +got):\n%s", diff)
	}
}

func TestResetIndexableRepositories(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	store := testStore()

	t1 := time.Unix(1587396557, 0).UTC()
	t2 := t1.Add(-time.Hour)
	t3 := t2.Add(-time.Hour)
	t4 := t3.Add(-time.Hour)
	t5 := t4.Add(-time.Hour)

	for i, now := range []time.Time{t1, t2, t3, t4, t5} {
		update := UpdateableIndexableRepository{
			RepositoryID: i + 1,
			SearchCount:  intptr((i + 1) * 10),
			PreciseCount: intptr((i + 1) * 10),
		}

		if err := store.UpdateIndexableRepository(context.Background(), update, now); err != nil {
			t.Fatalf("unexpected error while updating indexable repository: %s", err)
		}
	}

	err := store.ResetIndexableRepositories(context.Background(), t3.Add(time.Second))
	if err != nil {
		t.Fatalf("unexpected error while resetting indexable repositories: %s", err)
	}

	indexableRepositories, err := store.IndexableRepositories(context.Background(), IndexableRepositoryQueryOptions{
		Limit: 10,
	})
	if err != nil {
		t.Fatalf("unexpected error while fetching indexable repository: %s", err)
	}

	expectedIndexableRepositories := []IndexableRepository{
		{RepositoryID: 1, SearchCount: 10, PreciseCount: 10},
		{RepositoryID: 2, SearchCount: 20, PreciseCount: 20},
		{RepositoryID: 3, SearchCount: 0, PreciseCount: 0},
		{RepositoryID: 4, SearchCount: 0, PreciseCount: 0},
		{RepositoryID: 5, SearchCount: 0, PreciseCount: 0},
	}
	if diff := cmp.Diff(expectedIndexableRepositories, indexableRepositories); diff != "" {
		t.Errorf("unexpected ids (-want +got):\n%s", diff)
	}
}

func intptr(val int) *int {
	return &val
}

func boolptr(val bool) *bool {
	return &val
}
