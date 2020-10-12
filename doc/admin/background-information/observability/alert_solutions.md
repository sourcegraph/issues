# Alert solutions

This document contains possible solutions for when you find alerts are firing in Sourcegraph's monitoring.
If your alert isn't mentioned here, or if the solution doesn't help, [contact us](mailto:support@sourcegraph.com)
for assistance.

To learn more about Sourcegraph's alerting, see [our alerting documentation](https://docs.sourcegraph.com/admin/observability/alerting).

<!-- DO NOT EDIT: generated via: go generate ./monitoring -->

## frontend: 99th_percentile_search_request_duration

**Descriptions:**

- _frontend: 20s+ 99th percentile successful search request duration over 5m_

**Possible solutions:**

- **Get details on the exact queries that are slow** by configuring `"observability.logSlowSearches": 20,` in the site configuration and looking for `frontend` warning logs prefixed with `slow search request` for additional details.
- **Check that most repositories are indexed** by visiting https://sourcegraph.example.com/site-admin/repositories?filter=needs-index (it should show few or no results.)
- **Kubernetes:** Check CPU usage of zoekt-webserver in the indexed-search pod, consider increasing CPU limits in the `indexed-search.Deployment.yaml` if regularly hitting max CPU utilization.
- **Docker Compose:** Check CPU usage on the Zoekt Web Server dashboard, consider increasing `cpus:` of the zoekt-webserver container in `docker-compose.yml` if regularly hitting max CPU utilization.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_frontend_99th_percentile_search_request_duration"
]
```

## frontend: 90th_percentile_search_request_duration

**Descriptions:**

- _frontend: 15s+ 90th percentile successful search request duration over 5m_

**Possible solutions:**

- **Get details on the exact queries that are slow** by configuring `"observability.logSlowSearches": 15,` in the site configuration and looking for `frontend` warning logs prefixed with `slow search request` for additional details.
- **Check that most repositories are indexed** by visiting https://sourcegraph.example.com/site-admin/repositories?filter=needs-index (it should show few or no results.)
- **Kubernetes:** Check CPU usage of zoekt-webserver in the indexed-search pod, consider increasing CPU limits in the `indexed-search.Deployment.yaml` if regularly hitting max CPU utilization.
- **Docker Compose:** Check CPU usage on the Zoekt Web Server dashboard, consider increasing `cpus:` of the zoekt-webserver container in `docker-compose.yml` if regularly hitting max CPU utilization.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_frontend_90th_percentile_search_request_duration"
]
```

## frontend: hard_timeout_search_responses

**Descriptions:**

- _frontend: 2%+ hard timeout search responses every 5m for 15m0s_

- _frontend: 5%+ hard timeout search responses every 5m for 15m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_frontend_hard_timeout_search_responses",
  "critical_frontend_hard_timeout_search_responses"
]
```

## frontend: hard_error_search_responses

**Descriptions:**

- _frontend: 2%+ hard error search responses every 5m for 15m0s_

- _frontend: 5%+ hard error search responses every 5m for 15m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_frontend_hard_error_search_responses",
  "critical_frontend_hard_error_search_responses"
]
```

## frontend: partial_timeout_search_responses

**Descriptions:**

- _frontend: 5%+ partial timeout search responses every 5m for 15m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_frontend_partial_timeout_search_responses"
]
```

## frontend: search_alert_user_suggestions

**Descriptions:**

- _frontend: 5%+ search alert user suggestions shown every 5m for 15m0s_

**Possible solutions:**

- This indicates your user`s are making syntax errors or similar user errors.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_frontend_search_alert_user_suggestions"
]
```

## frontend: page_load_latency

**Descriptions:**

- _frontend: 2s+ 90th percentile page load latency over all routes over 10m_

**Possible solutions:**

- Confirm that the Sourcegraph frontend has enough CPU/memory using the provisioning panels.
- Trace a request to see what the slowest part is: https://docs.sourcegraph.com/admin/observability/tracing
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "critical_frontend_page_load_latency"
]
```

## frontend: blob_load_latency

**Descriptions:**

- _frontend: 5s+ 90th percentile blob load latency over 10m_

**Possible solutions:**

- Confirm that the Sourcegraph frontend has enough CPU/memory using the provisioning panels.
- Trace a request to see what the slowest part is: https://docs.sourcegraph.com/admin/observability/tracing
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "critical_frontend_blob_load_latency"
]
```

## frontend: 99th_percentile_search_codeintel_request_duration

**Descriptions:**

- _frontend: 20s+ 99th percentile code-intel successful search request duration over 5m_

**Possible solutions:**

- **Get details on the exact queries that are slow** by configuring `"observability.logSlowSearches": 20,` in the site configuration and looking for `frontend` warning logs prefixed with `slow search request` for additional details.
- **Check that most repositories are indexed** by visiting https://sourcegraph.example.com/site-admin/repositories?filter=needs-index (it should show few or no results.)
- **Kubernetes:** Check CPU usage of zoekt-webserver in the indexed-search pod, consider increasing CPU limits in the `indexed-search.Deployment.yaml` if regularly hitting max CPU utilization.
- **Docker Compose:** Check CPU usage on the Zoekt Web Server dashboard, consider increasing `cpus:` of the zoekt-webserver container in `docker-compose.yml` if regularly hitting max CPU utilization.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_frontend_99th_percentile_search_codeintel_request_duration"
]
```

## frontend: 90th_percentile_search_codeintel_request_duration

**Descriptions:**

- _frontend: 15s+ 90th percentile code-intel successful search request duration over 5m_

**Possible solutions:**

- **Get details on the exact queries that are slow** by configuring `"observability.logSlowSearches": 15,` in the site configuration and looking for `frontend` warning logs prefixed with `slow search request` for additional details.
- **Check that most repositories are indexed** by visiting https://sourcegraph.example.com/site-admin/repositories?filter=needs-index (it should show few or no results.)
- **Kubernetes:** Check CPU usage of zoekt-webserver in the indexed-search pod, consider increasing CPU limits in the `indexed-search.Deployment.yaml` if regularly hitting max CPU utilization.
- **Docker Compose:** Check CPU usage on the Zoekt Web Server dashboard, consider increasing `cpus:` of the zoekt-webserver container in `docker-compose.yml` if regularly hitting max CPU utilization.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_frontend_90th_percentile_search_codeintel_request_duration"
]
```

## frontend: hard_timeout_search_codeintel_responses

**Descriptions:**

- _frontend: 2%+ hard timeout search code-intel responses every 5m for 15m0s_

- _frontend: 5%+ hard timeout search code-intel responses every 5m for 15m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_frontend_hard_timeout_search_codeintel_responses",
  "critical_frontend_hard_timeout_search_codeintel_responses"
]
```

## frontend: hard_error_search_codeintel_responses

**Descriptions:**

- _frontend: 2%+ hard error search code-intel responses every 5m for 15m0s_

- _frontend: 5%+ hard error search code-intel responses every 5m for 15m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_frontend_hard_error_search_codeintel_responses",
  "critical_frontend_hard_error_search_codeintel_responses"
]
```

## frontend: partial_timeout_search_codeintel_responses

**Descriptions:**

- _frontend: 5%+ partial timeout search code-intel responses every 5m for 15m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_frontend_partial_timeout_search_codeintel_responses"
]
```

## frontend: search_codeintel_alert_user_suggestions

**Descriptions:**

- _frontend: 5%+ search code-intel alert user suggestions shown every 5m for 15m0s_

**Possible solutions:**

- This indicates a bug in Sourcegraph, please [open an issue](https://github.com/sourcegraph/sourcegraph/issues/new/choose).
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_frontend_search_codeintel_alert_user_suggestions"
]
```

## frontend: 99th_percentile_search_api_request_duration

**Descriptions:**

- _frontend: 50s+ 99th percentile successful search API request duration over 5m_

**Possible solutions:**

- **Get details on the exact queries that are slow** by configuring `"observability.logSlowSearches": 20,` in the site configuration and looking for `frontend` warning logs prefixed with `slow search request` for additional details.
- **If your users are requesting many results** with a large `count:` parameter, consider using our [search pagination API](../../api/graphql/search.md).
- **Check that most repositories are indexed** by visiting https://sourcegraph.example.com/site-admin/repositories?filter=needs-index (it should show few or no results.)
- **Kubernetes:** Check CPU usage of zoekt-webserver in the indexed-search pod, consider increasing CPU limits in the `indexed-search.Deployment.yaml` if regularly hitting max CPU utilization.
- **Docker Compose:** Check CPU usage on the Zoekt Web Server dashboard, consider increasing `cpus:` of the zoekt-webserver container in `docker-compose.yml` if regularly hitting max CPU utilization.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_frontend_99th_percentile_search_api_request_duration"
]
```

## frontend: 90th_percentile_search_api_request_duration

**Descriptions:**

- _frontend: 40s+ 90th percentile successful search API request duration over 5m_

**Possible solutions:**

- **Get details on the exact queries that are slow** by configuring `"observability.logSlowSearches": 15,` in the site configuration and looking for `frontend` warning logs prefixed with `slow search request` for additional details.
- **If your users are requesting many results** with a large `count:` parameter, consider using our [search pagination API](../../api/graphql/search.md).
- **Check that most repositories are indexed** by visiting https://sourcegraph.example.com/site-admin/repositories?filter=needs-index (it should show few or no results.)
- **Kubernetes:** Check CPU usage of zoekt-webserver in the indexed-search pod, consider increasing CPU limits in the `indexed-search.Deployment.yaml` if regularly hitting max CPU utilization.
- **Docker Compose:** Check CPU usage on the Zoekt Web Server dashboard, consider increasing `cpus:` of the zoekt-webserver container in `docker-compose.yml` if regularly hitting max CPU utilization.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_frontend_90th_percentile_search_api_request_duration"
]
```

## frontend: hard_timeout_search_api_responses

**Descriptions:**

- _frontend: 2%+ hard timeout search API responses every 5m for 15m0s_

- _frontend: 5%+ hard timeout search API responses every 5m for 15m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_frontend_hard_timeout_search_api_responses",
  "critical_frontend_hard_timeout_search_api_responses"
]
```

## frontend: hard_error_search_api_responses

**Descriptions:**

- _frontend: 2%+ hard error search API responses every 5m for 15m0s_

- _frontend: 5%+ hard error search API responses every 5m for 15m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_frontend_hard_error_search_api_responses",
  "critical_frontend_hard_error_search_api_responses"
]
```

## frontend: partial_timeout_search_api_responses

**Descriptions:**

- _frontend: 5%+ partial timeout search API responses every 5m for 15m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_frontend_partial_timeout_search_api_responses"
]
```

## frontend: search_api_alert_user_suggestions

**Descriptions:**

- _frontend: 5%+ search API alert user suggestions shown every 5m_

**Possible solutions:**

- This indicates your user`s search API requests have syntax errors or a similar user error. Check the responses the API sends back for an explanation.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_frontend_search_api_alert_user_suggestions"
]
```

## frontend: 99th_percentile_precise_code_intel_api_duration

**Descriptions:**

- _frontend: 20s+ 99th percentile successful precise code intel api query duration over 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_frontend_99th_percentile_precise_code_intel_api_duration"
]
```

## frontend: precise_code_intel_api_errors

**Descriptions:**

- _frontend: 5%+ precise code intel api errors every 5m for 15m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_frontend_precise_code_intel_api_errors"
]
```

## frontend: 99th_percentile_precise_code_intel_store_duration

**Descriptions:**

- _frontend: 20s+ 99th percentile successful precise code intel database query duration over 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_frontend_99th_percentile_precise_code_intel_store_duration"
]
```

## frontend: precise_code_intel_store_errors

**Descriptions:**

- _frontend: 5%+ precise code intel database errors every 5m for 15m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_frontend_precise_code_intel_store_errors"
]
```

## frontend: internal_indexed_search_error_responses

**Descriptions:**

- _frontend: 5%+ internal indexed search error responses every 5m for 15m0s_

**Possible solutions:**

- Check the Zoekt Web Server dashboard for indications it might be unhealthy.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_frontend_internal_indexed_search_error_responses"
]
```

## frontend: internal_unindexed_search_error_responses

**Descriptions:**

- _frontend: 5%+ internal unindexed search error responses every 5m for 15m0s_

**Possible solutions:**

- Check the Searcher dashboard for indications it might be unhealthy.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_frontend_internal_unindexed_search_error_responses"
]
```

## frontend: internal_api_error_responses

**Descriptions:**

- _frontend: 5%+ internal API error responses every 5m by route for 15m0s_

**Possible solutions:**

- May not be a substantial issue, check the `frontend` logs for potential causes.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_frontend_internal_api_error_responses"
]
```

## frontend: 99th_percentile_precise_code_intel_bundle_manager_query_duration

**Descriptions:**

- _frontend: 20s+ 99th percentile successful precise-code-intel-bundle-manager query duration over 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_frontend_99th_percentile_precise_code_intel_bundle_manager_query_duration"
]
```

## frontend: 99th_percentile_precise_code_intel_bundle_manager_transfer_duration

**Descriptions:**

- _frontend: 300s+ 99th percentile successful precise-code-intel-bundle-manager data transfer duration over 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_frontend_99th_percentile_precise_code_intel_bundle_manager_transfer_duration"
]
```

## frontend: precise_code_intel_bundle_manager_error_responses

**Descriptions:**

- _frontend: 5%+ precise-code-intel-bundle-manager error responses every 5m for 15m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_frontend_precise_code_intel_bundle_manager_error_responses"
]
```

## frontend: 99th_percentile_gitserver_duration

**Descriptions:**

- _frontend: 20s+ 99th percentile successful gitserver query duration over 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_frontend_99th_percentile_gitserver_duration"
]
```

## frontend: gitserver_error_responses

**Descriptions:**

- _frontend: 5%+ gitserver error responses every 5m for 15m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_frontend_gitserver_error_responses"
]
```

