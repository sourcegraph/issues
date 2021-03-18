# Sourcegraph release tool

This directory contains scripts and code to automate our releases. Refer to
[the handbook](https://about.sourcegraph.com/handbook/engineering/releases) for details
on our release process and how this tool is used.

To see all available steps:

```sh
yarn run release help # add 'all' to see test commands as well
```

Before using this tool, please verify that the [release configuration](./release-config.jsonc)
is set up correctly.

(c) Copyright Sourcegraph 2012-2023.