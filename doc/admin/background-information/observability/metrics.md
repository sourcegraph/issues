# Metrics

Sourcegraph uses [Prometheus](https://prometheus.io/) for metrics and [Grafana](https://grafana.com) for metrics dashboards.

If you're using the [Kubernetes cluster deployment
option](https://github.com/sourcegraph/deploy-sourcegraph), see the [Prometheus
README](https://github.com/sourcegraph/deploy-sourcegraph/blob/master/configure/prometheus/README.md)
for more information.

## Prometheus

Prometheus is a monitoring tool that collects application- and system-level metrics over time and
makes these accessible through a query language and simple UI.

### Accessing Prometheus

Most of the time, Sourcegraph site admins will monitor key metrics through the Grafana UI, rather
than through Prometheus directly. Grafana provides the dashboards that monitor the standard metrics
that indicate the health of the instance. Only if an admin wants to write a novel metrics formula or
query do they need to access the Prometheus UI.

If you are using single-container Sourcegraph, you will need to restart the Sourcegraph container
with a flag `--publish 9090:9090` in the `docker run` command. Subsequently, you can access
Prometheus at http://localhost:9090.

If you are using the Sourcegraph Kubernetes Cluster, port-forward the Prometheus service:

```
kubectl port-forward svc/prometheus 9090:30090
```

#### Configuration

Sourcegraph runs a slightly customized image of Prometheus, which packages a standard Prometheus
installation together with rules files and target files tailored to Sourcegraph.

A directory can be mounted at `/sg_prometheus_add_ons`. It can contain additional config files of two types:
  - rule files which must have the suffix `_rules.yml` in their filename (ie `gitserver_rules.yml`)
  - target files which must have the suffix `_targets.yml` in their filename (ie `local_targets.yml`)

[Rule files](https://prometheus.io/docs/prometheus/latest/configuration/recording_rules/) 
and [target files](https://prometheus.io/docs/guides/file-sd/) must use the latest Prometheus 2.x syntax.  

The environment variable `PROMETHEUS_ADDITIONAL_FLAGS` can be used to pass on additional flags to the `prometheus` executable running in the container.

## Grafana

Site admins can view the monitoring dashboards on a Sourcegraph instance:

1. Go to **User menu > Site admin**.
1. Open the **Monitoring** page (left sidebar). The URL is
   `https://sourcegraph.example.com/-/debug/grafana/?orgId=1`.
1. Read the [Sourcegraph Grafana dashboard descriptions](dashboards.md) before exploring
   the dashboards.

> NOTE: There is a [known issue](https://github.com/sourcegraph/sourcegraph/issues/6075) where
> attempting to edit a dashboard will result in a 403 response with "invalid CSRF token". As a
> workaround, site admins can connect to Grafana directly (described below) to edit the dashboards.

### Accessing Grafana directly

Follow the instructions below to access Grafana directly, and add, modify and delete your own dashboards and panels.

#### Kubernetes

If you're using the [Kubernetes cluster deployment
option](https://github.com/sourcegraph/deploy-sourcegraph), you can access Grafana directly using
Kubernetes port forwarding to your local machine:


```
kubectl port-forward svc/grafana 3370:30070
```

Now visit http://localhost:3370/-/debug/grafana.

#### Single-container server deployments

For simplicity, Grafana does not require authentication, as the port binding of 3370 is restricted to connections from localhost only.

Therefore, if accessing Grafana locally, the URL will be http://localhost:3370/-/debug/grafana. If Sourcegraph is deployed to a remote server, then access via an SSH tunnel using a tool
such as [sshuttle](https://github.com/sshuttle/sshuttle) is required to establish a secure connection to Grafana.
To access the remote server using `sshuttle` from your local machine:

```bash
sshuttle -r user@host 0/0
```

Then simply visit http://host:3370 in your browser.

#### Configuration

Sourcegraph runs a slightly customized image of Grafana, which includes a standard Grafana
installation initialized with Sourcegraph-specific dashboard definitions.

> NOTE: Our Grafana instance runs in anonymous mode with all authentication turned off. Please be careful when exposing it to external traffic.

A directory containing dashboard JSON specifications can be mounted in the Docker container at
`/sg_grafana_additional_dashboards`. Changes to files in that directory will be detected
automatically while Grafana is running.

More behavior can be controlled with
[environmental variables](https://grafana.com/docs/installation/configuration/).

### FAQ

#### Can I consume Sourcegraph's Prometheus metrics in my own monitoring system (Datadog, New Relic, etc.)?

It is technically possible to consume all of Sourcegraph's Prometheus metrics in any external monitoring system that supports Prometheus scraping (both Datadog and New Relic support this). However, we would advise against it because Sourcegraph is a very complex system and defining all of the alerting thresholds and rules that are needed to ensure Sourcegraph is healthy is very tedious and changes with each release of Sourcegraph.

One of the primary benefits of using Sourcegraph's builtin Prometheus and Grafana monitoring is that you get builtin dashboards and alerting thresholds out-of-the-box, and as Sourcegraph's internals change with each update you can rest assured that the metrics and information you are monitoring is up-to-date.

Most commonly [Sourcegraph's monitoring is configured to send alerts to your own PagerDuty, Slack, email, etc.](alerting.md). Less common approaches include:

- Using [the HTTP API to query which alerts are firing in Sourcegraph](alerting_custom_consumption.md) in order to pipe that information into your own monitoring platform.
- Using [Prometheus federation](https://prometheus.io/docs/prometheus/latest/federation/) to pull in Sourcegraph's metrics into your own Prometheus instance, which includes our [high-level alerting metrics](metrics_guide.md) which we define via baked-in Prometheus rules.
