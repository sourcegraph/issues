#!/usr/bin/env bash

# This script will fetch an image and then push it to multiple different
# locations. This is used in our CI pipeline. We build a private candidate
# image and once e2e test pass we publish it to dockerhub.
#
# USAGE: docker-publish.sh src dst...
#
# For example
#
#  docker-publish.sh \
#    us.gcr.io/sourcegraph-dev/searcher:358c739f2c8ba8618c4da31dfa08c08b38777164_71747_candidate \
#    index.docker.io/sourcegraph/searcher:71747_2020-08-25_358c739 \
#    index.docker.io/sourcegraph/searcher:insiders \
#    us.gcr.io/sourcegraph-dev/searcher:71747_2020-08-25_358c739 \
#    us.gcr.io/sourcegraph-dev/searcher:insiders

set -e

yes | gcloud auth configure-docker

src="$1"

echo "--- pulling $src"
docker pull "$src"

shift
for dst in "$@"; do
  echo "--- pushing $dst"
  docker tag "$src" "$dst"
  docker push "$dst"
done

echo "+++ summary"

id=$(docker inspect --format='{{index .RepoDigests 0}}' "$src" | grep -o '@sha256:.*')
for dst in "$@"; do
  echo "$dst$id"
done
