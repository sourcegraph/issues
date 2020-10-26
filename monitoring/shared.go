package main

import (
	"fmt"
	"strings"
	"time"
)

// This file contains shared declarations between dashboards. In general, you should NOT be making
// changes to this file: we use a declarative style for monitoring intentionally, so you should err
// on the side of repeating yourself and NOT writing shared or programatically generated monitoring.
//
// When editing this file or introducing any shared declarations, you should abide strictly by the
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

type sharedObservable func(containerName string, owner ObservableOwner) Observable

var sharedFrontendInternalAPIErrorResponses sharedObservable = func(containerName string, owner ObservableOwner) Observable {
	return Observable{
		Name:            "frontend_internal_api_error_responses",
		Description:     "frontend-internal API error responses every 5m by route",
		Query:           fmt.Sprintf(`sum by (category)(increase(src_frontend_internal_request_duration_seconds_count{job="%[1]s",code!~"2.."}[5m])) / ignoring(category) group_left sum(increase(src_frontend_internal_request_duration_seconds_count{job="%[1]s"}[5m]))`, containerName),
		DataMayNotExist: true,
		Warning:         Alert().GreaterOrEqual(2).For(5 * time.Minute),
		PanelOptions:    PanelOptions().LegendFormat("{{category}}").Unit(Percentage),
		Owner:           owner,
		PossibleSolutions: strings.Replace(`
			- **Single-container deployments:** Check 'docker logs $CONTAINER_ID' for logs starting with 'repo-updater' that indicate requests to the frontend service are failing.
			- **Kubernetes:**
				- Confirm that 'kubectl get pods' shows the 'frontend' pods are healthy.
				- Check 'kubectl logs {{CONTAINER_NAME}}' for logs indicate request failures to 'frontend' or 'frontend-internal'.
			- **Docker Compose:**
				- Confirm that 'docker ps' shows the 'frontend-internal' container is healthy.
				- Check 'docker logs {{CONTAINER_NAME}}' for logs indicating request failures to 'frontend' or 'frontend-internal'.
		`, "{{CONTAINER_NAME}}", containerName, -1),
	}
}

// promCadvisorContainerMatchers generates Prometheus matchers that capture metrics that match the given container name
// while excluding some irrelevant series
func promCadvisorContainerMatchers(containerName string) string {
	// This matcher excludes:
	// * jaeger sidecar (jaeger-agent)
	// * pod sidecars (_POD_)
	// as well as matching on the name of the container exactly with "_{container}_"
	return fmt.Sprintf(`name=~".*_%s_.*",name!~".*(_POD_|_jaeger-agent_).*"`, containerName)
}

// Container monitoring overviews - alert on all container failures, but only alert on extreme resource usage.
// More granular resource usage warnings are provided by the provisioning observables.

var sharedContainerRestarts sharedObservable = func(containerName string, owner ObservableOwner) Observable {
	return Observable{
		Name:            "container_restarts",
		Description:     "container restarts every 5m by instance",
		Query:           fmt.Sprintf(`increase(cadvisor_container_restart_count{%s}[5m])`, promCadvisorContainerMatchers(containerName)),
		DataMayNotExist: true,
		Warning:         Alert().GreaterOrEqual(1),
		PanelOptions:    PanelOptions().LegendFormat("{{name}}"),
		Owner:           owner,
		PossibleSolutions: strings.Replace(`
			- **Kubernetes:**
				- Determine if the pod was OOM killed using 'kubectl describe pod {{CONTAINER_NAME}}' (look for 'OOMKilled: true') and, if so, consider increasing the memory limit in the relevant 'Deployment.yaml'.
				- Check the logs before the container restarted to see if there are 'panic:' messages or similar using 'kubectl logs -p {{CONTAINER_NAME}}'.
			- **Docker Compose:**
				- Determine if the pod was OOM killed using 'docker inspect -f \'{{json .State}}\' {{CONTAINER_NAME}}' (look for '"OOMKilled":true') and, if so, consider increasing the memory limit of the {{CONTAINER_NAME}} container in 'docker-compose.yml'.
				- Check the logs before the container restarted to see if there are 'panic:' messages or similar using 'docker logs {{CONTAINER_NAME}}' (note this will include logs from the previous and currently running container).
		`, "{{CONTAINER_NAME}}", containerName, -1),
	}
}

