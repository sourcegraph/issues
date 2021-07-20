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
		{"DBStore/SyncRateLimiters", testSyncRateLimiters},
		{"DBStore/UpsertRepos", testStoreUpsertRepos},
		{"DBStore/UpsertSources", testStoreUpsertSources},
		{"DBStore/EnqueueSyncJobs", testStoreEnqueueSyncJobs},
		{"DBStore/EnqueueSingleSyncJob", testStoreEnqueueSingleSyncJob},
		{"DBStore/ListExternalRepoSpecs", testStoreListExternalRepoSpecs},
		{"DBStore/SetClonedRepos", testStoreSetClonedRepos},
		{"DBStore/CountNotClonedRepos", testStoreCountNotClonedRepos},
		{"DBStore/Syncer/SyncWorker", testSyncWorkerPlumbing},

		{"DBStore/Syncer/Batch/Sync", testSyncerBatchSync},
		{"DBStore/Syncer/Streaming/Sync", testSyncerStreamingSync},

		{"DBStore/Syncer/Batch/SyncRepo", testBatchSyncRepo},
		{"DBStore/Syncer/Streaming/SyncRepo", testStreamingSyncRepo},

		{"DBStore/Syncer/Batch/Run", testBatchSyncRun},
		{"DBStore/Syncer/Streaming/Run", testStreamingSyncRun},

		{"DBStore/Syncer/Batch/MultipleServices", testBatchSyncerMultipleServices},
		{"DBStore/Syncer/Streaming/MultipleServices", testStreamingSyncerMultipleServices},

		{"DBStore/Syncer/Batch/OrphanedRepos", testBatchOrphanedRepo},
		{"DBStore/Syncer/Streaming/OrphanedRepos", testStreamingOrphanedRepo},

		{"DBStore/Syncer/Batch/UserAddedRepos", testBatchUserAddedRepos},
		{"DBStore/Syncer/Streaming/UserAddedRepos", testStreamingUserAddedRepos},

		{"DBStore/Syncer/Batch/DeleteExternalService", testBatchDeleteExternalService},
		{"DBStore/Syncer/Batch/DeleteExternalService", testStreamingDeleteExternalService},

		// We don't run streaming versions of these two tests because the behaviour is completely different, and it is
		// tested by the streaming NameConflictOnRename test. Since we sync one repo at a time, there's no "sorting" by
		// IDs to pick winners - we always treat the just now sourced repo as the winner, and delete the conflicting one.
		{"DBStore/Syncer/Batch/NameConflictDiscardOld", testNameOnConflictDiscardOld},
		{"DBStore/Syncer/Batch/NameConflictDiscardNew", testNameOnConflictDiscardNew},

		{"DBStore/Syncer/Batch/NameConflictOnRename", testBatchNameOnConflictOnRename},
		{"DBStore/Syncer/Streaming/NameConflictOnRename", testStreamingNameOnConflictOnRename},

		{"DBStore/Syncer/Batch/ConflictingSyncers", testConflictingBatchSyncers},
		{"DBStore/Syncer/Streaming/ConflictingSyncers", testConflictingStreamingSyncers},

		{"DBStore/Syncer/Batch/SyncRepoMaintainsOtherSources", testBatchSyncRepoMaintainsOtherSources},
		{"DBStore/Syncer/Streaming/SyncRepoMaintainsOtherSources", testStreamingSyncRepoMaintainsOtherSources},
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
