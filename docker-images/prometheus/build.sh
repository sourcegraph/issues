#!/usr/bin/env bash

set -ex

cd "$(dirname "${BASH_SOURCE[0]}")"

# We build out of tree to prevent triggering dev watch scripts when we copy go
# files.
BUILDDIR=$(mktemp -d -t sgdockerbuild_XXXXXXX)
cleanup() {
  rm -rf "$BUILDDIR"
}
trap cleanup EXIT

# The prom-wrapper has a dependency on internal/conf which makes its dependency
# tree quite complicated. Cross-compile it separately before building the image.
export GO111MODULE=on
export GOARCH=amd64
export GOOS=linux
export CGO_ENABLED=0
go build \
  -trimpath \
  -installsuffix netgo \
  -tags "dist netgo" \
  -o "$BUILDDIR"/.bin/prom-wrapper ./cmd/prom-wrapper

# We copy just the monitoring directory and the root go.mod/go.sum so that we
# do not need to send the entire repository as build context to Docker. Additionally,
# we do not use a separate go.mod/go.sum in the monitoring/ directory because
# editor tooling would occassionally include and not include it in the root
# go.mod/go.sum.
cp -R . "$BUILDDIR"
cp -R ../../monitoring "$BUILDDIR"/
cp ../../go.* "$BUILDDIR"/monitoring

pushd "$BUILDDIR"

# Note: This chmod is so that both the `sourcegraph` user and host system user (what `whoami` reports on
# Linux) both have access to the files in the container AND files mounted by `-v` into the container without it
# running as root. For more details, see:
# https://github.com/sourcegraph/sourcegraph/pull/11832#discussion_r451109637
chmod -R 777 config

# Enable image build caching via CACHE=true (the jsonnet builds can take a long time)
BUILD_CACHE="--no-cache"
if [[ "$CACHE" == "true" ]]; then
  BUILD_CACHE=""
fi

# shellcheck disable=SC2086
docker build ${BUILD_CACHE} -t "${IMAGE:-sourcegraph/prometheus}" . \
  --progress=plain \
  --build-arg COMMIT_SHA \
  --build-arg DATE \
  --build-arg VERSION

# cd out of $BUILDDIR for cleanup
popd
