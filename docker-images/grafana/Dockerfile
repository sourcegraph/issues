# sourcegraph/grafana - learn more about this image in https://docs.sourcegraph.com/dev/background-information/observability/grafana

# Build monitoring definitions
FROM us.gcr.io/sourcegraph-dev/alpine:f42c99f6acf540d14c671ef7a1b2baa6b8be6a68_99360_candidate AS monitoring_builder
RUN mkdir -p '/generated/grafana'
COPY ./.bin/monitoring-generator /bin/monitoring-generator
RUN GRAFANA_DIR='/generated/grafana' PROMETHEUS_DIR='' DOCS_DIR='' NO_PRUNE=true /bin/monitoring-generator
RUN ls '/generated/grafana'

# Prepare final image
# when upgrading the Grafana version, please refer to https://docs.sourcegraph.com/dev/background-information/observability/grafana#upgrading-grafana
# DO NOT UPGRADE to AGPL Grafana without consulting Stephen+legal, Grafana >= 8.0 is AGPLv3 Licensed
# See https://docs.google.com/document/d/1nSmz1ChL_rBvX8FAKTB-CNzgcff083sUlIpoXEz6FHE/edit#heading=h.69clsrno4211
FROM grafana/grafana:7.5.7@sha256:c1134ea4c5a6bac38ddc246bd07eb1ff35680ea6cdda63a7bc30773e2f2a6085 as production
LABEL com.sourcegraph.grafana.version=7.5.7

ARG COMMIT_SHA="unknown"
ARG DATE="unknown"
ARG VERSION="unknown"

LABEL org.opencontainers.image.revision=${COMMIT_SHA}
LABEL org.opencontainers.image.created=${DATE}
LABEL org.opencontainers.image.version=${VERSION}
LABEL org.opencontainers.image.url=https://sourcegraph.com/
LABEL org.opencontainers.image.source=https://github.com/sourcegraph/sourcegraph/
LABEL org.opencontainers.image.documentation=https://docs.sourcegraph.com/

# hadolint ignore=DL3020
ADD config /sg_config_grafana
COPY home.json /usr/share/grafana/public/dashboards/home.json

COPY --from=monitoring_builder /generated/grafana/* /sg_config_grafana/provisioning/dashboards/sourcegraph/

# hadolint ignore=DL3020
ADD entry.sh /

USER root

EXPOSE 3370
USER grafana
ENTRYPOINT ["/entry.sh"]
