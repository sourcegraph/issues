#!/usr/bin/env bash

set -euxo pipefail

echo "--- add build dependencies"
apk add alpine-sdk sudo exa

echo "---  list versions of all installed packages (for logs and future debugging)"
mapfile -t packages < <(apk info)
apk info --description "${packages[@]}"

echo "--- setup sourcegraph user"
adduser -D sourcegraph
echo 'root ALL=(ALL) NOPASSWD:ALL' >>/etc/sudoers
echo 'sourcegraph ALL=(ALL) NOPASSWD:ALL' >>/etc/sudoers
addgroup sourcegraph abuild

echo "--- build git package"
sudo -u sourcegraph -s ./make-git.sh
