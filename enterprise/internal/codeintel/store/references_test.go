package store

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/types"
	"github.com/sourcegraph/sourcegraph/internal/db/basestore"
	"github.com/sourcegraph/sourcegraph/internal/db/dbconn"
	"github.com/sourcegraph/sourcegraph/internal/db/dbtesting"
)

func TestSameRepoPager(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	store := testStore()

	insertUploads(t, dbconn.Global,
		Upload{ID: 1, Commit: makeCommit(2), Root: "sub1/"},
		Upload{ID: 2, Commit: makeCommit(3), Root: "sub2/"},
		Upload{ID: 3, Commit: makeCommit(4), Root: "sub3/"},
		Upload{ID: 4, Commit: makeCommit(3), Root: "sub4/"},
		Upload{ID: 5, Commit: makeCommit(2), Root: "sub5/"},
	)

	insertNearestUploads(t, dbconn.Global, 50, map[string][]UploadMeta{
		makeCommit(1): {
			{UploadID: 1, Root: "sub1/", Distance: 1},
			{UploadID: 2, Root: "sub2/", Distance: 2},
			{UploadID: 3, Root: "sub3/", Distance: 3},
			{UploadID: 4, Root: "sub4/", Distance: 2},
			{UploadID: 5, Root: "sub5/", Distance: 1},
		},
		makeCommit(2): {
			{UploadID: 1, Root: "sub1/", Distance: 0},
			{UploadID: 2, Root: "sub2/", Distance: 1},
			{UploadID: 3, Root: "sub3/", Distance: 2},
			{UploadID: 4, Root: "sub4/", Distance: 1},
			{UploadID: 5, Root: "sub5/", Distance: 0},
		},
		makeCommit(3): {
			{UploadID: 1, Root: "sub1/", Distance: 1},
			{UploadID: 2, Root: "sub2/", Distance: 0},
			{UploadID: 3, Root: "sub3/", Distance: 1},
			{UploadID: 4, Root: "sub4/", Distance: 0},
			{UploadID: 5, Root: "sub5/", Distance: 1},
		},
		makeCommit(4): {
			{UploadID: 1, Root: "sub1/", Distance: 2},
			{UploadID: 2, Root: "sub2/", Distance: 1},
			{UploadID: 3, Root: "sub3/", Distance: 0},
			{UploadID: 4, Root: "sub4/", Distance: 1},
			{UploadID: 5, Root: "sub5/", Distance: 2},
		},
	})

	expected := []types.PackageReference{
		{DumpID: 1, Scheme: "gomod", Name: "leftpad", Version: "0.1.0", Filter: []byte("f1")},
		{DumpID: 2, Scheme: "gomod", Name: "leftpad", Version: "0.1.0", Filter: []byte("f2")},
		{DumpID: 3, Scheme: "gomod", Name: "leftpad", Version: "0.1.0", Filter: []byte("f3")},
		{DumpID: 4, Scheme: "gomod", Name: "leftpad", Version: "0.1.0", Filter: []byte("f4")},
		{DumpID: 5, Scheme: "gomod", Name: "leftpad", Version: "0.1.0", Filter: []byte("f5")},
	}
	insertPackageReferences(t, store, expected)

	totalCount, pager, err := store.SameRepoPager(context.Background(), 50, makeCommit(1), "gomod", "leftpad", "0.1.0", 5)
	if err != nil {
		t.Fatalf("unexpected error getting pager: %s", err)
	}
	defer func() { _ = pager.Done(nil) }()

	if totalCount != 5 {
		t.Errorf("unexpected dump. want=%d have=%d", 5, totalCount)
	}

	if references, err := pager.PageFromOffset(context.Background(), 0); err != nil {
		t.Fatalf("unexpected error getting next page: %s", err)
	} else if diff := cmp.Diff(expected, references); diff != "" {
		t.Errorf("unexpected references (-want +got):\n%s", diff)
	}
}

func TestSameRepoPagerEmpty(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	store := testStore()

	totalCount, pager, err := store.SameRepoPager(context.Background(), 50, makeCommit(1), "gomod", "leftpad", "0.1.0", 5)
	if err != nil {
		t.Fatalf("unexpected error getting pager: %s", err)
	}
	defer func() { _ = pager.Done(nil) }()

	if totalCount != 0 {
		t.Errorf("unexpected dump. want=%d have=%d", 0, totalCount)
	}
}

