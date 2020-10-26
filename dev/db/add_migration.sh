#!/bin/bash

cd "$(dirname "${BASH_SOURCE[0]}")"/../../migrations
set -e

if [ -z "$2" ]; then
  echo "USAGE: $0 <db_name> <name>"
  exit 1
fi

if [ ! -d "$1" ]; then
  echo "Unknown database '$1'"
  exit 1
fi
pushd "$1" >/dev/null || exit 1

# This simulates what "migrate create -ext sql -digits 10 -seq" does.
awkcmd=$(
  cat <<-EOF
BEGIN { FS="[_/]" }
{ n=\$2 }
END {
    gsub(/[^A-Za-z0-9]/, "_", name);
    printf("%s_%s.up.sql\n",   n + 1, name);
    printf("%s_%s.down.sql\n", n + 1, name);
}
EOF
)

files=$(find . -type f -name '[0-9]*.sql' | sort | awk -v name="$2" "$awkcmd")

for f in $files; do
  cat >"$f" <<EOF
BEGIN;

-- Insert migration here. See README.md. Highlights:
--  * Always use IF EXISTS. eg: DROP TABLE IF EXISTS global_dep_private;
--  * All migrations must be backward-compatible. Old versions of Sourcegraph
--    need to be able to read/write post migration.
--  * Historically we advised against transactions since we thought the
--    migrate library handled it. However, it does not! /facepalm

COMMIT;
EOF

  echo "Created migrations/$1/$f"
done
