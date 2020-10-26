package store

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/keegancsmith/sqlf"
	"github.com/sourcegraph/sourcegraph/internal/db/basestore"
	"github.com/sourcegraph/sourcegraph/internal/db/dbconn"
	"github.com/sourcegraph/sourcegraph/internal/db/dbtesting"
)

func TestGetUploadByID(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	store := testStore()

	// Upload does not exist initially
	if _, exists, err := store.GetUploadByID(context.Background(), 1); err != nil {
		t.Fatalf("unexpected error getting upload: %s", err)
	} else if exists {
		t.Fatal("unexpected record")
	}

	uploadedAt := time.Unix(1587396557, 0).UTC()
	startedAt := uploadedAt.Add(time.Minute)
	expected := Upload{
		ID:             1,
		Commit:         makeCommit(1),
		Root:           "sub/",
		VisibleAtTip:   true,
		UploadedAt:     uploadedAt,
		State:          "processing",
		FailureMessage: nil,
		StartedAt:      &startedAt,
		FinishedAt:     nil,
		RepositoryID:   123,
		RepositoryName: "n-123",
		Indexer:        "lsif-go",
		NumParts:       1,
		UploadedParts:  []int{},
		Rank:           nil,
	}

	insertUploads(t, dbconn.Global, expected)
	insertVisibleAtTip(t, dbconn.Global, 123, 1)

	if upload, exists, err := store.GetUploadByID(context.Background(), 1); err != nil {
		t.Fatalf("unexpected error getting upload: %s", err)
	} else if !exists {
		t.Fatal("expected record to exist")
	} else if diff := cmp.Diff(expected, upload); diff != "" {
		t.Errorf("unexpected upload (-want +got):\n%s", diff)
	}
}

func TestGetUploadByIDDeleted(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	store := testStore()

	// Upload does not exist initially
	if _, exists, err := store.GetUploadByID(context.Background(), 1); err != nil {
		t.Fatalf("unexpected error getting upload: %s", err)
	} else if exists {
		t.Fatal("unexpected record")
	}

	uploadedAt := time.Unix(1587396557, 0).UTC()
	startedAt := uploadedAt.Add(time.Minute)
	expected := Upload{
		ID:             1,
		Commit:         makeCommit(1),
		Root:           "sub/",
		VisibleAtTip:   true,
		UploadedAt:     uploadedAt,
		State:          "deleted",
		FailureMessage: nil,
		StartedAt:      &startedAt,
		FinishedAt:     nil,
		RepositoryID:   123,
		RepositoryName: "n-123",
		Indexer:        "lsif-go",
		NumParts:       1,
		UploadedParts:  []int{},
		Rank:           nil,
	}

	insertUploads(t, dbconn.Global, expected)

	// Should still not be queryable
	if _, exists, err := store.GetUploadByID(context.Background(), 1); err != nil {
		t.Fatalf("unexpected error getting upload: %s", err)
	} else if exists {
		t.Fatal("unexpected record")
	}
}

func TestGetQueuedUploadRank(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	store := testStore()

	t1 := time.Unix(1587396557, 0).UTC()
	t2 := t1.Add(+time.Minute * 6)
	t3 := t1.Add(+time.Minute * 3)
	t4 := t1.Add(+time.Minute * 1)
	t5 := t1.Add(+time.Minute * 4)
	t6 := t1.Add(+time.Minute * 2)
	t7 := t1.Add(+time.Minute * 5)

	insertUploads(t, dbconn.Global,
		Upload{ID: 1, UploadedAt: t1, State: "queued"},
		Upload{ID: 2, UploadedAt: t2, State: "queued"},
		Upload{ID: 3, UploadedAt: t3, State: "queued"},
		Upload{ID: 4, UploadedAt: t4, State: "queued"},
		Upload{ID: 5, UploadedAt: t5, State: "queued"},
		Upload{ID: 6, UploadedAt: t6, State: "processing"},
		Upload{ID: 7, UploadedAt: t1, State: "queued", ProcessAfter: &t7},
	)

	if upload, _, _ := store.GetUploadByID(context.Background(), 1); upload.Rank == nil || *upload.Rank != 1 {
		t.Errorf("unexpected rank. want=%d have=%s", 1, printableRank{upload.Rank})
	}
	if upload, _, _ := store.GetUploadByID(context.Background(), 2); upload.Rank == nil || *upload.Rank != 6 {
		t.Errorf("unexpected rank. want=%d have=%s", 5, printableRank{upload.Rank})
	}
	if upload, _, _ := store.GetUploadByID(context.Background(), 3); upload.Rank == nil || *upload.Rank != 3 {
		t.Errorf("unexpected rank. want=%d have=%s", 3, printableRank{upload.Rank})
	}
	if upload, _, _ := store.GetUploadByID(context.Background(), 4); upload.Rank == nil || *upload.Rank != 2 {
		t.Errorf("unexpected rank. want=%d have=%s", 2, printableRank{upload.Rank})
	}
	if upload, _, _ := store.GetUploadByID(context.Background(), 5); upload.Rank == nil || *upload.Rank != 4 {
		t.Errorf("unexpected rank. want=%d have=%s", 4, printableRank{upload.Rank})
	}

	// Only considers queued uploads to determine rank
	if upload, _, _ := store.GetUploadByID(context.Background(), 6); upload.Rank != nil {
		t.Errorf("unexpected rank. want=%s have=%s", "nil", printableRank{upload.Rank})
	}

	// Process after takes priority over upload time
	if upload, _, _ := store.GetUploadByID(context.Background(), 7); upload.Rank == nil || *upload.Rank != 5 {
		t.Errorf("unexpected rank. want=%d have=%s", 4, printableRank{upload.Rank})
	}
}

