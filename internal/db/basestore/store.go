package basestore

import (
	"context"
	"database/sql"

	"github.com/keegancsmith/sqlf"
	"github.com/sourcegraph/sourcegraph/internal/db/dbutil"
)

// Store is an abstract Postgres-backed data access layer. Instances of this struct
// should not be used directly, but should be used compositionally by other stores
// that implement logic specific to a domain.
//
// The following is a minimal example of decorating the base store that preserves
// the correct behavior of the underlying base store. Note that `With` and `Transact`
// must be re-defined in the outer layer in order to create a useful return value.
// Failure to re-define these methods will result in `With` and `Transact` methods that
// return a modified base store with no methods from the outer layer. All other methods
// of the base store are available on the outer layer without needing to be re-defined.
//
//     type SprocketStore struct {
//         *basestore.Store
//     }
//
//     func NewWithDB(db dbutil.DB) *SprocketStore {
//         return &SprocketStore{Store: basestore.NewWithDB(db, sql.TxOptions{})}
//     }
//
//     func (s *SprocketStore) With(other basestore.ShareableStore) *SprocketStore {
//         return &SprocketStore{Store: s.Store.With(other)}
//     }
//
//     func (s *SprocketStore) Transact(ctx context.Context) (*SprocketStore, error) {
//         txBase, err := s.Store.Transact(ctx)
//         return &SprocketStore{Store: txBase}, err
//     }
type Store struct {
	handle *TransactableHandle
}

// ShareableStore is implemented by stores to explicitly allow distinct store instances
// to reference the store's underlying handle. This is used to share transactions between
// multiple stores. See `Store.With` for additional details.
type ShareableStore interface {
	// Handle returns the underlying transactable database handle.
	Handle() *TransactableHandle
}

var _ ShareableStore = &Store{}

// New returns a new base store connected to the given dsn (data store name).
func New(postgresDSN, app string, txOptions sql.TxOptions) (*Store, error) {
	handle, err := NewHandle(postgresDSN, app, txOptions)
	if err != nil {
		return nil, err
	}

	return NewWithHandle(handle), nil
}

// NewHandleWithDB returns a new base store connected to the given connection.
func NewWithDB(db dbutil.DB, txOptions sql.TxOptions) *Store {
	return NewWithHandle(NewHandleWithDB(db, txOptions))
}

// NewWithHandle returns a new base store using the given database handle.
func NewWithHandle(handle *TransactableHandle) *Store {
	return &Store{handle: handle}
}

// Handle returns the underlying transactable database handle.
func (s *Store) Handle() *TransactableHandle {
	return s.handle
}

// With creates a new store with the underlying database handle from the given store.
// This method should be used when two distinct store instances need to perform an
// operation within the same shared transaction.
//
//     txn1 := store1.Transact(ctx) // Creates a transaction
//     txn2 := store2.With(txn1)    // References the same transaction
//
//     txn1.A(ctx) // Occurs within shared transaction
//     txn2.B(ctx) // Occurs within shared transaction
//     txn1.Done() // closes shared transaction
//
// Note that once a handle is shared between two stores, committing or rolling back
// a transaction will affect the handle of both stores. Most notably, two stores that
// share the same handle are unable to begin independent transactions.
func (s *Store) With(other ShareableStore) *Store {
	return &Store{handle: other.Handle()}
}

// Query performs QueryContext on the underlying connection.
func (s *Store) Query(ctx context.Context, query *sqlf.Query) (*sql.Rows, error) {
	return s.handle.db.QueryContext(ctx, query.Query(sqlf.PostgresBindVar), query.Args()...)
}

// Exec performs a query and throws away the result.
func (s *Store) Exec(ctx context.Context, query *sqlf.Query) error {
	rows, err := s.Query(ctx, query)
	if err != nil {
		return err
	}
	return CloseRows(rows, nil)
}

// InTransaction returns true if the underlying database handle is in a transaction.
func (s *Store) InTransaction() bool {
	return s.handle.InTransaction()
}

// Transact returns a new store whose methods operate within the context of a new transaction
// or a new savepoint. This method will return an error if the underlying connection cannot be
// interface upgraded to a TxBeginner.
func (s *Store) Transact(ctx context.Context) (*Store, error) {
	handle, err := s.handle.Transact(ctx)
	if err != nil {
		return nil, err
	}

	return &Store{handle: handle}, nil
}

// Done performs a commit or rollback of the underlying transaction/savepoint depending
// on the value of the error parameter. The resulting error value is a multierror containing
// the error parameter along with any error that occurs during commit or rollback of the
// transaction/savepoint. If the store does not wrap a transaction the original error value
// is returned unchanged.
func (s *Store) Done(err error) error {
	return s.handle.Done(err)
}
