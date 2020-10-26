package main

func Prometheus() *Container {
	return &Container{
		Name:        "prometheus",
		Title:       "Prometheus",
		Description: "Sourcegraph's all-in-one Prometheus and Alertmanager service.",
		Groups: []Group{
			{
				Title: "Metrics",
				Rows: []Row{
					{
						{
							Name:              "prometheus_metrics_bloat",
							Description:       "prometheus metrics payload size",
							Query:             `http_response_size_bytes{handler="prometheus",job!="kubernetes-apiservers",job!="kubernetes-nodes",quantile="0.5"}`,
							DataMayNotExist:   true,
							Warning:           Alert().GreaterOrEqual(20000),
							PanelOptions:      PanelOptions().Unit(Bytes).LegendFormat("{{instance}}"),
							Owner:             ObservableOwnerDistribution,
							PossibleSolutions: "none",
						},
					},
				},
			},
			{
				Title: "Alerts",
				Rows: []Row{
					{
						{
							Name:              "alertmanager_notifications_failed_total",
							Description:       "failed alertmanager notifications over 1m",
							Query:             `sum by(integration) (rate(alertmanager_notifications_failed_total[1m]))`,
							DataMayNotExist:   true,
							Warning:           Alert().GreaterOrEqual(1),
							PanelOptions:      PanelOptions().LegendFormat("{{integration}}"),
							Owner:             ObservableOwnerDistribution,
							PossibleSolutions: "Ensure that your [`observability.alerts` configuration](https://docs.sourcegraph.com/admin/observability/alerting#setting-up-alerting) (in site configuration) is valid.",
						},
					},
				},
			},
			{
				Title:  "Container monitoring (not available on server)",
				Hidden: true,
				Rows: []Row{
					{
						sharedContainerCPUUsage("prometheus", ObservableOwnerDistribution),
						sharedContainerMemoryUsage("prometheus", ObservableOwnerDistribution),
					},
					{
						sharedContainerRestarts("prometheus", ObservableOwnerDistribution),
						sharedContainerFsInodes("prometheus", ObservableOwnerDistribution),
					},
				},
			},
			{
				Title:  "Provisioning indicators (not available on server)",
				Hidden: true,
				Rows: []Row{
					{
						sharedProvisioningCPUUsageLongTerm("prometheus", ObservableOwnerDistribution),
						sharedProvisioningMemoryUsageLongTerm("prometheus", ObservableOwnerDistribution),
					},
					{
						sharedProvisioningCPUUsageShortTerm("prometheus", ObservableOwnerDistribution),
						sharedProvisioningMemoryUsageShortTerm("prometheus", ObservableOwnerDistribution),
					},
				},
			},
			{
				Title:  "Kubernetes monitoring (ignore if using Docker Compose or server)",
				Hidden: true,
				Rows: []Row{
					{
						sharedKubernetesPodsAvailable("prometheus", ObservableOwnerDistribution),
					},
				},
			},
		},
	}
}
