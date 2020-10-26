#!/bin/sh

# This script installs ctags within an alpine container.

set -eux

# Commit hash of github.com/universal-ctags/ctags
CTAGS_VERSION=03f933a96d3ef87adbf9d167462d45ce69577edb

apk --no-cache add \
  autoconf \
  automake \
  binutils \
  curl \
  g++ \
  gcc \
  jansson-dev \
  libseccomp-dev \
  jansson \
  libseccomp \
  linux-headers \
  make \
  pkgconfig

# Installation
curl "https://codeload.github.com/universal-ctags/ctags/tar.gz/$CTAGS_VERSION" | tar xz -C /tmp
cd /tmp/ctags-$CTAGS_VERSION
./autogen.sh
./configure --program-prefix=universal- --enable-json --enable-seccomp
make -j8
make install

# Cleanup
cd /
rm -rf /tmp/ctags-$CTAGS_VERSION
