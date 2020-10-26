package store

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/sourcegraph/sourcegraph/internal/db/dbconn"
	"github.com/sourcegraph/sourcegraph/internal/db/dbtesting"
)

func TestGetDumpByID(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	store := testStore()

	// Dump does not exist initially
	if _, exists, err := store.GetDumpByID(context.Background(), 1); err != nil {
		t.Fatalf("unexpected error getting dump: %s", err)
	} else if exists {
		t.Fatal("unexpected record")
	}

	uploadedAt := time.Unix(1587396557, 0).UTC()
	startedAt := uploadedAt.Add(time.Minute)
	finishedAt := uploadedAt.Add(time.Minute * 2)
	expected := Dump{
		ID:             1,
		Commit:         makeCommit(1),
		Root:           "sub/",
		VisibleAtTip:   true,
		UploadedAt:     uploadedAt,
		State:          "completed",
		FailureMessage: nil,
		StartedAt:      &startedAt,
		FinishedAt:     &finishedAt,
		RepositoryID:   50,
		RepositoryName: "n-50",
		Indexer:        "lsif-go",
	}

	insertUploads(t, dbconn.Global, Upload{
		ID:             expected.ID,
		Commit:         expected.Commit,
		Root:           expected.Root,
		UploadedAt:     expected.UploadedAt,
		State:          expected.State,
		FailureMessage: expected.FailureMessage,
		StartedAt:      expected.StartedAt,
		FinishedAt:     expected.FinishedAt,
		ProcessAfter:   expected.ProcessAfter,
		NumResets:      expected.NumResets,
		NumFailures:    expected.NumFailures,
		RepositoryID:   expected.RepositoryID,
		RepositoryName: expected.RepositoryName,
		Indexer:        expected.Indexer,
	})
	insertVisibleAtTip(t, dbconn.Global, 50, 1)

	if dump, exists, err := store.GetDumpByID(context.Background(), 1); err != nil {
		t.Fatalf("unexpected error getting dump: %s", err)
	} else if !exists {
		t.Fatal("expected record to exist")
	} else if diff := cmp.Diff(expected, dump); diff != "" {
		t.Errorf("unexpected dump (-want +got):\n%s", diff)
	}
}

func TestFindClosestDumps(t *testing.T) {
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

	visibleUploads, err := calculateVisibleUploads(graph, toUploadMeta(uploads))
	if err != nil {
		t.Fatalf("unexpected error while calculating visible uploads: %s", err)
	}
	insertNearestUploads(t, dbconn.Global, 50, visibleUploads)

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
	if diff := cmp.Diff(expectedVisibleUploads, normalizeVisibleUploads(visibleUploads), UploadMetaComparer); diff != "" {
		t.Errorf("unexpected visible uploads (-want +got):\n%s", diff)
	}

	testFindClosestDumps(t, store, []FindClosestDumpsTestCase{
		{commit: makeCommit(1), file: "file.ts", rootMustEnclosePath: true, graph: graph, anyOfIDs: []int{1}},
		{commit: makeCommit(2), file: "file.ts", rootMustEnclosePath: true, graph: graph, anyOfIDs: []int{1}},
		{commit: makeCommit(3), file: "file.ts", rootMustEnclosePath: true, graph: graph, anyOfIDs: []int{2}},
		{commit: makeCommit(4), file: "file.ts", rootMustEnclosePath: true, graph: graph, anyOfIDs: []int{2}},
		{commit: makeCommit(6), file: "file.ts", rootMustEnclosePath: true, graph: graph, anyOfIDs: []int{3}},
		{commit: makeCommit(7), file: "file.ts", rootMustEnclosePath: true, graph: graph, anyOfIDs: []int{3}},
		{commit: makeCommit(5), file: "file.ts", rootMustEnclosePath: true, graph: graph, anyOfIDs: []int{1, 2, 3}},
		{commit: makeCommit(8), file: "file.ts", rootMustEnclosePath: true, graph: graph, anyOfIDs: []int{1, 2}},
	})
}

