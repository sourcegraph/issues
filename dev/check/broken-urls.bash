#!/usr/bin/env bash

set -e
cd "$(dirname "${BASH_SOURCE[0]}")"/../..

# Skip on non-default branches to avoid preventing us from building historical commits (eg when
# backporting fixes).
if [[ ! "$BUILDKITE_BRANCH" =~ ^(master|main)$ ]]; then
    exit 0
fi

URL_MATCHES=$(git grep -h -e https://about.sourcegraph.com --and --not -e '^\s*//' --and --not -e 'CI\:URL_OK' -- '*.go' '*.js' '*.jsx' '*.ts' '*.tsx' '*.json' ':(exclude)vendor' | grep -Eo 'https://about.sourcegraph.com[^'"'"'`)>" ]+' | sed 's/\.$//' | sort -u)

for url in $URL_MATCHES; do
    if ! curl -fsSL -o /dev/null --max-time 5 --retry 3 --retry-max-time 5 --retry-delay 1 "$url"; then
        echo '     ' while trying to fetch "$url"
        echo
        BAD_URLS="${BAD_URLS} $url"
    fi
done

if [ -n "$BAD_URLS" ]; then
    echo
    echo "Error: Found broken about.sourcegraph.com URLs:"
    echo "$BAD_URLS" | sed 's/ /\n/g' | sed 's/^/  /'

    cat <<EOF

If the URL is really valid, add the string "CI:URL_OK" (in a comment) to the line(s) where the URL appears.

EOF

    exit 1
fi
