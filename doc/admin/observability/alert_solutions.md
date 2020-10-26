# Alert solutions

This document contains possible solutions for when you find alerts are firing in Sourcegraph's monitoring.
If your alert isn't mentioned here, or if the solution doesn't help, [contact us](mailto:support@sourcegraph.com)
for assistance.

To learn more about Sourcegraph's alerting, see [our alerting documentation](https://docs.sourcegraph.com/admin/observability/alerting).

<!-- DO NOT EDIT: generated via: go generate ./monitoring -->

## frontend: 99th_percentile_search_request_duration

<p class="subtitle">search: 99th percentile successful search request duration over 5m</p>**Descriptions:**

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

<br />
## frontend: 90th_percentile_search_request_duration

<p class="subtitle">search: 90th percentile successful search request duration over 5m</p>**Descriptions:**

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

<br />
## frontend: hard_timeout_search_responses

<p class="subtitle">search: hard timeout search responses every 5m</p>**Descriptions:**

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

<br />
## frontend: hard_error_search_responses

<p class="subtitle">search: hard error search responses every 5m</p>**Descriptions:**

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

<br />
## frontend: partial_timeout_search_responses

<p class="subtitle">search: partial timeout search responses every 5m</p>**Descriptions:**

- _frontend: 5%+ partial timeout search responses every 5m for 15m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_frontend_partial_timeout_search_responses"
]
```

<br />
## frontend: search_alert_user_suggestions

<p class="subtitle">search: search alert user suggestions shown every 5m</p>**Descriptions:**

- _frontend: 5%+ search alert user suggestions shown every 5m for 15m0s_

**Possible solutions:**

- This indicates your user`s are making syntax errors or similar user errors.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_frontend_search_alert_user_suggestions"
]
```

<br />
## frontend: page_load_latency

<p class="subtitle">cloud: 90th percentile page load latency over all routes over 10m</p>**Descriptions:**

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

<br />
## frontend: blob_load_latency

<p class="subtitle">cloud: 90th percentile blob load latency over 10m</p>**Descriptions:**

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

<br />
## frontend: 99th_percentile_search_codeintel_request_duration

<p class="subtitle">code-intel: 99th percentile code-intel successful search request duration over 5m</p>**Descriptions:**

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

<br />
## frontend: 90th_percentile_search_codeintel_request_duration

<p class="subtitle">code-intel: 90th percentile code-intel successful search request duration over 5m</p>**Descriptions:**

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

<br />
## frontend: hard_timeout_search_codeintel_responses

<p class="subtitle">code-intel: hard timeout search code-intel responses every 5m</p>**Descriptions:**

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

<br />
## frontend: hard_error_search_codeintel_responses

<p class="subtitle">code-intel: hard error search code-intel responses every 5m</p>**Descriptions:**

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

<br />
## frontend: partial_timeout_search_codeintel_responses

<p class="subtitle">code-intel: partial timeout search code-intel responses every 5m</p>**Descriptions:**

- _frontend: 5%+ partial timeout search code-intel responses every 5m for 15m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_frontend_partial_timeout_search_codeintel_responses"
]
```

<br />
## frontend: search_codeintel_alert_user_suggestions

<p class="subtitle">code-intel: search code-intel alert user suggestions shown every 5m</p>**Descriptions:**

- _frontend: 5%+ search code-intel alert user suggestions shown every 5m for 15m0s_

**Possible solutions:**

- This indicates a bug in Sourcegraph, please [open an issue](https://github.com/sourcegraph/sourcegraph/issues/new/choose).
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_frontend_search_codeintel_alert_user_suggestions"
]
```

<br />
## frontend: 99th_percentile_search_api_request_duration

<p class="subtitle">search: 99th percentile successful search API request duration over 5m</p>**Descriptions:**

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

<br />
## frontend: 90th_percentile_search_api_request_duration

<p class="subtitle">search: 90th percentile successful search API request duration over 5m</p>**Descriptions:**

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

<br />
## frontend: hard_timeout_search_api_responses

<p class="subtitle">search: hard timeout search API responses every 5m</p>**Descriptions:**

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

<br />
## frontend: hard_error_search_api_responses

<p class="subtitle">search: hard error search API responses every 5m</p>**Descriptions:**

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

<br />
## frontend: partial_timeout_search_api_responses

<p class="subtitle">search: partial timeout search API responses every 5m</p>**Descriptions:**

- _frontend: 5%+ partial timeout search API responses every 5m for 15m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_frontend_partial_timeout_search_api_responses"
]
```

<br />
## frontend: search_api_alert_user_suggestions

<p class="subtitle">search: search API alert user suggestions shown every 5m</p>**Descriptions:**

- _frontend: 5%+ search API alert user suggestions shown every 5m_

**Possible solutions:**

- This indicates your user`s search API requests have syntax errors or a similar user error. Check the responses the API sends back for an explanation.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_frontend_search_api_alert_user_suggestions"
]
```

<br />
## frontend: 99th_percentile_precise_code_intel_api_duration

<p class="subtitle">code-intel: 99th percentile successful precise code intel api query duration over 5m</p>**Descriptions:**

- _frontend: 20s+ 99th percentile successful precise code intel api query duration over 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_frontend_99th_percentile_precise_code_intel_api_duration"
]
```

<br />
## frontend: precise_code_intel_api_errors

<p class="subtitle">code-intel: precise code intel api errors every 5m</p>**Descriptions:**

- _frontend: 5%+ precise code intel api errors every 5m for 15m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_frontend_precise_code_intel_api_errors"
]
```

<br />
## frontend: code_intel_frontend_db_store_99th_percentile_duration

<p class="subtitle">code-intel: 99th percentile successful frontend database query duration over 5m</p>**Descriptions:**

- _frontend: 20s+ 99th percentile successful frontend database query duration over 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_frontend_code_intel_frontend_db_store_99th_percentile_duration"
]
```

<br />
## frontend: code_intel_frontend_db_store_errors

<p class="subtitle">code-intel: frontend database errors every 5m</p>**Descriptions:**

- _frontend: 20+ frontend database errors every 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_frontend_code_intel_frontend_db_store_errors"
]
```

<br />
## frontend: code_intel_codeintel_db_store_99th_percentile_duration

<p class="subtitle">code-intel: 99th percentile successful codeintel database query duration over 5m</p>**Descriptions:**

- _frontend: 20s+ 99th percentile successful codeintel database query duration over 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_frontend_code_intel_codeintel_db_store_99th_percentile_duration"
]
```

<br />
## frontend: code_intel_codeintel_db_store_errors

<p class="subtitle">code-intel: codeintel database every 5m</p>**Descriptions:**

- _frontend: 20+ codeintel database every 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_frontend_code_intel_codeintel_db_store_errors"
]
```

<br />
## frontend: code_intel_bundle_store_99th_percentile_duration

<p class="subtitle">code-intel: 99th percentile successful bundle database store operation duration over 5m</p>**Descriptions:**

- _frontend: 20s+ 99th percentile successful bundle database store operation duration over 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_frontend_code_intel_bundle_store_99th_percentile_duration"
]
```

<br />
## frontend: code_intel_bundle_store_errors

<p class="subtitle">code-intel: bundle store errors every 5m</p>**Descriptions:**

- _frontend: 20+ bundle store errors every 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_frontend_code_intel_bundle_store_errors"
]
```

<br />
## frontend: internal_indexed_search_error_responses

<p class="subtitle">search: internal indexed search error responses every 5m</p>**Descriptions:**

- _frontend: 5%+ internal indexed search error responses every 5m for 15m0s_

**Possible solutions:**

- Check the Zoekt Web Server dashboard for indications it might be unhealthy.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_frontend_internal_indexed_search_error_responses"
]
```

<br />
## frontend: internal_unindexed_search_error_responses

<p class="subtitle">search: internal unindexed search error responses every 5m</p>**Descriptions:**

- _frontend: 5%+ internal unindexed search error responses every 5m for 15m0s_

**Possible solutions:**

- Check the Searcher dashboard for indications it might be unhealthy.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_frontend_internal_unindexed_search_error_responses"
]
```

<br />
## frontend: internal_api_error_responses

<p class="subtitle">cloud: internal API error responses every 5m by route</p>**Descriptions:**

- _frontend: 5%+ internal API error responses every 5m by route for 15m0s_

**Possible solutions:**