var sharedContainerMemoryUsage sharedObservable = func(containerName string, owner ObservableOwner) Observable {
	return Observable{
		Name:            "container_memory_usage",
		Description:     "container memory usage by instance",
		Query:           fmt.Sprintf(`cadvisor_container_memory_usage_percentage_total{%s}`, promCadvisorContainerMatchers(containerName)),
		DataMayNotExist: true,
		Warning:         Alert().GreaterOrEqual(99),
		PanelOptions:    PanelOptions().LegendFormat("{{name}}").Unit(Percentage).Interval(100).Max(100).Min(0),
		Owner:           owner,
		PossibleSolutions: strings.Replace(`
			- **Kubernetes:** Consider increasing memory limit in relevant 'Deployment.yaml'.
			- **Docker Compose:** Consider increasing 'memory:' of {{CONTAINER_NAME}} container in 'docker-compose.yml'.
		`, "{{CONTAINER_NAME}}", containerName, -1),
	}
}

var sharedContainerCPUUsage sharedObservable = func(containerName string, owner ObservableOwner) Observable {
	return Observable{
		Name:            "container_cpu_usage",
		Description:     "container cpu usage total (1m average) across all cores by instance",
		Query:           fmt.Sprintf(`cadvisor_container_cpu_usage_percentage_total{%s}`, promCadvisorContainerMatchers(containerName)),
		DataMayNotExist: true,
		Warning:         Alert().GreaterOrEqual(99),
		PanelOptions:    PanelOptions().LegendFormat("{{name}}").Unit(Percentage).Interval(100).Max(100).Min(0),
		Owner:           owner,
		PossibleSolutions: strings.Replace(`
			- **Kubernetes:** Consider increasing CPU limits in the the relevant 'Deployment.yaml'.
			- **Docker Compose:** Consider increasing 'cpus:' of the {{CONTAINER_NAME}} container in 'docker-compose.yml'.
		`, "{{CONTAINER_NAME}}", containerName, -1),
	}
}

var sharedContainerFsInodes sharedObservable = func(containerName string, owner ObservableOwner) Observable {
	return Observable{
		Name:            "fs_inodes_used",
		Description:     "fs inodes in use by instance",
		Query:           fmt.Sprintf(`sum by (name)(container_fs_inodes_total{%s})`, promCadvisorContainerMatchers(containerName)),
		DataMayNotExist: true,
		Warning:         Alert().GreaterOrEqual(3e+06),
		PanelOptions:    PanelOptions().LegendFormat("{{name}}"),
		Owner:           owner,
		PossibleSolutions: `
			- Refer to your OS or cloud provider's documentation for how to increase inodes.
			- **Kubernetes:** consider provisioning more machines with less resources.`,
	}
}

// Warn that instances might need more resources if short-term usage is high.

var sharedProvisioningCPUUsageShortTerm sharedObservable = func(containerName string, owner ObservableOwner) Observable {
	return Observable{
		Name:            "provisioning_container_cpu_usage_short_term",
		Description:     "container cpu usage total (5m maximum) across all cores by instance",
		Query:           fmt.Sprintf(`max_over_time(cadvisor_container_cpu_usage_percentage_total{%s}[5m])`, promCadvisorContainerMatchers(containerName)),
		DataMayNotExist: true,
		Warning:         Alert().GreaterOrEqual(90).For(30 * time.Minute),
		PanelOptions:    PanelOptions().LegendFormat("{{name}}").Unit(Percentage).Interval(100).Max(100).Min(0),
		Owner:           owner,
		PossibleSolutions: strings.Replace(`
			- **Kubernetes:** Consider increasing CPU limits in the the relevant 'Deployment.yaml'.
			- **Docker Compose:** Consider increasing 'cpus:' of the {{CONTAINER_NAME}} container in 'docker-compose.yml'.
		`, "{{CONTAINER_NAME}}", containerName, -1),
	}
}

var sharedProvisioningMemoryUsageShortTerm sharedObservable = func(containerName string, owner ObservableOwner) Observable {
	return Observable{
		Name:            "provisioning_container_memory_usage_short_term",
		Description:     "container memory usage (5m maximum) by instance",
		Query:           fmt.Sprintf(`max_over_time(cadvisor_container_memory_usage_percentage_total{%s}[5m])`, promCadvisorContainerMatchers(containerName)),
		DataMayNotExist: true,
		Warning:         Alert().GreaterOrEqual(90),
		PanelOptions:    PanelOptions().LegendFormat("{{name}}").Unit(Percentage).Interval(100).Max(100).Min(0),
		Owner:           owner,
		PossibleSolutions: strings.Replace(`
			- **Kubernetes:** Consider increasing memory limit in relevant 'Deployment.yaml'.
			- **Docker Compose:** Consider increasing 'memory:' of {{CONTAINER_NAME}} container in 'docker-compose.yml'.
		`, "{{CONTAINER_NAME}}", containerName, -1),
	}
}

