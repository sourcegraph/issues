package definitions

import (
	"github.com/sourcegraph/sourcegraph/monitoring/definitions/shared"
	"github.com/sourcegraph/sourcegraph/monitoring/monitoring"
)

func PreciseCodeIntelWorker() *monitoring.Container {
	const containerName = "precise-code-intel-worker"

	return &monitoring.Container{
		Name:        "precise-code-intel-worker",
		Title:       "Precise Code Intel Worker",
		Description: "Handles conversion of uploaded precise code intelligence bundles.",
		Groups: []monitoring.Group{
			// src_codeintel_upload_total
			// src_codeintel_upload_processor_total
			shared.NewQueueSizeGroup(containerName, monitoring.ObservableOwnerCodeIntel, shared.QueueSizeGroupOptions{
				ObservableOptions: shared.ObservableOptions{
					Namespace:         "codeintel",
					GroupDescription:  "LSIF uploads",
					MetricName:        "codeintel_upload",
					MetricDescription: "unprocessed upload record",
				},
			}),

			// src_codeintel_upload_processor_total
			// src_codeintel_upload_processor_duration_seconds_bucket
			// src_codeintel_upload_processor_errors_total
			// src_codeintel_upload_processor_handlers
			shared.NewWorkerutilGroup(containerName, monitoring.ObservableOwnerCodeIntel, shared.WorkerutilGroupOptions{
				ObservableOptions: shared.ObservableOptions{
					Namespace:         "codeintel",
					GroupDescription:  "LSIF uploads",
					MetricName:        "codeintel_upload",
					MetricDescription: "handler",
				},
			}),

			// src_codeintel_dbstore_total
			// src_codeintel_dbstore_duration_seconds_bucket
			// src_codeintel_dbstore_errors_total
			shared.NewObservationGroup(containerName, monitoring.ObservableOwnerCodeIntel, shared.ObservationGroupOptions{
				ObservableOptions: shared.ObservableOptions{
					Namespace:         "codeintel",
					GroupDescription:  "dbstore stats (db=frontend)",
					MetricName:        "codeintel_dbstore",
					MetricDescription: "store",
					Hidden:            true,
				},
			}),

			// src_codeintel_lsifstore_total
			// src_codeintel_lsifstore_duration_seconds_bucket
			// src_codeintel_lsifstore_errors_total
			shared.NewObservationGroup(containerName, monitoring.ObservableOwnerCodeIntel, shared.ObservationGroupOptions{
				ObservableOptions: shared.ObservableOptions{
					Namespace:         "codeintel",
					GroupDescription:  "lsifstore stats (db=codeintel-db)",
					MetricName:        "codeintel_lsifstore",
					MetricDescription: "store",
					Hidden:            true,
				},
			}),

			// src_workerutil_dbworker_store_codeintel_upload_total
			// src_workerutil_dbworker_store_codeintel_upload_duration_seconds_bucket
			// src_workerutil_dbworker_store_codeintel_upload_errors_total
			shared.NewObservationGroup(containerName, monitoring.ObservableOwnerCodeIntel, shared.ObservationGroupOptions{
				ObservableOptions: shared.ObservableOptions{
					Namespace:         "workerutil",
					GroupDescription:  "dbworker/store stats (db=frontend, table=lsif_uploads)",
					MetricName:        "workerutil_dbworker_store_codeintel_upload",
					MetricDescription: "store",
					Hidden:            true,
				},
			}),

			// src_codeintel_gitserver_total
			// src_codeintel_gitserver_duration_seconds_bucket
			// src_codeintel_gitserver_errors_total
			shared.NewObservationGroup(containerName, monitoring.ObservableOwnerCodeIntel, shared.ObservationGroupOptions{
				ObservableOptions: shared.ObservableOptions{
					Namespace:         "codeintel",
					GroupDescription:  "gitserver client",
					MetricName:        "codeintel_gitserver",
					MetricDescription: "client",
					Hidden:            true,
				},
			}),

			// src_codeintel_uploadstore_total
			// src_codeintel_uploadstore_duration_seconds_bucket
			// src_codeintel_uploadstore_errors_total
			shared.NewObservationGroup(containerName, monitoring.ObservableOwnerCodeIntel, shared.ObservationGroupOptions{
				ObservableOptions: shared.ObservableOptions{
					Namespace:         "codeintel",
					GroupDescription:  "uploadstore stats (queries GCS/S3/MinIO)",
					MetricName:        "codeintel_uploadstore",
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