func TestDeleteUploadsStuckUploading(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	store := testStore()

	t1 := time.Unix(1587396557, 0).UTC()
	t2 := t1.Add(time.Minute * 1)
	t3 := t1.Add(time.Minute * 2)
	t4 := t1.Add(time.Minute * 3)
	t5 := t1.Add(time.Minute * 4)

	insertUploads(t, dbconn.Global,
		Upload{ID: 1, Commit: makeCommit(1111), UploadedAt: t1, State: "queued"},    // not uploading
		Upload{ID: 2, Commit: makeCommit(1112), UploadedAt: t2, State: "uploading"}, // deleted
		Upload{ID: 3, Commit: makeCommit(1113), UploadedAt: t3, State: "uploading"}, // deleted
		Upload{ID: 4, Commit: makeCommit(1114), UploadedAt: t4, State: "completed"}, // old, not uploading
		Upload{ID: 5, Commit: makeCommit(1115), UploadedAt: t5, State: "uploading"}, // old
	)

	count, err := store.DeleteUploadsStuckUploading(context.Background(), t1.Add(time.Minute*3))
	if err != nil {
		t.Fatalf("unexpected error deleting uploads stuck uploading: %s", err)
	}
	if count != 2 {
		t.Errorf("unexpected count. want=%d have=%d", 2, count)
	}

	uploads, totalCount, err := store.GetUploads(context.Background(), GetUploadsOptions{Limit: 5})
	if err != nil {
		t.Fatalf("unexpected error getting uploads: %s", err)
	}

	var ids []int
	for _, upload := range uploads {
		ids = append(ids, upload.ID)
	}
	sort.Ints(ids)

	expectedIDs := []int{1, 4, 5}

	if totalCount != len(expectedIDs) {
		t.Errorf("unexpected total count. want=%d have=%d", len(expectedIDs), totalCount)
	}
	if diff := cmp.Diff(expectedIDs, ids); diff != "" {
		t.Errorf("unexpected upload ids (-want +got):\n%s", diff)
	}
}