func TestFindClosestDumpsAlternateCommitGraph(t *testing.T) {
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

	visibleUploads, err := calculateVisibleUploads(graph, toUploadMeta(uploads))
	if err != nil {
		t.Fatalf("unexpected error while calculating visible uploads: %s", err)
	}
	insertNearestUploads(t, dbconn.Global, 50, visibleUploads)

	expectedVisibleUploads := map[string][]UploadMeta{
		makeCommit(1): {{UploadID: 1, Distance: 1}},
		makeCommit(2): {{UploadID: 1, Distance: 0}},
		makeCommit(3): {{UploadID: 1, Distance: 1}},
		makeCommit(4): nil,
		makeCommit(5): nil,
		makeCommit(6): nil,
		makeCommit(7): nil,
		makeCommit(8): nil,
	}
	normalizeVisibleUploads(visibleUploads)
	if diff := cmp.Diff(expectedVisibleUploads, normalizeVisibleUploads(visibleUploads), UploadMetaComparer); diff != "" {
		t.Errorf("unexpected visible uploads (-want +got):\n%s", diff)
	}

	testFindClosestDumps(t, store, []FindClosestDumpsTestCase{
		{commit: makeCommit(1), graph: graph, allOfIDs: []int{1}},
		{commit: makeCommit(2), graph: graph, allOfIDs: []int{1}},
		{commit: makeCommit(3), graph: graph, allOfIDs: []int{1}},
		{commit: makeCommit(4), graph: graph},
		{commit: makeCommit(6), graph: graph},
		{commit: makeCommit(7), graph: graph},
		{commit: makeCommit(5), graph: graph},
		{commit: makeCommit(8), graph: graph},
	})
}

func TestFindClosestDumpsDistinctRoots(t *testing.T) {
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

	visibleUploads, err := calculateVisibleUploads(graph, toUploadMeta(uploads))
	if err != nil {
		t.Fatalf("unexpected error while calculating visible uploads: %s", err)
	}
	insertNearestUploads(t, dbconn.Global, 50, visibleUploads)

	expectedVisibleUploads := map[string][]UploadMeta{
		makeCommit(1): {{UploadID: 1, Distance: 1}, {UploadID: 2, Distance: 1}},
		makeCommit(2): {{UploadID: 1, Distance: 0}, {UploadID: 2, Distance: 0}},
	}
	if diff := cmp.Diff(expectedVisibleUploads, normalizeVisibleUploads(visibleUploads), UploadMetaComparer); diff != "" {
		t.Errorf("unexpected visible uploads (-want +got):\n%s", diff)
	}

	testFindClosestDumps(t, store, []FindClosestDumpsTestCase{
		{commit: makeCommit(1), file: "blah", rootMustEnclosePath: true, graph: graph},
		{commit: makeCommit(2), file: "root1/file.ts", rootMustEnclosePath: true, graph: graph, allOfIDs: []int{1}},
		{commit: makeCommit(1), file: "root2/file.ts", rootMustEnclosePath: true, graph: graph, allOfIDs: []int{2}},
		{commit: makeCommit(2), file: "root2/file.ts", rootMustEnclosePath: true, graph: graph, allOfIDs: []int{2}},
		{commit: makeCommit(1), file: "root3/file.ts", rootMustEnclosePath: true, graph: graph},
	})
}

func TestFindClosestDumpsOverlappingRoots(t *testing.T) {
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

	visibleUploads, err := calculateVisibleUploads(graph, toUploadMeta(uploads))
	if err != nil {
		t.Fatalf("unexpected error while calculating visible uploads: %s", err)
	}
	insertNearestUploads(t, dbconn.Global, 50, visibleUploads)

	expectedVisibleUploads := map[string][]UploadMeta{
		makeCommit(1): {{UploadID: 1, Distance: 0}, {UploadID: 2, Distance: 0}, {UploadID: 3, Distance: 1}, {UploadID: 4, Distance: 1}, {UploadID: 5, Distance: 1}},
		makeCommit(2): {{UploadID: 1, Distance: 1}, {UploadID: 2, Distance: 1}, {UploadID: 3, Distance: 0}, {UploadID: 4, Distance: 0}, {UploadID: 5, Distance: 0}},
		makeCommit(3): {{UploadID: 1, Distance: 2}, {UploadID: 2, Distance: 2}, {UploadID: 4, Distance: 1}, {UploadID: 5, Distance: 1}, {UploadID: 6, Distance: 0}},
		makeCommit(4): {{UploadID: 1, Distance: 2}, {UploadID: 2, Distance: 2}, {UploadID: 3, Distance: 1}, {UploadID: 4, Distance: 1}, {UploadID: 7, Distance: 0}},
		makeCommit(5): {{UploadID: 1, Distance: 3}, {UploadID: 2, Distance: 3}, {UploadID: 6, Distance: 1}, {UploadID: 7, Distance: 1}, {UploadID: 8, Distance: 0}},
		makeCommit(6): {{UploadID: 1, Distance: 4}, {UploadID: 2, Distance: 4}, {UploadID: 7, Distance: 2}, {UploadID: 8, Distance: 1}, {UploadID: 9, Distance: 0}},
	}
	if diff := cmp.Diff(expectedVisibleUploads, normalizeVisibleUploads(visibleUploads), UploadMetaComparer); diff != "" {
		t.Errorf("unexpected visible uploads (-want +got):\n%s", diff)
	}

	testFindClosestDumps(t, store, []FindClosestDumpsTestCase{
		{commit: makeCommit(4), file: "root1/file.ts", rootMustEnclosePath: true, graph: graph, allOfIDs: []int{7, 3}},
		{commit: makeCommit(5), file: "root2/file.ts", rootMustEnclosePath: true, graph: graph, allOfIDs: []int{8, 7}},
		{commit: makeCommit(3), file: "root3/file.ts", rootMustEnclosePath: true, graph: graph, allOfIDs: []int{5, 1}},
		{commit: makeCommit(1), file: "root4/file.ts", rootMustEnclosePath: true, graph: graph, allOfIDs: []int{2, 5}},
		{commit: makeCommit(2), file: "root4/file.ts", rootMustEnclosePath: true, graph: graph, allOfIDs: []int{2, 5}},
	})
}

