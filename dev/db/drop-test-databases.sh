#!/usr/bin/env bash

set -eo pipefail

for dbname in $(psql -c "copy (select datname from pg_database where datname like 'sourcegraph-test-%') to stdout"); do
  dropdb "$dbname"
  echo "dropped $dbname"
done
