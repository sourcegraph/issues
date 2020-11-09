#!/usr/bin/env bash

cd "$(dirname "${BASH_SOURCE[0]}")/../.." || exit
set -x

# shellcheck disable=SC1091
set +x && source /root/.profile && set -x

test/setup-deps.sh
test/setup-display.sh

# ==========================

# Run and initialize an old Sourcegraph release
IMAGE=sourcegraph/server:$MINIMUM_UPGRADEABLE_VERSION ./dev/run-server-image.sh -d --name sourcegraph-old
sleep 15
go run test/init-server.go

# shellcheck disable=SC1091
set +x && source /root/.profile && set -x

# Stop old Sourcegraph release
docker container stop sourcegraph-old
sleep 5

# Upgrade to current candidate image. Capture logs for the attempted upgrade.
CONTAINER=sourcegraph-new
docker_logs() {
  LOGFILE=$(docker inspect ${CONTAINER} --format '{{.LogPath}}')
  cp "$LOGFILE" $CONTAINER.log
  chmod 744 $CONTAINER.log
}
IMAGE=us.gcr.io/sourcegraph-dev/server:$CANDIDATE_VERSION CLEAN="false" ./dev/run-server-image.sh -d --name $CONTAINER
trap docker_logs exit
sleep 15

# Run tests
echo "TEST: Running regression tests"
pushd client/web || exit
yarn run test:regression:core
popd || exit
echo "TEST: Checking Sourcegraph instance is accessible"
curl -f http://localhost:3370
curl -f http://localhost:3370/healthz

# ==========================

test/cleanup-display.sh
