# Buildkite Pipeline for sourcegraph/sourcegraph

We dynamically generate our CI pipeline for https://buildkite.com/sourcegraph/sourcegraph based on the output of [gen-pipeline.go](./gen-pipeline.go).

## Setup

In the [pipeline settings](https://buildkite.com/sourcegraph/sourcegraph/settings) ensure there is the following step:

```shell
go run ./enterprise/dev/ci/gen-pipeline.go | buildkite-agent pipeline upload
```

## Testing

To test this you can run `env BUILDKITE_BRANCH=TESTBRANCH go run ./enterprise/dev/ci/gen-pipeline.go` and inspect the YAML output. To change the behaviour set the relevant `BUILDKITE_` environment variables.

(c) Copyright Sourcegraph 2012-2023.