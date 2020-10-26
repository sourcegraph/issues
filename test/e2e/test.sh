#!/usr/bin/env bash

cd "$(dirname "${BASH_SOURCE[0]}")/../.." || exit
set -x

# shellcheck disable=SC1091
source /root/.profile
Xvfb "$DISPLAY" -screen 0 1280x1024x24 &
x11vnc -display "$DISPLAY" -forever -rfbport 5900 >/x11vnc.log 2>&1 &

asdf install
yarn
yarn generate
pushd enterprise || exit
./cmd/server/pre-build.sh
./cmd/server/build.sh
popd || exit
./dev/ci/e2e.sh
docker image rm -f "${IMAGE}"