## frontend: observability_test_alert_warning

**Descriptions:**

- _frontend: 1+ warning test alert metric_

**Possible solutions:**

- This alert is triggered via the `triggerObservabilityTestAlert` GraphQL endpoint, and will automatically resolve itself.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_frontend_observability_test_alert_warning"
]
```

## frontend: observability_test_alert_critical

**Descriptions:**

- _frontend: 1+ critical test alert metric_

**Possible solutions:**

- This alert is triggered via the `triggerObservabilityTestAlert` GraphQL endpoint, and will automatically resolve itself.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "critical_frontend_observability_test_alert_critical"
]
```

## frontend: container_cpu_usage

**Descriptions:**

- _frontend: 99%+ container cpu usage total (1m average) across all cores by instance_

**Possible solutions:**

- **Kubernetes:** Consider increasing CPU limits in the the relevant `Deployment.yaml`.
- **Docker Compose:** Consider increasing `cpus:` of the frontend container in `docker-compose.yml`.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_frontend_container_cpu_usage"
]
```

## frontend: container_memory_usage

**Descriptions:**

- _frontend: 99%+ container memory usage by instance_

**Possible solutions:**

- **Kubernetes:** Consider increasing memory limit in relevant `Deployment.yaml`.
- **Docker Compose:** Consider increasing `memory:` of frontend container in `docker-compose.yml`.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_frontend_container_memory_usage"
]
```

## frontend: container_restarts

**Descriptions:**

- _frontend: 1+ container restarts every 5m by instance_

**Possible solutions:**

- **Kubernetes:**
	- Determine if the pod was OOM killed using `kubectl describe pod frontend` (look for `OOMKilled: true`) and, if so, consider increasing the memory limit in the relevant `Deployment.yaml`.
	- Check the logs before the container restarted to see if there are `panic:` messages or similar using `kubectl logs -p frontend`.
- **Docker Compose:**
	- Determine if the pod was OOM killed using `docker inspect -f '{{json .State}}' frontend` (look for `"OOMKilled":true`) and, if so, consider increasing the memory limit of the frontend container in `docker-compose.yml`.
	- Check the logs before the container restarted to see if there are `panic:` messages or similar using `docker logs frontend` (note this will include logs from the previous and currently running container).
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_frontend_container_restarts"
]
```

## frontend: fs_inodes_used

**Descriptions:**

- _frontend: 3e+06+ fs inodes in use by instance_

**Possible solutions:**

- 			- Refer to your OS or cloud provider`s documentation for how to increase inodes.
			- **Kubernetes:** consider provisioning more machines with less resources.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_frontend_fs_inodes_used"
]
```

## frontend: provisioning_container_cpu_usage_long_term

**Descriptions:**

- _frontend: 80%+ or less than 30% container cpu usage total (90th percentile over 1d) across all cores by instance for 336h0m0s_

**Possible solutions:**

- If usage is high:
	- **Kubernetes:** Consider increasing CPU limits in the `Deployment.yaml` for the frontend service.
	- **Docker Compose:** Consider increasing `cpus:` of the frontend container in `docker-compose.yml`.
- If usage is low, consider decreasing the above values.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_frontend_provisioning_container_cpu_usage_long_term"
]
```

## frontend: provisioning_container_memory_usage_long_term

**Descriptions:**

- _frontend: 80%+ or less than 30% container memory usage (1d maximum) by instance for 336h0m0s_

**Possible solutions:**

- If usage is high:
	- **Kubernetes:** Consider increasing memory limits in the `Deployment.yaml` for the frontend service.
	- **Docker Compose:** Consider increasing `memory:` of the frontend container in `docker-compose.yml`.
- If usage is low, consider decreasing the above values.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_frontend_provisioning_container_memory_usage_long_term"
]
```

## frontend: provisioning_container_cpu_usage_short_term

**Descriptions:**

- _frontend: 90%+ container cpu usage total (5m maximum) across all cores by instance for 30m0s_

**Possible solutions:**

- **Kubernetes:** Consider increasing CPU limits in the the relevant `Deployment.yaml`.
- **Docker Compose:** Consider increasing `cpus:` of the frontend container in `docker-compose.yml`.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_frontend_provisioning_container_cpu_usage_short_term"
]
```

## frontend: provisioning_container_memory_usage_short_term

**Descriptions:**

- _frontend: 90%+ container memory usage (5m maximum) by instance_

**Possible solutions:**

- **Kubernetes:** Consider increasing memory limit in relevant `Deployment.yaml`.
- **Docker Compose:** Consider increasing `memory:` of frontend container in `docker-compose.yml`.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_frontend_provisioning_container_memory_usage_short_term"
]
```

## frontend: go_goroutines

**Descriptions:**

- _frontend: 10000+ maximum active goroutines for 10m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_frontend_go_goroutines"
]
```

## frontend: go_gc_duration_seconds

**Descriptions:**

- _frontend: 2s+ maximum go garbage collection duration_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_frontend_go_gc_duration_seconds"
]
```

## frontend: pods_available_percentage

**Descriptions:**

- _frontend: less than 90% percentage pods available for 10m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "critical_frontend_pods_available_percentage"
]
```

## gitserver: disk_space_remaining

**Descriptions:**

- _gitserver: less than 25% disk space remaining by instance_

- _gitserver: less than 15% disk space remaining by instance_

**Possible solutions:**

- **Provision more disk space:** Sourcegraph will begin deleting least-used repository clones at 10% disk space remaining which may result in decreased performance, users having to wait for repositories to clone, etc.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_gitserver_disk_space_remaining",
  "critical_gitserver_disk_space_remaining"
]
```

## gitserver: running_git_commands

**Descriptions:**

- _gitserver: 50+ running git commands (signals load) for 2m0s_

- _gitserver: 100+ running git commands (signals load) for 5m0s_

**Possible solutions:**

- **Check if the problem may be an intermittent and temporary peak** using the "Container monitoring" section at the bottom of the Git Server dashboard.
- **Single container deployments:** Consider upgrading to a [Docker Compose deployment](../install/docker-compose/migrate.md) which offers better scalability and resource isolation.
- **Kubernetes and Docker Compose:** Check that you are running a similar number of git server replicas and that their CPU/memory limits are allocated according to what is shown in the [Sourcegraph resource estimator](../install/resource_estimator.md).
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_gitserver_running_git_commands",
  "critical_gitserver_running_git_commands"
]
```

## gitserver: repository_clone_queue_size

**Descriptions:**

- _gitserver: 25+ repository clone queue size_

**Possible solutions:**

- **If you just added several repositories**, the warning may be expected.
- **Check which repositories need cloning**, by visiting e.g. https://sourcegraph.example.com/site-admin/repositories?filter=not-cloned
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_gitserver_repository_clone_queue_size"
]
```

## gitserver: repository_existence_check_queue_size

**Descriptions:**

- _gitserver: 25+ repository existence check queue size_

**Possible solutions:**

- **Check the code host status indicator for errors:** on the Sourcegraph app homepage, when signed in as an admin click the cloud icon in the top right corner of the page.
- **Check if the issue continues to happen after 30 minutes**, it may be temporary.
- **Check the gitserver logs for more information.**
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_gitserver_repository_existence_check_queue_size"
]
```

## gitserver: echo_command_duration_test

**Descriptions:**

- _gitserver: 1s+ echo command duration test_

- _gitserver: 2s+ echo command duration test_

**Possible solutions:**

- **Check if the problem may be an intermittent and temporary peak** using the "Container monitoring" section at the bottom of the Git Server dashboard.
- **Single container deployments:** Consider upgrading to a [Docker Compose deployment](../install/docker-compose/migrate.md) which offers better scalability and resource isolation.
- **Kubernetes and Docker Compose:** Check that you are running a similar number of git server replicas and that their CPU/memory limits are allocated according to what is shown in the [Sourcegraph resource estimator](../install/resource_estimator.md).
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_gitserver_echo_command_duration_test",
  "critical_gitserver_echo_command_duration_test"
]
```

## gitserver: frontend_internal_api_error_responses

**Descriptions:**

- _gitserver: 2%+ frontend-internal API error responses every 5m by route for 5m0s_

**Possible solutions:**

- **Single-container deployments:** Check `docker logs $CONTAINER_ID` for logs starting with `repo-updater` that indicate requests to the frontend service are failing.
- **Kubernetes:**
	- Confirm that `kubectl get pods` shows the `frontend` pods are healthy.
	- Check `kubectl logs gitserver` for logs indicate request failures to `frontend` or `frontend-internal`.
- **Docker Compose:**
	- Confirm that `docker ps` shows the `frontend-internal` container is healthy.
	- Check `docker logs gitserver` for logs indicating request failures to `frontend` or `frontend-internal`.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_gitserver_frontend_internal_api_error_responses"
]
```

## gitserver: container_cpu_usage

**Descriptions:**

- _gitserver: 99%+ container cpu usage total (1m average) across all cores by instance_

**Possible solutions:**

- **Kubernetes:** Consider increasing CPU limits in the the relevant `Deployment.yaml`.
- **Docker Compose:** Consider increasing `cpus:` of the gitserver container in `docker-compose.yml`.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_gitserver_container_cpu_usage"
]
```

## gitserver: container_memory_usage

**Descriptions:**

- _gitserver: 99%+ container memory usage by instance_

**Possible solutions:**

- **Kubernetes:** Consider increasing memory limit in relevant `Deployment.yaml`.
- **Docker Compose:** Consider increasing `memory:` of gitserver container in `docker-compose.yml`.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_gitserver_container_memory_usage"
]
```

## gitserver: container_restarts

**Descriptions:**

- _gitserver: 1+ container restarts every 5m by instance_

**Possible solutions:**

- **Kubernetes:**
	- Determine if the pod was OOM killed using `kubectl describe pod gitserver` (look for `OOMKilled: true`) and, if so, consider increasing the memory limit in the relevant `Deployment.yaml`.
	- Check the logs before the container restarted to see if there are `panic:` messages or similar using `kubectl logs -p gitserver`.
- **Docker Compose:**
	- Determine if the pod was OOM killed using `docker inspect -f '{{json .State}}' gitserver` (look for `"OOMKilled":true`) and, if so, consider increasing the memory limit of the gitserver container in `docker-compose.yml`.
	- Check the logs before the container restarted to see if there are `panic:` messages or similar using `docker logs gitserver` (note this will include logs from the previous and currently running container).
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_gitserver_container_restarts"
]
```

## gitserver: fs_inodes_used

**Descriptions:**

- _gitserver: 3e+06+ fs inodes in use by instance_

**Possible solutions:**

- 			- Refer to your OS or cloud provider`s documentation for how to increase inodes.
			- **Kubernetes:** consider provisioning more machines with less resources.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_gitserver_fs_inodes_used"
]
```

## gitserver: fs_io_operations

**Descriptions:**

- _gitserver: 5000+ filesystem reads and writes rate by instance over 1h_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_gitserver_fs_io_operations"
]
```

## gitserver: provisioning_container_cpu_usage_long_term

**Descriptions:**

- _gitserver: 80%+ or less than 30% container cpu usage total (90th percentile over 1d) across all cores by instance for 336h0m0s_

**Possible solutions:**

- If usage is high:
	- **Kubernetes:** Consider increasing CPU limits in the `Deployment.yaml` for the gitserver service.
	- **Docker Compose:** Consider increasing `cpus:` of the gitserver container in `docker-compose.yml`.
- If usage is low, consider decreasing the above values.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_gitserver_provisioning_container_cpu_usage_long_term"
]
```

## gitserver: provisioning_container_memory_usage_long_term

**Descriptions:**

- _gitserver: less than 30% container memory usage (1d maximum) by instance for 336h0m0s_

**Possible solutions:**

- If usage is high:
	- **Kubernetes:** Consider increasing memory limits in the `Deployment.yaml` for the gitserver service.
	- **Docker Compose:** Consider increasing `memory:` of the gitserver container in `docker-compose.yml`.
- If usage is low, consider decreasing the above values.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_gitserver_provisioning_container_memory_usage_long_term"
]
```

## gitserver: provisioning_container_cpu_usage_short_term

**Descriptions:**

- _gitserver: 90%+ container cpu usage total (5m maximum) across all cores by instance for 30m0s_

**Possible solutions:**

- **Kubernetes:** Consider increasing CPU limits in the the relevant `Deployment.yaml`.
- **Docker Compose:** Consider increasing `cpus:` of the gitserver container in `docker-compose.yml`.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_gitserver_provisioning_container_cpu_usage_short_term"
]
```

## gitserver: go_goroutines

**Descriptions:**

- _gitserver: 10000+ maximum active goroutines for 10m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_gitserver_go_goroutines"
]
```

## gitserver: go_gc_duration_seconds

**Descriptions:**

- _gitserver: 2s+ maximum go garbage collection duration_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_gitserver_go_gc_duration_seconds"
]
```

## gitserver: pods_available_percentage

**Descriptions:**

- _gitserver: less than 90% percentage pods available for 10m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "critical_gitserver_pods_available_percentage"
]
```

## github-proxy: github_core_rate_limit_remaining

**Descriptions:**

- _github-proxy: less than 500 remaining calls to GitHub before hitting the rate limit for 5m0s_

**Possible solutions:**

- Try restarting the pod to get a different public IP.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "critical_github-proxy_github_core_rate_limit_remaining"
]
```

## github-proxy: github_search_rate_limit_remaining

**Descriptions:**

- _github-proxy: less than 5 remaining calls to GitHub search before hitting the rate limit_

**Possible solutions:**

- Try restarting the pod to get a different public IP.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_github-proxy_github_search_rate_limit_remaining"
]
```

