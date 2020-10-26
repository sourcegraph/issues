package dbworker

import (
	"context"

	"github.com/sourcegraph/sourcegraph/internal/workerutil"
	"github.com/sourcegraph/sourcegraph/internal/workerutil/dbworker/store"
)

func NewWorker(ctx context.Context, store store.Store, handler Handler, options workerutil.WorkerOptions) *workerutil.Worker {
	return workerutil.NewWorker(ctx, newStoreShim(store), newHandlerShim(handler), options)
}
