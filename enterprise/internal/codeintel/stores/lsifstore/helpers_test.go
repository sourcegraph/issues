package lsifstore

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"

	"github.com/sourcegraph/sourcegraph/internal/database/basestore"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
	"github.com/sourcegraph/sourcegraph/internal/observation"
)

const testBundleID = 39162

func populateTestStore(t testing.TB, db dbutil.DB) *Store {
	contents, err := os.ReadFile("./testdata/lsif-go@ad3507cb.sql")
	if err != nil {
		t.Fatalf("unexpected error reading testdata: %s", err)
	}

	dbh := basestore.NewHandleWithDB(db, sql.TxOptions{})
	tx, err := dbh.Transact(context.Background())
	if err != nil {
		t.Fatalf("unexpected error starting transaction: %s", err)
	}
	defer func() {
		if err := tx.Done(err); err != nil {
			t.Fatalf("unexpected error finishing transaction: %s", err)
		}
	}()

	for _, line := range strings.Split(string(contents), "\n") {
		if line == "" || strings.HasPrefix(line, "---") {
			continue
		}

		if _, err := tx.DB().ExecContext(context.Background(), line); err != nil {
			t.Fatalf("unexpected error loading database data: %s", err)
		}
	}

	return NewStore(db, &observation.TestContext)
}
