#!/usr/bin/env bash

cd "$(dirname "${BASH_SOURCE[0]}")"/../..
set -ex

if [ -z "$IMAGE" ]; then
  echo "Must specify \$IMAGE."
  exit 1
fi

URL="http://localhost:7080"

if curl --output /dev/null --silent --head --fail $URL; then
  echo "❌ Can't run a new Sourcegraph instance on $URL because another instance is already running."
  echo "❌ The last time this happened, there was a runaway integration test run on the same Buildkite agent and the fix was to delete the pod and rebuild."
  exit 1
fi

echo "--- Running a daemonized $IMAGE as the test subject..."
CONTAINER="$(docker container run -d -e DEPLOY_TYPE=dev "$IMAGE")"
trap 'kill $(jobs -p -r)'" ; docker logs --timestamps $CONTAINER ; docker container rm -f $CONTAINER ; docker image rm -f $IMAGE" EXIT

docker exec "$CONTAINER" apk add --no-cache socat
# Connect the server container's port 7080 to localhost:7080 so that integration tests
# can hit it. This is similar to port-forwarding via SSH tunneling, but uses `docker exec`
# as the transport.
socat tcp-listen:7080,reuseaddr,fork system:"docker exec -i $CONTAINER socat stdio 'tcp:localhost:7080'" &

echo "--- Waiting for $URL to be up"
set +e
timeout 60s bash -c "until curl --output /dev/null --silent --head --fail $URL; do
    echo Waiting 5s for $URL...
    sleep 5
done"
# shellcheck disable=SC2181
if [ $? -ne 0 ]; then
  echo "^^^ +++"
  echo "$URL was not accessible within 60s. Here's the output of docker inspect and docker logs:"
  docker inspect "$CONTAINER"
  exit 1
fi
set -e
echo "Waiting for $URL... done"

echo '--- go test ./dev/gqltest -tags "gqltest" -v'
go test ./dev/gqltest -tags "gqltest" -v
