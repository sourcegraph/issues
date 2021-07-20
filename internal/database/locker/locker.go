package locker

import (
	"context"
	"database/sql"

	"github.com/cockroachdb/errors"
	"github.com/hashicorp/go-multierror"
	"github.com/keegancsmith/sqlf"
	"github.com/segmentio/fasthash/fnv1"

	"github.com/sourcegraph/sourcegraph/internal/database/basestore"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
)

// Locker is a wrapper around a base store with methods that control advisory locks.
// A locker should be used when work needs to be coordinated with other remote services.
//
// For example, an advisory lock can be taken around an expensive calculation related to
// a particular repository to ensure that no other service is performing the same task.
type Locker struct {
	*basestore.Store
	namespace int32
}

// NewWithDB creates a new Locker with the given namespace.
func NewWithDB(db dbutil.DB, namespace string) *Locker {
	return &Locker{
		Store:     basestore.NewWithDB(db, sql.TxOptions{}),
		namespace: int32(fnv1.HashString32(namespace)),
	}
}

func (l *Locker) With(other basestore.ShareableStore) *Locker {
	return &Locker{
		Store:     l.Store.With(other),
		namespace: l.namespace,
	}
}

func (l *Locker) Transact(ctx context.Context) (*Locker, error) {
	txBase, err := l.Store.Transact(ctx)
	if err != nil {
		return nil, err
	}

	return &Locker{
		Store:     txBase,
		namespace: l.namespace,
	}, nil
}

// UnlockFunc unlocks the advisory lock taken by a successful call to Lock. If an error
// occurs during unlock, the error is added to the resulting error value.
type UnlockFunc func(err error) error

// ErrTransaction occurs when Lock is called inside of a transaction.
var ErrTransaction = errors.New("locker: in a transaction")

// Lock creates a transactional store and calls its Lock method. This method expects that
// the locker is outside of a transaction. The transaction's lifetime is linked to the lock,
// so the internal locker will commit or rollback once the lock is released.
func (l *Locker) Lock(ctx context.Context, key int, blocking bool) (locked bool, _ UnlockFunc, err error) {
	if l.InTransaction() {
		return false, nil, ErrTransaction
	}

	tx, err := l.Transact(ctx)
	if err != nil {
		return false, nil, err
	}
	defer func() {
		if !locked {
			// Catch failure cases
			err = tx.Done(err)
		}
	}()

	locked, unlock, err := tx.LockInTransaction(ctx, key, blocking)
	if err != nil || !locked {
		return false, nil, err
	}

	return true, func(err error) error { return tx.Done(unlock(err)) }, nil
}

// ErrNoTransaction occurs when LockInTransaction is called outside of a transaction.
var ErrNoTransaction = errors.New("locker: not in a transaction")

// LockInTransaction attempts to take an advisory lock on the given key. If successful, this method
// will return a true-valued flag along with a function that must be called to release the lock. This
// method assumes that the locker is currently in a transaction.
func (l *Locker) LockInTransaction(ctx context.Context, key int, blocking bool) (locked bool, _ UnlockFunc, err error) {
	if !l.InTransaction() {
		return false, nil, ErrNoTransaction
	}

	if blocking {
		locked, err = l.selectAdvisoryLock(ctx, key)
	} else {
		locked, err = l.selectTryAdvisoryLock(ctx, key)
	}

	if err != nil || !locked {
		return false, nil, err
	}

	unlock := func(err error) error {
		if unlockErr := l.unlock(context.Background(), key); unlockErr != nil {
			err = multierror.Append(err, unlockErr)
		}

		return err
	}

	return true, unlock, nil
}

// selectAdvisoryLock blocks until an advisory lock is taken on the given key.
func (l *Locker) selectAdvisoryLock(ctx context.Context, key int) (bool, error) {
	err := l.Store.Exec(ctx, sqlf.Sprintf(selectAdvisoryLockQuery, l.namespace, key))
	if err != nil {
		return false, err
	}
	return true, nil
}

const selectAdvisoryLockQuery = `
-- source: internal/database/locker/locker.go:selectAdvisoryLock
SELECT pg_advisory_lock(%s, %s)
`

// selectTryAdvisoryLock attempts to take an advisory lock on the given key. Returns true
// on success and false on failure.
func (l *Locker) selectTryAdvisoryLock(ctx context.Context, key int) (bool, error) {
	ok, _, err := basestore.ScanFirstBool(l.Store.Query(ctx, sqlf.Sprintf(selectTryAdvisoryLockQuery, l.namespace, key)))
	if err != nil || !ok {
		return false, err
	}

	return true, nil
}

const selectTryAdvisoryLockQuery = `
-- source: internal/database/locker/locker.go:selectTryAdvisoryLock
SELECT pg_try_advisory_lock(%s, %s)
`

var ErrUnlock = errors.New("failed to unlock")

// unlock releases the advisory lock on the given key.
func (l *Locker) unlock(ctx context.Context, key int) error {
	ok, _, err := basestore.ScanFirstBool(l.Store.Query(ctx, sqlf.Sprintf(unlockQuery, l.namespace, key)))
	if !ok {
		if err == nil {
			err = ErrUnlock
		}

		return err
	}

	return nil
}

const unlockQuery = `
-- source: internal/database/locker/locker.go:unlock
SELECT pg_advisory_unlock(%s, %s)
`
