FROM golang:1.16-alpine AS builder

WORKDIR /go/src/tracking-issue
COPY . .
RUN go mod init tracking-issue
RUN CGO_ENABLED=0 go install .

FROM us.gcr.io/sourcegraph-dev/alpine:f42c99f6acf540d14c671ef7a1b2baa6b8be6a68_99360_candidate
COPY --from=builder /go/bin/* /usr/local/bin/
ENTRYPOINT ["tracking-issue"]
