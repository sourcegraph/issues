package store

import (
	"context"
	"fmt"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/sourcegraph/sourcegraph/internal/db/dbconn"
	"github.com/sourcegraph/sourcegraph/internal/db/dbtesting"
)

func TestHasRepository(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	store := testStore()

	testCases := []struct {
		repositoryID int
		exists       bool
	}{
		{50, true},
		{51, false},
		{52, false},
	}

	insertUploads(t, dbconn.Global, Upload{ID: 1, RepositoryID: 50})
	insertUploads(t, dbconn.Global, Upload{ID: 2, RepositoryID: 51, State: "deleted"})

	for _, testCase := range testCases {
		name := fmt.Sprintf("repositoryID=%d", testCase.repositoryID)

		t.Run(name, func(t *testing.T) {
			exists, err := store.HasRepository(context.Background(), testCase.repositoryID)
			if err != nil {
				t.Fatalf("unexpected error checking if repository exists: %s", err)
			}
			if exists != testCase.exists {
				t.Errorf("unexpected exists. want=%v have=%v", testCase.exists, exists)
			}
		})
	}
}

func TestHasCommit(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	store := testStore()

	testCases := []struct {
		repositoryID int
		commit       string
		exists       bool
	}{
		{50, makeCommit(1), true},
		{50, makeCommit(2), false},
		{51, makeCommit(1), false},
	}

	insertNearestUploads(t, dbconn.Global, 50, map[string][]UploadMeta{makeCommit(1): {{UploadID: 42, Distance: 1}}})
	insertNearestUploads(t, dbconn.Global, 51, map[string][]UploadMeta{makeCommit(2): {{UploadID: 43, Distance: 2}}})

	for _, testCase := range testCases {
		name := fmt.Sprintf("repositoryID=%d commit=%s", testCase.repositoryID, testCase.commit)

		t.Run(name, func(t *testing.T) {
			exists, err := store.HasCommit(context.Background(), testCase.repositoryID, testCase.commit)
			if err != nil {
				t.Fatalf("unexpected error checking if commit exists: %s", err)
			}
			if exists != testCase.exists {
				t.Errorf("unexpected exists. want=%v have=%v", testCase.exists, exists)
			}
		})
	}
}

