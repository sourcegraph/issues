// Package shared contains shared declarations between dashboards. In general, you should NOT be making
// changes to this package: we use a declarative style for monitoring intentionally, so you should err
// on the side of repeating yourself and NOT writing shared or programatically generated monitoring.
//
// When editing this package or introducing any shared declarations, you should abide strictly by the
// following rules:
//
// 1. Do NOT declare a shared definition unless 5+ dashboards will use it. Sharing dashboard
//    declarations means the codebase becomes more complex and non-declarative which we want to avoid
//    so repeat yourself instead if it applies to less than 5 dashboards.
//
// 2. ONLY declare shared Observables. Introducing shared Rows or Groups prevents individual dashboard
//    maintainers from holistically considering both the layout of dashboards as well as the
//    metrics and alerts defined within them -- which we do not want.
//
// 3. Use the sharedObservable type and do NOT parameterize more than just the container name. It may
//    be tempting to pass an alerting threshold as an argument, or parameterize whether a critical
//    alert is defined -- but this makes reasoning about alerts at a high level much more difficult.
//    If you have a need for this, it is a strong signal you should NOT be using the shared definition
//    anymore and should instead copy it and apply your modifications.
//
// Learn more about monitoring in https://about.sourcegraph.com/handbook/engineering/observability/monitoring_pillars
package shared

import (
	"fmt"
	"strings"

	"github.com/sourcegraph/sourcegraph/monitoring/monitoring"
)

// Observable is a variant of normal Observables that offer convenience functions for
// customizing shared observables.
type Observable monitoring.Observable

// Observable is a convenience adapter that casts this SharedObservable as an normal Observable.
func (o Observable) Observable() monitoring.Observable { return monitoring.Observable(o) }

// WithWarning overrides this Observable's warning-level alert with the given alert.
func (o Observable) WithWarning(a *monitoring.ObservableAlertDefinition) Observable {
	o.Warning = a
	if a != nil {
		o.NoAlert = false
	}
	return o
}

// WithCritical overrides this Observable's critical-level alert with the given alert.
func (o Observable) WithCritical(a *monitoring.ObservableAlertDefinition) Observable {
	o.Critical = a
	if a != nil {
		o.NoAlert = false
	}
	return o
}

// WithNoAlerts disables alerting on this Observable and sets the given interpretation instead.
func (o Observable) WithNoAlerts(interpretation string) Observable {
	o.Warning = nil
	o.Critical = nil
	o.NoAlert = true
	o.PossibleSolutions = ""
	o.Interpretation = interpretation
	return o
}

// ObservableOption is a function that transforms an observable.
type ObservableOption func(observable Observable) Observable

func (f ObservableOption) safeApply(observable Observable) Observable {
	if f == nil {
		return observable
	}

	return f(observable)
}

// sharedObservable defines the type all shared observable variables should have in this package.
type sharedObservable func(containerName string, owner monitoring.ObservableOwner) Observable

type ObservableOptions struct {
	// Namespace is displayed in the title of the group containing the observable. This
	// value should generally name a component or team owning the group of metrics.
	Namespace string

	// GroupDescription is a human-readable description of the group of metrics displayed
	// in the title of the group containing the observable.
	GroupDescription string

	// MetricName is a prometheus metric name or name fragment that is used to construct
	// the query for the target panel.
	MetricName string

	// MetricDescription is a human-readable name for the object represented by each metric.
	// This is used to disambiguate more generic terms such as "requests" or "records". The
	// value in the panel description or legend will be generated but made more specific
	// by this value (e.g. "code intel resolver operations").
	//          metric desc ^^^^^^^^^^^^^^^^^^^ ^^^^^^^^^^ generic term
	MetricDescription string

	// Filters are additional prometheus filter expressions used to select or hide values
	// for a given label pattern.
	Filters []string

	// By are label names that should not be aggregated together. Supplying labels here
	// will increase the number of series on the target panel. The legends for each series
	// will be updated to include the value of each label group supplied here.
	By []string

	// Hidden sets the Hidden field of the resulting group.
	Hidden bool
}

// observableConstructor is a type of constructor function used in this package that creates
// a shared observable given a set of common observable options.
type observableConstructor func(options ObservableOptions) sharedObservable

// CadvisorNameMatcher generates Prometheus matchers that capture metrics that match the
// given container name while excluding some irrelevant series.
func CadvisorNameMatcher(containerName string) string {
	// Name must start with the container name exactly.
	//
	// In docker-compose:
	// - `name` is just the container name
	// - suffix could be replica in docker-compose ('-0', '-1')
	//
	// In Kubernetes:
	// - a `metric_relabel_configs` generates a `name` with the format `CONTAINERNAME-PODNAME`,
	//   because cAdvisor does not consistently generate a name in all container runtimes.
	//   See https://sourcegraph.com/search?q=repo:%5Egithub%5C.com/sourcegraph/deploy-sourcegraph%24+target_label:+name&patternType=literal
	// - because of above, suffix could be pod name in Kubernetes
	return fmt.Sprintf(`name=~"^%s.*"`, containerName)
}

// makeFilters creates metric filters based on the given container name that matches
// against the container name as well as any additionally supplied label filter expressions.
func makeFilters(containerName string, filters ...string) string {
	filters = append(filters, fmt.Sprintf(`job=~"%s"`, containerName))
	return strings.Join(filters, ",")
}

// makeBy returns the suffix if the aggregator expression (e.g., max by (queue)),
//                                                                   ^^^^^^^^^^
// as well as a prefix to be used as part of the legend consisting of placeholder
// values that will render to the value of the label/variable in the Grafana UI.
func makeBy(labels ...string) (aggregateExprSuffix string, legendPrefix string) {
	if len(labels) == 0 {
		return "", ""
	}

	placeholders := make([]string, 0, len(labels))
	for _, label := range labels {
		placeholders = append(placeholders, fmt.Sprintf("{{%s}}", label))
	}

	aggregateExprSuffix = fmt.Sprintf(" by (%s)", strings.Join(labels, ","))
	legendPrefix = fmt.Sprintf("%s ", strings.Join(placeholders, "-"))

	return aggregateExprSuffix, legendPrefix
}
