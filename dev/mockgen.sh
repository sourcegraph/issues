#!/usr/bin/env bash

set -euf -o pipefail

ROOT="$(dirname "${BASH_SOURCE[0]}")/.."

export GOBIN="$ROOT/.bin"
export PATH=$GOBIN:$PATH
export GO111MODULE=on

# Keep this in sync with go.mod
REQUIRED_VERSION='1.1.2'

set +o pipefail
INSTALLED_VERSION="$(go-mockgen --version || :)"
set -o pipefail

if [[ "${INSTALLED_VERSION}" != "${REQUIRED_VERSION}" ]]; then
  echo "Updating local isntallation of go-mockgen"

  go get \
    "github.com/derision-test/go-mockgen/cmd/go-mockgen@v${REQUIRED_VERSION}" \
    golang.org/x/tools/cmd/goimports
fi

go-mockgen -f "$@"
