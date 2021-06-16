FROM us.gcr.io/sourcegraph-dev/alpine:f42c99f6acf540d14c671ef7a1b2baa6b8be6a68_99360_candidate

ARG COMMIT_SHA="unknown"
ARG DATE="unknown"
ARG VERSION="unknown"

LABEL org.opencontainers.image.revision=${COMMIT_SHA}
LABEL org.opencontainers.image.created=${DATE}
LABEL org.opencontainers.image.version=${VERSION}
LABEL com.sourcegraph.github.url=https://github.com/sourcegraph/sourcegraph/commit/${COMMIT_SHA}

# hadolint ignore=DL3018
RUN apk add --no-cache git

# Ensures that a directory with the correct permissions exist in the image. Without this, in Docker Compose
# deployments the Docker daemon would first create the volume directory and it would be owned by `root` and
# then a non-root process would be unable to create the `/app/data` because it
# would  be trying to do so in a directory owned by `root` as the user `sourcegraph`. And no, this is not
# dumb, this is just Docker: https://github.com/docker/compose/issues/3270#issuecomment-363478501.
USER root
RUN mkdir -p /app/data && chown -R sourcegraph:sourcegraph /app/data
USER sourcegraph

WORKDIR /app/data/
VOLUME ["/app/data"]
ENTRYPOINT ["/sbin/tini", "--", "/usr/local/bin/src-expose"]
EXPOSE 3434

COPY entry.sh src-expose /usr/local/bin/
