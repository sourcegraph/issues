FROM golang:1.16 AS builder
WORKDIR /build
COPY go.sum go.mod ./
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o searchblitz ./internal/cmd/search-blitz

FROM us.gcr.io/sourcegraph-dev/alpine:f42c99f6acf540d14c671ef7a1b2baa6b8be6a68_99360_candidate

COPY --from=builder /build/searchblitz /usr/local/bin
COPY internal/cmd/search-blitz/config.yaml /config.yaml

ARG COMMIT_SHA="unknown"

LABEL org.opencontainers.image.revision=${COMMIT_SHA}
LABEL org.opencontainers.image.source=https://github.com/sourcegraph/sourcegraph/internal/cmd/search-blitz

ENTRYPOINT ["/sbin/tini", "--", "/usr/local/bin/searchblitz"]
