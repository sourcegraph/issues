# Executor

The executor service polls the public frontend API for work to perform. The executor will pull a job from a particular queue (configured via the envvar `EXECUTOR_QUEUE_NAME`), then performs the job by running a sequence of docker and src-cli commands. This service is horizontally scalable.

See the executor-queue service for a complete list of queues.

(c) Copyright Sourcegraph 2012-2023.