## github-proxy: container_cpu_usage

**Descriptions:**

- _github-proxy: 99%+ container cpu usage total (1m average) across all cores by instance_

**Possible solutions:**

- **Kubernetes:** Consider increasing CPU limits in the the relevant `Deployment.yaml`.
- **Docker Compose:** Consider increasing `cpus:` of the github-proxy container in `docker-compose.yml`.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_github-proxy_container_cpu_usage"
]
```

## github-proxy: container_memory_usage

**Descriptions:**

- _github-proxy: 99%+ container memory usage by instance_

**Possible solutions:**

- **Kubernetes:** Consider increasing memory limit in relevant `Deployment.yaml`.
- **Docker Compose:** Consider increasing `memory:` of github-proxy container in `docker-compose.yml`.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_github-proxy_container_memory_usage"
]
```

## github-proxy: container_restarts

**Descriptions:**

- _github-proxy: 1+ container restarts every 5m by instance_

**Possible solutions:**

- **Kubernetes:**
	- Determine if the pod was OOM killed using `kubectl describe pod github-proxy` (look for `OOMKilled: true`) and, if so, consider increasing the memory limit in the relevant `Deployment.yaml`.
	- Check the logs before the container restarted to see if there are `panic:` messages or similar using `kubectl logs -p github-proxy`.
- **Docker Compose:**
	- Determine if the pod was OOM killed using `docker inspect -f '{{json .State}}' github-proxy` (look for `"OOMKilled":true`) and, if so, consider increasing the memory limit of the github-proxy container in `docker-compose.yml`.
	- Check the logs before the container restarted to see if there are `panic:` messages or similar using `docker logs github-proxy` (note this will include logs from the previous and currently running container).
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_github-proxy_container_restarts"
]
```

## github-proxy: fs_inodes_used

**Descriptions:**

- _github-proxy: 3e+06+ fs inodes in use by instance_

**Possible solutions:**

- 			- Refer to your OS or cloud provider`s documentation for how to increase inodes.
			- **Kubernetes:** consider provisioning more machines with less resources.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_github-proxy_fs_inodes_used"
]
```

## github-proxy: provisioning_container_cpu_usage_long_term

**Descriptions:**

- _github-proxy: 80%+ or less than 30% container cpu usage total (90th percentile over 1d) across all cores by instance for 336h0m0s_

**Possible solutions:**

- If usage is high:
	- **Kubernetes:** Consider increasing CPU limits in the `Deployment.yaml` for the github-proxy service.
	- **Docker Compose:** Consider increasing `cpus:` of the github-proxy container in `docker-compose.yml`.
- If usage is low, consider decreasing the above values.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_github-proxy_provisioning_container_cpu_usage_long_term"
]
```

## github-proxy: provisioning_container_memory_usage_long_term

**Descriptions:**

- _github-proxy: 80%+ or less than 30% container memory usage (1d maximum) by instance for 336h0m0s_

**Possible solutions:**

- If usage is high:
	- **Kubernetes:** Consider increasing memory limits in the `Deployment.yaml` for the github-proxy service.
	- **Docker Compose:** Consider increasing `memory:` of the github-proxy container in `docker-compose.yml`.
- If usage is low, consider decreasing the above values.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_github-proxy_provisioning_container_memory_usage_long_term"
]
```

## github-proxy: provisioning_container_cpu_usage_short_term

**Descriptions:**

- _github-proxy: 90%+ container cpu usage total (5m maximum) across all cores by instance for 30m0s_

**Possible solutions:**

- **Kubernetes:** Consider increasing CPU limits in the the relevant `Deployment.yaml`.
- **Docker Compose:** Consider increasing `cpus:` of the github-proxy container in `docker-compose.yml`.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_github-proxy_provisioning_container_cpu_usage_short_term"
]
```

## github-proxy: provisioning_container_memory_usage_short_term

**Descriptions:**

- _github-proxy: 90%+ container memory usage (5m maximum) by instance_

**Possible solutions:**

- **Kubernetes:** Consider increasing memory limit in relevant `Deployment.yaml`.
- **Docker Compose:** Consider increasing `memory:` of github-proxy container in `docker-compose.yml`.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_github-proxy_provisioning_container_memory_usage_short_term"
]
```

## github-proxy: go_goroutines

**Descriptions:**

- _github-proxy: 10000+ maximum active goroutines for 10m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_github-proxy_go_goroutines"
]
```

## github-proxy: go_gc_duration_seconds

**Descriptions:**

- _github-proxy: 2s+ maximum go garbage collection duration_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_github-proxy_go_gc_duration_seconds"
]
```

## github-proxy: pods_available_percentage

**Descriptions:**

- _github-proxy: less than 90% percentage pods available for 10m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "critical_github-proxy_pods_available_percentage"
]
```

## precise-code-intel-bundle-manager: 99th_percentile_bundle_database_duration

**Descriptions:**

- _precise-code-intel-bundle-manager: 20s+ 99th percentile successful bundle database query duration over 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-bundle-manager_99th_percentile_bundle_database_duration"
]
```

## precise-code-intel-bundle-manager: bundle_database_errors

**Descriptions:**

- _precise-code-intel-bundle-manager: 20+ bundle database errors every 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-bundle-manager_bundle_database_errors"
]
```

## precise-code-intel-bundle-manager: 99th_percentile_bundle_reader_duration

**Descriptions:**

- _precise-code-intel-bundle-manager: 20s+ 99th percentile successful bundle reader query duration over 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-bundle-manager_99th_percentile_bundle_reader_duration"
]
```

## precise-code-intel-bundle-manager: bundle_reader_errors

**Descriptions:**

- _precise-code-intel-bundle-manager: 20+ bundle reader errors every 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-bundle-manager_bundle_reader_errors"
]
```

## precise-code-intel-bundle-manager: disk_space_remaining

**Descriptions:**

- _precise-code-intel-bundle-manager: less than 25% disk space remaining by instance_

- _precise-code-intel-bundle-manager: less than 15% disk space remaining by instance_

**Possible solutions:**

- **Provision more disk space:** Sourcegraph will begin deleting the oldest uploaded bundle files at 10% disk space remaining.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-bundle-manager_disk_space_remaining",
  "critical_precise-code-intel-bundle-manager_disk_space_remaining"
]
```

## precise-code-intel-bundle-manager: janitor_errors

**Descriptions:**

- _precise-code-intel-bundle-manager: 20+ janitor errors every 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-bundle-manager_janitor_errors"
]
```

## precise-code-intel-bundle-manager: janitor_old_uploads_removed

**Descriptions:**

- _precise-code-intel-bundle-manager: 20+ upload files removed (due to age) every 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-bundle-manager_janitor_old_uploads_removed"
]
```

## precise-code-intel-bundle-manager: janitor_old_parts_removed

**Descriptions:**

- _precise-code-intel-bundle-manager: 20+ upload and database part files removed (due to age) every 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-bundle-manager_janitor_old_parts_removed"
]
```

## precise-code-intel-bundle-manager: janitor_old_dumps_removed

**Descriptions:**

- _precise-code-intel-bundle-manager: 20+ bundle files removed (due to low disk space) every 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-bundle-manager_janitor_old_dumps_removed"
]
```

## precise-code-intel-bundle-manager: janitor_orphans

**Descriptions:**

- _precise-code-intel-bundle-manager: 20+ bundle and upload files removed (with no corresponding database entry) every 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-bundle-manager_janitor_orphans"
]
```

## precise-code-intel-bundle-manager: janitor_uploads_removed

**Descriptions:**

- _precise-code-intel-bundle-manager: 20+ upload records removed every 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-bundle-manager_janitor_uploads_removed"
]
```

## precise-code-intel-bundle-manager: frontend_internal_api_error_responses

**Descriptions:**

- _precise-code-intel-bundle-manager: 2%+ frontend-internal API error responses every 5m by route for 5m0s_

**Possible solutions:**

- **Single-container deployments:** Check `docker logs $CONTAINER_ID` for logs starting with `repo-updater` that indicate requests to the frontend service are failing.
- **Kubernetes:**
	- Confirm that `kubectl get pods` shows the `frontend` pods are healthy.
	- Check `kubectl logs precise-code-intel-bundle-manager` for logs indicate request failures to `frontend` or `frontend-internal`.
- **Docker Compose:**
	- Confirm that `docker ps` shows the `frontend-internal` container is healthy.
	- Check `docker logs precise-code-intel-bundle-manager` for logs indicating request failures to `frontend` or `frontend-internal`.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-bundle-manager_frontend_internal_api_error_responses"
]
```

## precise-code-intel-bundle-manager: container_cpu_usage

**Descriptions:**

- _precise-code-intel-bundle-manager: 99%+ container cpu usage total (1m average) across all cores by instance_

**Possible solutions:**

- **Kubernetes:** Consider increasing CPU limits in the the relevant `Deployment.yaml`.
- **Docker Compose:** Consider increasing `cpus:` of the precise-code-intel-bundle-manager container in `docker-compose.yml`.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-bundle-manager_container_cpu_usage"
]
```

## precise-code-intel-bundle-manager: container_memory_usage

**Descriptions:**

- _precise-code-intel-bundle-manager: 99%+ container memory usage by instance_

**Possible solutions:**

- **Kubernetes:** Consider increasing memory limit in relevant `Deployment.yaml`.
- **Docker Compose:** Consider increasing `memory:` of precise-code-intel-bundle-manager container in `docker-compose.yml`.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-bundle-manager_container_memory_usage"
]
```

## precise-code-intel-bundle-manager: container_restarts

**Descriptions:**

- _precise-code-intel-bundle-manager: 1+ container restarts every 5m by instance_

**Possible solutions:**

- **Kubernetes:**
	- Determine if the pod was OOM killed using `kubectl describe pod precise-code-intel-bundle-manager` (look for `OOMKilled: true`) and, if so, consider increasing the memory limit in the relevant `Deployment.yaml`.
	- Check the logs before the container restarted to see if there are `panic:` messages or similar using `kubectl logs -p precise-code-intel-bundle-manager`.
- **Docker Compose:**
	- Determine if the pod was OOM killed using `docker inspect -f '{{json .State}}' precise-code-intel-bundle-manager` (look for `"OOMKilled":true`) and, if so, consider increasing the memory limit of the precise-code-intel-bundle-manager container in `docker-compose.yml`.
	- Check the logs before the container restarted to see if there are `panic:` messages or similar using `docker logs precise-code-intel-bundle-manager` (note this will include logs from the previous and currently running container).
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-bundle-manager_container_restarts"
]
```

## precise-code-intel-bundle-manager: fs_inodes_used

**Descriptions:**

- _precise-code-intel-bundle-manager: 3e+06+ fs inodes in use by instance_

**Possible solutions:**

- 			- Refer to your OS or cloud provider`s documentation for how to increase inodes.
			- **Kubernetes:** consider provisioning more machines with less resources.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-bundle-manager_fs_inodes_used"
]
```

## precise-code-intel-bundle-manager: provisioning_container_cpu_usage_long_term

**Descriptions:**

- _precise-code-intel-bundle-manager: 80%+ or less than 30% container cpu usage total (90th percentile over 1d) across all cores by instance for 336h0m0s_

**Possible solutions:**

- If usage is high:
	- **Kubernetes:** Consider increasing CPU limits in the `Deployment.yaml` for the precise-code-intel-bundle-manager service.
	- **Docker Compose:** Consider increasing `cpus:` of the precise-code-intel-bundle-manager container in `docker-compose.yml`.
- If usage is low, consider decreasing the above values.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-bundle-manager_provisioning_container_cpu_usage_long_term"
]
```

## precise-code-intel-bundle-manager: provisioning_container_memory_usage_long_term

**Descriptions:**

- _precise-code-intel-bundle-manager: 80%+ or less than 30% container memory usage (1d maximum) by instance for 336h0m0s_

**Possible solutions:**

- If usage is high:
	- **Kubernetes:** Consider increasing memory limits in the `Deployment.yaml` for the precise-code-intel-bundle-manager service.
	- **Docker Compose:** Consider increasing `memory:` of the precise-code-intel-bundle-manager container in `docker-compose.yml`.
- If usage is low, consider decreasing the above values.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-bundle-manager_provisioning_container_memory_usage_long_term"
]
```

## precise-code-intel-bundle-manager: provisioning_container_cpu_usage_short_term

**Descriptions:**

- _precise-code-intel-bundle-manager: 90%+ container cpu usage total (5m maximum) across all cores by instance for 30m0s_

**Possible solutions:**

- **Kubernetes:** Consider increasing CPU limits in the the relevant `Deployment.yaml`.
- **Docker Compose:** Consider increasing `cpus:` of the precise-code-intel-bundle-manager container in `docker-compose.yml`.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-bundle-manager_provisioning_container_cpu_usage_short_term"
]
```

## precise-code-intel-bundle-manager: provisioning_container_memory_usage_short_term

**Descriptions:**

- _precise-code-intel-bundle-manager: 90%+ container memory usage (5m maximum) by instance_

**Possible solutions:**

- **Kubernetes:** Consider increasing memory limit in relevant `Deployment.yaml`.
- **Docker Compose:** Consider increasing `memory:` of precise-code-intel-bundle-manager container in `docker-compose.yml`.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-bundle-manager_provisioning_container_memory_usage_short_term"
]
```

## precise-code-intel-bundle-manager: go_goroutines

**Descriptions:**

- _precise-code-intel-bundle-manager: 10000+ maximum active goroutines for 10m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-bundle-manager_go_goroutines"
]
```

## precise-code-intel-bundle-manager: go_gc_duration_seconds

**Descriptions:**

