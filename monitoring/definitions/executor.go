package definitions

import (
	"github.com/grafana-tools/sdk"

	"github.com/sourcegraph/sourcegraph/monitoring/definitions/shared"
	"github.com/sourcegraph/sourcegraph/monitoring/monitoring"
)

func Executor() *monitoring.Container {
	const (
		containerName      = "(executor|sourcegraph-code-intel-indexers|executor-batches)"
		queueContainerName = "(executor|sourcegraph-code-intel-indexers|executor-batches|executor-queue)"
	)

	filters := []string{`queue=~"${queue:regex}"`}

	return &monitoring.Container{
		Name:        "executor",
		Title:       "Executor",
		Description: `Executes jobs from the executor-queue.`,
		Templates: []sdk.TemplateVar{
			{
				Label:      "Queue name",
				Name:       "queue",
				AllValue:   ".*",
				Current:    sdk.Current{Text: &sdk.StringSliceString{Value: []string{"all"}, Valid: true}, Value: "$__all"},
				IncludeAll: true,
				Options: []sdk.Option{
					{Text: "all", Value: "$__all", Selected: true},
					{Text: "batches", Value: "batches"},
					{Text: "codeintel", Value: "codeintel"},
				},
				Query: "batches,codeintel",
				Type:  "custom",
			},
		},
		Groups: []monitoring.Group{
			// src_executor_total
			// src_executor_processor_total
			shared.NewQueueSizeGroup(queueContainerName, monitoring.ObservableOwnerCodeIntel, shared.QueueSizeGroupOptions{
				ObservableOptions: shared.ObservableOptions{
					Namespace:         "executor",
					GroupDescription:  "Executor jobs",
					MetricName:        "executor",
					MetricDescription: "unprocessed executor job",
					Filters:           filters,
				},
			}),

			// src_executor_processor_total
			// src_executor_processor_duration_seconds_bucket
			// src_executor_processor_errors_total
			// src_executor_processor_handlers
			shared.NewWorkerutilGroup(containerName, monitoring.ObservableOwnerCodeIntel, shared.WorkerutilGroupOptions{
				ObservableOptions: shared.ObservableOptions{
					Namespace:         "executor",
					GroupDescription:  "Executor jobs",
					MetricName:        "executor",
					MetricDescription: "handler",
					Filters:           filters,
				},
			}),

			// src_apiworker_apiclient_total
			// src_apiworker_apiclient_duration_seconds_bucket
			// src_apiworker_apiclient_errors_total
			shared.NewObservationGroup(containerName, monitoring.ObservableOwnerCodeIntel, shared.ObservationGroupOptions{
				ObservableOptions: shared.ObservableOptions{
					Namespace:         "executor",
					GroupDescription:  "Queue API client",
					MetricName:        "apiworker_apiclient",
					MetricDescription: "client",
					Filters:           nil, // note: shared between queues
					Hidden:            true,
				},
			}),

			// src_apiworker_command_total
			// src_apiworker_command_duration_seconds_bucket
			// src_apiworker_command_errors_total
			shared.NewObservationGroup(containerName, monitoring.ObservableOwnerCodeIntel, shared.ObservationGroupOptions{
				ObservableOptions: shared.ObservableOptions{
					Namespace:         "executor",
					GroupDescription:  "Subprocess execution (for job setup)",
					MetricName:        "apiworker_command",
					MetricDescription: "command",
					Filters:           []string{`op=~"setup.*"`}, // note: shared between queues
					Hidden:            true,
				},
			}),

			// src_apiworker_command_total
			// src_apiworker_command_duration_seconds_bucket
			// src_apiworker_command_errors_total
			shared.NewObservationGroup(containerName, monitoring.ObservableOwnerCodeIntel, shared.ObservationGroupOptions{
				ObservableOptions: shared.ObservableOptions{
					Namespace:         "executor",
					GroupDescription:  "Subprocess execution (for job execution)",
					MetricName:        "apiworker_command",
					MetricDescription: "command",
					Filters:           []string{`op=~"exec.*"`}, // note: shared between queues
					Hidden:            true,
				},
			}),

			// src_apiworker_command_total
			// src_apiworker_command_duration_seconds_bucket
			// src_apiworker_command_errors_total
			shared.NewObservationGroup(containerName, monitoring.ObservableOwnerCodeIntel, shared.ObservationGroupOptions{
				ObservableOptions: shared.ObservableOptions{
					Namespace:         "executor",
					GroupDescription:  "Subprocess execution (for job teardown)",
					MetricName:        "apiworker_command",
					MetricDescription: "command",
					Filters:           []string{`op=~"teardown.*"`}, // note: shared between queues
					Hidden:            true,
				},
			}),

			// Resource monitoring
			shared.NewContainerMonitoringGroup(containerName, monitoring.ObservableOwnerCodeIntel, nil),
			shared.NewProvisioningIndicatorsGroup(containerName, monitoring.ObservableOwnerCodeIntel, nil),
			shared.NewGolangMonitoringGroup(containerName, monitoring.ObservableOwnerCodeIntel, nil),
			shared.NewKubernetesMonitoringGroup(containerName, monitoring.ObservableOwnerCodeIntel, nil),
		},
	}
}
