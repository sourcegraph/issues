#!/usr/bin/env bash
# This runs a published or local server image.

DATA=/tmp/sourcegraph

# shellcheck disable=SC2153
case "$CLEAN" in

  "true")
    clean=y
    ;;
  "false")
    clean=n
    ;;
  *)
    echo -n "Do you want to delete $DATA and start clean? [Y/n] "
    read -r clean
    ;;
esac

if [ "$clean" != "n" ] && [ "$clean" != "N" ]; then
  echo "deleting $DATA"
  rm -rf $DATA
fi

# docker run pulls image if not found locally
IMAGE=${IMAGE:-sourcegraph/server:${TAG:-insiders}}
echo "starting server ${IMAGE}"
docker run "$@" \
  --publish 7080:7080 \
  --rm \
  -e SRC_LOG_LEVEL=dbug \
  -e DEBUG=t \
  --volume $DATA/config:/etc/sourcegraph \
  --volume $DATA/data:/var/opt/sourcegraph \
  "$IMAGE"
