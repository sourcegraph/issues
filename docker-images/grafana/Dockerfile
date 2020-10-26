FROM golang:1.13-alpine@sha256:ed003971a4809c9ae45afe2d318c24b9e3f6b30864a322877c69a46c504d852c AS builder

RUN apk add --no-cache bash=5.0.11-r1 git=2.24.3-r0 openssh=8.1_p1-r0 build-base=0.5-r1

# build the jsonnet dashboards
WORKDIR /opt
RUN git clone --branch v0.14.0 https://github.com/google/jsonnet.git
WORKDIR /opt/jsonnet
RUN make
RUN mkdir /dashboards
COPY jsonnet /dashboards
WORKDIR /dashboards
ENV PATH="/opt/jsonnet:${PATH}"
RUN ./build.sh

# build dashboards from monitoring declarations
COPY monitoring /go/src/monitoring-generator
WORKDIR /go/src/monitoring-generator
RUN CGO_ENABLED=0 go build -o /go/bin/monitoring-generator
RUN mkdir -p /generated/grafana
RUN DOC_SOLUTIONS_FILE='' PROMETHEUS_DIR='' GRAFANA_DIR=/generated/grafana /go/bin/monitoring-generator

# when upgrading the Grafana version, please refer to https://about.sourcegraph.com/handbook/engineering/distribution/observability/monitoring#upgrading-grafana
FROM grafana/grafana:7.2.0@sha256:b1c8a29dc7972bd5773bce5e564dd4bdcd96723dfeee6f5b754a2b3039e39dcb as production
LABEL com.sourcegraph.grafana.version=7.0.3

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
ADD --chown=grafana:grafana config /sg_config_grafana
COPY home.json /usr/share/grafana/public/dashboards/home.json

COPY --from=builder /dashboards/*.json /sg_config_grafana/provisioning/dashboards/sourcegraph_internal/
COPY --from=builder /generated/grafana/* /sg_config_grafana/provisioning/dashboards/sourcegraph/

# hadolint ignore=DL3020
ADD --chown=grafana:grafana entry.sh /

USER root
# hadolint ignore=DL3018
RUN apk add --no-cache curl

EXPOSE 3370
USER grafana
ENTRYPOINT ["/entry.sh"]
