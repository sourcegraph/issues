# How LSIF indexes are processed

An LSIF indexer produces a file containing the definition, reference, hover, and diagnostic data for a project. Users upload this index file to a Sourcegraph instance, which converts it into an internal format that can support [code intelligence queries](./queries.md).

The sequence of actions required to to upload and convert this data is shown below (click to enlarge).

<a href="diagrams/upload.svg" target="_blank">
  <img src="diagrams/upload.svg">
</a>

## Uploading

The API used to upload an LSIF index is modeled after the [S3 multipart upload API](https://docs.aws.amazon.com/AmazonS3/latest/dev/mpuoverview.html). Many LSIF uploads can be fairly large and the [network is generally not reliable](https://aphyr.com/posts/288-the-network-is-reliable). To get around frequent failure of large uploads (and to get around uploads limits in Cloudflare), the upload is broken into multiple, independently gzipped chunks. Each chunk is uploaded in sequence to the instances, where it is concatenated into a single file on the remote end. This allows us to retry chunks independently in the case of an upload failure without sacrificing the entire operation.

An initial request adds an upload into the database with the `uploading` state and marks the number of upload chunks it expects to see. The subsequent requests specify the upload identifier (returned in the initial request), and the index of the chunk that is being uploaded. If this upload part successfully makes it to disk, it is marked as received in the upload record. The last request is a request marking upload completion from the client. At this point, the frontend ensures that all the expected chunks have been received and reside on disk. The frontend informs the bundle manager to concatenate the files, and the upload record is moved from the `uploading` state to the `queued` state, where it is made visible to the worker process.

## Processing

The worker process polls Postgres for upload records in the `queued` state. When such a record is available, it is marked as `processing` and is locked in a transaction to ensure that it is not double-processed by another worker instance. The worker asks the bundle manager for the raw LSIF upload data. Because this data is generally large, the data is streamed to the worker while it is being processed (and retry logic inside the bundle manager client will retry the request from the last byte it received on transient failures).

The worker then converts the raw LSIF data into a SQLite database, producing a set of packages that the indexed source code _defines_ and a set of packages that the indexed source code _depends on_. This [portion of the conversion](https://sourcegraph.com/github.com/sourcegraph/sourcegraph/-/blob/enterprise/cmd/precise-code-intel-worker/internal/correlation/correlate.go#L20:6) is omitted from the diagram as it remains within the worker process (with one exception), but is explained below.

1. The [correlateFromReader](https://sourcegraph.com/github.com/sourcegraph/sourcegraph/-/blob/enterprise/cmd/precise-code-intel-worker/internal/correlation/correlate.go#L73:6) step streams raw LSIF data from the bundle manager and produces a stream of JSON objects. Each object in the stream is interpreted as an LSIF vertex or edge. Objects are validated, then inserted into an in-memory representation of the graph.
1. The [canonicalize](https://sourcegraph.com/github.com/sourcegraph/sourcegraph/-/blob/enterprise/cmd/precise-code-intel-worker/internal/correlation/canonicalize.go#L12:6) step collapses the in-memory representation of the graph produced by the previous step. Most notably, it ensures that the data attached to a range vertex _transitively_ is now attached to the range vertex _directly_.
1. The [prune](https://sourcegraph.com/github.com/sourcegraph/sourcegraph/-/blob/enterprise/cmd/precise-code-intel-worker/internal/correlation/prune.go#L14:6) step determines the set of documents that are present in the index but do not exist in git (via an efficient batch of calls to gitserver) and removes references to them from the in-memory representation of the graph. This prevents us from attempting to navigate to locations that are not visible within the instance (generated or vendored paths that are not committed).
1. The [groupBundleData](https://sourcegraph.com/github.com/sourcegraph/sourcegraph/-/blob/enterprise/cmd/precise-code-intel-worker/internal/correlation/group.go#L34:6) step converts the canonicalized and pruned in-memory representation of the graph into the shape that will reside within a SQLite bundle. This _rotates_ the data so that it can be efficiently read based on our [query access patterns](./queries.md).
1. The [sqlite writer](https://sourcegraph.com/github.com/sourcegraph/sourcegraph/-/blob/enterprise/internal/codeintel/bundles/persistence/sqlite/writer.go) writes the grouped bundle data from the previous step into a new SQLite database on disk.

The set of packages defined by and depended on by this index can be constructed from reading the package information attached to export and import monikers, respectively, from the correlated data. This data is inserted into Postgres to enable cross-repository definition and reference queries. 

Duplicate uploads (with the same repository, commit, and root) are removed to prevent the frontend from querying multiple indexes for the same data. This can happen if a user re-uploads the same index, or if an index is re-uploaded as part of a CI step that was re-run. In these cases we prefer to keep the newest upload.

The repository is marked as _dirty_, which informs a process that runs periodically to re-calculate the set of uploads visible to each commit. This process will refresh the commit graph for this repository stored in Postgres.

The SQLite database is sent to the bundle manager in chunks, as described in the previous section. 

Finally, if the previous steps have all completed without error, the transaction is committed, moving the upload record from the `processing` state to the `completed` state, where it is made visible to the frontend to answer code intelligence queries. If an error does occur, the upload record is instead moved to the `errored` state and marked with a failure reason.

## Code appendix

- src-cli: [lsif upload command](https://sourcegraph.com/github.com/sourcegraph/src-cli@main/-/blob/cmd/src/lsif_upload.go#L153:2)
- Worker: [abstract process](https://sourcegraph.com/github.com/sourcegraph/sourcegraph/-/blob/internal/workerutil/worker.go#L16:6), [upload processor](https://sourcegraph.com/github.com/sourcegraph/sourcegraph/-/blob/enterprise/cmd/precise-code-intel-worker/internal/worker/handler.go#L43:19), [correlator](https://sourcegraph.com/github.com/sourcegraph/sourcegraph/-/blob/enterprise/cmd/precise-code-intel-worker/internal/correlation/correlate.go#L20:6) (the heavy hitter)
- Store: [Dequeue](https://sourcegraph.com/github.com/sourcegraph/sourcegraph/-/blob/internal/workerutil/dbworker/store/store.go#L202:17), [InsertUpload](https://sourcegraph.com/github.com/sourcegraph/sourcegraph/-/blob/enterprise/internal/codeintel/store/uploads.go#L294:17), [AddUploadPart](https://sourcegraph.com/github.com/sourcegraph/sourcegraph/-/blob/enterprise/internal/codeintel/store/uploads.go#L330:17), [UpdatePackages](https://sourcegraph.com/github.com/sourcegraph/sourcegraph/-/blob/enterprise/internal/codeintel/store/packages.go#L37:17), [UpdatePackageReferences](https://sourcegraph.com/github.com/sourcegraph/sourcegraph/-/blob/enterprise/internal/codeintel/store/references.go#L115:17), [DeleteOverlappingDumps](https://sourcegraph.com/github.com/sourcegraph/sourcegraph/-/blob/enterprise/internal/codeintel/store/dumps.go#L192:17), [MarkRepositoryAsDirty](https://sourcegraph.com/github.com/sourcegraph/sourcegraph/-/blob/enterprise/internal/codeintel/store/commits.go#L62:17)
- Bundle Manager:
  - SendUploadPart - [client](https://sourcegraph.com/github.com/sourcegraph/sourcegraph/-/blob/enterprise/internal/codeintel/bundles/client/bundle_manager_client.go#L142:35), [server](https://sourcegraph.com/github.com/sourcegraph/sourcegraph/-/blob/enterprise/cmd/precise-code-intel-bundle-manager/internal/server/handler.go#L83:18)
  - StitchParts - [client](https://sourcegraph.com/github.com/sourcegraph/sourcegraph/-/blob/enterprise/internal/codeintel/bundles/client/bundle_manager_client.go#L154:35), [server](https://sourcegraph.com/github.com/sourcegraph/sourcegraph/-/blob/enterprise/cmd/precise-code-intel-bundle-manager/internal/server/handler.go#L92:18)
  - GetUpload - [client](https://sourcegraph.com/github.com/sourcegraph/sourcegraph/-/blob/enterprise/internal/codeintel/bundles/client/bundle_manager_client.go#L175:350), [server](https://sourcegraph.com/github.com/sourcegraph/sourcegraph/-/blob/enterprise/cmd/precise-code-intel-bundle-manager/internal/server/handler.go#L53:18)
  - SendDB - [client](https://sourcegraph.com/github.com/sourcegraph/sourcegraph/-/blob/enterprise/internal/codeintel/bundles/client/bundle_manager_client.go#L244:35), [server (send part)](https://sourcegraph.com/github.com/sourcegraph/sourcegraph/-/blob/enterprise/cmd/precise-code-intel-bundle-manager/internal/server/handler.go#L114:18), [server (stitch)](https://sourcegraph.com/github.com/sourcegraph/sourcegraph/-/blob/enterprise/cmd/precise-code-intel-bundle-manager/internal/server/handler.go#L123:18)