func TestMarkRepositoryAsDirty(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	store := testStore()

	for _, repositoryID := range []int{50, 51, 52, 51, 52} {
		if err := store.MarkRepositoryAsDirty(context.Background(), repositoryID); err != nil {
			t.Errorf("unexpected error marking repository as dirty: %s", err)
		}
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

	if diff := cmp.Diff([]int{50, 51, 52}, keys); diff != "" {
		t.Errorf("unexpected repository ids (-want +got):\n%s", diff)
	}
}

func TestCalculateVisibleUploads(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	store := testStore()

	// This database has the following commit graph:
	//
	// [1] --+--- 2 --------+--5 -- 6 --+-- [7]
	//       |              |           |
	//       +-- [3] -- 4 --+           +--- 8

	uploads := []Upload{
		{ID: 1, Commit: makeCommit(1)},
		{ID: 2, Commit: makeCommit(3)},
		{ID: 3, Commit: makeCommit(7)},
	}
	insertUploads(t, dbconn.Global, uploads...)

	graph := map[string][]string{
		makeCommit(1): {},
		makeCommit(2): {makeCommit(1)},
		makeCommit(3): {makeCommit(1)},
		makeCommit(4): {makeCommit(3)},
		makeCommit(5): {makeCommit(2), makeCommit(4)},
		makeCommit(6): {makeCommit(5)},
		makeCommit(7): {makeCommit(6)},
		makeCommit(8): {makeCommit(6)},
	}

	if err := store.CalculateVisibleUploads(context.Background(), 50, graph, makeCommit(8), 0); err != nil {
		t.Fatalf("unexpected error while calculating visible uploads: %s", err)
	}

	expectedVisibleUploads := map[string][]UploadMeta{
		makeCommit(1): {{UploadID: 1, Distance: 0}},
		makeCommit(2): {{UploadID: 1, Distance: 1}},
		makeCommit(3): {{UploadID: 2, Distance: 0}},
		makeCommit(4): {{UploadID: 2, Distance: 1}},
		makeCommit(5): {{UploadID: 1, Distance: 2}},
		makeCommit(6): {{UploadID: 3, Distance: 1}},
		makeCommit(7): {{UploadID: 3, Distance: 0}},
		makeCommit(8): {{UploadID: 1, Distance: 4}},
	}
	if diff := cmp.Diff(expectedVisibleUploads, getVisibleUploads(t, dbconn.Global, 50), UploadMetaComparer); diff != "" {
		t.Errorf("unexpected visible uploads (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff([]int{1}, getUploadsVisibleAtTip(t, dbconn.Global, 50)); diff != "" {
		t.Errorf("unexpected uploads visible at tip (-want +got):\n%s", diff)
	}
}

func TestCalculateVisibleUploadsAlternateCommitGraph(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	store := testStore()

	// This database has the following commit graph:
	//
	// 1 --+-- [2] ---- 3
	//     |
	//     +--- 4 --+-- 5 -- 6
	//              |
	//              +-- 7 -- 8

	uploads := []Upload{
		{ID: 1, Commit: makeCommit(2)},
	}
	insertUploads(t, dbconn.Global, uploads...)

	graph := map[string][]string{
		makeCommit(1): {},
		makeCommit(2): {makeCommit(1)},
		makeCommit(3): {makeCommit(2)},
		makeCommit(4): {makeCommit(1)},
		makeCommit(5): {makeCommit(4)},
		makeCommit(6): {makeCommit(5)},
		makeCommit(7): {makeCommit(4)},
		makeCommit(8): {makeCommit(7)},
	}

	if err := store.CalculateVisibleUploads(context.Background(), 50, graph, makeCommit(3), 0); err != nil {
		t.Fatalf("unexpected error while calculating visible uploads: %s", err)
	}

	expectedVisibleUploads := map[string][]UploadMeta{
		makeCommit(1): {{UploadID: 1, Distance: 1}},
		makeCommit(2): {{UploadID: 1, Distance: 0}},
		makeCommit(3): {{UploadID: 1, Distance: 1}},
	}
	if diff := cmp.Diff(expectedVisibleUploads, getVisibleUploads(t, dbconn.Global, 50), UploadMetaComparer); diff != "" {
		t.Errorf("unexpected visible uploads (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff([]int{1}, getUploadsVisibleAtTip(t, dbconn.Global, 50)); diff != "" {
		t.Errorf("unexpected uploads visible at tip (-want +got):\n%s", diff)
	}
}

func TestCalculateVisibleUploadsDistinctRoots(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	store := testStore()

	// This database has the following commit graph:
	//
	// 1 --+-- [2]

	uploads := []Upload{
		{ID: 1, Commit: makeCommit(2), Root: "root1/"},
		{ID: 2, Commit: makeCommit(2), Root: "root2/"},
	}
	insertUploads(t, dbconn.Global, uploads...)

	graph := map[string][]string{
		makeCommit(1): {},
		makeCommit(2): {makeCommit(1)},
	}

	if err := store.CalculateVisibleUploads(context.Background(), 50, graph, makeCommit(2), 0); err != nil {
		t.Fatalf("unexpected error while calculating visible uploads: %s", err)
	}

	expectedVisibleUploads := map[string][]UploadMeta{
		makeCommit(1): {{UploadID: 1, Distance: 1}, {UploadID: 2, Distance: 1}},
		makeCommit(2): {{UploadID: 1, Distance: 0}, {UploadID: 2, Distance: 0}},
	}
	if diff := cmp.Diff(expectedVisibleUploads, getVisibleUploads(t, dbconn.Global, 50), UploadMetaComparer); diff != "" {
		t.Errorf("unexpected visible uploads (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff([]int{1, 2}, getUploadsVisibleAtTip(t, dbconn.Global, 50)); diff != "" {
		t.Errorf("unexpected uploads visible at tip (-want +got):\n%s", diff)
	}
}

func TestCalculateVisibleUploadsOverlappingRoots(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	store := testStore()

	// This database has the following commit graph:
	//
	// 1 -- 2 --+-- 3 --+-- 5 -- 6
	//          |       |
	//          +-- 4 --+
	//
	// With the following LSIF dumps:
	//
	// | UploadID | Commit | Root    | Indexer |
	// | -------- + ------ + ------- + ------- |
	// | 1        | 1      | root3/  | lsif-go |
	// | 2        | 1      | root4/  | lsif-py |
	// | 3        | 2      | root1/  | lsif-go |
	// | 4        | 2      | root2/  | lsif-go |
	// | 5        | 2      |         | lsif-py | (overwrites root4/ at commit 1)
	// | 6        | 3      | root1/  | lsif-go | (overwrites root1/ at commit 2)
	// | 7        | 4      |         | lsif-py | (overwrites (root) at commit 2)
	// | 8        | 5      | root2/  | lsif-go | (overwrites root2/ at commit 2)
	// | 9        | 6      | root1/  | lsif-go | (overwrites root1/ at commit 2)

	uploads := []Upload{
		{ID: 1, Commit: makeCommit(1), Indexer: "lsif-go", Root: "root3/"},
		{ID: 2, Commit: makeCommit(1), Indexer: "lsif-py", Root: "root4/"},
		{ID: 3, Commit: makeCommit(2), Indexer: "lsif-go", Root: "root1/"},
		{ID: 4, Commit: makeCommit(2), Indexer: "lsif-go", Root: "root2/"},
		{ID: 5, Commit: makeCommit(2), Indexer: "lsif-py", Root: ""},
		{ID: 6, Commit: makeCommit(3), Indexer: "lsif-go", Root: "root1/"},
		{ID: 7, Commit: makeCommit(4), Indexer: "lsif-py", Root: ""},
		{ID: 8, Commit: makeCommit(5), Indexer: "lsif-go", Root: "root2/"},
		{ID: 9, Commit: makeCommit(6), Indexer: "lsif-go", Root: "root1/"},
	}
	insertUploads(t, dbconn.Global, uploads...)

	graph := map[string][]string{
		makeCommit(1): {},
		makeCommit(2): {makeCommit(1)},
		makeCommit(3): {makeCommit(2)},
		makeCommit(4): {makeCommit(2)},
		makeCommit(5): {makeCommit(3), makeCommit(4)},
		makeCommit(6): {makeCommit(5)},
	}

	if err := store.CalculateVisibleUploads(context.Background(), 50, graph, makeCommit(6), 0); err != nil {
		t.Fatalf("unexpected error while calculating visible uploads: %s", err)
	}

	expectedVisibleUploads := map[string][]UploadMeta{
		makeCommit(1): {{UploadID: 1, Distance: 0}, {UploadID: 2, Distance: 0}, {UploadID: 3, Distance: 1}, {UploadID: 4, Distance: 1}, {UploadID: 5, Distance: 1}},
		makeCommit(2): {{UploadID: 1, Distance: 1}, {UploadID: 2, Distance: 1}, {UploadID: 3, Distance: 0}, {UploadID: 4, Distance: 0}, {UploadID: 5, Distance: 0}},
		makeCommit(3): {{UploadID: 1, Distance: 2}, {UploadID: 2, Distance: 2}, {UploadID: 4, Distance: 1}, {UploadID: 5, Distance: 1}, {UploadID: 6, Distance: 0}},
		makeCommit(4): {{UploadID: 1, Distance: 2}, {UploadID: 2, Distance: 2}, {UploadID: 3, Distance: 1}, {UploadID: 4, Distance: 1}, {UploadID: 7, Distance: 0}},
		makeCommit(5): {{UploadID: 1, Distance: 3}, {UploadID: 2, Distance: 3}, {UploadID: 6, Distance: 1}, {UploadID: 7, Distance: 1}, {UploadID: 8, Distance: 0}},
		makeCommit(6): {{UploadID: 1, Distance: 4}, {UploadID: 2, Distance: 4}, {UploadID: 7, Distance: 2}, {UploadID: 8, Distance: 1}, {UploadID: 9, Distance: 0}},
	}
	if diff := cmp.Diff(expectedVisibleUploads, getVisibleUploads(t, dbconn.Global, 50), UploadMetaComparer); diff != "" {
		t.Errorf("unexpected visible uploads (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff([]int{1, 2, 7, 8, 9}, getUploadsVisibleAtTip(t, dbconn.Global, 50)); diff != "" {
		t.Errorf("unexpected uploads visible at tip (-want +got):\n%s", diff)
	}
}

func TestCalculateVisibleUploadsIndexerName(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	store := testStore()

	// This database has the following commit graph:
	//
	// [1] --+-- [2] --+-- [3] --+-- [4] --+-- 5

	uploads := []Upload{
		{ID: 1, Commit: makeCommit(1), Root: "root1/", Indexer: "idx1"},
		{ID: 2, Commit: makeCommit(2), Root: "root2/", Indexer: "idx1"},
		{ID: 3, Commit: makeCommit(3), Root: "root3/", Indexer: "idx1"},
		{ID: 4, Commit: makeCommit(4), Root: "root4/", Indexer: "idx1"},
		{ID: 5, Commit: makeCommit(1), Root: "root1/", Indexer: "idx2"},
		{ID: 6, Commit: makeCommit(2), Root: "root2/", Indexer: "idx2"},
		{ID: 7, Commit: makeCommit(3), Root: "root3/", Indexer: "idx2"},
		{ID: 8, Commit: makeCommit(4), Root: "root4/", Indexer: "idx2"},
	}
	insertUploads(t, dbconn.Global, uploads...)

	graph := map[string][]string{
		makeCommit(1): {},
		makeCommit(2): {makeCommit(1)},
		makeCommit(3): {makeCommit(2)},
		makeCommit(4): {makeCommit(3)},
		makeCommit(5): {makeCommit(4)},
	}

	if err := store.CalculateVisibleUploads(context.Background(), 50, graph, makeCommit(5), 0); err != nil {
		t.Fatalf("unexpected error while calculating visible uploads: %s", err)
	}

	expectedVisibleUploads := map[string][]UploadMeta{
		makeCommit(1): {
			{UploadID: 1, Distance: 0}, {UploadID: 2, Distance: 1}, {UploadID: 3, Distance: 2}, {UploadID: 4, Distance: 3},
			{UploadID: 5, Distance: 0}, {UploadID: 6, Distance: 1}, {UploadID: 7, Distance: 2}, {UploadID: 8, Distance: 3},
		},
		makeCommit(2): {
			{UploadID: 1, Distance: 1}, {UploadID: 2, Distance: 0}, {UploadID: 3, Distance: 1}, {UploadID: 4, Distance: 2},
			{UploadID: 5, Distance: 1}, {UploadID: 6, Distance: 0}, {UploadID: 7, Distance: 1}, {UploadID: 8, Distance: 2},
		},
		makeCommit(3): {
			{UploadID: 1, Distance: 2}, {UploadID: 2, Distance: 1}, {UploadID: 3, Distance: 0}, {UploadID: 4, Distance: 1},
			{UploadID: 5, Distance: 2}, {UploadID: 6, Distance: 1}, {UploadID: 7, Distance: 0}, {UploadID: 8, Distance: 1},
		},
		makeCommit(4): {
			{UploadID: 1, Distance: 3}, {UploadID: 2, Distance: 2}, {UploadID: 3, Distance: 1}, {UploadID: 4, Distance: 0},
			{UploadID: 5, Distance: 3}, {UploadID: 6, Distance: 2}, {UploadID: 7, Distance: 1}, {UploadID: 8, Distance: 0},
		},
		makeCommit(5): {
			{UploadID: 1, Distance: 4}, {UploadID: 2, Distance: 3}, {UploadID: 3, Distance: 2}, {UploadID: 4, Distance: 1},
			{UploadID: 5, Distance: 4}, {UploadID: 6, Distance: 3}, {UploadID: 7, Distance: 2}, {UploadID: 8, Distance: 1},
		},
	}
	if diff := cmp.Diff(expectedVisibleUploads, getVisibleUploads(t, dbconn.Global, 50), UploadMetaComparer); diff != "" {
		t.Errorf("unexpected visible uploads (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff([]int{1, 2, 3, 4, 5, 6, 7, 8}, getUploadsVisibleAtTip(t, dbconn.Global, 50)); diff != "" {
		t.Errorf("unexpected uploads visible at tip (-want +got):\n%s", diff)
	}
}

func TestCalculateVisibleUploadsResetsDirtyFlag(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	store := testStore()

	uploads := []Upload{
		{ID: 1, Commit: makeCommit(1)},
		{ID: 2, Commit: makeCommit(2)},
		{ID: 3, Commit: makeCommit(3)},
	}
	insertUploads(t, dbconn.Global, uploads...)

	graph := map[string][]string{
		makeCommit(1): {},
		makeCommit(2): {makeCommit(1)},
		makeCommit(3): {makeCommit(2)},
	}

	for i := 0; i < 3; i++ {
		// Set dirty token to 3
		if err := store.MarkRepositoryAsDirty(context.Background(), 50); err != nil {
			t.Fatalf("unexpected error marking repository as dirty: %s", err)
		}
	}

	// Non-latest dirty token - should not clear flag
	if err := store.CalculateVisibleUploads(context.Background(), 50, graph, makeCommit(3), 2); err != nil {
		t.Fatalf("unexpected error while calculating visible uploads: %s", err)
	}

	repositoryIDs, err := store.DirtyRepositories(context.Background())
	if err != nil {
		t.Fatalf("unexpected error listing dirty repositories: %s", err)
	}

	if len(repositoryIDs) == 0 {
		t.Errorf("did not expect repository to be unmarked")
	}

	// Latest dirty token - should clear flag
	if err := store.CalculateVisibleUploads(context.Background(), 50, graph, makeCommit(3), 3); err != nil {
		t.Fatalf("unexpected error while calculating visible uploads: %s", err)
	}

	repositoryIDs, err = store.DirtyRepositories(context.Background())
	if err != nil {
		t.Fatalf("unexpected error listing dirty repositories: %s", err)
	}

	if len(repositoryIDs) != 0 {
		t.Errorf("expected repository to be unmarked")
	}
}