func TestSameRepoPagerMultiplePages(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	store := testStore()

	insertUploads(t, dbconn.Global,
		Upload{ID: 1, Commit: makeCommit(1), Root: "sub1/"},
		Upload{ID: 2, Commit: makeCommit(1), Root: "sub2/"},
		Upload{ID: 3, Commit: makeCommit(1), Root: "sub3/"},
		Upload{ID: 4, Commit: makeCommit(1), Root: "sub4/"},
		Upload{ID: 5, Commit: makeCommit(1), Root: "sub5/"},
		Upload{ID: 6, Commit: makeCommit(1), Root: "sub6/"},
		Upload{ID: 7, Commit: makeCommit(1), Root: "sub7/"},
		Upload{ID: 8, Commit: makeCommit(1), Root: "sub8/"},
		Upload{ID: 9, Commit: makeCommit(1), Root: "sub9/"},
	)

	insertNearestUploads(t, dbconn.Global, 50, map[string][]UploadMeta{
		makeCommit(1): {
			{UploadID: 1, Root: "sub1/", Distance: 0},
			{UploadID: 2, Root: "sub2/", Distance: 0},
			{UploadID: 3, Root: "sub3/", Distance: 0},
			{UploadID: 4, Root: "sub4/", Distance: 0},
			{UploadID: 5, Root: "sub5/", Distance: 0},
			{UploadID: 6, Root: "sub6/", Distance: 0},
			{UploadID: 7, Root: "sub7/", Distance: 0},
			{UploadID: 8, Root: "sub8/", Distance: 0},
			{UploadID: 9, Root: "sub9/", Distance: 0},
		},
	})

	expected := []types.PackageReference{
		{DumpID: 1, Scheme: "gomod", Name: "leftpad", Version: "0.1.0", Filter: []byte("f1")},
		{DumpID: 2, Scheme: "gomod", Name: "leftpad", Version: "0.1.0", Filter: []byte("f2")},
		{DumpID: 3, Scheme: "gomod", Name: "leftpad", Version: "0.1.0", Filter: []byte("f3")},
		{DumpID: 4, Scheme: "gomod", Name: "leftpad", Version: "0.1.0", Filter: []byte("f4")},
		{DumpID: 5, Scheme: "gomod", Name: "leftpad", Version: "0.1.0", Filter: []byte("f5")},
		{DumpID: 6, Scheme: "gomod", Name: "leftpad", Version: "0.1.0", Filter: []byte("f6")},
		{DumpID: 7, Scheme: "gomod", Name: "leftpad", Version: "0.1.0", Filter: []byte("f7")},
		{DumpID: 8, Scheme: "gomod", Name: "leftpad", Version: "0.1.0", Filter: []byte("f8")},
		{DumpID: 9, Scheme: "gomod", Name: "leftpad", Version: "0.1.0", Filter: []byte("f9")},
	}
	insertPackageReferences(t, store, expected)

	totalCount, pager, err := store.SameRepoPager(context.Background(), 50, makeCommit(1), "gomod", "leftpad", "0.1.0", 3)
	if err != nil {
		t.Fatalf("unexpected error getting pager: %s", err)
	}
	defer func() { _ = pager.Done(nil) }()

	if totalCount != 9 {
		t.Errorf("unexpected dump. want=%d have=%d", 9, totalCount)
	}

	for lo := 0; lo < len(expected); lo++ {
		hi := lo + 3
		if hi > len(expected) {
			hi = len(expected)
		}

		if references, err := pager.PageFromOffset(context.Background(), lo); err != nil {
			t.Fatalf("unexpected error getting page at offset %d: %s", lo, err)
		} else if diff := cmp.Diff(expected[lo:hi], references); diff != "" {
			t.Errorf("unexpected references at offset %d (-want +got):\n%s", lo, diff)
		}
	}
}

