#!/usr/bin/env bash

# This function installs the comby dependency for cmd/searcher.
# The CI pipeline calls this script to install or upgrade comby
# for tests or development environments.
REQUIRE_VERSION="0.18.1"

RELEASE_VERSION=$REQUIRE_VERSION
RELEASE_TAG=$REQUIRE_VERSION
RELEASE_URL="https://github.com/comby-tools/comby/releases"

INSTALL_DIR=/usr/local/bin

function ctrl_c() {
  rm -f "$TMP/$RELEASE_BIN" &>/dev/null
  printf "[-] Installation cancelled. Please see https://github.com/comby-tools/comby/releases if you prefer to install manually.\n"
  exit 1
}

trap ctrl_c INT

EXISTS=$(command -v comby || echo)

# Exit if comby exists with the desired version.
if [ "$EXISTS" ] && [ "$(comby -version)" == "$REQUIRE_VERSION" ]; then
  exit 0
fi

if [ -n "$EXISTS" ]; then
  INSTALL_DIR=$(dirname "$EXISTS")
fi

if [ ! -d "$INSTALL_DIR" ]; then
  printf "%s does not exist. Please download the binary from %s and install it manually.\n" "$INSTALL_DIR" "$RELEASE_URL"
  exit 1
fi

TMP=${TMPDIR:-/tmp}

ARCH=$(uname -m || echo)
case "$ARCH" in
  x86_64 | amd64) ARCH="x86_64" ;;
  *) ARCH="OTHER" ;;
esac

OS=$(uname -s || echo)
if [ "$OS" = "Darwin" ]; then
  OS=macos
fi

RELEASE_BIN="comby-${RELEASE_TAG}-${ARCH}-${OS}"
RELEASE_URL="https://github.com/comby-tools/comby/releases/download/${RELEASE_TAG}/${RELEASE_BIN}"

if [ ! -e "$TMP/$RELEASE_BIN" ]; then
  printf "[+] Downloading comby %s\n" "$RELEASE_VERSION"

  SUCCESS=$(curl -s -L -o "$TMP/$RELEASE_BIN" "$RELEASE_URL" --write-out "%{http_code}")

  if [ "$SUCCESS" == "404" ]; then
    printf "[-] No binary release available for your system.\n"
    rm -f "$TMP/$RELEASE_BIN"
    exit 1
  fi
  printf "[+] Download complete.\n"
fi

chmod 755 "$TMP/$RELEASE_BIN"
echo "[+] Installing comby to $INSTALL_DIR"
if [ ! $OS == "macos" ]; then
  printf "[*] To install comby to %s requires sudo access. Please type the sudo password in the prompt below.\n" "$INSTALL_DIR"
  sudo cp "$TMP/$RELEASE_BIN" "$INSTALL_DIR/comby"
else
  cp "$TMP/$RELEASE_BIN" "$INSTALL_DIR/comby"
fi

SUCCESS_IN_PATH=$(command -v comby || echo notinpath)

if [ "$SUCCESS_IN_PATH" == "notinpath" ]; then
  printf "[*] Comby is not in your PATH. You should add %s to your PATH.\n" "$INSTALL_DIR"
  rm -f "$TMP/$RELEASE_BIN"
  exit 1
fi

CHECK=$(printf 'printf("hello world!\\\n");' | "$INSTALL_DIR"/comby 'printf("hello :[1]!\\n");' 'printf("hello comby!\\n");' .c -stdin || echo broken)
if [ "$CHECK" == "broken" ]; then
  printf "[-] comby did not install correctly.\n"
  printf "[-] My guess is that you need to install the pcre library on your system. Try:\n"
  if [ $OS == "macos" ]; then
    printf "[*] brew install comby\n"
  else
    printf "[*] sudo apt-get install libpcre3-dev && bash <(curl -sL get.comby.dev)\n"
  fi
  rm -f "$TMP/$RELEASE_BIN"
  exit 1
fi

rm -f "$TMP/$RELEASE_BIN"
printf "[+] comby upgraded to %s\n" "$REQUIRE_VERSION"