- May not be a substantial issue, check the `frontend` logs for potential causes.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_frontend_internal_api_error_responses"
]
```

<br />
## frontend: 99th_percentile_precise_code_intel_bundle_manager_query_duration

<p class="subtitle">code-intel: 99th percentile successful precise-code-intel-bundle-manager query duration over 5m</p>**Descriptions:**

- _frontend: 20s+ 99th percentile successful precise-code-intel-bundle-manager query duration over 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_frontend_99th_percentile_precise_code_intel_bundle_manager_query_duration"
]
```

<br />
## frontend: 99th_percentile_precise_code_intel_bundle_manager_transfer_duration

<p class="subtitle">code-intel: 99th percentile successful precise-code-intel-bundle-manager data transfer duration over 5m</p>**Descriptions:**

- _frontend: 300s+ 99th percentile successful precise-code-intel-bundle-manager data transfer duration over 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_frontend_99th_percentile_precise_code_intel_bundle_manager_transfer_duration"
]
```

<br />
## frontend: precise_code_intel_bundle_manager_error_responses

<p class="subtitle">code-intel: precise-code-intel-bundle-manager error responses every 5m</p>**Descriptions:**

- _frontend: 5%+ precise-code-intel-bundle-manager error responses every 5m for 15m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_frontend_precise_code_intel_bundle_manager_error_responses"
]
```

<br />
## frontend: 99th_percentile_gitserver_duration

<p class="subtitle">cloud: 99th percentile successful gitserver query duration over 5m</p>**Descriptions:**

- _frontend: 20s+ 99th percentile successful gitserver query duration over 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_frontend_99th_percentile_gitserver_duration"
]
```

<br />
## frontend: gitserver_error_responses

<p class="subtitle">cloud: gitserver error responses every 5m</p>**Descriptions:**

- _frontend: 5%+ gitserver error responses every 5m for 15m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_frontend_gitserver_error_responses"
]
```

<br />
## frontend: observability_test_alert_warning

<p class="subtitle">distribution: warning test alert metric</p>**Descriptions:**

- _frontend: 1+ warning test alert metric_

**Possible solutions:**

- This alert is triggered via the `triggerObservabilityTestAlert` GraphQL endpoint, and will automatically resolve itself.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_frontend_observability_test_alert_warning"
]
```

<br />
## frontend: observability_test_alert_critical

<p class="subtitle">distribution: critical test alert metric</p>**Descriptions:**

- _frontend: 1+ critical test alert metric_

**Possible solutions:**

- This alert is triggered via the `triggerObservabilityTestAlert` GraphQL endpoint, and will automatically resolve itself.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "critical_frontend_observability_test_alert_critical"
]
```

<br />
## frontend: container_cpu_usage

<p class="subtitle">cloud: container cpu usage total (1m average) across all cores by instance</p>**Descriptions:**

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

<br />
## frontend: container_memory_usage

<p class="subtitle">cloud: container memory usage by instance</p>**Descriptions:**

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

<br />
## frontend: container_restarts

<p class="subtitle">cloud: container restarts every 5m by instance</p>**Descriptions:**

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

<br />
## frontend: fs_inodes_used

<p class="subtitle">cloud: fs inodes in use by instance</p>**Descriptions:**

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

<br />
## frontend: provisioning_container_cpu_usage_long_term

<p class="subtitle">cloud: container cpu usage total (90th percentile over 1d) across all cores by instance</p>**Descriptions:**

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

<br />
## frontend: provisioning_container_memory_usage_long_term

<p class="subtitle">cloud: container memory usage (1d maximum) by instance</p>**Descriptions:**

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

<br />
## frontend: provisioning_container_cpu_usage_short_term

<p class="subtitle">cloud: container cpu usage total (5m maximum) across all cores by instance</p>**Descriptions:**

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

<br />
## frontend: provisioning_container_memory_usage_short_term

<p class="subtitle">cloud: container memory usage (5m maximum) by instance</p>**Descriptions:**

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

<br />
## frontend: go_goroutines

<p class="subtitle">cloud: maximum active goroutines</p>**Descriptions:**

- _frontend: 10000+ maximum active goroutines for 10m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_frontend_go_goroutines"
]
```

<br />
## frontend: go_gc_duration_seconds

<p class="subtitle">cloud: maximum go garbage collection duration</p>**Descriptions:**

- _frontend: 2s+ maximum go garbage collection duration_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_frontend_go_gc_duration_seconds"
]
```

<br />
## frontend: pods_available_percentage

<p class="subtitle">cloud: percentage pods available</p>**Descriptions:**

- _frontend: less than 90% percentage pods available for 10m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "critical_frontend_pods_available_percentage"
]
```

<br />
## gitserver: disk_space_remaining

<p class="subtitle">cloud: disk space remaining by instance</p>**Descriptions:**

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

<br />
## gitserver: running_git_commands

<p class="subtitle">cloud: running git commands (signals load)</p>**Descriptions:**

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

<br />
## gitserver: repository_clone_queue_size

<p class="subtitle">cloud: repository clone queue size</p>**Descriptions:**

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

<br />
## gitserver: repository_existence_check_queue_size

<p class="subtitle">cloud: repository existence check queue size</p>**Descriptions:**

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

<br />
## gitserver: echo_command_duration_test

<p class="subtitle">cloud: echo command duration test</p>**Descriptions:**

- _gitserver: 1s+ echo command duration test_
- _gitserver: 2s+ echo command duration test_

**Possible solutions:**

- **Query a graph for individual commands** using `sum by (cmd)(src_gitserver_exec_running)` in Grafana (`/-/debug/grafana`) to see if a command might be spiking in frequency.
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

<br />
## gitserver: frontend_internal_api_error_responses

<p class="subtitle">cloud: frontend-internal API error responses every 5m by route</p>**Descriptions:**

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

<br />
## gitserver: container_cpu_usage

<p class="subtitle">cloud: container cpu usage total (1m average) across all cores by instance</p>**Descriptions:**

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

<br />
## gitserver: container_memory_usage

<p class="subtitle">cloud: container memory usage by instance</p>**Descriptions:**

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

<br />
## gitserver: container_restarts

<p class="subtitle">cloud: container restarts every 5m by instance</p>**Descriptions:**

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

<br />
## gitserver: fs_inodes_used

<p class="subtitle">cloud: fs inodes in use by instance</p>**Descriptions:**

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

<br />
## gitserver: fs_io_operations

<p class="subtitle">search: filesystem reads and writes rate by instance over 1h</p>**Descriptions:**

- _gitserver: 5000+ filesystem reads and writes rate by instance over 1h_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_gitserver_fs_io_operations"
]
```

<br />
## gitserver: provisioning_container_cpu_usage_long_term

<p class="subtitle">cloud: container cpu usage total (90th percentile over 1d) across all cores by instance</p>**Descriptions:**

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

<br />
## gitserver: provisioning_container_memory_usage_long_term

<p class="subtitle">distribution: container memory usage (1d maximum) by instance</p>**Descriptions:**

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

<br />
## gitserver: provisioning_container_cpu_usage_short_term

<p class="subtitle">cloud: container cpu usage total (5m maximum) across all cores by instance</p>**Descriptions:**

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

<br />
## gitserver: go_goroutines

<p class="subtitle">cloud: maximum active goroutines</p>**Descriptions:**

- _gitserver: 10000+ maximum active goroutines for 10m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_gitserver_go_goroutines"
]
```

<br />
## gitserver: go_gc_duration_seconds

<p class="subtitle">cloud: maximum go garbage collection duration</p>**Descriptions:**

- _gitserver: 2s+ maximum go garbage collection duration_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_gitserver_go_gc_duration_seconds"
]
```

<br />
## gitserver: pods_available_percentage

<p class="subtitle">cloud: percentage pods available</p>**Descriptions:**

- _gitserver: less than 90% percentage pods available for 10m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "critical_gitserver_pods_available_percentage"
]
```

<br />
## github-proxy: github_core_rate_limit_remaining

<p class="subtitle">cloud: remaining calls to GitHub before hitting the rate limit</p>**Descriptions:**

- _github-proxy: less than 500 remaining calls to GitHub before hitting the rate limit for 5m0s_

**Possible solutions:**

- Try restarting the pod to get a different public IP.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "critical_github-proxy_github_core_rate_limit_remaining"
]
```

<br />
## github-proxy: github_search_rate_limit_remaining

<p class="subtitle">cloud: remaining calls to GitHub search before hitting the rate limit</p>**Descriptions:**

- _github-proxy: less than 5 remaining calls to GitHub search before hitting the rate limit_

**Possible solutions:**

