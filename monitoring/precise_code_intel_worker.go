package main

import "time"

func PreciseCodeIntelWorker() *Container {
	return &Container{
		Name:        "precise-code-intel-worker",
		Title:       "Precise Code Intel Worker",
		Description: "Handles conversion of uploaded precise code intelligence bundles.",
		Groups: []Group{
			{
				Title: "General",
				Rows: []Row{
					{
						{
							Name:              "upload_queue_size",
							Description:       "upload queue size",
							Query:             `max(src_upload_queue_uploads_total)`,
							DataMayNotExist:   true,
							Warning:           Alert().GreaterOrEqual(100),
							PanelOptions:      PanelOptions().LegendFormat("uploads queued for processing"),
							Owner:             ObservableOwnerCodeIntel,
							PossibleSolutions: "none",
						},
						{
							Name:            "upload_queue_growth_rate",
							Description:     "upload queue growth rate every 5m",
							Query:           `sum(increase(src_upload_queue_uploads_total[30m])) / sum(increase(src_upload_queue_processor_total[30m]))`,
							DataMayNotExist: true,

							Warning:           Alert().GreaterOrEqual(5),
							PanelOptions:      PanelOptions().LegendFormat("upload queue growth rate"),
							Owner:             ObservableOwnerCodeIntel,
							PossibleSolutions: "none",
						},
						{
							Name:        "upload_process_errors",
							Description: "upload process errors every 5m",
							// TODO(efritz) - ensure these differentiate malformed dumps and system errors
							Query:             `sum(increase(src_upload_queue_processor_errors_total[5m]))`,
							DataMayNotExist:   true,
							Warning:           Alert().GreaterOrEqual(20),
							PanelOptions:      PanelOptions().LegendFormat("errors"),
							Owner:             ObservableOwnerCodeIntel,
							PossibleSolutions: "none",
						},
					},
				},
			},
			{
				Title: "Database",
				Rows: []Row{
					{
						{
							Name:        "code_intel_frontend_db_store_99th_percentile_duration",
							Description: "99th percentile successful frontend database query duration over 5m",
							// TODO(efritz) - exclude error durations
							Query:             `histogram_quantile(0.99, sum by (le)(rate(src_code_intel_frontend_db_store_duration_seconds_bucket{job="precise-code-intel-worker"}[5m])))`,
							DataMayNotExist:   true,
							Warning:           Alert().GreaterOrEqual(20),
							PanelOptions:      PanelOptions().LegendFormat("store operation").Unit(Seconds),
							Owner:             ObservableOwnerCodeIntel,
							PossibleSolutions: "none",
						},
						{
							Name:              "code_intel_frontend_db_store_errors",
							Description:       "frontend database errors every 5m",
							Query:             `increase(src_code_intel_frontend_db_store_errors_total{job="precise-code-intel-worker"}[5m])`,
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
							Query:             `histogram_quantile(0.99, sum by (le)(rate(src_code_intel_codeintel_db_store_duration_seconds_bucket{job="precise-code-intel-worker"}[5m])))`,
							DataMayNotExist:   true,
							Warning:           Alert().GreaterOrEqual(20),
							PanelOptions:      PanelOptions().LegendFormat("store operation").Unit(Seconds),
							Owner:             ObservableOwnerCodeIntel,
							PossibleSolutions: "none",
						},
						{
							Name:              "code_intel_codeintel_db_store_errors",
							Description:       "codeintel database errors every 5m",
							Query:             `increase(src_code_intel_codeintel_db_store_errors_total{job="precise-code-intel-worker"}[5m])`,
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
				Title:  "Upload resetter - re-queues uploads that did not complete processing",
				Hidden: true,
				Rows: []Row{
					{
						{
							Name:              "processing_uploads_reset",
							Description:       "uploads reset to queued state every 5m",
							Query:             `sum(increase(src_upload_queue_resets_total[5m]))`,
							DataMayNotExist:   true,
							Warning:           Alert().GreaterOrEqual(20),
							PanelOptions:      PanelOptions().LegendFormat("uploads"),
							Owner:             ObservableOwnerCodeIntel,
							PossibleSolutions: "none",
						},
						{
							Name:              "processing_uploads_reset_failures",
							Description:       "uploads errored after repeated resets every 5m",
							Query:             `sum(increase(src_upload_queue_max_resets_total[5m]))`,
							DataMayNotExist:   true,
							Warning:           Alert().GreaterOrEqual(20),
							PanelOptions:      PanelOptions().LegendFormat("uploads"),
							Owner:             ObservableOwnerCodeIntel,
							PossibleSolutions: "none",
						},
						{
							Name:              "upload_resetter_errors",
							Description:       "upload resetter errors every 5m",
							Query:             `sum(increase(src_upload_queue_reset_errors_total[5m]))`,
							DataMayNotExist:   true,
							Warning:           Alert().GreaterOrEqual(20),
							PanelOptions:      PanelOptions().LegendFormat("errors"),
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
						{
							Name:              "99th_percentile_bundle_manager_transfer_duration",
							Description:       "99th percentile successful bundle manager data transfer duration over 5m",
							Query:             `histogram_quantile(0.99, sum by (le,category)(rate(src_precise_code_intel_bundle_manager_request_duration_seconds_bucket{job="precise-code-intel-worker",category="transfer"}[5m])))`,
							DataMayNotExist:   true,
							Warning:           Alert().GreaterOrEqual(300),
							PanelOptions:      PanelOptions().LegendFormat("{{category}}").Unit(Seconds),
							Owner:             ObservableOwnerCodeIntel,
							PossibleSolutions: "none",
						},
						{
							Name:              "bundle_manager_error_responses",
							Description:       "bundle manager error responses every 5m",
							Query:             `sum by (category)(increase(src_precise_code_intel_bundle_manager_request_duration_seconds_count{job="precise-code-intel-worker",code!~"2.."}[5m]))`,
							DataMayNotExist:   true,
							Warning:           Alert().GreaterOrEqual(5),
							PanelOptions:      PanelOptions().LegendFormat("{{category}}"),
							Owner:             ObservableOwnerCodeIntel,
							PossibleSolutions: "none",
						},
					},
					{
						{
							Name:              "99th_percentile_gitserver_duration",
							Description:       "99th percentile successful gitserver query duration over 5m",
							Query:             `histogram_quantile(0.99, sum by (le,category)(rate(src_gitserver_request_duration_seconds_bucket{job="precise-code-intel-worker"}[5m])))`,
							DataMayNotExist:   true,
							Warning:           Alert().GreaterOrEqual(20),
							PanelOptions:      PanelOptions().LegendFormat("{{category}}").Unit(Seconds),
							Owner:             ObservableOwnerCodeIntel,
							PossibleSolutions: "none",
						},
						{
							Name:            "gitserver_error_responses",
							Description:     "gitserver error responses every 5m",
							Query:           `sum by (category)(increase(src_gitserver_request_duration_seconds_count{job="precise-code-intel-worker",code!~"2.."}[5m])) / ignoring(code) group_left sum by (category)(increase(src_gitserver_request_duration_seconds_count{job="precise-code-intel-worker"}[5m])) * 100`,
							DataMayNotExist: true,

							Warning:           Alert().GreaterOrEqual(5).For(15 * time.Minute),
							PanelOptions:      PanelOptions().LegendFormat("{{category}}").Unit(Percentage),
							Owner:             ObservableOwnerCodeIntel,
							PossibleSolutions: "none",
						},
					},
					{
						sharedFrontendInternalAPIErrorResponses("precise-code-intel-worker", ObservableOwnerCodeIntel),
					},
				},
			},
			{
				Title:  "Container monitoring (not available on server)",
				Hidden: true,
				Rows: []Row{
					{
						sharedContainerCPUUsage("precise-code-intel-worker", ObservableOwnerCodeIntel),
						sharedContainerMemoryUsage("precise-code-intel-worker", ObservableOwnerCodeIntel),
					},
					{
						sharedContainerRestarts("precise-code-intel-worker", ObservableOwnerCodeIntel),
						sharedContainerFsInodes("precise-code-intel-worker", ObservableOwnerCodeIntel),
					},
				},
			},
			{
				Title:  "Provisioning indicators (not available on server)",
				Hidden: true,
				Rows: []Row{
					{
						sharedProvisioningCPUUsageLongTerm("precise-code-intel-worker", ObservableOwnerCodeIntel),
						sharedProvisioningMemoryUsageLongTerm("precise-code-intel-worker", ObservableOwnerCodeIntel),
					},
					{
						sharedProvisioningCPUUsageShortTerm("precise-code-intel-worker", ObservableOwnerCodeIntel),
						sharedProvisioningMemoryUsageShortTerm("precise-code-intel-worker", ObservableOwnerCodeIntel),
					},
				},
			},
			{
				Title:  "Golang runtime monitoring",
				Hidden: true,
				Rows: []Row{
					{
						sharedGoGoroutines("precise-code-intel-worker", ObservableOwnerCodeIntel),
						sharedGoGcDuration("precise-code-intel-worker", ObservableOwnerCodeIntel),
					},
				},
			},
			{
				Title:  "Kubernetes monitoring (ignore if using Docker Compose or server)",
				Hidden: true,
				Rows: []Row{
					{
						sharedKubernetesPodsAvailable("precise-code-intel-worker", ObservableOwnerCodeIntel),
					},
				},
			},
		},
	}
}
