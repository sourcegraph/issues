package basestore

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/sourcegraph/sourcegraph/internal/db/dbutil"
)

const savepointQuery = "SAVEPOINT %s"
const commitSavepointQuery = "RELEASE %s"
const rollbackSavepointQuery = "ROLLBACK TO %s"

// savepoint is a small wrapper around committing/rolling back a "nested transaction".
// Each savepoint has an identifier unique to that connection and must be referenced by
// name on finalization. The transactional database handler takes care to finalize the
// savepoints in the same order they were created for a particular store.
type savepoint struct {
	db          dbutil.DB
	savepointID string
}

func newSavepoint(ctx context.Context, db dbutil.DB) (*savepoint, error) {
	savepointID, err := makeSavepointID()
	if err != nil {
		return nil, err
	}

	if _, err := db.ExecContext(ctx, fmt.Sprintf(savepointQuery, savepointID)); err != nil {
		return nil, err
	}

	return &savepoint{db, savepointID}, nil
}

func (s *savepoint) Commit() error {
	return s.apply(commitSavepointQuery)
}

func (s *savepoint) Rollback() error {
	return s.apply(rollbackSavepointQuery)
}

func (s *savepoint) apply(query string) error {
	_, err := s.db.ExecContext(context.Background(), fmt.Sprintf(query, s.savepointID))
	return err
}

func makeSavepointID() (string, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("sp_%s", strings.ReplaceAll(id.String(), "-", "_")), nil
}
