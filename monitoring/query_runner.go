package main

func QueryRunner() *Container {
	return &Container{
		Name:        "query-runner",
		Title:       "Query Runner",
		Description: "Periodically runs saved searches and instructs the frontend to send out notifications.",
		Groups: []Group{
			{
				Title: "General",
				Rows: []Row{
					{
						sharedFrontendInternalAPIErrorResponses("query-runner", ObservableOwnerSearch),
					},
				},
			},
			{
				Title:  "Container monitoring (not available on server)",
				Hidden: true,
				Rows: []Row{
					{
						sharedContainerMemoryUsage("query-runner", ObservableOwnerSearch),
						sharedContainerCPUUsage("query-runner", ObservableOwnerSearch),
					},
					{
						sharedContainerRestarts("query-runner", ObservableOwnerSearch),
						sharedContainerFsInodes("query-runner", ObservableOwnerSearch),
					},
				},
			},
			{
				Title:  "Provisioning indicators (not available on server)",
				Hidden: true,
				Rows: []Row{
					{
						sharedProvisioningCPUUsageLongTerm("query-runner", ObservableOwnerSearch),
						sharedProvisioningMemoryUsageLongTerm("query-runner", ObservableOwnerSearch),
					},
					{
						sharedProvisioningCPUUsageShortTerm("query-runner", ObservableOwnerSearch),
						sharedProvisioningMemoryUsageShortTerm("query-runner", ObservableOwnerSearch),
					},
				},
			},
			{
				Title:  "Golang runtime monitoring",
				Hidden: true,
				Rows: []Row{
					{
						sharedGoGoroutines("query-runner", ObservableOwnerSearch),
						sharedGoGcDuration("query-runner", ObservableOwnerSearch),
					},
				},
			},
			{
				Title:  "Kubernetes monitoring (ignore if using Docker Compose or server)",
				Hidden: true,
				Rows: []Row{
					{
						sharedKubernetesPodsAvailable("query-runner", ObservableOwnerSearch),
					},
				},
			},
		},
	}
}