- _precise-code-intel-bundle-manager: 2s+ maximum go garbage collection duration_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-bundle-manager_go_gc_duration_seconds"
]
```

## precise-code-intel-bundle-manager: pods_available_percentage

**Descriptions:**

- _precise-code-intel-bundle-manager: less than 90% percentage pods available for 10m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "critical_precise-code-intel-bundle-manager_pods_available_percentage"
]
```

## precise-code-intel-worker: upload_queue_size

**Descriptions:**

- _precise-code-intel-worker: 100+ upload queue size_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-worker_upload_queue_size"
]
```

## precise-code-intel-worker: upload_queue_growth_rate

**Descriptions:**

- _precise-code-intel-worker: 5+ upload queue growth rate every 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-worker_upload_queue_growth_rate"
]
```

## precise-code-intel-worker: upload_process_errors

**Descriptions:**

- _precise-code-intel-worker: 20+ upload process errors every 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-worker_upload_process_errors"
]
```

## precise-code-intel-worker: 99th_percentile_store_duration

**Descriptions:**

- _precise-code-intel-worker: 20s+ 99th percentile successful database query duration over 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-worker_99th_percentile_store_duration"
]
```

## precise-code-intel-worker: store_errors

**Descriptions:**

- _precise-code-intel-worker: 20+ database errors every 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-worker_store_errors"
]
```

## precise-code-intel-worker: processing_uploads_reset

**Descriptions:**

- _precise-code-intel-worker: 20+ uploads reset to queued state every 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-worker_processing_uploads_reset"
]
```

## precise-code-intel-worker: processing_uploads_reset_failures

**Descriptions:**

- _precise-code-intel-worker: 20+ uploads errored after repeated resets every 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-worker_processing_uploads_reset_failures"
]
```

## precise-code-intel-worker: upload_resetter_errors

**Descriptions:**

- _precise-code-intel-worker: 20+ upload resetter errors every 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-worker_upload_resetter_errors"
]
```

## precise-code-intel-worker: 99th_percentile_bundle_manager_transfer_duration

**Descriptions:**

- _precise-code-intel-worker: 300s+ 99th percentile successful bundle manager data transfer duration over 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-worker_99th_percentile_bundle_manager_transfer_duration"
]
```

## precise-code-intel-worker: bundle_manager_error_responses

**Descriptions:**

- _precise-code-intel-worker: 5+ bundle manager error responses every 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-worker_bundle_manager_error_responses"
]
```

## precise-code-intel-worker: 99th_percentile_gitserver_duration

**Descriptions:**

- _precise-code-intel-worker: 20s+ 99th percentile successful gitserver query duration over 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-worker_99th_percentile_gitserver_duration"
]
```

## precise-code-intel-worker: gitserver_error_responses

**Descriptions:**

- _precise-code-intel-worker: 5%+ gitserver error responses every 5m for 15m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-worker_gitserver_error_responses"
]
```

## precise-code-intel-worker: frontend_internal_api_error_responses

**Descriptions:**

- _precise-code-intel-worker: 2%+ frontend-internal API error responses every 5m by route for 5m0s_

**Possible solutions:**

- **Single-container deployments:** Check `docker logs $CONTAINER_ID` for logs starting with `repo-updater` that indicate requests to the frontend service are failing.
- **Kubernetes:**
	- Confirm that `kubectl get pods` shows the `frontend` pods are healthy.
	- Check `kubectl logs precise-code-intel-worker` for logs indicate request failures to `frontend` or `frontend-internal`.
- **Docker Compose:**
	- Confirm that `docker ps` shows the `frontend-internal` container is healthy.
	- Check `docker logs precise-code-intel-worker` for logs indicating request failures to `frontend` or `frontend-internal`.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-worker_frontend_internal_api_error_responses"
]
```

## precise-code-intel-worker: container_cpu_usage

**Descriptions:**

- _precise-code-intel-worker: 99%+ container cpu usage total (1m average) across all cores by instance_

**Possible solutions:**

- **Kubernetes:** Consider increasing CPU limits in the the relevant `Deployment.yaml`.
- **Docker Compose:** Consider increasing `cpus:` of the precise-code-intel-worker container in `docker-compose.yml`.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-worker_container_cpu_usage"
]
```

## precise-code-intel-worker: container_memory_usage

**Descriptions:**

- _precise-code-intel-worker: 99%+ container memory usage by instance_

**Possible solutions:**

- **Kubernetes:** Consider increasing memory limit in relevant `Deployment.yaml`.
- **Docker Compose:** Consider increasing `memory:` of precise-code-intel-worker container in `docker-compose.yml`.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-worker_container_memory_usage"
]
```

## precise-code-intel-worker: container_restarts

**Descriptions:**

- _precise-code-intel-worker: 1+ container restarts every 5m by instance_

**Possible solutions:**

- **Kubernetes:**
	- Determine if the pod was OOM killed using `kubectl describe pod precise-code-intel-worker` (look for `OOMKilled: true`) and, if so, consider increasing the memory limit in the relevant `Deployment.yaml`.
	- Check the logs before the container restarted to see if there are `panic:` messages or similar using `kubectl logs -p precise-code-intel-worker`.
- **Docker Compose:**
	- Determine if the pod was OOM killed using `docker inspect -f '{{json .State}}' precise-code-intel-worker` (look for `"OOMKilled":true`) and, if so, consider increasing the memory limit of the precise-code-intel-worker container in `docker-compose.yml`.
	- Check the logs before the container restarted to see if there are `panic:` messages or similar using `docker logs precise-code-intel-worker` (note this will include logs from the previous and currently running container).
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-worker_container_restarts"
]
```

## precise-code-intel-worker: fs_inodes_used

**Descriptions:**

- _precise-code-intel-worker: 3e+06+ fs inodes in use by instance_

**Possible solutions:**

- 			- Refer to your OS or cloud provider`s documentation for how to increase inodes.
			- **Kubernetes:** consider provisioning more machines with less resources.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-worker_fs_inodes_used"
]
```

## precise-code-intel-worker: provisioning_container_cpu_usage_long_term

**Descriptions:**

- _precise-code-intel-worker: 80%+ or less than 30% container cpu usage total (90th percentile over 1d) across all cores by instance for 336h0m0s_

**Possible solutions:**

- If usage is high:
	- **Kubernetes:** Consider increasing CPU limits in the `Deployment.yaml` for the precise-code-intel-worker service.
	- **Docker Compose:** Consider increasing `cpus:` of the precise-code-intel-worker container in `docker-compose.yml`.
- If usage is low, consider decreasing the above values.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-worker_provisioning_container_cpu_usage_long_term"
]
```

## precise-code-intel-worker: provisioning_container_memory_usage_long_term

**Descriptions:**

- _precise-code-intel-worker: 80%+ or less than 30% container memory usage (1d maximum) by instance for 336h0m0s_

**Possible solutions:**

- If usage is high:
	- **Kubernetes:** Consider increasing memory limits in the `Deployment.yaml` for the precise-code-intel-worker service.
	- **Docker Compose:** Consider increasing `memory:` of the precise-code-intel-worker container in `docker-compose.yml`.
- If usage is low, consider decreasing the above values.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-worker_provisioning_container_memory_usage_long_term"
]
```

## precise-code-intel-worker: provisioning_container_cpu_usage_short_term

**Descriptions:**

- _precise-code-intel-worker: 90%+ container cpu usage total (5m maximum) across all cores by instance for 30m0s_

**Possible solutions:**

- **Kubernetes:** Consider increasing CPU limits in the the relevant `Deployment.yaml`.
- **Docker Compose:** Consider increasing `cpus:` of the precise-code-intel-worker container in `docker-compose.yml`.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-worker_provisioning_container_cpu_usage_short_term"
]
```

## precise-code-intel-worker: provisioning_container_memory_usage_short_term

**Descriptions:**

- _precise-code-intel-worker: 90%+ container memory usage (5m maximum) by instance_

**Possible solutions:**

- **Kubernetes:** Consider increasing memory limit in relevant `Deployment.yaml`.
- **Docker Compose:** Consider increasing `memory:` of precise-code-intel-worker container in `docker-compose.yml`.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-worker_provisioning_container_memory_usage_short_term"
]
```

## precise-code-intel-worker: go_goroutines

**Descriptions:**

- _precise-code-intel-worker: 10000+ maximum active goroutines for 10m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-worker_go_goroutines"
]
```

## precise-code-intel-worker: go_gc_duration_seconds

**Descriptions:**

- _precise-code-intel-worker: 2s+ maximum go garbage collection duration_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-worker_go_gc_duration_seconds"
]
```

## precise-code-intel-worker: pods_available_percentage

**Descriptions:**

- _precise-code-intel-worker: less than 90% percentage pods available for 10m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "critical_precise-code-intel-worker_pods_available_percentage"
]
```

## precise-code-intel-indexer: index_queue_size

**Descriptions:**

- _precise-code-intel-indexer: 100+ index queue size_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-indexer_index_queue_size"
]
```

## precise-code-intel-indexer: index_queue_growth_rate

**Descriptions:**

- _precise-code-intel-indexer: 5+ index queue growth rate every 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-indexer_index_queue_growth_rate"
]
```

## precise-code-intel-indexer: index_process_errors

**Descriptions:**

- _precise-code-intel-indexer: 20+ index process errors every 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-indexer_index_process_errors"
]
```

## precise-code-intel-indexer: 99th_percentile_store_duration

**Descriptions:**

- _precise-code-intel-indexer: 20s+ 99th percentile successful database query duration over 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-indexer_99th_percentile_store_duration"
]
```

## precise-code-intel-indexer: store_errors

**Descriptions:**

- _precise-code-intel-indexer: 20+ database errors every 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-indexer_store_errors"
]
```

## precise-code-intel-indexer: indexability_updater_errors

**Descriptions:**

- _precise-code-intel-indexer: 20+ indexability updater errors every 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-indexer_indexability_updater_errors"
]
```

## precise-code-intel-indexer: index_scheduler_errors

**Descriptions:**

- _precise-code-intel-indexer: 20+ index scheduler errors every 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-indexer_index_scheduler_errors"
]
```

## precise-code-intel-indexer: processing_indexes_reset

**Descriptions:**

- _precise-code-intel-indexer: 20+ indexes reset to queued state every 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-indexer_processing_indexes_reset"
]
```

## precise-code-intel-indexer: processing_indexes_reset_failures

**Descriptions:**

- _precise-code-intel-indexer: 20+ indexes errored after repeated resets every 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-indexer_processing_indexes_reset_failures"
]
```

## precise-code-intel-indexer: index_resetter_errors

**Descriptions:**

- _precise-code-intel-indexer: 20+ index resetter errors every 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-indexer_index_resetter_errors"
]
```

## precise-code-intel-indexer: janitor_errors

**Descriptions:**

- _precise-code-intel-indexer: 20+ janitor errors every 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-indexer_janitor_errors"
]
```

## precise-code-intel-indexer: janitor_indexes_removed

**Descriptions:**

- _precise-code-intel-indexer: 20+ index records removed every 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-indexer_janitor_indexes_removed"
]
```

## precise-code-intel-indexer: 99th_percentile_gitserver_duration

**Descriptions:**

- _precise-code-intel-indexer: 20s+ 99th percentile successful gitserver query duration over 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-indexer_99th_percentile_gitserver_duration"
]
```

## precise-code-intel-indexer: gitserver_error_responses

**Descriptions:**

- _precise-code-intel-indexer: 5%+ gitserver error responses every 5m for 15m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-indexer_gitserver_error_responses"
]
```

## precise-code-intel-indexer: frontend_internal_api_error_responses

**Descriptions:**

- _precise-code-intel-indexer: 2%+ frontend-internal API error responses every 5m by route for 5m0s_

**Possible solutions:**

- **Single-container deployments:** Check `docker logs $CONTAINER_ID` for logs starting with `repo-updater` that indicate requests to the frontend service are failing.
- **Kubernetes:**
	- Confirm that `kubectl get pods` shows the `frontend` pods are healthy.
	- Check `kubectl logs precise-code-intel-indexer` for logs indicate request failures to `frontend` or `frontend-internal`.
- **Docker Compose:**
	- Confirm that `docker ps` shows the `frontend-internal` container is healthy.
	- Check `docker logs precise-code-intel-indexer` for logs indicating request failures to `frontend` or `frontend-internal`.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-indexer_frontend_internal_api_error_responses"
]
```

## precise-code-intel-indexer: container_cpu_usage

**Descriptions:**

- _precise-code-intel-indexer: 99%+ container cpu usage total (1m average) across all cores by instance_

**Possible solutions:**

- **Kubernetes:** Consider increasing CPU limits in the the relevant `Deployment.yaml`.
- **Docker Compose:** Consider increasing `cpus:` of the precise-code-intel-indexer container in `docker-compose.yml`.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-indexer_container_cpu_usage"
]
```

## precise-code-intel-indexer: container_memory_usage

**Descriptions:**

- _precise-code-intel-indexer: 99%+ container memory usage by instance_

**Possible solutions:**

- **Kubernetes:** Consider increasing memory limit in relevant `Deployment.yaml`.
- **Docker Compose:** Consider increasing `memory:` of precise-code-intel-indexer container in `docker-compose.yml`.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-indexer_container_memory_usage"
]
```

## precise-code-intel-indexer: container_restarts

**Descriptions:**

- _precise-code-intel-indexer: 1+ container restarts every 5m by instance_

**Possible solutions:**

- **Kubernetes:**
	- Determine if the pod was OOM killed using `kubectl describe pod precise-code-intel-indexer` (look for `OOMKilled: true`) and, if so, consider increasing the memory limit in the relevant `Deployment.yaml`.
	- Check the logs before the container restarted to see if there are `panic:` messages or similar using `kubectl logs -p precise-code-intel-indexer`.
