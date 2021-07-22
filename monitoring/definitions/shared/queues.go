package shared

import (
	"fmt"

	"github.com/sourcegraph/sourcegraph/monitoring/monitoring"
)

var (
	// WorkerutilQueueSize creates an observable from the given options backed by
	// the gauge specifying the number of pending records in a given queue.
	//
	// Requires a gauge of the format `src_{options.MetricName}_total`
	WorkerutilQueueSize observableConstructor = func(options ObservableOptions) sharedObservable {
		return func(containerName string, owner monitoring.ObservableOwner) Observable {
			filters := makeFilters(containerName, options.Filters...)
			by, legendPrefix := makeBy(options.By...)

			return Observable{
				Name:           fmt.Sprintf("%s_queue_size", options.MetricName),
				Description:    fmt.Sprintf("%s queue size", options.MetricDescription),
				Query:          fmt.Sprintf(`max%s(src_%s_total{%s})`, by, options.MetricName, filters),
				Panel:          monitoring.Panel().LegendFormat(fmt.Sprintf("%s records", legendPrefix)),
				Owner:          owner,
				NoAlert:        true,
				Interpretation: "none",
			}
		}
	}

	// WorkerutilQueueGrowthRate creates an observable from the given options backed by
	// the rate of increase of enqueues compared to the processing rate.
	//
	// Requires a:
	//   - gauge of the format `src_{options.MetricName}_total`
	//   - counter of the format `src_{options.MetricName}_processor_total`
	WorkerutilQueueGrowthRate observableConstructor = func(options ObservableOptions) sharedObservable {
		return func(containerName string, owner monitoring.ObservableOwner) Observable {
			filters := makeFilters(containerName, options.Filters...)
			by, legendPrefix := makeBy(options.By...)

			return Observable{
				Name:           fmt.Sprintf("%s_queue_growth_rate", options.MetricName),
				Description:    fmt.Sprintf("%s queue growth rate over 30m", options.MetricDescription),
				Query:          fmt.Sprintf(`sum%[1]s(increase(src_%[2]s_total{%[3]s}[30m])) / sum%[1]s(increase(src_%[2]s_processor_total{%[3]s}[30m]))`, by, options.MetricName, filters),
				Panel:          monitoring.Panel().LegendFormat(fmt.Sprintf("%s queue growth rate", legendPrefix)),
				Owner:          owner,
				NoAlert:        true,
				Interpretation: "none",
			}
		}
	}
)

type QueueSizeGroupOptions struct {
	ObservableOptions

	// QueueSize transforms the default observable used to construct the queue sizes panel.
	QueueSize ObservableOption

	// QueueGrowthRate transforms the default observable used to construct the queue growth rate panel.
	QueueGrowthRate ObservableOption
}

// NewQueueSizeGroup creates a group containing panels displaying metrics to monitor the
// size and growth rate of a queue of work within the given container.
//
// Requires a:
//   - gauge of the format `src_{options.MetricName}_total`
//   - counter of the format `src_{options.MetricName}_processor_total`
func NewQueueSizeGroup(containerName string, owner monitoring.ObservableOwner, options QueueSizeGroupOptions) monitoring.Group {
	return monitoring.Group{
		Title:  fmt.Sprintf("[%s] Queue: %s", options.Namespace, options.GroupDescription),
		Hidden: options.Hidden,
		Rows: []monitoring.Row{
			{
				options.QueueSize.safeApply(WorkerutilQueueSize(options.ObservableOptions)(containerName, owner)).Observable(),
				options.QueueGrowthRate.safeApply(WorkerutilQueueGrowthRate(options.ObservableOptions)(containerName, owner)).Observable(),
			},
		},
	}
}
