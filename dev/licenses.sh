#!/usr/bin/env bash

# This script either generates a report of third-party dependencies, or runs a check that fails
# if there are any unapproved dependencies ('action items').
#
# Please refer to the handbook entry for more details: https://about.sourcegraph.com/handbook/engineering/continuous_integration#third-party-licenses

set -euf -o pipefail

# by default, generate a report. this does not care if there are pending action items
COMMAND="report --format=csv --save=./third-party-licenses/ThirdPartyLicenses.csv --write-headers"

# if LICENSE_CHECK=true, report unapproved dependencies and error if there are any ('action items')
if [[ "${LICENSE_CHECK:-''}" == "true" ]]; then
  COMMAND="action_items"
fi

tmpdir=$(mktemp -d -t src-gocache.XXXXXXXX)
export GOPATH=$tmpdir # stuff in cache causes strange things to happen
echo "Using $(go env GOPATH) as GOPATH"
function cleanup() {
  go clean -modcache # need to remove modcache from tmpdir before we can remove
  echo "Removing $tmpdir"
  rm -rf "$tmpdir"
}
trap cleanup EXIT

# prepare dependencies
go mod tidy
go mod vendor # go mod download does not work with license_finder
yarn --mutex network --frozen-lockfile

# report license_finder configuration
license_finder permitted_licenses list
license_finder restricted_licenses list
license_finder ignored_groups list
license_finder ignored_dependencies list
license_finder dependencies list

# run license check
echo "Running license_finder - if this fails, refer to our handbook: https://about.sourcegraph.com/handbook/engineering/continuous_integration#third-party-licenses"
license_finder ${COMMAND} --columns=package_manager name version licenses homepage approved
