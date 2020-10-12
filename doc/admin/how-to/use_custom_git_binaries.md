# Custom git binaries

This is most common for [large monorepos](../background-information/monorepo.md)

## Install the custom binary

This can be done by inheriting from the `gitserver` docker image and installing the custom `git` onto the `$PATH`.

## Optimize `git fetch`

Some monorepos use a custom command for `git fetch` to speed up fetch. Sourcegraph provides the `experimentalFeatures.customGitFetch` site setting to specify the custom command.
