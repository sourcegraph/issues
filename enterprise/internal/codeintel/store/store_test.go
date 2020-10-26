package store

import (
	"database/sql"

	"github.com/sourcegraph/sourcegraph/internal/db/basestore"
	"github.com/sourcegraph/sourcegraph/internal/db/dbconn"
	"github.com/sourcegraph/sourcegraph/internal/db/dbtesting"
	"github.com/sourcegraph/sourcegraph/internal/observation"
)

func init() {
	dbtesting.DBNameSuffix = "codeintel"
}

func testStore() Store {
	// Wrap in observed, as that's how it's used in production
	return NewObserved(&store{Store: basestore.NewWithDB(dbconn.Global, sql.TxOptions{})}, &observation.TestContext)
}
