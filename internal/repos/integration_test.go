package repos_test

import (
	"database/sql"
	"flag"
	"testing"

	"github.com/cockroachdb/errors"
	"github.com/inconshreveable/log15"
	"github.com/opentracing/opentracing-go"

	"github.com/sourcegraph/sourcegraph/internal/database/dbconn"
	"github.com/sourcegraph/sourcegraph/internal/database/dbtest"
	"github.com/sourcegraph/sourcegraph/internal/repos"
	"github.com/sourcegraph/sourcegraph/internal/trace"
)

// This error is passed to txstore.Done in order to always
// roll-back the transaction a test case executes in.
// This is meant to ensure each test case has a clean slate.
var errRollback = errors.New("tx: rollback")

var dsn = flag.String("dsn", "", "Database connection string to use in integration tests")

func TestIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	t.Parallel()

	for _, tc := range []struct {
		name string
		test func(*repos.Store) func(*testing.T)
	}{
		{"SyncRateLimiters", testSyncRateLimiters},
		{"EnqueueSyncJobs", testStoreEnqueueSyncJobs},
		{"EnqueueSingleSyncJob", testStoreEnqueueSingleSyncJob},
		{"Syncer/SyncWorker", testSyncWorkerPlumbing},
		{"Syncer/Sync", testSyncerSync},
		{"Syncer/SyncRepo", testSyncRepo},
		{"Syncer/Run", testSyncRun},
		{"Syncer/MultipleServices", testSyncerMultipleServices},
		{"Syncer/OrphanedRepos", testOrphanedRepo},
		{"Syncer/UserAddedRepos", testUserAddedRepos},
		{"Syncer/NameConflictOnRename", testNameOnConflictOnRename},
		{"Syncer/ConflictingSyncers", testConflictingSyncers},
		{"Syncer/SyncRepoMaintainsOtherSources", testSyncRepoMaintainsOtherSources},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := dbtest.NewDB(t, *dsn)
			dbconn.Global = db

			store := repos.NewStore(db, sql.TxOptions{Isolation: sql.LevelReadCommitted})

			lg := log15.New()
			lg.SetHandler(log15.DiscardHandler())

			store.Log = lg
			store.Metrics = repos.NewStoreMetrics()
			store.Tracer = trace.Tracer{Tracer: opentracing.GlobalTracer()}

			t.Cleanup(func() { dbconn.Global = nil })

			tc.test(store)(t)
		})
	}
}
