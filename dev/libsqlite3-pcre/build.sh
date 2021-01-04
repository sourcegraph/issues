#!/usr/bin/env bash

# This script will ensure that the libsqlite3-pcre dynamic library exists in the root of this
# repository (either libsqlite3-pcre.dylib for Darwin or libsqlite3-pcre.so for linux). This
# script is used by run the symbol service locally, which compiles against the shared library.
#
# Invocation:
# - `./libsqlite3-pcre/build.sh`         : build the library
# - `./libsqlite3-pcre/build.sh libpath` : output its path

cd "$(dirname "${BASH_SOURCE[0]}")/../.."
set -eu

OUTPUT=$(mktemp -d -t sgdockerbuild_XXXXXXX)
cleanup() {
  rm -rf "$OUTPUT"
}
trap cleanup EXIT

# Print the absolute path of the sqlite3 shared library for this platform,
# or terminate with an error.
function libpath() {
  case "$OSTYPE" in
    darwin*)
      echo "$PWD/libsqlite3-pcre.dylib"
      ;;

    linux*)
      echo "$PWD/libsqlite3-pcre.so"
      ;;

    *)
      echo "Unknown platform $OSTYPE"
      exit 1
      ;;
  esac
}

function build() {
  local libsqlite_path
  libsqlite_path=$(libpath)

  if [[ -f "$libsqlite_path" ]]; then
    # Already exists
    exit 0
  fi

  echo "--- libsqlite3-pcre build"

  if ! command -v pkg-config >/dev/null 2>&1 || ! command -v pkg-config --cflags sqlite3 libpcre >/dev/null 2>&1; then
    echo "Missing sqlite dependencies."
    case "$OSTYPE" in
      darwin*)
        echo "Install them by running 'brew install pkg-config sqlite pcre FiloSottile/musl-cross/musl-cross'"
        ;;

      linux*)
        echo "Install them by running 'apt-get install libpcre3-dev libsqlite3-dev pkg-config musl-tools'"
        ;;

      *)
        echo "See the local development documentation: https://github.com/sourcegraph/sourcegraph/blob/main/doc/dev/local_development.md#step-2-install-dependencies"
        ;;
    esac

    exit 1
  fi

  echo "--- $libsqlite_path build"
  curl -fsSL https://codeload.github.com/ralight/sqlite3-pcre/tar.gz/c98da412b431edb4db22d3245c99e6c198d49f7a | tar -C "$OUTPUT" -xzvf - --strip 1
  cd "$OUTPUT"

  case "$OSTYPE" in
    darwin*)
      # pkg-config spits out multiple arguments and must not be quoted.
      # shellcheck disable=SC2046
      gcc -fno-common -dynamiclib pcre.c -o "$libsqlite_path" $(pkg-config --cflags sqlite3 libpcre) $(pkg-config --libs libpcre) -fPIC || echo "if this failed with 'ld: symbol(s) not found for architecture x86_64' \n try running export PKG_CONFIG_PATH='/usr/local/opt/sqlite/lib/pkgconfig'"
      exit 0
      ;;

    linux*)
      # pkg-config spits out multiple arguments and must not be quoted.
      # shellcheck disable=SC2046
      gcc -shared -o "$libsqlite_path" $(pkg-config --cflags sqlite3 libpcre) -fPIC -W -Werror pcre.c $(pkg-config --libs libpcre) -Wl,-z,defs
      exit 0
      ;;

    *)
      echo "See the local development documentation: https://github.com/sourcegraph/sourcegraph/blob/main/doc/dev/local_development.md#step-2-install-dependencies"
      echo "Unknown platform $OSTYPE"
      exit 1
      ;;
  esac
}

# Execute $1 (build by default)
"${1:-build}"