func TestSameRepoPagerVisibility(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	store := testStore()

	insertUploads(t, dbconn.Global,
		Upload{ID: 1, Commit: makeCommit(1), Root: "sub1/"}, // not visible
		Upload{ID: 2, Commit: makeCommit(2), Root: "sub2/"}, // not visible
		Upload{ID: 3, Commit: makeCommit(3), Root: "sub1/"},
		Upload{ID: 4, Commit: makeCommit(4), Root: "sub2/"},
		Upload{ID: 5, Commit: makeCommit(5), Root: "sub5/"},
	)

	insertNearestUploads(t, dbconn.Global, 50, map[string][]UploadMeta{
		makeCommit(1): {{UploadID: 1, Root: "sub1/", Distance: 0}},
		makeCommit(2): {{UploadID: 2, Root: "sub2/", Distance: 0}},
		makeCommit(3): {{UploadID: 3, Root: "sub1/", Distance: 0}},
		makeCommit(4): {{UploadID: 4, Root: "sub2/", Distance: 0}},
		makeCommit(5): {{UploadID: 5, Root: "sub5/", Distance: 0}},
		makeCommit(6): {
			{UploadID: 3, Root: "sub1/", Distance: 3},
			{UploadID: 4, Root: "sub2/", Distance: 2},
			{UploadID: 5, Root: "sub5/", Distance: 1},
		},
	})

	expected := []types.PackageReference{
		{DumpID: 3, Scheme: "gomod", Name: "leftpad", Version: "0.1.0", Filter: []byte("f3")},
		{DumpID: 4, Scheme: "gomod", Name: "leftpad", Version: "0.1.0", Filter: []byte("f4")},
		{DumpID: 5, Scheme: "gomod", Name: "leftpad", Version: "0.1.0", Filter: []byte("f5")},
	}
	insertPackageReferences(t, store, append([]types.PackageReference{
		{DumpID: 1, Scheme: "gomod", Name: "leftpad", Version: "0.1.0", Filter: []byte("f1")},
		{DumpID: 2, Scheme: "gomod", Name: "leftpad", Version: "0.1.0", Filter: []byte("f2")},
	}, expected...))

	totalCount, pager, err := store.SameRepoPager(context.Background(), 50, makeCommit(6), "gomod", "leftpad", "0.1.0", 5)
	if err != nil {
		t.Fatalf("unexpected error getting pager: %s", err)
	}
	defer func() { _ = pager.Done(nil) }()

	if totalCount != 3 {
		t.Errorf("unexpected dump. want=%d have=%d", 5, totalCount)
	}

	if references, err := pager.PageFromOffset(context.Background(), 0); err != nil {
		t.Fatalf("unexpected error getting next page: %s", err)
	} else if diff := cmp.Diff(expected, references); diff != "" {
		t.Errorf("unexpected references (-want +got):\n%s", diff)
	}
}

func TestPackageReferencePager(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	store := testStore()

	insertUploads(t, dbconn.Global,
		Upload{ID: 1, Commit: makeCommit(1)},
		Upload{ID: 2, Commit: makeCommit(2), RepositoryID: 51},
		Upload{ID: 3, Commit: makeCommit(3), RepositoryID: 52},
		Upload{ID: 4, Commit: makeCommit(4), RepositoryID: 53},
		Upload{ID: 5, Commit: makeCommit(5), RepositoryID: 54},
		Upload{ID: 6, Commit: makeCommit(6), RepositoryID: 55},
		Upload{ID: 7, Commit: makeCommit(6), RepositoryID: 56},
	)
	insertVisibleAtTip(t, dbconn.Global, 50, 1)
	insertVisibleAtTip(t, dbconn.Global, 51, 2)
	insertVisibleAtTip(t, dbconn.Global, 52, 3)
	insertVisibleAtTip(t, dbconn.Global, 53, 4)
	insertVisibleAtTip(t, dbconn.Global, 54, 5)
	insertVisibleAtTip(t, dbconn.Global, 56, 7)

	expected := []types.PackageReference{
		{DumpID: 2, Scheme: "gomod", Name: "leftpad", Version: "0.1.0", Filter: []byte("f2")},
		{DumpID: 3, Scheme: "gomod", Name: "leftpad", Version: "0.1.0", Filter: []byte("f3")},
		{DumpID: 4, Scheme: "gomod", Name: "leftpad", Version: "0.1.0", Filter: []byte("f4")},
		{DumpID: 5, Scheme: "gomod", Name: "leftpad", Version: "0.1.0", Filter: []byte("f5")},
		{DumpID: 7, Scheme: "gomod", Name: "leftpad", Version: "0.1.0", Filter: []byte("f7")},
	}
	insertPackageReferences(t, store, append([]types.PackageReference{
		{DumpID: 1, Scheme: "gomod", Name: "leftpad", Version: "0.1.0", Filter: []byte("f1")},
		{DumpID: 6, Scheme: "gomod", Name: "leftpad", Version: "0.1.0", Filter: []byte("f6")},
	}, expected...))

	totalCount, pager, err := store.PackageReferencePager(context.Background(), "gomod", "leftpad", "0.1.0", 50, 5)
	if err != nil {
		t.Fatalf("unexpected error getting pager: %s", err)
	}
	defer func() { _ = pager.Done(nil) }()

	if totalCount != 5 {
		t.Errorf("unexpected dump. want=%d have=%d", 5, totalCount)
	}

	if references, err := pager.PageFromOffset(context.Background(), 0); err != nil {
		t.Fatalf("unexpected error getting next page: %s", err)
	} else if diff := cmp.Diff(expected, references); diff != "" {
		t.Errorf("unexpected references (-want +got):\n%s", diff)
	}
}

