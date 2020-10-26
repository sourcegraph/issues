#!/usr/bin/env bash

# This script ensures pkg/db/dbconn is only imported by services allowed to
# directly speak with the database.

echo "--- go dbconn import"

set -euf -o pipefail

allowed='^github.com/sourcegraph/sourcegraph/cmd/frontend|github.com/sourcegraph/sourcegraph/enterprise/cmd/frontend|github.com/sourcegraph/sourcegraph/enterprise/cmd/repo-updater|github.com/sourcegraph/sourcegraph/enterprise/cmd/precise-code-intel-*'
# shellcheck disable=SC2016
template='{{with $pkg := .}}{{ range $pkg.Deps }}{{ printf "%s imports %s\n" $pkg.ImportPath .}}{{end}}{{end}}'

if go list ./../../cmd/... ../../enterprise/cmd/... |
  grep -Ev "$allowed" |
  xargs go list -f "$template" |
  grep "github.com/sourcegraph/sourcegraph/internal/db/dbconn"; then
  echo "Error: the above service(s) are not allowed to import pkg/db/dbconn"
  exit 1
fi
