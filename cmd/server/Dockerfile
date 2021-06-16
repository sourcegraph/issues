FROM us.gcr.io/sourcegraph-dev/alpine:f42c99f6acf540d14c671ef7a1b2baa6b8be6a68_99360_candidate as libsqlite3-pcre

COPY libsqlite3-pcre-install-alpine.sh /libsqlite3-pcre-install-alpine.sh
RUN /libsqlite3-pcre-install-alpine.sh

# Install p4 CLI (keep this up to date with cmd/gitserver/Dockerfile)
FROM us.gcr.io/sourcegraph-dev/alpine:f42c99f6acf540d14c671ef7a1b2baa6b8be6a68_99360_candidate AS p4cli

# hadolint ignore=DL3003
RUN wget http://cdist2.perforce.com/perforce/r20.1/bin.linux26x86_64/p4 && \
    mv p4 /usr/local/bin/p4 && \
    chmod +x /usr/local/bin/p4

FROM us.gcr.io/sourcegraph-dev/alpine:f42c99f6acf540d14c671ef7a1b2baa6b8be6a68_99360_candidate
# TODO(security): This container should not be running as root!
#
# The default user in sourcegraph/alpine is a non-root `sourcegraph` user but because old deployments
# cannot be easily migrated we have not changed this from root -> sourcegraph. See:
# https://github.com/sourcegraph/sourcegraph/issues/13238
# hadolint ignore=DL3002
USER root

ARG COMMIT_SHA="unknown"
ARG DATE="unknown"
ARG VERSION="unknown"

LABEL org.opencontainers.image.revision=${COMMIT_SHA}
LABEL org.opencontainers.image.created=${DATE}
LABEL org.opencontainers.image.version=${VERSION}
LABEL com.sourcegraph.github.url=https://github.com/sourcegraph/sourcegraph/commit/${COMMIT_SHA}

# hadolint ignore=DL3018
RUN apk update && apk add --no-cache \
    # NOTE that the Postgres version we run is different
    # from our *Minimum Supported Version* which alone dictates
    # the features we can depend on. See this link for more information:
    # https://github.com/sourcegraph/sourcegraph/blob/main/doc/dev/postgresql.md#version-requirements
    'bash=5.0.17-r0' \
    'redis=5.0.11-r0' \
    # Gitserver requires Git protocol v2 https://github.com/sourcegraph/sourcegraph/issues/13168
    'git>=2.18' \
    git-p4 \
    python2 \
    'nginx>=1.18.0' openssh-client pcre sqlite-libs su-exec 'nodejs-current=14.5.0-r0' \
    postgresql=12.7-r0 \
    postgresql-contrib

# IMPORTANT: If you update the syntect_server version below, you MUST confirm
# the ENV variables from its Dockerfile (https://github.com/sourcegraph/syntect_server/blob/master/Dockerfile)
# have been appropriately set in cmd/server/shared/shared.go.
# hadolint ignore=DL3022
COPY --from=comby/comby:0.18.4@sha256:b47ce282778bfea7f80d45f5ef0cc546ba0d6347baccebaf171a7866143b2593 /usr/local/bin/comby /usr/local/bin/comby
# hadolint ignore=DL3022
COPY --from=docker.io/sourcegraph/syntect_server:137d7de@sha256:36e5f85519052bc660345753cdb4774ea38f81b3b3ba65eec184ac53a3ad1c7b /syntect_server /usr/local/bin/


# install minio (keep this up to date with docker-images/minio/Dockerfile)
ENV MINIO_VERSION=RELEASE.2021-04-22T15-44-28Z
RUN wget "https://dl.min.io/server/minio/release/linux-amd64/archive/minio.$MINIO_VERSION" && \
    chmod +x "minio.$MINIO_VERSION" && \
    mv "minio.$MINIO_VERSION" /usr/local/bin/minio

COPY ctags-install-alpine.sh /ctags-install-alpine.sh
RUN /ctags-install-alpine.sh

# hadolint ignore=DL3022
COPY --from=sourcegraph/prometheus:server /bin/prom-wrapper /bin
# hadolint ignore=DL3022
COPY --from=sourcegraph/prometheus:server /bin/alertmanager /bin
# hadolint ignore=DL3022
COPY --from=sourcegraph/prometheus:server /alertmanager.sh /alertmanager.sh
# hadolint ignore=DL3022
COPY --from=sourcegraph/prometheus:server /bin/prometheus /bin
# hadolint ignore=DL3022
COPY --from=sourcegraph/prometheus:server /prometheus.sh /prometheus.sh
# hadolint ignore=DL3022
COPY --from=sourcegraph/prometheus:server /usr/share/prometheus /usr/share/prometheus

# hadolint ignore=DL3018
RUN set -ex && \
    addgroup -S grafana && \
    adduser -S -G grafana grafana && \
    apk add --no-cache libc6-compat ca-certificates su-exec

# hadolint ignore=DL3022
COPY --from=sourcegraph/grafana:server /usr/share/grafana /usr/share/grafana

# hadolint ignore=DL3022
COPY --from=libsqlite3-pcre /sqlite3-pcre/pcre.so /libsqlite3-pcre.so
ENV LIBSQLITE3_PCRE /libsqlite3-pcre.so
COPY . /

# hadolint ignore=DL3022
COPY --from=p4cli /usr/local/bin/p4 /usr/local/bin/p4

# This is a trick to include libraries required by p4,
# please refer to https://blog.tilander.org/docker-perforce/
ADD https://github.com/jtilander/p4d/raw/4600d741720f85d77852dcca7c182e96ad613358/lib/lib-x64.tgz /
RUN tar zxf /lib-x64.tgz --directory /

# hadolint ignore=DL3022
COPY --from=sourcegraph/grafana:server /sg_config_grafana/provisioning/dashboards /sg_config_grafana/provisioning/dashboards

# hadolint ignore=DL3022
COPY --from=sourcegraph/postgres_exporter:server /usr/local/bin/postgres_exporter /usr/local/bin/postgres_exporter

RUN echo "hosts: files dns" > /etc/nsswitch.conf

ENV GO111MODULES=on LANG=en_US.utf8
ENTRYPOINT ["/sbin/tini", "--", "/usr/local/bin/server"]
