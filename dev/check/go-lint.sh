#!/bin/bash
set -e
cd "$(dirname "${BASH_SOURCE[0]}")/../.."

export GOBIN="$PWD/.bin"
export PATH=$GOBIN:$PATH
export GO111MODULE=on

pkgs=${@:-./...}

go install github.com/golangci/golangci-lint/cmd/golangci-lint

echo "--- go install"
go install -buildmode=archive ${pkgs}

echo "--- lint"
golangci-lint run -v ${pkgs}