- **Docker Compose:**
	- Determine if the pod was OOM killed using `docker inspect -f '{{json .State}}' precise-code-intel-indexer` (look for `"OOMKilled":true`) and, if so, consider increasing the memory limit of the precise-code-intel-indexer container in `docker-compose.yml`.
	- Check the logs before the container restarted to see if there are `panic:` messages or similar using `docker logs precise-code-intel-indexer` (note this will include logs from the previous and currently running container).
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-indexer_container_restarts"
]
```

## precise-code-intel-indexer: fs_inodes_used

**Descriptions:**

- _precise-code-intel-indexer: 3e+06+ fs inodes in use by instance_

**Possible solutions:**

- 			- Refer to your OS or cloud provider`s documentation for how to increase inodes.
			- **Kubernetes:** consider provisioning more machines with less resources.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-indexer_fs_inodes_used"
]
```

## precise-code-intel-indexer: provisioning_container_cpu_usage_long_term

**Descriptions:**

- _precise-code-intel-indexer: 80%+ or less than 30% container cpu usage total (90th percentile over 1d) across all cores by instance for 336h0m0s_

**Possible solutions:**

- If usage is high:
	- **Kubernetes:** Consider increasing CPU limits in the `Deployment.yaml` for the precise-code-intel-indexer service.
	- **Docker Compose:** Consider increasing `cpus:` of the precise-code-intel-indexer container in `docker-compose.yml`.
- If usage is low, consider decreasing the above values.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-indexer_provisioning_container_cpu_usage_long_term"
]
```

## precise-code-intel-indexer: provisioning_container_memory_usage_long_term

**Descriptions:**

- _precise-code-intel-indexer: 80%+ or less than 30% container memory usage (1d maximum) by instance for 336h0m0s_

**Possible solutions:**

- If usage is high:
	- **Kubernetes:** Consider increasing memory limits in the `Deployment.yaml` for the precise-code-intel-indexer service.
	- **Docker Compose:** Consider increasing `memory:` of the precise-code-intel-indexer container in `docker-compose.yml`.
- If usage is low, consider decreasing the above values.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-indexer_provisioning_container_memory_usage_long_term"
]
```

## precise-code-intel-indexer: provisioning_container_cpu_usage_short_term

**Descriptions:**

- _precise-code-intel-indexer: 90%+ container cpu usage total (5m maximum) across all cores by instance for 30m0s_

**Possible solutions:**

- **Kubernetes:** Consider increasing CPU limits in the the relevant `Deployment.yaml`.
- **Docker Compose:** Consider increasing `cpus:` of the precise-code-intel-indexer container in `docker-compose.yml`.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-indexer_provisioning_container_cpu_usage_short_term"
]
```

## precise-code-intel-indexer: provisioning_container_memory_usage_short_term

**Descriptions:**

- _precise-code-intel-indexer: 90%+ container memory usage (5m maximum) by instance_

**Possible solutions:**

- **Kubernetes:** Consider increasing memory limit in relevant `Deployment.yaml`.
- **Docker Compose:** Consider increasing `memory:` of precise-code-intel-indexer container in `docker-compose.yml`.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-indexer_provisioning_container_memory_usage_short_term"
]
```

## precise-code-intel-indexer: go_goroutines

**Descriptions:**

- _precise-code-intel-indexer: 10000+ maximum active goroutines for 10m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-indexer_go_goroutines"
]
```

## precise-code-intel-indexer: go_gc_duration_seconds

**Descriptions:**

- _precise-code-intel-indexer: 2s+ maximum go garbage collection duration_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-indexer_go_gc_duration_seconds"
]
```

## precise-code-intel-indexer: pods_available_percentage

**Descriptions:**

- _precise-code-intel-indexer: less than 90% percentage pods available for 10m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "critical_precise-code-intel-indexer_pods_available_percentage"
]
```

## query-runner: frontend_internal_api_error_responses

**Descriptions:**

- _query-runner: 2%+ frontend-internal API error responses every 5m by route for 5m0s_

**Possible solutions:**

- **Single-container deployments:** Check `docker logs $CONTAINER_ID` for logs starting with `repo-updater` that indicate requests to the frontend service are failing.
- **Kubernetes:**
	- Confirm that `kubectl get pods` shows the `frontend` pods are healthy.
	- Check `kubectl logs query-runner` for logs indicate request failures to `frontend` or `frontend-internal`.
- **Docker Compose:**
	- Confirm that `docker ps` shows the `frontend-internal` container is healthy.
	- Check `docker logs query-runner` for logs indicating request failures to `frontend` or `frontend-internal`.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_query-runner_frontend_internal_api_error_responses"
]
```

## query-runner: container_memory_usage

**Descriptions:**

- _query-runner: 99%+ container memory usage by instance_

**Possible solutions:**

- **Kubernetes:** Consider increasing memory limit in relevant `Deployment.yaml`.
- **Docker Compose:** Consider increasing `memory:` of query-runner container in `docker-compose.yml`.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_query-runner_container_memory_usage"
]
```

## query-runner: container_cpu_usage

**Descriptions:**

- _query-runner: 99%+ container cpu usage total (1m average) across all cores by instance_

**Possible solutions:**

- **Kubernetes:** Consider increasing CPU limits in the the relevant `Deployment.yaml`.
- **Docker Compose:** Consider increasing `cpus:` of the query-runner container in `docker-compose.yml`.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_query-runner_container_cpu_usage"
]
```

## query-runner: container_restarts

**Descriptions:**

- _query-runner: 1+ container restarts every 5m by instance_

**Possible solutions:**

- **Kubernetes:**
	- Determine if the pod was OOM killed using `kubectl describe pod query-runner` (look for `OOMKilled: true`) and, if so, consider increasing the memory limit in the relevant `Deployment.yaml`.
	- Check the logs before the container restarted to see if there are `panic:` messages or similar using `kubectl logs -p query-runner`.
- **Docker Compose:**
	- Determine if the pod was OOM killed using `docker inspect -f '{{json .State}}' query-runner` (look for `"OOMKilled":true`) and, if so, consider increasing the memory limit of the query-runner container in `docker-compose.yml`.
	- Check the logs before the container restarted to see if there are `panic:` messages or similar using `docker logs query-runner` (note this will include logs from the previous and currently running container).
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_query-runner_container_restarts"
]
```

## query-runner: fs_inodes_used

**Descriptions:**

- _query-runner: 3e+06+ fs inodes in use by instance_

**Possible solutions:**

- 			- Refer to your OS or cloud provider`s documentation for how to increase inodes.
			- **Kubernetes:** consider provisioning more machines with less resources.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_query-runner_fs_inodes_used"
]
```

## query-runner: provisioning_container_cpu_usage_long_term

**Descriptions:**

- _query-runner: 80%+ or less than 30% container cpu usage total (90th percentile over 1d) across all cores by instance for 336h0m0s_

**Possible solutions:**

- If usage is high:
	- **Kubernetes:** Consider increasing CPU limits in the `Deployment.yaml` for the query-runner service.
	- **Docker Compose:** Consider increasing `cpus:` of the query-runner container in `docker-compose.yml`.
- If usage is low, consider decreasing the above values.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_query-runner_provisioning_container_cpu_usage_long_term"
]
```

## query-runner: provisioning_container_memory_usage_long_term

**Descriptions:**

- _query-runner: 80%+ or less than 30% container memory usage (1d maximum) by instance for 336h0m0s_

**Possible solutions:**

- If usage is high:
	- **Kubernetes:** Consider increasing memory limits in the `Deployment.yaml` for the query-runner service.
	- **Docker Compose:** Consider increasing `memory:` of the query-runner container in `docker-compose.yml`.
- If usage is low, consider decreasing the above values.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_query-runner_provisioning_container_memory_usage_long_term"
]
```

## query-runner: provisioning_container_cpu_usage_short_term

**Descriptions:**

- _query-runner: 90%+ container cpu usage total (5m maximum) across all cores by instance for 30m0s_

**Possible solutions:**

- **Kubernetes:** Consider increasing CPU limits in the the relevant `Deployment.yaml`.
- **Docker Compose:** Consider increasing `cpus:` of the query-runner container in `docker-compose.yml`.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_query-runner_provisioning_container_cpu_usage_short_term"
]
```

## query-runner: provisioning_container_memory_usage_short_term

**Descriptions:**

- _query-runner: 90%+ container memory usage (5m maximum) by instance_

**Possible solutions:**

- **Kubernetes:** Consider increasing memory limit in relevant `Deployment.yaml`.
- **Docker Compose:** Consider increasing `memory:` of query-runner container in `docker-compose.yml`.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_query-runner_provisioning_container_memory_usage_short_term"
]
```

## query-runner: go_goroutines

**Descriptions:**

- _query-runner: 10000+ maximum active goroutines for 10m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_query-runner_go_goroutines"
]
```

## query-runner: go_gc_duration_seconds

**Descriptions:**

- _query-runner: 2s+ maximum go garbage collection duration_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_query-runner_go_gc_duration_seconds"
]
```

## query-runner: pods_available_percentage

**Descriptions:**

- _query-runner: less than 90% percentage pods available for 10m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "critical_query-runner_pods_available_percentage"
]
```

## repo-updater: frontend_internal_api_error_responses

**Descriptions:**

- _repo-updater: 2%+ frontend-internal API error responses every 5m by route for 5m0s_

**Possible solutions:**

- **Single-container deployments:** Check `docker logs $CONTAINER_ID` for logs starting with `repo-updater` that indicate requests to the frontend service are failing.
- **Kubernetes:**
	- Confirm that `kubectl get pods` shows the `frontend` pods are healthy.
	- Check `kubectl logs repo-updater` for logs indicate request failures to `frontend` or `frontend-internal`.
- **Docker Compose:**
	- Confirm that `docker ps` shows the `frontend-internal` container is healthy.
	- Check `docker logs repo-updater` for logs indicating request failures to `frontend` or `frontend-internal`.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_repo-updater_frontend_internal_api_error_responses"
]
```

## repo-updater: syncer_sync_last_time

**Descriptions:**

- _repo-updater: 3600s+ time since last sync for 5m0s_

**Possible solutions:**

- Make sure there are external services added with valid tokens
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_repo-updater_syncer_sync_last_time"
]
```

## repo-updater: src_repoupdater_max_sync_backoff

**Descriptions:**

- _repo-updater: 28800s+ time since oldest sync for 10m0s_

**Possible solutions:**

- Make sure there are external services added with valid tokens
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "critical_repo-updater_src_repoupdater_max_sync_backoff"
]
```

## repo-updater: syncer_sync_start

**Descriptions:**

- _repo-updater: less than 1 sync was started for 8h0m0s_

**Possible solutions:**

- Check repo-updater logs for errors. Ignore this alert if only one code host connection is defined
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_repo-updater_syncer_sync_start"
]
```

## repo-updater: syncer_sync_duration

**Descriptions:**

- _repo-updater: 30s+ 95th repositories sync duration for 5m0s_

**Possible solutions:**

- Check the network latency is reasonable (<50ms) between the Sourcegraph and the code host
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_repo-updater_syncer_sync_duration"
]
```

## repo-updater: source_duration

**Descriptions:**

- _repo-updater: 30s+ 95th repositories source duration for 5m0s_

**Possible solutions:**

- Check the network latency is reasonable (<50ms) between the Sourcegraph and the code host
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_repo-updater_source_duration"
]
```

## repo-updater: syncer_synced_repos

**Descriptions:**

- _repo-updater: less than 1 repositories synced for 8h0m0s_

**Possible solutions:**

- Check network connectivity to code hosts
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_repo-updater_syncer_synced_repos"
]
```

## repo-updater: sourced_repos

**Descriptions:**

- _repo-updater: less than 1 repositories sourced for 8h0m0s_

**Possible solutions:**

- Check network connectivity to code hosts
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_repo-updater_sourced_repos"
]
```

## repo-updater: user_added_repos

**Descriptions:**

- _repo-updater: 180000+ total number of user added repos for 5m0s_

**Possible solutions:**

- Check for unusual spikes in user added repos. Each user is only allowed to add 2000
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "critical_repo-updater_user_added_repos"
]
```

## repo-updater: purge_failed

**Descriptions:**

- _repo-updater: 1+ repositories purge failed for 5m0s_

**Possible solutions:**

- Check repo-updater`s connectivity with gitserver and gitserver logs
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_repo-updater_purge_failed"
]
```

## repo-updater: sched_auto_fetch

**Descriptions:**

- _repo-updater: less than 1 repositories scheduled due to hitting a deadline for 8h0m0s_

**Possible solutions:**

- Check repo-updater logs. This is expected to fire if there are no user added code hosts
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_repo-updater_sched_auto_fetch"
]
```

## repo-updater: sched_manual_fetch

**Descriptions:**

- _repo-updater: less than 1 repositories scheduled due to user traffic for 8h0m0s_

**Possible solutions:**

- Check repo-updater logs. This is expected to fire if there are no user added code hosts
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_repo-updater_sched_manual_fetch"
]
```

## repo-updater: sched_known_repos

**Descriptions:**

- _repo-updater: less than 1 repositories managed by the scheduler for 10m0s_

**Possible solutions:**

- Check repo-updater logs. This is expected to fire if there are no user added code hosts
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_repo-updater_sched_known_repos"
]
```

## repo-updater: sched_update_queue_length

**Descriptions:**

- _repo-updater: 1000+ repositories queued for update for 5m0s_

**Possible solutions:**

- Check repo-updater logs. The queue should drop as items are sent to GitServer
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "critical_repo-updater_sched_update_queue_length"
]
```

## repo-updater: sched_loops

**Descriptions:**

- _repo-updater: less than 1 scheduler loops for 8h0m0s_

**Possible solutions:**

