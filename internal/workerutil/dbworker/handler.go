package dbworker

import (
	"context"

	"github.com/sourcegraph/sourcegraph/internal/workerutil"
	"github.com/sourcegraph/sourcegraph/internal/workerutil/dbworker/store"
)

// Handler is a version of workerutil.Handler that refines the store type.
type Handler interface {
	// Handle processes a single record. The store provided by this method a store backed
	// by the transaction that is locking the given record. If use of a database is necessary
	// within this handler, other stores should take the underlying handler to keep work
	// within the same transaction.
	//
	//     func (h *handler) Handle(ctx context.Context, tx dbworker.Store, record workerutil.Record) error {
	//         myStore := h.myStore.With(tx) // combine store handles
	//         myRecord := record.(MyType)   // convert type of record
	//         // do processing ...
	//         return nil
	//     }
	Handle(ctx context.Context, store store.Store, record workerutil.Record) error
}

// HandlerFunc is a function version of the Handler interface.
type HandlerFunc func(ctx context.Context, store store.Store, record workerutil.Record) error

// Handle processes a single record. See the Handler interface for additional details.
func (f HandlerFunc) Handle(ctx context.Context, store store.Store, record workerutil.Record) error {
	return f(ctx, store, record)
}