func TestGetUploads(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	store := testStore()

	t1 := time.Unix(1587396557, 0).UTC()
	t2 := t1.Add(-time.Minute * 1)
	t3 := t1.Add(-time.Minute * 2)
	t4 := t1.Add(-time.Minute * 3)
	t5 := t1.Add(-time.Minute * 4)
	t6 := t1.Add(-time.Minute * 5)
	t7 := t1.Add(-time.Minute * 6)
	t8 := t1.Add(-time.Minute * 7)
	t9 := t1.Add(-time.Minute * 8)
	t10 := t1.Add(-time.Minute * 9)
	failureMessage := "unlucky 333"

	insertUploads(t, dbconn.Global,
		Upload{ID: 1, Commit: makeCommit(3331), UploadedAt: t1, Root: "sub1/", State: "queued"},
		Upload{ID: 2, UploadedAt: t2, State: "errored", FailureMessage: &failureMessage, Indexer: "lsif-tsc"},
		Upload{ID: 3, Commit: makeCommit(3333), UploadedAt: t3, Root: "sub2/", State: "queued"},
		Upload{ID: 4, UploadedAt: t4, State: "queued", RepositoryID: 51, RepositoryName: "foo bar x"},
		Upload{ID: 5, Commit: makeCommit(3333), UploadedAt: t5, Root: "sub1/", State: "processing", Indexer: "lsif-tsc"},
		Upload{ID: 6, UploadedAt: t6, Root: "sub2/", State: "processing", RepositoryID: 52, RepositoryName: "foo bar y"},
		Upload{ID: 7, UploadedAt: t7, Root: "sub1/", Indexer: "lsif-tsc"},
		Upload{ID: 8, UploadedAt: t8, Indexer: "lsif-tsc"},
		Upload{ID: 9, UploadedAt: t9, State: "queued"},
		Upload{ID: 10, UploadedAt: t10, Root: "sub1/", Indexer: "lsif-tsc"},

		// Deleted duplicates
		Upload{ID: 11, Commit: makeCommit(3331), UploadedAt: t1, Root: "sub1/", State: "deleted"},
		Upload{ID: 12, UploadedAt: t2, State: "deleted", FailureMessage: &failureMessage, Indexer: "lsif-tsc"},
		Upload{ID: 13, Commit: makeCommit(3333), UploadedAt: t3, Root: "sub2/", State: "deleted"},
	)
	insertVisibleAtTip(t, dbconn.Global, 50, 2, 5, 7, 8)

	testCases := []struct {
		repositoryID   int
		state          string
		term           string
		visibleAtTip   bool
		uploadedBefore *time.Time
		expectedIDs    []int
	}{
		{expectedIDs: []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}},
		{repositoryID: 50, expectedIDs: []int{1, 2, 3, 5, 7, 8, 9, 10}},
		{state: "completed", expectedIDs: []int{7, 8, 10}},
		{term: "sub", expectedIDs: []int{1, 3, 5, 6, 7, 10}}, // searches root
		{term: "003", expectedIDs: []int{1, 3, 5}},           // searches commits
		{term: "333", expectedIDs: []int{1, 2, 3, 5}},        // searches commits and failure message
		{term: "tsc", expectedIDs: []int{2, 5, 7, 8, 10}},    // searches indexer
		{term: "QuEuEd", expectedIDs: []int{1, 3, 4, 9}},     // searches text status
		{term: "bAr", expectedIDs: []int{4, 6}},              // search repo names
		{visibleAtTip: true, expectedIDs: []int{2, 5, 7, 8}},
		{uploadedBefore: &t5, expectedIDs: []int{6, 7, 8, 9, 10}},
	}

	for _, testCase := range testCases {
		for lo := 0; lo < len(testCase.expectedIDs); lo++ {
			hi := lo + 3
			if hi > len(testCase.expectedIDs) {
				hi = len(testCase.expectedIDs)
			}

			name := fmt.Sprintf(
				"repositoryID=%d state=%s term=%s visibleAtTip=%v offset=%d",
				testCase.repositoryID,
				testCase.state,
				testCase.term,
				testCase.visibleAtTip,
				lo,
			)

			t.Run(name, func(t *testing.T) {
				uploads, totalCount, err := store.GetUploads(context.Background(), GetUploadsOptions{
					RepositoryID:   testCase.repositoryID,
					State:          testCase.state,
					Term:           testCase.term,
					VisibleAtTip:   testCase.visibleAtTip,
					UploadedBefore: testCase.uploadedBefore,
					Limit:          3,
					Offset:         lo,
				})
				if err != nil {
					t.Fatalf("unexpected error getting uploads for repo: %s", err)
				}
				if totalCount != len(testCase.expectedIDs) {
					t.Errorf("unexpected total count. want=%d have=%d", len(testCase.expectedIDs), totalCount)
				}

				var ids []int
				for _, upload := range uploads {
					ids = append(ids, upload.ID)
				}

				if diff := cmp.Diff(testCase.expectedIDs[lo:hi], ids); diff != "" {
					t.Errorf("unexpected upload ids at offset %d (-want +got):\n%s", lo, diff)
				}
			})
		}
	}
}

