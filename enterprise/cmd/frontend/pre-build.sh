#!/usr/bin/env bash

set -exuo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"/../../..

parallel_run() {
  ./dev/ci/parallel_run.sh "$@"
}

echo "--- yarn root"
# mutex is necessary since frontend and the management-console can
# run concurrent "yarn" installs
# TODO: This is no longer needed since the management console was removed.
yarn --mutex network --frozen-lockfile --network-timeout 60000

MAYBE_TIME_PREFIX=""
if [[ "${CI_DEBUG_PROFILE:-"false"}" == "true" ]]; then
  MAYBE_TIME_PREFIX="env time -v"
fi

build_browser() {
  echo "--- yarn browser"
  (cd client/browser && TARGETS=phabricator eval "${MAYBE_TIME_PREFIX} yarn build")
}

build_web() {
  echo "--- yarn web"
  NODE_ENV=production eval "${MAYBE_TIME_PREFIX} yarn -s run build-web --color"
}

export -f build_browser
export -f build_web

echo "--- (enterprise) build browser and web concurrently"
parallel_run ::: build_browser build_web

echo "--- (enterprise) generate"
./enterprise/dev/generate.sh
