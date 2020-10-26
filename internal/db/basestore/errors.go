package basestore

import "errors"

// ErrNotTransactable occurs when Transact is called on a Store instance whose underlying
// database handle does not support beginning a transaction.
var ErrNotTransactable = errors.New("store: not transactable")
