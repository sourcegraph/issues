#!/usr/bin/env bash
cd "$(dirname "${BASH_SOURCE[0]}")"
set -ex

# This merely re-tags the image to match our official versioning scheme. The
# actual image currently lives here: https://github.com/sourcegraph/infrastructure/tree/master/docker-images
#
# TODO: Move the image to this directory so it is open-source and built in CI automatically.
docker pull index.docker.io/sourcegraph/syntect_server:67fa4c1@sha256:e50ed88f971f7b08698abbc87a10fe18f2f8bbb6472766a15677621c59ca5185
docker tag index.docker.io/sourcegraph/syntect_server:67fa4c1@sha256:e50ed88f971f7b08698abbc87a10fe18f2f8bbb6472766a15677621c59ca5185 "$IMAGE"
