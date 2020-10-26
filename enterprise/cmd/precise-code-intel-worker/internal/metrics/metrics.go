package metrics

import (
	"github.com/sourcegraph/sourcegraph/internal/metrics"
	"github.com/sourcegraph/sourcegraph/internal/observation"
)

type WorkerMetrics struct {
	ProcessOperation             *observation.Operation
	RepoStateOperation           *observation.Operation
	CorrelateOperation           *observation.Operation
	CanonicalizeOperation        *observation.Operation
	PruneOperation               *observation.Operation
	GroupBundleDataOperation     *observation.Operation
	WriteOperation               *observation.Operation
	UpdateXrepoDatabaseOperation *observation.Operation
	SendDBOperation              *observation.Operation
}

func NewWorkerMetrics(observationContext *observation.Context) WorkerMetrics {
	metrics := metrics.NewOperationMetrics(
		observationContext.Registerer,
		"upload_queue_processor",
		metrics.WithLabels("op"),
		metrics.WithCountHelp("Total number of records processed"),
	)

	return WorkerMetrics{
		ProcessOperation: observationContext.Operation(observation.Op{
			Name:         "Processor.Process",
			MetricLabels: []string{"process"},
			Metrics:      metrics,
		}),
		RepoStateOperation: observationContext.Operation(observation.Op{
			Name:         "Processor.RepoState",
			MetricLabels: []string{"repo_state"},
		}),
		CorrelateOperation: observationContext.Operation(observation.Op{
			Name:         "Processor.Correlate",
			MetricLabels: []string{"correlate"},
		}),
		CanonicalizeOperation: observationContext.Operation(observation.Op{
			Name:         "Processor.Canonicalize",
			MetricLabels: []string{"canonicalize"},
		}),
		PruneOperation: observationContext.Operation(observation.Op{
			Name:         "Processor.Prune",
			MetricLabels: []string{"prune"},
		}),
		GroupBundleDataOperation: observationContext.Operation(observation.Op{
			Name:         "Processor.GroupBundleData",
			MetricLabels: []string{"groupBundleData"},
		}),
		WriteOperation: observationContext.Operation(observation.Op{
			Name:         "Processor.Write",
			MetricLabels: []string{"write"},
		}),
		UpdateXrepoDatabaseOperation: observationContext.Operation(observation.Op{
			Name:         "Processor.UpdateXrepoDatabase",
			MetricLabels: []string{"update_xrepo_database"},
		}),
		SendDBOperation: observationContext.Operation(observation.Op{
			Name:         "Processor.SendDB",
			MetricLabels: []string{"send_db"},
		}),
	}
}
