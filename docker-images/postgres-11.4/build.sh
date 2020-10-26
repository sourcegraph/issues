#!/usr/bin/env bash

set -ex
cd "$(dirname "${BASH_SOURCE[0]}")"

# This merely re-tags the image to match our official versioning scheme. The
# actual image currently lives here: https://github.com/sourcegraph/infrastructure/tree/master/docker-images
#
# TODO: Move the image to this directory so it is open-source and built in CI automatically.
docker pull index.docker.io/sourcegraph/postgres-11.4:20-10-21_9ae31d46@sha256:a55fea6638d478c2368c227d06a1a2b7a2056b693967628427d41c92d9209e97
docker tag index.docker.io/sourcegraph/postgres-11.4:20-10-21_9ae31d46@sha256:a55fea6638d478c2368c227d06a1a2b7a2056b693967628427d41c92d9209e97 "$IMAGE"