func TestQueueSize(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	store := testStore()

	insertUploads(t, dbconn.Global,
		Upload{ID: 1, State: "queued"},
		Upload{ID: 2, State: "errored"},
		Upload{ID: 3, State: "processing"},
		Upload{ID: 4, State: "completed"},
		Upload{ID: 5, State: "completed"},
		Upload{ID: 6, State: "queued"},
		Upload{ID: 7, State: "processing"},
		Upload{ID: 8, State: "completed"},
		Upload{ID: 9, State: "processing"},
		Upload{ID: 10, State: "queued"},
	)

	count, err := store.QueueSize(context.Background())
	if err != nil {
		t.Fatalf("unexpected error getting queue size: %s", err)
	}
	if count != 3 {
		t.Errorf("unexpected count. want=%d have=%d", 3, count)
	}
}

func TestInsertUploadUploading(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	store := testStore()

	insertRepo(t, dbconn.Global, 50, "")

	id, err := store.InsertUpload(context.Background(), Upload{
		Commit:       makeCommit(1),
		Root:         "sub/",
		State:        "uploading",
		RepositoryID: 50,
		Indexer:      "lsif-go",
		NumParts:     3,
	})
	if err != nil {
		t.Fatalf("unexpected error enqueueing upload: %s", err)
	}

	expected := Upload{
		ID:             id,
		Commit:         makeCommit(1),
		Root:           "sub/",
		VisibleAtTip:   false,
		UploadedAt:     time.Time{},
		State:          "uploading",
		FailureMessage: nil,
		StartedAt:      nil,
		FinishedAt:     nil,
		RepositoryID:   50,
		RepositoryName: "n-50",
		Indexer:        "lsif-go",
		NumParts:       3,
		UploadedParts:  []int{},
	}

	if upload, exists, err := store.GetUploadByID(context.Background(), id); err != nil {
		t.Fatalf("unexpected error getting upload: %s", err)
	} else if !exists {
		t.Fatal("expected record to exist")
	} else {
		// Update auto-generated timestamp
		expected.UploadedAt = upload.UploadedAt

		if diff := cmp.Diff(expected, upload); diff != "" {
			t.Errorf("unexpected upload (-want +got):\n%s", diff)
		}
	}
}

