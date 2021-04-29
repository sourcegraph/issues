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
  echo "❌ The last time this happened, there was a runaway e2e test run on the same Buildkite agent and the fix was to delete the pod and rebuild."
  exit 1
fi

echo "--- Running a daemonized $IMAGE as the test subject..."
CONTAINER="$(docker container run -d "$IMAGE")"
function cleanup() {
  exit_status=$?
  if [ $exit_status -ne 0 ]; then
    # Expand the output if our run failed.
    echo "^^^ +++"
  fi

  jobs -p -r | xargs kill
  echo "--- server logs"
  docker logs --timestamps "$CONTAINER"
  echo "--- docker cleanup"
  docker container rm -f "$CONTAINER"
  docker image rm -f "$IMAGE"

  if [ $exit_status -ne 0 ]; then
    # This command will fail, so our last step will be expanded. We don't want
    # to expand "docker cleanup" so we add in a dummy section.
    echo "--- e2e test failed"
    echo "See yarn run test-e2e section for test runner logs."
  fi
}
trap cleanup EXIT

docker exec "$CONTAINER" apk add --no-cache socat
# Connect the server container's port 7080 to localhost:7080 so that e2e tests
# can hit it. This is similar to port-forwarding via SSH tunneling, but uses
# docker exec as the transport.
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

echo "--- yarn run test-e2e"
env SOURCEGRAPH_BASE_URL="$URL" PERCY_ON=true PERCY_BROWSER_EXECUTABLE=./node_modules/puppeteer/.local-chromium/linux-818858/chrome-linux/chrome ./node_modules/.bin/percy exec -- yarn run cover-e2e

echo "--- coverage"
yarn nyc report -r json
# Upload the coverage under the "e2e" flag (toggleable in the CodeCov UI)
./dev/ci/codecov.sh -F e2e
