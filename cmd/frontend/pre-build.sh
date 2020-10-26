#!/usr/bin/env bash
set -ex
cd "$(dirname "${BASH_SOURCE[0]}")"/../..

# Build the webapp typescript code.
echo "--- yarn"
# mutex is necessary since CI runs various yarn installs in parallel
if [[ -z "${CI}" ]]; then
  yarn --mutex network
else
  yarn --mutex network --frozen-lockfile --network-timeout 60000
fi

echo "--- yarn run build-web"
NODE_ENV=production DISABLE_TYPECHECKING=true yarn run build-web

echo "--- go generate"
go generate ./cmd/frontend/internal/app/assets ./cmd/frontend/internal/app/templates
