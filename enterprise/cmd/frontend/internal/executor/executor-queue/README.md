# Executor queue

The executor-queue service maintains the executor work queues. Executor instances poll for and perform jobs from a particular queue. The executor-queue service locks in-progress jobs via a Postgres transaction. Executors and the executor-queue will periodically reconcile in-progress jobs via heartbeat requests. This is a singleton service that is accessible via a frontend proxy authenticated via a shared token.

## Work queues

- The `codeintel` queue contains unprocessed lsif_index records
- The `batches` queue contains unprocessed batch_spec_execution records
