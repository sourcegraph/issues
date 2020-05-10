package worker

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sourcegraph/sourcegraph/internal/metrics"
)

type WorkerMetrics struct {
	Processor *metrics.OperationMetrics
}

func NewWorkerMetrics(r prometheus.Registerer) WorkerMetrics {
	processor := metrics.NewOperationMetrics(
		r,
		"upload_queue", // temporary
		"processor",
		// metrics.WithSubsystem("upload_queue"),
	)

	return WorkerMetrics{
		Processor: processor,
	}
}
