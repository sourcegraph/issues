# Precise code intel tester

This package provides integration and load testing utilities for precise code intel services.

## Prerequisites

Ensure that the following tools are available on your path:

- [`src`](https://github.com/sourcegraph/src-cli)
- [`lsif-go`](https://github.com/sourcegraph/lsif-go)
- [`gsutil`](https://cloud.google.com/storage/docs/gsutil_install) (and authenticated to the `sourcegraph-dev` project)

## Testing

1. Ensure these repositories exist on your instance (in `Site Admin` -> `Manage repositories` -> `GitHub`):

```
  "repos": [
    "sourcegraph-testing/etcd",
    "sourcegraph-testing/tidb",
    "sourcegraph-testing/titan",
    "sourcegraph-testing/zap"
  ],
```

2. Download the test indexes by running the following command:

```
./scripts/download.sh
```

Alternatively, generate them by running the following command (this takes much longer):

```
./scripts/clone-and-index.sh
```

Upload the indexes to your the target instance by running the following command:

```
go build && ./precise-code-intel-tester upload
```

Then run test queries against the target instance by running the following command:

```
go build && ./precise-code-intel-tester query
```

## Refreshing indexes

If there is a change to an indexer that needs to be tested, the indexes can be regenerated and uploaded to gcloud for future test runs.

Generate indexes by running the following command:

```
./scripts/clone-and-index.sh
```

Upload the generated indexes by running the following command:

```
./scripts/upload.sh
```

(c) Copyright Sourcegraph 2012-2023.