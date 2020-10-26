package main

import (
	"fmt"
	"time"
)

func ZoektWebServer() *Container {
	return &Container{
		Name:        "zoekt-webserver",
		Title:       "Zoekt Web Server",
		Description: "Serves indexed search requests using the search index.",
		Groups: []Group{
			{
				Title: "General",
				Rows: []Row{
					{
						{
							Name:              "indexed_search_request_errors",
							Description:       "indexed search request errors every 5m by code",
							Query:             `sum by (code)(increase(src_zoekt_request_duration_seconds_count{code!~"2.."}[5m])) / ignoring(code) group_left sum(increase(src_zoekt_request_duration_seconds_count[5m])) * 100`,
							DataMayNotExist:   true,
							Warning:           Alert().GreaterOrEqual(5).For(5 * time.Minute),
							PanelOptions:      PanelOptions().LegendFormat("{{code}}").Unit(Percentage),
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
						sharedContainerCPUUsage("zoekt-webserver", ObservableOwnerSearch),
						sharedContainerMemoryUsage("zoekt-webserver", ObservableOwnerSearch),
					},
					{
						sharedContainerRestarts("zoekt-webserver", ObservableOwnerSearch),
						sharedContainerFsInodes("zoekt-webserver", ObservableOwnerSearch),
					},
					{
						{
							Name:              "fs_io_operations",
							Description:       "filesystem reads and writes by instance rate over 1h",
							Query:             fmt.Sprintf(`sum by(name) (rate(container_fs_reads_total{%[1]s}[1h]) + rate(container_fs_writes_total{%[1]s}[1h]))`, promCadvisorContainerMatchers("zoekt-webserver")),
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
						sharedProvisioningCPUUsageLongTerm("zoekt-webserver", ObservableOwnerSearch),
						sharedProvisioningMemoryUsageLongTerm("zoekt-webserver", ObservableOwnerSearch),
					},
					{
						sharedProvisioningCPUUsageShortTerm("zoekt-webserver", ObservableOwnerSearch),
						sharedProvisioningMemoryUsageShortTerm("zoekt-webserver", ObservableOwnerSearch),
					},
				},
			},
			// kubernetes monitoring for zoekt-web-server is provided by zoekt-index-server,
			// since both services are deployed together
		},
	}
}
