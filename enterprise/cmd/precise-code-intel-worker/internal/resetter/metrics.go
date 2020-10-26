package resetter

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sourcegraph/sourcegraph/internal/workerutil/dbworker"
)

func NewResetterMetrics(r prometheus.Registerer) dbworker.ResetterMetrics {
	uploadResets := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "src_upload_queue_resets_total",
		Help: "Total number of uploads put back into queued state",
	})
	r.MustRegister(uploadResets)

	uploadResetFailures := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "src_upload_queue_max_resets_total",
		Help: "Total number of uploads that exceed the max number of resets",
	})
	r.MustRegister(uploadResetFailures)

	errors := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "src_upload_queue_reset_errors_total",
		Help: "Total number of errors when running the upload resetter",
	})
	r.MustRegister(errors)

	return dbworker.ResetterMetrics{
		RecordResets:        uploadResets,
		RecordResetFailures: uploadResetFailures,
		Errors:              errors,
	}
}
