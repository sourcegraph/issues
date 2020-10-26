package lsifstore

import (
	"context"
	"database/sql"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/keegancsmith/sqlf"
	"github.com/sourcegraph/sourcegraph/internal/db/basestore"
	"github.com/sourcegraph/sourcegraph/internal/db/dbconn"
	"github.com/sourcegraph/sourcegraph/internal/db/dbtesting"
)

func init() {
	dbtesting.DBNameSuffix = "lsifstore"
}

func TestDumpIDs(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	store := &store{Store: basestore.NewWithDB(dbconn.Global, sql.TxOptions{})}

	for i := 0; i < 10; i++ {
		query := sqlf.Sprintf("INSERT INTO lsif_data_metadata (dump_id, num_result_chunks) VALUES (%s, 0)", i+1)

		if _, err := dbconn.Global.Exec(query.Query(sqlf.PostgresBindVar), query.Args()...); err != nil {
			t.Fatalf("unexpected error inserting repo: %s", err)
		}
	}

	dumpIDs, err := store.DumpIDs(context.Background(), 5, 2)
	if err != nil {
		t.Fatalf("unexpected error fetching dump identifiers: %s", err)
	}

	if diff := cmp.Diff([]int{3, 4, 5, 6, 7}, dumpIDs); diff != "" {
		t.Errorf("unexpected dump identifiers (-want +got):\n%s", diff)
	}
}

func TestClear(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	store := &store{Store: basestore.NewWithDB(dbconn.Global, sql.TxOptions{})}

	for i := 0; i < 5; i++ {
		query := sqlf.Sprintf("INSERT INTO lsif_data_metadata (dump_id, num_result_chunks) VALUES (%s, 0)", i+1)

		if _, err := dbconn.Global.Exec(query.Query(sqlf.PostgresBindVar), query.Args()...); err != nil {
			t.Fatalf("unexpected error inserting repo: %s", err)
		}
	}

	if err := store.Clear(context.Background(), 2, 4); err != nil {
		t.Fatalf("unexpected error clearing bundle data: %s", err)
	}

	dumpIDs, err := basestore.ScanInts(dbconn.Global.Query("SELECT dump_id FROM lsif_data_metadata"))
	if err != nil {
		t.Fatalf("Unexpected error querying dump identifiers: %s", err)
	}

	if diff := cmp.Diff([]int{1, 3, 5}, dumpIDs); diff != "" {
		t.Errorf("unexpected dump identifiers (-want +got):\n%s", diff)
	}
}
