package dbstore

import (
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
	"github.com/sourcegraph/sourcegraph/internal/observation"
)

func testStore(db dbutil.DB) *Store {
	return NewWithDB(db, &observation.TestContext)
}
