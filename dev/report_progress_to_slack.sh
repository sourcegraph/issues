#!/bin/bash
#
# Usage:
#
#   If you want to change the behavior of this script (e.g., the formatting of the Slack message), you can run
#   this script directly in your local dev environment like this (setting DIFF_PATH=. will post all added lines
#   in the entire diff of the latest commit, not just CHANGELOG.md, to the Slack message):
#
#     WEBHOOK_URL=$WEBHOOK_TESTING_CHANNEL_URL DIFF_PATH=. ./report_progress_to_slack.sh
#
#   In prod, run WEBHOOK_URL=$WEBHOOK_PROGRESS_CHANNEL_URL DIFF_PATH=CHANGELOG.md ./report_progress_to_slack.sh

set -e

test ! -z "$WEBHOOK_URL"
test ! -z "$DIFF_PATH" # typically CHANGELOG.md, but use . when testing to diff all files
cd "$(dirname "${BASH_SOURCE[0]}")/.."

export COMMIT=HEAD
git diff $COMMIT~1..$COMMIT -- "$DIFF_PATH" >/tmp/diff.txt
if [ -z "$(cat /tmp/diff.txt)" ]; then
  echo "Diff was empty, not posting to Slack"
  exit 0
fi

tail -n +6 </tmp/diff.txt | grep "^\+[^\+]" | cut -c 2- >/tmp/added.txt
tail -n +6 </tmp/diff.txt | grep "^\-" | cut -c 2- >/tmp/removed.txt

if [ -z "$(cat /tmp/added.txt)" ]; then
  echo "Items were only removed, not posting to Slack"
  exit 0
fi

HASH="$(git log -n1 --pretty=format:%h)"
AUTHOR="$(git log -n1 --pretty=format:%an)"
echo ":rocket: Changelog added to by $AUTHOR in commit <https://github.com/sourcegraph/sourcegraph/commit/$HASH|$HASH> (<https://github.com/sourcegraph/sourcegraph/blob/main/.github/workflows/progress.yml|edit bot>):" >/tmp/message.txt
if [ -n "$(cat /tmp/added.txt)" ]; then
  cat /tmp/added.txt >>/tmp/message.txt
fi
if [ -n "$(cat /tmp/removed.txt)" ]; then
  echo "(Some items were also removed. Click the commit link for more details.)" >>/tmp/message.txt
fi

cat /tmp/message.txt
python -c 'import json,sys; print(json.dumps(sys.stdin.read()))' </tmp/message.txt >/tmp/message.json

if [ -n "$(cat /tmp/diff.txt)" ]; then
  echo "Posting added items to #progress channel in Slack"
  curl -XPOST "$WEBHOOK_URL" -d "{ \"text\": $(cat /tmp/message.json) }"
fi

# Clean up
rm -f /tmp/message.txt /tmp/message.json /tmp/diff.txt /tmp/added.txt /tmp/removed.txt
