#!/bin/sh
set -e

# shellcheck disable=SC2086
exec /bin/alertmanager --storage.path=/alertmanager --data.retention=168h $ALERTMANAGER_ADDITIONAL_FLAGS "$@"