- Check repo-updater logs for errors. This is expected to fire if there are no user added code hosts
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_repo-updater_sched_loops"
]
```

## repo-updater: sched_error

**Descriptions:**

- _repo-updater: 1+ repositories schedule error rate for 1m0s_

**Possible solutions:**

- Check repo-updater logs for errors
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "critical_repo-updater_sched_error"
]
```

## repo-updater: perms_syncer_perms

**Descriptions:**

- _repo-updater: 259200s+ time gap between least and most up to date permissions for 5m0s_

**Possible solutions:**

- Increase the API rate limit to [GitHub](https://docs.sourcegraph.com/admin/external_service/github#github-com-rate-limits), [GitLab](https://docs.sourcegraph.com/admin/external_service/gitlab#internal-rate-limits) or [Bitbucket Server](https://docs.sourcegraph.com/admin/external_service/bitbucket_server#internal-rate-limits).
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_repo-updater_perms_syncer_perms"
]
```

## repo-updater: perms_syncer_stale_perms

**Descriptions:**

- _repo-updater: 100+ number of entities with stale permissions for 5m0s_

**Possible solutions:**

- Increase the API rate limit to [GitHub](https://docs.sourcegraph.com/admin/external_service/github#github-com-rate-limits), [GitLab](https://docs.sourcegraph.com/admin/external_service/gitlab#internal-rate-limits) or [Bitbucket Server](https://docs.sourcegraph.com/admin/external_service/bitbucket_server#internal-rate-limits).
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_repo-updater_perms_syncer_stale_perms"
]
```

## repo-updater: perms_syncer_no_perms

**Descriptions:**

- _repo-updater: 100+ number of entities with no permissions for 5m0s_

**Possible solutions:**

- **Enabled permissions for the first time:** Wait for few minutes and see if the number goes down.
- **Otherwise:** Increase the API rate limit to [GitHub](https://docs.sourcegraph.com/admin/external_service/github#github-com-rate-limits), [GitLab](https://docs.sourcegraph.com/admin/external_service/gitlab#internal-rate-limits) or [Bitbucket Server](https://docs.sourcegraph.com/admin/external_service/bitbucket_server#internal-rate-limits).
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_repo-updater_perms_syncer_no_perms"
]
```

## repo-updater: perms_syncer_sync_duration

**Descriptions:**

- _repo-updater: 30s+ 95th permissions sync duration for 5m0s_

**Possible solutions:**

- Check the network latency is reasonable (<50ms) between the Sourcegraph and the code host.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_repo-updater_perms_syncer_sync_duration"
]
```

## repo-updater: perms_syncer_queue_size

**Descriptions:**

- _repo-updater: 100+ permissions sync queued items for 5m0s_

**Possible solutions:**

- **Enabled permissions for the first time:** Wait for few minutes and see if the number goes down.
- **Otherwise:** Increase the API rate limit to [GitHub](https://docs.sourcegraph.com/admin/external_service/github#github-com-rate-limits), [GitLab](https://docs.sourcegraph.com/admin/external_service/gitlab#internal-rate-limits) or [Bitbucket Server](https://docs.sourcegraph.com/admin/external_service/bitbucket_server#internal-rate-limits).
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_repo-updater_perms_syncer_queue_size"
]
```

## repo-updater: authz_filter_duration

**Descriptions:**

- _repo-updater: 1s+ 95th authorization duration for 1m0s_

**Possible solutions:**

- Check if database is overloaded.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "critical_repo-updater_authz_filter_duration"
]
```

## repo-updater: perms_syncer_sync_errors

**Descriptions:**

- _repo-updater: 1+ permissions sync error rate for 1m0s_

**Possible solutions:**

- Check the network connectivity the Sourcegraph and the code host.
- Check if API rate limit quota is exhausted on the code host.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "critical_repo-updater_perms_syncer_sync_errors"
]
```

## repo-updater: src_repoupdater_external_services_total

**Descriptions:**

- _repo-updater: 20000+ the total number of external services for 1h0m0s_

**Possible solutions:**

- Check for spikes in external services, could be abuse
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "critical_repo-updater_src_repoupdater_external_services_total"
]
```

## repo-updater: src_repoupdater_user_external_services_total

**Descriptions:**

- _repo-updater: 20000+ the total number of user added external services for 1h0m0s_

**Possible solutions:**

- Check for spikes in external services, could be abuse
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_repo-updater_src_repoupdater_user_external_services_total"
]
```

## repo-updater: repoupdater_queued_sync_jobs_total

**Descriptions:**

- _repo-updater: 100+ the total number of queued sync jobs for 1h0m0s_

**Possible solutions:**

- **Check if jobs are failing to sync:** "SELECT * FROM external_service_sync_jobs WHERE state = `errored`";
- **Increase the number of workers** using the `repoConcurrentExternalServiceSyncers` site config.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_repo-updater_repoupdater_queued_sync_jobs_total"
]
```

## repo-updater: repoupdater_completed_sync_jobs_total

**Descriptions:**

- _repo-updater: 100000+ the total number of completed sync jobs for 1h0m0s_

**Possible solutions:**

- Check repo-updater logs. Jobs older than 1 day should have been removed.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_repo-updater_repoupdater_completed_sync_jobs_total"
]
```

## repo-updater: repoupdater_errored_sync_jobs_total

**Descriptions:**

- _repo-updater: 100+ the total number of errored sync jobs for 1h0m0s_

**Possible solutions:**

- Check repo-updater logs. Check code host connectivity
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_repo-updater_repoupdater_errored_sync_jobs_total"
]
```

## repo-updater: container_cpu_usage

**Descriptions:**

- _repo-updater: 99%+ container cpu usage total (1m average) across all cores by instance_

**Possible solutions:**

- **Kubernetes:** Consider increasing CPU limits in the the relevant `Deployment.yaml`.
- **Docker Compose:** Consider increasing `cpus:` of the repo-updater container in `docker-compose.yml`.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_repo-updater_container_cpu_usage"
]
```

## repo-updater: container_memory_usage

**Descriptions:**

- _repo-updater: 99%+ container memory usage by instance_

**Possible solutions:**

- **Kubernetes:** Consider increasing memory limit in relevant `Deployment.yaml`.
- **Docker Compose:** Consider increasing `memory:` of repo-updater container in `docker-compose.yml`.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_repo-updater_container_memory_usage"
]
```

## repo-updater: container_restarts

**Descriptions:**

- _repo-updater: 1+ container restarts every 5m by instance_

**Possible solutions:**

- **Kubernetes:**
	- Determine if the pod was OOM killed using `kubectl describe pod repo-updater` (look for `OOMKilled: true`) and, if so, consider increasing the memory limit in the relevant `Deployment.yaml`.
	- Check the logs before the container restarted to see if there are `panic:` messages or similar using `kubectl logs -p repo-updater`.
- **Docker Compose:**
	- Determine if the pod was OOM killed using `docker inspect -f '{{json .State}}' repo-updater` (look for `"OOMKilled":true`) and, if so, consider increasing the memory limit of the repo-updater container in `docker-compose.yml`.
	- Check the logs before the container restarted to see if there are `panic:` messages or similar using `docker logs repo-updater` (note this will include logs from the previous and currently running container).
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_repo-updater_container_restarts"
]
```

## repo-updater: fs_inodes_used

**Descriptions:**

- _repo-updater: 3e+06+ fs inodes in use by instance_

**Possible solutions:**

- 			- Refer to your OS or cloud provider`s documentation for how to increase inodes.
			- **Kubernetes:** consider provisioning more machines with less resources.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_repo-updater_fs_inodes_used"
]
```

## repo-updater: provisioning_container_cpu_usage_long_term

**Descriptions:**

- _repo-updater: 80%+ or less than 30% container cpu usage total (90th percentile over 1d) across all cores by instance for 336h0m0s_

**Possible solutions:**

- If usage is high:
	- **Kubernetes:** Consider increasing CPU limits in the `Deployment.yaml` for the repo-updater service.
	- **Docker Compose:** Consider increasing `cpus:` of the repo-updater container in `docker-compose.yml`.
- If usage is low, consider decreasing the above values.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_repo-updater_provisioning_container_cpu_usage_long_term"
]
```

## repo-updater: provisioning_container_memory_usage_long_term

**Descriptions:**

- _repo-updater: 80%+ or less than 30% container memory usage (1d maximum) by instance for 336h0m0s_

**Possible solutions:**

- If usage is high:
	- **Kubernetes:** Consider increasing memory limits in the `Deployment.yaml` for the repo-updater service.
	- **Docker Compose:** Consider increasing `memory:` of the repo-updater container in `docker-compose.yml`.
- If usage is low, consider decreasing the above values.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_repo-updater_provisioning_container_memory_usage_long_term"
]
```

## repo-updater: provisioning_container_cpu_usage_short_term

**Descriptions:**

- _repo-updater: 90%+ container cpu usage total (5m maximum) across all cores by instance for 30m0s_

**Possible solutions:**

- **Kubernetes:** Consider increasing CPU limits in the the relevant `Deployment.yaml`.
- **Docker Compose:** Consider increasing `cpus:` of the repo-updater container in `docker-compose.yml`.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_repo-updater_provisioning_container_cpu_usage_short_term"
]
```

## repo-updater: provisioning_container_memory_usage_short_term

**Descriptions:**

- _repo-updater: 90%+ container memory usage (5m maximum) by instance_

**Possible solutions:**

- **Kubernetes:** Consider increasing memory limit in relevant `Deployment.yaml`.
- **Docker Compose:** Consider increasing `memory:` of repo-updater container in `docker-compose.yml`.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_repo-updater_provisioning_container_memory_usage_short_term"
]
```

## repo-updater: go_goroutines

**Descriptions:**

- _repo-updater: 10000+ maximum active goroutines for 10m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_repo-updater_go_goroutines"
]
```

## repo-updater: go_gc_duration_seconds

**Descriptions:**

- _repo-updater: 2s+ maximum go garbage collection duration_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_repo-updater_go_gc_duration_seconds"
]
```

## repo-updater: pods_available_percentage

**Descriptions:**

- _repo-updater: less than 90% percentage pods available for 10m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "critical_repo-updater_pods_available_percentage"
]
```

## searcher: unindexed_search_request_errors

**Descriptions:**

- _searcher: 5%+ unindexed search request errors every 5m by code for 5m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_searcher_unindexed_search_request_errors"
]
```

## searcher: replica_traffic

**Descriptions:**

- _searcher: 5+ requests per second over 10m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_searcher_replica_traffic"
]
```

## searcher: frontend_internal_api_error_responses

**Descriptions:**

- _searcher: 2%+ frontend-internal API error responses every 5m by route for 5m0s_

**Possible solutions:**

- **Single-container deployments:** Check `docker logs $CONTAINER_ID` for logs starting with `repo-updater` that indicate requests to the frontend service are failing.
- **Kubernetes:**
	- Confirm that `kubectl get pods` shows the `frontend` pods are healthy.
	- Check `kubectl logs searcher` for logs indicate request failures to `frontend` or `frontend-internal`.
- **Docker Compose:**
	- Confirm that `docker ps` shows the `frontend-internal` container is healthy.
	- Check `docker logs searcher` for logs indicating request failures to `frontend` or `frontend-internal`.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_searcher_frontend_internal_api_error_responses"
]
```

## searcher: container_cpu_usage

**Descriptions:**

- _searcher: 99%+ container cpu usage total (1m average) across all cores by instance_

**Possible solutions:**

- **Kubernetes:** Consider increasing CPU limits in the the relevant `Deployment.yaml`.
- **Docker Compose:** Consider increasing `cpus:` of the searcher container in `docker-compose.yml`.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_searcher_container_cpu_usage"
]
```

## searcher: container_memory_usage

**Descriptions:**

- _searcher: 99%+ container memory usage by instance_

**Possible solutions:**

- **Kubernetes:** Consider increasing memory limit in relevant `Deployment.yaml`.
- **Docker Compose:** Consider increasing `memory:` of searcher container in `docker-compose.yml`.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_searcher_container_memory_usage"
]
```

## searcher: container_restarts

**Descriptions:**

- _searcher: 1+ container restarts every 5m by instance_

**Possible solutions:**

- **Kubernetes:**
	- Determine if the pod was OOM killed using `kubectl describe pod searcher` (look for `OOMKilled: true`) and, if so, consider increasing the memory limit in the relevant `Deployment.yaml`.
	- Check the logs before the container restarted to see if there are `panic:` messages or similar using `kubectl logs -p searcher`.
- **Docker Compose:**
	- Determine if the pod was OOM killed using `docker inspect -f '{{json .State}}' searcher` (look for `"OOMKilled":true`) and, if so, consider increasing the memory limit of the searcher container in `docker-compose.yml`.
	- Check the logs before the container restarted to see if there are `panic:` messages or similar using `docker logs searcher` (note this will include logs from the previous and currently running container).
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_searcher_container_restarts"
]
```

## searcher: fs_inodes_used

**Descriptions:**

- _searcher: 3e+06+ fs inodes in use by instance_

**Possible solutions:**

- 			- Refer to your OS or cloud provider`s documentation for how to increase inodes.
			- **Kubernetes:** consider provisioning more machines with less resources.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_searcher_fs_inodes_used"
]
```

## searcher: provisioning_container_cpu_usage_long_term

**Descriptions:**

- _searcher: 80%+ or less than 30% container cpu usage total (90th percentile over 1d) across all cores by instance for 336h0m0s_

**Possible solutions:**

- If usage is high:
	- **Kubernetes:** Consider increasing CPU limits in the `Deployment.yaml` for the searcher service.
	- **Docker Compose:** Consider increasing `cpus:` of the searcher container in `docker-compose.yml`.
