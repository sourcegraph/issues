#!/usr/bin/env bash

# We want to build multiple go binaries, so we use a custom build step on CI.
set -ex
cd "$(dirname "${BASH_SOURCE[0]}")"/../../..

OUTPUT="${1:?no output path provided}"

# Environment for building linux binaries
export GO111MODULE=on
export GOARCH=amd64
export GOOS=linux

# Get additional build args
. ./dev/libsqlite3-pcre/go-build-args.sh

echo "--- go build"
pkg=github.com/sourcegraph/sourcegraph/enterprise/cmd/frontend
go build -trimpath -ldflags "-X github.com/sourcegraph/sourcegraph/internal/version.version=$VERSION  -X github.com/sourcegraph/sourcegraph/internal/version.timestamp=$(date +%s)" -buildmode exe -tags dist -o "$OUTPUT/$(basename $pkg)" "$pkg"
