package repos_test

import (
	"database/sql"
	"flag"
	"testing"

	"github.com/inconshreveable/log15"
	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"

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

	db := dbtest.NewDB(t, *dsn)

	store := repos.NewStore(db, sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
	})

	lg := log15.New()
	lg.SetHandler(log15.DiscardHandler())

	store.Log = lg
	store.Metrics = repos.NewStoreMetrics()
	store.Tracer = trace.Tracer{Tracer: opentracing.GlobalTracer()}

	userID := insertTestUser(t, db)

	for _, tc := range []struct {
		name string
		test func(*testing.T, *repos.Store) func(*testing.T)
	}{
		{"DBStore/SyncRateLimiters", testSyncRateLimiters},
		{"DBStore/UpsertRepos", testStoreUpsertRepos},
		{"DBStore/UpsertSources", testStoreUpsertSources},
		{"DBStore/EnqueueSyncJobs", testStoreEnqueueSyncJobs(db, store)},
		{"DBStore/EnqueueSingleSyncJob", testStoreEnqueueSingleSyncJob(db)},
		{"DBStore/ListExternalRepoSpecs", testStoreListExternalRepoSpecs(db)},
		{"DBStore/SetClonedRepos", testStoreSetClonedRepos},
		{"DBStore/CountNotClonedRepos", testStoreCountNotClonedRepos},
		{"DBStore/Syncer/Sync", testSyncerSync},
		{"DBStore/Syncer/SyncRepo", testSyncRepo},
		{"DBStore/Syncer/SyncWorker", testSyncWorkerPlumbing(db)},
		{"DBStore/Syncer/Run", testSyncRun(db)},
		{"DBStore/Syncer/MultipleServices", testSyncer(db)},
		{"DBStore/Syncer/OrphanedRepos", testOrphanedRepo(db)},
		{"DBStore/Syncer/UserAddedRepos", testUserAddedRepos(db, userID)},
		{"DBStore/Syncer/DeleteExternalService", testDeleteExternalService(db)},
		{"DBStore/Syncer/NameConflictDiscardOld", testNameOnConflictDiscardOld(db)},
		{"DBStore/Syncer/NameConflictDiscardNew", testNameOnConflictDiscardNew(db)},
		{"DBStore/Syncer/NameConflictOnRename", testNameOnConflictOnRename(db)},
		{"DBStore/Syncer/ConflictingSyncers", testConflictingSyncers(db)},
		{"DBStore/Syncer/SyncRepoMaintainsOtherSources", testSyncRepoMaintainsOtherSources(db)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Cleanup(func() {
				if t.Failed() {
					return
				}
				if _, err := db.Exec(`
DELETE FROM external_service_sync_jobs;
DELETE FROM external_service_repos;
DELETE FROM external_services;
DELETE FROM repo;
`); err != nil {
					t.Fatalf("cleaning up external services failed: %v", err)
				}
			})

			tc.test(t, store)(t)
		})
	}
}

func insertTestUser(t *testing.T, db *sql.DB) (userID int32) {
	t.Helper()

	err := db.QueryRow("INSERT INTO users (username) VALUES ('bbs-admin') RETURNING id").Scan(&userID)
	if err != nil {
		t.Fatal(err)
	}

	return userID
}
