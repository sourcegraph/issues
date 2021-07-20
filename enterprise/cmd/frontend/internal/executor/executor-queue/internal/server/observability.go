package server

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/sourcegraph/sourcegraph/internal/observation"
)

type QueueMetrics struct {
	NumJobs      *prometheus.GaugeVec
	NumExecutors *prometheus.GaugeVec
}

func newQueueMetrics(observationContext *observation.Context) *QueueMetrics {
	gaugeVec := func(name, help string) *prometheus.GaugeVec {
		gaugeVec := prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: name,
			Help: help,
		}, []string{"queue"})

		observationContext.Registerer.MustRegister(gaugeVec)
		return gaugeVec
	}

	numJobs := gaugeVec(
		"src_apiworker_apiserver_jobs_total",
		"The number of processing job records.",
	)
	numExecutors := gaugeVec(
		"src_apiworker_apiserver_executors_total",
		"The number of executors attached to a queue.",
	)

	return &QueueMetrics{
		NumJobs:      numJobs,
		NumExecutors: numExecutors,
	}
}
