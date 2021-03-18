# Prometheus configuration

This directory contains configuration for the [Prometheus](https://prometheus.io/) metrics deployment.

This directory is mounted into the `prometheus` container. After making your changes to this directory,
simply `docker restart prometheus` for your changes to take effect (depending on your change, Prometheus
may respond to it as soon as you save the file).

(c) Copyright Sourcegraph 2012-2023.