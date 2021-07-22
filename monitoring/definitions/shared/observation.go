package shared

import (
	"fmt"

	"github.com/sourcegraph/sourcegraph/monitoring/monitoring"
)

var (
	// ObservationTotal creates an observable from the given options backed by
	// the counter specifying the number of operatons.
	//
	// Requires a counter of the format `src_{options.MetricName}_total`
	ObservationTotal observableConstructor = StandardCount("operations")

	// ObservationDuration creates an observable from the given options backed by
	// the histogram specifying the duration of operatons.
	//
	// Requires a histogram of the format `src_{options.MetricName}_duration_seconds_bucket`
	ObservationDuration observableConstructor = StandardDuration("operation")

	// ObservationErrors creates an observable from the given options backed by
	// the counter specifying the number of operatons that resulted in an error.
	//
	// Requires a counter of the format `src_{options.MetricName}_errors_total`
	ObservationErrors observableConstructor = StandardErrors("operation")
)

type ObservationGroupOptions struct {
	ObservableOptions

	// Total transforms the default observable used to construct the operation count panel.
	Total ObservableOption

	// Duration transforms the default observable used to construct the duration histogram panel.
	Duration ObservableOption

	// Errors transforms the default observable used to construct the error rate panel.
	Errors ObservableOption
}

// NewObservationGroup creates a group containing panels displaying the total number of operations,
// operation duration histogram, and number of errors for the given observable within the given
// container.
//
// Requires a:
//   - counter of the format `src_{options.MetricName}_total`
//   - histogram of the format `src_{options.MetricName}_duration_seconds_bucket`
//   - counter of the format `src_{options.MetricName}_errors_total`
func NewObservationGroup(containerName string, owner monitoring.ObservableOwner, options ObservationGroupOptions) monitoring.Group {
	return monitoring.Group{
		Title:  fmt.Sprintf("[%s] Observable: %s", options.Namespace, options.GroupDescription),
		Hidden: options.Hidden,
		Rows: []monitoring.Row{
			{
				options.Total.safeApply(ObservationTotal(options.ObservableOptions)(containerName, owner)).Observable(),
				options.Duration.safeApply(ObservationDuration(options.ObservableOptions)(containerName, owner)).Observable(),
				options.Errors.safeApply(ObservationErrors(options.ObservableOptions)(containerName, owner)).Observable(),
			},
		},
	}
}
