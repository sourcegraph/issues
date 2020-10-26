#!/usr/bin/env bash

set -e
cd "$(dirname "${BASH_SOURCE[0]}")/.." # cd to repo root dir

generate_graphql=false
generate_dashboards=false
generate_monitoring=false
generate_schema=false
cmdlist=()
all_cmds=false
failed=false

for i; do
  case $i in
    "cmd/frontend/graphqlbackend/schema.graphql")
      generate_graphql=true
      ;;
    docker-images/grafana/jsonnet/*.jsonnet)
      generate_dashboards=true
      ;;
    monitoring/*)
      generate_monitoring=true
      ;;
    schema/*.json)
      generate_schema=true
      ;;
    cmd/*)
      cmd=${i#cmd/}
      cmd=${cmd%%/*}
      case " ${cmdlist[*]} " in
        " $cmd ") ;;

        *)
          cmdlist+=("$cmd")
          ;;
      esac
      ;;
    enterprise/cmd/*)
      cmd=${i#enterprise/cmd/}
      cmd=${cmd%%/*}
      case " ${cmdlist[*]} " in
        " $cmd ") ;;

        *)
          cmdlist+=("$cmd")
          ;;
      esac
      ;;
    *)
      all_cmds=true
      ;;
  esac
done

$generate_graphql && { go generate github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend || failed=true; }
$generate_dashboards && { docker-images/grafana/jsonnet/build.sh || failed=true; }
$generate_monitoring && { pushd monitoring >/dev/null && DEV=true go generate && popd >/dev/null || failed=true; }
$generate_schema && { go generate github.com/sourcegraph/sourcegraph/schema || failed=true; }

if $all_cmds; then
  if ! mapfile -t rebuilt < <(./dev/go-install.sh -v); then
    failed=true
  fi
elif [ ${#cmdlist[@]} -gt 0 ]; then
  if ! mapfile -t rebuilt < <(./dev/go-install.sh -v "${cmdlist[@]}"); then
    failed=true
  fi
fi

if [ ${#rebuilt[@]} -gt 0 ]; then
  echo >&2 "Rebuilt: ${rebuilt[*]}"

  if [ -n "$GOREMAN" ]; then
    for cmd in "${rebuilt[@]}"; do
      if $GOREMAN run list | grep -Ee "^${cmd}$"; then
        $GOREMAN run restart "${cmd}"
      fi
    done
  fi

else
  echo >&2 "Nothing to rebuild or rebuilds failed."
fi

! $failed