- Try restarting the pod to get a different public IP.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_github-proxy_github_search_rate_limit_remaining"
]
```

<br />
## github-proxy: container_cpu_usage

<p class="subtitle">cloud: container cpu usage total (1m average) across all cores by instance</p>**Descriptions:**

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

<br />
## github-proxy: container_memory_usage

<p class="subtitle">cloud: container memory usage by instance</p>**Descriptions:**

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

<br />
## github-proxy: container_restarts

<p class="subtitle">cloud: container restarts every 5m by instance</p>**Descriptions:**

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

<br />
## github-proxy: fs_inodes_used

<p class="subtitle">cloud: fs inodes in use by instance</p>**Descriptions:**

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

<br />
## github-proxy: provisioning_container_cpu_usage_long_term

<p class="subtitle">cloud: container cpu usage total (90th percentile over 1d) across all cores by instance</p>**Descriptions:**

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

<br />
## github-proxy: provisioning_container_memory_usage_long_term

<p class="subtitle">cloud: container memory usage (1d maximum) by instance</p>**Descriptions:**

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

<br />
## github-proxy: provisioning_container_cpu_usage_short_term

<p class="subtitle">cloud: container cpu usage total (5m maximum) across all cores by instance</p>**Descriptions:**

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

<br />
## github-proxy: provisioning_container_memory_usage_short_term

<p class="subtitle">cloud: container memory usage (5m maximum) by instance</p>**Descriptions:**

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

<br />
## github-proxy: go_goroutines

<p class="subtitle">cloud: maximum active goroutines</p>**Descriptions:**

- _github-proxy: 10000+ maximum active goroutines for 10m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_github-proxy_go_goroutines"
]
```

<br />
## github-proxy: go_gc_duration_seconds

<p class="subtitle">cloud: maximum go garbage collection duration</p>**Descriptions:**

- _github-proxy: 2s+ maximum go garbage collection duration_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_github-proxy_go_gc_duration_seconds"
]
```

<br />
## github-proxy: pods_available_percentage

<p class="subtitle">cloud: percentage pods available</p>**Descriptions:**

- _github-proxy: less than 90% percentage pods available for 10m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "critical_github-proxy_pods_available_percentage"
]
```

<br />
## precise-code-intel-bundle-manager: code_intel_frontend_db_store_99th_percentile_duration

<p class="subtitle">code-intel: 99th percentile successful frontend database query duration over 5m</p>**Descriptions:**

- _precise-code-intel-bundle-manager: 20s+ 99th percentile successful frontend database query duration over 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-bundle-manager_code_intel_frontend_db_store_99th_percentile_duration"
]
```

<br />
## precise-code-intel-bundle-manager: code_intel_frontend_db_store_errors

<p class="subtitle">code-intel: frontend database errors every 5m</p>**Descriptions:**

- _precise-code-intel-bundle-manager: 20+ frontend database errors every 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-bundle-manager_code_intel_frontend_db_store_errors"
]
```

<br />
## precise-code-intel-bundle-manager: code_intel_codeintel_db_store_99th_percentile_duration

<p class="subtitle">code-intel: 99th percentile successful codeintel database query duration over 5m</p>**Descriptions:**

- _precise-code-intel-bundle-manager: 20s+ 99th percentile successful codeintel database query duration over 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-bundle-manager_code_intel_codeintel_db_store_99th_percentile_duration"
]
```

<br />
## precise-code-intel-bundle-manager: code_intel_codeintel_db_store_errors

<p class="subtitle">code-intel: codeintel database every 5m</p>**Descriptions:**

- _precise-code-intel-bundle-manager: 20+ codeintel database every 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-bundle-manager_code_intel_codeintel_db_store_errors"
]
```

<br />
## precise-code-intel-bundle-manager: code_intel_bundle_store_99th_percentile_duration

<p class="subtitle">code-intel: 99th percentile successful bundle database store operation duration over 5m</p>**Descriptions:**

- _precise-code-intel-bundle-manager: 20s+ 99th percentile successful bundle database store operation duration over 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-bundle-manager_code_intel_bundle_store_99th_percentile_duration"
]
```

<br />
## precise-code-intel-bundle-manager: code_intel_bundle_store_errors

<p class="subtitle">code-intel: bundle store errors every 5m</p>**Descriptions:**

- _precise-code-intel-bundle-manager: 20+ bundle store errors every 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-bundle-manager_code_intel_bundle_store_errors"
]
```

<br />
## precise-code-intel-bundle-manager: disk_space_remaining

<p class="subtitle">code-intel: disk space remaining by instance</p>**Descriptions:**

- _precise-code-intel-bundle-manager: less than 25% disk space remaining by instance_
- _precise-code-intel-bundle-manager: less than 15% disk space remaining by instance_

**Possible solutions:**

- 	- **Provision more disk space:** Sourcegraph will begin deleting the oldest uploaded bundle files at 10% disk space remaining.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-bundle-manager_disk_space_remaining",
  "critical_precise-code-intel-bundle-manager_disk_space_remaining"
]
```

<br />
## precise-code-intel-bundle-manager: janitor_errors

<p class="subtitle">code-intel: janitor errors every 5m</p>**Descriptions:**

- _precise-code-intel-bundle-manager: 20+ janitor errors every 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-bundle-manager_janitor_errors"
]
```

<br />
## precise-code-intel-bundle-manager: janitor_upload_files_removed

<p class="subtitle">code-intel: upload files removed (due to age) every 5m</p>**Descriptions:**

- _precise-code-intel-bundle-manager: 20+ upload files removed (due to age) every 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-bundle-manager_janitor_upload_files_removed"
]
```

<br />
## precise-code-intel-bundle-manager: janitor_upload_part_files_removed

<p class="subtitle">code-intel: upload part files removed (due to age) every 5m</p>**Descriptions:**

- _precise-code-intel-bundle-manager: 20+ upload part files removed (due to age) every 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-bundle-manager_janitor_upload_part_files_removed"
]
```

<br />
## precise-code-intel-bundle-manager: janitor_upload_records_removed

<p class="subtitle">code-intel: upload records removed every 5m</p>**Descriptions:**

- _precise-code-intel-bundle-manager: 20+ upload records removed every 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-bundle-manager_janitor_upload_records_removed"
]
```

<br />
## precise-code-intel-bundle-manager: janitor_data_rows_removed

<p class="subtitle">code-intel: codeintel database rows removed (due to deleted upload) every 5m</p>**Descriptions:**

- _precise-code-intel-bundle-manager: 20+ codeintel database rows removed (due to deleted upload) every 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-bundle-manager_janitor_data_rows_removed"
]
```

<br />
## precise-code-intel-bundle-manager: frontend_internal_api_error_responses

<p class="subtitle">code-intel: frontend-internal API error responses every 5m by route</p>**Descriptions:**

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

<br />
## precise-code-intel-bundle-manager: container_cpu_usage

<p class="subtitle">code-intel: container cpu usage total (1m average) across all cores by instance</p>**Descriptions:**

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

<br />
## precise-code-intel-bundle-manager: container_memory_usage

<p class="subtitle">code-intel: container memory usage by instance</p>**Descriptions:**

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

<br />
## precise-code-intel-bundle-manager: container_restarts

<p class="subtitle">code-intel: container restarts every 5m by instance</p>**Descriptions:**

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

<br />
## precise-code-intel-bundle-manager: fs_inodes_used

<p class="subtitle">code-intel: fs inodes in use by instance</p>**Descriptions:**

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

<br />
## precise-code-intel-bundle-manager: provisioning_container_cpu_usage_long_term

<p class="subtitle">code-intel: container cpu usage total (90th percentile over 1d) across all cores by instance</p>**Descriptions:**

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

<br />
## precise-code-intel-bundle-manager: provisioning_container_memory_usage_long_term

<p class="subtitle">code-intel: container memory usage (1d maximum) by instance</p>**Descriptions:**

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

<br />
## precise-code-intel-bundle-manager: provisioning_container_cpu_usage_short_term

<p class="subtitle">code-intel: container cpu usage total (5m maximum) across all cores by instance</p>**Descriptions:**

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

<br />
## precise-code-intel-bundle-manager: provisioning_container_memory_usage_short_term

<p class="subtitle">code-intel: container memory usage (5m maximum) by instance</p>**Descriptions:**

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

<br />
## precise-code-intel-bundle-manager: go_goroutines

<p class="subtitle">code-intel: maximum active goroutines</p>**Descriptions:**

