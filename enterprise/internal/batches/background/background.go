package background

import (
	"context"

	"github.com/inconshreveable/log15"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/sourcegraph/sourcegraph/enterprise/internal/batches/scheduler"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/batches/sources"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/batches/store"
	"github.com/sourcegraph/sourcegraph/internal/gitserver"
	"github.com/sourcegraph/sourcegraph/internal/goroutine"
	"github.com/sourcegraph/sourcegraph/internal/httpcli"
	"github.com/sourcegraph/sourcegraph/internal/observation"
	"github.com/sourcegraph/sourcegraph/internal/trace"
)

func Routines(ctx context.Context, batchesStore *store.Store, cf *httpcli.Factory) []goroutine.BackgroundRoutine {
	sourcer := sources.NewSourcer(cf)
	observationContext := &observation.Context{
		Logger:     log15.Root(),
		Tracer:     &trace.Tracer{Tracer: opentracing.GlobalTracer()},
		Registerer: prometheus.DefaultRegisterer,
	}
	metrics := newMetrics(observationContext)

	routines := []goroutine.BackgroundRoutine{
		newReconcilerWorker(ctx, batchesStore, gitserver.DefaultClient, sourcer, metrics),
		newReconcilerWorkerResetter(batchesStore, metrics),

		newSpecExpireWorker(ctx, batchesStore),

		scheduler.NewScheduler(ctx, batchesStore),

		newBulkOperationWorker(ctx, batchesStore, sourcer, metrics),
		newBulkOperationWorkerResetter(batchesStore, metrics),

		// newBatchSpecExecutionResetter(batchesStore, observationContext, metrics),
	}
	return routines
}
