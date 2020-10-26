#!/usr/bin/env bash

# Bash script to query sourcegraph search API via graphql. It only returns
# meta information (such as result count), not the actual results. This is
# especially useful for viewing traces with large result.

# shellcheck disable=SC2016
query='query($q: String!) {
  site {
    buildVersion
  }
  search(query: $q) {
    results {
      limitHit
      resultCount
      elapsedMilliseconds
      ...SearchResultsAlertFields
    }
  }
}

fragment SearchResultsAlertFields on SearchResults {
  alert {
    title
    description
    proposedQueries {
      description
      query
    }
  }
}'

q="$1"
body="$(jq -n --arg query "$query" --arg q "$q" '{"query": $query, "variables": {"q": $q}}')"
endpoint=${SRC_ENDPOINT:-https://sourcegraph.com}

# Create and capture request/response headers
headers="$(mktemp -d)" || exit 1
trap 'rm -rf "$headers"' EXIT

echo 'X-Sourcegraph-Should-Trace: 1' >>"$headers/request"
if [ -n "$SRC_ACCESS_TOKEN" ]; then
  echo "Authorization: token $SRC_ACCESS_TOKEN" >>"$headers/request"
fi

curl --silent \
  -H "@$headers/request" \
  -d "$body" \
  -D "$headers/response" \
  "$endpoint/.api/graphql?SearchMeta" | jq .

grep x-trace "$headers/response"
