package campaigns

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/sourcegraph/sourcegraph/cmd/repo-updater/repos"
	"github.com/sourcegraph/sourcegraph/internal/db/dbtest"
)

type clock interface {
	now() time.Time
	add(time.Duration) time.Time
}

type testClock struct {
	t time.Time
}

func (c *testClock) now() time.Time                { return c.t }
func (c *testClock) add(d time.Duration) time.Time { c.t = c.t.Add(d); return c.t }

type storeTestFunc func(*testing.T, context.Context, *Store, repos.Store, clock)

// storeTest converts a storeTestFunc into a func(*testing.T) in which all
// dependencies are set up and injected into the storeTestFunc.
func storeTest(db *sql.DB, f storeTestFunc) func(*testing.T) {
	return func(t *testing.T) {
		c := &testClock{t: time.Now().UTC().Truncate(time.Microsecond)}

		// Store tests all run in a transaction that's rolled back at the end
		// of the tests, so that foreign key constraints can be deferred and we
		// don't need to insert a lot of dependencies into the DB (users,
		// repos, ...) to setup the tests.
		tx := dbtest.NewTx(t, db)
		s := NewStoreWithClock(tx, c.now)

		rs := repos.NewDBStore(db, sql.TxOptions{})

		f(t, context.Background(), s, rs, c)
	}
}
