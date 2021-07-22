package definitions

import (
	"github.com/sourcegraph/sourcegraph/monitoring/definitions/shared"
	"github.com/sourcegraph/sourcegraph/monitoring/monitoring"
)

func ExecutorQueue() *monitoring.Container {
	const (
		containerName      = "executor-queue"
		queueContainerName = "(executor|sourcegraph-code-intel-indexers|executor-batches|executor-queue)"
	)

	return &monitoring.Container{
		Name:        "executor-queue",
		Title:       "Executor Queue",
		Description: "Coordinates the executor work queues.",
		Groups: []monitoring.Group{
			// src_executor_total
			// src_executor_processor_total
			shared.NewQueueSizeGroup(queueContainerName, monitoring.ObservableOwnerCodeIntel, shared.QueueSizeGroupOptions{
				ObservableOptions: shared.ObservableOptions{
					Namespace:         "executor",
					GroupDescription:  "Executor jobs",
					MetricName:        "executor",
					MetricDescription: "unprocessed executor job",
					By:                []string{"queue"},
				},
			}),

			// src_workerutil_dbworker_store_codeintel_index_total
			// src_workerutil_dbworker_store_codeintel_index_duration_seconds_bucket
			// src_workerutil_dbworker_store_codeintel_index_errors_total
			shared.NewObservationGroup(containerName, monitoring.ObservableOwnerCodeIntel, shared.ObservationGroupOptions{
				ObservableOptions: shared.ObservableOptions{
					Namespace:         "workerutil",
					GroupDescription:  "dbworker/store stats (db=frontend, table=lsif_indexes)",
					MetricName:        "workerutil_dbworker_store_codeintel_index",
					MetricDescription: "store",
					Hidden:            true,
				},
			}),

			// Resource monitoring
			shared.NewFrontendInternalAPIErrorResponseMonitoringGroup(containerName, monitoring.ObservableOwnerCodeIntel, nil),
			shared.NewDatabaseConnectionsMonitoringGroup(containerName),
			shared.NewContainerMonitoringGroup(containerName, monitoring.ObservableOwnerCodeIntel, nil),
			shared.NewProvisioningIndicatorsGroup(containerName, monitoring.ObservableOwnerCodeIntel, nil),
			shared.NewGolangMonitoringGroup(containerName, monitoring.ObservableOwnerCodeIntel, nil),
			shared.NewKubernetesMonitoringGroup(containerName, monitoring.ObservableOwnerCodeIntel, nil),
		},
	}
}
