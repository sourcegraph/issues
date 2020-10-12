# Observability

Sourcegraph is designed to meet enterprise production readiness criteria. A key pillar of production
readiness is the ability to observe, monitor, and analyze the health and state of the
system.

> NOTE: If you're using the [Kubernetes cluster deployment
> option](https://github.com/sourcegraph/deploy-sourcegraph), see the [Kubernetes cluster
> administrator
> guide](https://github.com/sourcegraph/deploy-sourcegraph/blob/master/docs/admin-guide.md) for more
> information.

Sourcegraph ships with a number of observability tools and capabilities:

* [Metrics and dashboards](metrics.md) via Prometheus and Grafana
* [Tracing](tracing.md)
* [Alerting](alerting.md)
  * [Alerting: custom consumption](alerting_custom_consumption.md)
* [Logs](#logs)
* [Health checks](#health-checks)
* [Other tools](#other-tools)

If you are investigating a specific production issue, consult the [troubleshooting guide](troubleshooting.md).

## Logs

A Sourcegraph service's log level is configured via the environment variable `SRC_LOG_LEVEL`. The valid values (from most to least verbose) are:

* `dbug`: Debug. Output all logs. Default in cluster deployments.
* `info`: Informational.
* `warn`: Warning. Default in Docker deployments.
* `eror`: Error.
* `crit`: Critical.

If you are having issues with repository syncing, view the output of `repo-updater`'s logs.

## Health checks

An application health check status endpoint is available at the URL path `/healthz`. It returns HTTP 200 if and only if the main frontend server and databases (PostgreSQL and Redis) are available.

The [Kubernetes cluster deployment option](https://github.com/sourcegraph/deploy-sourcegraph) ships with comprehensive health checks for each Kubernetes deployment.

## Other tools

- [Sentry](https://sentry.io) error reporting, configured via the `sentry` property in [site configuration](../config/site_config.md)
- [Go net/trace](#viewing-go-net-trace-information)
- [Honeycomb](https://honeycomb.io/)

## Support

For help configuring monitoring and tracing on your Sourcegraph instance, use our [public issue
tracker](https://github.com/sourcegraph/issues/issues).
