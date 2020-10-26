package batch

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/sourcegraph/sourcegraph/internal/db/dbconn"
	"github.com/sourcegraph/sourcegraph/internal/db/dbtesting"
)

func init() {
	dbtesting.DBNameSuffix = "batch"
}

func TestBatchInserter(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)

	createTableQuery := `
		CREATE TABLE batch_inserter_test (
			id SERIAL,
			col1 integer NOT NULL,
			col2 integer NOT NULL,
			col3 integer NOT NULL,
			col4 integer NOT NULL,
			col5 integer NOT NULL
		)
	`
	if _, err := dbconn.Global.Exec(createTableQuery); err != nil {
		t.Fatalf("unexpected error creating test table: %s", err)
	}

	var expectedValues [][]interface{}
	for i := 0; i < maxNumParameters*2; i++ {
		expectedValues = append(expectedValues, []interface{}{i, i + 1, i + 2, i + 3, i + 4})
	}

	ctx := context.Background()

	inserter := NewBatchInserter(ctx, dbconn.Global, "batch_inserter_test", "col1", "col2", "col3", "col4", "col5")
	for _, values := range expectedValues {
		if err := inserter.Insert(ctx, values...); err != nil {
			t.Fatalf("unexpected error inserting values: %s", err)
		}
	}

	if err := inserter.Flush(ctx); err != nil {
		t.Fatalf("unexpected error flushing: %s", err)
	}

	rows, err := dbconn.Global.Query("SELECT col1, col2, col3, col4, col5 from batch_inserter_test")
	if err != nil {
		t.Fatalf("unexpected error querying data: %s", err)
	}
	defer rows.Close()

	var values [][]interface{}
	for rows.Next() {
		var v1, v2, v3, v4, v5 int
		if err := rows.Scan(&v1, &v2, &v3, &v4, &v5); err != nil {
			t.Fatalf("unexpected error scanning data: %s", err)
		}

		values = append(values, []interface{}{v1, v2, v3, v4, v5})
	}

	if diff := cmp.Diff(expectedValues, values); diff != "" {
		t.Errorf("unexpected table contents (-want +got):\n%s", diff)
	}
}