func TestPackageReferencePagerEmpty(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	store := testStore()

	totalCount, pager, err := store.PackageReferencePager(context.Background(), "gomod", "leftpad", "0.1.0", 50, 5)
	if err != nil {
		t.Fatalf("unexpected error getting pager: %s", err)
	}
	defer func() { _ = pager.Done(nil) }()

	if totalCount != 0 {
		t.Errorf("unexpected dump. want=%d have=%d", 0, totalCount)
	}
}

func TestPackageReferencePagerPages(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	store := testStore()

	insertUploads(t, dbconn.Global,
		Upload{ID: 1, Commit: makeCommit(1), RepositoryID: 51},
		Upload{ID: 2, Commit: makeCommit(2), RepositoryID: 52},
		Upload{ID: 3, Commit: makeCommit(3), RepositoryID: 53},
		Upload{ID: 4, Commit: makeCommit(4), RepositoryID: 54},
		Upload{ID: 5, Commit: makeCommit(5), RepositoryID: 55},
		Upload{ID: 6, Commit: makeCommit(6), RepositoryID: 56},
		Upload{ID: 7, Commit: makeCommit(7), RepositoryID: 57},
		Upload{ID: 8, Commit: makeCommit(8), RepositoryID: 58},
		Upload{ID: 9, Commit: makeCommit(9), RepositoryID: 59},
	)
	insertVisibleAtTip(t, dbconn.Global, 51, 1)
	insertVisibleAtTip(t, dbconn.Global, 52, 2)
	insertVisibleAtTip(t, dbconn.Global, 53, 3)
	insertVisibleAtTip(t, dbconn.Global, 54, 4)
	insertVisibleAtTip(t, dbconn.Global, 55, 5)
	insertVisibleAtTip(t, dbconn.Global, 56, 6)
	insertVisibleAtTip(t, dbconn.Global, 57, 7)
	insertVisibleAtTip(t, dbconn.Global, 58, 8)
	insertVisibleAtTip(t, dbconn.Global, 59, 9)

	expected := []types.PackageReference{
		{DumpID: 1, Scheme: "gomod", Name: "leftpad", Version: "0.1.0", Filter: []byte("f1")},
		{DumpID: 2, Scheme: "gomod", Name: "leftpad", Version: "0.1.0", Filter: []byte("f2")},
		{DumpID: 3, Scheme: "gomod", Name: "leftpad", Version: "0.1.0", Filter: []byte("f3")},
		{DumpID: 4, Scheme: "gomod", Name: "leftpad", Version: "0.1.0", Filter: []byte("f4")},
		{DumpID: 5, Scheme: "gomod", Name: "leftpad", Version: "0.1.0", Filter: []byte("f5")},
		{DumpID: 6, Scheme: "gomod", Name: "leftpad", Version: "0.1.0", Filter: []byte("f6")},
		{DumpID: 7, Scheme: "gomod", Name: "leftpad", Version: "0.1.0", Filter: []byte("f7")},
		{DumpID: 8, Scheme: "gomod", Name: "leftpad", Version: "0.1.0", Filter: []byte("f8")},
		{DumpID: 9, Scheme: "gomod", Name: "leftpad", Version: "0.1.0", Filter: []byte("f9")},
	}
	insertPackageReferences(t, store, expected)

	totalCount, pager, err := store.PackageReferencePager(context.Background(), "gomod", "leftpad", "0.1.0", 50, 3)
	if err != nil {
		t.Fatalf("unexpected error getting pager: %s", err)
	}
	defer func() { _ = pager.Done(nil) }()

	if totalCount != 9 {
		t.Errorf("unexpected dump. want=%d have=%d", 9, totalCount)
	}

	testCases := []struct {
		offset int
		lo     int
		hi     int
	}{
		{0, 0, 3},
		{1, 1, 4},
		{2, 2, 5},
		{3, 3, 6},
		{4, 4, 7},
		{5, 5, 8},
		{6, 6, 9},
		{7, 7, 9},
		{8, 8, 9},
	}

	for _, testCase := range testCases {
		if references, err := pager.PageFromOffset(context.Background(), testCase.offset); err != nil {
			t.Fatalf("unexpected error getting page at offset %d: %s", testCase.offset, err)
		} else if diff := cmp.Diff(expected[testCase.lo:testCase.hi], references); diff != "" {
			t.Errorf("unexpected references at offset %d (-want +got):\n%s", testCase.offset, diff)
		}
	}
}

