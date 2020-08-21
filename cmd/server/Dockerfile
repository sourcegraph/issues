FROM alpine:3.12@sha256:185518070891758909c9f839cf4ca393ee977ac378609f700f60a771a2dfe321 as libsqlite3-pcre

COPY libsqlite3-pcre-install-alpine.sh /libsqlite3-pcre-install-alpine.sh
RUN /libsqlite3-pcre-install-alpine.sh

# TODO: Make this image use our sourcegraph/alpine:3.12 base image
FROM alpine:3.12@sha256:185518070891758909c9f839cf4ca393ee977ac378609f700f60a771a2dfe321

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
    'redis=5.0.9-r0' bind-tools ca-certificates \
    mailcap 'nginx=1.18.0-r0' openssh-client pcre sqlite-libs su-exec tini 'nodejs-current=14.5.0-r0' curl

# IMPORTANT: If you update the syntect_server version below, you MUST confirm
# the ENV variables from its Dockerfile (https://github.com/sourcegraph/syntect_server/blob/master/Dockerfile)
# have been appropriately set in cmd/server/shared/shared.go.
# hadolint ignore=DL3022
COPY --from=comby/comby:0.15.0@sha256:b3b57ed810be1489ccd1ef8fa42210d87d2d396e416c62cee345f615a0dcb09b /usr/local/bin/comby /usr/local/bin/comby
# hadolint ignore=DL3022
COPY --from=sourcegraph/syntect_server:ff37f90@sha256:aa93514b7bc3aaf7a4e9c92e5ff52ee5052db6fb101255a69f054e5b8cdb46ff /syntect_server /usr/local/bin/

# install postgres 11
# hadolint ignore=DL3022,DL3003,DL3018,DL3019,SC2035
RUN wget https://storage.googleapis.com/apktmp/postgres-11-5-for-alpine-3-12-4cc8106a5e.tar.gz && \
    tar -xzf postgres-11-5-for-alpine-3-12-4cc8106a5e.tar.gz && \
    cd target/main/x86_64/ && \
    apk add --allow-untrusted *.apk && \
    cd ../../../ && \
    rm -rf target/

# install git edge
# hadolint ignore=DL3022
COPY --from=sourcegraph/alpine-builder-git:2.28@sha256:7d11a83f0359c6aebc5c9457eda1a8e7199375c7dd879b13e0a36de5dafd9116 /target/main/x86_64/ target/main/x86_64/
# hadolint ignore=DL3022,DL3003,DL3018,DL3019,SC2035
RUN cd target/main/x86_64/ && \
    apk add --allow-untrusted *.apk && \
    cd ../../../ && \
    rm -rf target/

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
COPY --from=sourcegraph/grafana:server /sg_config_grafana/provisioning/dashboards /sg_config_grafana/provisioning/dashboards

# hadolint ignore=DL3022
COPY --from=wrouesnel/postgres_exporter:v0.7.0@sha256:785c919627c06f540d515aac88b7966f352403f73e931e70dc2cbf783146a98b /postgres_exporter /usr/local/bin/postgres_exporter

RUN echo "hosts: files dns" > /etc/nsswitch.conf

ENV GO111MODULES=on LANG=en_US.utf8
ENTRYPOINT ["/sbin/tini", "--", "/usr/local/bin/server"]
