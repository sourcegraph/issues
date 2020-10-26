#!/bin/bash
set -e

export GF_PATHS_PROVISIONING=/sg_config_grafana/provisioning
export GF_PATHS_CONFIG=/sg_config_grafana/grafana.ini

exec "/run.sh"
