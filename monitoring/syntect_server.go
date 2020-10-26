package main

import "time"

func SyntectServer() *Container {
	return &Container{
		Name:        "syntect-server",
		Title:       "Syntect Server",
		Description: "Handles syntax highlighting for code files.",
		Groups: []Group{
			{
				Title: "General",
				Rows: []Row{
					{
						{
							Name:              "syntax_highlighting_errors",
							Description:       "syntax highlighting errors every 5m",
							Query:             `sum(increase(src_syntax_highlighting_requests{status="error"}[5m])) / sum(increase(src_syntax_highlighting_requests[5m])) * 100`,
							DataMayNotExist:   true,
							Warning:           Alert().GreaterOrEqual(5).For(5 * time.Minute),
							PanelOptions:      PanelOptions().LegendFormat("error").Unit(Percentage),
							Owner:             ObservableOwnerCodeIntel,
							PossibleSolutions: "none",
						},
						{
							Name:              "syntax_highlighting_timeouts",
							Description:       "syntax highlighting timeouts every 5m",
							Query:             `sum(increase(src_syntax_highlighting_requests{status="timeout"}[5m])) / sum(increase(src_syntax_highlighting_requests[5m])) * 100`,
							DataMayNotExist:   true,
							Warning:           Alert().GreaterOrEqual(5).For(5 * time.Minute),
							PanelOptions:      PanelOptions().LegendFormat("timeout").Unit(Percentage),
							Owner:             ObservableOwnerCodeIntel,
							PossibleSolutions: "none",
						},
					},
					{
						{
							Name:              "syntax_highlighting_panics",
							Description:       "syntax highlighting panics every 5m",
							Query:             `sum(increase(src_syntax_highlighting_requests{status="panic"}[5m]))`,
							DataMayNotExist:   true,
							Warning:           Alert().GreaterOrEqual(5),
							PanelOptions:      PanelOptions().LegendFormat("panic"),
							Owner:             ObservableOwnerCodeIntel,
							PossibleSolutions: "none",
						},
						{
							Name:              "syntax_highlighting_worker_deaths",
							Description:       "syntax highlighter worker deaths every 5m",
							Query:             `sum(increase(src_syntax_highlighting_requests{status="hss_worker_timeout"}[5m]))`,
							DataMayNotExist:   true,
							Warning:           Alert().GreaterOrEqual(1),
							PanelOptions:      PanelOptions().LegendFormat("worker death"),
							Owner:             ObservableOwnerCodeIntel,
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
						sharedContainerCPUUsage("syntect-server", ObservableOwnerCodeIntel),
						sharedContainerMemoryUsage("syntect-server", ObservableOwnerCodeIntel),
					},
					{
						sharedContainerRestarts("syntect-server", ObservableOwnerCodeIntel),
						sharedContainerFsInodes("syntect-server", ObservableOwnerCodeIntel),
					},
				},
			},
			{
				Title:  "Provisioning indicators (not available on server)",
				Hidden: true,
				Rows: []Row{
					{
						sharedProvisioningCPUUsageLongTerm("syntect-server", ObservableOwnerCodeIntel),
						sharedProvisioningMemoryUsageLongTerm("syntect-server", ObservableOwnerCodeIntel),
					},
					{
						sharedProvisioningCPUUsageShortTerm("syntect-server", ObservableOwnerCodeIntel),
						sharedProvisioningMemoryUsageShortTerm("syntect-server", ObservableOwnerCodeIntel),
					},
				},
			},
			{
				Title:  "Kubernetes monitoring (ignore if using Docker Compose or server)",
				Hidden: true,
				Rows: []Row{
					{
						sharedKubernetesPodsAvailable("syntect-server", ObservableOwnerCodeIntel),
					},
				},
			},
		},
	}
}
