package workerutil

import (
	"context"
)

// Handler is the configurable consumer within a worker. Types that conform to this
// interface may also optionally conform to the PreDequeuer, PreHandler, and PostHandler
// interfaces to further configure the behavior of the worker routine.
type Handler interface {
	// Handle processes a single record. The store provided by this method is the store
	// returned from the Dequeue method returning the associated record.
	Handle(ctx context.Context, store Store, record Record) error
}

// WithPreDequeue is an extension of the Handler interface.
type WithPreDequeue interface {
	// PreDequeue is called, if implemented, directly before a call to the store's Dequeue method.
	// If this method returns false, then the current worker iteration is skipped and the next iteration
	// will begin after waiting for the configured polling interval. Any value returned by this method
	// will be used as additional parameters to the store's Dequeue method.
	PreDequeue(ctx context.Context) (dequeueable bool, extraDequeueArguments interface{}, err error)
}

// WithHooks is an extension of the Handler interface.
//
// Example use case:
// The processor for LSIF uploads has a maximum budget based on input size. PreHandle will subtract
// the input size (atomically) from the budget and PostHandle will restore the input size back to the
// budget. The PreDequeue hook is also implemented to supply additional SQL conditions that ensures no
// record with a larger input sizes than the current budget will be dequeued by the worker process.
type WithHooks interface {
	// PreHandle is called, if implemented, directly before a invoking the handler with the given
	// record. This method is invoked before starting a handler goroutine - therefore, any expensive
	// operations in this method will block the dequeue loop from proceeding.
	PreHandle(ctx context.Context, record Record)

	// PostHandle is called, if implemented, directly after the handler for the given record has
	// completed. This method is invoked inside the handler goroutine. Note that if PreHandle and
	// PostHandle both operate on shared data, that they will be operating on the data from different
	// goroutines and it is up to the caller to properly synchronize access to it.
	PostHandle(ctx context.Context, record Record)
}
