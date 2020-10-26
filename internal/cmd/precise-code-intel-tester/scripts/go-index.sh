#!/usr/bin/env bash

set -eux
cd "$(dirname "${BASH_SOURCE[0]}")/../../../.."
DATADIR=$(realpath './internal/cmd/precise-code-intel-tester/testdata')
REPODIR="${DATADIR}/repos"
INDEXDIR="${DATADIR}/indexes"

NAME="$1"
REV="$2"

REVDIR="${REPODIR}/${NAME}-${REV}"
INDEXFILE="${INDEXDIR}/${NAME}.${REV}.dump"

# Early-out if there's already a dump file
if [ -f "${INDEXFILE}" ]; then
  exit 0
fi

# Ensure target dir exists
mkdir -p "${INDEXDIR}"

# Copy repo to temporary directory
cp -r "${REPODIR}/${NAME}" "${REVDIR}"
cleanup() {
  rm -rf "${REVDIR}"
}
trap cleanup EXIT

# Check out revision
pushd "${REVDIR}" || exit 1
git checkout "${REV}" 2>/dev/null

# Index revision
go mod vendor && lsif-go -o "${INDEXFILE}"
V=$?
exit $V
