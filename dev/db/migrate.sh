#!/usr/bin/env bash

set -eo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/../../migrations"

if [ -z "$1" ]; then
  echo "USAGE: $0 <db_name> [ <command> ]"
  exit 1
fi

if [ ! -d "$1" ]; then
  echo "Unknown database '$1'"
  exit 1
fi
pushd "$1" >/dev/null || exit 1

migrations_table='schema_migrations'
if [ "$1" != "frontend" ]; then
  migrations_table="$1_${migrations_table}"
fi

hash migrate 2>/dev/null || {
  if [[ $(uname) == "Darwin" ]]; then
    brew install golang-migrate
  else
    echo "You need to install the 'migrate' tool: https://github.com/golang-migrate/migrate/"
    exit 1
  fi
}

shift # get rid of db name
migrate -database "postgres://${PGHOST}:${PGPORT}/${PGDATABASE}?x-migrations-table=${migrations_table}" -path . "$@"
