# This Dockerfile was generated from github.com/sourcegraph/godockerize. It
# was not written by a human, and as such looks janky. As you change this
# file, please don't be scared to make it more pleasant / remove hadolint
# ignores.

FROM sourcegraph/alpine-3.12:99421_2021-06-16_8e20ec8@sha256:838b96d93f073aa773718e22be54362b721569e360b445675f4cab904abc9304

# hadolint ignore=DL3018
RUN apk --no-cache add pcre sqlite-libs

# The comby/comby image is a small binary-only distribution. See the bin and src directories
# here: https://github.com/comby-tools/comby/tree/master/dockerfiles/alpine
# hadolint ignore=DL3022
COPY --from=comby/comby:1.0.0@sha256:0c102a00852dba6f0f2dcff8fa7ee77ab3a9a4e3c74307aa7ceadc0cf12713b1 /usr/local/bin/comby /usr/local/bin/comby

ARG COMMIT_SHA="unknown"
ARG DATE="unknown"
ARG VERSION="unknown"

LABEL org.opencontainers.image.revision=${COMMIT_SHA}
LABEL org.opencontainers.image.created=${DATE}
LABEL org.opencontainers.image.version=${VERSION}
LABEL com.sourcegraph.github.url=https://github.com/sourcegraph/sourcegraph/commit/${COMMIT_SHA}

ENV CACHE_DIR=/mnt/cache/searcher
RUN mkdir -p ${CACHE_DIR} && chown -R sourcegraph:sourcegraph ${CACHE_DIR}
USER sourcegraph
ENTRYPOINT ["/sbin/tini", "--", "/usr/local/bin/searcher"]
COPY searcher /usr/local/bin/