func TestFindClosestDumpsIndexerName(t *testing.T) {
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

	visibleUploads, err := calculateVisibleUploads(graph, toUploadMeta(uploads))
	if err != nil {
		t.Fatalf("unexpected error while calculating visible uploads: %s", err)
	}
	insertNearestUploads(t, dbconn.Global, 50, visibleUploads)

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
	if diff := cmp.Diff(expectedVisibleUploads, normalizeVisibleUploads(visibleUploads), UploadMetaComparer); diff != "" {
		t.Errorf("unexpected visible uploads (-want +got):\n%s", diff)
	}

	testFindClosestDumps(t, store, []FindClosestDumpsTestCase{
		{commit: makeCommit(5), file: "root1/file.ts", indexer: "idx1", graph: graph, allOfIDs: []int{1}},
		{commit: makeCommit(5), file: "root2/file.ts", indexer: "idx1", graph: graph, allOfIDs: []int{2}},
		{commit: makeCommit(5), file: "root3/file.ts", indexer: "idx1", graph: graph, allOfIDs: []int{3}},
		{commit: makeCommit(5), file: "root4/file.ts", indexer: "idx1", graph: graph, allOfIDs: []int{4}},
		{commit: makeCommit(5), file: "root1/file.ts", indexer: "idx2", graph: graph, allOfIDs: []int{5}},
		{commit: makeCommit(5), file: "root2/file.ts", indexer: "idx2", graph: graph, allOfIDs: []int{6}},
		{commit: makeCommit(5), file: "root3/file.ts", indexer: "idx2", graph: graph, allOfIDs: []int{7}},
		{commit: makeCommit(5), file: "root4/file.ts", indexer: "idx2", graph: graph, allOfIDs: []int{8}},
	})
}

func TestFindClosestDumpsIntersectingPath(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	store := testStore()

	// This database has the following commit graph:
	//
	// [1]

	uploads := []Upload{
		{ID: 1, Commit: makeCommit(1), Root: "web/src/", Indexer: "lsif-eslint"},
	}
	insertUploads(t, dbconn.Global, uploads...)

	graph := map[string][]string{
		makeCommit(1): {},
	}

	visibleUploads, err := calculateVisibleUploads(graph, toUploadMeta(uploads))
	if err != nil {
		t.Fatalf("unexpected error while calculating visible uploads: %s", err)
	}
	insertNearestUploads(t, dbconn.Global, 50, visibleUploads)

	expectedVisibleUploads := map[string][]UploadMeta{
		makeCommit(1): {{UploadID: 1, Distance: 0}},
	}
	if diff := cmp.Diff(expectedVisibleUploads, normalizeVisibleUploads(visibleUploads), UploadMetaComparer); diff != "" {
		t.Errorf("unexpected visible uploads (-want +got):\n%s", diff)
	}

	testFindClosestDumps(t, store, []FindClosestDumpsTestCase{
		{commit: makeCommit(1), file: "", rootMustEnclosePath: false, graph: graph, allOfIDs: []int{1}},
		{commit: makeCommit(1), file: "web/", rootMustEnclosePath: false, graph: graph, allOfIDs: []int{1}},
		{commit: makeCommit(1), file: "web/src/file.ts", rootMustEnclosePath: false, graph: graph, allOfIDs: []int{1}},
	})
}

