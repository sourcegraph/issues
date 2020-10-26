#!/usr/bin/env bash

set -euf -o pipefail

pushd "$(dirname "${BASH_SOURCE[0]}")/.." >/dev/null

mkdir -p .bin

version=v1.6.0
suffix="${version}_$(go env GOOS)_$(go env GOARCH)"
target="$PWD/.bin/docsite_${suffix}"
url="https://github.com/sourcegraph/docsite/releases/download/${version}/docsite_${suffix}"

if [ ! -f "${target}" ]; then
  echo "downloading ${url}" 1>&2
  curl -sS -L -f "${url}" -o "${target}.tmp"
  mv "${target}.tmp" "${target}"
fi

chmod +x "${target}"

popd >/dev/null

exec "${target}" "$@"
