#!/usr/bin/env bash

set -eo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/../../migrations"

hash migrate 2>/dev/null || {
  if [[ $(uname) == "Darwin" ]]; then
    brew install golang-migrate
  else
    echo "You need to install the 'migrate' tool: https://github.com/golang-migrate/migrate/"
    exit 1
  fi
}

if [ -z "$2" ]; then
  echo "USAGE: $0 <db_name> <tag>"
  echo ""
  echo "This tool will squash all migrations up to and including the last migration defined"
  echo "in the given tag branch. The input to this tool should be three minor releases before"
  echo "the current release. For example, if we're currently on 3.10, the input should be the"
  echo "tag 'v3.7.0' as we need to maintain compatibility with versions 3.8 and 3.9."
  echo ""
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

target='./'
if [ -z "$(git ls-tree -r --name-only "$2" "./")" ]; then
  if [ "$1" != "frontend" ]; then
    echo "database does not exist at this version - nothing to squash"
    exit 0
  fi

  # If we're squashing migrations no a tagged version where the
  # migrations/frontend directory does not exist, scan the files
  # in the parent directory where they were located previously.
  target='../'
fi

# Find the last migration defined in the given tag
VERSION=$(
  git ls-tree -r --name-only "$2" "${target}" |
    cut -d'_' -f1 |    # Keep only prefix
    cut -d'/' -f2 |    # Remove any leading ../
    grep -v "[^0-9]" | # Remove non-numeric remainders
    sort |             # Sort by id prefix
    tail -n1           # Get latest migration
)

if [ -z "${VERSION}" ]; then
  echo "failed to retrieve migration version"
  exit 1
fi

DBNAME='squasher'
SERVER_VERSION=$(psql --version)

if [ "${SERVER_VERSION}" != 9.6 ]; then
  echo "running PostgreSQL 9.6 in docker since local version is ${SERVER_VERSION}"
  docker image inspect postgres:9.6 >/dev/null || docker pull postgres:9.6
  docker rm --force "${DBNAME}" 2>/dev/null || true
  docker run --rm --name "${DBNAME}" -p 5433:5432 -e POSTGRES_HOST_AUTH_METHOD=trust -d postgres:9.6

  function kill() {
    docker kill "${DBNAME}" >/dev/null
  }
  trap kill EXIT

  sleep 5
  docker exec -u postgres "${DBNAME}" createdb "${DBNAME}"
  export PGHOST=127.0.0.1
  export PGPORT=5433
  export PGDATABASE="${DBNAME}"
  export PGUSER=postgres
fi

# First, apply migrations up to the version we want to squash
migrate -database "postgres://${PGHOST}:${PGPORT}/${PGDATABASE}?sslmode=disable&x-migrations-table=${migrations_table}" -path . goto "${VERSION}"

# Dump the database into a temporary file that we need to post-process
pg_dump --schema-only --no-owner --no-comments --exclude-table='*schema_migrations' -f tmp_squashed.sql

# Remove settings header from pg_dump output
sed -i '' -e 's/^SET .*$//g' tmp_squashed.sql
sed -i '' -e 's/^SELECT pg_catalog.set_config.*$//g' tmp_squashed.sql

# Do not drop extensions if they already exist. This causes some
# weird problems with the back-compat tests as the extensions are
# not dropped in the correct order to honor dependencies.
sed -i '' -e 's/^DROP EXTENSION .*$//g' tmp_squashed.sql

# Remove references to public schema
sed -i '' -e 's/public\.//g' tmp_squashed.sql
sed -i '' -e 's/ WITH SCHEMA public//g' tmp_squashed.sql

# Remove comments, multiple blank lines
sed -i '' -e 's/^--.*$//g' tmp_squashed.sql
sed -i '' -e '/^$/N;/^\n$/D' tmp_squashed.sql

# Now clean up all of the old migration files. `ls` will return files in
# alphabetical order, so we can delete all files from the migration directory
# until we hit our squashed migration.

for file in *.sql; do
  rm "$file"
  echo "squashed migration $file"

  # There should be two files prefixed with this schema version. The down
  # version comes first, then the up version. Make sure we only break the
  # loop once we remove both files.

  if [[ "$file" == "${VERSION}"* && "$file" == *'up.sql' ]]; then
    break
  fi
done

# Wrap squashed migration in transaction
printf "BEGIN;\n" >"./${VERSION}_squashed_migrations.up.sql"
cat tmp_squashed.sql >>"./${VERSION}_squashed_migrations.up.sql"
printf "\nCOMMIT;\n" >>"./${VERSION}_squashed_migrations.up.sql"
rm tmp_squashed.sql

cat >"./${VERSION}_squashed_migrations.down.sql" <<EOL
DROP SCHEMA IF EXISTS public CASCADE;
CREATE SCHEMA public;

CREATE TABLE IF NOT EXISTS ${migrations_table} (
    version bigint NOT NULL PRIMARY KEY,
    dirty boolean NOT NULL
);
EOL

echo ""
echo "squashed migrations written to ${VERSION}_squashed_migrations.{up,down}.sql"

# Regenerate bindata
go generate

# Update test with new lowest migration
sed -i '' "s/const FirstMigration = [0-9]*/const FirstMigration = ${VERSION}/" ./migrations_test.go
