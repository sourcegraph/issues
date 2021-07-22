# PostgreSQL configuration

Sourcegraph Kubernetes cluster site admins can override the default PostgreSQL configuration by supplying their own `postgresql.conf` file contents. These are specified in [`pgsql.ConfigMap.yaml`](https://github.com/sourcegraph/deploy-sourcegraph/blob/master/base/pgsql/pgsql.ConfigMap.yaml).

There is no officially supported way of customizing the PostgreSQL configuration in the single Docker image.

## Suggested configuration

### Static configuration

We have found the following values to work well for our Cloud instance in practice. These settings are independent of the resources given to the database container and are recommended to use across the board.

| Setting                    | Default value | Suggested value |
| -------------------------- | ------------- | --------------- |
| `bgwriter_delay`           | `200ms`       | `50ms`          |
| `bgwriter_lru_maxpages`    | `100`         | `200`           |
| `effective_io_concurrency` | `1`           | `200`           |
| `max_wal_size`             | `1GB`         | `8GB`           |
| `min_wal_size`             | `80MB`        | `2GB`           |
| `random_page_cost`         | `4.0`         | `1.1`           |
| `temp_file_limit`          | `-1`          | `20GB`          |
| `wal_buffers`              | `-1`          | `16MB`          |

The suggested values for the `effective_io_concurrency` and `random_page_cost` settings assume SSD disks are in-use for the Postgres data volume (recommended). If you are instead using HDDs, these values should be set to `2` and `4` (the defaults), respectively. These values control the cost heuristic of fetching data from disk, and using the supplied configuration on spinning media will cause the query planner to fetch from disk much more aggressively than it should.

### Resource-dependent configuration

The following settings are dependent on the number of CPUs and the amount of memory given to the database container, as well as the expected number maximum connections.

| Setting                            | Default value | Suggested value | Suggested maximum |
| ---------------------------------- | ------------- | --------------- | ----------------- |
| `effective_cache_size`             | `4GB`         | `mem * 3 / 4`   | |
| `maintenance_work_mem`             | `64MB`        | `mem / 16`      | `2gb`
| `max_connections`                  | `100`         | `100` to start  | |
| `max_parallel_maintenance_workers` | `2`           | `# of CPUs`     | |
| `max_parallel_workers_per_gather`  | `2`           | `# of CPUs / 2` | `4`
| `max_parallel_workers`             | `8`           | `# of CPUs`     | |
| `max_worker_processes`             | `8`           | `# of CPUs`     | |
| `shared_buffers`                   | `32MB`        | `mem / 4`       | |
| `work_mem`                         | `4MB`         | `mem / (4 * max_connections * max_parallel_workers_per_gather)` | |

The setting `effective_cache_size` acts as a hint to Postgres on how to adjust its own I/O cache and does not require the configured amount of memory to be used. This value should reflect the amount of memory available to Postgres. This should be the amount of memory given to the container minus some slack for memory used by the kernel, I/O devices, and other daemons running in the same container.

The setting `max_connections` determines the number of active connections that can exist before new connections will start to be declined. This number is dependent on the replica factor of the containers that require a database connection. These containers include:

- frontend
- gitserver
- repo-updater
- precise-code-intel-worker

Each of these containers open a pool of connections not exceeding the pool capacity indicated by the `SRC_PGSQL_MAX_OPEN` environment variable. The maximum number of connections for your instance can be determined by summing the connection pool capacity of every container in this list. By default, `SRC_PGSQL_MAX_OPEN` is `30`.

The setting `max_parallel_workers_per_gather` controls how many _additional_ workers to launch for operations such as parallel sequential scan. We see diminishing returns around four workers per query. Also notice that increasing this value will *multiplicatively* increase the amount of memory required for each worker to operate safely; doubling this
value will effectively half the maximum number of connections. Most workloads should be perfectly fine with only two workers per query.

The settings `shared_buffers` and `work_mem` control how much memory is allocated to different parts of Postgres. The size of the shared buffers, which we recommend to set to 25% of the container's total memory, determines the size of the disk page cache that is usable by every worker (and, therefore, every query). The remaining free memory is allocated to workers such that the maximum number of concurrently executing workers will not exceed the remaining 75% (minus some proportional buffer) of the container's total memory.

A `work_mem` setting of `32MB` has been sufficient for our Cloud environment as well as high-usage enterprise instances. Smaller instances and memory-constrained deployments may get away with a smaller value, but this is highly dependent on the set of features in use and their exact usage.

If you are seeing the database instance restarting due to a backend OOM condition or any Postgres logs similar to the following, it is likely that your `work_mem` setting is too low for your instance's query patterns. It's advised to raise the memory on the database container and re-adjust the settings above. If you cannot easily raise memory, you can alternatively lower `max_connections` or `max_parallel_workers_per_gather` to buy a bit of headroom with your current resources.

```
2021-04-26 10:11:12.123 UTC [33330] ERROR:  could not read block 1234 in file "base/123456789/123456789": Cannot allocate memory
2021-04-26 10:11:12.123 UTC [33330] ERROR:  could not read block 1234 in file "base/123456789/123456789": read only 1234 of 1234 bytes
```
