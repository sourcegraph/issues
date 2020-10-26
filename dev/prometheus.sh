#!/usr/bin/env bash

# Description: Prometheus collects metrics and aggregates them into graphs.
# Most of this script is very similar to /dev/grafana.sh - refer to inline documentation there.

set -euf -o pipefail
pushd "$(dirname "${BASH_SOURCE[0]}")/.." >/dev/null

PROMETHEUS_DISK="${HOME}/.sourcegraph-dev/data/prometheus"
if [ ! -e "${PROMETHEUS_DISK}" ]; then
  mkdir -p "${PROMETHEUS_DISK}"
fi
IMAGE=sourcegraph/prometheus:dev
CONTAINER=prometheus

CONFIG_DIR="$(pwd)/docker-images/prometheus/config"
DOCKER_NET=""
DOCKER_USER=""
PROM_TARGETS="dev/prometheus/all/prometheus_targets.yml"
SRC_FRONTEND_INTERNAL="host.docker.internal:3090"

if [[ "$OSTYPE" == "linux-gnu" ]]; then
  DOCKER_USER="--user=$UID"
  PROM_TARGETS="dev/prometheus/linux/prometheus_targets.yml"

  # Frontend generally runs outside of Docker, so to access it we need to be
  # able to access ports on the host. --net=host is a very dirty way of
  # enabling this.
  DOCKER_NET="--net=host"
  SRC_FRONTEND_INTERNAL="localhost:3090"
fi

docker inspect $CONTAINER >/dev/null 2>&1 && docker rm -f $CONTAINER

cp ${PROM_TARGETS} "${CONFIG_DIR}"/prometheus_targets.yml

pushd monitoring >/dev/null || exit 1
DEV=true RELOAD=false go generate
popd >/dev/null || exit 1

PROMETHEUS_LOGS="${HOME}/.sourcegraph-dev/logs/prometheus"
mkdir -p "${PROMETHEUS_LOGS}"
PROMETHEUS_LOG_FILE="${PROMETHEUS_LOGS}/prometheus.log"

# Quickly build image
IMAGE=${IMAGE} CACHE=true ./docker-images/prometheus/build.sh >"${PROMETHEUS_LOG_FILE}" 2>&1 ||
  (BUILD_EXIT_CODE=$? && echo "build failed; dumping log:" && cat "${PROMETHEUS_LOG_FILE}" && exit $BUILD_EXIT_CODE)

function finish() {
  PROMETHEUS_EXIT_CODE=$?

  if [ $PROMETHEUS_EXIT_CODE -ne 0 ] && [ $PROMETHEUS_EXIT_CODE -ne 2 ]; then
    echo "Prometheus exited with unexpected code ${PROMETHEUS_EXIT_CODE}; dumping log:"
    cat "${PROMETHEUS_LOG_FILE}"
  fi

  rm -f "${CONFIG_DIR}"/prometheus_targets.yml
  return $PROMETHEUS_EXIT_CODE
}

docker run --rm ${DOCKER_NET} ${DOCKER_USER} \
  --name=${CONTAINER} \
  --cpus=1 \
  --memory=4g \
  -p 0.0.0.0:9090:9090 \
  -v "${PROMETHEUS_DISK}":/prometheus \
  -v "${CONFIG_DIR}":/sg_prometheus_add_ons \
  -e SRC_FRONTEND_INTERNAL="${SRC_FRONTEND_INTERNAL}" \
  ${IMAGE} >"${PROMETHEUS_LOG_FILE}" 2>&1 || finish
