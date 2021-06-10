package dbstore

import (
	"context"
	"testing"

	"github.com/sourcegraph/sourcegraph/internal/database/basestore"
	"github.com/sourcegraph/sourcegraph/internal/database/dbtest"
	"github.com/sourcegraph/sourcegraph/lib/codeintel/semantic"
)

func TestUpdatePackageReferences(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := dbtest.NewDB(t, "")
	store := testStore(db)

	// for foreign key relation
	insertUploads(t, db, Upload{ID: 42})

	if err := store.UpdatePackageReferences(context.Background(), 42, []semantic.PackageReference{
		{Package: semantic.Package{Scheme: "s0", Name: "n0", Version: "v0"}},
		{Package: semantic.Package{Scheme: "s1", Name: "n1", Version: "v1"}},
		{Package: semantic.Package{Scheme: "s2", Name: "n2", Version: "v2"}},
		{Package: semantic.Package{Scheme: "s3", Name: "n3", Version: "v3"}},
		{Package: semantic.Package{Scheme: "s4", Name: "n4", Version: "v4"}},
		{Package: semantic.Package{Scheme: "s5", Name: "n5", Version: "v5"}},
		{Package: semantic.Package{Scheme: "s6", Name: "n6", Version: "v6"}},
		{Package: semantic.Package{Scheme: "s7", Name: "n7", Version: "v7"}},
		{Package: semantic.Package{Scheme: "s8", Name: "n8", Version: "v8"}},
		{Package: semantic.Package{Scheme: "s9", Name: "n9", Version: "v9"}},
	}); err != nil {
		t.Fatalf("unexpected error updating references: %s", err)
	}

	count, _, err := basestore.ScanFirstInt(db.Query("SELECT COUNT(*) FROM lsif_references"))
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
	db := dbtest.NewDB(t, "")
	store := testStore(db)

	if err := store.UpdatePackageReferences(context.Background(), 0, nil); err != nil {
		t.Fatalf("unexpected error updating references: %s", err)
	}

	count, _, err := basestore.ScanFirstInt(db.Query("SELECT COUNT(*) FROM lsif_references"))
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
	db := dbtest.NewDB(t, "")
	store := testStore(db)

	// for foreign key relation
	insertUploads(t, db, Upload{ID: 42})

	if err := store.UpdatePackageReferences(context.Background(), 42, []semantic.PackageReference{
		{Package: semantic.Package{Scheme: "s0", Name: "n0", Version: "v0"}},
		{Package: semantic.Package{Scheme: "s1", Name: "n1", Version: "v1"}},
		{Package: semantic.Package{Scheme: "s2", Name: "n2", Version: "v2"}},
		{Package: semantic.Package{Scheme: "s3", Name: "n3", Version: "v3"}},
	}); err != nil {
		t.Fatalf("unexpected error updating references: %s", err)
	}

	if err := store.UpdatePackageReferences(context.Background(), 42, []semantic.PackageReference{
		{Package: semantic.Package{Scheme: "s0", Name: "n0", Version: "v0"}}, // two copies
		{Package: semantic.Package{Scheme: "s2", Name: "n2", Version: "v2"}}, // two copies
		{Package: semantic.Package{Scheme: "s4", Name: "n4", Version: "v4"}},
		{Package: semantic.Package{Scheme: "s5", Name: "n5", Version: "v5"}},
		{Package: semantic.Package{Scheme: "s6", Name: "n6", Version: "v6"}},
		{Package: semantic.Package{Scheme: "s7", Name: "n7", Version: "v7"}},
		{Package: semantic.Package{Scheme: "s8", Name: "n8", Version: "v8"}},
		{Package: semantic.Package{Scheme: "s9", Name: "n9", Version: "v9"}},
	}); err != nil {
		t.Fatalf("unexpected error updating references: %s", err)
	}

	count, _, err := basestore.ScanFirstInt(db.Query("SELECT COUNT(*) FROM lsif_references"))
	if err != nil {
		t.Fatalf("unexpected error checking reference count: %s", err)
	}
	if count != 12 {
		t.Errorf("unexpected reference count. want=%d have=%d", 12, count)
	}
}
