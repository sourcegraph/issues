FROM golang:1.16-alpine AS builder

WORKDIR /go/src/progress-bot

COPY go.* ./
RUN go mod download

COPY *.go ./
RUN go build -o /bin/progress-bot

FROM us.gcr.io/sourcegraph-dev/alpine:f42c99f6acf540d14c671ef7a1b2baa6b8be6a68_99360_candidate
# TODO(security): This container should not be running as root!
# hadolint ignore=DL3002
USER root

# hadolint ignore=DL3018
RUN apk add --no-cache ca-certificates git bash

WORKDIR /

COPY --from=builder /bin/progress-bot /usr/local/bin/
COPY run.sh .
RUN chmod +x run.sh

ENV SINCE=24h DRY=false CHANNEL=progress

ENTRYPOINT ["/run.sh"]
