#!/usr/bin/env bash

# We want to build multiple go binaries, so we use a custom build step on CI.
cd "$(dirname "${BASH_SOURCE[0]}")/../.."
set -eux

# Fail early if env vars are not set
[ -n "$VERSION" ]
[ -n "$IMAGE" ]

OUTPUT=$(mktemp -d -t sgserver_XXXXXXX)
export OUTPUT
cleanup() {
  rm -rf "$OUTPUT"
}
trap cleanup EXIT

parallel_run() {
  ./dev/ci/parallel_run.sh "$@"
}
export -f parallel_run

# Environment for building linux binaries
export GO111MODULE=on
export GOARCH=amd64
export GOOS=linux
export CGO_ENABLED=0

# Additional images passed in here when this script is called externally by our
# enterprise build scripts.
additional_images=()
if [ $# -eq 0 ]; then
  additional_images+=("github.com/sourcegraph/sourcegraph/cmd/frontend" "github.com/sourcegraph/sourcegraph/cmd/repo-updater")
else
  additional_images+=("$@")
fi
export additional_images

# Overridable server package path for when this script is called externally by
# our enterprise build scripts.
export server_pkg=${SERVER_PKG:-github.com/sourcegraph/sourcegraph/cmd/server}

cp -a ./cmd/server/rootfs/. "$OUTPUT"
export BINDIR="$OUTPUT/usr/local/bin"
mkdir -p "$BINDIR"

go_build() {
  local package="$1"

  if [[ "${CI_DEBUG_PROFILE:-"false"}" == "true" ]]; then
    env time -v ./cmd/server/go-build.sh "$package"
  else
    ./cmd/server/go-build.sh "$package"
  fi
}
export -f go_build

echo "--- go build"

PACKAGES=(
  github.com/sourcegraph/sourcegraph/cmd/github-proxy
  github.com/sourcegraph/sourcegraph/cmd/gitserver
  github.com/sourcegraph/sourcegraph/cmd/query-runner
  github.com/sourcegraph/sourcegraph/cmd/searcher
  github.com/sourcegraph/sourcegraph/cmd/symbols
  github.com/google/zoekt/cmd/zoekt-archive-index
  github.com/google/zoekt/cmd/zoekt-git-index
  github.com/google/zoekt/cmd/zoekt-sourcegraph-indexserver
  github.com/google/zoekt/cmd/zoekt-webserver
)

PACKAGES+=("${additional_images[@]}")
PACKAGES+=("$server_pkg")

parallel_run go_build {} ::: "${PACKAGES[@]}"

echo "--- ctags"
cp -a ./cmd/symbols/ctags-install-alpine.sh "$OUTPUT"
cp -a ./dev/libsqlite3-pcre/install-alpine.sh "$OUTPUT/libsqlite3-pcre-install-alpine.sh"

echo "--- monitoring generation"
# For code generation we need to match the local machine so we can run the generator
if [[ "$OSTYPE" == "darwin"* ]]; then
  pushd monitoring && GOOS=darwin go generate && popd
else
  pushd monitoring && go generate && popd
fi

echo "--- prometheus"
cp -r docker-images/prometheus/config "$OUTPUT/sg_config_prometheus"
mkdir "$OUTPUT/sg_prometheus_add_ons"
cp dev/prometheus/linux/prometheus_targets.yml "$OUTPUT/sg_prometheus_add_ons"
IMAGE=sourcegraph/prometheus:server CACHE=true docker-images/prometheus/build.sh

echo "--- grafana"
cp -r docker-images/grafana/config "$OUTPUT/sg_config_grafana"
cp -r dev/grafana/linux "$OUTPUT/sg_config_grafana/provisioning/datasources"
IMAGE=sourcegraph/grafana:server CACHE=true docker-images/grafana/build.sh

echo "--- jaeger-all-in-one binary"
cmd/server/jaeger.sh

echo "--- docker build"
docker build -f cmd/server/Dockerfile -t "$IMAGE" "$OUTPUT" \
  --progress=plain \
  --build-arg COMMIT_SHA \
  --build-arg DATE \
  --build-arg VERSION