func TestFindClosestDumpsFromGraphFragment(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	store := testStore()

	// This database has the following commit graph:
	//
	//       <- known commits || new commits ->
	//                        ||
	// [1] --+--- 2 --- 3 --  || -- 4 --+-- 7
	//       |                ||       /
	//       +-- [5] -- 6 --- || -----+

	uploads := []Upload{
		{ID: 1, Commit: makeCommit(1)},
		{ID: 2, Commit: makeCommit(5)},
	}
	insertUploads(t, dbconn.Global, uploads...)

	currentGraph := map[string][]string{
		makeCommit(1): {},
		makeCommit(2): {makeCommit(1)},
		makeCommit(3): {makeCommit(2)},
		makeCommit(5): {makeCommit(1)},
		makeCommit(6): {makeCommit(5)},
	}

	visibleUploads, err := calculateVisibleUploads(currentGraph, toUploadMeta(uploads))
	if err != nil {
		t.Fatalf("unexpected error while calculating visible uploads: %s", err)
	}
	insertNearestUploads(t, dbconn.Global, 50, visibleUploads)

	expectedVisibleUploads := map[string][]UploadMeta{
		makeCommit(1): {{UploadID: 1, Distance: 0}},
		makeCommit(2): {{UploadID: 1, Distance: 1}},
		makeCommit(3): {{UploadID: 1, Distance: 2}},
		makeCommit(5): {{UploadID: 2, Distance: 0}},
		makeCommit(6): {{UploadID: 2, Distance: 1}},
	}
	if diff := cmp.Diff(expectedVisibleUploads, normalizeVisibleUploads(visibleUploads), UploadMetaComparer); diff != "" {
		t.Errorf("unexpected visible uploads (-want +got):\n%s", diff)
	}

	graphFragment := map[string][]string{
		makeCommit(3): {},
		makeCommit(6): {},
		makeCommit(4): {makeCommit(3)},
		makeCommit(7): {makeCommit(4), makeCommit(6)},
	}

	testFindClosestDumps(t, store, []FindClosestDumpsTestCase{
		// Note: Can't query anything outside of the graph fragment
		{commit: makeCommit(3), file: "file.ts", rootMustEnclosePath: true, graph: graphFragment, anyOfIDs: []int{1}},
		{commit: makeCommit(6), file: "file.ts", rootMustEnclosePath: true, graph: graphFragment, anyOfIDs: []int{2}},
		{commit: makeCommit(4), file: "file.ts", rootMustEnclosePath: true, graph: graphFragment, graphFragmentOnly: true, anyOfIDs: []int{1}},
		{commit: makeCommit(7), file: "file.ts", rootMustEnclosePath: true, graph: graphFragment, graphFragmentOnly: true, anyOfIDs: []int{2}},
	})
}

func TestDeleteOldestDump(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	store := testStore()

	// Cannot prune empty dump set
	if _, prunable, err := store.DeleteOldestDump(context.Background()); err != nil {
		t.Fatalf("unexpected error pruning dumps: %s", err)
	} else if prunable {
		t.Fatal("unexpectedly prunable")
	}

	t1 := time.Unix(1587396557, 0).UTC()
	t2 := t1.Add(time.Minute)
	t3 := t1.Add(time.Minute * 2)
	t4 := t1.Add(time.Minute * 3)

	insertUploads(t, dbconn.Global,
		Upload{ID: 1, UploadedAt: t1},
		Upload{ID: 2, UploadedAt: t2},
		Upload{ID: 3, UploadedAt: t3, State: "queued"},
		Upload{ID: 4, UploadedAt: t3},
		Upload{ID: 5, UploadedAt: t4},
	)
	insertVisibleAtTip(t, dbconn.Global, 50, 2)

	// Prune oldest
	if id, prunable, err := store.DeleteOldestDump(context.Background()); err != nil {
		t.Fatalf("unexpected error pruning dumps: %s", err)
	} else if !prunable {
		t.Fatal("unexpectedly non-prunable")
	} else if id != 1 {
		t.Errorf("unexpected pruned identifier. want=%d have=%d", 1, id)
	}

	// Ensure record was deleted
	if states, err := store.GetStates(context.Background(), []int{1}); err != nil {
		t.Fatalf("unexpected error getting states: %s", err)
	} else if diff := cmp.Diff(map[int]string{1: "deleted"}, states); diff != "" {
		t.Errorf("unexpected dump (-want +got):\n%s", diff)
	}

	// Ensure repository was marked as dirty
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

	// Prune next oldest (skips visible at tip, non-completed uploads)
	if id, prunable, err := store.DeleteOldestDump(context.Background()); err != nil {
		t.Fatalf("unexpected error pruning dumps: %s", err)
	} else if !prunable {
		t.Fatal("unexpectedly non-prunable")
	} else if id != 4 {
		t.Errorf("unexpected pruned identifier. want=%d have=%d", 4, id)
	}

	// Ensure record was deleted
	if states, err := store.GetStates(context.Background(), []int{4}); err != nil {
		t.Fatalf("unexpected error getting states: %s", err)
	} else if diff := cmp.Diff(map[int]string{4: "deleted"}, states); diff != "" {
		t.Errorf("unexpected dump (-want +got):\n%s", diff)
	}
}

