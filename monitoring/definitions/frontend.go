package definitions

import (
	"time"

	"github.com/sourcegraph/sourcegraph/monitoring/definitions/shared"
	"github.com/sourcegraph/sourcegraph/monitoring/monitoring"
)

func Frontend() *monitoring.Container {
	// frontend is sometimes called sourcegraph-frontend in various contexts
	const containerName = "(frontend|sourcegraph-frontend)"

	return &monitoring.Container{
		Name:        "frontend",
		Title:       "Frontend",
		Description: "Serves all end-user browser and API requests.",
		Groups: []monitoring.Group{
			{
				Title: "Search at a glance",
				Rows: []monitoring.Row{
					{
						{
							Name:        "99th_percentile_search_request_duration",
							Description: "99th percentile successful search request duration over 5m",
							Query:       `histogram_quantile(0.99, sum by (le)(rate(src_graphql_field_seconds_bucket{type="Search",field="results",error="false",source="browser",request_name!="CodeIntelSearch"}[5m])))`,

							Warning: monitoring.Alert().GreaterOrEqual(20, nil),
							Panel:   monitoring.Panel().LegendFormat("duration").Unit(monitoring.Seconds),
							Owner:   monitoring.ObservableOwnerSearch,
							PossibleSolutions: `
								- **Get details on the exact queries that are slow** by configuring '"observability.logSlowSearches": 20,' in the site configuration and looking for 'frontend' warning logs prefixed with 'slow search request' for additional details.
								- **Check that most repositories are indexed** by visiting https://sourcegraph.example.com/site-admin/repositories?filter=needs-index (it should show few or no results.)
								- **Kubernetes:** Check CPU usage of zoekt-webserver in the indexed-search pod, consider increasing CPU limits in the 'indexed-search.Deployment.yaml' if regularly hitting max CPU utilization.
								- **Docker Compose:** Check CPU usage on the Zoekt Web Server dashboard, consider increasing 'cpus:' of the zoekt-webserver container in 'docker-compose.yml' if regularly hitting max CPU utilization.
							`,
						},
						{
							Name:        "90th_percentile_search_request_duration",
							Description: "90th percentile successful search request duration over 5m",
							Query:       `histogram_quantile(0.90, sum by (le)(rate(src_graphql_field_seconds_bucket{type="Search",field="results",error="false",source="browser",request_name!="CodeIntelSearch"}[5m])))`,

							Warning: monitoring.Alert().GreaterOrEqual(15, nil),
							Panel:   monitoring.Panel().LegendFormat("duration").Unit(monitoring.Seconds),
							Owner:   monitoring.ObservableOwnerSearch,
							PossibleSolutions: `
								- **Get details on the exact queries that are slow** by configuring '"observability.logSlowSearches": 15,' in the site configuration and looking for 'frontend' warning logs prefixed with 'slow search request' for additional details.
								- **Check that most repositories are indexed** by visiting https://sourcegraph.example.com/site-admin/repositories?filter=needs-index (it should show few or no results.)
								- **Kubernetes:** Check CPU usage of zoekt-webserver in the indexed-search pod, consider increasing CPU limits in the 'indexed-search.Deployment.yaml' if regularly hitting max CPU utilization.
								- **Docker Compose:** Check CPU usage on the Zoekt Web Server dashboard, consider increasing 'cpus:' of the zoekt-webserver container in 'docker-compose.yml' if regularly hitting max CPU utilization.
							`,
						},
					},
					{
						{
							Name:        "hard_timeout_search_responses",
							Description: "hard timeout search responses every 5m",
							Query:       `(sum(increase(src_graphql_search_response{status="timeout",source="browser",request_name!="CodeIntelSearch"}[5m])) + sum(increase(src_graphql_search_response{status="alert",alert_type="timed_out",source="browser",request_name!="CodeIntelSearch"}[5m]))) / sum(increase(src_graphql_search_response{source="browser",request_name!="CodeIntelSearch"}[5m])) * 100`,

							Warning:           monitoring.Alert().GreaterOrEqual(2, nil).For(15 * time.Minute),
							Critical:          monitoring.Alert().GreaterOrEqual(5, nil).For(15 * time.Minute),
							Panel:             monitoring.Panel().LegendFormat("hard timeout").Unit(monitoring.Percentage),
							Owner:             monitoring.ObservableOwnerSearch,
							PossibleSolutions: "none",
						},
						{
							Name:        "hard_error_search_responses",
							Description: "hard error search responses every 5m",
							Query:       `sum by (status)(increase(src_graphql_search_response{status=~"error",source="browser",request_name!="CodeIntelSearch"}[5m])) / ignoring(status) group_left sum(increase(src_graphql_search_response{source="browser",request_name!="CodeIntelSearch"}[5m])) * 100`,

							Warning:           monitoring.Alert().GreaterOrEqual(2, nil).For(15 * time.Minute),
							Critical:          monitoring.Alert().GreaterOrEqual(5, nil).For(15 * time.Minute),
							Panel:             monitoring.Panel().LegendFormat("{{status}}").Unit(monitoring.Percentage),
							Owner:             monitoring.ObservableOwnerSearch,
							PossibleSolutions: "none",
						},
						{
							Name:        "partial_timeout_search_responses",
							Description: "partial timeout search responses every 5m",
							Query:       `sum by (status)(increase(src_graphql_search_response{status="partial_timeout",source="browser",request_name!="CodeIntelSearch"}[5m])) / ignoring(status) group_left sum(increase(src_graphql_search_response{source="browser",request_name!="CodeIntelSearch"}[5m])) * 100`,

							Warning:           monitoring.Alert().GreaterOrEqual(5, nil).For(15 * time.Minute),
							Panel:             monitoring.Panel().LegendFormat("{{status}}").Unit(monitoring.Percentage),
							Owner:             monitoring.ObservableOwnerSearch,
							PossibleSolutions: "none",
						},
						{
							Name:        "search_alert_user_suggestions",
							Description: "search alert user suggestions shown every 5m",
							Query:       `sum by (alert_type)(increase(src_graphql_search_response{status="alert",alert_type!~"timed_out|no_results__suggest_quotes",source="browser",request_name!="CodeIntelSearch"}[5m])) / ignoring(alert_type) group_left sum(increase(src_graphql_search_response{source="browser",request_name!="CodeIntelSearch"}[5m])) * 100`,

							Warning: monitoring.Alert().GreaterOrEqual(5, nil).For(15 * time.Minute),
							Panel:   monitoring.Panel().LegendFormat("{{alert_type}}").Unit(monitoring.Percentage),
							Owner:   monitoring.ObservableOwnerSearch,
							PossibleSolutions: `
								- This indicates your user's are making syntax errors or similar user errors.
							`,
						},
					},
					{
						{
							Name:        "page_load_latency",
							Description: "90th percentile page load latency over all routes over 10m",
							Query:       `histogram_quantile(0.9, sum by(le) (rate(src_http_request_duration_seconds_bucket{route!="raw",route!="blob",route!~"graphql.*"}[10m])))`,

							Critical: monitoring.Alert().GreaterOrEqual(2, nil),
							Panel:    monitoring.Panel().LegendFormat("latency").Unit(monitoring.Seconds),
							Owner:    monitoring.ObservableOwnerCoreApplication,
							PossibleSolutions: `
								- Confirm that the Sourcegraph frontend has enough CPU/memory using the provisioning panels.
								- Trace a request to see what the slowest part is: https://docs.sourcegraph.com/admin/observability/tracing
							`,
						},
						{
							Name:        "blob_load_latency",
							Description: "90th percentile blob load latency over 10m",
							Query:       `histogram_quantile(0.9, sum by(le) (rate(src_http_request_duration_seconds_bucket{route="blob"}[10m])))`,
							Critical:    monitoring.Alert().GreaterOrEqual(5, nil),
							Panel:       monitoring.Panel().LegendFormat("latency").Unit(monitoring.Seconds),
							Owner:       monitoring.ObservableOwnerCoreApplication,
							PossibleSolutions: `
								- Confirm that the Sourcegraph frontend has enough CPU/memory using the provisioning panels.
								- Trace a request to see what the slowest part is: https://docs.sourcegraph.com/admin/observability/tracing
							`,
						},
					},
				},
			},
			{
				Title:  "Search-based code intelligence at a glance",
				Hidden: true,
				Rows: []monitoring.Row{
					{
						{
							Name:        "99th_percentile_search_codeintel_request_duration",
							Description: "99th percentile code-intel successful search request duration over 5m",
							Owner:       monitoring.ObservableOwnerCodeIntel,
							Query:       `histogram_quantile(0.99, sum by (le)(rate(src_graphql_field_seconds_bucket{type="Search",field="results",error="false",source="browser",request_name="CodeIntelSearch"}[5m])))`,

							Warning: monitoring.Alert().GreaterOrEqual(20, nil),
							Panel:   monitoring.Panel().LegendFormat("duration").Unit(monitoring.Seconds),
							PossibleSolutions: `
								- **Get details on the exact queries that are slow** by configuring '"observability.logSlowSearches": 20,' in the site configuration and looking for 'frontend' warning logs prefixed with 'slow search request' for additional details.
								- **Check that most repositories are indexed** by visiting https://sourcegraph.example.com/site-admin/repositories?filter=needs-index (it should show few or no results.)
								- **Kubernetes:** Check CPU usage of zoekt-webserver in the indexed-search pod, consider increasing CPU limits in the 'indexed-search.Deployment.yaml' if regularly hitting max CPU utilization.
								- **Docker Compose:** Check CPU usage on the Zoekt Web Server dashboard, consider increasing 'cpus:' of the zoekt-webserver container in 'docker-compose.yml' if regularly hitting max CPU utilization.
							`,
						},
						{
							Name:        "90th_percentile_search_codeintel_request_duration",
							Description: "90th percentile code-intel successful search request duration over 5m",
							Query:       `histogram_quantile(0.90, sum by (le)(rate(src_graphql_field_seconds_bucket{type="Search",field="results",error="false",source="browser",request_name="CodeIntelSearch"}[5m])))`,

							Warning: monitoring.Alert().GreaterOrEqual(15, nil),
							Panel:   monitoring.Panel().LegendFormat("duration").Unit(monitoring.Seconds),
							Owner:   monitoring.ObservableOwnerCodeIntel,
							PossibleSolutions: `
								- **Get details on the exact queries that are slow** by configuring '"observability.logSlowSearches": 15,' in the site configuration and looking for 'frontend' warning logs prefixed with 'slow search request' for additional details.
								- **Check that most repositories are indexed** by visiting https://sourcegraph.example.com/site-admin/repositories?filter=needs-index (it should show few or no results.)
								- **Kubernetes:** Check CPU usage of zoekt-webserver in the indexed-search pod, consider increasing CPU limits in the 'indexed-search.Deployment.yaml' if regularly hitting max CPU utilization.
								- **Docker Compose:** Check CPU usage on the Zoekt Web Server dashboard, consider increasing 'cpus:' of the zoekt-webserver container in 'docker-compose.yml' if regularly hitting max CPU utilization.
							`,
						},
					},
					{
						{
							Name:        "hard_timeout_search_codeintel_responses",
							Description: "hard timeout search code-intel responses every 5m",
							Query:       `(sum(increase(src_graphql_search_response{status="timeout",source="browser",request_name="CodeIntelSearch"}[5m])) + sum(increase(src_graphql_search_response{status="alert",alert_type="timed_out",source="browser",request_name="CodeIntelSearch"}[5m]))) / sum(increase(src_graphql_search_response{source="browser",request_name="CodeIntelSearch"}[5m])) * 100`,

							Warning:           monitoring.Alert().GreaterOrEqual(2, nil).For(15 * time.Minute),
							Critical:          monitoring.Alert().GreaterOrEqual(5, nil).For(15 * time.Minute),
							Panel:             monitoring.Panel().LegendFormat("hard timeout").Unit(monitoring.Percentage),
							Owner:             monitoring.ObservableOwnerCodeIntel,
							PossibleSolutions: "none",
						},
						{
							Name:        "hard_error_search_codeintel_responses",
							Description: "hard error search code-intel responses every 5m",
							Query:       `sum by (status)(increase(src_graphql_search_response{status=~"error",source="browser",request_name="CodeIntelSearch"}[5m])) / ignoring(status) group_left sum(increase(src_graphql_search_response{source="browser",request_name="CodeIntelSearch"}[5m])) * 100`,

							Warning:           monitoring.Alert().GreaterOrEqual(2, nil).For(15 * time.Minute),
							Critical:          monitoring.Alert().GreaterOrEqual(5, nil).For(15 * time.Minute),
							Panel:             monitoring.Panel().LegendFormat("hard error").Unit(monitoring.Percentage),
							Owner:             monitoring.ObservableOwnerCodeIntel,
							PossibleSolutions: "none",
						},
						{
							Name:        "partial_timeout_search_codeintel_responses",
							Description: "partial timeout search code-intel responses every 5m",
							Query:       `sum by (status)(increase(src_graphql_search_response{status="partial_timeout",source="browser",request_name="CodeIntelSearch"}[5m])) / ignoring(status) group_left sum(increase(src_graphql_search_response{status="partial_timeout",source="browser",request_name="CodeIntelSearch"}[5m])) * 100`,

							Warning:           monitoring.Alert().GreaterOrEqual(5, nil).For(15 * time.Minute),
							Panel:             monitoring.Panel().LegendFormat("partial timeout").Unit(monitoring.Percentage),
							Owner:             monitoring.ObservableOwnerCodeIntel,
							PossibleSolutions: "none",
						},
						{
							Name:        "search_codeintel_alert_user_suggestions",
							Description: "search code-intel alert user suggestions shown every 5m",
							Query:       `sum by (alert_type)(increase(src_graphql_search_response{status="alert",alert_type!~"timed_out",source="browser",request_name="CodeIntelSearch"}[5m])) / ignoring(alert_type) group_left sum(increase(src_graphql_search_response{source="browser",request_name="CodeIntelSearch"}[5m])) * 100`,

							Warning: monitoring.Alert().GreaterOrEqual(5, nil).For(15 * time.Minute),
							Panel:   monitoring.Panel().LegendFormat("{{alert_type}}").Unit(monitoring.Percentage),
							Owner:   monitoring.ObservableOwnerCodeIntel,
							PossibleSolutions: `
								- This indicates a bug in Sourcegraph, please [open an issue](https://github.com/sourcegraph/sourcegraph/issues/new/choose).
							`,
						},
					},
				},
			},
			{
				Title:  "Search API usage at a glance",
				Hidden: true,
				Rows: []monitoring.Row{
					{
						{
							Name:        "99th_percentile_search_api_request_duration",
							Description: "99th percentile successful search API request duration over 5m",
							Query:       `histogram_quantile(0.99, sum by (le)(rate(src_graphql_field_seconds_bucket{type="Search",field="results",error="false",source="other"}[5m])))`,

							Warning: monitoring.Alert().GreaterOrEqual(50, nil),
							Panel:   monitoring.Panel().LegendFormat("duration").Unit(monitoring.Seconds),
							Owner:   monitoring.ObservableOwnerSearch,
							PossibleSolutions: `
								- **Get details on the exact queries that are slow** by configuring '"observability.logSlowSearches": 20,' in the site configuration and looking for 'frontend' warning logs prefixed with 'slow search request' for additional details.
								- **Check that most repositories are indexed** by visiting https://sourcegraph.example.com/site-admin/repositories?filter=needs-index (it should show few or no results.)
								- **Kubernetes:** Check CPU usage of zoekt-webserver in the indexed-search pod, consider increasing CPU limits in the 'indexed-search.Deployment.yaml' if regularly hitting max CPU utilization.
								- **Docker Compose:** Check CPU usage on the Zoekt Web Server dashboard, consider increasing 'cpus:' of the zoekt-webserver container in 'docker-compose.yml' if regularly hitting max CPU utilization.
							`,
						},
						{
							Name:        "90th_percentile_search_api_request_duration",
							Description: "90th percentile successful search API request duration over 5m",
							Query:       `histogram_quantile(0.90, sum by (le)(rate(src_graphql_field_seconds_bucket{type="Search",field="results",error="false",source="other"}[5m])))`,

							Warning: monitoring.Alert().GreaterOrEqual(40, nil),
							Panel:   monitoring.Panel().LegendFormat("duration").Unit(monitoring.Seconds),
							Owner:   monitoring.ObservableOwnerSearch,
							PossibleSolutions: `
								- **Get details on the exact queries that are slow** by configuring '"observability.logSlowSearches": 15,' in the site configuration and looking for 'frontend' warning logs prefixed with 'slow search request' for additional details.
								- **Check that most repositories are indexed** by visiting https://sourcegraph.example.com/site-admin/repositories?filter=needs-index (it should show few or no results.)
								- **Kubernetes:** Check CPU usage of zoekt-webserver in the indexed-search pod, consider increasing CPU limits in the 'indexed-search.Deployment.yaml' if regularly hitting max CPU utilization.
								- **Docker Compose:** Check CPU usage on the Zoekt Web Server dashboard, consider increasing 'cpus:' of the zoekt-webserver container in 'docker-compose.yml' if regularly hitting max CPU utilization.
							`,
						},
					},
					{
						{
							Name:        "hard_timeout_search_api_responses",
							Description: "hard timeout search API responses every 5m",
							Query:       `(sum(increase(src_graphql_search_response{status="timeout",source="other"}[5m])) + sum(increase(src_graphql_search_response{status="alert",alert_type="timed_out",source="other"}[5m]))) / sum(increase(src_graphql_search_response{source="other"}[5m])) * 100`,

							Warning:           monitoring.Alert().GreaterOrEqual(2, nil).For(15 * time.Minute),
							Critical:          monitoring.Alert().GreaterOrEqual(5, nil).For(15 * time.Minute),
							Panel:             monitoring.Panel().LegendFormat("hard timeout").Unit(monitoring.Percentage),
							Owner:             monitoring.ObservableOwnerSearch,
							PossibleSolutions: "none",
						},
						{
							Name:        "hard_error_search_api_responses",
							Description: "hard error search API responses every 5m",
							Query:       `sum by (status)(increase(src_graphql_search_response{status=~"error",source="other"}[5m])) / ignoring(status) group_left sum(increase(src_graphql_search_response{source="other"}[5m]))`,

							Warning:           monitoring.Alert().GreaterOrEqual(2, nil).For(15 * time.Minute),
							Critical:          monitoring.Alert().GreaterOrEqual(5, nil).For(15 * time.Minute),
							Panel:             monitoring.Panel().LegendFormat("{{status}}").Unit(monitoring.Percentage),
							Owner:             monitoring.ObservableOwnerSearch,
							PossibleSolutions: "none",
						},
						{
							Name:        "partial_timeout_search_api_responses",
							Description: "partial timeout search API responses every 5m",
							Query:       `sum(increase(src_graphql_search_response{status="partial_timeout",source="other"}[5m])) / sum(increase(src_graphql_search_response{source="other"}[5m]))`,

							Warning:           monitoring.Alert().GreaterOrEqual(5, nil).For(15 * time.Minute),
							Panel:             monitoring.Panel().LegendFormat("partial timeout").Unit(monitoring.Percentage),
							Owner:             monitoring.ObservableOwnerSearch,
							PossibleSolutions: "none",
						},
						{
							Name:        "search_api_alert_user_suggestions",
							Description: "search API alert user suggestions shown every 5m",
							Query:       `sum by (alert_type)(increase(src_graphql_search_response{status="alert",alert_type!~"timed_out|no_results__suggest_quotes",source="other"}[5m])) / ignoring(alert_type) group_left sum(increase(src_graphql_search_response{status="alert",source="other"}[5m]))`,

							Warning: monitoring.Alert().GreaterOrEqual(5, nil),
							Panel:   monitoring.Panel().LegendFormat("{{alert_type}}").Unit(monitoring.Percentage),
							Owner:   monitoring.ObservableOwnerSearch,
							PossibleSolutions: `
								- This indicates your user's search API requests have syntax errors or a similar user error. Check the responses the API sends back for an explanation.
							`,
						},
					},
				},
			},

			// src_codeintel_resolvers_total
			// src_codeintel_resolvers_duration_seconds_bucket
			// src_codeintel_resolvers_errors_total
			shared.NewObservationGroup(containerName, monitoring.ObservableOwnerCodeIntel, shared.ObservationGroupOptions{
				ObservableOptions: shared.ObservableOptions{
					Namespace:         "codeintel",
					GroupDescription:  "Precise code intelligence usage at a glance",
					MetricName:        "codeintel_resolvers",
					MetricDescription: "graphql",
					Hidden:            true,
				},
			}),

			// src_codeintel_autoindex_enqueuer_total
			// src_codeintel_autoindex_enqueuer_duration_seconds_bucket
			// src_codeintel_autoindex_enqueuer_errors_total
			shared.NewObservationGroup(containerName, monitoring.ObservableOwnerCodeIntel, shared.ObservationGroupOptions{
				ObservableOptions: shared.ObservableOptions{
					Namespace:         "codeintel",
					GroupDescription:  "Auto-index enqueuer",
					MetricName:        "codeintel_autoindex_enqueuer",
					MetricDescription: "enqueuer",
					Hidden:            true,
				},
			}),

			// src_codeintel_dbstore_total
			// src_codeintel_dbstore_duration_seconds_bucket
			// src_codeintel_dbstore_errors_total
			shared.NewObservationGroup(containerName, monitoring.ObservableOwnerCodeIntel, shared.ObservationGroupOptions{
				ObservableOptions: shared.ObservableOptions{
					Namespace:         "codeintel",
					GroupDescription:  "dbstore stats (db=frontend)",
					MetricName:        "codeintel_dbstore",
					MetricDescription: "store",
					Hidden:            true,
				},
			}),

			// src_codeintel_lsifstore_total
			// src_codeintel_lsifstore_duration_seconds_bucket
			// src_codeintel_lsifstore_errors_total
			shared.NewObservationGroup(containerName, monitoring.ObservableOwnerCodeIntel, shared.ObservationGroupOptions{
				ObservableOptions: shared.ObservableOptions{
					Namespace:         "codeintel",
					GroupDescription:  "lsifstore stats (db=codeintel-db)",
					MetricName:        "codeintel_lsifstore",
					MetricDescription: "store",
					Hidden:            true,
				},
			}),

			// src_codeintel_gitserver_total
			// src_codeintel_gitserver_duration_seconds_bucket
			// src_codeintel_gitserver_errors_total
			shared.NewObservationGroup(containerName, monitoring.ObservableOwnerCodeIntel, shared.ObservationGroupOptions{
				ObservableOptions: shared.ObservableOptions{
					Namespace:         "codeintel",
					GroupDescription:  "gitserver client",
					MetricName:        "codeintel_gitserver",
					MetricDescription: "client",
					Hidden:            true,
				},
			}),

			// src_codeintel_uploadstore_total
			// src_codeintel_uploadstore_duration_seconds_bucket
			// src_codeintel_uploadstore_errors_total
			shared.NewObservationGroup(containerName, monitoring.ObservableOwnerCodeIntel, shared.ObservationGroupOptions{
				ObservableOptions: shared.ObservableOptions{
					Namespace:         "codeintel",
					GroupDescription:  "uploadstore stats (queries GCS/S3/MinIO)",
					MetricName:        "codeintel_uploadstore",
					MetricDescription: "store",
					Hidden:            true,
				},
			}),

			// src_oobmigration_total
			// src_oobmigration_duration_seconds_bucket
			// src_oobmigration_errors_total
			shared.NewObservationGroup(containerName, monitoring.ObservableOwnerCodeIntel, shared.ObservationGroupOptions{
				ObservableOptions: shared.ObservableOptions{
					Namespace:         "Out-of-band migrations",
					GroupDescription:  "up migration invocation (one batch processed)",
					MetricName:        "oobmigration",
					MetricDescription: "migration handler",
					Filters:           []string{`op="up"`},
					By:                []string{"migration"},
					Hidden:            true,
				},
			}),

			// src_oobmigration_total
			// src_oobmigration_duration_seconds_bucket
			// src_oobmigration_errors_total
			shared.NewObservationGroup(containerName, monitoring.ObservableOwnerCodeIntel, shared.ObservationGroupOptions{
				ObservableOptions: shared.ObservableOptions{
					Namespace:         "Out-of-band migrations",
					GroupDescription:  "down migration invocation (one batch processed)",
					MetricName:        "oobmigration",
					MetricDescription: "migration handler",
					Filters:           []string{`op="down"`},
					By:                []string{"migration"},
					Hidden:            true,
				},
			}),

			{
				Title:  "Internal service requests",
				Hidden: true,
				Rows: []monitoring.Row{
					{
						{
							Name:        "internal_indexed_search_error_responses",
							Description: "internal indexed search error responses every 5m",
							Query:       `sum by(code) (increase(src_zoekt_request_duration_seconds_count{code!~"2.."}[5m])) / ignoring(code) group_left sum(increase(src_zoekt_request_duration_seconds_count[5m])) * 100`,
							Warning:     monitoring.Alert().GreaterOrEqual(5, nil).For(15 * time.Minute),
							Panel:       monitoring.Panel().LegendFormat("{{code}}").Unit(monitoring.Percentage),
							Owner:       monitoring.ObservableOwnerSearch,
							PossibleSolutions: `
								- Check the Zoekt Web Server dashboard for indications it might be unhealthy.
							`,
						},
						{
							Name:        "internal_unindexed_search_error_responses",
							Description: "internal unindexed search error responses every 5m",
							Query:       `sum by(code) (increase(searcher_service_request_total{code!~"2.."}[5m])) / ignoring(code) group_left sum(increase(searcher_service_request_total[5m])) * 100`,
							Warning:     monitoring.Alert().GreaterOrEqual(5, nil).For(15 * time.Minute),
							Panel:       monitoring.Panel().LegendFormat("{{code}}").Unit(monitoring.Percentage),
							Owner:       monitoring.ObservableOwnerSearch,
							PossibleSolutions: `
								- Check the Searcher dashboard for indications it might be unhealthy.
							`,
						},
						{
							Name:        "internal_api_error_responses",
							Description: "internal API error responses every 5m by route",
							Query:       `sum by(category) (increase(src_frontend_internal_request_duration_seconds_count{code!~"2.."}[5m])) / ignoring(code) group_left sum(increase(src_frontend_internal_request_duration_seconds_count[5m])) * 100`,
							Warning:     monitoring.Alert().GreaterOrEqual(5, nil).For(15 * time.Minute),
							Panel:       monitoring.Panel().LegendFormat("{{category}}").Unit(monitoring.Percentage),
							Owner:       monitoring.ObservableOwnerCoreApplication,
							PossibleSolutions: `
								- May not be a substantial issue, check the 'frontend' logs for potential causes.
							`,
						},
					},
					{
						{
							Name:              "99th_percentile_gitserver_duration",
							Description:       "99th percentile successful gitserver query duration over 5m",
							Query:             `histogram_quantile(0.99, sum by (le,category)(rate(src_gitserver_request_duration_seconds_bucket{job=~"(sourcegraph-)?frontend"}[5m])))`,
							Warning:           monitoring.Alert().GreaterOrEqual(20, nil),
							Panel:             monitoring.Panel().LegendFormat("{{category}}").Unit(monitoring.Seconds),
							Owner:             monitoring.ObservableOwnerCoreApplication,
							PossibleSolutions: "none",
						},
						{
							Name:              "gitserver_error_responses",
							Description:       "gitserver error responses every 5m",
							Query:             `sum by (category)(increase(src_gitserver_request_duration_seconds_count{job=~"(sourcegraph-)?frontend",code!~"2.."}[5m])) / ignoring(code) group_left sum by (category)(increase(src_gitserver_request_duration_seconds_count{job=~"(sourcegraph-)?frontend"}[5m])) * 100`,
							Warning:           monitoring.Alert().GreaterOrEqual(5, nil).For(15 * time.Minute),
							Panel:             monitoring.Panel().LegendFormat("{{category}}").Unit(monitoring.Percentage),
							Owner:             monitoring.ObservableOwnerCoreApplication,
							PossibleSolutions: "none",
						},
					},
					{
						{
							Name:              "observability_test_alert_warning",
							Description:       "warning test alert metric",
							Query:             `max by(owner) (observability_test_metric_warning)`,
							Warning:           monitoring.Alert().GreaterOrEqual(1, nil),
							Panel:             monitoring.Panel().Max(1),
							Owner:             monitoring.ObservableOwnerDistribution,
							PossibleSolutions: "This alert is triggered via the `triggerObservabilityTestAlert` GraphQL endpoint, and will automatically resolve itself.",
						},
						{
							Name:              "observability_test_alert_critical",
							Description:       "critical test alert metric",
							Query:             `max by(owner) (observability_test_metric_critical)`,
							Critical:          monitoring.Alert().GreaterOrEqual(1, nil),
							Panel:             monitoring.Panel().Max(1),
							Owner:             monitoring.ObservableOwnerDistribution,
							PossibleSolutions: "This alert is triggered via the `triggerObservabilityTestAlert` GraphQL endpoint, and will automatically resolve itself.",
						},
					},
				},
			},

			// Resource monitoring
			shared.NewDatabaseConnectionsMonitoringGroup("frontend"),
			shared.NewContainerMonitoringGroup(containerName, monitoring.ObservableOwnerCoreApplication, nil),
			shared.NewProvisioningIndicatorsGroup(containerName, monitoring.ObservableOwnerCoreApplication, nil),
			shared.NewGolangMonitoringGroup(containerName, monitoring.ObservableOwnerCoreApplication, nil),
			shared.NewKubernetesMonitoringGroup(containerName, monitoring.ObservableOwnerCoreApplication, nil),

			{
				Title:  "Sentinel queries (only on sourcegraph.com)",
				Hidden: true,
				Rows: []monitoring.Row{
					{
						{
							Name:        "mean_successful_sentinel_duration_5m",
							Description: "mean successful sentinel search duration over 5m",
							// WARNING: if you change this, ensure that it will not trigger alerts on a customer instance
							// since these panels relate to metrics that don't exist on a customer instance.
							Query:    `sum(rate(src_search_response_latency_seconds_sum{source=~"searchblitz.*", status="success"}[5m])) / sum(rate(src_search_response_latency_seconds_count{source=~"searchblitz.*", status="success"}[5m]))`,
							Warning:  monitoring.Alert().GreaterOrEqual(5, nil).For(15 * time.Minute),
							Critical: monitoring.Alert().GreaterOrEqual(8, nil).For(30 * time.Minute),
							Panel:    monitoring.Panel().LegendFormat("duration").Unit(monitoring.Seconds).With(monitoring.PanelOptions.NoLegend()),
							Owner:    monitoring.ObservableOwnerSearch,
							PossibleSolutions: `
								- Look at the breakdown by query to determine if a specific query type is being affected
								- Check for high CPU usage on zoekt-webserver
								- Check Honeycomb for unusual activity
							`,
						},
						{
							Name:        "mean_sentinel_stream_latency_5m",
							Description: "mean sentinel stream latency over 5m",
							// WARNING: if you change this, ensure that it will not trigger alerts on a customer instance
							// since these panels relate to metrics that don't exist on a customer instance.
							Query:    `sum(rate(src_search_streaming_latency_seconds_sum{source=~"searchblitz.*"}[5m])) / sum(rate(src_search_streaming_latency_seconds_count{source=~"searchblitz.*"}[5m]))`,
							Warning:  monitoring.Alert().GreaterOrEqual(2, nil).For(15 * time.Minute),
							Critical: monitoring.Alert().GreaterOrEqual(3, nil).For(30 * time.Minute),
							Panel: monitoring.Panel().LegendFormat("latency").Unit(monitoring.Seconds).With(
								monitoring.PanelOptions.NoLegend(),
								monitoring.PanelOptions.ColorOverride("latency", "#8AB8FF"),
							),
							Owner: monitoring.ObservableOwnerSearch,
							PossibleSolutions: `
								- Look at the breakdown by query to determine if a specific query type is being affected
								- Check for high CPU usage on zoekt-webserver
								- Check Honeycomb for unusual activity
							`,
						},
					},
					{
						{
							Name:        "90th_percentile_successful_sentinel_duration_5m",
							Description: "90th percentile successful sentinel search duration over 5m",
							// WARNING: if you change this, ensure that it will not trigger alerts on a customer instance
							// since these panels relate to metrics that don't exist on a customer instance.
							Query:    `histogram_quantile(0.90, sum by (le)(label_replace(rate(src_search_response_latency_seconds_bucket{source=~"searchblitz.*", status="success"}[5m]), "source", "$1", "source", "searchblitz_(.*)")))`,
							Warning:  monitoring.Alert().GreaterOrEqual(5, nil).For(15 * time.Minute),
							Critical: monitoring.Alert().GreaterOrEqual(10, nil).For(30 * time.Minute),
							Panel:    monitoring.Panel().LegendFormat("duration").Unit(monitoring.Seconds).With(monitoring.PanelOptions.NoLegend()),
							Owner:    monitoring.ObservableOwnerSearch,
							PossibleSolutions: `
								- Look at the breakdown by query to determine if a specific query type is being affected
								- Check for high CPU usage on zoekt-webserver
								- Check Honeycomb for unusual activity
							`,
						},
						{
							Name:        "90th_percentile_sentinel_stream_latency_5m",
							Description: "90th percentile sentinel stream latency over 5m",
							// WARNING: if you change this, ensure that it will not trigger alerts on a customer instance
							// since these panels relate to metrics that don't exist on a customer instance.
							Query:    `histogram_quantile(0.90, sum by (le)(label_replace(rate(src_search_streaming_latency_seconds_bucket{source=~"searchblitz.*"}[5m]), "source", "$1", "source", "searchblitz_(.*)")))`,
							Warning:  monitoring.Alert().GreaterOrEqual(4, nil).For(15 * time.Minute),
							Critical: monitoring.Alert().GreaterOrEqual(6, nil).For(30 * time.Minute),
							Panel: monitoring.Panel().LegendFormat("latency").Unit(monitoring.Seconds).With(
								monitoring.PanelOptions.NoLegend(),
								monitoring.PanelOptions.ColorOverride("latency", "#8AB8FF"),
							),
							Owner: monitoring.ObservableOwnerSearch,
							PossibleSolutions: `
								- Look at the breakdown by query to determine if a specific query type is being affected
								- Check for high CPU usage on zoekt-webserver
								- Check Honeycomb for unusual activity
							`,
						},
					},
					{
						{
							Name:        "mean_successful_sentinel_duration_by_query_5m",
							Description: "mean successful sentinel search duration by query over 5m",
							Query:       `sum(rate(src_search_response_latency_seconds_sum{source=~"searchblitz.*", status="success"}[5m])) by (source) / sum(rate(src_search_response_latency_seconds_count{source=~"searchblitz.*", status="success"}[5m])) by (source)`,
							NoAlert:     true,
							Panel: monitoring.Panel().LegendFormat("{{query}}").Unit(monitoring.Seconds).With(
								monitoring.PanelOptions.LegendOnRight(),
								monitoring.PanelOptions.HoverShowAll(),
								monitoring.PanelOptions.HoverSort("descending"),
								monitoring.PanelOptions.Fill(0),
							),
							Owner: monitoring.ObservableOwnerSearch,
							Interpretation: `
								- The mean search duration for sentinel queries, broken down by query. Useful for debugging whether a slowdown is limited to a specific type of query.
							`,
						},
						{
							Name:        "mean_sentinel_stream_latency_by_query_5m",
							Description: "mean sentinel stream latency by query over 5m",
							Query:       `sum(rate(src_search_streaming_latency_seconds_sum{source=~"searchblitz.*"}[5m])) by (source) / sum(rate(src_search_streaming_latency_seconds_count{source=~"searchblitz.*"}[5m])) by (source)`,
							NoAlert:     true,
							Panel: monitoring.Panel().LegendFormat("{{query}}").Unit(monitoring.Seconds).With(
								monitoring.PanelOptions.LegendOnRight(),
								monitoring.PanelOptions.HoverShowAll(),
								monitoring.PanelOptions.HoverSort("descending"),
								monitoring.PanelOptions.Fill(0),
							),
							Owner: monitoring.ObservableOwnerSearch,
							Interpretation: `
								- The mean streaming search latency for sentinel queries, broken down by query. Useful for debugging whether a slowdown is limited to a specific type of query.
							`,
						},
					},
					{
						{
							Name:        "unsuccessful_status_rate_5m",
							Description: "unsuccessful status rate per 5m",
							Query:       `sum(rate(src_graphql_search_response{source=~"searchblitz.*", status!="success"}[5m])) by (status)`,
							NoAlert:     true,
							Panel:       monitoring.Panel().LegendFormat("{{status}}"),
							Owner:       monitoring.ObservableOwnerSearch,
							Interpretation: `
								- The rate of unsuccessful sentinel query, broken down by failure type
							`,
						},
					},
				},
			},
		},
	}
}
