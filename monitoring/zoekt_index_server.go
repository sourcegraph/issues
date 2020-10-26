package main

import "fmt"

func ZoektIndexServer() *Container {
	return &Container{
		Name:        "zoekt-indexserver",
		Title:       "Zoekt Index Server",
		Description: "Indexes repositories and populates the search index.",
		Groups: []Group{
			{
				Title: "General",
				Rows: []Row{
					{
						{
							Name:              "average_resolve_revision_duration",
							Description:       "average resolve revision duration over 5m",
							Query:             `sum(rate(resolve_revision_seconds_sum[5m])) / sum(rate(resolve_revision_seconds_count[5m]))`,
							DataMayNotExist:   true,
							Warning:           Alert().GreaterOrEqual(15),
							Critical:          Alert().GreaterOrEqual(30),
							PanelOptions:      PanelOptions().LegendFormat("{{duration}}").Unit(Seconds),
							Owner:             ObservableOwnerSearch,
							PossibleSolutions: "none",
						},
					},
				},
			},
			{
				Title:  "Container monitoring (not available on server)",
				Hidden: true,
				Rows: []Row{
					{
						sharedContainerCPUUsage("zoekt-indexserver", ObservableOwnerSearch),
						sharedContainerMemoryUsage("zoekt-indexserver", ObservableOwnerSearch),
					},
					{
						sharedContainerRestarts("zoekt-indexserver", ObservableOwnerSearch),
						sharedContainerFsInodes("zoekt-indexserver", ObservableOwnerSearch),
					},
					{
						{
							Name:              "fs_io_operations",
							Description:       "filesystem reads and writes rate by instance over 1h",
							Query:             fmt.Sprintf(`sum by(name) (rate(container_fs_reads_total{%[1]s}[1h]) + rate(container_fs_writes_total{%[1]s}[1h]))`, promCadvisorContainerMatchers("zoekt-indexserver")),
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
						sharedProvisioningCPUUsageLongTerm("zoekt-indexserver", ObservableOwnerSearch),
						sharedProvisioningMemoryUsageLongTerm("zoekt-indexserver", ObservableOwnerSearch),
					},
					{
						sharedProvisioningCPUUsageShortTerm("zoekt-indexserver", ObservableOwnerSearch),
						sharedProvisioningMemoryUsageShortTerm("zoekt-indexserver", ObservableOwnerSearch),
					},
				},
			},
			{
				Title:  "Kubernetes monitoring (ignore if using Docker Compose or server)",
				Hidden: true,
				Rows: []Row{
					{
						// zoekt_index_server, zoekt_web_server are deployed together
						// as part of the indexed-search service, so only show pod
						// availability here.
						sharedKubernetesPodsAvailable("indexed-search", ObservableOwnerSearch),
					},
				},
			},
		},
	}
}
