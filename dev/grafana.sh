#!/usr/bin/env bash

# Description: Dashboards and graphs for Grafana metrics.

set -euf -o pipefail
pushd "$(dirname "${BASH_SOURCE[0]}")/.." >/dev/null

GRAFANA_DISK="${HOME}/.sourcegraph-dev/data/grafana"
if [ ! -e "${GRAFANA_DISK}" ]; then
  mkdir -p "${GRAFANA_DISK}"
fi

IMAGE=sourcegraph/grafana:dev
CONTAINER=grafana

# docker containers must access things via docker host on non-linux platforms
CONFIG_SUB_DIR="all"
DOCKER_USER=""

if [[ "$OSTYPE" == "linux-gnu" ]]; then
  CONFIG_SUB_DIR="linux"

  # Docker users on Linux will generally be using direct user mapping, which
  # means that they'll want the data in the volume mount to be owned by the
  # same user as is running this script. Fortunately, the Grafana container
  # doesn't really care what user it runs as, so long as it can write to
  # /var/lib/grafana.
  DOCKER_USER="--user=$UID"
fi

docker inspect $CONTAINER >/dev/null 2>&1 && docker rm -f $CONTAINER

# Generate Grafana dashboards
pushd monitoring >/dev/null || exit 1
DEV=true RELOAD=false go generate
popd >/dev/null || exit 1

# Log file location: since we log outside of the Docker container, we should
# log somewhere that's _not_ ~/.sourcegraph-dev/data/grafana, since that gets
# volume mounted into the container and therefore has its own ownership
# semantics.
GRAFANA_LOGS="${HOME}/.sourcegraph-dev/logs/grafana"
mkdir -p "${GRAFANA_LOGS}"

# Now for the actual logging. Grafana's output gets sent to stdout and stderr.
# We want to capture that output, but because it's fairly noisy, don't want to
# display it in the normal case.
GRAFANA_LOG_FILE="${GRAFANA_LOGS}/grafana.log"

# Quickly build image
IMAGE=${IMAGE} CACHE=true ./docker-images/grafana/build.sh >"${GRAFANA_LOG_FILE}" 2>&1 ||
  (BUILD_EXIT_CODE=$? && echo "build failed; dumping log:" && cat "${GRAFANA_LOG_FILE}" && exit $BUILD_EXIT_CODE)

function finish() {
  GRAFANA_EXIT_CODE=$?

  # Exit code 2 indicates a normal Ctrl-C termination via goreman, so we'll
  # only dump the log if it's not 0 _or_ 2.
  if [ $GRAFANA_EXIT_CODE -ne 0 ] && [ $GRAFANA_EXIT_CODE -ne 2 ]; then
    echo "Grafana exited with unexpected code ${GRAFANA_EXIT_CODE}; dumping log:"
    cat "${GRAFANA_LOG_FILE}"
  fi

  # Ensure that we still return the same code so that goreman can do sensible
  # things once this script exits.
  return $GRAFANA_EXIT_CODE
}

docker run --rm ${DOCKER_USER} \
  --name=${CONTAINER} \
  --cpus=1 \
  --memory=1g \
  -p 0.0.0.0:3370:3370 \
  -v "${GRAFANA_DISK}":/var/lib/grafana \
  -v "$(pwd)"/dev/grafana/${CONFIG_SUB_DIR}:/sg_config_grafana/provisioning/datasources \
  -v "$(pwd)"/docker-images/grafana/config/provisioning/dashboards:/sg_grafana_additional_dashboards \
  -v "$(pwd)"/docker-images/grafana/jsonnet:/sg_grafana_additional_dashboards/legacy \
  -e DISABLE_SOURCEGRAPH_CONFIG="${DISABLE_SOURCEGRAPH_CONFIG:-'false'}" \
  ${IMAGE} >"${GRAFANA_LOG_FILE}" 2>&1 || finish

# Add the following lines above if you wish to use an auth proxy with Grafana:
#
# -e GF_AUTH_PROXY_ENABLED=true \
# -e GF_AUTH_PROXY_HEADER_NAME='X-Forwarded-User' \
# -e GF_SERVER_ROOT_URL='https://grafana.example.com' \
