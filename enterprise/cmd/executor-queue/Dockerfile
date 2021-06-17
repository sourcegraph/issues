FROM sourcegraph/alpine-3.12:99422_2021-06-16_cfa0e09@sha256:ffc2ad13f9d44afe8e20fd9d4f9a23af6851d96ccf2ff3eb149ea93fc0420b65

ARG COMMIT_SHA="unknown"
ARG DATE="unknown"
ARG VERSION="unknown"

LABEL org.opencontainers.image.revision=${COMMIT_SHA}
LABEL org.opencontainers.image.created=${DATE}
LABEL org.opencontainers.image.version=${VERSION}
LABEL com.sourcegraph.github.url=https://github.com/sourcegraph/sourcegraph/commit/${COMMIT_SHA}

# hadolint ignore=DL3018
RUN apk update && apk add --no-cache \
    tini

USER sourcegraph
EXPOSE 3191
ENTRYPOINT ["/sbin/tini", "--", "/usr/local/bin/executor-queue"]
COPY executor-queue /usr/local/bin/
