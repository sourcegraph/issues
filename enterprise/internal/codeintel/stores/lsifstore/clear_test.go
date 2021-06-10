package lsifstore

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/keegancsmith/sqlf"

	"github.com/sourcegraph/sourcegraph/internal/database/basestore"
	"github.com/sourcegraph/sourcegraph/internal/database/dbtest"
	"github.com/sourcegraph/sourcegraph/internal/observation"
)

func TestClear(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := dbtest.NewDB(t, "")
	store := NewStore(db, &observation.TestContext)

	for i := 0; i < 5; i++ {
		query := sqlf.Sprintf("INSERT INTO lsif_data_metadata (dump_id, num_result_chunks) VALUES (%s, 0)", i+1)

		if _, err := db.Exec(query.Query(sqlf.PostgresBindVar), query.Args()...); err != nil {
			t.Fatalf("unexpected error inserting repo: %s", err)
		}
	}

	if err := store.Clear(context.Background(), 2, 4); err != nil {
		t.Fatalf("unexpected error clearing bundle data: %s", err)
	}

	dumpIDs, err := basestore.ScanInts(db.Query("SELECT dump_id FROM lsif_data_metadata"))
	if err != nil {
		t.Fatalf("Unexpected error querying dump identifiers: %s", err)
	}

	if diff := cmp.Diff([]int{1, 3, 5}, dumpIDs); diff != "" {
		t.Errorf("unexpected dump identifiers (-want +got):\n%s", diff)
	}
}