- _precise-code-intel-bundle-manager: 10000+ maximum active goroutines for 10m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-bundle-manager_go_goroutines"
]
```

<br />
## precise-code-intel-bundle-manager: go_gc_duration_seconds

<p class="subtitle">code-intel: maximum go garbage collection duration</p>**Descriptions:**

- _precise-code-intel-bundle-manager: 2s+ maximum go garbage collection duration_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-bundle-manager_go_gc_duration_seconds"
]
```

<br />
## precise-code-intel-bundle-manager: pods_available_percentage

<p class="subtitle">code-intel: percentage pods available</p>**Descriptions:**

- _precise-code-intel-bundle-manager: less than 90% percentage pods available for 10m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "critical_precise-code-intel-bundle-manager_pods_available_percentage"
]
```

<br />
## precise-code-intel-worker: upload_queue_size

<p class="subtitle">code-intel: upload queue size</p>**Descriptions:**

- _precise-code-intel-worker: 100+ upload queue size_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-worker_upload_queue_size"
]
```

<br />
## precise-code-intel-worker: upload_queue_growth_rate

<p class="subtitle">code-intel: upload queue growth rate every 5m</p>**Descriptions:**

- _precise-code-intel-worker: 5+ upload queue growth rate every 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-worker_upload_queue_growth_rate"
]
```

<br />
## precise-code-intel-worker: upload_process_errors

<p class="subtitle">code-intel: upload process errors every 5m</p>**Descriptions:**

- _precise-code-intel-worker: 20+ upload process errors every 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-worker_upload_process_errors"
]
```

<br />
## precise-code-intel-worker: code_intel_frontend_db_store_99th_percentile_duration

<p class="subtitle">code-intel: 99th percentile successful frontend database query duration over 5m</p>**Descriptions:**

- _precise-code-intel-worker: 20s+ 99th percentile successful frontend database query duration over 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-worker_code_intel_frontend_db_store_99th_percentile_duration"
]
```

<br />
## precise-code-intel-worker: code_intel_frontend_db_store_errors

<p class="subtitle">code-intel: frontend database errors every 5m</p>**Descriptions:**

- _precise-code-intel-worker: 20+ frontend database errors every 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-worker_code_intel_frontend_db_store_errors"
]
```

<br />
## precise-code-intel-worker: code_intel_codeintel_db_store_99th_percentile_duration

<p class="subtitle">code-intel: 99th percentile successful codeintel database query duration over 5m</p>**Descriptions:**

- _precise-code-intel-worker: 20s+ 99th percentile successful codeintel database query duration over 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-worker_code_intel_codeintel_db_store_99th_percentile_duration"
]
```

<br />
## precise-code-intel-worker: code_intel_codeintel_db_store_errors

<p class="subtitle">code-intel: codeintel database errors every 5m</p>**Descriptions:**

- _precise-code-intel-worker: 20+ codeintel database errors every 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-worker_code_intel_codeintel_db_store_errors"
]
```

<br />
## precise-code-intel-worker: processing_uploads_reset

<p class="subtitle">code-intel: uploads reset to queued state every 5m</p>**Descriptions:**

- _precise-code-intel-worker: 20+ uploads reset to queued state every 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-worker_processing_uploads_reset"
]
```

<br />
## precise-code-intel-worker: processing_uploads_reset_failures

<p class="subtitle">code-intel: uploads errored after repeated resets every 5m</p>**Descriptions:**

- _precise-code-intel-worker: 20+ uploads errored after repeated resets every 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-worker_processing_uploads_reset_failures"
]
```

<br />
## precise-code-intel-worker: upload_resetter_errors

<p class="subtitle">code-intel: upload resetter errors every 5m</p>**Descriptions:**

- _precise-code-intel-worker: 20+ upload resetter errors every 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-worker_upload_resetter_errors"
]
```

<br />
## precise-code-intel-worker: 99th_percentile_bundle_manager_transfer_duration

<p class="subtitle">code-intel: 99th percentile successful bundle manager data transfer duration over 5m</p>**Descriptions:**

- _precise-code-intel-worker: 300s+ 99th percentile successful bundle manager data transfer duration over 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-worker_99th_percentile_bundle_manager_transfer_duration"
]
```

<br />
## precise-code-intel-worker: bundle_manager_error_responses

<p class="subtitle">code-intel: bundle manager error responses every 5m</p>**Descriptions:**

- _precise-code-intel-worker: 5+ bundle manager error responses every 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-worker_bundle_manager_error_responses"
]
```

<br />
## precise-code-intel-worker: 99th_percentile_gitserver_duration

<p class="subtitle">code-intel: 99th percentile successful gitserver query duration over 5m</p>**Descriptions:**

- _precise-code-intel-worker: 20s+ 99th percentile successful gitserver query duration over 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-worker_99th_percentile_gitserver_duration"
]
```

<br />
## precise-code-intel-worker: gitserver_error_responses

<p class="subtitle">code-intel: gitserver error responses every 5m</p>**Descriptions:**

- _precise-code-intel-worker: 5%+ gitserver error responses every 5m for 15m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-worker_gitserver_error_responses"
]
```

<br />
## precise-code-intel-worker: frontend_internal_api_error_responses

<p class="subtitle">code-intel: frontend-internal API error responses every 5m by route</p>**Descriptions:**

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

<br />
## precise-code-intel-worker: container_cpu_usage

<p class="subtitle">code-intel: container cpu usage total (1m average) across all cores by instance</p>**Descriptions:**

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

<br />
## precise-code-intel-worker: container_memory_usage

<p class="subtitle">code-intel: container memory usage by instance</p>**Descriptions:**

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

<br />
## precise-code-intel-worker: container_restarts

<p class="subtitle">code-intel: container restarts every 5m by instance</p>**Descriptions:**

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

<br />
## precise-code-intel-worker: fs_inodes_used

<p class="subtitle">code-intel: fs inodes in use by instance</p>**Descriptions:**

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

<br />
## precise-code-intel-worker: provisioning_container_cpu_usage_long_term

<p class="subtitle">code-intel: container cpu usage total (90th percentile over 1d) across all cores by instance</p>**Descriptions:**

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

<br />
## precise-code-intel-worker: provisioning_container_memory_usage_long_term

<p class="subtitle">code-intel: container memory usage (1d maximum) by instance</p>**Descriptions:**

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

<br />
## precise-code-intel-worker: provisioning_container_cpu_usage_short_term

<p class="subtitle">code-intel: container cpu usage total (5m maximum) across all cores by instance</p>**Descriptions:**

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

<br />
## precise-code-intel-worker: provisioning_container_memory_usage_short_term

<p class="subtitle">code-intel: container memory usage (5m maximum) by instance</p>**Descriptions:**

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

<br />
## precise-code-intel-worker: go_goroutines

<p class="subtitle">code-intel: maximum active goroutines</p>**Descriptions:**

- _precise-code-intel-worker: 10000+ maximum active goroutines for 10m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-worker_go_goroutines"
]
```

<br />
## precise-code-intel-worker: go_gc_duration_seconds

<p class="subtitle">code-intel: maximum go garbage collection duration</p>**Descriptions:**

- _precise-code-intel-worker: 2s+ maximum go garbage collection duration_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-worker_go_gc_duration_seconds"
]
```

<br />
## precise-code-intel-worker: pods_available_percentage

<p class="subtitle">code-intel: percentage pods available</p>**Descriptions:**

- _precise-code-intel-worker: less than 90% percentage pods available for 10m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "critical_precise-code-intel-worker_pods_available_percentage"
]
```

<br />
## precise-code-intel-indexer: index_queue_size

<p class="subtitle">code-intel: index queue size</p>**Descriptions:**

- _precise-code-intel-indexer: 100+ index queue size_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-indexer_index_queue_size"
]
```

<br />
## precise-code-intel-indexer: index_queue_growth_rate

<p class="subtitle">code-intel: index queue growth rate every 5m</p>**Descriptions:**

- _precise-code-intel-indexer: 5+ index queue growth rate every 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-indexer_index_queue_growth_rate"
]
```

<br />
## precise-code-intel-indexer: index_process_errors

<p class="subtitle">code-intel: index process errors every 5m</p>**Descriptions:**

- _precise-code-intel-indexer: 20+ index process errors every 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-indexer_index_process_errors"
]
```

<br />
## precise-code-intel-indexer: code_intel_frontend_db_store_99th_percentile_duration

<p class="subtitle">code-intel: 99th percentile successful frontend database query duration over 5m</p>**Descriptions:**

- _precise-code-intel-indexer: 20s+ 99th percentile successful frontend database query duration over 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-indexer_code_intel_frontend_db_store_99th_percentile_duration"
]
```

