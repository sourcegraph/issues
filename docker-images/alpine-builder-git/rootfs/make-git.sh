#!/usr/bin/env bash

set -euxo pipefail

cd /home/sourcegraph

git config --global user.name "Sourcegraph"
git config --global user.email "support@sourcegraph.com"
git clone git://git.alpinelinux.org/aports

OUTPUT_DIR="/target"
DIST_FILES_DIR="/var/cache/distfiles"

sudo mkdir -p "${DIST_FILES_DIR}"
sudo chmod a+w "${DIST_FILES_DIR}"
sudo echo -ne '\n' | abuild-keygen -a -i
sudo mkdir "${OUTPUT_DIR}"
sudo chmod 0777 "${OUTPUT_DIR}"

cd aports/main/git

echo "--- building git binaries to output directory: ${OUTPUT_DIR}"
abuild -c -r -P "${OUTPUT_DIR}"

exa --long --tree --level 3 "${OUTPUT_DIR}/main/x86_64/"
