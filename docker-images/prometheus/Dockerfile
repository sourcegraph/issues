# sourcegraph/prometheus - learn more about this image in https://docs.sourcegraph.com/dev/background-information/observability/prometheus

# Build monitoring definitions
FROM sourcegraph/alpine-3.12:99422_2021-06-16_cfa0e09@sha256:ffc2ad13f9d44afe8e20fd9d4f9a23af6851d96ccf2ff3eb149ea93fc0420b65
RUN mkdir -p '/generated/prometheus'
COPY ./.bin/monitoring-generator /bin/monitoring-generator
RUN PROMETHEUS_DIR='/generated/prometheus' GRAFANA_DIR='' DOCS_DIR='' NO_PRUNE=true /bin/monitoring-generator
RUN ls '/generated/prometheus'

# To upgrade Prometheus or Alertmanager, see https://docs.sourcegraph.com/dev/background-information/observability/prometheus#upgrading-prometheus-or-alertmanager
FROM prom/prometheus:v2.23.0@sha256:3d4983246977fd7fe398fd65e90c46a68ae7178cfe70912fb689d9aca5468975 AS prom_upstream
FROM prom/alertmanager:v0.21.0@sha256:913293083cb14085bfc01018bb30d1dcbbc9ed197ae21ef2ca917b0d29265198 AS am_upstream

# Prepare final image
# hadolint ignore=DL3007
FROM quay.io/prometheus/busybox-linux-amd64:latest@sha256:de4af55df1f648a334e16437c550a2907e0aed4f0b0edf454b0b215a9349bdbb

# Should reflect versions above
LABEL com.sourcegraph.prometheus.version=v2.23.0
LABEL com.sourcegraph.alertmanager.version=v0.21.0

ARG COMMIT_SHA="unknown"
ARG DATE="unknown"
ARG VERSION="unknown"

LABEL org.opencontainers.image.revision=${COMMIT_SHA}
LABEL org.opencontainers.image.created=${DATE}
LABEL org.opencontainers.image.version=${VERSION}
LABEL org.opencontainers.image.url=https://sourcegraph.com/
LABEL org.opencontainers.image.source=https://github.com/sourcegraph/sourcegraph/
LABEL org.opencontainers.image.documentation=https://docs.sourcegraph.com/

# Prometheus - extended from https://github.com/prometheus/prometheus/blob/VERSION/Dockerfile
# Check the upstream image (replacing VERSION with the appropriate Prometheus version) when upgrading
COPY --from=prom_upstream /bin/prometheus /bin/prometheus
COPY --from=prom_upstream /bin/promtool /bin/promtoool
COPY --from=prom_upstream /etc/prometheus/prometheus.yml /etc/prometheus/prometheus.yml
COPY --from=prom_upstream /usr/share/prometheus/console_libraries/ /usr/share/prometheus/console_libraries/
COPY --from=prom_upstream /usr/share/prometheus/consoles/ /usr/share/prometheus/consoles/
COPY --from=prom_upstream /LICENSE /LICENSE
COPY --from=prom_upstream /NOTICE /NOTICE
# hadolint ignore=DL3010
COPY --from=prom_upstream /npm_licenses.tar.bz2 /npm_licenses.tar.bz2

# Alertmanager - extended from https://github.com/prometheus/alertmanager/blob/VERSION/Dockerfile
# Check the upstream image (replacing VERSION with the appropriate Prometheus version) when upgrading
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

COPY ./.bin/prom-wrapper /bin/prom-wrapper
COPY ./prometheus.sh /prometheus.sh
COPY ./alertmanager.sh /alertmanager.sh

# Copy config
COPY --from=monitoring_builder /generated/prometheus/* /sg_config_prometheus/
COPY config/*_rules.yml /sg_config_prometheus/
COPY config/prometheus.yml /sg_config_prometheus/
COPY config/alertmanager.yml /sg_config_prometheus/

ENTRYPOINT ["/bin/prom-wrapper"]
# Note that upstream's 'VOLUME' directive was deliberately removed. Including it makes it impossible
# to chmod the directory to our 'sourcegraph' user.
WORKDIR    /prometheus
# Prometheus is reverse-proxied from 9092 to 9090
EXPOSE     9090