<br />
## precise-code-intel-indexer: code_intel_frontend_db_store_errors

<p class="subtitle">code-intel: frontend database errors every 5m</p>**Descriptions:**

- _precise-code-intel-indexer: 20+ frontend database errors every 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-indexer_code_intel_frontend_db_store_errors"
]
```

<br />
## precise-code-intel-indexer: indexability_updater_errors

<p class="subtitle">code-intel: indexability updater errors every 5m</p>**Descriptions:**

- _precise-code-intel-indexer: 20+ indexability updater errors every 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-indexer_indexability_updater_errors"
]
```

<br />
## precise-code-intel-indexer: index_scheduler_errors

<p class="subtitle">code-intel: index scheduler errors every 5m</p>**Descriptions:**

- _precise-code-intel-indexer: 20+ index scheduler errors every 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-indexer_index_scheduler_errors"
]
```

<br />
## precise-code-intel-indexer: processing_indexes_reset

<p class="subtitle">code-intel: indexes reset to queued state every 5m</p>**Descriptions:**

- _precise-code-intel-indexer: 20+ indexes reset to queued state every 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-indexer_processing_indexes_reset"
]
```

<br />
## precise-code-intel-indexer: processing_indexes_reset_failures

<p class="subtitle">code-intel: indexes errored after repeated resets every 5m</p>**Descriptions:**

- _precise-code-intel-indexer: 20+ indexes errored after repeated resets every 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-indexer_processing_indexes_reset_failures"
]
```

<br />
## precise-code-intel-indexer: index_resetter_errors

<p class="subtitle">code-intel: index resetter errors every 5m</p>**Descriptions:**

- _precise-code-intel-indexer: 20+ index resetter errors every 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-indexer_index_resetter_errors"
]
```

<br />
## precise-code-intel-indexer: janitor_errors

<p class="subtitle">code-intel: janitor errors every 5m</p>**Descriptions:**

- _precise-code-intel-indexer: 20+ janitor errors every 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-indexer_janitor_errors"
]
```

<br />
## precise-code-intel-indexer: janitor_indexes_removed

<p class="subtitle">code-intel: index records removed every 5m</p>**Descriptions:**

- _precise-code-intel-indexer: 20+ index records removed every 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-indexer_janitor_indexes_removed"
]
```

<br />
## precise-code-intel-indexer: 99th_percentile_gitserver_duration

<p class="subtitle">code-intel: 99th percentile successful gitserver query duration over 5m</p>**Descriptions:**

- _precise-code-intel-indexer: 20s+ 99th percentile successful gitserver query duration over 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-indexer_99th_percentile_gitserver_duration"
]
```

<br />
## precise-code-intel-indexer: gitserver_error_responses

<p class="subtitle">code-intel: gitserver error responses every 5m</p>**Descriptions:**

- _precise-code-intel-indexer: 5%+ gitserver error responses every 5m for 15m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-indexer_gitserver_error_responses"
]
```

<br />
## precise-code-intel-indexer: frontend_internal_api_error_responses

<p class="subtitle">code-intel: frontend-internal API error responses every 5m by route</p>**Descriptions:**

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

<br />
## precise-code-intel-indexer: container_cpu_usage

<p class="subtitle">code-intel: container cpu usage total (1m average) across all cores by instance</p>**Descriptions:**

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

<br />
## precise-code-intel-indexer: container_memory_usage

<p class="subtitle">code-intel: container memory usage by instance</p>**Descriptions:**

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

<br />
## precise-code-intel-indexer: container_restarts

<p class="subtitle">code-intel: container restarts every 5m by instance</p>**Descriptions:**

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

<br />
## precise-code-intel-indexer: fs_inodes_used

<p class="subtitle">code-intel: fs inodes in use by instance</p>**Descriptions:**

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

<br />
## precise-code-intel-indexer: provisioning_container_cpu_usage_long_term

<p class="subtitle">code-intel: container cpu usage total (90th percentile over 1d) across all cores by instance</p>**Descriptions:**

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

<br />
## precise-code-intel-indexer: provisioning_container_memory_usage_long_term

<p class="subtitle">code-intel: container memory usage (1d maximum) by instance</p>**Descriptions:**

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

<br />
## precise-code-intel-indexer: provisioning_container_cpu_usage_short_term

<p class="subtitle">code-intel: container cpu usage total (5m maximum) across all cores by instance</p>**Descriptions:**

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

<br />
## precise-code-intel-indexer: provisioning_container_memory_usage_short_term

<p class="subtitle">code-intel: container memory usage (5m maximum) by instance</p>**Descriptions:**

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

<br />
## precise-code-intel-indexer: go_goroutines

<p class="subtitle">code-intel: maximum active goroutines</p>**Descriptions:**

- _precise-code-intel-indexer: 10000+ maximum active goroutines for 10m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-indexer_go_goroutines"
]
```

<br />
## precise-code-intel-indexer: go_gc_duration_seconds

<p class="subtitle">code-intel: maximum go garbage collection duration</p>**Descriptions:**

- _precise-code-intel-indexer: 2s+ maximum go garbage collection duration_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_precise-code-intel-indexer_go_gc_duration_seconds"
]
```

<br />
## precise-code-intel-indexer: pods_available_percentage

<p class="subtitle">code-intel: percentage pods available</p>**Descriptions:**

- _precise-code-intel-indexer: less than 90% percentage pods available for 10m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "critical_precise-code-intel-indexer_pods_available_percentage"
]
```

<br />
## query-runner: frontend_internal_api_error_responses

<p class="subtitle">search: frontend-internal API error responses every 5m by route</p>**Descriptions:**

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

<br />
## query-runner: container_memory_usage

<p class="subtitle">search: container memory usage by instance</p>**Descriptions:**

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

<br />
## query-runner: container_cpu_usage

<p class="subtitle">search: container cpu usage total (1m average) across all cores by instance</p>**Descriptions:**

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

<br />
## query-runner: container_restarts

<p class="subtitle">search: container restarts every 5m by instance</p>**Descriptions:**

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

<br />
## query-runner: fs_inodes_used

<p class="subtitle">search: fs inodes in use by instance</p>**Descriptions:**

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

<br />
## query-runner: provisioning_container_cpu_usage_long_term

<p class="subtitle">search: container cpu usage total (90th percentile over 1d) across all cores by instance</p>**Descriptions:**

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

<br />
## query-runner: provisioning_container_memory_usage_long_term

<p class="subtitle">search: container memory usage (1d maximum) by instance</p>**Descriptions:**

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

<br />
## query-runner: provisioning_container_cpu_usage_short_term

<p class="subtitle">search: container cpu usage total (5m maximum) across all cores by instance</p>**Descriptions:**

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

<br />
## query-runner: provisioning_container_memory_usage_short_term

<p class="subtitle">search: container memory usage (5m maximum) by instance</p>**Descriptions:**

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

<br />
## query-runner: go_goroutines

<p class="subtitle">search: maximum active goroutines</p>**Descriptions:**

- _query-runner: 10000+ maximum active goroutines for 10m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_query-runner_go_goroutines"
]
```

<br />
## query-runner: go_gc_duration_seconds

<p class="subtitle">search: maximum go garbage collection duration</p>**Descriptions:**

- _query-runner: 2s+ maximum go garbage collection duration_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_query-runner_go_gc_duration_seconds"
]
```

<br />
## query-runner: pods_available_percentage

<p class="subtitle">search: percentage pods available</p>**Descriptions:**

- _query-runner: less than 90% percentage pods available for 10m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "critical_query-runner_pods_available_percentage"
]
```

<br />
## repo-updater: frontend_internal_api_error_responses

<p class="subtitle">cloud: frontend-internal API error responses every 5m by route</p>**Descriptions:**

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

<br />
## repo-updater: syncer_sync_last_time

<p class="subtitle">cloud: time since last sync</p>**Descriptions:**

- _repo-updater: 3600s+ time since last sync for 5m0s_

**Possible solutions:**

- Make sure there are external services added with valid tokens
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_repo-updater_syncer_sync_last_time"
]
```

<br />
## repo-updater: src_repoupdater_max_sync_backoff

<p class="subtitle">cloud: time since oldest sync</p>**Descriptions:**

- _repo-updater: 32400s+ time since oldest sync for 10m0s_

**Possible solutions:**

- Make sure there are external services added with valid tokens
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "critical_repo-updater_src_repoupdater_max_sync_backoff"
]
```

