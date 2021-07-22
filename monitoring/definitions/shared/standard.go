package shared

import (
	"fmt"

	"github.com/sourcegraph/sourcegraph/monitoring/monitoring"
)

var (
	// StandardCount creates an observable from the given options backed by
	// the counter specifying the number of operatons. The legend name supplied
	// to the outermost function will be used as the panel's dataset legend
	// (supplemented by label values if By is also assigned).
	//
	// Requires a counter of the format `src_{options.MetricName}_total`
	StandardCount func(legend string) observableConstructor = func(legend string) observableConstructor {
		if legend != "" {
			legend = " " + legend
		}

		return func(options ObservableOptions) sharedObservable {
			return func(containerName string, owner monitoring.ObservableOwner) Observable {
				filters := makeFilters(containerName, options.Filters...)
				by, legendPrefix := makeBy(options.By...)

				return Observable{
					Name:           fmt.Sprintf("%s_total", options.MetricName),
					Description:    fmt.Sprintf("%s%s every 5m", options.MetricDescription, legend),
					Query:          fmt.Sprintf(`sum%s(increase(src_%s_total{%s}[5m]))`, by, options.MetricName, filters),
					Panel:          monitoring.Panel().LegendFormat(fmt.Sprintf("%s%s", legendPrefix, legend)),
					Owner:          owner,
					NoAlert:        true,
					Interpretation: "none",
				}
			}
		}
	}

	// StandardDuration creates an observable from the given options backed by
	// the histogram specifying the duration of operatons.
	//
	// Requires a histogram of the format `src_{options.MetricName}_duration_seconds_bucket`
	StandardDuration func(legend string) observableConstructor = func(legend string) observableConstructor {
		if legend != "" {
			legend = " " + legend
		}

		return func(options ObservableOptions) sharedObservable {
			return func(containerName string, owner monitoring.ObservableOwner) Observable {
				filters := makeFilters(containerName, options.Filters...)
				by, _ := makeBy(append([]string{"le"}, options.By...)...)
				_, legendPrefix := makeBy(options.By...)

				return Observable{
					Name:           fmt.Sprintf("%s_99th_percentile_duration", options.MetricName),
					Description:    fmt.Sprintf("99th percentile successful %s%s duration over 5m", options.MetricDescription, legend),
					Query:          fmt.Sprintf(`histogram_quantile(0.99, sum %s(rate(src_%s_duration_seconds_bucket{%s}[5m])))`, by, options.MetricName, filters),
					Panel:          monitoring.Panel().LegendFormat(fmt.Sprintf("%s%s", legendPrefix, legend)).Unit(monitoring.Seconds),
					Owner:          owner,
					NoAlert:        true,
					Interpretation: "none",
				}
			}
		}
	}

	// StandardErrors creates an observable from the given options backed by
	// the counter specifying the number of operatons that resulted in an error.
	//
	// Requires a counter of the format `src_{options.MetricName}_errors_total`
	StandardErrors func(legend string) observableConstructor = func(legend string) observableConstructor {
		if legend != "" {
			legend = " " + legend
		}

		return func(options ObservableOptions) sharedObservable {
			return func(containerName string, owner monitoring.ObservableOwner) Observable {
				filters := makeFilters(containerName, options.Filters...)
				by, legendPrefix := makeBy(options.By...)

				return Observable{
					Name:           fmt.Sprintf("%s_errors_total", options.MetricName),
					Description:    fmt.Sprintf("%s%s errors every 5m", options.MetricDescription, legend),
					Query:          fmt.Sprintf(`sum%s(increase(src_%s_errors_total{%s}[5m]))`, by, options.MetricName, filters),
					Panel:          monitoring.Panel().LegendFormat(fmt.Sprintf("%s%s errors", legendPrefix, legend)),
					Owner:          owner,
					NoAlert:        true,
					Interpretation: "none",
				}
			}
		}
	}
)
