package dbworker

import (
	"context"
	"errors"

	"github.com/keegancsmith/sqlf"
	"github.com/sourcegraph/sourcegraph/internal/workerutil"
	"github.com/sourcegraph/sourcegraph/internal/workerutil/dbworker/store"
)

// storeShim converts a store.Store into a workerutil.Store.
type storeShim struct {
	store.Store
}

var _ workerutil.Store = &storeShim{}

// newStoreShim wraps the given store in a shim.
func newStoreShim(store store.Store) workerutil.Store {
	if store == nil {
		return nil
	}

	return &storeShim{Store: store}
}

// QueuedCount calls into the inner store.
func (s *storeShim) QueuedCount(ctx context.Context, extraArguments interface{}) (int, error) {
	conditions, err := convertArguments(extraArguments)
	if err != nil {
		return 0, err
	}

	return s.Store.QueuedCount(ctx, conditions)
}

// Dequeue calls into the inner store.
func (s *storeShim) Dequeue(ctx context.Context, extraArguments interface{}) (workerutil.Record, workerutil.Store, bool, error) {
	conditions, err := convertArguments(extraArguments)
	if err != nil {
		return nil, nil, false, err
	}

	record, tx, dequeued, err := s.Store.Dequeue(ctx, conditions)
	return record, newStoreShim(tx), dequeued, err
}

// ErrNotConditions occurs when a PreDequeue handler returns non-sql query extra arguments.
var ErrNotConditions = errors.New("expected slice of *sqlf.Query values")

// convertArguments converts the given interface value into a slice of *sqlf.Query values.
func convertArguments(v interface{}) ([]*sqlf.Query, error) {
	if v == nil {
		return nil, nil
	}

	if conditions, ok := v.([]*sqlf.Query); ok {
		return conditions, nil
	}

	return nil, ErrNotConditions
}
