package dbworker

import (
	"context"

	"github.com/sourcegraph/sourcegraph/internal/workerutil"
)

// handlerShim converts a Handler into a workerutil.Handler.
type handlerShim struct {
	Handler
}

var _ workerutil.Handler = &handlerShim{}
var _ workerutil.WithPreDequeue = &handlerShim{}
var _ workerutil.WithHooks = &handlerShim{}

// newHandlerShim wraps the given handler in a shim.
func newHandlerShim(handler Handler) workerutil.Handler {
	return &handlerShim{Handler: handler}
}

// Handle processes a single record.
func (s *handlerShim) Handle(ctx context.Context, store workerutil.Store, record workerutil.Record) error {
	return s.Handler.Handle(ctx, store.(*storeShim).Store, record)
}

// PreDequeue calls into the inner handler if it implements the HandlerWithPreDequeue interface.
func (s *handlerShim) PreDequeue(ctx context.Context) (dequeueable bool, extraDequeueArguments interface{}, err error) {
	if h, ok := s.Handler.(workerutil.WithPreDequeue); ok {
		return h.PreDequeue(ctx)
	}

	return true, nil, nil
}

// PreHandle calls into the inner handler if it implements the HandlerWithHooks interface.
func (s *handlerShim) PreHandle(ctx context.Context, record workerutil.Record) {
	if h, ok := s.Handler.(workerutil.WithHooks); ok {
		h.PreHandle(ctx, record)
	}
}

// PostHandle calls into the inner handler if it implements the HandlerWithHooks interface.
func (s *handlerShim) PostHandle(ctx context.Context, record workerutil.Record) {
	if h, ok := s.Handler.(workerutil.WithHooks); ok {
		h.PostHandle(ctx, record)
	}
}
