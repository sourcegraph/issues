#!/usr/bin/env bash

set -e

echo "--- yarn"
# mutex is necessary since CI runs various yarn installs in parallel
yarn --mutex network --frozen-lockfile --network-timeout 60000
yarn --mutex network --cwd dev/release --frozen-lockfile --network-timeout 60000

echo "--- generate"
yarn gulp generate

for cmd in "$@"; do
  echo "--- $cmd"
  yarn -s run "$cmd"
done
