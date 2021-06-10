package dbstore

import (
	"context"
	"testing"

	"github.com/sourcegraph/sourcegraph/internal/database/dbtest"
)

func TestInsertDependencyIndexingJob(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := dbtest.NewDB(t, "")
	store := testStore(db)

	insertRepo(t, db, 50, "")

	uploadID, err := store.InsertUpload(context.Background(), Upload{
		Commit:        makeCommit(1),
		Root:          "sub/",
		State:         "queued",
		RepositoryID:  50,
		Indexer:       "lsif-go",
		NumParts:      1,
		UploadedParts: []int{0},
	})
	if err != nil {
		t.Fatalf("unexpected error enqueueing upload: %s", err)
	}

	// No error if upload exists
	if _, err := store.InsertDependencyIndexingJob(context.Background(), uploadID); err != nil {
		t.Fatalf("unexpected error enqueueing dependency indexing job: %s", err)
	}

	// Error with unknown identifier
	if _, err := store.InsertDependencyIndexingJob(context.Background(), uploadID+1); err == nil {
		t.Fatalf("expected error enqueueing dependency indexing job for unknown upload")
	}
}
