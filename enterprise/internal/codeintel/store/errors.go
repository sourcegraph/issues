package store

import "github.com/pkg/errors"

// ErrNotTransactable occurs when Transact is called on a store whose underlying
// store handle does not support beginning a transaction.
var ErrNotTransactable = errors.New("store: not transactable")

// ErrNoTransaction occurs when Savepoint or RollbackToSavepoint is called outside of a transaction.
var ErrNoTransaction = errors.New("store: not in a transaction")

// ErrDequeueTransaction occurs when Dequeue is called from inside a transaction.
var ErrDequeueTransaction = errors.New("unexpected transaction")

// ErrDequeueRace occurs when an upload selected for dequeue has been locked by another worker.
var ErrDequeueRace = errors.New("dequeue race")

// ErrNoSavepoint occurs when there is no savepont to rollback to.
var ErrNoSavepoint = errors.New("no savepoint defined")

// ErrUnknownRepository occurs when a repository does not exist.
var ErrUnknownRepository = errors.New("unknown repository")

// ErrIllegalLimit occurs when a limit is not strictly positive.
var ErrIllegalLimit = errors.New("illegal limit")