func TestInsertUploadQueued(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	store := testStore()

	insertRepo(t, dbconn.Global, 50, "")

	id, err := store.InsertUpload(context.Background(), Upload{
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

	rank := 1
	expected := Upload{
		ID:             id,
		Commit:         makeCommit(1),
		Root:           "sub/",
		VisibleAtTip:   false,
		UploadedAt:     time.Time{},
		State:          "queued",
		FailureMessage: nil,
		StartedAt:      nil,
		FinishedAt:     nil,
		RepositoryID:   50,
		RepositoryName: "n-50",
		Indexer:        "lsif-go",
		NumParts:       1,
		UploadedParts:  []int{0},
		Rank:           &rank,
	}

	if upload, exists, err := store.GetUploadByID(context.Background(), id); err != nil {
		t.Fatalf("unexpected error getting upload: %s", err)
	} else if !exists {
		t.Fatal("expected record to exist")
	} else {
		// Update auto-generated timestamp
		expected.UploadedAt = upload.UploadedAt

		if diff := cmp.Diff(expected, upload); diff != "" {
			t.Errorf("unexpected upload (-want +got):\n%s", diff)
		}
	}
}

func TestMarkQueued(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	store := testStore()

	insertUploads(t, dbconn.Global, Upload{ID: 1, State: "uploading"})

	uploadSize := 300
	if err := store.MarkQueued(context.Background(), 1, &uploadSize); err != nil {
		t.Fatalf("unexpected error marking upload as queued: %s", err)
	}

	if upload, exists, err := store.GetUploadByID(context.Background(), 1); err != nil {
		t.Fatalf("unexpected error getting upload: %s", err)
	} else if !exists {
		t.Fatal("expected record to exist")
	} else if upload.State != "queued" {
		t.Errorf("unexpected state. want=%q have=%q", "queued", upload.State)
	} else if upload.UploadSize == nil || *upload.UploadSize != 300 {
		t.Errorf("unexpected upload size. want=%v have=%v", 300, upload.UploadSize)
	}
}

func TestAddUploadPart(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	store := testStore()

	insertUploads(t, dbconn.Global, Upload{ID: 1, State: "uploading"})

	for _, part := range []int{1, 5, 2, 3, 2, 2, 1, 6} {
		if err := store.AddUploadPart(context.Background(), 1, part); err != nil {
			t.Fatalf("unexpected error adding upload part: %s", err)
		}
	}
	if upload, exists, err := store.GetUploadByID(context.Background(), 1); err != nil {
		t.Fatalf("unexpected error getting upload: %s", err)
	} else if !exists {
		t.Fatal("expected record to exist")
	} else {
		sort.Ints(upload.UploadedParts)
		if diff := cmp.Diff([]int{1, 2, 3, 5, 6}, upload.UploadedParts); diff != "" {
			t.Errorf("unexpected upload parts (-want +got):\n%s", diff)
		}
	}
}

func TestMarkComplete(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	store := testStore()

	insertUploads(t, dbconn.Global, Upload{ID: 1, State: "queued"})

	if err := store.MarkComplete(context.Background(), 1); err != nil {
		t.Fatalf("unexpected error marking upload as completed: %s", err)
	}

	if upload, exists, err := store.GetUploadByID(context.Background(), 1); err != nil {
		t.Fatalf("unexpected error getting upload: %s", err)
	} else if !exists {
		t.Fatal("expected record to exist")
	} else if upload.State != "completed" {
		t.Errorf("unexpected state. want=%q have=%q", "completed", upload.State)
	}
}

func TestMarkErrored(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	store := testStore()

	insertUploads(t, dbconn.Global, Upload{ID: 1, State: "queued"})

	if err := store.MarkErrored(context.Background(), 1, "oops"); err != nil {
		t.Fatalf("unexpected error marking upload as errored: %s", err)
	}

	if upload, exists, err := store.GetUploadByID(context.Background(), 1); err != nil {
		t.Fatalf("unexpected error getting upload: %s", err)
	} else if !exists {
		t.Fatal("expected record to exist")
	} else if upload.State != "errored" {
		t.Errorf("unexpected state. want=%q have=%q", "errored", upload.State)
	}
}

func TestDequeueConversionSuccess(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	store := testStore()

	// Add dequeueable upload
	insertUploads(t, dbconn.Global, Upload{ID: 1, State: "queued"})

	upload, tx, ok, err := store.Dequeue(context.Background(), 0)
	if err != nil {
		t.Fatalf("unexpected error dequeueing upload: %s", err)
	}
	if !ok {
		t.Fatalf("expected something to be dequeueable")
	}

	if upload.ID != 1 {
		t.Errorf("unexpected upload id. want=%d have=%d", 1, upload.ID)
	}
	if upload.State != "processing" {
		t.Errorf("unexpected state. want=%s have=%s", "processing", upload.State)
	}

	if state, _, err := basestore.ScanFirstString(dbconn.Global.Query("SELECT state FROM lsif_uploads WHERE id = 1")); err != nil {
		t.Errorf("unexpected error getting state: %s", err)
	} else if state != "processing" {
		t.Errorf("unexpected state outside of txn. want=%s have=%s", "processing", state)
	}

	if err := tx.MarkComplete(context.Background(), upload.ID); err != nil {
		t.Fatalf("unexpected error marking upload complete: %s", err)
	}
	_ = tx.Done(nil)

	if state, _, err := basestore.ScanFirstString(dbconn.Global.Query("SELECT state FROM lsif_uploads WHERE id = 1")); err != nil {
		t.Errorf("unexpected error getting state: %s", err)
	} else if state != "completed" {
		t.Errorf("unexpected state outside of txn. want=%s have=%s", "completed", state)
	}
}

func TestDequeueConversionError(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	store := testStore()

	// Add dequeueable upload
	insertUploads(t, dbconn.Global, Upload{ID: 1, State: "queued"})

	upload, tx, ok, err := store.Dequeue(context.Background(), 0)
	if err != nil {
		t.Fatalf("unexpected error dequeueing upload: %s", err)
	}
	if !ok {
		t.Fatalf("expected something to be dequeueable")
	}

	if upload.ID != 1 {
		t.Errorf("unexpected upload id. want=%d have=%d", 1, upload.ID)
	}
	if upload.State != "processing" {
		t.Errorf("unexpected state. want=%s have=%s", "processing", upload.State)
	}

	if state, _, err := basestore.ScanFirstString(dbconn.Global.Query("SELECT state FROM lsif_uploads WHERE id = 1")); err != nil {
		t.Errorf("unexpected error getting state: %s", err)
	} else if state != "processing" {
		t.Errorf("unexpected state outside of txn. want=%s have=%s", "processing", state)
	}

	if err := tx.MarkErrored(context.Background(), upload.ID, "test message"); err != nil {
		t.Fatalf("unexpected error marking upload complete: %s", err)
	}
	_ = tx.Done(nil)

	if state, _, err := basestore.ScanFirstString(dbconn.Global.Query("SELECT state FROM lsif_uploads WHERE id = 1")); err != nil {
		t.Errorf("unexpected error getting state: %s", err)
	} else if state != "errored" {
		t.Errorf("unexpected state outside of txn. want=%s have=%s", "errored", state)
	}

	if message, _, err := basestore.ScanFirstString(dbconn.Global.Query("SELECT failure_message FROM lsif_uploads WHERE id = 1")); err != nil {
		t.Errorf("unexpected error getting failure_message: %s", err)
	} else if message != "test message" {
		t.Errorf("unexpected failure message outside of txn. want=%s have=%s", "test message", message)
	}
}

func TestDequeueSkipsLocked(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	store := testStore()

	t1 := time.Unix(1587396557, 0).UTC()
	t2 := t1.Add(time.Minute)
	t3 := t2.Add(time.Minute)
	insertUploads(
		t,
		dbconn.Global,
		Upload{ID: 1, State: "queued", UploadedAt: t1},
		Upload{ID: 2, State: "processing", UploadedAt: t2},
		Upload{ID: 3, State: "queued", UploadedAt: t3},
	)

	tx1, err := dbconn.Global.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx1.Rollback() }()

	// Row lock upload 1 in a transaction which should be skipped by ResetStalled
	if _, err := tx1.Query(`SELECT * FROM lsif_uploads WHERE id = 1 FOR UPDATE`); err != nil {
		t.Fatal(err)
	}

	upload, tx2, ok, err := store.Dequeue(context.Background(), 0)
	if err != nil {
		t.Fatalf("unexpected error dequeueing upload: %s", err)
	}
	if !ok {
		t.Fatalf("expected something to be dequeueable")
	}
	defer func() { _ = tx2.Done(nil) }()

	if upload.ID != 3 {
		t.Errorf("unexpected upload id. want=%d have=%d", 3, upload.ID)
	}
	if upload.State != "processing" {
		t.Errorf("unexpected state. want=%s have=%s", "processing", upload.State)
	}
}

func TestDequeueSkipsDelayed(t *testing.T) {
	t.Skip()

	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	store := testStore()

	t1 := time.Unix(1587396557, 0).UTC()
	t2 := t1.Add(time.Minute)
	t3 := t2.Add(time.Minute)
	insertUploads(
		t,
		dbconn.Global,
		Upload{ID: 1, State: "queued", UploadedAt: t1, ProcessAfter: &t2},
		Upload{ID: 2, State: "processing", UploadedAt: t2},
		Upload{ID: 3, State: "queued", UploadedAt: t3},
	)

	tx1, err := dbconn.Global.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx1.Rollback() }()

	upload, tx2, ok, err := store.Dequeue(context.Background(), 0)
	if err != nil {
		t.Fatalf("unexpected error dequeueing upload: %s", err)
	}
	if !ok {
		t.Fatalf("expected something to be dequeueable")
	}
	defer func() { _ = tx2.Done(nil) }()

	if upload.ID != 3 {
		t.Errorf("unexpected upload id. want=%d have=%d", 3, upload.ID)
	}
	if upload.State != "processing" {
		t.Errorf("unexpected state. want=%s have=%s", "processing", upload.State)
	}
}