// Warn that instances might need more/less resources if long-term usage is high or low.

var sharedProvisioningCPUUsageLongTerm sharedObservable = func(containerName string, owner ObservableOwner) Observable {
	return Observable{
		Name:            "provisioning_container_cpu_usage_long_term",
		Description:     "container cpu usage total (90th percentile over 1d) across all cores by instance",
		Query:           fmt.Sprintf(`quantile_over_time(0.9, cadvisor_container_cpu_usage_percentage_total{%s}[1d])`, promCadvisorContainerMatchers(containerName)),
		DataMayNotExist: true,
		Warning:         Alert().LessOrEqual(30).GreaterOrEqual(80).For(14 * 24 * time.Hour),
		PanelOptions:    PanelOptions().LegendFormat("{{name}}").Unit(Percentage).Max(100).Min(0),
		Owner:           owner,
		PossibleSolutions: strings.Replace(`
			- If usage is high:
				- **Kubernetes:** Consider increasing CPU limits in the 'Deployment.yaml' for the {{CONTAINER_NAME}} service.
				- **Docker Compose:** Consider increasing 'cpus:' of the {{CONTAINER_NAME}} container in 'docker-compose.yml'.
			- If usage is low, consider decreasing the above values.
		`, "{{CONTAINER_NAME}}", containerName, -1),
	}
}

var sharedProvisioningMemoryUsageLongTerm sharedObservable = func(containerName string, owner ObservableOwner) Observable {
	return Observable{
		Name:            "provisioning_container_memory_usage_long_term",
		Description:     "container memory usage (1d maximum) by instance",
		Query:           fmt.Sprintf(`max_over_time(cadvisor_container_memory_usage_percentage_total{%s}[1d])`, promCadvisorContainerMatchers(containerName)),
		DataMayNotExist: true,
		Warning:         Alert().LessOrEqual(30).GreaterOrEqual(80).For(14 * 24 * time.Hour),
		PanelOptions:    PanelOptions().LegendFormat("{{name}}").Unit(Percentage).Max(100).Min(0),
		Owner:           owner,
		PossibleSolutions: strings.Replace(`
			- If usage is high:
				- **Kubernetes:** Consider increasing memory limits in the 'Deployment.yaml' for the {{CONTAINER_NAME}} service.
				- **Docker Compose:** Consider increasing 'memory:' of the {{CONTAINER_NAME}} container in 'docker-compose.yml'.
			- If usage is low, consider decreasing the above values.
		`, "{{CONTAINER_NAME}}", containerName, -1),
	}
}

// Golang monitoring overviews

var sharedGoGoroutines sharedObservable = func(containerName string, owner ObservableOwner) Observable {
	return Observable{
		Name:              "go_goroutines",
		Description:       "maximum active goroutines",
		Query:             fmt.Sprintf(`max by(instance) (go_goroutines{job=~".*%s"})`, containerName),
		DataMayNotExist:   true,
		Warning:           Alert().GreaterOrEqual(10000).For(10 * time.Minute),
		PanelOptions:      PanelOptions().LegendFormat("{{name}}"),
		Owner:             owner,
		PossibleSolutions: "none",
	}
}

var sharedGoGcDuration sharedObservable = func(containerName string, owner ObservableOwner) Observable {
	return Observable{
		Name:              "go_gc_duration_seconds",
		Description:       "maximum go garbage collection duration",
		Query:             fmt.Sprintf(`max by(instance) (go_gc_duration_seconds{job=~".*%s"})`, containerName),
		DataMayNotExist:   true,
		Warning:           Alert().GreaterOrEqual(2),
		PanelOptions:      PanelOptions().LegendFormat("{{name}}").Unit(Seconds),
		Owner:             owner,
		PossibleSolutions: "none",
	}
}

// Kubernetes monitoring overviews

var sharedKubernetesPodsAvailable sharedObservable = func(containerName string, owner ObservableOwner) Observable {
	return Observable{
		Name:              "pods_available_percentage",
		Description:       "percentage pods available",
		Query:             fmt.Sprintf(`sum by(app) (up{app=~".*%[1]s"}) / count by (app) (up{app=~".*%[1]s"}) * 100`, containerName),
		Critical:          Alert().LessOrEqual(90).For(10 * time.Minute),
		DataMayNotExist:   true,
		PanelOptions:      PanelOptions().LegendFormat("{{name}}").Unit(Percentage).Max(100).Min(0),
		Owner:             owner,
		PossibleSolutions: "none",
	}
}
