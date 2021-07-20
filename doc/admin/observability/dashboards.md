# Dashboards reference

<!-- DO NOT EDIT: generated via: go generate ./monitoring -->

This document contains a complete reference on Sourcegraph's available dashboards, as well as details on how to interpret the panels and metrics.

To learn more about Sourcegraph's metrics and how to view these dashboards, see [our metrics guide](https://docs.sourcegraph.com/admin/observability/metrics).

## Frontend

<p class="subtitle">Serves all end-user browser and API requests.</p>

### Frontend: Search at a glance

#### frontend: 99th_percentile_search_request_duration

This panel indicates 99th percentile successful search request duration over 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#frontend-99th-percentile-search-request-duration).

<sub>*Managed by the [Sourcegraph Search team](https://about.sourcegraph.com/handbook/engineering/search).*</sub>

<br />

#### frontend: 90th_percentile_search_request_duration

This panel indicates 90th percentile successful search request duration over 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#frontend-90th-percentile-search-request-duration).

<sub>*Managed by the [Sourcegraph Search team](https://about.sourcegraph.com/handbook/engineering/search).*</sub>

<br />

#### frontend: hard_timeout_search_responses

This panel indicates hard timeout search responses every 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#frontend-hard-timeout-search-responses).

<sub>*Managed by the [Sourcegraph Search team](https://about.sourcegraph.com/handbook/engineering/search).*</sub>

<br />

#### frontend: hard_error_search_responses

This panel indicates hard error search responses every 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#frontend-hard-error-search-responses).

<sub>*Managed by the [Sourcegraph Search team](https://about.sourcegraph.com/handbook/engineering/search).*</sub>

<br />

#### frontend: partial_timeout_search_responses

This panel indicates partial timeout search responses every 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#frontend-partial-timeout-search-responses).

<sub>*Managed by the [Sourcegraph Search team](https://about.sourcegraph.com/handbook/engineering/search).*</sub>

<br />

#### frontend: search_alert_user_suggestions

This panel indicates search alert user suggestions shown every 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#frontend-search-alert-user-suggestions).

<sub>*Managed by the [Sourcegraph Search team](https://about.sourcegraph.com/handbook/engineering/search).*</sub>

<br />

#### frontend: page_load_latency

This panel indicates 90th percentile page load latency over all routes over 10m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#frontend-page-load-latency).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### frontend: blob_load_latency

This panel indicates 90th percentile blob load latency over 10m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#frontend-blob-load-latency).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

### Frontend: Search-based code intelligence at a glance

#### frontend: 99th_percentile_search_codeintel_request_duration

This panel indicates 99th percentile code-intel successful search request duration over 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#frontend-99th-percentile-search-codeintel-request-duration).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### frontend: 90th_percentile_search_codeintel_request_duration

This panel indicates 90th percentile code-intel successful search request duration over 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#frontend-90th-percentile-search-codeintel-request-duration).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### frontend: hard_timeout_search_codeintel_responses

This panel indicates hard timeout search code-intel responses every 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#frontend-hard-timeout-search-codeintel-responses).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### frontend: hard_error_search_codeintel_responses

This panel indicates hard error search code-intel responses every 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#frontend-hard-error-search-codeintel-responses).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### frontend: partial_timeout_search_codeintel_responses

This panel indicates partial timeout search code-intel responses every 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#frontend-partial-timeout-search-codeintel-responses).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### frontend: search_codeintel_alert_user_suggestions

This panel indicates search code-intel alert user suggestions shown every 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#frontend-search-codeintel-alert-user-suggestions).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

### Frontend: Search API usage at a glance

#### frontend: 99th_percentile_search_api_request_duration

This panel indicates 99th percentile successful search API request duration over 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#frontend-99th-percentile-search-api-request-duration).

<sub>*Managed by the [Sourcegraph Search team](https://about.sourcegraph.com/handbook/engineering/search).*</sub>

<br />

#### frontend: 90th_percentile_search_api_request_duration

This panel indicates 90th percentile successful search API request duration over 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#frontend-90th-percentile-search-api-request-duration).

<sub>*Managed by the [Sourcegraph Search team](https://about.sourcegraph.com/handbook/engineering/search).*</sub>

<br />

#### frontend: hard_timeout_search_api_responses

This panel indicates hard timeout search API responses every 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#frontend-hard-timeout-search-api-responses).

<sub>*Managed by the [Sourcegraph Search team](https://about.sourcegraph.com/handbook/engineering/search).*</sub>

<br />

#### frontend: hard_error_search_api_responses

This panel indicates hard error search API responses every 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#frontend-hard-error-search-api-responses).

<sub>*Managed by the [Sourcegraph Search team](https://about.sourcegraph.com/handbook/engineering/search).*</sub>

<br />

#### frontend: partial_timeout_search_api_responses

This panel indicates partial timeout search API responses every 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#frontend-partial-timeout-search-api-responses).

<sub>*Managed by the [Sourcegraph Search team](https://about.sourcegraph.com/handbook/engineering/search).*</sub>

<br />

#### frontend: search_api_alert_user_suggestions

This panel indicates search API alert user suggestions shown every 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#frontend-search-api-alert-user-suggestions).

<sub>*Managed by the [Sourcegraph Search team](https://about.sourcegraph.com/handbook/engineering/search).*</sub>

<br />

### Frontend: Precise code intelligence usage at a glance

#### frontend: codeintel_resolvers_99th_percentile_duration

This panel indicates 99th percentile successful resolver duration over 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#frontend-codeintel-resolvers-99th-percentile-duration).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### frontend: codeintel_resolvers_errors

This panel indicates resolver errors every 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#frontend-codeintel-resolvers-errors).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

### Frontend: Precise code intelligence stores and clients

#### frontend: codeintel_dbstore_99th_percentile_duration

This panel indicates 99th percentile successful database store operation duration over 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#frontend-codeintel-dbstore-99th-percentile-duration).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### frontend: codeintel_dbstore_errors

This panel indicates database store errors every 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#frontend-codeintel-dbstore-errors).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### frontend: codeintel_upload_workerstore_99th_percentile_duration

This panel indicates 99th percentile successful upload worker store operation duration over 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#frontend-codeintel-upload-workerstore-99th-percentile-duration).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### frontend: codeintel_upload_workerstore_errors

This panel indicates upload worker store errors every 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#frontend-codeintel-upload-workerstore-errors).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### frontend: codeintel_index_workerstore_99th_percentile_duration

This panel indicates 99th percentile successful index worker store operation duration over 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#frontend-codeintel-index-workerstore-99th-percentile-duration).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### frontend: codeintel_index_workerstore_errors

This panel indicates index worker store errors every 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#frontend-codeintel-index-workerstore-errors).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### frontend: codeintel_lsifstore_99th_percentile_duration

This panel indicates 99th percentile successful LSIF store operation duration over 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#frontend-codeintel-lsifstore-99th-percentile-duration).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### frontend: codeintel_lsifstore_errors

This panel indicates lSIF store errors every 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#frontend-codeintel-lsifstore-errors).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### frontend: codeintel_uploadstore_99th_percentile_duration

This panel indicates 99th percentile successful upload store operation duration over 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#frontend-codeintel-uploadstore-99th-percentile-duration).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### frontend: codeintel_uploadstore_errors

This panel indicates upload store errors every 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#frontend-codeintel-uploadstore-errors).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### frontend: codeintel_gitserverclient_99th_percentile_duration

This panel indicates 99th percentile successful gitserver client operation duration over 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#frontend-codeintel-gitserverclient-99th-percentile-duration).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### frontend: codeintel_gitserverclient_errors

This panel indicates gitserver client errors every 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#frontend-codeintel-gitserverclient-errors).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

### Frontend: Out of band migrations

#### frontend: out_of_band_migrations_up_99th_percentile_duration

This panel indicates 99th percentile successful out-of-band up migration invocation (single batch processed) duration over 5m.

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### frontend: out_of_band_migrations_up_errors

This panel indicates out-of-band up migration errors every 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#frontend-out-of-band-migrations-up-errors).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### frontend: out_of_band_migrations_down_99th_percentile_duration

This panel indicates 99th percentile successful out-of-band down migration invocation (single batch processed) duration over 5m.

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### frontend: out_of_band_migrations_down_errors

This panel indicates out-of-band down migration errors every 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#frontend-out-of-band-migrations-down-errors).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

### Frontend: Database connections

#### frontend: max_open_conns

This panel indicates maximum open.

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### frontend: open_conns

This panel indicates established.

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### frontend: in_use

This panel indicates used.

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### frontend: idle

This panel indicates idle.

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### frontend: mean_blocked_seconds_per_conn_request

This panel indicates mean blocked seconds per conn request.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#frontend-mean-blocked-seconds-per-conn-request).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### frontend: closed_max_idle

This panel indicates closed by SetMaxIdleConns.

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### frontend: closed_max_lifetime

This panel indicates closed by SetConnMaxLifetime.

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### frontend: closed_max_idle_time

This panel indicates closed by SetConnMaxIdleTime.

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

### Frontend: Internal service requests

#### frontend: internal_indexed_search_error_responses

This panel indicates internal indexed search error responses every 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#frontend-internal-indexed-search-error-responses).

<sub>*Managed by the [Sourcegraph Search team](https://about.sourcegraph.com/handbook/engineering/search).*</sub>

<br />

#### frontend: internal_unindexed_search_error_responses

This panel indicates internal unindexed search error responses every 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#frontend-internal-unindexed-search-error-responses).

<sub>*Managed by the [Sourcegraph Search team](https://about.sourcegraph.com/handbook/engineering/search).*</sub>

<br />

#### frontend: internal_api_error_responses

This panel indicates internal API error responses every 5m by route.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#frontend-internal-api-error-responses).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### frontend: 99th_percentile_gitserver_duration

This panel indicates 99th percentile successful gitserver query duration over 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#frontend-99th-percentile-gitserver-duration).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### frontend: gitserver_error_responses

This panel indicates gitserver error responses every 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#frontend-gitserver-error-responses).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### frontend: observability_test_alert_warning

This panel indicates warning test alert metric.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#frontend-observability-test-alert-warning).

<sub>*Managed by the [Sourcegraph Distribution team](https://about.sourcegraph.com/handbook/engineering/distribution).*</sub>

<br />

#### frontend: observability_test_alert_critical

This panel indicates critical test alert metric.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#frontend-observability-test-alert-critical).

<sub>*Managed by the [Sourcegraph Distribution team](https://about.sourcegraph.com/handbook/engineering/distribution).*</sub>

<br />

### Frontend: Container monitoring (not available on server)

#### frontend: container_cpu_usage

This panel indicates container cpu usage total (1m average) across all cores by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#frontend-container-cpu-usage).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### frontend: container_memory_usage

This panel indicates container memory usage by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#frontend-container-memory-usage).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### frontend: container_missing

This panel indicates container missing.

This value is the number of times a container has not been seen for more than one minute. If you observe this
value change independent of deployment events (such as an upgrade), it could indicate pods are being OOM killed or terminated for some other reasons.

- **Kubernetes:**
	- Determine if the pod was OOM killed using `kubectl describe pod (frontend|sourcegraph-frontend)` (look for `OOMKilled: true`) and, if so, consider increasing the memory limit in the relevant `Deployment.yaml`.
	- Check the logs before the container restarted to see if there are `panic:` messages or similar using `kubectl logs -p (frontend|sourcegraph-frontend)`.
- **Docker Compose:**
	- Determine if the pod was OOM killed using `docker inspect -f '{{json .State}}' (frontend|sourcegraph-frontend)` (look for `"OOMKilled":true`) and, if so, consider increasing the memory limit of the (frontend|sourcegraph-frontend) container in `docker-compose.yml`.
	- Check the logs before the container restarted to see if there are `panic:` messages or similar using `docker logs (frontend|sourcegraph-frontend)` (note this will include logs from the previous and currently running container).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

### Frontend: Provisioning indicators (not available on server)

#### frontend: provisioning_container_cpu_usage_long_term

This panel indicates container cpu usage total (90th percentile over 1d) across all cores by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#frontend-provisioning-container-cpu-usage-long-term).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### frontend: provisioning_container_memory_usage_long_term

This panel indicates container memory usage (1d maximum) by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#frontend-provisioning-container-memory-usage-long-term).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### frontend: provisioning_container_cpu_usage_short_term

This panel indicates container cpu usage total (5m maximum) across all cores by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#frontend-provisioning-container-cpu-usage-short-term).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### frontend: provisioning_container_memory_usage_short_term

This panel indicates container memory usage (5m maximum) by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#frontend-provisioning-container-memory-usage-short-term).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

### Frontend: Golang runtime monitoring

#### frontend: go_goroutines

This panel indicates maximum active goroutines.

A high value here indicates a possible goroutine leak.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#frontend-go-goroutines).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### frontend: go_gc_duration_seconds

This panel indicates maximum go garbage collection duration.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#frontend-go-gc-duration-seconds).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

### Frontend: Kubernetes monitoring (only available on Kubernetes)

#### frontend: pods_available_percentage

This panel indicates percentage pods available.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#frontend-pods-available-percentage).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

### Frontend: Sentinel queries (only on sourcegraph.com)

#### frontend: mean_successful_sentinel_duration_5m

This panel indicates mean successful sentinel search duration over 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#frontend-mean-successful-sentinel-duration-5m).

<sub>*Managed by the [Sourcegraph Search team](https://about.sourcegraph.com/handbook/engineering/search).*</sub>

<br />

#### frontend: mean_sentinel_stream_latency_5m

This panel indicates mean sentinel stream latency over 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#frontend-mean-sentinel-stream-latency-5m).

<sub>*Managed by the [Sourcegraph Search team](https://about.sourcegraph.com/handbook/engineering/search).*</sub>

<br />

#### frontend: 90th_percentile_successful_sentinel_duration_5m

This panel indicates 90th percentile successful sentinel search duration over 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#frontend-90th-percentile-successful-sentinel-duration-5m).

<sub>*Managed by the [Sourcegraph Search team](https://about.sourcegraph.com/handbook/engineering/search).*</sub>

<br />

#### frontend: 90th_percentile_sentinel_stream_latency_5m

This panel indicates 90th percentile sentinel stream latency over 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#frontend-90th-percentile-sentinel-stream-latency-5m).

<sub>*Managed by the [Sourcegraph Search team](https://about.sourcegraph.com/handbook/engineering/search).*</sub>

<br />

#### frontend: mean_successful_sentinel_duration_by_query_5m

This panel indicates mean successful sentinel search duration by query over 5m.

- The mean search duration for sentinel queries, broken down by query. Useful for debugging whether a slowdown is limited to a specific type of query.

<sub>*Managed by the [Sourcegraph Search team](https://about.sourcegraph.com/handbook/engineering/search).*</sub>

<br />

#### frontend: mean_sentinel_stream_latency_by_query_5m

This panel indicates mean sentinel stream latency by query over 5m.

- The mean streaming search latency for sentinel queries, broken down by query. Useful for debugging whether a slowdown is limited to a specific type of query.

<sub>*Managed by the [Sourcegraph Search team](https://about.sourcegraph.com/handbook/engineering/search).*</sub>

<br />

#### frontend: unsuccessful_status_rate_5m

This panel indicates unsuccessful status rate per 5m.

- The rate of unsuccessful sentinel query, broken down by failure type

<sub>*Managed by the [Sourcegraph Search team](https://about.sourcegraph.com/handbook/engineering/search).*</sub>

<br />

## Git Server

<p class="subtitle">Stores, manages, and operates Git repositories.</p>

#### gitserver: memory_working_set

This panel indicates memory working set.



<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### gitserver: go_routines

This panel indicates go routines.



<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### gitserver: cpu_throttling_time

This panel indicates container CPU throttling time %.



<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### gitserver: cpu_usage_seconds

This panel indicates cpu usage seconds.



<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### gitserver: disk_space_remaining

This panel indicates disk space remaining by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#gitserver-disk-space-remaining).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### gitserver: io_reads_total

This panel indicates i/o reads total.



<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### gitserver: io_writes_total

This panel indicates i/o writes total.



<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### gitserver: io_reads

This panel indicates i/o reads.



<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### gitserver: io_writes

This panel indicates i/o writes.



<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### gitserver: io_read_througput

This panel indicates i/o read throughput.



<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### gitserver: io_write_throughput

This panel indicates i/o write throughput.



<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### gitserver: running_git_commands

This panel indicates git commands running on each gitserver instance.

A high value signals load.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#gitserver-running-git-commands).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### gitserver: git_commands_received

This panel indicates rate of git commands received across all instances.

per second rate per command across all instances

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### gitserver: repository_clone_queue_size

This panel indicates repository clone queue size.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#gitserver-repository-clone-queue-size).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### gitserver: repository_existence_check_queue_size

This panel indicates repository existence check queue size.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#gitserver-repository-existence-check-queue-size).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### gitserver: echo_command_duration_test

This panel indicates echo test command duration.

A high value here likely indicates a problem, especially if consistently high.
You can query for individual commands using `sum by (cmd)(src_gitserver_exec_running)` in Grafana (`/-/debug/grafana`) to see if a specific Git Server command might be spiking in frequency.

If this value is consistently high, consider the following:

- **Single container deployments:** Upgrade to a [Docker Compose deployment](../install/docker-compose/migrate.md) which offers better scalability and resource isolation.
- **Kubernetes and Docker Compose:** Check that you are running a similar number of git server replicas and that their CPU/memory limits are allocated according to what is shown in the [Sourcegraph resource estimator](../install/resource_estimator.md).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### gitserver: frontend_internal_api_error_responses

This panel indicates frontend-internal API error responses every 5m by route.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#gitserver-frontend-internal-api-error-responses).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

### Git Server: Gitserver cleanup jobs

#### gitserver: janitor_running

This panel indicates if the janitor process is running.

1, if the janitor process is currently running

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### gitserver: janitor_job_duration

This panel indicates 95th percentile job run duration.

95th percentile job run duration

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### gitserver: repos_removed

This panel indicates repositories removed due to disk pressure.

Repositories removed due to disk pressure

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

### Git Server: Database connections

#### gitserver: max_open_conns

This panel indicates maximum open.

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### gitserver: open_conns

This panel indicates established.

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### gitserver: in_use

This panel indicates used.

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### gitserver: idle

This panel indicates idle.

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### gitserver: mean_blocked_seconds_per_conn_request

This panel indicates mean blocked seconds per conn request.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#gitserver-mean-blocked-seconds-per-conn-request).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### gitserver: closed_max_idle

This panel indicates closed by SetMaxIdleConns.

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### gitserver: closed_max_lifetime

This panel indicates closed by SetConnMaxLifetime.

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### gitserver: closed_max_idle_time

This panel indicates closed by SetConnMaxIdleTime.

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

### Git Server: Container monitoring (not available on server)

#### gitserver: container_cpu_usage

This panel indicates container cpu usage total (1m average) across all cores by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#gitserver-container-cpu-usage).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### gitserver: container_memory_usage

This panel indicates container memory usage by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#gitserver-container-memory-usage).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### gitserver: container_missing

This panel indicates container missing.

This value is the number of times a container has not been seen for more than one minute. If you observe this
value change independent of deployment events (such as an upgrade), it could indicate pods are being OOM killed or terminated for some other reasons.

- **Kubernetes:**
	- Determine if the pod was OOM killed using `kubectl describe pod gitserver` (look for `OOMKilled: true`) and, if so, consider increasing the memory limit in the relevant `Deployment.yaml`.
	- Check the logs before the container restarted to see if there are `panic:` messages or similar using `kubectl logs -p gitserver`.
- **Docker Compose:**
	- Determine if the pod was OOM killed using `docker inspect -f '{{json .State}}' gitserver` (look for `"OOMKilled":true`) and, if so, consider increasing the memory limit of the gitserver container in `docker-compose.yml`.
	- Check the logs before the container restarted to see if there are `panic:` messages or similar using `docker logs gitserver` (note this will include logs from the previous and currently running container).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### gitserver: fs_io_operations

This panel indicates filesystem reads and writes rate by instance over 1h.

This value indicates the number of filesystem read and write operations by containers of this service.
When extremely high, this can indicate a resource usage problem, or can cause problems with the service itself, especially if high values or spikes correlate with {{CONTAINER_NAME}} issues.

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

### Git Server: Provisioning indicators (not available on server)

#### gitserver: provisioning_container_cpu_usage_long_term

This panel indicates container cpu usage total (90th percentile over 1d) across all cores by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#gitserver-provisioning-container-cpu-usage-long-term).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### gitserver: provisioning_container_memory_usage_long_term

This panel indicates container memory usage (1d maximum) by instance.

Git Server is expected to use up all the memory it is provided.

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### gitserver: provisioning_container_cpu_usage_short_term

This panel indicates container cpu usage total (5m maximum) across all cores by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#gitserver-provisioning-container-cpu-usage-short-term).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### gitserver: provisioning_container_memory_usage_short_term

This panel indicates container memory usage (5m maximum) by instance.

Git Server is expected to use up all the memory it is provided.

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

### Git Server: Golang runtime monitoring

#### gitserver: go_goroutines

This panel indicates maximum active goroutines.

A high value here indicates a possible goroutine leak.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#gitserver-go-goroutines).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### gitserver: go_gc_duration_seconds

This panel indicates maximum go garbage collection duration.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#gitserver-go-gc-duration-seconds).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

### Git Server: Kubernetes monitoring (ignore if using Docker Compose or server)

#### gitserver: pods_available_percentage

This panel indicates percentage pods available.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#gitserver-pods-available-percentage).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

## GitHub Proxy

<p class="subtitle">Proxies all requests to github.com, keeping track of and managing rate limits.</p>

### GitHub Proxy: GitHub API monitoring

#### github-proxy: github_proxy_waiting_requests

This panel indicates number of requests waiting on the global mutex.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#github-proxy-github-proxy-waiting-requests).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

### GitHub Proxy: Container monitoring (not available on server)

#### github-proxy: container_cpu_usage

This panel indicates container cpu usage total (1m average) across all cores by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#github-proxy-container-cpu-usage).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### github-proxy: container_memory_usage

This panel indicates container memory usage by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#github-proxy-container-memory-usage).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### github-proxy: container_missing

This panel indicates container missing.

This value is the number of times a container has not been seen for more than one minute. If you observe this
value change independent of deployment events (such as an upgrade), it could indicate pods are being OOM killed or terminated for some other reasons.

- **Kubernetes:**
	- Determine if the pod was OOM killed using `kubectl describe pod github-proxy` (look for `OOMKilled: true`) and, if so, consider increasing the memory limit in the relevant `Deployment.yaml`.
	- Check the logs before the container restarted to see if there are `panic:` messages or similar using `kubectl logs -p github-proxy`.
- **Docker Compose:**
	- Determine if the pod was OOM killed using `docker inspect -f '{{json .State}}' github-proxy` (look for `"OOMKilled":true`) and, if so, consider increasing the memory limit of the github-proxy container in `docker-compose.yml`.
	- Check the logs before the container restarted to see if there are `panic:` messages or similar using `docker logs github-proxy` (note this will include logs from the previous and currently running container).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

### GitHub Proxy: Provisioning indicators (not available on server)

#### github-proxy: provisioning_container_cpu_usage_long_term

This panel indicates container cpu usage total (90th percentile over 1d) across all cores by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#github-proxy-provisioning-container-cpu-usage-long-term).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### github-proxy: provisioning_container_memory_usage_long_term

This panel indicates container memory usage (1d maximum) by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#github-proxy-provisioning-container-memory-usage-long-term).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### github-proxy: provisioning_container_cpu_usage_short_term

This panel indicates container cpu usage total (5m maximum) across all cores by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#github-proxy-provisioning-container-cpu-usage-short-term).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### github-proxy: provisioning_container_memory_usage_short_term

This panel indicates container memory usage (5m maximum) by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#github-proxy-provisioning-container-memory-usage-short-term).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

### GitHub Proxy: Golang runtime monitoring

#### github-proxy: go_goroutines

This panel indicates maximum active goroutines.

A high value here indicates a possible goroutine leak.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#github-proxy-go-goroutines).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### github-proxy: go_gc_duration_seconds

This panel indicates maximum go garbage collection duration.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#github-proxy-go-gc-duration-seconds).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

### GitHub Proxy: Kubernetes monitoring (only available on Kubernetes)

#### github-proxy: pods_available_percentage

This panel indicates percentage pods available.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#github-proxy-pods-available-percentage).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

## Postgres

<p class="subtitle">Postgres metrics, exported from postgres_exporter (only available on Kubernetes).</p>

#### postgres: connections

This panel indicates active connections.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#postgres-connections).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### postgres: transaction_durations

This panel indicates maximum transaction durations.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#postgres-transaction-durations).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

### Postgres: Database and collector status

#### postgres: postgres_up

This panel indicates database availability.

A non-zero value indicates the database is online.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#postgres-postgres-up).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### postgres: invalid_indexes

This panel indicates invalid indexes (unusable by the query planner).

A non-zero value indicates the that Postgres failed to build an index. Expect degraded performance until the index is manually rebuilt.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#postgres-invalid-indexes).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### postgres: pg_exporter_err

This panel indicates errors scraping postgres exporter.

This value indicates issues retrieving metrics from postgres_exporter.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#postgres-pg-exporter-err).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### postgres: migration_in_progress

This panel indicates active schema migration.

A 0 value indicates that no migration is in progress.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#postgres-migration-in-progress).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

### Postgres: Object size and bloat

#### postgres: pg_table_size

This panel indicates table size.

Total size of this table

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### postgres: pg_table_bloat_ratio

This panel indicates table bloat ratio.

Estimated bloat ratio of this table (high bloat = high overhead)

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### postgres: pg_index_size

This panel indicates index size.

Total size of this index

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### postgres: pg_index_bloat_ratio

This panel indicates index bloat ratio.

Estimated bloat ratio of this index (high bloat = high overhead)

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

### Postgres: Provisioning indicators (not available on server)

#### postgres: provisioning_container_cpu_usage_long_term

This panel indicates container cpu usage total (90th percentile over 1d) across all cores by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#postgres-provisioning-container-cpu-usage-long-term).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### postgres: provisioning_container_memory_usage_long_term

This panel indicates container memory usage (1d maximum) by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#postgres-provisioning-container-memory-usage-long-term).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### postgres: provisioning_container_cpu_usage_short_term

This panel indicates container cpu usage total (5m maximum) across all cores by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#postgres-provisioning-container-cpu-usage-short-term).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### postgres: provisioning_container_memory_usage_short_term

This panel indicates container memory usage (5m maximum) by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#postgres-provisioning-container-memory-usage-short-term).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

### Postgres: Kubernetes monitoring (only available on Kubernetes)

#### postgres: pods_available_percentage

This panel indicates percentage pods available.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#postgres-pods-available-percentage).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

## Precise Code Intel Worker

<p class="subtitle">Handles conversion of uploaded precise code intelligence bundles.</p>

### Precise Code Intel Worker: Upload queue

#### precise-code-intel-worker: upload_queue_size

This panel indicates queue size.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#precise-code-intel-worker-upload-queue-size).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### precise-code-intel-worker: upload_queue_growth_rate

This panel indicates queue growth rate over 30m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#precise-code-intel-worker-upload-queue-growth-rate).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### precise-code-intel-worker: job_errors

This panel indicates job errors errors every 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#precise-code-intel-worker-job-errors).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### precise-code-intel-worker: active_workers

This panel indicates active workers processing uploads.

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### precise-code-intel-worker: active_jobs

This panel indicates active jobs.

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

### Precise Code Intel Worker: Workers

#### precise-code-intel-worker: job_99th_percentile_duration

This panel indicates 99th percentile successful job duration over 5m.

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

### Precise Code Intel Worker: Stores and clients

#### precise-code-intel-worker: codeintel_dbstore_99th_percentile_duration

This panel indicates 99th percentile successful database store operation duration over 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#precise-code-intel-worker-codeintel-dbstore-99th-percentile-duration).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### precise-code-intel-worker: codeintel_dbstore_errors

This panel indicates database store errors every 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#precise-code-intel-worker-codeintel-dbstore-errors).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### precise-code-intel-worker: codeintel_workerstore_99th_percentile_duration

This panel indicates 99th percentile successful worker store operation duration over 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#precise-code-intel-worker-codeintel-workerstore-99th-percentile-duration).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### precise-code-intel-worker: codeintel_workerstore_errors

This panel indicates worker store errors every 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#precise-code-intel-worker-codeintel-workerstore-errors).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### precise-code-intel-worker: codeintel_lsifstore_99th_percentile_duration

This panel indicates 99th percentile successful LSIF store operation duration over 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#precise-code-intel-worker-codeintel-lsifstore-99th-percentile-duration).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### precise-code-intel-worker: codeintel_lsifstore_errors

This panel indicates lSIF store errors every 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#precise-code-intel-worker-codeintel-lsifstore-errors).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### precise-code-intel-worker: codeintel_uploadstore_99th_percentile_duration

This panel indicates 99th percentile successful upload store operation duration over 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#precise-code-intel-worker-codeintel-uploadstore-99th-percentile-duration).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### precise-code-intel-worker: codeintel_uploadstore_errors

This panel indicates upload store errors every 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#precise-code-intel-worker-codeintel-uploadstore-errors).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### precise-code-intel-worker: codeintel_gitserverclient_99th_percentile_duration

This panel indicates 99th percentile successful gitserver client operation duration over 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#precise-code-intel-worker-codeintel-gitserverclient-99th-percentile-duration).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### precise-code-intel-worker: codeintel_gitserverclient_errors

This panel indicates gitserver client errors every 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#precise-code-intel-worker-codeintel-gitserverclient-errors).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

### Precise Code Intel Worker: Database connections

#### precise-code-intel-worker: max_open_conns

This panel indicates maximum open.

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### precise-code-intel-worker: open_conns

This panel indicates established.

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### precise-code-intel-worker: in_use

This panel indicates used.

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### precise-code-intel-worker: idle

This panel indicates idle.

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### precise-code-intel-worker: mean_blocked_seconds_per_conn_request

This panel indicates mean blocked seconds per conn request.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#precise-code-intel-worker-mean-blocked-seconds-per-conn-request).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### precise-code-intel-worker: closed_max_idle

This panel indicates closed by SetMaxIdleConns.

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### precise-code-intel-worker: closed_max_lifetime

This panel indicates closed by SetConnMaxLifetime.

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### precise-code-intel-worker: closed_max_idle_time

This panel indicates closed by SetConnMaxIdleTime.

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

### Precise Code Intel Worker: Internal service requests

#### precise-code-intel-worker: frontend_internal_api_error_responses

This panel indicates frontend-internal API error responses every 5m by route.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#precise-code-intel-worker-frontend-internal-api-error-responses).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

### Precise Code Intel Worker: Container monitoring (not available on server)

#### precise-code-intel-worker: container_cpu_usage

This panel indicates container cpu usage total (1m average) across all cores by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#precise-code-intel-worker-container-cpu-usage).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### precise-code-intel-worker: container_memory_usage

This panel indicates container memory usage by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#precise-code-intel-worker-container-memory-usage).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### precise-code-intel-worker: container_missing

This panel indicates container missing.

This value is the number of times a container has not been seen for more than one minute. If you observe this
value change independent of deployment events (such as an upgrade), it could indicate pods are being OOM killed or terminated for some other reasons.

- **Kubernetes:**
	- Determine if the pod was OOM killed using `kubectl describe pod precise-code-intel-worker` (look for `OOMKilled: true`) and, if so, consider increasing the memory limit in the relevant `Deployment.yaml`.
	- Check the logs before the container restarted to see if there are `panic:` messages or similar using `kubectl logs -p precise-code-intel-worker`.
- **Docker Compose:**
	- Determine if the pod was OOM killed using `docker inspect -f '{{json .State}}' precise-code-intel-worker` (look for `"OOMKilled":true`) and, if so, consider increasing the memory limit of the precise-code-intel-worker container in `docker-compose.yml`.
	- Check the logs before the container restarted to see if there are `panic:` messages or similar using `docker logs precise-code-intel-worker` (note this will include logs from the previous and currently running container).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

### Precise Code Intel Worker: Provisioning indicators (not available on server)

#### precise-code-intel-worker: provisioning_container_cpu_usage_long_term

This panel indicates container cpu usage total (90th percentile over 1d) across all cores by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#precise-code-intel-worker-provisioning-container-cpu-usage-long-term).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### precise-code-intel-worker: provisioning_container_memory_usage_long_term

This panel indicates container memory usage (1d maximum) by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#precise-code-intel-worker-provisioning-container-memory-usage-long-term).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### precise-code-intel-worker: provisioning_container_cpu_usage_short_term

This panel indicates container cpu usage total (5m maximum) across all cores by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#precise-code-intel-worker-provisioning-container-cpu-usage-short-term).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### precise-code-intel-worker: provisioning_container_memory_usage_short_term

This panel indicates container memory usage (5m maximum) by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#precise-code-intel-worker-provisioning-container-memory-usage-short-term).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

### Precise Code Intel Worker: Golang runtime monitoring

#### precise-code-intel-worker: go_goroutines

This panel indicates maximum active goroutines.

A high value here indicates a possible goroutine leak.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#precise-code-intel-worker-go-goroutines).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### precise-code-intel-worker: go_gc_duration_seconds

This panel indicates maximum go garbage collection duration.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#precise-code-intel-worker-go-gc-duration-seconds).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

### Precise Code Intel Worker: Kubernetes monitoring (only available on Kubernetes)

#### precise-code-intel-worker: pods_available_percentage

This panel indicates percentage pods available.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#precise-code-intel-worker-pods-available-percentage).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

## Query Runner

<p class="subtitle">Periodically runs saved searches and instructs the frontend to send out notifications.</p>

#### query-runner: frontend_internal_api_error_responses

This panel indicates frontend-internal API error responses every 5m by route.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#query-runner-frontend-internal-api-error-responses).

<sub>*Managed by the [Sourcegraph Search team](https://about.sourcegraph.com/handbook/engineering/search).*</sub>

<br />

### Query Runner: Container monitoring (not available on server)

#### query-runner: container_memory_usage

This panel indicates container memory usage by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#query-runner-container-memory-usage).

<sub>*Managed by the [Sourcegraph Search team](https://about.sourcegraph.com/handbook/engineering/search).*</sub>

<br />

#### query-runner: container_cpu_usage

This panel indicates container cpu usage total (1m average) across all cores by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#query-runner-container-cpu-usage).

<sub>*Managed by the [Sourcegraph Search team](https://about.sourcegraph.com/handbook/engineering/search).*</sub>

<br />

#### query-runner: container_missing

This panel indicates container missing.

This value is the number of times a container has not been seen for more than one minute. If you observe this
value change independent of deployment events (such as an upgrade), it could indicate pods are being OOM killed or terminated for some other reasons.

- **Kubernetes:**
	- Determine if the pod was OOM killed using `kubectl describe pod query-runner` (look for `OOMKilled: true`) and, if so, consider increasing the memory limit in the relevant `Deployment.yaml`.
	- Check the logs before the container restarted to see if there are `panic:` messages or similar using `kubectl logs -p query-runner`.
- **Docker Compose:**
	- Determine if the pod was OOM killed using `docker inspect -f '{{json .State}}' query-runner` (look for `"OOMKilled":true`) and, if so, consider increasing the memory limit of the query-runner container in `docker-compose.yml`.
	- Check the logs before the container restarted to see if there are `panic:` messages or similar using `docker logs query-runner` (note this will include logs from the previous and currently running container).

<sub>*Managed by the [Sourcegraph Search team](https://about.sourcegraph.com/handbook/engineering/search).*</sub>

<br />

### Query Runner: Provisioning indicators (not available on server)

#### query-runner: provisioning_container_cpu_usage_long_term

This panel indicates container cpu usage total (90th percentile over 1d) across all cores by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#query-runner-provisioning-container-cpu-usage-long-term).

<sub>*Managed by the [Sourcegraph Search team](https://about.sourcegraph.com/handbook/engineering/search).*</sub>

<br />

#### query-runner: provisioning_container_memory_usage_long_term

This panel indicates container memory usage (1d maximum) by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#query-runner-provisioning-container-memory-usage-long-term).

<sub>*Managed by the [Sourcegraph Search team](https://about.sourcegraph.com/handbook/engineering/search).*</sub>

<br />

#### query-runner: provisioning_container_cpu_usage_short_term

This panel indicates container cpu usage total (5m maximum) across all cores by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#query-runner-provisioning-container-cpu-usage-short-term).

<sub>*Managed by the [Sourcegraph Search team](https://about.sourcegraph.com/handbook/engineering/search).*</sub>

<br />

#### query-runner: provisioning_container_memory_usage_short_term

This panel indicates container memory usage (5m maximum) by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#query-runner-provisioning-container-memory-usage-short-term).

<sub>*Managed by the [Sourcegraph Search team](https://about.sourcegraph.com/handbook/engineering/search).*</sub>

<br />

### Query Runner: Golang runtime monitoring

#### query-runner: go_goroutines

This panel indicates maximum active goroutines.

A high value here indicates a possible goroutine leak.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#query-runner-go-goroutines).

<sub>*Managed by the [Sourcegraph Search team](https://about.sourcegraph.com/handbook/engineering/search).*</sub>

<br />

#### query-runner: go_gc_duration_seconds

This panel indicates maximum go garbage collection duration.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#query-runner-go-gc-duration-seconds).

<sub>*Managed by the [Sourcegraph Search team](https://about.sourcegraph.com/handbook/engineering/search).*</sub>

<br />

### Query Runner: Kubernetes monitoring (only available on Kubernetes)

#### query-runner: pods_available_percentage

This panel indicates percentage pods available.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#query-runner-pods-available-percentage).

<sub>*Managed by the [Sourcegraph Search team](https://about.sourcegraph.com/handbook/engineering/search).*</sub>

<br />

## Worker

<p class="subtitle">Manages background processes.</p>

### Worker: Active jobs

#### worker: worker_job_count

This panel indicates number of worker instances running each job.

The number of worker instances running each job type.
It is necessary for each job type to be managed by at least one worker instance.


<br />

#### worker: worker_job_codeintel-janitor_count

This panel indicates number of worker instances running the codeintel-janitor job.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#worker-worker-job-codeintel-janitor-count).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### worker: worker_job_codeintel-commitgraph_count

This panel indicates number of worker instances running the codeintel-commitgraph job.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#worker-worker-job-codeintel-commitgraph-count).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### worker: worker_job_codeintel-auto-indexing_count

This panel indicates number of worker instances running the codeintel-auto-indexing job.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#worker-worker-job-codeintel-auto-indexing-count).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

### Worker: Precise code intelligence commit graph updater

#### worker: codeintel_commit_graph_queue_size

This panel indicates commit graph queue size.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#worker-codeintel-commit-graph-queue-size).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### worker: codeintel_commit_graph_queue_growth_rate

This panel indicates commit graph queue growth rate over 30m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#worker-codeintel-commit-graph-queue-growth-rate).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### worker: codeintel_commit_graph_updater_99th_percentile_duration

This panel indicates 99th percentile successful commit graph updater operation duration over 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#worker-codeintel-commit-graph-updater-99th-percentile-duration).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### worker: codeintel_commit_graph_updater_errors

This panel indicates commit graph updater errors every 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#worker-codeintel-commit-graph-updater-errors).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

### Worker: Precise code intelligence janitor

#### worker: codeintel_janitor_errors

This panel indicates janitor errors every 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#worker-codeintel-janitor-errors).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### worker: codeintel_upload_records_removed

This panel indicates upload records expired or deleted every 5m.

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### worker: codeintel_index_records_removed

This panel indicates index records expired or deleted every 5m.

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### worker: codeintel_lsif_data_removed

This panel indicates data for unreferenced upload records removed every 5m.

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### worker: codeintel_background_upload_resets

This panel indicates upload records re-queued (due to unresponsive worker) every 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#worker-codeintel-background-upload-resets).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### worker: codeintel_background_upload_reset_failures

This panel indicates upload records errored due to repeated reset every 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#worker-codeintel-background-upload-reset-failures).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### worker: codeintel_background_index_resets

This panel indicates index records re-queued (due to unresponsive indexer) every 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#worker-codeintel-background-index-resets).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### worker: codeintel_background_index_reset_failures

This panel indicates index records errored due to repeated reset every 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#worker-codeintel-background-index-reset-failures).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

### Worker: Auto-indexing

#### worker: codeintel_indexing_99th_percentile_duration

This panel indicates 99th percentile successful indexing operation duration over 5m.

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### worker: codeintel_indexing_errors

This panel indicates indexing errors every 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#worker-codeintel-indexing-errors).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### worker: codeintel_autoindex_enqueuer_99th_percentile_duration

This panel indicates 99th percentile successful index enqueuer operation duration over 5m.

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### worker: codeintel_autoindex_enqueuer_errors

This panel indicates index enqueuer errors every 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#worker-codeintel-autoindex-enqueuer-errors).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

### Worker: Database connections

#### worker: max_open_conns

This panel indicates maximum open.

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### worker: open_conns

This panel indicates established.

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### worker: in_use

This panel indicates used.

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### worker: idle

This panel indicates idle.

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### worker: mean_blocked_seconds_per_conn_request

This panel indicates mean blocked seconds per conn request.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#worker-mean-blocked-seconds-per-conn-request).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### worker: closed_max_idle

This panel indicates closed by SetMaxIdleConns.

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### worker: closed_max_lifetime

This panel indicates closed by SetConnMaxLifetime.

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### worker: closed_max_idle_time

This panel indicates closed by SetConnMaxIdleTime.

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

### Worker: Internal service requests

#### worker: frontend_internal_api_error_responses

This panel indicates frontend-internal API error responses every 5m by route.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#worker-frontend-internal-api-error-responses).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

### Worker: Container monitoring (not available on server)

#### worker: container_cpu_usage

This panel indicates container cpu usage total (1m average) across all cores by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#worker-container-cpu-usage).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### worker: container_memory_usage

This panel indicates container memory usage by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#worker-container-memory-usage).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### worker: container_missing

This panel indicates container missing.

This value is the number of times a container has not been seen for more than one minute. If you observe this
value change independent of deployment events (such as an upgrade), it could indicate pods are being OOM killed or terminated for some other reasons.

- **Kubernetes:**
	- Determine if the pod was OOM killed using `kubectl describe pod worker` (look for `OOMKilled: true`) and, if so, consider increasing the memory limit in the relevant `Deployment.yaml`.
	- Check the logs before the container restarted to see if there are `panic:` messages or similar using `kubectl logs -p worker`.
- **Docker Compose:**
	- Determine if the pod was OOM killed using `docker inspect -f '{{json .State}}' worker` (look for `"OOMKilled":true`) and, if so, consider increasing the memory limit of the worker container in `docker-compose.yml`.
	- Check the logs before the container restarted to see if there are `panic:` messages or similar using `docker logs worker` (note this will include logs from the previous and currently running container).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

### Worker: Provisioning indicators (not available on server)

#### worker: provisioning_container_cpu_usage_long_term

This panel indicates container cpu usage total (90th percentile over 1d) across all cores by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#worker-provisioning-container-cpu-usage-long-term).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### worker: provisioning_container_memory_usage_long_term

This panel indicates container memory usage (1d maximum) by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#worker-provisioning-container-memory-usage-long-term).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### worker: provisioning_container_cpu_usage_short_term

This panel indicates container cpu usage total (5m maximum) across all cores by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#worker-provisioning-container-cpu-usage-short-term).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### worker: provisioning_container_memory_usage_short_term

This panel indicates container memory usage (5m maximum) by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#worker-provisioning-container-memory-usage-short-term).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

### Worker: Golang runtime monitoring

#### worker: go_goroutines

This panel indicates maximum active goroutines.

A high value here indicates a possible goroutine leak.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#worker-go-goroutines).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### worker: go_gc_duration_seconds

This panel indicates maximum go garbage collection duration.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#worker-go-gc-duration-seconds).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

### Worker: Kubernetes monitoring (only available on Kubernetes)

#### worker: pods_available_percentage

This panel indicates percentage pods available.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#worker-pods-available-percentage).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

## Repo Updater

<p class="subtitle">Manages interaction with code hosts, instructs Gitserver to update repositories.</p>

#### repo-updater: frontend_internal_api_error_responses

This panel indicates frontend-internal API error responses every 5m by route.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#repo-updater-frontend-internal-api-error-responses).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

### Repo Updater: Repositories

#### repo-updater: syncer_sync_last_time

This panel indicates time since last sync.

A high value here indicates issues synchronizing repo metadata.
If the value is persistently high, make sure all external services have valid tokens.

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### repo-updater: src_repoupdater_max_sync_backoff

This panel indicates time since oldest sync.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#repo-updater-src-repoupdater-max-sync-backoff).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### repo-updater: src_repoupdater_syncer_sync_errors_total

This panel indicates site level external service sync error rate.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#repo-updater-src-repoupdater-syncer-sync-errors-total).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### repo-updater: syncer_sync_start

This panel indicates repo metadata sync was started.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#repo-updater-syncer-sync-start).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### repo-updater: syncer_sync_duration

This panel indicates 95th repositories sync duration.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#repo-updater-syncer-sync-duration).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### repo-updater: source_duration

This panel indicates 95th repositories source duration.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#repo-updater-source-duration).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### repo-updater: syncer_synced_repos

This panel indicates repositories synced.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#repo-updater-syncer-synced-repos).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### repo-updater: sourced_repos

This panel indicates repositories sourced.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#repo-updater-sourced-repos).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### repo-updater: user_added_repos

This panel indicates total number of user added repos.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#repo-updater-user-added-repos).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### repo-updater: purge_failed

This panel indicates repositories purge failed.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#repo-updater-purge-failed).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### repo-updater: sched_auto_fetch

This panel indicates repositories scheduled due to hitting a deadline.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#repo-updater-sched-auto-fetch).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### repo-updater: sched_manual_fetch

This panel indicates repositories scheduled due to user traffic.

Check repo-updater logs if this value is persistently high.
This does not indicate anything if there are no user added code hosts.

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### repo-updater: sched_known_repos

This panel indicates repositories managed by the scheduler.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#repo-updater-sched-known-repos).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### repo-updater: sched_update_queue_length

This panel indicates rate of growth of update queue length over 5 minutes.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#repo-updater-sched-update-queue-length).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### repo-updater: sched_loops

This panel indicates scheduler loops.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#repo-updater-sched-loops).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### repo-updater: sched_error

This panel indicates repositories schedule error rate.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#repo-updater-sched-error).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

### Repo Updater: Permissions

#### repo-updater: perms_syncer_perms

This panel indicates time gap between least and most up to date permissions.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#repo-updater-perms-syncer-perms).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### repo-updater: perms_syncer_stale_perms

This panel indicates number of entities with stale permissions.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#repo-updater-perms-syncer-stale-perms).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### repo-updater: perms_syncer_no_perms

This panel indicates number of entities with no permissions.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#repo-updater-perms-syncer-no-perms).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### repo-updater: perms_syncer_sync_duration

This panel indicates 95th permissions sync duration.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#repo-updater-perms-syncer-sync-duration).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### repo-updater: perms_syncer_queue_size

This panel indicates permissions sync queued items.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#repo-updater-perms-syncer-queue-size).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### repo-updater: perms_syncer_sync_errors

This panel indicates permissions sync error rate.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#repo-updater-perms-syncer-sync-errors).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

### Repo Updater: External services

#### repo-updater: src_repoupdater_external_services_total

This panel indicates the total number of external services.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#repo-updater-src-repoupdater-external-services-total).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### repo-updater: src_repoupdater_user_external_services_total

This panel indicates the total number of user added external services.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#repo-updater-src-repoupdater-user-external-services-total).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### repo-updater: repoupdater_queued_sync_jobs_total

This panel indicates the total number of queued sync jobs.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#repo-updater-repoupdater-queued-sync-jobs-total).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### repo-updater: repoupdater_completed_sync_jobs_total

This panel indicates the total number of completed sync jobs.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#repo-updater-repoupdater-completed-sync-jobs-total).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### repo-updater: repoupdater_errored_sync_jobs_percentage

This panel indicates the percentage of external services that have failed their most recent sync.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#repo-updater-repoupdater-errored-sync-jobs-percentage).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### repo-updater: github_graphql_rate_limit_remaining

This panel indicates remaining calls to GitHub graphql API before hitting the rate limit.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#repo-updater-github-graphql-rate-limit-remaining).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### repo-updater: github_rest_rate_limit_remaining

This panel indicates remaining calls to GitHub rest API before hitting the rate limit.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#repo-updater-github-rest-rate-limit-remaining).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### repo-updater: github_search_rate_limit_remaining

This panel indicates remaining calls to GitHub search API before hitting the rate limit.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#repo-updater-github-search-rate-limit-remaining).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### repo-updater: github_graphql_rate_limit_wait_duration

This panel indicates time spent waiting for the GitHub graphql API rate limiter.

Indicates how long we`re waiting on the rate limit once it has been exceeded

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### repo-updater: github_rest_rate_limit_wait_duration

This panel indicates time spent waiting for the GitHub rest API rate limiter.

Indicates how long we`re waiting on the rate limit once it has been exceeded

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### repo-updater: github_search_rate_limit_wait_duration

This panel indicates time spent waiting for the GitHub search API rate limiter.

Indicates how long we`re waiting on the rate limit once it has been exceeded

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### repo-updater: gitlab_rest_rate_limit_remaining

This panel indicates remaining calls to GitLab rest API before hitting the rate limit.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#repo-updater-gitlab-rest-rate-limit-remaining).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### repo-updater: gitlab_rest_rate_limit_wait_duration

This panel indicates time spent waiting for the GitLab rest API rate limiter.

Indicates how long we`re waiting on the rate limit once it has been exceeded

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

### Repo Updater: Database connections

#### repo-updater: max_open_conns

This panel indicates maximum open.

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### repo-updater: open_conns

This panel indicates established.

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### repo-updater: in_use

This panel indicates used.

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### repo-updater: idle

This panel indicates idle.

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### repo-updater: mean_blocked_seconds_per_conn_request

This panel indicates mean blocked seconds per conn request.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#repo-updater-mean-blocked-seconds-per-conn-request).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### repo-updater: closed_max_idle

This panel indicates closed by SetMaxIdleConns.

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### repo-updater: closed_max_lifetime

This panel indicates closed by SetConnMaxLifetime.

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### repo-updater: closed_max_idle_time

This panel indicates closed by SetConnMaxIdleTime.

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

### Repo Updater: Container monitoring (not available on server)

#### repo-updater: container_cpu_usage

This panel indicates container cpu usage total (1m average) across all cores by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#repo-updater-container-cpu-usage).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### repo-updater: container_memory_usage

This panel indicates container memory usage by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#repo-updater-container-memory-usage).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### repo-updater: container_missing

This panel indicates container missing.

This value is the number of times a container has not been seen for more than one minute. If you observe this
value change independent of deployment events (such as an upgrade), it could indicate pods are being OOM killed or terminated for some other reasons.

- **Kubernetes:**
	- Determine if the pod was OOM killed using `kubectl describe pod repo-updater` (look for `OOMKilled: true`) and, if so, consider increasing the memory limit in the relevant `Deployment.yaml`.
	- Check the logs before the container restarted to see if there are `panic:` messages or similar using `kubectl logs -p repo-updater`.
- **Docker Compose:**
	- Determine if the pod was OOM killed using `docker inspect -f '{{json .State}}' repo-updater` (look for `"OOMKilled":true`) and, if so, consider increasing the memory limit of the repo-updater container in `docker-compose.yml`.
	- Check the logs before the container restarted to see if there are `panic:` messages or similar using `docker logs repo-updater` (note this will include logs from the previous and currently running container).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

### Repo Updater: Provisioning indicators (not available on server)

#### repo-updater: provisioning_container_cpu_usage_long_term

This panel indicates container cpu usage total (90th percentile over 1d) across all cores by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#repo-updater-provisioning-container-cpu-usage-long-term).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### repo-updater: provisioning_container_memory_usage_long_term

This panel indicates container memory usage (1d maximum) by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#repo-updater-provisioning-container-memory-usage-long-term).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### repo-updater: provisioning_container_cpu_usage_short_term

This panel indicates container cpu usage total (5m maximum) across all cores by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#repo-updater-provisioning-container-cpu-usage-short-term).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### repo-updater: provisioning_container_memory_usage_short_term

This panel indicates container memory usage (5m maximum) by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#repo-updater-provisioning-container-memory-usage-short-term).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

### Repo Updater: Golang runtime monitoring

#### repo-updater: go_goroutines

This panel indicates maximum active goroutines.

A high value here indicates a possible goroutine leak.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#repo-updater-go-goroutines).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### repo-updater: go_gc_duration_seconds

This panel indicates maximum go garbage collection duration.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#repo-updater-go-gc-duration-seconds).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

### Repo Updater: Kubernetes monitoring (only available on Kubernetes)

#### repo-updater: pods_available_percentage

This panel indicates percentage pods available.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#repo-updater-pods-available-percentage).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

## Searcher

<p class="subtitle">Performs unindexed searches (diff and commit search, text search for unindexed branches).</p>

#### searcher: unindexed_search_request_errors

This panel indicates unindexed search request errors every 5m by code.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#searcher-unindexed-search-request-errors).

<sub>*Managed by the [Sourcegraph Search team](https://about.sourcegraph.com/handbook/engineering/search).*</sub>

<br />

#### searcher: replica_traffic

This panel indicates requests per second over 10m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#searcher-replica-traffic).

<sub>*Managed by the [Sourcegraph Search team](https://about.sourcegraph.com/handbook/engineering/search).*</sub>

<br />

#### searcher: frontend_internal_api_error_responses

This panel indicates frontend-internal API error responses every 5m by route.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#searcher-frontend-internal-api-error-responses).

<sub>*Managed by the [Sourcegraph Search team](https://about.sourcegraph.com/handbook/engineering/search).*</sub>

<br />

### Searcher: Container monitoring (not available on server)

#### searcher: container_cpu_usage

This panel indicates container cpu usage total (1m average) across all cores by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#searcher-container-cpu-usage).

<sub>*Managed by the [Sourcegraph Search team](https://about.sourcegraph.com/handbook/engineering/search).*</sub>

<br />

#### searcher: container_memory_usage

This panel indicates container memory usage by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#searcher-container-memory-usage).

<sub>*Managed by the [Sourcegraph Search team](https://about.sourcegraph.com/handbook/engineering/search).*</sub>

<br />

#### searcher: container_missing

This panel indicates container missing.

This value is the number of times a container has not been seen for more than one minute. If you observe this
value change independent of deployment events (such as an upgrade), it could indicate pods are being OOM killed or terminated for some other reasons.

- **Kubernetes:**
	- Determine if the pod was OOM killed using `kubectl describe pod searcher` (look for `OOMKilled: true`) and, if so, consider increasing the memory limit in the relevant `Deployment.yaml`.
	- Check the logs before the container restarted to see if there are `panic:` messages or similar using `kubectl logs -p searcher`.
- **Docker Compose:**
	- Determine if the pod was OOM killed using `docker inspect -f '{{json .State}}' searcher` (look for `"OOMKilled":true`) and, if so, consider increasing the memory limit of the searcher container in `docker-compose.yml`.
	- Check the logs before the container restarted to see if there are `panic:` messages or similar using `docker logs searcher` (note this will include logs from the previous and currently running container).

<sub>*Managed by the [Sourcegraph Search team](https://about.sourcegraph.com/handbook/engineering/search).*</sub>

<br />

### Searcher: Provisioning indicators (not available on server)

#### searcher: provisioning_container_cpu_usage_long_term

This panel indicates container cpu usage total (90th percentile over 1d) across all cores by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#searcher-provisioning-container-cpu-usage-long-term).

<sub>*Managed by the [Sourcegraph Search team](https://about.sourcegraph.com/handbook/engineering/search).*</sub>

<br />

#### searcher: provisioning_container_memory_usage_long_term

This panel indicates container memory usage (1d maximum) by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#searcher-provisioning-container-memory-usage-long-term).

<sub>*Managed by the [Sourcegraph Search team](https://about.sourcegraph.com/handbook/engineering/search).*</sub>

<br />

#### searcher: provisioning_container_cpu_usage_short_term

This panel indicates container cpu usage total (5m maximum) across all cores by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#searcher-provisioning-container-cpu-usage-short-term).

<sub>*Managed by the [Sourcegraph Search team](https://about.sourcegraph.com/handbook/engineering/search).*</sub>

<br />

#### searcher: provisioning_container_memory_usage_short_term

This panel indicates container memory usage (5m maximum) by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#searcher-provisioning-container-memory-usage-short-term).

<sub>*Managed by the [Sourcegraph Search team](https://about.sourcegraph.com/handbook/engineering/search).*</sub>

<br />

### Searcher: Golang runtime monitoring

#### searcher: go_goroutines

This panel indicates maximum active goroutines.

A high value here indicates a possible goroutine leak.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#searcher-go-goroutines).

<sub>*Managed by the [Sourcegraph Search team](https://about.sourcegraph.com/handbook/engineering/search).*</sub>

<br />

#### searcher: go_gc_duration_seconds

This panel indicates maximum go garbage collection duration.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#searcher-go-gc-duration-seconds).

<sub>*Managed by the [Sourcegraph Search team](https://about.sourcegraph.com/handbook/engineering/search).*</sub>

<br />

### Searcher: Kubernetes monitoring (only available on Kubernetes)

#### searcher: pods_available_percentage

This panel indicates percentage pods available.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#searcher-pods-available-percentage).

<sub>*Managed by the [Sourcegraph Search team](https://about.sourcegraph.com/handbook/engineering/search).*</sub>

<br />

## Symbols

<p class="subtitle">Handles symbol searches for unindexed branches.</p>

#### symbols: store_fetch_failures

This panel indicates store fetch failures every 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#symbols-store-fetch-failures).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### symbols: current_fetch_queue_size

This panel indicates current fetch queue size.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#symbols-current-fetch-queue-size).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### symbols: frontend_internal_api_error_responses

This panel indicates frontend-internal API error responses every 5m by route.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#symbols-frontend-internal-api-error-responses).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

### Symbols: Container monitoring (not available on server)

#### symbols: container_cpu_usage

This panel indicates container cpu usage total (1m average) across all cores by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#symbols-container-cpu-usage).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### symbols: container_memory_usage

This panel indicates container memory usage by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#symbols-container-memory-usage).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### symbols: container_missing

This panel indicates container missing.

This value is the number of times a container has not been seen for more than one minute. If you observe this
value change independent of deployment events (such as an upgrade), it could indicate pods are being OOM killed or terminated for some other reasons.

- **Kubernetes:**
	- Determine if the pod was OOM killed using `kubectl describe pod symbols` (look for `OOMKilled: true`) and, if so, consider increasing the memory limit in the relevant `Deployment.yaml`.
	- Check the logs before the container restarted to see if there are `panic:` messages or similar using `kubectl logs -p symbols`.
- **Docker Compose:**
	- Determine if the pod was OOM killed using `docker inspect -f '{{json .State}}' symbols` (look for `"OOMKilled":true`) and, if so, consider increasing the memory limit of the symbols container in `docker-compose.yml`.
	- Check the logs before the container restarted to see if there are `panic:` messages or similar using `docker logs symbols` (note this will include logs from the previous and currently running container).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

### Symbols: Provisioning indicators (not available on server)

#### symbols: provisioning_container_cpu_usage_long_term

This panel indicates container cpu usage total (90th percentile over 1d) across all cores by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#symbols-provisioning-container-cpu-usage-long-term).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### symbols: provisioning_container_memory_usage_long_term

This panel indicates container memory usage (1d maximum) by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#symbols-provisioning-container-memory-usage-long-term).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### symbols: provisioning_container_cpu_usage_short_term

This panel indicates container cpu usage total (5m maximum) across all cores by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#symbols-provisioning-container-cpu-usage-short-term).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### symbols: provisioning_container_memory_usage_short_term

This panel indicates container memory usage (5m maximum) by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#symbols-provisioning-container-memory-usage-short-term).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

### Symbols: Golang runtime monitoring

#### symbols: go_goroutines

This panel indicates maximum active goroutines.

A high value here indicates a possible goroutine leak.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#symbols-go-goroutines).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### symbols: go_gc_duration_seconds

This panel indicates maximum go garbage collection duration.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#symbols-go-gc-duration-seconds).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

### Symbols: Kubernetes monitoring (only available on Kubernetes)

#### symbols: pods_available_percentage

This panel indicates percentage pods available.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#symbols-pods-available-percentage).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

## Syntect Server

<p class="subtitle">Handles syntax highlighting for code files.</p>

#### syntect-server: syntax_highlighting_errors

This panel indicates syntax highlighting errors every 5m.

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### syntect-server: syntax_highlighting_timeouts

This panel indicates syntax highlighting timeouts every 5m.

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### syntect-server: syntax_highlighting_panics

This panel indicates syntax highlighting panics every 5m.

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### syntect-server: syntax_highlighting_worker_deaths

This panel indicates syntax highlighter worker deaths every 5m.

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

### Syntect Server: Container monitoring (not available on server)

#### syntect-server: container_cpu_usage

This panel indicates container cpu usage total (1m average) across all cores by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#syntect-server-container-cpu-usage).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### syntect-server: container_memory_usage

This panel indicates container memory usage by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#syntect-server-container-memory-usage).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### syntect-server: container_missing

This panel indicates container missing.

This value is the number of times a container has not been seen for more than one minute. If you observe this
value change independent of deployment events (such as an upgrade), it could indicate pods are being OOM killed or terminated for some other reasons.

- **Kubernetes:**
	- Determine if the pod was OOM killed using `kubectl describe pod syntect-server` (look for `OOMKilled: true`) and, if so, consider increasing the memory limit in the relevant `Deployment.yaml`.
	- Check the logs before the container restarted to see if there are `panic:` messages or similar using `kubectl logs -p syntect-server`.
- **Docker Compose:**
	- Determine if the pod was OOM killed using `docker inspect -f '{{json .State}}' syntect-server` (look for `"OOMKilled":true`) and, if so, consider increasing the memory limit of the syntect-server container in `docker-compose.yml`.
	- Check the logs before the container restarted to see if there are `panic:` messages or similar using `docker logs syntect-server` (note this will include logs from the previous and currently running container).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

### Syntect Server: Provisioning indicators (not available on server)

#### syntect-server: provisioning_container_cpu_usage_long_term

This panel indicates container cpu usage total (90th percentile over 1d) across all cores by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#syntect-server-provisioning-container-cpu-usage-long-term).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### syntect-server: provisioning_container_memory_usage_long_term

This panel indicates container memory usage (1d maximum) by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#syntect-server-provisioning-container-memory-usage-long-term).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### syntect-server: provisioning_container_cpu_usage_short_term

This panel indicates container cpu usage total (5m maximum) across all cores by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#syntect-server-provisioning-container-cpu-usage-short-term).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### syntect-server: provisioning_container_memory_usage_short_term

This panel indicates container memory usage (5m maximum) by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#syntect-server-provisioning-container-memory-usage-short-term).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

### Syntect Server: Kubernetes monitoring (only available on Kubernetes)

#### syntect-server: pods_available_percentage

This panel indicates percentage pods available.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#syntect-server-pods-available-percentage).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

## Zoekt Index Server

<p class="subtitle">Indexes repositories and populates the search index.</p>

#### zoekt-indexserver: repos_assigned

This panel indicates total number of repos.

Sudden changes should be caused by indexing configuration changes.

<sub>*Managed by the [Sourcegraph Search team](https://about.sourcegraph.com/handbook/engineering/search).*</sub>

<br />

#### zoekt-indexserver: repos_priorities

This panel indicates total number of repos with priorities for ranking.

Sudden changes should be caused by indexing configuration changes.

<sub>*Managed by the [Sourcegraph Search team](https://about.sourcegraph.com/handbook/engineering/search).*</sub>

<br />

#### zoekt-indexserver: repo_index_state

This panel indicates indexing results over 5m (noop=no changes, empty=no branches to index).

A persistent failing state indicates some repositories cannot be indexed, perhaps due to size and timeouts.

<sub>*Managed by the [Sourcegraph Search team](https://about.sourcegraph.com/handbook/engineering/search).*</sub>

<br />

#### zoekt-indexserver: repo_index_success_speed

This panel indicates successful indexing durations.

Latency increases can indicate bottlenecks in the indexserver.

<sub>*Managed by the [Sourcegraph Search team](https://about.sourcegraph.com/handbook/engineering/search).*</sub>

<br />

#### zoekt-indexserver: repo_index_fail_speed

This panel indicates failed indexing durations.

Failures happening after a long time indicates timeouts.

<sub>*Managed by the [Sourcegraph Search team](https://about.sourcegraph.com/handbook/engineering/search).*</sub>

<br />

#### zoekt-indexserver: average_resolve_revision_duration

This panel indicates average resolve revision duration over 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#zoekt-indexserver-average-resolve-revision-duration).

<sub>*Managed by the [Sourcegraph Search team](https://about.sourcegraph.com/handbook/engineering/search).*</sub>

<br />

### Zoekt Index Server: Container monitoring (not available on server)

#### zoekt-indexserver: container_cpu_usage

This panel indicates container cpu usage total (1m average) across all cores by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#zoekt-indexserver-container-cpu-usage).

<sub>*Managed by the [Sourcegraph Search team](https://about.sourcegraph.com/handbook/engineering/search).*</sub>

<br />

#### zoekt-indexserver: container_memory_usage

This panel indicates container memory usage by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#zoekt-indexserver-container-memory-usage).

<sub>*Managed by the [Sourcegraph Search team](https://about.sourcegraph.com/handbook/engineering/search).*</sub>

<br />

#### zoekt-indexserver: container_missing

This panel indicates container missing.

This value is the number of times a container has not been seen for more than one minute. If you observe this
value change independent of deployment events (such as an upgrade), it could indicate pods are being OOM killed or terminated for some other reasons.

- **Kubernetes:**
	- Determine if the pod was OOM killed using `kubectl describe pod zoekt-indexserver` (look for `OOMKilled: true`) and, if so, consider increasing the memory limit in the relevant `Deployment.yaml`.
	- Check the logs before the container restarted to see if there are `panic:` messages or similar using `kubectl logs -p zoekt-indexserver`.
- **Docker Compose:**
	- Determine if the pod was OOM killed using `docker inspect -f '{{json .State}}' zoekt-indexserver` (look for `"OOMKilled":true`) and, if so, consider increasing the memory limit of the zoekt-indexserver container in `docker-compose.yml`.
	- Check the logs before the container restarted to see if there are `panic:` messages or similar using `docker logs zoekt-indexserver` (note this will include logs from the previous and currently running container).

<sub>*Managed by the [Sourcegraph Search team](https://about.sourcegraph.com/handbook/engineering/search).*</sub>

<br />

#### zoekt-indexserver: fs_io_operations

This panel indicates filesystem reads and writes rate by instance over 1h.

This value indicates the number of filesystem read and write operations by containers of this service.
When extremely high, this can indicate a resource usage problem, or can cause problems with the service itself, especially if high values or spikes correlate with {{CONTAINER_NAME}} issues.

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

### Zoekt Index Server: Provisioning indicators (not available on server)

#### zoekt-indexserver: provisioning_container_cpu_usage_long_term

This panel indicates container cpu usage total (90th percentile over 1d) across all cores by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#zoekt-indexserver-provisioning-container-cpu-usage-long-term).

<sub>*Managed by the [Sourcegraph Search team](https://about.sourcegraph.com/handbook/engineering/search).*</sub>

<br />

#### zoekt-indexserver: provisioning_container_memory_usage_long_term

This panel indicates container memory usage (1d maximum) by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#zoekt-indexserver-provisioning-container-memory-usage-long-term).

<sub>*Managed by the [Sourcegraph Search team](https://about.sourcegraph.com/handbook/engineering/search).*</sub>

<br />

#### zoekt-indexserver: provisioning_container_cpu_usage_short_term

This panel indicates container cpu usage total (5m maximum) across all cores by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#zoekt-indexserver-provisioning-container-cpu-usage-short-term).

<sub>*Managed by the [Sourcegraph Search team](https://about.sourcegraph.com/handbook/engineering/search).*</sub>

<br />

#### zoekt-indexserver: provisioning_container_memory_usage_short_term

This panel indicates container memory usage (5m maximum) by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#zoekt-indexserver-provisioning-container-memory-usage-short-term).

<sub>*Managed by the [Sourcegraph Search team](https://about.sourcegraph.com/handbook/engineering/search).*</sub>

<br />

### Zoekt Index Server: Kubernetes monitoring (only available on Kubernetes)

#### zoekt-indexserver: pods_available_percentage

This panel indicates percentage pods available.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#zoekt-indexserver-pods-available-percentage).

<sub>*Managed by the [Sourcegraph Search team](https://about.sourcegraph.com/handbook/engineering/search).*</sub>

<br />

## Zoekt Web Server

<p class="subtitle">Serves indexed search requests using the search index.</p>

#### zoekt-webserver: indexed_search_request_errors

This panel indicates indexed search request errors every 5m by code.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#zoekt-webserver-indexed-search-request-errors).

<sub>*Managed by the [Sourcegraph Search team](https://about.sourcegraph.com/handbook/engineering/search).*</sub>

<br />

### Zoekt Web Server: Container monitoring (not available on server)

#### zoekt-webserver: container_cpu_usage

This panel indicates container cpu usage total (1m average) across all cores by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#zoekt-webserver-container-cpu-usage).

<sub>*Managed by the [Sourcegraph Search team](https://about.sourcegraph.com/handbook/engineering/search).*</sub>

<br />

#### zoekt-webserver: container_memory_usage

This panel indicates container memory usage by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#zoekt-webserver-container-memory-usage).

<sub>*Managed by the [Sourcegraph Search team](https://about.sourcegraph.com/handbook/engineering/search).*</sub>

<br />

#### zoekt-webserver: container_missing

This panel indicates container missing.

This value is the number of times a container has not been seen for more than one minute. If you observe this
value change independent of deployment events (such as an upgrade), it could indicate pods are being OOM killed or terminated for some other reasons.

- **Kubernetes:**
	- Determine if the pod was OOM killed using `kubectl describe pod zoekt-webserver` (look for `OOMKilled: true`) and, if so, consider increasing the memory limit in the relevant `Deployment.yaml`.
	- Check the logs before the container restarted to see if there are `panic:` messages or similar using `kubectl logs -p zoekt-webserver`.
- **Docker Compose:**
	- Determine if the pod was OOM killed using `docker inspect -f '{{json .State}}' zoekt-webserver` (look for `"OOMKilled":true`) and, if so, consider increasing the memory limit of the zoekt-webserver container in `docker-compose.yml`.
	- Check the logs before the container restarted to see if there are `panic:` messages or similar using `docker logs zoekt-webserver` (note this will include logs from the previous and currently running container).

<sub>*Managed by the [Sourcegraph Search team](https://about.sourcegraph.com/handbook/engineering/search).*</sub>

<br />

#### zoekt-webserver: fs_io_operations

This panel indicates filesystem reads and writes rate by instance over 1h.

This value indicates the number of filesystem read and write operations by containers of this service.
When extremely high, this can indicate a resource usage problem, or can cause problems with the service itself, especially if high values or spikes correlate with {{CONTAINER_NAME}} issues.

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

### Zoekt Web Server: Provisioning indicators (not available on server)

#### zoekt-webserver: provisioning_container_cpu_usage_long_term

This panel indicates container cpu usage total (90th percentile over 1d) across all cores by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#zoekt-webserver-provisioning-container-cpu-usage-long-term).

<sub>*Managed by the [Sourcegraph Search team](https://about.sourcegraph.com/handbook/engineering/search).*</sub>

<br />

#### zoekt-webserver: provisioning_container_memory_usage_long_term

This panel indicates container memory usage (1d maximum) by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#zoekt-webserver-provisioning-container-memory-usage-long-term).

<sub>*Managed by the [Sourcegraph Search team](https://about.sourcegraph.com/handbook/engineering/search).*</sub>

<br />

#### zoekt-webserver: provisioning_container_cpu_usage_short_term

This panel indicates container cpu usage total (5m maximum) across all cores by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#zoekt-webserver-provisioning-container-cpu-usage-short-term).

<sub>*Managed by the [Sourcegraph Search team](https://about.sourcegraph.com/handbook/engineering/search).*</sub>

<br />

#### zoekt-webserver: provisioning_container_memory_usage_short_term

This panel indicates container memory usage (5m maximum) by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#zoekt-webserver-provisioning-container-memory-usage-short-term).

<sub>*Managed by the [Sourcegraph Search team](https://about.sourcegraph.com/handbook/engineering/search).*</sub>

<br />

## Prometheus

<p class="subtitle">Sourcegraph's all-in-one Prometheus and Alertmanager service.</p>

### Prometheus: Metrics

#### prometheus: prometheus_rule_eval_duration

This panel indicates average prometheus rule group evaluation duration over 10m by rule group.

A high value here indicates Prometheus rule evaluation is taking longer than expected.
It might indicate that certain rule groups are taking too long to evaluate, or Prometheus is underprovisioned.

Rules that Sourcegraph ships with are grouped under `/sg_config_prometheus`. [Custom rules are grouped under `/sg_prometheus_addons`](https://docs.sourcegraph.com/admin/observability/metrics#prometheus-configuration).

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#prometheus-prometheus-rule-eval-duration).

<sub>*Managed by the [Sourcegraph Distribution team](https://about.sourcegraph.com/handbook/engineering/distribution).*</sub>

<br />

#### prometheus: prometheus_rule_eval_failures

This panel indicates failed prometheus rule evaluations over 5m by rule group.

Rules that Sourcegraph ships with are grouped under `/sg_config_prometheus`. [Custom rules are grouped under `/sg_prometheus_addons`](https://docs.sourcegraph.com/admin/observability/metrics#prometheus-configuration).

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#prometheus-prometheus-rule-eval-failures).

<sub>*Managed by the [Sourcegraph Distribution team](https://about.sourcegraph.com/handbook/engineering/distribution).*</sub>

<br />

### Prometheus: Alerts

#### prometheus: alertmanager_notification_latency

This panel indicates alertmanager notification latency over 1m by integration.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#prometheus-alertmanager-notification-latency).

<sub>*Managed by the [Sourcegraph Distribution team](https://about.sourcegraph.com/handbook/engineering/distribution).*</sub>

<br />

#### prometheus: alertmanager_notification_failures

This panel indicates failed alertmanager notifications over 1m by integration.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#prometheus-alertmanager-notification-failures).

<sub>*Managed by the [Sourcegraph Distribution team](https://about.sourcegraph.com/handbook/engineering/distribution).*</sub>

<br />

### Prometheus: Internals

#### prometheus: prometheus_config_status

This panel indicates prometheus configuration reload status.

A `1` indicates Prometheus reloaded its configuration successfully.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#prometheus-prometheus-config-status).

<sub>*Managed by the [Sourcegraph Distribution team](https://about.sourcegraph.com/handbook/engineering/distribution).*</sub>

<br />

#### prometheus: alertmanager_config_status

This panel indicates alertmanager configuration reload status.

A `1` indicates Alertmanager reloaded its configuration successfully.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#prometheus-alertmanager-config-status).

<sub>*Managed by the [Sourcegraph Distribution team](https://about.sourcegraph.com/handbook/engineering/distribution).*</sub>

<br />

#### prometheus: prometheus_tsdb_op_failure

This panel indicates prometheus tsdb failures by operation over 1m by operation.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#prometheus-prometheus-tsdb-op-failure).

<sub>*Managed by the [Sourcegraph Distribution team](https://about.sourcegraph.com/handbook/engineering/distribution).*</sub>

<br />

#### prometheus: prometheus_target_sample_exceeded

This panel indicates prometheus scrapes that exceed the sample limit over 10m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#prometheus-prometheus-target-sample-exceeded).

<sub>*Managed by the [Sourcegraph Distribution team](https://about.sourcegraph.com/handbook/engineering/distribution).*</sub>

<br />

#### prometheus: prometheus_target_sample_duplicate

This panel indicates prometheus scrapes rejected due to duplicate timestamps over 10m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#prometheus-prometheus-target-sample-duplicate).

<sub>*Managed by the [Sourcegraph Distribution team](https://about.sourcegraph.com/handbook/engineering/distribution).*</sub>

<br />

### Prometheus: Container monitoring (not available on server)

#### prometheus: container_cpu_usage

This panel indicates container cpu usage total (1m average) across all cores by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#prometheus-container-cpu-usage).

<sub>*Managed by the [Sourcegraph Distribution team](https://about.sourcegraph.com/handbook/engineering/distribution).*</sub>

<br />

#### prometheus: container_memory_usage

This panel indicates container memory usage by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#prometheus-container-memory-usage).

<sub>*Managed by the [Sourcegraph Distribution team](https://about.sourcegraph.com/handbook/engineering/distribution).*</sub>

<br />

#### prometheus: container_missing

This panel indicates container missing.

This value is the number of times a container has not been seen for more than one minute. If you observe this
value change independent of deployment events (such as an upgrade), it could indicate pods are being OOM killed or terminated for some other reasons.

- **Kubernetes:**
	- Determine if the pod was OOM killed using `kubectl describe pod prometheus` (look for `OOMKilled: true`) and, if so, consider increasing the memory limit in the relevant `Deployment.yaml`.
	- Check the logs before the container restarted to see if there are `panic:` messages or similar using `kubectl logs -p prometheus`.
- **Docker Compose:**
	- Determine if the pod was OOM killed using `docker inspect -f '{{json .State}}' prometheus` (look for `"OOMKilled":true`) and, if so, consider increasing the memory limit of the prometheus container in `docker-compose.yml`.
	- Check the logs before the container restarted to see if there are `panic:` messages or similar using `docker logs prometheus` (note this will include logs from the previous and currently running container).

<sub>*Managed by the [Sourcegraph Distribution team](https://about.sourcegraph.com/handbook/engineering/distribution).*</sub>

<br />

### Prometheus: Provisioning indicators (not available on server)

#### prometheus: provisioning_container_cpu_usage_long_term

This panel indicates container cpu usage total (90th percentile over 1d) across all cores by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#prometheus-provisioning-container-cpu-usage-long-term).

<sub>*Managed by the [Sourcegraph Distribution team](https://about.sourcegraph.com/handbook/engineering/distribution).*</sub>

<br />

#### prometheus: provisioning_container_memory_usage_long_term

This panel indicates container memory usage (1d maximum) by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#prometheus-provisioning-container-memory-usage-long-term).

<sub>*Managed by the [Sourcegraph Distribution team](https://about.sourcegraph.com/handbook/engineering/distribution).*</sub>

<br />

#### prometheus: provisioning_container_cpu_usage_short_term

This panel indicates container cpu usage total (5m maximum) across all cores by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#prometheus-provisioning-container-cpu-usage-short-term).

<sub>*Managed by the [Sourcegraph Distribution team](https://about.sourcegraph.com/handbook/engineering/distribution).*</sub>

<br />

#### prometheus: provisioning_container_memory_usage_short_term

This panel indicates container memory usage (5m maximum) by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#prometheus-provisioning-container-memory-usage-short-term).

<sub>*Managed by the [Sourcegraph Distribution team](https://about.sourcegraph.com/handbook/engineering/distribution).*</sub>

<br />

### Prometheus: Kubernetes monitoring (only available on Kubernetes)

#### prometheus: pods_available_percentage

This panel indicates percentage pods available.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#prometheus-pods-available-percentage).

<sub>*Managed by the [Sourcegraph Distribution team](https://about.sourcegraph.com/handbook/engineering/distribution).*</sub>

<br />

## Executor Queue

<p class="subtitle">Coordinates the executor work queues.</p>

### Executor Queue: Code intelligence queue

#### executor-queue: codeintel_queue_size

This panel indicates queue size.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#executor-queue-codeintel-queue-size).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### executor-queue: codeintel_queue_growth_rate

This panel indicates queue growth rate over 30m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#executor-queue-codeintel-queue-growth-rate).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### executor-queue: codeintel_job_errors

This panel indicates job errors every 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#executor-queue-codeintel-job-errors).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### executor-queue: codeintel_active_executors

This panel indicates active executors processing codeintel jobs.

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### executor-queue: codeintel_active_jobs

This panel indicates active jobs.

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

### Executor Queue: Stores and clients

#### executor-queue: codeintel_workerstore_99th_percentile_duration

This panel indicates 99th percentile successful worker store operation duration over 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#executor-queue-codeintel-workerstore-99th-percentile-duration).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### executor-queue: codeintel_workerstore_errors

This panel indicates worker store errors every 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#executor-queue-codeintel-workerstore-errors).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

### Executor Queue: Database connections

#### executor-queue: max_open_conns

This panel indicates maximum open.

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### executor-queue: open_conns

This panel indicates established.

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### executor-queue: in_use

This panel indicates used.

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### executor-queue: idle

This panel indicates idle.

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### executor-queue: mean_blocked_seconds_per_conn_request

This panel indicates mean blocked seconds per conn request.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#executor-queue-mean-blocked-seconds-per-conn-request).

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### executor-queue: closed_max_idle

This panel indicates closed by SetMaxIdleConns.

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### executor-queue: closed_max_lifetime

This panel indicates closed by SetConnMaxLifetime.

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

#### executor-queue: closed_max_idle_time

This panel indicates closed by SetConnMaxIdleTime.

<sub>*Managed by the [Sourcegraph Core application team](https://about.sourcegraph.com/handbook/engineering/core-application).*</sub>

<br />

### Executor Queue: Internal service requests

#### executor-queue: frontend_internal_api_error_responses

This panel indicates frontend-internal API error responses every 5m by route.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#executor-queue-frontend-internal-api-error-responses).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

### Executor Queue: Container monitoring (not available on server)

#### executor-queue: container_cpu_usage

This panel indicates container cpu usage total (1m average) across all cores by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#executor-queue-container-cpu-usage).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### executor-queue: container_memory_usage

This panel indicates container memory usage by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#executor-queue-container-memory-usage).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### executor-queue: container_missing

This panel indicates container missing.

This value is the number of times a container has not been seen for more than one minute. If you observe this
value change independent of deployment events (such as an upgrade), it could indicate pods are being OOM killed or terminated for some other reasons.

- **Kubernetes:**
	- Determine if the pod was OOM killed using `kubectl describe pod executor-queue` (look for `OOMKilled: true`) and, if so, consider increasing the memory limit in the relevant `Deployment.yaml`.
	- Check the logs before the container restarted to see if there are `panic:` messages or similar using `kubectl logs -p executor-queue`.
- **Docker Compose:**
	- Determine if the pod was OOM killed using `docker inspect -f '{{json .State}}' executor-queue` (look for `"OOMKilled":true`) and, if so, consider increasing the memory limit of the executor-queue container in `docker-compose.yml`.
	- Check the logs before the container restarted to see if there are `panic:` messages or similar using `docker logs executor-queue` (note this will include logs from the previous and currently running container).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

### Executor Queue: Provisioning indicators (not available on server)

#### executor-queue: provisioning_container_cpu_usage_long_term

This panel indicates container cpu usage total (90th percentile over 1d) across all cores by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#executor-queue-provisioning-container-cpu-usage-long-term).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### executor-queue: provisioning_container_memory_usage_long_term

This panel indicates container memory usage (1d maximum) by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#executor-queue-provisioning-container-memory-usage-long-term).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### executor-queue: provisioning_container_cpu_usage_short_term

This panel indicates container cpu usage total (5m maximum) across all cores by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#executor-queue-provisioning-container-cpu-usage-short-term).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### executor-queue: provisioning_container_memory_usage_short_term

This panel indicates container memory usage (5m maximum) by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#executor-queue-provisioning-container-memory-usage-short-term).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

### Executor Queue: Golang runtime monitoring

#### executor-queue: go_goroutines

This panel indicates maximum active goroutines.

A high value here indicates a possible goroutine leak.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#executor-queue-go-goroutines).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### executor-queue: go_gc_duration_seconds

This panel indicates maximum go garbage collection duration.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#executor-queue-go-gc-duration-seconds).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

### Executor Queue: Kubernetes monitoring (only available on Kubernetes)

#### executor-queue: pods_available_percentage

This panel indicates percentage pods available.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#executor-queue-pods-available-percentage).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

## Precise Code Intel Indexer

<p class="subtitle">Executes jobs from the "codeintel" work queue.</p>

### Precise Code Intel Indexer: Executor

#### precise-code-intel-indexer: codeintel_job_99th_percentile_duration

This panel indicates 99th percentile successful job duration over 5m.

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### precise-code-intel-indexer: codeintel_active_handlers

This panel indicates active handlers processing jobs.

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### precise-code-intel-indexer: codeintel_job_errors

This panel indicates job errors every 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#precise-code-intel-indexer-codeintel-job-errors).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

### Precise Code Intel Indexer: Stores and clients

#### precise-code-intel-indexer: executor_apiclient_99th_percentile_duration

This panel indicates 99th percentile successful API request duration over 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#precise-code-intel-indexer-executor-apiclient-99th-percentile-duration).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### precise-code-intel-indexer: executor_apiclient_errors

This panel indicates aPI errors every 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#precise-code-intel-indexer-executor-apiclient-errors).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

### Precise Code Intel Indexer: Commands

#### precise-code-intel-indexer: executor_setup_command_99th_percentile_duration

This panel indicates 99th percentile successful setup command duration over 5m.

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### precise-code-intel-indexer: executor_setup_command_errors

This panel indicates setup command errors every 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#precise-code-intel-indexer-executor-setup-command-errors).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### precise-code-intel-indexer: executor_exec_command_99th_percentile_duration

This panel indicates 99th percentile successful exec command duration over 5m.

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### precise-code-intel-indexer: executor_exec_command_errors

This panel indicates exec command errors every 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#precise-code-intel-indexer-executor-exec-command-errors).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### precise-code-intel-indexer: executor_teardown_command_99th_percentile_duration

This panel indicates 99th percentile successful teardown command duration over 5m.

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### precise-code-intel-indexer: executor_teardown_command_errors

This panel indicates teardown command errors every 5m.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#precise-code-intel-indexer-executor-teardown-command-errors).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

### Precise Code Intel Indexer: Container monitoring (not available on server)

#### precise-code-intel-indexer: container_cpu_usage

This panel indicates container cpu usage total (1m average) across all cores by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#precise-code-intel-indexer-container-cpu-usage).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### precise-code-intel-indexer: container_memory_usage

This panel indicates container memory usage by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#precise-code-intel-indexer-container-memory-usage).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### precise-code-intel-indexer: container_missing

This panel indicates container missing.

This value is the number of times a container has not been seen for more than one minute. If you observe this
value change independent of deployment events (such as an upgrade), it could indicate pods are being OOM killed or terminated for some other reasons.

- **Kubernetes:**
	- Determine if the pod was OOM killed using `kubectl describe pod precise-code-intel-worker` (look for `OOMKilled: true`) and, if so, consider increasing the memory limit in the relevant `Deployment.yaml`.
	- Check the logs before the container restarted to see if there are `panic:` messages or similar using `kubectl logs -p precise-code-intel-worker`.
- **Docker Compose:**
	- Determine if the pod was OOM killed using `docker inspect -f '{{json .State}}' precise-code-intel-worker` (look for `"OOMKilled":true`) and, if so, consider increasing the memory limit of the precise-code-intel-worker container in `docker-compose.yml`.
	- Check the logs before the container restarted to see if there are `panic:` messages or similar using `docker logs precise-code-intel-worker` (note this will include logs from the previous and currently running container).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

### Precise Code Intel Indexer: Provisioning indicators (not available on server)

#### precise-code-intel-indexer: provisioning_container_cpu_usage_long_term

This panel indicates container cpu usage total (90th percentile over 1d) across all cores by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#precise-code-intel-indexer-provisioning-container-cpu-usage-long-term).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### precise-code-intel-indexer: provisioning_container_memory_usage_long_term

This panel indicates container memory usage (1d maximum) by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#precise-code-intel-indexer-provisioning-container-memory-usage-long-term).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### precise-code-intel-indexer: provisioning_container_cpu_usage_short_term

This panel indicates container cpu usage total (5m maximum) across all cores by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#precise-code-intel-indexer-provisioning-container-cpu-usage-short-term).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### precise-code-intel-indexer: provisioning_container_memory_usage_short_term

This panel indicates container memory usage (5m maximum) by instance.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#precise-code-intel-indexer-provisioning-container-memory-usage-short-term).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

### Precise Code Intel Indexer: Golang runtime monitoring

#### precise-code-intel-indexer: go_goroutines

This panel indicates maximum active goroutines.

A high value here indicates a possible goroutine leak.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#precise-code-intel-indexer-go-goroutines).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

#### precise-code-intel-indexer: go_gc_duration_seconds

This panel indicates maximum go garbage collection duration.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#precise-code-intel-indexer-go-gc-duration-seconds).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

### Precise Code Intel Indexer: Kubernetes monitoring (only available on Kubernetes)

#### precise-code-intel-indexer: pods_available_percentage

This panel indicates percentage pods available.

> NOTE: Alerts related to this panel are documented in the [alert solutions reference](./alert_solutions.md#precise-code-intel-indexer-pods-available-percentage).

<sub>*Managed by the [Sourcegraph Code-intelligence team](https://about.sourcegraph.com/handbook/engineering/code-intelligence).*</sub>

<br />