func TestDequeueSkipsOversizedUploads(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	store := testStore()

	t1 := time.Unix(1587396557, 0).UTC()
	t2 := t1.Add(time.Minute)
	t3 := t2.Add(time.Minute)

	s1 := int64(90)
	s2 := int64(45)
	s3 := int64(50)

	insertUploads(
		t,
		dbconn.Global,
		Upload{ID: 1, State: "queued", UploadedAt: t1, UploadSize: &s1},
		Upload{ID: 2, State: "queued", UploadedAt: t2, UploadSize: &s2},
		Upload{ID: 3, State: "queued", UploadedAt: t3, UploadSize: &s3},
	)

	tx1, err := dbconn.Global.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx1.Rollback() }()

	upload, tx2, ok, err := store.Dequeue(context.Background(), 50)
	if err != nil {
		t.Fatalf("unexpected error dequeueing upload: %s", err)
	}
	if !ok {
		t.Fatalf("expected something to be dequeueable")
	}
	defer func() { _ = tx2.Done(nil) }()

	if upload.ID != 2 {
		t.Errorf("unexpected upload id. want=%d have=%d", 2, upload.ID)
	}
	if upload.State != "processing" {
		t.Errorf("unexpected state. want=%s have=%s", "processing", upload.State)
	}
}

