package store

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/types"
	"github.com/sourcegraph/sourcegraph/internal/db/basestore"
	"github.com/sourcegraph/sourcegraph/internal/db/dbconn"
	"github.com/sourcegraph/sourcegraph/internal/db/dbtesting"
)

func TestGetPackage(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	store := testStore()

	// Package does not exist initially
	if _, exists, err := store.GetPackage(context.Background(), "gomod", "leftpad", "0.1.0"); err != nil {
		t.Fatalf("unexpected error getting package: %s", err)
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

	if err := store.UpdatePackages(context.Background(), []types.Package{
		{DumpID: 1, Scheme: "gomod", Name: "leftpad", Version: "0.1.0"},
		{DumpID: 1, Scheme: "gomod", Name: "leftpad", Version: "0.1.0"},
	}); err != nil {
		t.Fatalf("unexpected error updating packages: %s", err)
	}

	if dump, exists, err := store.GetPackage(context.Background(), "gomod", "leftpad", "0.1.0"); err != nil {
		t.Fatalf("unexpected error getting package: %s", err)
	} else if !exists {
		t.Fatal("expected record to exist")
	} else if diff := cmp.Diff(expected, dump); diff != "" {
		t.Errorf("unexpected dump (-want +got):\n%s", diff)
	}
}

func TestUpdatePackages(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	store := testStore()

	// for foreign key relation
	insertUploads(t, dbconn.Global, Upload{ID: 42})

	if err := store.UpdatePackages(context.Background(), []types.Package{
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
		t.Fatalf("unexpected error updating packages: %s", err)
	}

	count, _, err := basestore.ScanFirstInt(dbconn.Global.Query("SELECT COUNT(*) FROM lsif_packages"))
	if err != nil {
		t.Fatalf("unexpected error checking package count: %s", err)
	}
	if count != 10 {
		t.Errorf("unexpected package count. want=%d have=%d", 10, count)
	}
}

func TestUpdatePackagesEmpty(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	store := testStore()

	if err := store.UpdatePackages(context.Background(), nil); err != nil {
		t.Fatalf("unexpected error updating packages: %s", err)
	}

	count, _, err := basestore.ScanFirstInt(dbconn.Global.Query("SELECT COUNT(*) FROM lsif_packages"))
	if err != nil {
		t.Fatalf("unexpected error checking package count: %s", err)
	}
	if count != 0 {
		t.Errorf("unexpected package count. want=%d have=%d", 0, count)
	}
}
