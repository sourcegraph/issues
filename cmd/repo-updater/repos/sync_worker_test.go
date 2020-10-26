package repos_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/sourcegraph/sourcegraph/cmd/repo-updater/repos"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/workerutil"
	dbws "github.com/sourcegraph/sourcegraph/internal/workerutil/dbworker/store"
)

func testSyncWorkerPlumbing(db *sql.DB) func(t *testing.T, repoStore repos.Store) func(t *testing.T) {
	// The reason we need two higher level functions is because we need access to the *sql.DB struct but also
	// need to satisfy the signature expected by integration tests.
	return func(t *testing.T, repoStore repos.Store) func(t *testing.T) {
		return func(t *testing.T) {
			ctx := context.Background()
			testSvc := &repos.ExternalService{
				Kind:        extsvc.KindGitHub,
				DisplayName: "TestService",
				Config:      "{}",
			}

			// Create external service
			err := repoStore.UpsertExternalServices(ctx, testSvc)
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("Test service created, ID: %d", testSvc.ID)

			// Add item to queue
			result, err := db.ExecContext(ctx, `insert into external_service_sync_jobs (external_service_id) values ($1);`, testSvc.ID)
			if err != nil {
				t.Fatal(err)
			}
			rowsAffected, err := result.RowsAffected()
			if err != nil {
				t.Fatal(err)
			}
			if rowsAffected != 1 {
				t.Fatalf("Expected 1 row to be affected, got %d", rowsAffected)
			}

			jobChan := make(chan *repos.SyncJob)

			h := &fakeRepoSyncHandler{
				jobChan: jobChan,
			}
			worker, resetter := repos.NewSyncWorker(ctx, db, h, repos.SyncWorkerOptions{
				NumHandlers:    1,
				WorkerInterval: 1 * time.Millisecond,
			})
			go worker.Start()
			go resetter.Start()

			// There is a race between the worker being stopped and the worker util
			// finalising the row which means that when running tests in verbose mode we'll
			// see "sql: transaction has already been committed or rolled back". These
			// errors can be ignored.
			defer worker.Stop()
			defer resetter.Stop()

			var job *repos.SyncJob
			select {
			case job = <-jobChan:
				t.Log("Job received")
			case <-time.After(5 * time.Second):
				t.Fatal("Timeout")
			}

			if job.ExternalServiceID != testSvc.ID {
				t.Fatalf("Expected %d, got %d", testSvc.ID, job.ExternalServiceID)
			}
		}
	}
}

type fakeRepoSyncHandler struct {
	jobChan chan *repos.SyncJob
}

func (h *fakeRepoSyncHandler) Handle(ctx context.Context, tx dbws.Store, record workerutil.Record) error {
	sj, ok := record.(*repos.SyncJob)
	if !ok {
		return fmt.Errorf("expected repos.SyncJob, got %T", record)
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case h.jobChan <- sj:
		return nil
	}
}
