package main

func PreciseCodeIntelBundleManager() *Container {
	return &Container{
		Name:        "precise-code-intel-bundle-manager",
		Title:       "Precise Code Intel Bundle Manager",
		Description: "Stores and manages precise code intelligence bundles.",
		Groups: []Group{
			{
				Title: "Database",
				Rows: []Row{
					{
						{
							Name:        "code_intel_frontend_db_store_99th_percentile_duration",
							Description: "99th percentile successful frontend database query duration over 5m",
							// TODO(efritz) - exclude error durations
							Query:             `histogram_quantile(0.99, sum by (le)(rate(src_code_intel_frontend_db_store_duration_seconds_bucket{job="precise-code-intel-bundle-manager"}[5m])))`,
							DataMayNotExist:   true,
							Warning:           Alert().GreaterOrEqual(20),
							PanelOptions:      PanelOptions().LegendFormat("store operation").Unit(Seconds),
							Owner:             ObservableOwnerCodeIntel,
							PossibleSolutions: "none",
						},
						{
							Name:              "code_intel_frontend_db_store_errors",
							Description:       "frontend database errors every 5m",
							Query:             `increase(src_code_intel_frontend_db_store_errors_total{job="precise-code-intel-bundle-manager"}[5m])`,
							DataMayNotExist:   true,
							Warning:           Alert().GreaterOrEqual(20),
							PanelOptions:      PanelOptions().LegendFormat("error"),
							Owner:             ObservableOwnerCodeIntel,
							PossibleSolutions: "none",
						},
					},
					{
						{
							Name:        "code_intel_codeintel_db_store_99th_percentile_duration",
							Description: "99th percentile successful codeintel database query duration over 5m",
							// TODO(efritz) - exclude error durations
							Query:             `histogram_quantile(0.99, sum by (le)(rate(src_code_intel_codeintel_db_store_duration_seconds_bucket{job="precise-code-intel-bundle-manager"}[5m])))`,
							DataMayNotExist:   true,
							Warning:           Alert().GreaterOrEqual(20),
							PanelOptions:      PanelOptions().LegendFormat("store operation").Unit(Seconds),
							Owner:             ObservableOwnerCodeIntel,
							PossibleSolutions: "none",
						},
						{
							Name:              "code_intel_codeintel_db_store_errors",
							Description:       "codeintel database every 5m",
							Query:             `increase(src_code_intel_codeintel_db_store_errors_total{job="precise-code-intel-bundle-manager"}[5m])`,
							DataMayNotExist:   true,
							Warning:           Alert().GreaterOrEqual(20),
							PanelOptions:      PanelOptions().LegendFormat("error"),
							Owner:             ObservableOwnerCodeIntel,
							PossibleSolutions: "none",
						},
					},
					{
						{
							Name:        "code_intel_bundle_store_99th_percentile_duration",
							Description: "99th percentile successful bundle database store operation duration over 5m",
							// TODO(efritz) - exclude error durations
							Query:             `histogram_quantile(0.99, sum by (le)(rate(src_code_intel_bundle_store_duration_seconds_bucket{job="precise-code-intel-bundle-manager"}[5m])))`,
							DataMayNotExist:   true,
							Warning:           Alert().GreaterOrEqual(20),
							PanelOptions:      PanelOptions().LegendFormat("store operation").Unit(Seconds),
							Owner:             ObservableOwnerCodeIntel,
							PossibleSolutions: "none",
						},
						{
							Name:              "code_intel_bundle_store_errors",
							Description:       "bundle store errors every 5m",
							Query:             `increase(src_code_intel_bundle_store_errors_total{job="precise-code-intel-bundle-manager"}[5m])`,
							DataMayNotExist:   true,
							Warning:           Alert().GreaterOrEqual(20),
							PanelOptions:      PanelOptions().LegendFormat("error"),
							Owner:             ObservableOwnerCodeIntel,
							PossibleSolutions: "none",
						},
					},
				},
			},
			{
				Title:  "Janitor - expires old data in the database and on disk",
				Hidden: true,
				Rows: []Row{
					{
						{
							Name:            "disk_space_remaining",
							Description:     "disk space remaining by instance",
							Query:           `(src_disk_space_available_bytes{job="precise-code-intel-bundle-manager"} / src_disk_space_total_bytes{job="precise-code-intel-bundle-manager"}) * 100`,
							DataMayNotExist: true,
							Warning:         Alert().LessOrEqual(25),
							Critical:        Alert().LessOrEqual(15),
							PanelOptions:    PanelOptions().LegendFormat("{{instance}}").Unit(Percentage),
							Owner:           ObservableOwnerCodeIntel,
							PossibleSolutions: `
									- **Provision more disk space:** Sourcegraph will begin deleting the oldest uploaded bundle files at 10% disk space remaining.
							`,
						},
					},
					{
						{
							Name:              "janitor_errors",
							Description:       "janitor errors every 5m",
							Query:             `sum(increase(src_bundle_manager_janitor_errors_total[5m]))`,
							DataMayNotExist:   true,
							Warning:           Alert().GreaterOrEqual(20),
							PanelOptions:      PanelOptions().LegendFormat("errors"),
							Owner:             ObservableOwnerCodeIntel,
							PossibleSolutions: "none",
						},
						{
							Name:              "janitor_upload_files_removed",
							Description:       "upload files removed (due to age) every 5m",
							Query:             `sum(increase(src_bundle_manager_janitor_upload_files_removed_total[5m]))`,
							DataMayNotExist:   true,
							Warning:           Alert().GreaterOrEqual(20),
							PanelOptions:      PanelOptions().LegendFormat("files removed"),
							Owner:             ObservableOwnerCodeIntel,
							PossibleSolutions: "none",
						},
						{
							Name:              "janitor_upload_part_files_removed",
							Description:       "upload part files removed (due to age) every 5m",
							Query:             `sum(increase(src_bundle_manager_janitor_upload_part_files_removed_total[5m]))`,
							DataMayNotExist:   true,
							Warning:           Alert().GreaterOrEqual(20),
							PanelOptions:      PanelOptions().LegendFormat("files removed"),
							Owner:             ObservableOwnerCodeIntel,
							PossibleSolutions: "none",
						},
					},
					{
						{
							Name:              "janitor_upload_records_removed",
							Description:       "upload records removed every 5m",
							Query:             `sum(increase(src_bundle_manager_janitor_upload_records_removed_total[5m]))`,
							DataMayNotExist:   true,
							Warning:           Alert().GreaterOrEqual(20),
							PanelOptions:      PanelOptions().LegendFormat("records removed"),
							Owner:             ObservableOwnerCodeIntel,
							PossibleSolutions: "none",
						},
						{
							Name:              "janitor_data_rows_removed",
							Description:       "codeintel database rows removed (due to deleted upload) every 5m",
							Query:             `sum(increase(src_bundle_manager_janitor_data_rows_removed_total[5m]))`,
							DataMayNotExist:   true,
							Warning:           Alert().GreaterOrEqual(20),
							PanelOptions:      PanelOptions().LegendFormat("records removed"),
							Owner:             ObservableOwnerCodeIntel,
							PossibleSolutions: "none",
						},
					},
				},
			},
			{
				Title:  "Internal service requests",
				Hidden: true,
				Rows: []Row{
					{
						sharedFrontendInternalAPIErrorResponses("precise-code-intel-bundle-manager", ObservableOwnerCodeIntel),
					},
				},
			},
			{
				Title:  "Container monitoring (not available on server)",
				Hidden: true,
				Rows: []Row{
					{
						sharedContainerCPUUsage("precise-code-intel-bundle-manager", ObservableOwnerCodeIntel),
						sharedContainerMemoryUsage("precise-code-intel-bundle-manager", ObservableOwnerCodeIntel),
					},
					{
						sharedContainerRestarts("precise-code-intel-bundle-manager", ObservableOwnerCodeIntel),
						sharedContainerFsInodes("precise-code-intel-bundle-manager", ObservableOwnerCodeIntel),
					},
				},
			},
			{
				Title:  "Provisioning indicators (not available on server)",
				Hidden: true,
				Rows: []Row{
					{
						sharedProvisioningCPUUsageLongTerm("precise-code-intel-bundle-manager", ObservableOwnerCodeIntel),
						sharedProvisioningMemoryUsageLongTerm("precise-code-intel-bundle-manager", ObservableOwnerCodeIntel),
					},
					{
						sharedProvisioningCPUUsageShortTerm("precise-code-intel-bundle-manager", ObservableOwnerCodeIntel),
						sharedProvisioningMemoryUsageShortTerm("precise-code-intel-bundle-manager", ObservableOwnerCodeIntel),
					},
				},
			},
			{
				Title:  "Golang runtime monitoring",
				Hidden: true,
				Rows: []Row{
					{
						sharedGoGoroutines("precise-code-intel-bundle-manager", ObservableOwnerCodeIntel),
						sharedGoGcDuration("precise-code-intel-bundle-manager", ObservableOwnerCodeIntel),
					},
				},
			},
			{
				Title:  "Kubernetes monitoring (ignore if using Docker Compose or server)",
				Hidden: true,
				Rows: []Row{
					{
						sharedKubernetesPodsAvailable("precise-code-intel-bundle-manager", ObservableOwnerCodeIntel),
					},
				},
			},
		},
	}
}
