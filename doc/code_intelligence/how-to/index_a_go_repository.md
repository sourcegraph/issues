# Go LSIF Indexing

This guide is meant to provide specific instructions to get you producing index data in LSIF as quickly as possible. The [LSIF quick start](../lsif_quickstart.md) and [CI configuration](adding_lsif_to_workflows.md) guides provide more in depth descriptions of each step and a lot of helpful context that we haven't duplicated in each language guide.

## Automated indexing

We provide the docker images `sourcegraph/lsif-go` and `sourcegraph/src-cli` to make automating this process in your favorite CI framework as easy as possible. Note that the `lsif-go` image bundles `src-cli` so the second image may not be necessary.

Here's some examples in a couple popular frameworks, just substitute the indexer and upload commands with what works for your project locally. If you end up implementing this in a different framework, feel free to edit this page with instructions!

### GitHub Actions

```yaml
on:
  - push

jobs:
  lsif-go:
    # this line will prevent forks of this repo from uploading lsif indexes
    if: github.repository == '<insert your repo name>'
    runs-on: ubuntu-latest
    container: sourcegraph/lsif-go:latest
    steps:
      - uses: actions/checkout@v1
      - name: Generate LSIF data
        run: lsif-go
      - name: Upload LSIF data
        # this will upload to Sourcegraph.com, you may need to substitute a different command.
        # by default, we ignore failures to avoid disrupting CI pipelines with non-critical errors.
        run: src lsif upload -github-token=${{ secrets.GITHUB_TOKEN }} -ignore-upload-failure
```

The following projects have example GitHub Action workflows to generate and upload LSIF indexes.

- [golang/go](https://github.com/sourcegraph-codeintel-showcase/go/blob/f40606b1241b0ca4802d7b00a763241b03404eea/.github/workflows/lsif.yml)
- [kubernetes/kubernetes](https://github.com/sourcegraph-codeintel-showcase/kubernetes/blob/359b6469d85cc7cd4f6634e50651633eefeaea4e/.github/workflows/lsif.yml)
- [moby/moby](https://github.com/sourcegraph-codeintel-showcase/moby/blob/380429abb05846de773d5aa07de052f40c9e8208/.github/workflows/lsif.yml)

### CircleCI

```yaml
version: 2.1

jobs:
  lsif-go:
    docker:
      - image: sourcegraph/lsif-go:latest
    steps:
      - checkout
      - run: lsif-go
        # this will upload to Sourcegraph.com, you may need to substitute a different command.
        # by default, we ignore failures to avoid disrupting CI pipelines with non-critical errors.
      - run: src lsif upload -github-token=<<parameters.github-token>> -ignore-upload-failure

workflows:
  lsif-node:
    jobs:
      - lsif-node
```

The following projects have example CircleCI configurations to generate and upload LSIF indexes.

- [grafana](https://github.com/sourcegraph-codeintel-showcase/grafana/blob/664a694955ea40575a1cffe9db47a7adf4d3c2bb/.circleci/config.yml)
- [helm](https://github.com/sourcegraph-codeintel-showcase/helm/blob/62c38f152d0802719aad1ec4c1c281f01dc75173/.circleci/config.yml)
- [prometheus/prometheus](https://github.com/sourcegraph-codeintel-showcase/prometheus/blob/a0a8a249fff9d1c6ce4c097ccc4f5e120c723c51/.circleci/config.yml)

### Travis CI

```yaml
services:
  - docker

jobs:
  include:
    - stage: lsif-go
      # this will upload to Sourcegraph.com, you may need to substitute a different command.
      # by default, we ignore failures to avoid disrupting CI pipelines with non-critical errors.
      script:
      - |
        docker run --rm -v $(pwd):/src -w /src sourcegraph/lsif-go:latest /bin/sh -c \
          "lsif-go; src lsif upload -github-token=$GITHUB_TOKEN -ignore-upload-failure"
```

The following projects have example Travis CI configurations to generate and upload LSIF indexes.

- [aws/aws-sdk-go](https://github.com/sourcegraph-codeintel-showcase/aws-sdk-go/blob/92f67a061fcdd46d6a418b28838b10b6ac63a880/.travis.yml)
- [etcd-io/etcd](https://github.com/sourcegraph-codeintel-showcase/etcd/blob/eae726706fe8ebf7e08b45ba29a70388595db31b/.travis.yml)
- [hugo/hugo](https://github.com/sourcegraph-codeintel-showcase/hugo/blob/6704b7c125d7b21ccf2048d7bff0f1ffe2b0867d/.travis.yml)

## Manual indexing

1. Install [lsif-go](https://github.com/sourcegraph/lsif-go) with `go get github.com/sourcegraph/lsif-go/cmd/lsif-go` and ensure it's on your path.

1. Install the [Sourcegraph CLI](https://github.com/sourcegraph/src-cli) with

   ```
   curl -L https://sourcegraph.com/.api/src-cli/src_linux_amd64 -o /usr/local/bin/src
   chmod +x /usr/local/bin/src
   ```

   - **macOS**: replace `linux` with `darwin` in the URL
   - **Windows**: visit [the CLI's repo](https://github.com/sourcegraph/src-cli) for further instructions

1. `cd` into your Go project's root (where the go.mod file lives, if you have one) and run:

   ```
   lsif-go # generates a file named dump.lsif
   ```

1. Upload the data to a Sourcegraph instance with

   ```
   # for private instances
   src -endpoint=<your sourcegraph endpoint> lsif upload
   # for public instances
   src lsif upload -github-token=<your github token>
   ```

   Visit the [LSIF quickstart](../lsif_quickstart.md) for more information about the upload command.

The upload command will provide a URL you can visit to see the upload's status, and when it's done you can visit the repo and check out the difference in code navigation quality! To troubleshoot issues, visit the more in depth [LSIF quickstart](../lsif_quickstart.md) guide and check out the documentation for the `lsif-go` and `src-cli` tools.
