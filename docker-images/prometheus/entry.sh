#!/bin/sh
set -e

CONFIG_FILE=/prometheus/sg_config/prometheus.yml

if test "$USE_KUBERNETES_DISCOVERY" = 'true'; then
    CONFIG_FILE=/prometheus/sg_config/prometheus_k8s.yml
fi

sh -c "/bin/prometheus --config.file=$CONFIG_FILE --storage.tsdb.path=/prometheus --web.console.libraries=/usr/share/prometheus/console_libraries --web.console.templates=/usr/share/prometheus/consoles"