- If usage is low, consider decreasing the above values.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_searcher_provisioning_container_cpu_usage_long_term"
]
```

## searcher: provisioning_container_memory_usage_long_term

**Descriptions:**

- _searcher: 80%+ or less than 30% container memory usage (1d maximum) by instance for 336h0m0s_

**Possible solutions:**

- If usage is high:
	- **Kubernetes:** Consider increasing memory limits in the `Deployment.yaml` for the searcher service.
	- **Docker Compose:** Consider increasing `memory:` of the searcher container in `docker-compose.yml`.
- If usage is low, consider decreasing the above values.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_searcher_provisioning_container_memory_usage_long_term"
]
```

## searcher: provisioning_container_cpu_usage_short_term

**Descriptions:**

- _searcher: 90%+ container cpu usage total (5m maximum) across all cores by instance for 30m0s_

**Possible solutions:**

- **Kubernetes:** Consider increasing CPU limits in the the relevant `Deployment.yaml`.
- **Docker Compose:** Consider increasing `cpus:` of the searcher container in `docker-compose.yml`.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_searcher_provisioning_container_cpu_usage_short_term"
]
```

## searcher: provisioning_container_memory_usage_short_term

**Descriptions:**

- _searcher: 90%+ container memory usage (5m maximum) by instance_

**Possible solutions:**

- **Kubernetes:** Consider increasing memory limit in relevant `Deployment.yaml`.
- **Docker Compose:** Consider increasing `memory:` of searcher container in `docker-compose.yml`.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_searcher_provisioning_container_memory_usage_short_term"
]
```

## searcher: go_goroutines

**Descriptions:**

- _searcher: 10000+ maximum active goroutines for 10m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_searcher_go_goroutines"
]
```

## searcher: go_gc_duration_seconds

**Descriptions:**

- _searcher: 2s+ maximum go garbage collection duration_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_searcher_go_gc_duration_seconds"
]
```

## searcher: pods_available_percentage

**Descriptions:**

- _searcher: less than 90% percentage pods available for 10m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "critical_searcher_pods_available_percentage"
]
```

## symbols: store_fetch_failures

**Descriptions:**

- _symbols: 5+ store fetch failures every 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_symbols_store_fetch_failures"
]
```

## symbols: current_fetch_queue_size

**Descriptions:**

- _symbols: 25+ current fetch queue size_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_symbols_current_fetch_queue_size"
]
```

## symbols: frontend_internal_api_error_responses

**Descriptions:**

- _symbols: 2%+ frontend-internal API error responses every 5m by route for 5m0s_

**Possible solutions:**

- **Single-container deployments:** Check `docker logs $CONTAINER_ID` for logs starting with `repo-updater` that indicate requests to the frontend service are failing.
- **Kubernetes:**
	- Confirm that `kubectl get pods` shows the `frontend` pods are healthy.
	- Check `kubectl logs symbols` for logs indicate request failures to `frontend` or `frontend-internal`.
- **Docker Compose:**
	- Confirm that `docker ps` shows the `frontend-internal` container is healthy.
	- Check `docker logs symbols` for logs indicating request failures to `frontend` or `frontend-internal`.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_symbols_frontend_internal_api_error_responses"
]
```

## symbols: container_cpu_usage

**Descriptions:**

- _symbols: 99%+ container cpu usage total (1m average) across all cores by instance_

**Possible solutions:**

- **Kubernetes:** Consider increasing CPU limits in the the relevant `Deployment.yaml`.
- **Docker Compose:** Consider increasing `cpus:` of the symbols container in `docker-compose.yml`.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_symbols_container_cpu_usage"
]
```

## symbols: container_memory_usage

**Descriptions:**

- _symbols: 99%+ container memory usage by instance_

**Possible solutions:**

- **Kubernetes:** Consider increasing memory limit in relevant `Deployment.yaml`.
- **Docker Compose:** Consider increasing `memory:` of symbols container in `docker-compose.yml`.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_symbols_container_memory_usage"
]
```

## symbols: container_restarts

**Descriptions:**

- _symbols: 1+ container restarts every 5m by instance_

**Possible solutions:**

- **Kubernetes:**
	- Determine if the pod was OOM killed using `kubectl describe pod symbols` (look for `OOMKilled: true`) and, if so, consider increasing the memory limit in the relevant `Deployment.yaml`.
	- Check the logs before the container restarted to see if there are `panic:` messages or similar using `kubectl logs -p symbols`.
- **Docker Compose:**
	- Determine if the pod was OOM killed using `docker inspect -f '{{json .State}}' symbols` (look for `"OOMKilled":true`) and, if so, consider increasing the memory limit of the symbols container in `docker-compose.yml`.
	- Check the logs before the container restarted to see if there are `panic:` messages or similar using `docker logs symbols` (note this will include logs from the previous and currently running container).
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_symbols_container_restarts"
]
```

## symbols: fs_inodes_used

**Descriptions:**

- _symbols: 3e+06+ fs inodes in use by instance_

**Possible solutions:**

- 			- Refer to your OS or cloud provider`s documentation for how to increase inodes.
			- **Kubernetes:** consider provisioning more machines with less resources.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_symbols_fs_inodes_used"
]
```

## symbols: provisioning_container_cpu_usage_long_term

**Descriptions:**

- _symbols: 80%+ or less than 30% container cpu usage total (90th percentile over 1d) across all cores by instance for 336h0m0s_

**Possible solutions:**

- If usage is high:
	- **Kubernetes:** Consider increasing CPU limits in the `Deployment.yaml` for the symbols service.
	- **Docker Compose:** Consider increasing `cpus:` of the symbols container in `docker-compose.yml`.
- If usage is low, consider decreasing the above values.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_symbols_provisioning_container_cpu_usage_long_term"
]
```

## symbols: provisioning_container_memory_usage_long_term

**Descriptions:**

- _symbols: 80%+ or less than 30% container memory usage (1d maximum) by instance for 336h0m0s_

**Possible solutions:**

- If usage is high:
	- **Kubernetes:** Consider increasing memory limits in the `Deployment.yaml` for the symbols service.
	- **Docker Compose:** Consider increasing `memory:` of the symbols container in `docker-compose.yml`.
- If usage is low, consider decreasing the above values.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_symbols_provisioning_container_memory_usage_long_term"
]
```

## symbols: provisioning_container_cpu_usage_short_term

**Descriptions:**

- _symbols: 90%+ container cpu usage total (5m maximum) across all cores by instance for 30m0s_

**Possible solutions:**

- **Kubernetes:** Consider increasing CPU limits in the the relevant `Deployment.yaml`.
- **Docker Compose:** Consider increasing `cpus:` of the symbols container in `docker-compose.yml`.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_symbols_provisioning_container_cpu_usage_short_term"
]
```

## symbols: provisioning_container_memory_usage_short_term

**Descriptions:**

- _symbols: 90%+ container memory usage (5m maximum) by instance_

**Possible solutions:**

- **Kubernetes:** Consider increasing memory limit in relevant `Deployment.yaml`.
- **Docker Compose:** Consider increasing `memory:` of symbols container in `docker-compose.yml`.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_symbols_provisioning_container_memory_usage_short_term"
]
```

## symbols: go_goroutines

**Descriptions:**

- _symbols: 10000+ maximum active goroutines for 10m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_symbols_go_goroutines"
]
```

## symbols: go_gc_duration_seconds

**Descriptions:**

- _symbols: 2s+ maximum go garbage collection duration_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_symbols_go_gc_duration_seconds"
]
```

## symbols: pods_available_percentage

**Descriptions:**

- _symbols: less than 90% percentage pods available for 10m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "critical_symbols_pods_available_percentage"
]
```

## syntect-server: syntax_highlighting_errors

**Descriptions:**

- _syntect-server: 5%+ syntax highlighting errors every 5m for 5m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_syntect-server_syntax_highlighting_errors"
]
```

## syntect-server: syntax_highlighting_timeouts

**Descriptions:**

- _syntect-server: 5%+ syntax highlighting timeouts every 5m for 5m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_syntect-server_syntax_highlighting_timeouts"
]
```

## syntect-server: syntax_highlighting_panics

**Descriptions:**

- _syntect-server: 5+ syntax highlighting panics every 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_syntect-server_syntax_highlighting_panics"
]
```

## syntect-server: syntax_highlighting_worker_deaths

**Descriptions:**

- _syntect-server: 1+ syntax highlighter worker deaths every 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_syntect-server_syntax_highlighting_worker_deaths"
]
```

## syntect-server: container_cpu_usage

**Descriptions:**

- _syntect-server: 99%+ container cpu usage total (1m average) across all cores by instance_

**Possible solutions:**

- **Kubernetes:** Consider increasing CPU limits in the the relevant `Deployment.yaml`.
- **Docker Compose:** Consider increasing `cpus:` of the syntect-server container in `docker-compose.yml`.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_syntect-server_container_cpu_usage"
]
```

## syntect-server: container_memory_usage

**Descriptions:**

- _syntect-server: 99%+ container memory usage by instance_

**Possible solutions:**

- **Kubernetes:** Consider increasing memory limit in relevant `Deployment.yaml`.
- **Docker Compose:** Consider increasing `memory:` of syntect-server container in `docker-compose.yml`.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_syntect-server_container_memory_usage"
]
```

## syntect-server: container_restarts

**Descriptions:**

- _syntect-server: 1+ container restarts every 5m by instance_

**Possible solutions:**

- **Kubernetes:**
	- Determine if the pod was OOM killed using `kubectl describe pod syntect-server` (look for `OOMKilled: true`) and, if so, consider increasing the memory limit in the relevant `Deployment.yaml`.
	- Check the logs before the container restarted to see if there are `panic:` messages or similar using `kubectl logs -p syntect-server`.
- **Docker Compose:**
	- Determine if the pod was OOM killed using `docker inspect -f '{{json .State}}' syntect-server` (look for `"OOMKilled":true`) and, if so, consider increasing the memory limit of the syntect-server container in `docker-compose.yml`.
	- Check the logs before the container restarted to see if there are `panic:` messages or similar using `docker logs syntect-server` (note this will include logs from the previous and currently running container).
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_syntect-server_container_restarts"
]
```

## syntect-server: fs_inodes_used

**Descriptions:**

- _syntect-server: 3e+06+ fs inodes in use by instance_

**Possible solutions:**

- 			- Refer to your OS or cloud provider`s documentation for how to increase inodes.
			- **Kubernetes:** consider provisioning more machines with less resources.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_syntect-server_fs_inodes_used"
]
```

## syntect-server: provisioning_container_cpu_usage_long_term

**Descriptions:**

- _syntect-server: 80%+ or less than 30% container cpu usage total (90th percentile over 1d) across all cores by instance for 336h0m0s_

**Possible solutions:**

- If usage is high:
	- **Kubernetes:** Consider increasing CPU limits in the `Deployment.yaml` for the syntect-server service.
	- **Docker Compose:** Consider increasing `cpus:` of the syntect-server container in `docker-compose.yml`.
- If usage is low, consider decreasing the above values.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_syntect-server_provisioning_container_cpu_usage_long_term"
]
```

## syntect-server: provisioning_container_memory_usage_long_term

**Descriptions:**

- _syntect-server: 80%+ or less than 30% container memory usage (1d maximum) by instance for 336h0m0s_

**Possible solutions:**

- If usage is high:
	- **Kubernetes:** Consider increasing memory limits in the `Deployment.yaml` for the syntect-server service.
	- **Docker Compose:** Consider increasing `memory:` of the syntect-server container in `docker-compose.yml`.
- If usage is low, consider decreasing the above values.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_syntect-server_provisioning_container_memory_usage_long_term"
]
```

## syntect-server: provisioning_container_cpu_usage_short_term

**Descriptions:**

- _syntect-server: 90%+ container cpu usage total (5m maximum) across all cores by instance for 30m0s_

**Possible solutions:**

- **Kubernetes:** Consider increasing CPU limits in the the relevant `Deployment.yaml`.
- **Docker Compose:** Consider increasing `cpus:` of the syntect-server container in `docker-compose.yml`.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_syntect-server_provisioning_container_cpu_usage_short_term"
]
```

## syntect-server: provisioning_container_memory_usage_short_term

**Descriptions:**

- _syntect-server: 90%+ container memory usage (5m maximum) by instance_

**Possible solutions:**

- **Kubernetes:** Consider increasing memory limit in relevant `Deployment.yaml`.
- **Docker Compose:** Consider increasing `memory:` of syntect-server container in `docker-compose.yml`.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_syntect-server_provisioning_container_memory_usage_short_term"
]
```

## syntect-server: pods_available_percentage

**Descriptions:**

- _syntect-server: less than 90% percentage pods available for 10m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "critical_syntect-server_pods_available_percentage"
]
```

## zoekt-indexserver: average_resolve_revision_duration

**Descriptions:**

- _zoekt-indexserver: 15s+ average resolve revision duration over 5m_

- _zoekt-indexserver: 30s+ average resolve revision duration over 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_zoekt-indexserver_average_resolve_revision_duration",
  "critical_zoekt-indexserver_average_resolve_revision_duration"
]
```

## zoekt-indexserver: container_cpu_usage

**Descriptions:**

- _zoekt-indexserver: 99%+ container cpu usage total (1m average) across all cores by instance_

**Possible solutions:**

- **Kubernetes:** Consider increasing CPU limits in the the relevant `Deployment.yaml`.
- **Docker Compose:** Consider increasing `cpus:` of the zoekt-indexserver container in `docker-compose.yml`.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_zoekt-indexserver_container_cpu_usage"
]
```

## zoekt-indexserver: container_memory_usage

**Descriptions:**

- _zoekt-indexserver: 99%+ container memory usage by instance_

**Possible solutions:**

