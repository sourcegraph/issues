#!/bin/bash

set -eu

# (cd web && NODE_ENV=production DISABLE_TYPECHECKING=true yarn run build)

# go generate ./cmd/frontend/internal/app/assets ./cmd/frontend/internal/app/templates
go install -buildmode exe -tags dist ./cmd/sourcegraph

# TODO(sqs): workaround for issue where package conf runs even in the RunAll process
export SRC_FRONTEND_INTERNAL=localhost:7078

unset CDPATH
cd "$(dirname "${BASH_SOURCE[0]}")"
PGDATABASE=cmd_sourcegraph_shared_test SITE_CONFIG_FILE=site-config.json EXTSVC_CONFIG_FILE=external-services-config.json sourcegraph
