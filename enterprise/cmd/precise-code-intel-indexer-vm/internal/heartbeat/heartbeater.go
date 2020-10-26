package heartbeat

import (
	"context"
	"time"

	"github.com/inconshreveable/log15"
	indexmanager "github.com/sourcegraph/sourcegraph/enterprise/cmd/precise-code-intel-indexer-vm/internal/index_manager"
	queue "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/queue/client"
	"github.com/sourcegraph/sourcegraph/internal/goroutine"
)

type Heartbeater struct {
	queueClient  queue.Client
	indexManager *indexmanager.Manager
}

var _ goroutine.Handler = &Heartbeater{}

type HeartbeaterOptions struct {
	Interval time.Duration
}

func NewHeartbeater(ctx context.Context, queueClient queue.Client, indexManager *indexmanager.Manager, options HeartbeaterOptions) goroutine.BackgroundRoutine {
	return goroutine.NewPeriodicGoroutine(context.Background(), options.Interval, &Heartbeater{
		queueClient:  queueClient,
		indexManager: indexManager,
	})
}

func (w *Heartbeater) Handle(ctx context.Context) error {
	return w.queueClient.Heartbeat(ctx, w.indexManager.GetIDs())
}

func (w *Heartbeater) HandleError(err error) {
	log15.Error("Failed to perform heartbeat", "err", err)
}
