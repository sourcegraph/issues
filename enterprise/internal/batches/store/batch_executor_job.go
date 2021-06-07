package store

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/keegancsmith/sqlf"
	"github.com/lib/pq"

	btypes "github.com/sourcegraph/sourcegraph/enterprise/internal/batches/types"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/executor"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
)

func (s *Store) CreateBatchExecutorJob(ctx context.Context, job executor.Job) (*btypes.BatchExecutorJob, error) {
	bej := &btypes.BatchExecutorJob{
		CreatedAt: s.now(),
		UpdatedAt: s.now(),
		Job:       job,
	}

	q, err := createBatchExecutorJobQuery(bej)
	if err != nil {
		return nil, err
	}

	if err := s.query(ctx, q, func(sc scanner) error {
		return scanBatchExecutorJob(bej, sc)
	}); err != nil {
		return nil, err
	}
	return bej, nil
}

const createBatchExecutorJobQueryFmtstr = `
-- source: enterprise/internal/batches/store/batch_executor_job.go:CreateBatchExecutorJob
INSERT INTO batch_executor_jobs (
	created_at,
	updated_at,
	job
)
VALUES
	(%s, %s, %s)
RETURNING
	%s
`

func createBatchExecutorJobQuery(bej *btypes.BatchExecutorJob) (*sqlf.Query, error) {
	job, err := json.Marshal(bej.Job)
	if err != nil {
		return nil, err
	}

	return sqlf.Sprintf(
		createBatchExecutorJobQueryFmtstr,
		bej.CreatedAt,
		bej.UpdatedAt,
		job,
		sqlf.Join(BatchExecutorJobColumns, ","),
	), nil
}

func ScanFirstBatchExecutorJob(rows *sql.Rows, err error) (*btypes.BatchExecutorJob, bool, error) {
	if err != nil {
		return nil, false, err
	}

	bejes, err := scanBatchExecutorJobs(rows)
	if err != nil || len(bejes) == 0 {
		return &btypes.BatchExecutorJob{}, false, err
	}
	return bejes[0], true, nil
}

func scanBatchExecutorJobs(rows *sql.Rows) ([]*btypes.BatchExecutorJob, error) {
	var bejes []*btypes.BatchExecutorJob

	return bejes, scanAll(rows, func(sc scanner) error {
		var bej btypes.BatchExecutorJob
		if err := scanBatchExecutorJob(&bej, sc); err != nil {
			return err
		}
		bejes = append(bejes, &bej)
		return nil
	})
}

func scanBatchExecutorJob(bej *btypes.BatchExecutorJob, sc scanner) error {
	var job []byte

	if err := sc.Scan(
		&bej.ID,
		&bej.State,
		&dbutil.NullString{S: &bej.FailureMessage},
		&dbutil.NullTime{Time: &bej.StartedAt},
		&dbutil.NullTime{Time: &bej.FinishedAt},
		&dbutil.NullTime{Time: &bej.ProcessAfter},
		&bej.NumResets,
		&bej.NumFailures,
		pq.Array(&bej.ExecutionLogs),
		&bej.CreatedAt,
		&bej.UpdatedAt,
		&job,
	); err != nil {
		return err
	}

	return json.Unmarshal(job, &bej.Job)
}

var BatchExecutorJobColumns = []*sqlf.Query{
	sqlf.Sprintf("id"),
	sqlf.Sprintf("state"),
	sqlf.Sprintf("failure_message"),
	sqlf.Sprintf("started_at"),
	sqlf.Sprintf("finished_at"),
	sqlf.Sprintf("process_after"),
	sqlf.Sprintf("num_resets"),
	sqlf.Sprintf("num_failures"),
	sqlf.Sprintf("execution_logs"),
	sqlf.Sprintf("created_at"),
	sqlf.Sprintf("updated_at"),
	sqlf.Sprintf("job"),
}
