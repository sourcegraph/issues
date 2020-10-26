package main

import "time"

func GitHubProxy() *Container {
	return &Container{
		Name:        "github-proxy",
		Title:       "GitHub Proxy",
		Description: "Proxies all requests to github.com, keeping track of and managing rate limits.",
		Groups: []Group{
			{
				Title: "GitHub API monitoring",
				Rows: []Row{
					{
						{
							Name:              "github_core_rate_limit_remaining",
							Description:       "remaining calls to GitHub before hitting the rate limit",
							Query:             `src_github_rate_limit_remaining{resource="core"}`,
							DataMayNotExist:   true,
							Critical:          Alert().LessOrEqual(500).For(5 * time.Minute),
							PanelOptions:      PanelOptions().LegendFormat("calls remaining"),
							Owner:             ObservableOwnerCloud,
							PossibleSolutions: `Try restarting the pod to get a different public IP.`,
						},
						{
							Name:              "github_search_rate_limit_remaining",
							Description:       "remaining calls to GitHub search before hitting the rate limit",
							Query:             `src_github_rate_limit_remaining{resource="search"}`,
							DataMayNotExist:   true,
							Warning:           Alert().LessOrEqual(5),
							PanelOptions:      PanelOptions().LegendFormat("calls remaining"),
							Owner:             ObservableOwnerCloud,
							PossibleSolutions: `Try restarting the pod to get a different public IP.`,
						},
					},
				},
			},
			{
				Title:  "Container monitoring (not available on server)",
				Hidden: true,
				Rows: []Row{
					{
						sharedContainerCPUUsage("github-proxy", ObservableOwnerCloud),
						sharedContainerMemoryUsage("github-proxy", ObservableOwnerCloud),
					},
					{
						sharedContainerRestarts("github-proxy", ObservableOwnerCloud),
						sharedContainerFsInodes("github-proxy", ObservableOwnerCloud),
					},
				},
			},
			{
				Title:  "Provisioning indicators (not available on server)",
				Hidden: true,
				Rows: []Row{
					{
						sharedProvisioningCPUUsageLongTerm("github-proxy", ObservableOwnerCloud),
						sharedProvisioningMemoryUsageLongTerm("github-proxy", ObservableOwnerCloud),
					},
					{
						sharedProvisioningCPUUsageShortTerm("github-proxy", ObservableOwnerCloud),
						sharedProvisioningMemoryUsageShortTerm("github-proxy", ObservableOwnerCloud),
					},
				},
			},
			{
				Title:  "Golang runtime monitoring",
				Hidden: true,
				Rows: []Row{
					{
						sharedGoGoroutines("github-proxy", ObservableOwnerCloud),
						sharedGoGcDuration("github-proxy", ObservableOwnerCloud),
					},
				},
			},
			{
				Title:  "Kubernetes monitoring (ignore if using Docker Compose or server)",
				Hidden: true,
				Rows: []Row{
					{
						sharedKubernetesPodsAvailable("github-proxy", ObservableOwnerCloud),
					},
				},
			},
		},
	}
}
