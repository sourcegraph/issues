#!/usr/bin/env bash

set -e

echo "--- Phabricator"

# shellcheck source=./start.sh
source ./dev/phabricator/start.sh

export PHABRICATOR_CONTAINER
PHABRICATOR_CONTAINER="$(docker ps -aq -f name=phabricator$)"

# Install the Sourcegraph native integration
# shellcheck source=./install-sourcegraph.sh
source ./dev/phabricator/install-sourcegraph.sh

pushd client/web
yarn run test-phabricator-e2e
popd
