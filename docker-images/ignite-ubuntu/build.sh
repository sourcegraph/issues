#!/usr/bin/env bash

set -ex
cd "$(dirname "${BASH_SOURCE[0]}")"

docker build -t "${IMAGE:-sourcegraph/ignite-ubuntu}" --build-arg SRC_CLI_VERSION="$(go run ../../internal/cmd/src-cli-version/main.go)" .
