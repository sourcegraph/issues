#!/usr/bin/env bash

cd "$(dirname "${BASH_SOURCE[0]}")/.."
set -euxo pipefail

box=$1
exit_code=0

pushd test/

plugins=(vagrant-google vagrant-env vagrant-scp)

for i in "${plugins[@]}"; do
  if ! vagrant plugin list --no-tty | grep "$i"; then
    vagrant plugin install "$i"
  fi
done

vagrant up "$box" --provider=google || exit_code=$?
vagrant scp "$box":/sourcegraph/puppeteer/*.png ../
vagrant scp "$box":/sourcegraph/e2e.mp4 ../
vagrant scp "$box":/sourcegraph/ffmpeg.log ../
vagrant destroy -f "$box"

if [ "$exit_code" != 0 ]; then
  exit 1
fi
