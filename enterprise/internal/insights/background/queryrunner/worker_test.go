package queryrunner

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/hexops/autogold"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/internal/database/basestore"
	"github.com/sourcegraph/sourcegraph/internal/database/dbtest"
)

// TestJobQueue tests that EnqueueJob and dequeueJob work mutually to transfer jobs to/from the
// database.
func TestJobQueue(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	t.Parallel()

	ctx := backend.WithAuthzBypass(context.Background())

	mainAppDB := dbtest.NewDB(t, "")
	workerBaseStore := basestore.NewWithDB(mainAppDB, sql.TxOptions{})

	// Check we get no dequeued job first.
	recordID := 0
	job, err := dequeueJob(ctx, workerBaseStore, recordID)
	autogold.Want("0", (*Job)(nil)).Equal(t, job)
	autogold.Want("1", "expected 1 job to dequeue, found 0").Equal(t, fmt.Sprint(err))

	// Now enqueue two jobs.
	firstJobID, err := EnqueueJob(ctx, workerBaseStore, &Job{
		SeriesID:    "job 1",
		SearchQuery: "our search 1",
	})
	if err != nil {
		t.Fatal(err)
	}
	secondJobID, err := EnqueueJob(ctx, workerBaseStore, &Job{
		SeriesID:    "job 2",
		SearchQuery: "our search 2",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Check the information we care about got transferred properly.
	firstJob, err := dequeueJob(ctx, workerBaseStore, firstJobID)
	autogold.Want("2", &Job{
		SeriesID: "job 1", SearchQuery: "our search 1",
		ID: 1,
	}).Equal(t, firstJob)
	autogold.Want("3", "<nil>").Equal(t, fmt.Sprint(err))
	secondJob, err := dequeueJob(ctx, workerBaseStore, secondJobID)
	autogold.Want("4", &Job{
		SeriesID: "job 2", SearchQuery: "our search 2",
		ID: 2,
	}).Equal(t, secondJob)
	autogold.Want("5", "<nil>").Equal(t, fmt.Sprint(err))
}