func TestDequeueEmpty(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	store := testStore()

	_, tx, ok, err := store.Dequeue(context.Background(), 0)
	if err != nil {
		t.Fatalf("unexpected error dequeueing upload: %s", err)
	}
	if ok {
		_ = tx.Done(nil)
		t.Fatalf("unexpected dequeue")
	}
}

func TestRequeue(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	store := testStore()

	insertUploads(t, dbconn.Global, Upload{ID: 1, State: "processing"})

	after := time.Unix(1587396557, 0).UTC().Add(time.Hour)

	if err := store.Requeue(context.Background(), 1, after); err != nil {
		t.Fatalf("unexpected error requeueing index: %s", err)
	}

	if upload, exists, err := store.GetUploadByID(context.Background(), 1); err != nil {
		t.Fatalf("unexpected error getting index: %s", err)
	} else if !exists {
		t.Fatal("expected record to exist")
	} else if upload.State != "queued" {
		t.Errorf("unexpected state. want=%q have=%q", "queued", upload.State)
	} else if upload.ProcessAfter == nil || *upload.ProcessAfter != after {
		t.Errorf("unexpected process after. want=%s have=%s", after, upload.ProcessAfter)
	}
}

func TestGetStates(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	store := testStore()

	insertUploads(t, dbconn.Global,
		Upload{ID: 1, State: "queued"},
		Upload{ID: 2},
		Upload{ID: 3, State: "processing"},
		Upload{ID: 4, State: "errored"},
	)

	expected := map[int]string{
		1: "queued",
		2: "completed",
		4: "errored",
	}

	if states, err := store.GetStates(context.Background(), []int{1, 2, 4, 6}); err != nil {
		t.Fatalf("unexpected error getting states: %s", err)
	} else if diff := cmp.Diff(expected, states); diff != "" {
		t.Errorf("unexpected upload states (-want +got):\n%s", diff)
	}
}

func TestDeleteUploadByID(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	store := testStore()

	insertUploads(t, dbconn.Global,
		Upload{ID: 1, RepositoryID: 50},
	)

	if found, err := store.DeleteUploadByID(context.Background(), 1); err != nil {
		t.Fatalf("unexpected error deleting upload: %s", err)
	} else if !found {
		t.Fatalf("expected record to exist")
	}

	// Ensure record was deleted
	if states, err := store.GetStates(context.Background(), []int{1}); err != nil {
		t.Fatalf("unexpected error getting states: %s", err)
	} else if diff := cmp.Diff(map[int]string{1: "deleted"}, states); diff != "" {
		t.Errorf("unexpected dump (-want +got):\n%s", diff)
	}

	repositoryIDs, err := store.DirtyRepositories(context.Background())
	if err != nil {
		t.Fatalf("unexpected error listing dirty repositories: %s", err)
	}

	var keys []int
	for repositoryID := range repositoryIDs {
		keys = append(keys, repositoryID)
	}
	sort.Ints(keys)

	if len(keys) != 1 || keys[0] != 50 {
		t.Errorf("expected repository to be marked dirty")
	}
}

func TestDeleteUploadByIDMissingRow(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	store := testStore()

	if found, err := store.DeleteUploadByID(context.Background(), 1); err != nil {
		t.Fatalf("unexpected error deleting upload: %s", err)
	} else if found {
		t.Fatalf("unexpected record")
	}
}

