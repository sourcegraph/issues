package background

import (
	"fmt"

	"github.com/inconshreveable/log15"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/sourcegraph/sourcegraph/internal/observation"
	"github.com/sourcegraph/sourcegraph/internal/trace"
	"github.com/sourcegraph/sourcegraph/internal/workerutil"
	"github.com/sourcegraph/sourcegraph/internal/workerutil/dbworker"
)

type batchChangesMetrics struct {
	reconcilerWorkerMetrics               workerutil.WorkerMetrics
	bulkProcessorWorkerMetrics            workerutil.WorkerMetrics
	pendingBatchSpecWorkerMetrics         workerutil.WorkerMetrics
	reconcilerWorkerResetterMetrics       dbworker.ResetterMetrics
	bulkProcessorWorkerResetterMetrics    dbworker.ResetterMetrics
	pendingBatchSpecWorkerResetterMetrics dbworker.ResetterMetrics
}

func newMetrics() batchChangesMetrics {
	observationContext := &observation.Context{
		Logger:     log15.Root(),
		Tracer:     &trace.Tracer{Tracer: opentracing.GlobalTracer()},
		Registerer: prometheus.DefaultRegisterer,
	}

	return batchChangesMetrics{
		reconcilerWorkerMetrics:               workerutil.NewMetrics(observationContext, "batch_changes_reconciler", nil),
		bulkProcessorWorkerMetrics:            workerutil.NewMetrics(observationContext, "batch_changes_bulk_processor", nil),
		pendingBatchSpecWorkerMetrics:         workerutil.NewMetrics(observationContext, "batch_changes_pending_batch_specs", nil),
		reconcilerWorkerResetterMetrics:       makeResetterMetrics(observationContext, "batch_changes_reconciler"),
		bulkProcessorWorkerResetterMetrics:    makeResetterMetrics(observationContext, "batch_changes_bulk_processor"),
		pendingBatchSpecWorkerResetterMetrics: makeResetterMetrics(observationContext, "batch_changes_pending_batch_specs"),
	}
}

func makeResetterMetrics(observationContext *observation.Context, workerName string) dbworker.ResetterMetrics {
	resetFailures := prometheus.NewCounter(prometheus.CounterOpts{
		Name: fmt.Sprintf("src_%s_reset_failures_total", workerName),
		Help: "The number of reset failures.",
	})
	observationContext.Registerer.MustRegister(resetFailures)

	resets := prometheus.NewCounter(prometheus.CounterOpts{
		Name: fmt.Sprintf("src_%s_resets_total", workerName),
		Help: "The number of records reset.",
	})
	observationContext.Registerer.MustRegister(resets)

	errors := prometheus.NewCounter(prometheus.CounterOpts{
		Name: fmt.Sprintf("src_%s_reset_errors_total", workerName),
		Help: "The number of errors that occur when resetting records.",
	})
	observationContext.Registerer.MustRegister(errors)
	return dbworker.ResetterMetrics{
		RecordResets:        resets,
		RecordResetFailures: resetFailures,
		Errors:              errors,
	}
}
