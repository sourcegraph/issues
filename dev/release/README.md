This directory contains scripts and code to automate our releases. Run `yarn run release` to see a
list of automated steps.

Run `yarn run build` to build, `yarn run watch` to build on any changes to files.

## Cutting a release

First, ensure you are on `main` and have the latest version of this code built:

```sh
git checkout main
git pull
cd dev/release
yarn install
yarn run build
```

To cut a patch release:

```sh
yarn run release patch:issue <version>
```

Or to cut a major release:

```sh
yarn run release tracking-issue:create <version>
```
