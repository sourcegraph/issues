package store

import "errors"

// ErrDequeueTransaction occurs when Dequeue is called from inside a transaction.
var ErrDequeueTransaction = errors.New("unexpected transaction")

// ErrDequeueRace occurs when a record selected for dequeue has been locked by another worker.
var ErrDequeueRace = errors.New("dequeue race")

// ErrNoRecord occurs when a record cannot be selected after it has been locked.
var ErrNoRecord = errors.New("locked record not found")
