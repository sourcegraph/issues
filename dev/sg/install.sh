#!/usr/bin/env bash

set -euf -o pipefail
pushd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null

go install .

echo "          _____                    _____          "
echo "         /\    \                  /\    \         "
echo "        /::\    \                /::\    \        "
echo "       /::::\    \              /::::\    \       "
echo "      /::::::\    \            /::::::\    \      "
echo "     /:::/\:::\    \          /:::/\:::\    \     "
echo "    /:::/__\:::\    \        /:::/  \:::\    \    "
echo "    \:::\   \:::\    \      /:::/    \:::\    \   "
echo "  ___\:::\   \:::\    \    /:::/    / \:::\    \  "
echo " /\   \:::\   \:::\    \  /:::/    /   \:::\ ___\ "
echo "/::\   \:::\   \:::\____\/:::/____/  ___\:::|    |"
echo "\:::\   \:::\   \::/    /\:::\    \ /\  /:::|____|"
echo " \:::\   \:::\   \/____/  \:::\    /::\ \::/    / "
echo "  \:::\   \:::\    \       \:::\   \:::\ \/____/  "
echo "   \:::\   \:::\____\       \:::\   \:::\____\    "
echo "    \:::\  /:::/    /        \:::\  /:::/    /    "
echo "     \:::\/:::/    /          \:::\/:::/    /     "
echo "      \::::::/    /            \::::::/    /      "
echo "       \::::/    /              \::::/    /       "
echo "        \::/    /                \::/____/        "
echo "         \/____/                                  "
echo "                                                  "
echo "                                                  "
echo "  sg installed to $(which sg)."
echo "                                                  "
echo "                  Happy hacking!"
