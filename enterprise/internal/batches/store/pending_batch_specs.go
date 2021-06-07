package store

import (
	"context"
	"database/sql"

	"github.com/keegancsmith/sqlf"

	btypes "github.com/sourcegraph/sourcegraph/enterprise/internal/batches/types"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
)

func (s *Store) CreatePendingBatchSpec(ctx context.Context, spec string, userID int32) (*btypes.PendingBatchSpec, error) {
	pbs := &btypes.PendingBatchSpec{
		CreatedAt:     s.now(),
		UpdatedAt:     s.now(),
		CreatorUserID: userID,
		Spec:          spec,
	}

	q := createPendingBatchSpecQuery(pbs)
	if err := s.query(ctx, q, func(sc scanner) error {
		return scanPendingBatchSpec(pbs, sc)
	}); err != nil {
		return nil, err
	}
	return pbs, nil
}

const createPendingBatchSpecQueryFmtstr = `
-- source: enterprise/internal/batches/store/pending_batch_specs.go:CreatePendingBatchSpec
INSERT INTO pending_batch_specs (
	created_at,
	updated_at,
	creator_user_id,
	spec
)
VALUES
	(%s, %s, %s, %s)
RETURNING
	%s
`

func createPendingBatchSpecQuery(pbs *btypes.PendingBatchSpec) *sqlf.Query {
	return sqlf.Sprintf(
		createPendingBatchSpecQueryFmtstr,
		pbs.CreatedAt,
		pbs.UpdatedAt,
		pbs.CreatorUserID,
		pbs.Spec,
		sqlf.Join(PendingBatchSpecColumns, ","),
	)
}

func ScanFirstPendingBatchSpec(rows *sql.Rows, err error) (*btypes.PendingBatchSpec, bool, error) {
	if err != nil {
		return nil, false, err
	}

	pbses, err := scanPendingBatchSpecs(rows)
	if err != nil || len(pbses) == 0 {
		return &btypes.PendingBatchSpec{}, false, err
	}
	return pbses[0], true, nil
}

func scanPendingBatchSpecs(rows *sql.Rows) ([]*btypes.PendingBatchSpec, error) {
	var pbses []*btypes.PendingBatchSpec

	return pbses, scanAll(rows, func(sc scanner) error {
		var pbs btypes.PendingBatchSpec
		if err := scanPendingBatchSpec(&pbs, sc); err != nil {
			return err
		}
		pbses = append(pbses, &pbs)
		return nil
	})
}

var PendingBatchSpecColumns = []*sqlf.Query{
	sqlf.Sprintf("id"),
	sqlf.Sprintf("state"),
	sqlf.Sprintf("failure_message"),
	sqlf.Sprintf("started_at"),
	sqlf.Sprintf("finished_at"),
	sqlf.Sprintf("process_after"),
	sqlf.Sprintf("num_resets"),
	sqlf.Sprintf("num_failures"),
	sqlf.Sprintf("created_at"),
	sqlf.Sprintf("updated_at"),
	sqlf.Sprintf("creator_user_id"),
	sqlf.Sprintf("spec"),
}

func scanPendingBatchSpec(pbs *btypes.PendingBatchSpec, sc scanner) error {
	return sc.Scan(
		&pbs.ID,
		&pbs.State,
		&dbutil.NullString{S: &pbs.FailureMessage},
		&dbutil.NullTime{Time: &pbs.StartedAt},
		&dbutil.NullTime{Time: &pbs.FinishedAt},
		&dbutil.NullTime{Time: &pbs.ProcessAfter},
		&pbs.NumResets,
		&pbs.NumFailures,
		&pbs.CreatedAt,
		&pbs.UpdatedAt,
		&pbs.CreatorUserID,
		&pbs.Spec,
	)
}