<br />
## repo-updater: syncer_sync_start

<p class="subtitle">cloud: sync was started</p>**Descriptions:**

- _repo-updater: less than 0 sync was started for 9h0m0s_

**Possible solutions:**

- Check repo-updater logs for errors.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_repo-updater_syncer_sync_start"
]
```

<br />
## repo-updater: syncer_sync_duration

<p class="subtitle">cloud: 95th repositories sync duration</p>**Descriptions:**

- _repo-updater: 30s+ 95th repositories sync duration for 5m0s_

**Possible solutions:**

- Check the network latency is reasonable (<50ms) between the Sourcegraph and the code host
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_repo-updater_syncer_sync_duration"
]
```

<br />
## repo-updater: source_duration

<p class="subtitle">cloud: 95th repositories source duration</p>**Descriptions:**

- _repo-updater: 30s+ 95th repositories source duration for 5m0s_

**Possible solutions:**

- Check the network latency is reasonable (<50ms) between the Sourcegraph and the code host
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_repo-updater_source_duration"
]
```

<br />
## repo-updater: syncer_synced_repos

<p class="subtitle">cloud: repositories synced</p>**Descriptions:**

- _repo-updater: less than 0 repositories synced for 9h0m0s_

**Possible solutions:**

- Check network connectivity to code hosts
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_repo-updater_syncer_synced_repos"
]
```

<br />
## repo-updater: sourced_repos

<p class="subtitle">cloud: repositories sourced</p>**Descriptions:**

- _repo-updater: less than 0 repositories sourced for 9h0m0s_

**Possible solutions:**

- Check network connectivity to code hosts
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_repo-updater_sourced_repos"
]
```

<br />
## repo-updater: user_added_repos

<p class="subtitle">cloud: total number of user added repos</p>**Descriptions:**

- _repo-updater: 180000+ total number of user added repos for 5m0s_

**Possible solutions:**

- Check for unusual spikes in user added repos. Each user is only allowed to add 2000
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "critical_repo-updater_user_added_repos"
]
```

<br />
## repo-updater: purge_failed

<p class="subtitle">cloud: repositories purge failed</p>**Descriptions:**

- _repo-updater: 0+ repositories purge failed for 5m0s_

**Possible solutions:**

- Check repo-updater`s connectivity with gitserver and gitserver logs
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_repo-updater_purge_failed"
]
```

<br />
## repo-updater: sched_auto_fetch

<p class="subtitle">cloud: repositories scheduled due to hitting a deadline</p>**Descriptions:**

- _repo-updater: less than 0 repositories scheduled due to hitting a deadline for 9h0m0s_

**Possible solutions:**

- Check repo-updater logs. This is expected to fire if there are no user added code hosts
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_repo-updater_sched_auto_fetch"
]
```

<br />
## repo-updater: sched_manual_fetch

<p class="subtitle">cloud: repositories scheduled due to user traffic</p>**Descriptions:**

- _repo-updater: less than 0 repositories scheduled due to user traffic for 9h0m0s_

**Possible solutions:**

- Check repo-updater logs. This is expected to fire if there are no user added code hosts
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_repo-updater_sched_manual_fetch"
]
```

<br />
## repo-updater: sched_known_repos

<p class="subtitle">cloud: repositories managed by the scheduler</p>**Descriptions:**

- _repo-updater: less than 0 repositories managed by the scheduler for 10m0s_

**Possible solutions:**

- Check repo-updater logs. This is expected to fire if there are no user added code hosts
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_repo-updater_sched_known_repos"
]
```

<br />
## repo-updater: sched_update_queue_length

<p class="subtitle">cloud: rate of growth of update queue length over 5 minutes</p>**Descriptions:**

- _repo-updater: 0+ rate of growth of update queue length over 5 minutes for 30m0s_

**Possible solutions:**

- Check repo-updater logs for indications that the queue is not being processed. The queue length should trend downwards over time as items are sent to GitServer
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "critical_repo-updater_sched_update_queue_length"
]
```

<br />
## repo-updater: sched_loops

<p class="subtitle">cloud: scheduler loops</p>**Descriptions:**

- _repo-updater: less than 0 scheduler loops for 9h0m0s_

**Possible solutions:**

- Check repo-updater logs for errors. This is expected to fire if there are no user added code hosts
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_repo-updater_sched_loops"
]
```

<br />
## repo-updater: sched_error

<p class="subtitle">cloud: repositories schedule error rate</p>**Descriptions:**

- _repo-updater: 1+ repositories schedule error rate for 1m0s_

**Possible solutions:**

- Check repo-updater logs for errors
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "critical_repo-updater_sched_error"
]
```

<br />
## repo-updater: perms_syncer_perms

<p class="subtitle">cloud: time gap between least and most up to date permissions</p>**Descriptions:**

- _repo-updater: 259200s+ time gap between least and most up to date permissions for 5m0s_

**Possible solutions:**

- Increase the API rate limit to [GitHub](https://docs.sourcegraph.com/admin/external_service/github#github-com-rate-limits), [GitLab](https://docs.sourcegraph.com/admin/external_service/gitlab#internal-rate-limits) or [Bitbucket Server](https://docs.sourcegraph.com/admin/external_service/bitbucket_server#internal-rate-limits).
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_repo-updater_perms_syncer_perms"
]
```

<br />
## repo-updater: perms_syncer_stale_perms

<p class="subtitle">cloud: number of entities with stale permissions</p>**Descriptions:**

- _repo-updater: 100+ number of entities with stale permissions for 5m0s_

**Possible solutions:**

- Increase the API rate limit to [GitHub](https://docs.sourcegraph.com/admin/external_service/github#github-com-rate-limits), [GitLab](https://docs.sourcegraph.com/admin/external_service/gitlab#internal-rate-limits) or [Bitbucket Server](https://docs.sourcegraph.com/admin/external_service/bitbucket_server#internal-rate-limits).
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_repo-updater_perms_syncer_stale_perms"
]
```

<br />
## repo-updater: perms_syncer_no_perms

<p class="subtitle">cloud: number of entities with no permissions</p>**Descriptions:**

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

<br />
## repo-updater: perms_syncer_sync_duration

<p class="subtitle">cloud: 95th permissions sync duration</p>**Descriptions:**

- _repo-updater: 30s+ 95th permissions sync duration for 5m0s_

**Possible solutions:**

- Check the network latency is reasonable (<50ms) between the Sourcegraph and the code host.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_repo-updater_perms_syncer_sync_duration"
]
```

<br />
## repo-updater: perms_syncer_queue_size

<p class="subtitle">cloud: permissions sync queued items</p>**Descriptions:**

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

<br />
## repo-updater: authz_filter_duration

<p class="subtitle">cloud: 95th authorization duration</p>**Descriptions:**

- _repo-updater: 1s+ 95th authorization duration for 1m0s_

**Possible solutions:**

- Check if database is overloaded.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "critical_repo-updater_authz_filter_duration"
]
```

<br />
## repo-updater: perms_syncer_sync_errors

<p class="subtitle">cloud: permissions sync error rate</p>**Descriptions:**

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

<br />
## repo-updater: src_repoupdater_external_services_total

<p class="subtitle">cloud: the total number of external services</p>**Descriptions:**

- _repo-updater: 20000+ the total number of external services for 1h0m0s_

**Possible solutions:**

- Check for spikes in external services, could be abuse
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "critical_repo-updater_src_repoupdater_external_services_total"
]
```

<br />
## repo-updater: src_repoupdater_user_external_services_total

<p class="subtitle">cloud: the total number of user added external services</p>**Descriptions:**

- _repo-updater: 20000+ the total number of user added external services for 1h0m0s_

**Possible solutions:**

- Check for spikes in external services, could be abuse
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_repo-updater_src_repoupdater_user_external_services_total"
]
```

<br />
## repo-updater: repoupdater_queued_sync_jobs_total

<p class="subtitle">cloud: the total number of queued sync jobs</p>**Descriptions:**

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

<br />
## repo-updater: repoupdater_completed_sync_jobs_total

<p class="subtitle">cloud: the total number of completed sync jobs</p>**Descriptions:**

- _repo-updater: 100000+ the total number of completed sync jobs for 1h0m0s_

**Possible solutions:**

