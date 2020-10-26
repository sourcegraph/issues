#!/usr/bin/env bash

# UGH
export CLOUDSDK_PYTHON=/usr/bin/python3

set -eu
cd "$(dirname "${BASH_SOURCE[0]}")/../../../.."
DATADIR=$(realpath './internal/cmd/precise-code-intel-tester/testdata')
INDEXDIR="${DATADIR}/indexes"

# Ensure target dir exists
mkdir -p "${INDEXDIR}"

# Download all comprssed index files in parallel
gsutil -m cp gs://precise-code-intel-integration-testdata/* "${INDEXDIR}"
gunzip "${INDEXDIR}"/*
