# TypeScript and JavaScript

This guide is meant to provide specific instructions to get you producing index data in LSIF as quickly as possible. The [LSIF quick start](../lsif_quickstart.md) and [CI configuration](../how-to/adding_lsif_to_workflows.md) guides provide more in depth descriptions of each step and a lot of helpful context that we haven't duplicated in each language guide.

## Manual indexing

1. Install [lsif-node](https://github.com/sourcegraph/lsif-node) with `npm install -g @sourcegraph/lsif-tsc` or your favorite method of installing npm packages.

1. Install the [Sourcegraph CLI](https://github.com/sourcegraph/src-cli) with

   ```
   curl -L https://sourcegraph.com/.api/src-cli/src_linux_amd64 -o /usr/local/bin/src
   chmod +x /usr/local/bin/src
   ```

   - **macOS**: replace `linux` with `darwin` in the URL
   - **Windows**: visit [the CLI's repo](https://github.com/sourcegraph/src-cli) for further instructions

1. `cd` into your project's root (where the package.json/tsconfig.json) and run the following:

   ```
   # for typescript projects
   lsif-tsc -p .
   # for javascript projects
   lsif-tsc **/*.js --allowJs --checkJs
   ```

   Check out the tool's documentation if you're having trouble getting `lsif-tsc` to work. It accepts any options `tsc` does, so it shouldn't be too hard to get it running on your project.

1. Upload the data to a Sourcegraph instance with

   ```
   # for private instances
   src -endpoint=<your sourcegraph endpoint> lsif upload
   # for public instances
   src lsif upload -github-token=<your github token>
   ```
   Visit the [LSIF quickstart](../lsif_quickstart.md) for more information about the upload command.

The upload command will provide a URL you can visit to see the upload's status, and when it's done you can visit the repo and check out the difference in code navigation quality! To troubleshoot issues, visit the more in depth [LSIF quickstart](../lsif_quickstart.md) guide and check out the documentation for the `lsif-node` and `src-cli` tools.

## Automated indexing

We provide the docker images `sourcegraph/lsif-node` and `sourcegraph/src-cli` to make automating this process in your favorite CI framework as easy as possible. Note that the `lsif-node` image bundles `src-cli` so the second image may not be necessary.

Here's some examples in a couple popular frameworks, just substitute the indexer and upload commands with what works for your project locally:

### GitHub Actions

```yaml
jobs:
  lsif-node:
    # this line will prevent forks of this repo from uploading lsif indexes
    if: github.repository == '<insert your repo name>'
    runs-on: ubuntu-latest
    container: sourcegraph/lsif-node:latest
    steps:
      - uses: actions/checkout@v1
      - name: Install dependencies
        run: npm install
      - name: Generate LSIF data
        run: lsif-tsc -p .
      - name: Upload LSIF data
        # this will upload to Sourcegraph.com, you may need to substitute a different command
        # by default, we ignore failures to avoid disrupting CI pipelines with non-critical errors.
        run: src lsif upload -github-token=${{ secrets.GITHUB_TOKEN }}
```

Note that if you need to install your dependencies in a custom container, you can use our containers as github actions. Try these steps instead:

```yaml
jobs:
  lsif-node:
    # this line will prevent forks of this repo from uploading lsif indexes
    if: github.repository == '<insert your repo name>'
    runs-on: ubuntu-latest
    container: my-awesome-container
    steps:
      - uses: actions/checkout@v1
      - name: Install dependencies
        run: <install dependencies>
      - name: Generate LSIF data
        uses: docker://sourcegraph/lsif-node:latest
        with:
          args: lsif-tsc -p .
      - name: Upload LSIF data
        uses: docker://sourcegraph/src-cli:latest
        with:
          # this will upload to Sourcegraph.com, you may need to substitute a different command
          # by default, we ignore failures to avoid disrupting CI pipelines with non-critical errors.
          args: lsif upload -github-token=${{ secrets.GITHUB_TOKEN }}
```

The following projects have example GitHub Action workflows to generate and upload LSIF indexes.

- [elastic/kibana](https://github.com/sourcegraph-codeintel-showcase/kibana/blob/7ed559df0e2036487ae6d606e9ffa29d90d49e38/.github/workflows/lsif.yml)
- [lodash/lodash](https://github.com/sourcegraph-codeintel-showcase/lodash/blob/b90ea221bd1b1e036f2dfcd199a2327883f9451f/.github/workflows/lsif.yml)
- [ReactiveX/IxJS](https://github.com/sourcegraph-codeintel-showcase/IxJS/blob/e53d323314043afb016b6deceaeb068d8d23c303/.github/workflows/lsif.yml)

### CircleCI

```yaml
version: 2.1

jobs:
  lsif-node:
    docker:
      - image: sourcegraph/lsif-node:latest
    steps:
      - checkout
      - run: npm install
      - run: lsif-tsc -p .
        # this will upload to Sourcegraph.com, you may need to substitute a different command
        # by default, we ignore failures to avoid disrupting CI pipelines with non-critical errors.
      - run: src lsif upload -github-token=<<parameters.github-token>>

workflows:
  lsif-node:
    jobs:
      - lsif-node
```

Note that if you need to install your dependencies in a custom container, may need to use CircleCI's caching features to share the build environment with our container. It may alternately be easier to add our tools to your container, but here's an example using caches:

```yaml
jobs:
  install-deps:
    docker:
      - image: my-awesome-container
    steps:
      - checkout
      - <install dependencies>
      - save_cache:
          paths:
            - node_modules
          key: dependencies

jobs:
  lsif-node:
    docker:
      - image: sourcegraph/lsif-node:latest
    steps:
      - checkout
      - restore_cache:
          keys:
            - dependencies
      - run: lsif-tsc -p .
        # this will upload to Sourcegraph.com, you may need to substitute a different command
        # by default, we ignore failures to avoid disrupting CI pipelines with non-critical errors.
      - run: src lsif upload -github-token=<<parameters.github-token>>

workflows:
  lsif-node:
    jobs:
      - install-deps
      - lsif-node:
          requires:
            - install-deps
```

The following projects have example CircleCI configurations to generate and upload LSIF indexes.

- [angular/angular](https://github.com/sourcegraph-codeintel-showcase/angular/blob/f06eec98cadab2ff7a1cef2a03ba7c42015eb399/.circleci/config.yml)
- [facebook/jest](https://github.com/sourcegraph-codeintel-showcase/jest/blob/b781fa2b6683f04324edbc4b41552a94f97cd479/.circleci/config.yml)
- [facebook/react](https://github.com/sourcegraph-codeintel-showcase/react/blob/e488420f686b88803cfb1bb09bbc4d3991db8c55/.circleci/config.yml)
- [grafana](https://github.com/sourcegraph-codeintel-showcase/grafana/blob/664a694955ea40575a1cffe9db47a7adf4d3c2bb/.circleci/config.yml)
- [ReactiveX/rxjs](https://github.com/sourcegraph-codeintel-showcase/rxjs/blob/c9d3c1a76a68273863fc59075a71b4cc43c06114/.circleci/config.yml)

### Travis CI

```yaml
services:
  - docker

jobs:
  include:
    - stage: lsif-node
      # this will upload to Sourcegraph.com, you may need to substitute a different command
      # by default, we ignore failures to avoid disrupting CI pipelines with non-critical errors.
      script:
      - |
        docker run --rm -v $(pwd):/src -w /src sourcegraph/lsif-node:latest /bin/sh -c \
          "lsif-tsc -p .; src lsif upload -github-token=$GITHUB_TOKEN"
```

The following projects have example Travis CI configurations to generate and upload LSIF indexes.

- [expressjs/express](https://github.com/sourcegraph-codeintel-showcase/express/blob/bd1ae153f19656183257ed223d518aeb9f5091ec/.travis.yml)
- [Microsoft/TypeScript](https://github.com/sourcegraph-codeintel-showcase/TypeScript/blob/f37f1dee1b3e63b12df2935590c8707a5ec3993b/.travis.yml)
- [moment/moment](https://github.com/sourcegraph-codeintel-showcase/moment/blob/eedccdc2c07fb5abe931b427d50f5b3c3f44ac95/.travis.yml)
- [sindresorhus/got](https://github.com/sourcegraph-codeintel-showcase/got/blob/164d55a029512cea7f245de870cbb1eaba114734/.travis.yml)
