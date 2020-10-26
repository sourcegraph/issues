package basestore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/keegancsmith/sqlf"
	"github.com/sourcegraph/sourcegraph/internal/db/dbconn"
	"github.com/sourcegraph/sourcegraph/internal/db/dbtesting"
	"github.com/sourcegraph/sourcegraph/internal/db/dbutil"
)

func init() {
	dbtesting.DBNameSuffix = "base-store"
}

func TestTransaction(t *testing.T) {
	setupStoreTest(t)
	store := testStore()

	// Add record outside of transaction, ensure it's visible
	if err := store.Exec(context.Background(), sqlf.Sprintf(`INSERT INTO store_counts_test VALUES (1, 42)`)); err != nil {
		t.Fatalf("unexpected error inserting count: %s", err)
	}
	assertCounts(t, dbconn.Global, map[int]int{1: 42})

	// Add record inside of a transaction
	tx1, err := store.Transact(context.Background())
	if err != nil {
		t.Fatalf("unexpected error creating transaction: %s", err)
	}
	if err := tx1.Exec(context.Background(), sqlf.Sprintf(`INSERT INTO store_counts_test VALUES (2, 43)`)); err != nil {
		t.Fatalf("unexpected error inserting count: %s", err)
	}

	// Add record inside of another transaction
	tx2, err := store.Transact(context.Background())
	if err != nil {
		t.Fatalf("unexpected error creating transaction: %s", err)
	}
	if err := tx2.Exec(context.Background(), sqlf.Sprintf(`INSERT INTO store_counts_test VALUES (3, 44)`)); err != nil {
		t.Fatalf("unexpected error inserting count: %s", err)
	}

	// Check what's visible pre-commit/rollback
	assertCounts(t, dbconn.Global, map[int]int{1: 42})
	assertCounts(t, tx1.handle.db, map[int]int{1: 42, 2: 43})
	assertCounts(t, tx2.handle.db, map[int]int{1: 42, 3: 44})

	// Finalize transactions
	rollbackErr := errors.New("rollback")
	if err := tx1.Done(rollbackErr); err != rollbackErr {
		t.Fatalf("unexpected error rolling back transaction. want=%q have=%q", rollbackErr, err)
	}
	if err := tx2.Done(nil); err != nil {
		t.Fatalf("unexpected error committing transaction: %s", err)
	}

	// Check what's visible post-commit/rollback
	assertCounts(t, dbconn.Global, map[int]int{1: 42, 3: 44})
}

func TestSavepoints(t *testing.T) {
	setupStoreTest(t)

	NumSavepointTests := 10

	for i := 0; i < NumSavepointTests; i++ {
		t.Run(fmt.Sprintf("i=%d", i), func(t *testing.T) {
			if _, err := dbconn.Global.Exec(`TRUNCATE store_counts_test`); err != nil {
				t.Fatalf("unexpected error truncating table: %s", err)
			}

			// Make `NumSavepointTests` "nested transactions", where the transaction
			// or savepoint at index `i` will be rolled back. Note that all of the
			// actions in any savepoint after this index will also be rolled back.
			recurSavepoints(t, testStore(), NumSavepointTests, i)

			expected := map[int]int{}
			for j := NumSavepointTests; j > i; j-- {
				expected[j] = j * 2
			}
			assertCounts(t, dbconn.Global, expected)
		})
	}
}

func recurSavepoints(t *testing.T, store *Store, index, rollbackAt int) {
	if index == 0 {
		return
	}

	tx, err := store.Transact(context.Background())
	if err != nil {
		t.Fatalf("unexpected error creating transaction: %s", err)
	}
	defer func() {
		var doneErr error
		if index == rollbackAt {
			doneErr = errors.New("rollback")
		}

		if err := tx.Done(doneErr); err != doneErr {
			t.Fatalf("unexpected error closing transaction. want=%q have=%q", doneErr, err)
		}
	}()

	if err := tx.Exec(context.Background(), sqlf.Sprintf(`INSERT INTO store_counts_test VALUES (%s, %s)`, index, index*2)); err != nil {
		t.Fatalf("unexpected error inserting count: %s", err)
	}

	recurSavepoints(t, tx, index-1, rollbackAt)
}

func testStore() *Store {
	return NewWithDB(dbconn.Global, sql.TxOptions{})
}

func assertCounts(t *testing.T, db dbutil.DB, expectedCounts map[int]int) {
	rows, err := db.QueryContext(context.Background(), `SELECT id, value FROM store_counts_test`)
	if err != nil {
		t.Fatalf("unexpected error querying counts: %s", err)
	}
	defer func() { _ = CloseRows(rows, nil) }()

	counts := map[int]int{}
	for rows.Next() {
		var id, count int
		if err := rows.Scan(&id, &count); err != nil {
			t.Fatalf("unexpected error scanning row: %s", err)
		}

		counts[id] = count
	}

	if diff := cmp.Diff(expectedCounts, counts); diff != "" {
		t.Errorf("unexpected counts args (-want +got):\n%s", diff)
	}
}

// setupStoreTest creates a table used only for testing. This table does not need to be truncated
// between tests as all tables in the test database are truncated by SetupGlobalTestDB.
func setupStoreTest(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)

	if _, err := dbconn.Global.Exec(`
		CREATE TABLE IF NOT EXISTS store_counts_test (
			id    integer NOT NULL,
			value integer NOT NULL
		)
	`); err != nil {
		t.Fatalf("unexpected error creating test table: %s", err)
	}
}
