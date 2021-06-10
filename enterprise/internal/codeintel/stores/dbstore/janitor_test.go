package dbstore

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/sourcegraph/sourcegraph/internal/database/dbtest"
)

func TestStaleSourcedCommits(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := dbtest.NewDB(t, "")
	store := testStore(db)

	now := time.Unix(1587396557, 0).UTC()

	insertUploads(t, db,
		Upload{ID: 1, RepositoryID: 50, Commit: makeCommit(1)},
		Upload{ID: 2, RepositoryID: 50, Commit: makeCommit(1), Root: "sub/"},
		Upload{ID: 3, RepositoryID: 51, Commit: makeCommit(4)},
		Upload{ID: 4, RepositoryID: 51, Commit: makeCommit(5)},
		Upload{ID: 5, RepositoryID: 52, Commit: makeCommit(7)},
	)
	insertIndexes(t, db,
		Index{ID: 1, RepositoryID: 50, Commit: makeCommit(1)},
		Index{ID: 2, RepositoryID: 50, Commit: makeCommit(2)},
		Index{ID: 3, RepositoryID: 50, Commit: makeCommit(3)},
		Index{ID: 4, RepositoryID: 51, Commit: makeCommit(6)},
		Index{ID: 5, RepositoryID: 52, Commit: makeCommit(7)},
	)

	sourcedCommits, err := store.StaleSourcedCommits(context.Background(), time.Minute, 5, now)
	if err != nil {
		t.Fatalf("unexpected error getting stale sourced commits")
	}
	expectedCommits := []SourcedCommits{
		{RepositoryID: 50, RepositoryName: "n-50", Commits: []string{makeCommit(1), makeCommit(2), makeCommit(3)}},
		{RepositoryID: 51, RepositoryName: "n-51", Commits: []string{makeCommit(4), makeCommit(5)}},
	}
	if diff := cmp.Diff(expectedCommits, sourcedCommits); diff != "" {
		t.Errorf("unexpected sourced commits (-want +got):\n%s", diff)
	}

	// 120s away from next check (threshold is 60s)
	if _, _, err := store.RefreshCommitResolvability(context.Background(), 50, makeCommit(1), false, now); err != nil {
		t.Fatalf("unexpected error refreshing commit resolvability")
	}

	// 30s away from next check (threshold is 60s)
	if _, _, err := store.RefreshCommitResolvability(context.Background(), 50, makeCommit(2), false, now.Add(time.Second*90)); err != nil {
		t.Fatalf("unexpected error refreshing commit resolvability")
	}

	sourcedCommits, err = store.StaleSourcedCommits(context.Background(), time.Minute, 5, now.Add(time.Minute*2))
	if err != nil {
		t.Fatalf("unexpected error getting stale sourced commits")
	}
	expectedCommits = []SourcedCommits{
		{RepositoryID: 50, RepositoryName: "n-50", Commits: []string{makeCommit(1), makeCommit(3)}},
		{RepositoryID: 51, RepositoryName: "n-51", Commits: []string{makeCommit(4), makeCommit(5), makeCommit(6)}},
	}
	if diff := cmp.Diff(expectedCommits, sourcedCommits); diff != "" {
		t.Errorf("unexpected sourced commits (-want +got):\n%s", diff)
	}
}

func TestRefreshCommitResolvability(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := dbtest.NewDB(t, "")
	store := testStore(db)

	now := time.Unix(1587396557, 0).UTC()

	insertUploads(t, db,
		Upload{ID: 1, RepositoryID: 50, Commit: makeCommit(1)},
		Upload{ID: 2, RepositoryID: 50, Commit: makeCommit(1), Root: "sub/"},
		Upload{ID: 3, RepositoryID: 51, Commit: makeCommit(4)},
		Upload{ID: 4, RepositoryID: 51, Commit: makeCommit(5)},
		Upload{ID: 5, RepositoryID: 52, Commit: makeCommit(7)},
	)
	insertIndexes(t, db,
		Index{ID: 1, RepositoryID: 50, Commit: makeCommit(3)},
		Index{ID: 2, RepositoryID: 50, Commit: makeCommit(2)},
		Index{ID: 3, RepositoryID: 52, Commit: makeCommit(7)},
		Index{ID: 4, RepositoryID: 51, Commit: makeCommit(6)},
		Index{ID: 5, RepositoryID: 50, Commit: makeCommit(1)},
	)

	uploadsUpdated, indexesUpdated, err := store.RefreshCommitResolvability(context.Background(), 50, makeCommit(1), false, now)
	if err != nil {
		t.Fatalf("unexpected error refreshing commit resolvability")
	}
	if uploadsUpdated != 2 {
		t.Fatalf("unexpected uploads updated. want=%d have=%d", 2, uploadsUpdated)
	}
	if indexesUpdated != 1 {
		t.Fatalf("unexpected indexes updated. want=%d have=%d", 1, indexesUpdated)
	}

	uploadsUpdated, indexesUpdated, err = store.RefreshCommitResolvability(context.Background(), 52, makeCommit(7), true, now)
	if err != nil {
		t.Fatalf("unexpected error refreshing commit resolvability")
	}
	if uploadsUpdated != 1 {
		t.Fatalf("unexpected uploads updated. want=%d have=%d", 1, uploadsUpdated)
	}
	if indexesUpdated != 1 {
		t.Fatalf("unexpected indexes updated. want=%d have=%d", 1, indexesUpdated)
	}

	uploadStates, err := getUploadStates(db, 1, 2, 3, 4, 5)
	if err != nil {
		t.Fatalf("unexpected error fetching upload states: %s", err)
	}
	expectedUploadStates := map[int]string{
		1: "completed",
		2: "completed",
		3: "completed",
		4: "completed",
		5: "deleted",
	}
	if diff := cmp.Diff(expectedUploadStates, uploadStates); diff != "" {
		t.Errorf("unexpected upload states (-want +got):\n%s", diff)
	}

	indexStates, err := getIndexStates(db, 1, 2, 3, 4, 5)
	if err != nil {
		t.Fatalf("unexpected error fetching index states: %s", err)
	}
	expectedIndexStates := map[int]string{
		1: "completed",
		2: "completed",
		3: "deleted",
		4: "completed",
		5: "completed",
	}
	if diff := cmp.Diff(expectedIndexStates, indexStates); diff != "" {
		t.Errorf("unexpected index states (-want +got):\n%s", diff)
	}
}
