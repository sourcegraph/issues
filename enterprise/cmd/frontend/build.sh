#!/usr/bin/env bash

# We want to build multiple go binaries, so we use a custom build step on CI.
set -ex
cd "$(dirname "${BASH_SOURCE[0]}")"/../../..

OUTPUT=$(mktemp -d -t sgdockerbuild_XXXXXXX)
cleanup() {
  rm -rf "$OUTPUT"
}
trap cleanup EXIT

cp -a ./dev/libsqlite3-pcre/install-alpine.sh "$OUTPUT/libsqlite3-pcre-install-alpine.sh"

# Build go binary into $OUTPUT
./enterprise/cmd/frontend/go-build.sh "$OUTPUT"

echo "--- docker build"
docker build -f enterprise/cmd/frontend/Dockerfile -t "$IMAGE" "$OUTPUT" \
  --progress=plain \
  --build-arg COMMIT_SHA \
  --build-arg DATE \
  --build-arg VERSION