func TestDeleteUploadsWithoutRepository(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	store := testStore()

	var uploads []Upload
	for i := 0; i < 25; i++ {
		for j := 0; j < 10+i; j++ {
			uploads = append(uploads, Upload{ID: len(uploads) + 1, RepositoryID: 50 + i})
		}
	}
	insertUploads(t, dbconn.Global, uploads...)

	t1 := time.Unix(1587396557, 0).UTC()
	t2 := t1.Add(-DeletedRepositoryGracePeriod + time.Minute)
	t3 := t1.Add(-DeletedRepositoryGracePeriod - time.Minute)

	deletions := map[int]time.Time{
		52: t2, 54: t2, 56: t2, // deleted too recently
		61: t3, 63: t3, 65: t3, // deleted
	}

	for repositoryID, deletedAt := range deletions {
		query := sqlf.Sprintf(`UPDATE repo SET deleted_at=%s WHERE id=%s`, deletedAt, repositoryID)

		if _, err := dbconn.Global.Query(query.Query(sqlf.PostgresBindVar), query.Args()...); err != nil {
			t.Fatalf("Failed to update repository: %s", err)
		}
	}

	deletedCounts, err := store.DeleteUploadsWithoutRepository(context.Background(), t1)
	if err != nil {
		t.Fatalf("unexpected error deleting uploads: %s", err)
	}

	expected := map[int]int{
		61: 21,
		63: 23,
		65: 25,
	}
	if diff := cmp.Diff(expected, deletedCounts); diff != "" {
		t.Errorf("unexpected deletedCounts (-want +got):\n%s", diff)
	}

	var uploadIDs []int
	for i := range uploads {
		uploadIDs = append(uploadIDs, i+1)
	}

	// Ensure records were deleted
	if states, err := store.GetStates(context.Background(), uploadIDs); err != nil {
		t.Fatalf("unexpected error getting states: %s", err)
	} else {
		deletedStates := 0
		for _, state := range states {
			if state == "deleted" {
				deletedStates++
			}
		}

		expected := 0
		for _, deletedCount := range deletedCounts {
			expected += deletedCount
		}

		if deletedStates != expected {
			t.Errorf("unexpected number of deleted records. want=%d have=%d", expected, deletedStates)
		}
	}
}

func TestHardDeleteUploadByID(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	store := testStore()

	insertUploads(t, dbconn.Global, Upload{ID: 1, State: "deleted"})

	if err := store.HardDeleteUploadByID(context.Background(), 1); err != nil {
		t.Fatalf("unexpected error deleting upload: %s", err)
	}

	// Ensure records were deleted
	if states, err := store.GetStates(context.Background(), []int{1}); err != nil {
		t.Fatalf("unexpected error getting states: %s", err)
	} else if len(states) != 0 {
		t.Fatalf("unexpected record")
	}
}

func TestResetStalled(t *testing.T) {
	t.Skip()

	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	store := testStore()

	now := time.Unix(1587396557, 0).UTC()
	t1 := now.Add(-time.Second * 6) // old
	t2 := now.Add(-time.Second * 2) // new enough
	t3 := now.Add(-time.Second * 3) // new enough
	t4 := now.Add(-time.Second * 8) // old
	t5 := now.Add(-time.Second * 8) // old

	insertUploads(t, dbconn.Global,
		Upload{ID: 1, State: "processing", StartedAt: &t1, NumResets: 1},
		Upload{ID: 2, State: "processing", StartedAt: &t2},
		Upload{ID: 3, State: "processing", StartedAt: &t3},
		Upload{ID: 4, State: "processing", StartedAt: &t4},
		Upload{ID: 5, State: "processing", StartedAt: &t5},
		Upload{ID: 6, State: "processing", StartedAt: &t1, NumResets: UploadMaxNumResets},
		Upload{ID: 7, State: "processing", StartedAt: &t4, NumResets: UploadMaxNumResets},
	)

	tx, err := dbconn.Global.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback() }()

	// Row lock upload 5 in a transaction which should be skipped by ResetStalled
	if _, err := tx.Query(`SELECT * FROM lsif_uploads WHERE id = 5 FOR UPDATE`); err != nil {
		t.Fatal(err)
	}

	resetIDs, erroredIDs, err := store.ResetStalled(context.Background(), now)
	if err != nil {
		t.Fatalf("unexpected error resetting stalled uploads: %s", err)
	}
	sort.Ints(resetIDs)
	sort.Ints(erroredIDs)

	expectedReset := []int{1, 4}
	if diff := cmp.Diff(expectedReset, resetIDs); diff != "" {
		t.Errorf("unexpected reset ids (-want +got):\n%s", diff)
	}

	expectedErrored := []int{6, 7}
	if diff := cmp.Diff(expectedErrored, erroredIDs); diff != "" {
		t.Errorf("unexpected errored ids (-want +got):\n%s", diff)
	}

	upload, _, err := store.GetUploadByID(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error getting upload: %s", err)
	}
	if upload.NumResets != 2 {
		t.Errorf("unexpected num resets. want=%d have=%d", 2, upload.NumResets)
	}
}
