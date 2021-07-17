# Running Sourcegraph executors on bare servers

<style>
@import url(draft.css);
</style>

<div id="draft"><span>DRAFT</span></div>

In this model, changesets are computed in Docker containers run directly on the server.

## Pros

* Deployment is simple.
* I/O performance can be extremely high if physically attached storage is used.

## Cons

* If required, scaling must be managed outside the container environment through something like EC2 auto scaling.
* Resource limits are harder to apply on individual changeset steps.

## Installation

<!--
aharvey: I originally had a few paragraphs on packaged RPM/DEB deployments. I think we might _eventually_ have demand for that because it's easier to manage upgrades, but it's far enough down the road that I don't want to get into it here. What might be a middle ground before that would be shipping AMIs with extremely bare bones Debian images with just Docker and the executor preconfigured in systemd.

For now I've also skipped any mention of Windows or macOS, but there's nothing stopping us from shipping those. (Although it's probably of relatively little use, given the hard Docker requirement; why not just run Linux natively?)
-->

The Sourcegraph executor is a service, written in Go, that is shipped as a static binary for Linux. The service must be configured to start after Docker, and is [configured through environment variables](configuration.md).

On a systemd-based distribution, the following steps will install the executor:

<ol>
<li>Download the executor: https://github.com/sourcegraph/sourcegraph/releases/XXX/sourcegraph-executor.tar.gz

<li>Extract it to a suitable location using tar; eg: `tar -C /opt/sourcegraph -zxf sourcegraph-executor.tar.gz`

<li>Add a systemd service to start the executor; for example by creating the below unit in `/etc/systemd/system/sourcegraph-executor.service`:

</ol>

```ini
[Unit]
Description=Sourcegraph executor
After=docker.service
Requires=docker.socket

[Service]
Type=simple
ExecStart=/opt/sourcegraph/bin/executor
EnvironmentFile=/opt/sourcegraph/etc/executor.env

[Install]
WantedBy=multi-user.target
```
<ol start="4">
<li>Configure the executor by setting [the required environment variables](configuration.md) in the environment file (`/opt/sourcegraph/etc/executor.env`, in this example). For example:
</ol>

```bash
EXECUTOR_QUEUE_NAME=batches
EXECUTOR_FRONTEND_URL=https://sourcegraph.at.my.company/
EXECUTOR_FRONTEND_USERNAME=executor
EXECUTOR_FRONTEND_PASSWORD=a-highly-secure-password
EXECUTOR_BACKEND=docker
EXECUTOR_MAX_NUM_JOBS=4
```

<ol start="5">
<li>Enable the executor service: `systemctl enable sourcegraph-executor.service`

<li>Start the executor service: `systemctl start sourcegraph-executor`
</ol>
