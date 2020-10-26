FROM golang:1.13-alpine@sha256:ed003971a4809c9ae45afe2d318c24b9e3f6b30864a322877c69a46c504d852c AS builder

COPY monitoring /go/src/monitoring-generator
WORKDIR /go/src/monitoring-generator
RUN CGO_ENABLED=0 go build -o /go/bin/monitoring-generator
RUN mkdir -p /generated/prometheus
RUN DOC_SOLUTIONS_FILE='' PROMETHEUS_DIR=/generated/prometheus GRAFANA_DIR='' /go/bin/monitoring-generator

# This Dockerfile is adapted from https://github.com/prometheus/prometheus/blob/v2.15.2/Dockerfile with
# our own customizations.
#
# When upgrading prometheus, be sure to check https://github.com/prometheus/prometheus/blob/master/Dockerfile
# (replacing master with the version number) to see if any changes (added/removed files, etc.) need to be
# synchronized with this derivative Dockerfile. Also make sure to update the appropriate com.sourcegraph.*.version
# labels below.
FROM prom/prometheus:v2.21.0@sha256:f3ada803723ccbc443ebea19f7ab24d3323def496e222134bf9ed54ae5b787bd AS prom_upstream

# Similarly to above, this Dockerfile also extends https://github.com/prometheus/alertmanager/blob/v0.21.0/Dockerfile
# When upgrading alertmanager, make sure to make similar checks to the one above.
FROM prom/alertmanager:v0.21.0@sha256:913293083cb14085bfc01018bb30d1dcbbc9ed197ae21ef2ca917b0d29265198 AS am_upstream

# hadolint ignore=DL3007
FROM quay.io/prometheus/busybox-linux-amd64:latest@sha256:248b7ec76e03e6b4fbb796fc3cdd2f91dad45546a6d7dee61c322475e0e8a08f

ARG COMMIT_SHA="unknown"
ARG DATE="unknown"
ARG VERSION="unknown"

LABEL org.opencontainers.image.revision=${COMMIT_SHA}
LABEL org.opencontainers.image.created=${DATE}
LABEL org.opencontainers.image.version=${VERSION}
LABEL org.opencontainers.image.url=https://sourcegraph.com/
LABEL org.opencontainers.image.source=https://github.com/sourcegraph/sourcegraph/
LABEL org.opencontainers.image.documentation=https://docs.sourcegraph.com/

LABEL com.sourcegraph.prometheus.version=v2.16.0
LABEL com.sourcegraph.alertmanager.version=v0.21.0

# Prometheus
COPY --from=prom_upstream /bin/prometheus /bin/prometheus
COPY --from=prom_upstream /bin/promtool /bin/promtoool
COPY --from=prom_upstream /etc/prometheus/prometheus.yml /etc/prometheus/prometheus.yml
COPY --from=prom_upstream /usr/share/prometheus/console_libraries/ /usr/share/prometheus/console_libraries/
COPY --from=prom_upstream /usr/share/prometheus/consoles/ /usr/share/prometheus/consoles/
COPY --from=prom_upstream /LICENSE /LICENSE
COPY --from=prom_upstream /NOTICE /NOTICE
# hadolint ignore=DL3010
COPY --from=prom_upstream /npm_licenses.tar.bz2 /npm_licenses.tar.bz2

# Alertmanager
COPY --from=am_upstream /bin/alertmanager /bin/alertmanager

RUN ln -s /usr/share/prometheus/console_libraries /usr/share/prometheus/consoles/ /etc/prometheus/

# TODO(uwe): remove "USER root" line once https://github.com/prometheus/prometheus/issues/3441 is resolved
#
# This is needed currently because the upstream image has us running as "nobody"
# which cannot create the sourcegraph user below.
USER root
# Add the sourcegraph group, user, and create the home directory.
#
# We use a static GID/UID assignment to ensure files can be chown'd to this
# user on the host machine (where this user does not exist).
# See https://github.com/sourcegraph/sourcegraph/issues/1884
#
# Note: This mirrors what we do in e.g. our base alpine image: https://github.com/sourcegraph/sourcegraph/blob/main/docker-images/alpine/Dockerfile#L10-L15
RUN addgroup -g 101 -S sourcegraph && adduser -u 100 -S -G sourcegraph -h /home/sourcegraph sourcegraph
RUN mkdir -p /prometheus && chown -R sourcegraph:sourcegraph /prometheus
RUN mkdir -p /alertmanager && chown -R sourcegraph:sourcegraph /alertmanager
USER sourcegraph

COPY --from=builder /generated/prometheus/* /sg_config_prometheus/
COPY ./.bin/prom-wrapper /bin/prom-wrapper
COPY ./prometheus.sh /prometheus.sh
COPY ./alertmanager.sh /alertmanager.sh
COPY config/*_rules.yml /sg_config_prometheus/
COPY config/prometheus.yml /sg_config_prometheus/
COPY config/alertmanager.yml /sg_config_prometheus/

ENTRYPOINT ["/bin/prom-wrapper"]
# Note that upstream's 'VOLUME' directive was deliberately removed. Including it makes it impossible
# to chmod the directory to our 'sourcegraph' user.
WORKDIR    /prometheus
# Prometheus is reverse-proxied from 9092 to 9090
EXPOSE     9090
