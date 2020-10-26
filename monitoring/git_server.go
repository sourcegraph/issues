package main

import (
	"fmt"
	"strings"
	"time"
)

func GitServer() *Container {
	return &Container{
		Name:        "gitserver",
		Title:       "Git Server",
		Description: "Stores, manages, and operates Git repositories.",
		Groups: []Group{
			{
				Title: "General",
				Rows: []Row{
					{
						{
							Name:            "disk_space_remaining",
							Description:     "disk space remaining by instance",
							Query:           `(src_gitserver_disk_space_available / src_gitserver_disk_space_total) * 100`,
							DataMayNotExist: true,
							Warning:         Alert().LessOrEqual(25),
							Critical:        Alert().LessOrEqual(15),
							PanelOptions:    PanelOptions().LegendFormat("{{instance}}").Unit(Percentage),
							Owner:           ObservableOwnerCloud,
							PossibleSolutions: `
								- **Provision more disk space:** Sourcegraph will begin deleting least-used repository clones at 10% disk space remaining which may result in decreased performance, users having to wait for repositories to clone, etc.
							`,
						},
						{
							Name:            "running_git_commands",
							Description:     "running git commands (signals load)",
							Query:           "max(src_gitserver_exec_running)",
							DataMayNotExist: true,
							Warning:         Alert().GreaterOrEqual(50).For(2 * time.Minute),
							Critical:        Alert().GreaterOrEqual(100).For(5 * time.Minute),
							PanelOptions:    PanelOptions().LegendFormat("running commands"),
							Owner:           ObservableOwnerCloud,
							PossibleSolutions: `
								- **Check if the problem may be an intermittent and temporary peak** using the "Container monitoring" section at the bottom of the Git Server dashboard.
								- **Single container deployments:** Consider upgrading to a [Docker Compose deployment](../install/docker-compose/migrate.md) which offers better scalability and resource isolation.
								- **Kubernetes and Docker Compose:** Check that you are running a similar number of git server replicas and that their CPU/memory limits are allocated according to what is shown in the [Sourcegraph resource estimator](../install/resource_estimator.md).
							`,
						},
					}, {
						{
							Name:            "repository_clone_queue_size",
							Description:     "repository clone queue size",
							Query:           "sum(src_gitserver_clone_queue)",
							DataMayNotExist: true,
							Warning:         Alert().GreaterOrEqual(25),
							PanelOptions:    PanelOptions().LegendFormat("queue size"),
							Owner:           ObservableOwnerCloud,
							PossibleSolutions: `
								- **If you just added several repositories**, the warning may be expected.
								- **Check which repositories need cloning**, by visiting e.g. https://sourcegraph.example.com/site-admin/repositories?filter=not-cloned
							`,
						},
						{
							Name:            "repository_existence_check_queue_size",
							Description:     "repository existence check queue size",
							Query:           "sum(src_gitserver_lsremote_queue)",
							DataMayNotExist: true,
							Warning:         Alert().GreaterOrEqual(25),
							PanelOptions:    PanelOptions().LegendFormat("queue size"),
							Owner:           ObservableOwnerCloud,
							PossibleSolutions: `
								- **Check the code host status indicator for errors:** on the Sourcegraph app homepage, when signed in as an admin click the cloud icon in the top right corner of the page.
								- **Check if the issue continues to happen after 30 minutes**, it may be temporary.
								- **Check the gitserver logs for more information.**
							`,
						},
					}, {
						{
							Name:            "echo_command_duration_test",
							Description:     "echo command duration test",
							Query:           "max(src_gitserver_echo_duration_seconds)",
							DataMayNotExist: true,
							Warning:         Alert().GreaterOrEqual(1),
							Critical:        Alert().GreaterOrEqual(2),
							PanelOptions:    PanelOptions().LegendFormat("running commands").Unit(Seconds),
							Owner:           ObservableOwnerCloud,
							PossibleSolutions: `
								- **Query a graph for individual commands** using 'sum by (cmd)(src_gitserver_exec_running)' in Grafana ('/-/debug/grafana') to see if a command might be spiking in frequency.
								- **Check if the problem may be an intermittent and temporary peak** using the "Container monitoring" section at the bottom of the Git Server dashboard.
								- **Single container deployments:** Consider upgrading to a [Docker Compose deployment](../install/docker-compose/migrate.md) which offers better scalability and resource isolation.
								- **Kubernetes and Docker Compose:** Check that you are running a similar number of git server replicas and that their CPU/memory limits are allocated according to what is shown in the [Sourcegraph resource estimator](../install/resource_estimator.md).
							`,
						},
						sharedFrontendInternalAPIErrorResponses("gitserver", ObservableOwnerCloud),
					},
				},
			},
			{
				Title:  "Container monitoring (not available on server)",
				Hidden: true,
				Rows: []Row{
					{
						sharedContainerCPUUsage("gitserver", ObservableOwnerCloud),
						sharedContainerMemoryUsage("gitserver", ObservableOwnerCloud),
					},
					{
						sharedContainerRestarts("gitserver", ObservableOwnerCloud),
						sharedContainerFsInodes("gitserver", ObservableOwnerCloud),
					},
					{
						{
							Name:              "fs_io_operations",
							Description:       "filesystem reads and writes rate by instance over 1h",
							Query:             fmt.Sprintf(`sum by(name) (rate(container_fs_reads_total{%[1]s}[1h]) + rate(container_fs_writes_total{%[1]s}[1h]))`, promCadvisorContainerMatchers("gitserver")),
							DataMayNotExist:   true,
							Warning:           Alert().GreaterOrEqual(5000),
							PanelOptions:      PanelOptions().LegendFormat("{{name}}"),
							Owner:             ObservableOwnerSearch,
							PossibleSolutions: "none",
						},
					},
				},
			},
			{
				Title:  "Provisioning indicators (not available on server)",
				Hidden: true,
				Rows: []Row{
					{
						sharedProvisioningCPUUsageLongTerm("gitserver", ObservableOwnerCloud),
						// gitserver generally uses up all the memory it gets, so
						// alerting on long-term high memory usage is not very useful
						{
							Name:            "provisioning_container_memory_usage_long_term",
							Description:     "container memory usage (1d maximum) by instance",
							Query:           fmt.Sprintf(`max_over_time(cadvisor_container_memory_usage_percentage_total{%s}[1d])`, promCadvisorContainerMatchers("gitserver")),
							DataMayNotExist: true,
							Warning:         Alert().LessOrEqual(30).For(14 * 24 * time.Hour),
							PanelOptions:    PanelOptions().LegendFormat("{{name}}").Unit(Percentage).Max(100).Min(0),
							Owner:           ObservableOwnerDistribution,
							PossibleSolutions: strings.Replace(`
								- If usage is high:
									- **Kubernetes:** Consider increasing memory limits in the 'Deployment.yaml' for the {{CONTAINER_NAME}} service.
									- **Docker Compose:** Consider increasing 'memory:' of the {{CONTAINER_NAME}} container in 'docker-compose.yml'.
								- If usage is low, consider decreasing the above values.
							`, "{{CONTAINER_NAME}}", "gitserver", -1),
						},
					},
					{
						sharedProvisioningCPUUsageShortTerm("gitserver", ObservableOwnerCloud),
					},
				},
			},
			{
				Title:  "Golang runtime monitoring",
				Hidden: true,
				Rows: []Row{
					{
						sharedGoGoroutines("gitserver", ObservableOwnerCloud),
						sharedGoGcDuration("gitserver", ObservableOwnerCloud),
					},
				},
			},
			{
				Title:  "Kubernetes monitoring (ignore if using Docker Compose or server)",
				Hidden: true,
				Rows: []Row{
					{
						sharedKubernetesPodsAvailable("gitserver", ObservableOwnerCloud),
					},
				},
			},
		},
	}
}
