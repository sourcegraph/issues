#!/usr/bin/env bash

set -e
unset CDPATH
cd "$(dirname "${BASH_SOURCE[0]}")/.." # cd to repo root dir

# Wrapper for watchman. To debug which changes it detect set the environment
# variable WATCHMAN_DEBUG=t

if [ ! -x "$(command -v watchman)" ]; then
  echo "Please install watchman. https://facebook.github.io/watchman/docs/install.html"
  echo
  echo "  brew install watchman"
  exit 1
fi

exec go run ./dev/watchmanwrapper dev/handle-change.sh <dev/watchmanwrapper/watch.json
