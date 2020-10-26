package resetter

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sourcegraph/sourcegraph/internal/workerutil/dbworker"
)

func NewResetterMetrics(r prometheus.Registerer) dbworker.ResetterMetrics {
	indexResets := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "src_index_queue_reset_total",
		Help: "Total number of indexes put back into queued state",
	})
	r.MustRegister(indexResets)

	indexResetFailures := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "src_index_queue_max_resets_total",
		Help: "Total number of indexes that exceed the max number of resets",
	})
	r.MustRegister(indexResetFailures)

	errors := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "src_index_queue_reset_errors_total",
		Help: "Total number of errors when running the index resetter",
	})
	r.MustRegister(errors)

	return dbworker.ResetterMetrics{
		RecordResets:        indexResets,
		RecordResetFailures: indexResetFailures,
		Errors:              errors,
	}
}