- Check repo-updater logs. Jobs older than 1 day should have been removed.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_repo-updater_repoupdater_completed_sync_jobs_total"
]
```

<br />
## repo-updater: repoupdater_errored_sync_jobs_total

<p class="subtitle">cloud: the total number of errored sync jobs</p>**Descriptions:**

- _repo-updater: 100+ the total number of errored sync jobs for 1h0m0s_

**Possible solutions:**

- Check repo-updater logs. Check code host connectivity
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_repo-updater_repoupdater_errored_sync_jobs_total"
]
```

<br />
## repo-updater: container_cpu_usage

<p class="subtitle">cloud: container cpu usage total (1m average) across all cores by instance</p>**Descriptions:**

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

<br />
## repo-updater: container_memory_usage

<p class="subtitle">cloud: container memory usage by instance</p>**Descriptions:**

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

<br />
## repo-updater: container_restarts

<p class="subtitle">cloud: container restarts every 5m by instance</p>**Descriptions:**

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

<br />
## repo-updater: fs_inodes_used

<p class="subtitle">cloud: fs inodes in use by instance</p>**Descriptions:**

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

<br />
## repo-updater: provisioning_container_cpu_usage_long_term

<p class="subtitle">cloud: container cpu usage total (90th percentile over 1d) across all cores by instance</p>**Descriptions:**

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

<br />
## repo-updater: provisioning_container_memory_usage_long_term

<p class="subtitle">cloud: container memory usage (1d maximum) by instance</p>**Descriptions:**

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

<br />
## repo-updater: provisioning_container_cpu_usage_short_term

<p class="subtitle">cloud: container cpu usage total (5m maximum) across all cores by instance</p>**Descriptions:**

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

<br />
## repo-updater: provisioning_container_memory_usage_short_term

<p class="subtitle">cloud: container memory usage (5m maximum) by instance</p>**Descriptions:**

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

<br />
## repo-updater: go_goroutines

<p class="subtitle">cloud: maximum active goroutines</p>**Descriptions:**

- _repo-updater: 10000+ maximum active goroutines for 10m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_repo-updater_go_goroutines"
]
```

<br />
## repo-updater: go_gc_duration_seconds

<p class="subtitle">cloud: maximum go garbage collection duration</p>**Descriptions:**

- _repo-updater: 2s+ maximum go garbage collection duration_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_repo-updater_go_gc_duration_seconds"
]
```

<br />
## repo-updater: pods_available_percentage

<p class="subtitle">cloud: percentage pods available</p>**Descriptions:**

- _repo-updater: less than 90% percentage pods available for 10m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "critical_repo-updater_pods_available_percentage"
]
```

<br />
## searcher: unindexed_search_request_errors

<p class="subtitle">search: unindexed search request errors every 5m by code</p>**Descriptions:**

- _searcher: 5%+ unindexed search request errors every 5m by code for 5m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_searcher_unindexed_search_request_errors"
]
```

<br />
## searcher: replica_traffic

<p class="subtitle">search: requests per second over 10m</p>**Descriptions:**

- _searcher: 5+ requests per second over 10m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_searcher_replica_traffic"
]
```

<br />
## searcher: frontend_internal_api_error_responses

<p class="subtitle">search: frontend-internal API error responses every 5m by route</p>**Descriptions:**

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

<br />
## searcher: container_cpu_usage

<p class="subtitle">search: container cpu usage total (1m average) across all cores by instance</p>**Descriptions:**

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

<br />
## searcher: container_memory_usage

<p class="subtitle">search: container memory usage by instance</p>**Descriptions:**

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

<br />
## searcher: container_restarts

<p class="subtitle">search: container restarts every 5m by instance</p>**Descriptions:**

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

<br />
## searcher: fs_inodes_used

<p class="subtitle">search: fs inodes in use by instance</p>**Descriptions:**

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

<br />
## searcher: provisioning_container_cpu_usage_long_term

<p class="subtitle">search: container cpu usage total (90th percentile over 1d) across all cores by instance</p>**Descriptions:**

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

<br />
## searcher: provisioning_container_memory_usage_long_term

<p class="subtitle">search: container memory usage (1d maximum) by instance</p>**Descriptions:**

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

<br />
## searcher: provisioning_container_cpu_usage_short_term

<p class="subtitle">search: container cpu usage total (5m maximum) across all cores by instance</p>**Descriptions:**

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

<br />
## searcher: provisioning_container_memory_usage_short_term

<p class="subtitle">search: container memory usage (5m maximum) by instance</p>**Descriptions:**

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

<br />
## searcher: go_goroutines

<p class="subtitle">search: maximum active goroutines</p>**Descriptions:**

- _searcher: 10000+ maximum active goroutines for 10m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_searcher_go_goroutines"
]
```

<br />
## searcher: go_gc_duration_seconds

<p class="subtitle">search: maximum go garbage collection duration</p>**Descriptions:**

- _searcher: 2s+ maximum go garbage collection duration_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_searcher_go_gc_duration_seconds"
]
```

<br />
## searcher: pods_available_percentage

<p class="subtitle">search: percentage pods available</p>**Descriptions:**

- _searcher: less than 90% percentage pods available for 10m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "critical_searcher_pods_available_percentage"
]
```

<br />
## symbols: store_fetch_failures

<p class="subtitle">code-intel: store fetch failures every 5m</p>**Descriptions:**

- _symbols: 5+ store fetch failures every 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_symbols_store_fetch_failures"
]
```

<br />
## symbols: current_fetch_queue_size

<p class="subtitle">code-intel: current fetch queue size</p>**Descriptions:**

- _symbols: 25+ current fetch queue size_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_symbols_current_fetch_queue_size"
]
```

<br />
## symbols: frontend_internal_api_error_responses

<p class="subtitle">code-intel: frontend-internal API error responses every 5m by route</p>**Descriptions:**

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

<br />
## symbols: container_cpu_usage

<p class="subtitle">code-intel: container cpu usage total (1m average) across all cores by instance</p>**Descriptions:**

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

<br />
## symbols: container_memory_usage

<p class="subtitle">code-intel: container memory usage by instance</p>**Descriptions:**

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

<br />
## symbols: container_restarts

<p class="subtitle">code-intel: container restarts every 5m by instance</p>**Descriptions:**

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

<br />
## symbols: fs_inodes_used

<p class="subtitle">code-intel: fs inodes in use by instance</p>**Descriptions:**

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

<br />
## symbols: provisioning_container_cpu_usage_long_term

<p class="subtitle">code-intel: container cpu usage total (90th percentile over 1d) across all cores by instance</p>**Descriptions:**

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

<br />
## symbols: provisioning_container_memory_usage_long_term

<p class="subtitle">code-intel: container memory usage (1d maximum) by instance</p>**Descriptions:**

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

<br />
## symbols: provisioning_container_cpu_usage_short_term

<p class="subtitle">code-intel: container cpu usage total (5m maximum) across all cores by instance</p>**Descriptions:**

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

<br />
## symbols: provisioning_container_memory_usage_short_term

<p class="subtitle">code-intel: container memory usage (5m maximum) by instance</p>**Descriptions:**

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

<br />
## symbols: go_goroutines

<p class="subtitle">code-intel: maximum active goroutines</p>**Descriptions:**

- _symbols: 10000+ maximum active goroutines for 10m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_symbols_go_goroutines"
]
```

<br />
## symbols: go_gc_duration_seconds

<p class="subtitle">code-intel: maximum go garbage collection duration</p>**Descriptions:**

- _symbols: 2s+ maximum go garbage collection duration_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_symbols_go_gc_duration_seconds"
]
```

<br />
## symbols: pods_available_percentage

<p class="subtitle">code-intel: percentage pods available</p>**Descriptions:**

- _symbols: less than 90% percentage pods available for 10m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "critical_symbols_pods_available_percentage"
]
```

<br />
## syntect-server: syntax_highlighting_errors

<p class="subtitle">code-intel: syntax highlighting errors every 5m</p>**Descriptions:**

- _syntect-server: 5%+ syntax highlighting errors every 5m for 5m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_syntect-server_syntax_highlighting_errors"
]
```

<br />
## syntect-server: syntax_highlighting_timeouts

<p class="subtitle">code-intel: syntax highlighting timeouts every 5m</p>**Descriptions:**

- _syntect-server: 5%+ syntax highlighting timeouts every 5m for 5m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_syntect-server_syntax_highlighting_timeouts"
]
```

<br />
## syntect-server: syntax_highlighting_panics

<p class="subtitle">code-intel: syntax highlighting panics every 5m</p>**Descriptions:**

- _syntect-server: 5+ syntax highlighting panics every 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_syntect-server_syntax_highlighting_panics"
]
```

