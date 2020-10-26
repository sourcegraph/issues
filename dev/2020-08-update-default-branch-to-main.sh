#!/bin/sh

# TODO remove this script in 1 week. Temporary transition script.

set -eux

echo "This script will update your Sourcegraph repository to our new default branch 'main'."
echo "See https://github.com/sourcegraph/sourcegraph/pull/11453 for more information."
echo

git checkout master
git branch -m master main
git fetch
git branch --unset-upstream
git branch -u origin/main
git symbolic-ref refs/remotes/origin/HEAD refs/remotes/origin/main

echo
echo "Your default branch is now main."
echo "See https://github.com/sourcegraph/sourcegraph/pull/11453 for more information."
echo