func TestUpdatePackageReferences(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	store := testStore()

	// for foreign key relation
	insertUploads(t, dbconn.Global, Upload{ID: 42})

	if err := store.UpdatePackageReferences(context.Background(), []types.PackageReference{
		{DumpID: 42, Scheme: "s0", Name: "n0", Version: "v0"},
		{DumpID: 42, Scheme: "s1", Name: "n1", Version: "v1"},
		{DumpID: 42, Scheme: "s2", Name: "n2", Version: "v2"},
		{DumpID: 42, Scheme: "s3", Name: "n3", Version: "v3"},
		{DumpID: 42, Scheme: "s4", Name: "n4", Version: "v4"},
		{DumpID: 42, Scheme: "s5", Name: "n5", Version: "v5"},
		{DumpID: 42, Scheme: "s6", Name: "n6", Version: "v6"},
		{DumpID: 42, Scheme: "s7", Name: "n7", Version: "v7"},
		{DumpID: 42, Scheme: "s8", Name: "n8", Version: "v8"},
		{DumpID: 42, Scheme: "s9", Name: "n9", Version: "v9"},
	}); err != nil {
		t.Fatalf("unexpected error updating references: %s", err)
	}

	count, _, err := basestore.ScanFirstInt(dbconn.Global.Query("SELECT COUNT(*) FROM lsif_references"))
	if err != nil {
		t.Fatalf("unexpected error checking reference count: %s", err)
	}
	if count != 10 {
		t.Errorf("unexpected reference count. want=%d have=%d", 10, count)
	}
}

func TestUpdatePackageReferencesEmpty(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	store := testStore()

	if err := store.UpdatePackageReferences(context.Background(), nil); err != nil {
		t.Fatalf("unexpected error updating references: %s", err)
	}

	count, _, err := basestore.ScanFirstInt(dbconn.Global.Query("SELECT COUNT(*) FROM lsif_references"))
	if err != nil {
		t.Fatalf("unexpected error checking reference count: %s", err)
	}
	if count != 0 {
		t.Errorf("unexpected reference count. want=%d have=%d", 0, count)
	}
}

func TestUpdatePackageReferencesWithDuplicates(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	store := testStore()

	// for foreign key relation
	insertUploads(t, dbconn.Global, Upload{ID: 42})

	if err := store.UpdatePackageReferences(context.Background(), []types.PackageReference{
		{DumpID: 42, Scheme: "s0", Name: "n0", Version: "v0"},
		{DumpID: 42, Scheme: "s1", Name: "n1", Version: "v1"},
		{DumpID: 42, Scheme: "s2", Name: "n2", Version: "v2"},
		{DumpID: 42, Scheme: "s3", Name: "n3", Version: "v3"},
	}); err != nil {
		t.Fatalf("unexpected error updating references: %s", err)
	}

	if err := store.UpdatePackageReferences(context.Background(), []types.PackageReference{
		{DumpID: 42, Scheme: "s0", Name: "n0", Version: "v0"}, // two copies
		{DumpID: 42, Scheme: "s2", Name: "n2", Version: "v2"}, // two copies
		{DumpID: 42, Scheme: "s4", Name: "n4", Version: "v4"},
		{DumpID: 42, Scheme: "s5", Name: "n5", Version: "v5"},
		{DumpID: 42, Scheme: "s6", Name: "n6", Version: "v6"},
		{DumpID: 42, Scheme: "s7", Name: "n7", Version: "v7"},
		{DumpID: 42, Scheme: "s8", Name: "n8", Version: "v8"},
		{DumpID: 42, Scheme: "s9", Name: "n9", Version: "v9"},
	}); err != nil {
		t.Fatalf("unexpected error updating references: %s", err)
	}

	count, _, err := basestore.ScanFirstInt(dbconn.Global.Query("SELECT COUNT(*) FROM lsif_references"))
	if err != nil {
		t.Fatalf("unexpected error checking reference count: %s", err)
	}
	if count != 12 {
		t.Errorf("unexpected reference count. want=%d have=%d", 12, count)
	}
}