<br />
## syntect-server: syntax_highlighting_worker_deaths

<p class="subtitle">code-intel: syntax highlighter worker deaths every 5m</p>**Descriptions:**

- _syntect-server: 1+ syntax highlighter worker deaths every 5m_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_syntect-server_syntax_highlighting_worker_deaths"
]
```

<br />
## syntect-server: container_cpu_usage

<p class="subtitle">code-intel: container cpu usage total (1m average) across all cores by instance</p>**Descriptions:**

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

<br />
## syntect-server: container_memory_usage

<p class="subtitle">code-intel: container memory usage by instance</p>**Descriptions:**

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

<br />
## syntect-server: container_restarts

<p class="subtitle">code-intel: container restarts every 5m by instance</p>**Descriptions:**

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

<br />
## syntect-server: fs_inodes_used

<p class="subtitle">code-intel: fs inodes in use by instance</p>**Descriptions:**

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

<br />
## syntect-server: provisioning_container_cpu_usage_long_term

<p class="subtitle">code-intel: container cpu usage total (90th percentile over 1d) across all cores by instance</p>**Descriptions:**

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

<br />
## syntect-server: provisioning_container_memory_usage_long_term

<p class="subtitle">code-intel: container memory usage (1d maximum) by instance</p>**Descriptions:**

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

<br />
## syntect-server: provisioning_container_cpu_usage_short_term

<p class="subtitle">code-intel: container cpu usage total (5m maximum) across all cores by instance</p>**Descriptions:**

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

<br />
## syntect-server: provisioning_container_memory_usage_short_term

<p class="subtitle">code-intel: container memory usage (5m maximum) by instance</p>**Descriptions:**

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

<br />
## syntect-server: pods_available_percentage

<p class="subtitle">code-intel: percentage pods available</p>**Descriptions:**

- _syntect-server: less than 90% percentage pods available for 10m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "critical_syntect-server_pods_available_percentage"
]
```

<br />
## zoekt-indexserver: average_resolve_revision_duration

<p class="subtitle">search: average resolve revision duration over 5m</p>**Descriptions:**

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

<br />
## zoekt-indexserver: container_cpu_usage

<p class="subtitle">search: container cpu usage total (1m average) across all cores by instance</p>**Descriptions:**

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

<br />
## zoekt-indexserver: container_memory_usage

<p class="subtitle">search: container memory usage by instance</p>**Descriptions:**

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

<br />
## zoekt-indexserver: container_restarts

<p class="subtitle">search: container restarts every 5m by instance</p>**Descriptions:**

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

<br />
## zoekt-indexserver: fs_inodes_used

<p class="subtitle">search: fs inodes in use by instance</p>**Descriptions:**

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

<br />
## zoekt-indexserver: fs_io_operations

<p class="subtitle">search: filesystem reads and writes rate by instance over 1h</p>**Descriptions:**

- _zoekt-indexserver: 5000+ filesystem reads and writes rate by instance over 1h_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_zoekt-indexserver_fs_io_operations"
]
```

<br />
## zoekt-indexserver: provisioning_container_cpu_usage_long_term

<p class="subtitle">search: container cpu usage total (90th percentile over 1d) across all cores by instance</p>**Descriptions:**

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

<br />
## zoekt-indexserver: provisioning_container_memory_usage_long_term

<p class="subtitle">search: container memory usage (1d maximum) by instance</p>**Descriptions:**

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

<br />
## zoekt-indexserver: provisioning_container_cpu_usage_short_term

<p class="subtitle">search: container cpu usage total (5m maximum) across all cores by instance</p>**Descriptions:**

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

<br />
## zoekt-indexserver: provisioning_container_memory_usage_short_term

<p class="subtitle">search: container memory usage (5m maximum) by instance</p>**Descriptions:**

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

<br />
## zoekt-indexserver: pods_available_percentage

<p class="subtitle">search: percentage pods available</p>**Descriptions:**

- _zoekt-indexserver: less than 90% percentage pods available for 10m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "critical_zoekt-indexserver_pods_available_percentage"
]
```

<br />
## zoekt-webserver: indexed_search_request_errors

<p class="subtitle">search: indexed search request errors every 5m by code</p>**Descriptions:**

- _zoekt-webserver: 5%+ indexed search request errors every 5m by code for 5m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_zoekt-webserver_indexed_search_request_errors"
]
```

<br />
## zoekt-webserver: container_cpu_usage

<p class="subtitle">search: container cpu usage total (1m average) across all cores by instance</p>**Descriptions:**

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

<br />
## zoekt-webserver: container_memory_usage

<p class="subtitle">search: container memory usage by instance</p>**Descriptions:**

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

<br />
## zoekt-webserver: container_restarts

<p class="subtitle">search: container restarts every 5m by instance</p>**Descriptions:**

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

<br />
## zoekt-webserver: fs_inodes_used

<p class="subtitle">search: fs inodes in use by instance</p>**Descriptions:**

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

<br />
## zoekt-webserver: fs_io_operations

<p class="subtitle">search: filesystem reads and writes by instance rate over 1h</p>**Descriptions:**

- _zoekt-webserver: 5000+ filesystem reads and writes by instance rate over 1h_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_zoekt-webserver_fs_io_operations"
]
```

<br />
## zoekt-webserver: provisioning_container_cpu_usage_long_term

<p class="subtitle">search: container cpu usage total (90th percentile over 1d) across all cores by instance</p>**Descriptions:**

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

<br />
## zoekt-webserver: provisioning_container_memory_usage_long_term

<p class="subtitle">search: container memory usage (1d maximum) by instance</p>**Descriptions:**

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

<br />
## zoekt-webserver: provisioning_container_cpu_usage_short_term

<p class="subtitle">search: container cpu usage total (5m maximum) across all cores by instance</p>**Descriptions:**

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

<br />
## zoekt-webserver: provisioning_container_memory_usage_short_term

<p class="subtitle">search: container memory usage (5m maximum) by instance</p>**Descriptions:**

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

<br />
## prometheus: prometheus_metrics_bloat

<p class="subtitle">distribution: prometheus metrics payload size</p>**Descriptions:**

- _prometheus: 20000B+ prometheus metrics payload size_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_prometheus_prometheus_metrics_bloat"
]
```

<br />
## prometheus: alertmanager_notifications_failed_total

<p class="subtitle">distribution: failed alertmanager notifications over 1m</p>**Descriptions:**

- _prometheus: 1+ failed alertmanager notifications over 1m_

**Possible solutions:**

- Ensure that your [`observability.alerts` configuration](https://docs.sourcegraph.com/admin/observability/alerting#setting-up-alerting) (in site configuration) is valid.
- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "warning_prometheus_alertmanager_notifications_failed_total"
]
```

<br />
## prometheus: container_cpu_usage

<p class="subtitle">distribution: container cpu usage total (1m average) across all cores by instance</p>**Descriptions:**

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

<br />
## prometheus: container_memory_usage

<p class="subtitle">distribution: container memory usage by instance</p>**Descriptions:**

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

<br />
## prometheus: container_restarts

<p class="subtitle">distribution: container restarts every 5m by instance</p>**Descriptions:**

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

<br />
## prometheus: fs_inodes_used

<p class="subtitle">distribution: fs inodes in use by instance</p>**Descriptions:**

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

<br />
## prometheus: provisioning_container_cpu_usage_long_term

<p class="subtitle">distribution: container cpu usage total (90th percentile over 1d) across all cores by instance</p>**Descriptions:**

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

<br />
## prometheus: provisioning_container_memory_usage_long_term

<p class="subtitle">distribution: container memory usage (1d maximum) by instance</p>**Descriptions:**

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

<br />
## prometheus: provisioning_container_cpu_usage_short_term

<p class="subtitle">distribution: container cpu usage total (5m maximum) across all cores by instance</p>**Descriptions:**

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

<br />
## prometheus: provisioning_container_memory_usage_short_term

<p class="subtitle">distribution: container memory usage (5m maximum) by instance</p>**Descriptions:**

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

<br />
## prometheus: pods_available_percentage

<p class="subtitle">distribution: percentage pods available</p>**Descriptions:**

- _prometheus: less than 90% percentage pods available for 10m0s_

**Possible solutions:**

- **Silence this alert:** If you are aware of this alert and want to silence notifications for it, add the following to your site configuration and set a reminder to re-evaluate the alert:

```json
"observability.silenceAlerts": [
  "critical_prometheus_pods_available_percentage"
]
```

<br />
