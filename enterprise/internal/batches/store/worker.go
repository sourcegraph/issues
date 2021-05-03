package store

import (
	"context"
	"encoding/hex"
	"math/rand"

	"github.com/keegancsmith/sqlf"
	"github.com/pkg/errors"

	btypes "github.com/sourcegraph/sourcegraph/enterprise/internal/batches/types"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
)

func (s *Store) CreateWorker(ctx context.Context, worker *btypes.Worker) error {
	if worker.CreatedAt.IsZero() {
		worker.CreatedAt = s.now()
	}
	if worker.UpdatedAt.IsZero() {
		worker.UpdatedAt = worker.CreatedAt
	}
	if worker.LastSeenAt.IsZero() {
		worker.LastSeenAt = worker.UpdatedAt
	}

	if worker.Token == nil {
		// This logic is shamelessly lifted from AccessTokenStore.Create().
		var b [20]byte
		if _, err := rand.Read(b[:]); err != nil {
			return errors.Wrap(err, "reading random data to create token")
		}

		// We'll apply a prefix to make it easier to detect leaked tokens in the
		// future.
		token := "sourcegraph-batch-change-worker-" + hex.EncodeToString(b[:])
		worker.Token = &token
	}

	q := createWorkerQuery(worker)
	return s.query(ctx, q, func(sc scanner) error {
		return scanWorker(worker, sc)
	})
}

const createWorkerQueryFmtstr = `
-- source: enterprise/internal/batches/store/worker.go:CreateWorker
INSERT INTO batch_worker
	(
		name,
		token,
		created_at,
		updated_at,
		last_seen_at
	)
VALUES
	(%s, %s, %s, %s, %s)
RETURNING
	%s
`

func createWorkerQuery(worker *btypes.Worker) *sqlf.Query {
	return sqlf.Sprintf(
		createWorkerQueryFmtstr,
		worker.Name,
		dbutil.NullString{S: worker.Token},
		worker.CreatedAt,
		worker.UpdatedAt,
		worker.LastSeenAt,
		sqlf.Join(workerColumns, ","),
	)
}

var workerColumns = []*sqlf.Query{
	sqlf.Sprintf("id"),
	sqlf.Sprintf("name"),
	sqlf.Sprintf("token"),
	sqlf.Sprintf("created_at"),
	sqlf.Sprintf("updated_at"),
	sqlf.Sprintf("last_seen_at"),
}

func scanWorker(worker *btypes.Worker, sc scanner) error {
	var token string

	if err := sc.Scan(
		&worker.ID,
		&worker.Name,
		dbutil.NullString{S: &token},
		&worker.CreatedAt,
		&worker.UpdatedAt,
		&worker.LastSeenAt,
	); err != nil {
		return err
	}

	if token == "" {
		worker.Token = nil
	} else {
		worker.Token = &token
	}

	return nil
}
