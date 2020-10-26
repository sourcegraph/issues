package indexer

import (
	"context"
	"time"

	"github.com/google/uuid"
	indexmanager "github.com/sourcegraph/sourcegraph/enterprise/cmd/precise-code-intel-indexer-vm/internal/index_manager"
	queue "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/queue/client"
	"github.com/sourcegraph/sourcegraph/internal/workerutil"
)

type IndexerOptions struct {
	NumIndexers    int
	Interval       time.Duration
	Metrics        IndexerMetrics
	HandlerOptions HandlerOptions
}

func NewIndexer(ctx context.Context, queueClient queue.Client, indexManager *indexmanager.Manager, options IndexerOptions) *workerutil.Worker {
	handler := &Handler{
		queueClient:   queueClient,
		indexManager:  indexManager,
		newCommander:  NewDefaultCommander,
		options:       options.HandlerOptions,
		uuidGenerator: uuid.NewRandom,
	}

	workerMetrics := workerutil.WorkerMetrics{
		HandleOperation: options.Metrics.ProcessOperation,
	}

	return workerutil.NewWorker(ctx, &storeShim{queueClient}, handler, workerutil.WorkerOptions{
		NumHandlers: options.NumIndexers,
		Interval:    options.Interval,
		Metrics:     workerMetrics,
	})
}