- **Kubernetes:** Consider increasing memory limit in relevant `Deployment.yaml`.
- **Docker Compose:** Consider increasing `memory:` of zoekt-indexserver container in `docker-compose.yml`.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_zoekt-indexserver_container_memory_usage"
]
```

## zoekt-indexserver: container_restarts

**Descriptions:**

- _zoekt-indexserver: 1+ container restarts every 5m by instance_

**Possible solutions:**

- **Kubernetes:**
	- Determine if the pod was OOM killed using `kubectl describe pod zoekt-indexserver` (look for `OOMKilled: true`) and, if so, consider increasing the memory limit in the relevant `Deployment.yaml`.
	- Check the logs before the container restarted to see if there are `panic:` messages or similar using `kubectl logs -p zoekt-indexserver`.
- **Docker Compose:**
	- Determine if the pod was OOM killed using `docker inspect -f '{{json .State}}' zoekt-indexserver` (look for `"OOMKilled":true`) and, if so, consider increasing the memory limit of the zoekt-indexserver container in `docker-compose.yml`.
	- Check the logs before the container restarted to see if there are `panic:` messages or similar using `docker logs zoekt-indexserver` (note this will include logs from the previous and currently running container).
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_zoekt-indexserver_container_restarts"
]
```

## zoekt-indexserver: fs_inodes_used

**Descriptions:**

- _zoekt-indexserver: 3e+06+ fs inodes in use by instance_

**Possible solutions:**

- 			- Refer to your OS or cloud provider`s documentation for how to increase inodes.
			- **Kubernetes:** consider provisioning more machines with less resources.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_zoekt-indexserver_fs_inodes_used"
]
```

## zoekt-indexserver: fs_io_operations

**Descriptions:**

- _zoekt-indexserver: 5000+ filesystem reads and writes rate by instance over 1h_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_zoekt-indexserver_fs_io_operations"
]
```

## zoekt-indexserver: provisioning_container_cpu_usage_long_term

**Descriptions:**

- _zoekt-indexserver: 80%+ or less than 30% container cpu usage total (90th percentile over 1d) across all cores by instance for 336h0m0s_

**Possible solutions:**

- If usage is high:
	- **Kubernetes:** Consider increasing CPU limits in the `Deployment.yaml` for the zoekt-indexserver service.
	- **Docker Compose:** Consider increasing `cpus:` of the zoekt-indexserver container in `docker-compose.yml`.
- If usage is low, consider decreasing the above values.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_zoekt-indexserver_provisioning_container_cpu_usage_long_term"
]
```

## zoekt-indexserver: provisioning_container_memory_usage_long_term

**Descriptions:**

- _zoekt-indexserver: 80%+ or less than 30% container memory usage (1d maximum) by instance for 336h0m0s_

**Possible solutions:**

- If usage is high:
	- **Kubernetes:** Consider increasing memory limits in the `Deployment.yaml` for the zoekt-indexserver service.
	- **Docker Compose:** Consider increasing `memory:` of the zoekt-indexserver container in `docker-compose.yml`.
- If usage is low, consider decreasing the above values.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_zoekt-indexserver_provisioning_container_memory_usage_long_term"
]
```

## zoekt-indexserver: provisioning_container_cpu_usage_short_term

**Descriptions:**

- _zoekt-indexserver: 90%+ container cpu usage total (5m maximum) across all cores by instance for 30m0s_

**Possible solutions:**

- **Kubernetes:** Consider increasing CPU limits in the the relevant `Deployment.yaml`.
- **Docker Compose:** Consider increasing `cpus:` of the zoekt-indexserver container in `docker-compose.yml`.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_zoekt-indexserver_provisioning_container_cpu_usage_short_term"
]
```

## zoekt-indexserver: provisioning_container_memory_usage_short_term

**Descriptions:**

- _zoekt-indexserver: 90%+ container memory usage (5m maximum) by instance_

**Possible solutions:**

- **Kubernetes:** Consider increasing memory limit in relevant `Deployment.yaml`.
- **Docker Compose:** Consider increasing `memory:` of zoekt-indexserver container in `docker-compose.yml`.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_zoekt-indexserver_provisioning_container_memory_usage_short_term"
]
```

## zoekt-indexserver: pods_available_percentage

**Descriptions:**

- _zoekt-indexserver: less than 90% percentage pods available for 10m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "critical_zoekt-indexserver_pods_available_percentage"
]
```

## zoekt-webserver: indexed_search_request_errors

**Descriptions:**

- _zoekt-webserver: 5%+ indexed search request errors every 5m by code for 5m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_zoekt-webserver_indexed_search_request_errors"
]
```

## zoekt-webserver: container_cpu_usage

**Descriptions:**

- _zoekt-webserver: 99%+ container cpu usage total (1m average) across all cores by instance_

**Possible solutions:**

- **Kubernetes:** Consider increasing CPU limits in the the relevant `Deployment.yaml`.
- **Docker Compose:** Consider increasing `cpus:` of the zoekt-webserver container in `docker-compose.yml`.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_zoekt-webserver_container_cpu_usage"
]
```

## zoekt-webserver: container_memory_usage

**Descriptions:**

- _zoekt-webserver: 99%+ container memory usage by instance_

**Possible solutions:**

- **Kubernetes:** Consider increasing memory limit in relevant `Deployment.yaml`.
- **Docker Compose:** Consider increasing `memory:` of zoekt-webserver container in `docker-compose.yml`.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_zoekt-webserver_container_memory_usage"
]
```

## zoekt-webserver: container_restarts

**Descriptions:**

- _zoekt-webserver: 1+ container restarts every 5m by instance_

**Possible solutions:**

- **Kubernetes:**
	- Determine if the pod was OOM killed using `kubectl describe pod zoekt-webserver` (look for `OOMKilled: true`) and, if so, consider increasing the memory limit in the relevant `Deployment.yaml`.
	- Check the logs before the container restarted to see if there are `panic:` messages or similar using `kubectl logs -p zoekt-webserver`.
- **Docker Compose:**
	- Determine if the pod was OOM killed using `docker inspect -f '{{json .State}}' zoekt-webserver` (look for `"OOMKilled":true`) and, if so, consider increasing the memory limit of the zoekt-webserver container in `docker-compose.yml`.
	- Check the logs before the container restarted to see if there are `panic:` messages or similar using `docker logs zoekt-webserver` (note this will include logs from the previous and currently running container).
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_zoekt-webserver_container_restarts"
]
```

## zoekt-webserver: fs_inodes_used

**Descriptions:**

- _zoekt-webserver: 3e+06+ fs inodes in use by instance_

**Possible solutions:**

- 			- Refer to your OS or cloud provider`s documentation for how to increase inodes.
			- **Kubernetes:** consider provisioning more machines with less resources.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_zoekt-webserver_fs_inodes_used"
]
```

## zoekt-webserver: fs_io_operations

**Descriptions:**

- _zoekt-webserver: 5000+ filesystem reads and writes by instance rate over 1h_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_zoekt-webserver_fs_io_operations"
]
```

## zoekt-webserver: provisioning_container_cpu_usage_long_term

**Descriptions:**

- _zoekt-webserver: 80%+ or less than 30% container cpu usage total (90th percentile over 1d) across all cores by instance for 336h0m0s_

**Possible solutions:**

- If usage is high:
	- **Kubernetes:** Consider increasing CPU limits in the `Deployment.yaml` for the zoekt-webserver service.
	- **Docker Compose:** Consider increasing `cpus:` of the zoekt-webserver container in `docker-compose.yml`.
- If usage is low, consider decreasing the above values.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_zoekt-webserver_provisioning_container_cpu_usage_long_term"
]
```

## zoekt-webserver: provisioning_container_memory_usage_long_term

**Descriptions:**

- _zoekt-webserver: 80%+ or less than 30% container memory usage (1d maximum) by instance for 336h0m0s_

**Possible solutions:**

- If usage is high:
	- **Kubernetes:** Consider increasing memory limits in the `Deployment.yaml` for the zoekt-webserver service.
	- **Docker Compose:** Consider increasing `memory:` of the zoekt-webserver container in `docker-compose.yml`.
- If usage is low, consider decreasing the above values.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_zoekt-webserver_provisioning_container_memory_usage_long_term"
]
```

## zoekt-webserver: provisioning_container_cpu_usage_short_term

**Descriptions:**

- _zoekt-webserver: 90%+ container cpu usage total (5m maximum) across all cores by instance for 30m0s_

**Possible solutions:**

- **Kubernetes:** Consider increasing CPU limits in the the relevant `Deployment.yaml`.
- **Docker Compose:** Consider increasing `cpus:` of the zoekt-webserver container in `docker-compose.yml`.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_zoekt-webserver_provisioning_container_cpu_usage_short_term"
]
```

## zoekt-webserver: provisioning_container_memory_usage_short_term

**Descriptions:**

- _zoekt-webserver: 90%+ container memory usage (5m maximum) by instance_

**Possible solutions:**

- **Kubernetes:** Consider increasing memory limit in relevant `Deployment.yaml`.
- **Docker Compose:** Consider increasing `memory:` of zoekt-webserver container in `docker-compose.yml`.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_zoekt-webserver_provisioning_container_memory_usage_short_term"
]
```

## prometheus: prometheus_metrics_bloat

**Descriptions:**

- _prometheus: 20000B+ prometheus metrics payload size_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_prometheus_prometheus_metrics_bloat"
]
```

## prometheus: alertmanager_notifications_failed_total

**Descriptions:**

- _prometheus: 1+ failed alertmanager notifications over 1m_

**Possible solutions:**

- Ensure that your [`observability.alerts` configuration](https://docs.sourcegraph.com/admin/observability/alerting#setting-up-alerting) (in site configuration) is valid.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_prometheus_alertmanager_notifications_failed_total"
]
```

## prometheus: container_cpu_usage

**Descriptions:**

- _prometheus: 99%+ container cpu usage total (1m average) across all cores by instance_

**Possible solutions:**

- **Kubernetes:** Consider increasing CPU limits in the the relevant `Deployment.yaml`.
- **Docker Compose:** Consider increasing `cpus:` of the prometheus container in `docker-compose.yml`.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_prometheus_container_cpu_usage"
]
```

## prometheus: container_memory_usage

**Descriptions:**

- _prometheus: 99%+ container memory usage by instance_

**Possible solutions:**

- **Kubernetes:** Consider increasing memory limit in relevant `Deployment.yaml`.
- **Docker Compose:** Consider increasing `memory:` of prometheus container in `docker-compose.yml`.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_prometheus_container_memory_usage"
]
```

## prometheus: container_restarts

**Descriptions:**

- _prometheus: 1+ container restarts every 5m by instance_

**Possible solutions:**

- **Kubernetes:**
	- Determine if the pod was OOM killed using `kubectl describe pod prometheus` (look for `OOMKilled: true`) and, if so, consider increasing the memory limit in the relevant `Deployment.yaml`.
	- Check the logs before the container restarted to see if there are `panic:` messages or similar using `kubectl logs -p prometheus`.
- **Docker Compose:**
	- Determine if the pod was OOM killed using `docker inspect -f '{{json .State}}' prometheus` (look for `"OOMKilled":true`) and, if so, consider increasing the memory limit of the prometheus container in `docker-compose.yml`.
	- Check the logs before the container restarted to see if there are `panic:` messages or similar using `docker logs prometheus` (note this will include logs from the previous and currently running container).
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_prometheus_container_restarts"
]
```

## prometheus: fs_inodes_used

**Descriptions:**

- _prometheus: 3e+06+ fs inodes in use by instance_

**Possible solutions:**

- 			- Refer to your OS or cloud provider`s documentation for how to increase inodes.
			- **Kubernetes:** consider provisioning more machines with less resources.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_prometheus_fs_inodes_used"
]
```

## prometheus: provisioning_container_cpu_usage_long_term

**Descriptions:**

- _prometheus: 80%+ or less than 30% container cpu usage total (90th percentile over 1d) across all cores by instance for 336h0m0s_

**Possible solutions:**

- If usage is high:
	- **Kubernetes:** Consider increasing CPU limits in the `Deployment.yaml` for the prometheus service.
	- **Docker Compose:** Consider increasing `cpus:` of the prometheus container in `docker-compose.yml`.
- If usage is low, consider decreasing the above values.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_prometheus_provisioning_container_cpu_usage_long_term"
]
```

## prometheus: provisioning_container_memory_usage_long_term

**Descriptions:**

- _prometheus: 80%+ or less than 30% container memory usage (1d maximum) by instance for 336h0m0s_

**Possible solutions:**

- If usage is high:
	- **Kubernetes:** Consider increasing memory limits in the `Deployment.yaml` for the prometheus service.
	- **Docker Compose:** Consider increasing `memory:` of the prometheus container in `docker-compose.yml`.
- If usage is low, consider decreasing the above values.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_prometheus_provisioning_container_memory_usage_long_term"
]
```

## prometheus: provisioning_container_cpu_usage_short_term

**Descriptions:**

- _prometheus: 90%+ container cpu usage total (5m maximum) across all cores by instance for 30m0s_

**Possible solutions:**

- **Kubernetes:** Consider increasing CPU limits in the the relevant `Deployment.yaml`.
- **Docker Compose:** Consider increasing `cpus:` of the prometheus container in `docker-compose.yml`.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_prometheus_provisioning_container_cpu_usage_short_term"
]
```

## prometheus: provisioning_container_memory_usage_short_term

**Descriptions:**

- _prometheus: 90%+ container memory usage (5m maximum) by instance_

**Possible solutions:**

- **Kubernetes:** Consider increasing memory limit in relevant `Deployment.yaml`.
- **Docker Compose:** Consider increasing `memory:` of prometheus container in `docker-compose.yml`.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_prometheus_provisioning_container_memory_usage_short_term"
]
```

## prometheus: pods_available_percentage

**Descriptions:**

- _prometheus: less than 90% percentage pods available for 10m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "critical_prometheus_pods_available_percentage"
]
```

