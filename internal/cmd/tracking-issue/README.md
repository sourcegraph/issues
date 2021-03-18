# Tracking issue tool

## Usage

You can read about how to use the tracking-issue tool in its [handbook page](https://about.sourcegraph.com/handbook/engineering/tracking_issues).

## Running

```console
Usage of ./tracking-issue:
  -dry
        If true, do not update GitHub tracking issues in-place, but print them to stdout
  -org string
        GitHub organization to list issues from (default "sourcegraph")
  -token string
        GitHub personal access token (default "$GITHUB_TOKEN")
  -verbose
        If true, print the resulting tracking issue bodies to stdout
```

## Deployment

The `tracking-issue` tool is deployed with [this GitHub action](../../../.github/workflows/tracking-issue.yml).

In order to deploy a new version, first run `docker build -t sourcegraph/tracking-issue .` and then `docker push sourcegraph/tracking-issue`.

## Testing

Run the tests with `go test`, update fixtures (i.e. GitHub issues and PRs data) with `go test -update.fixture` and update the generated tracking issue golden file with `go test -update`.

You can also run the tool manually in `-dry` mode with `-verbose` output to visualize the resulting tracking issues without updating them on GitHub.

(c) Copyright Sourcegraph 2012-2023.