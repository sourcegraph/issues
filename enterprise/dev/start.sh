#!/usr/bin/env bash

set -euf -o pipefail

DEV_PRIVATE_PATH=$PWD/../dev-private

if [ ! -d "$DEV_PRIVATE_PATH" ]; then
  echo "Expected to find github.com/sourcegraph/dev-private checked out to $DEV_PRIVATE_PATH, but path wasn't a directory" 1>&2
  exit 1
fi

# Warn if dev-private needs to be updated.
required_commit="a661975799edd759b3d6e4c4027e69a9727bdffb"
if ! git -C "$DEV_PRIVATE_PATH" merge-base --is-ancestor $required_commit HEAD; then
  echo "Error: You need to update dev-private to a commit that incorporates https://github.com/sourcegraph/dev-private/commit/$required_commit."
  echo
  echo "Try running:"
  echo
  echo "    cd $DEV_PRIVATE_PATH && git pull"
  echo
  exit 1
fi

# shellcheck disable=SC1090
source "$DEV_PRIVATE_PATH/enterprise/dev/env"

if [ -z "${DEV_NO_CONFIG-}" ]; then
  export SITE_CONFIG_FILE=${SITE_CONFIG_FILE:-$DEV_PRIVATE_PATH/enterprise/dev/site-config.json}
  export EXTSVC_CONFIG_FILE=${EXTSVC_CONFIG_FILE:-$DEV_PRIVATE_PATH/enterprise/dev/external-services-config.json}
  export GLOBAL_SETTINGS_FILE=${GLOBAL_SETTINGS_FILE:-$PWD/dev/global-settings.json}
  export SITE_CONFIG_ALLOW_EDITS=true
  export GLOBAL_SETTINGS_ALLOW_EDITS=true
  export EXTSVC_CONFIG_ALLOW_EDITS=true
fi

SOURCEGRAPH_LICENSE_GENERATION_KEY=$(cat "$DEV_PRIVATE_PATH"/enterprise/dev/test-license-generation-key.pem)
export SOURCEGRAPH_LICENSE_GENERATION_KEY

export PRECISE_CODE_INTEL_BUNDLE_MANAGER_URL=http://localhost:3187
export PRECISE_CODE_INTEL_BUNDLE_DIR=$HOME/.sourcegraph/lsif-storage

export PRECISE_CODE_INTEL_EXTERNAL_URL=http://localhost:3080
export PRECISE_CODE_INTEL_EXTERNAL_URL_FROM_DOCKER=http://host.docker.internal:3080
export PRECISE_CODE_INTEL_INDEX_MANAGER_URL=http://localhost:3189
export PRECISE_CODE_INTEL_INTERNAL_PROXY_AUTH_TOKEN=hunter2
export PRECISE_CODE_INTEL_USE_FIRECRACKER=false
export PRECISE_CODE_INTEL_IMAGE_ARCHIVE_PATH=$HOME/.sourcegraph/images
export DISABLE_CNCF=notonmybox

export WATCH_ADDITIONAL_GO_DIRS="enterprise/cmd enterprise/dev enterprise/internal"
export ENTERPRISE_ONLY_COMMANDS=" precise-code-intel-bundle-manager precise-code-intel-indexer precise-code-intel-indexer-vm precise-code-intel-worker "
export ENTERPRISE_COMMANDS="frontend repo-updater ${ENTERPRISE_ONLY_COMMANDS}"
export ENTERPRISE=1
export PROCFILE=enterprise/dev/Procfile
./dev/start.sh "$@"
