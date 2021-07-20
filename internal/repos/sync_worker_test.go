package repos_test

import (
	"context"
	"testing"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/keegancsmith/sqlf"

	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/repos"
	"github.com/sourcegraph/sourcegraph/internal/types"
	"github.com/sourcegraph/sourcegraph/internal/workerutil"
)

func testSyncWorkerPlumbing(repoStore *repos.Store) func(t *testing.T) {
	return func(t *testing.T) {
		ctx := context.Background()
		testSvc := &types.ExternalService{
			Kind:        extsvc.KindGitHub,
			DisplayName: "TestService",
			Config:      "{}",
		}

		// Create external service
		err := repoStore.ExternalServiceStore.Upsert(ctx, testSvc)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("Test service created, ID: %d", testSvc.ID)

		// Add item to queue
		result, err := repoStore.ExecResult(ctx, sqlf.Sprintf(`insert into external_service_sync_jobs (external_service_id) values (%s);`, testSvc.ID))
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
		worker, resetter := repos.NewSyncWorker(ctx, repoStore.Handle().DB(), h, repos.SyncWorkerOptions{
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

type fakeRepoSyncHandler struct {
	jobChan chan *repos.SyncJob
}

func (h *fakeRepoSyncHandler) Handle(ctx context.Context, record workerutil.Record) error {
	sj, ok := record.(*repos.SyncJob)
	if !ok {
		return errors.Errorf("expected repos.SyncJob, got %T", record)
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case h.jobChan <- sj:
		return nil
	}
}
