#!/usr/bin/env bash

set -e

# For symbol tests
echo "--- build libsqlite"
./dev/libsqlite3-pcre/build.sh

# For searcher
echo "--- comby install"
./dev/comby-install-or-upgrade.sh

# Separate out time for go mod from go test
echo "--- go mod download"
go mod download

echo "--- go test"
go test -timeout 4m -coverprofile=coverage.txt -covermode=atomic -race ./...