func TestSoftDeleteOldDumps(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	store := testStore()

	t1 := time.Unix(1587396557, 0).UTC()
	t2 := t1.Add(time.Minute)
	t3 := t1.Add(time.Minute * 4)
	t4 := t1.Add(time.Minute * 6)

	insertUploads(t, dbconn.Global,
		Upload{ID: 1, FinishedAt: &t1},
		Upload{ID: 2, FinishedAt: &t2}, // visible
		Upload{ID: 3, FinishedAt: &t2},
		Upload{ID: 4, FinishedAt: &t3}, // visible
		Upload{ID: 5, FinishedAt: &t3},
		Upload{ID: 6, FinishedAt: &t4}, // too new
		Upload{ID: 7, FinishedAt: &t4}, // too new
	)
	insertVisibleAtTip(t, dbconn.Global, 50, 2, 4)

	if count, err := store.SoftDeleteOldDumps(context.Background(), time.Minute, t1.Add(time.Minute*6)); err != nil {
		t.Fatalf("unexpected error pruning dumps: %s", err)
	} else if count != 3 {
		t.Fatalf("unexpected number of uploads deleted: want=%d have=%d", 3, count)
	}

	expectedStates := map[int]string{
		1: "deleted",
		2: "completed",
		3: "deleted",
		4: "completed",
		5: "deleted",
		6: "completed",
		7: "completed",
	}

	// Ensure record was deleted
	if states, err := store.GetStates(context.Background(), []int{1, 2, 3, 4, 5, 6, 7}); err != nil {
		t.Fatalf("unexpected error getting states: %s", err)
	} else if diff := cmp.Diff(expectedStates, states); diff != "" {
		t.Errorf("unexpected dump (-want +got):\n%s", diff)
	}

	// Ensure repository was marked as dirty
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

type FindClosestDumpsTestCase struct {
	commit              string
	file                string
	rootMustEnclosePath bool
	indexer             string
	graph               map[string][]string
	graphFragmentOnly   bool
	anyOfIDs            []int
	allOfIDs            []int
}

func testFindClosestDumps(t *testing.T, store Store, testCases []FindClosestDumpsTestCase) {
	for _, testCase := range testCases {
		name := fmt.Sprintf(
			"commit=%s file=%s rootMustEnclosePath=%v indexer=%s",
			testCase.commit,
			testCase.file,
			testCase.rootMustEnclosePath,
			testCase.indexer,
		)

		assertDumpIDs := func(t *testing.T, dumps []Dump) {
			if len(testCase.anyOfIDs) > 0 {
				testAnyOf(t, dumps, testCase.anyOfIDs)
				return
			}

			if len(testCase.allOfIDs) > 0 {
				testAllOf(t, dumps, testCase.allOfIDs)
				return
			}

			if len(dumps) != 0 {
				t.Errorf("unexpected nearest dump length. want=%d have=%d", 0, len(dumps))
				return
			}
		}

		if !testCase.graphFragmentOnly {
			t.Run(name, func(t *testing.T) {
				dumps, err := store.FindClosestDumps(context.Background(), 50, testCase.commit, testCase.file, testCase.rootMustEnclosePath, testCase.indexer)
				if err != nil {
					t.Fatalf("unexpected error finding closest dumps: %s", err)
				}

				assertDumpIDs(t, dumps)
			})
		}

		if testCase.graph != nil {
			t.Run(name+" [graph-fragment]", func(t *testing.T) {
				dumps, err := store.FindClosestDumpsFromGraphFragment(context.Background(), 50, testCase.commit, testCase.file, testCase.rootMustEnclosePath, testCase.indexer, testCase.graph)
				if err != nil {
					t.Fatalf("unexpected error finding closest dumps: %s", err)
				}

				assertDumpIDs(t, dumps)
			})
		}
	}
}

func testAnyOf(t *testing.T, dumps []Dump, expectedIDs []int) {
	if len(dumps) != 1 {
		t.Errorf("unexpected nearest dump length. want=%d have=%d", 1, len(dumps))
		return
	}

	if !testPresence(dumps[0].ID, expectedIDs) {
		t.Errorf("unexpected nearest dump ids. want one of %v have=%v", expectedIDs, dumps[0].ID)
	}
}

func testAllOf(t *testing.T, dumps []Dump, expectedIDs []int) {
	if len(dumps) != len(expectedIDs) {
		t.Errorf("unexpected nearest dump length. want=%d have=%d", 1, len(dumps))
	}

	var dumpIDs []int
	for _, dump := range dumps {
		dumpIDs = append(dumpIDs, dump.ID)
	}

	for _, expectedID := range expectedIDs {
		if !testPresence(expectedID, dumpIDs) {
			t.Errorf("unexpected nearest dump ids. want all of %v have=%v", expectedIDs, dumpIDs)
			return
		}
	}

}

func testPresence(needle int, haystack []int) bool {
	for _, candidate := range haystack {
		if needle == candidate {
			return true
		}
	}

	return false
}

func TestDeleteOverlappingDumps(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	store := testStore()

	insertUploads(t, dbconn.Global, Upload{
		ID:      1,
		Commit:  makeCommit(1),
		Root:    "cmd/",
		Indexer: "lsif-go",
	})

	err := store.DeleteOverlappingDumps(context.Background(), 50, makeCommit(1), "cmd/", "lsif-go")
	if err != nil {
		t.Fatalf("unexpected error deleting dump: %s", err)
	}

	// Ensure record was deleted
	if states, err := store.GetStates(context.Background(), []int{1}); err != nil {
		t.Fatalf("unexpected error getting states: %s", err)
	} else if diff := cmp.Diff(map[int]string{1: "deleted"}, states); diff != "" {
		t.Errorf("unexpected dump (-want +got):\n%s", diff)
	}
}

func TestDeleteOverlappingDumpsNoMatches(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	store := testStore()

	insertUploads(t, dbconn.Global, Upload{
		ID:      1,
		Commit:  makeCommit(1),
		Root:    "cmd/",
		Indexer: "lsif-go",
	})

	testCases := []struct {
		commit  string
		root    string
		indexer string
	}{
		{makeCommit(2), "cmd/", "lsif-go"},
		{makeCommit(1), "cmds/", "lsif-go"},
		{makeCommit(1), "cmd/", "lsif-tsc"},
	}

	for _, testCase := range testCases {
		err := store.DeleteOverlappingDumps(context.Background(), 50, testCase.commit, testCase.root, testCase.indexer)
		if err != nil {
			t.Fatalf("unexpected error deleting dump: %s", err)
		}
	}

	// Original dump still exists
	if _, exists, err := store.GetDumpByID(context.Background(), 1); err != nil {
		t.Fatalf("unexpected error getting dump: %s", err)
	} else if !exists {
		t.Fatal("expected dump record to still exist")
	}
}

func TestDeleteOverlappingDumpsIgnoresIncompleteUploads(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	store := testStore()

	insertUploads(t, dbconn.Global, Upload{
		ID:      1,
		Commit:  makeCommit(1),
		Root:    "cmd/",
		Indexer: "lsif-go",
		State:   "queued",
	})

	err := store.DeleteOverlappingDumps(context.Background(), 50, makeCommit(1), "cmd/", "lsif-go")
	if err != nil {
		t.Fatalf("unexpected error deleting dump: %s", err)
	}

	// Original upload still exists
	if _, exists, err := store.GetUploadByID(context.Background(), 1); err != nil {
		t.Fatalf("unexpected error getting dump: %s", err)
	} else if !exists {
		t.Fatal("expected dump record to still exist")
	}
}
