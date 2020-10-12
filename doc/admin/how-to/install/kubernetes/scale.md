# Scaling

Sourcegraph can be configured to scale to very large codebases and large numbers of
users. If you notice latency for search or code intelligence is higher than desired, changing these
parameters can yield a drastic improvement in performance.

> For assistance when scaling and tuning Sourcegraph, [contact us](https://about.sourcegraph.com/contact/). We're happy to help!

## Tuning replica counts for horizontal scalability

By default, your cluster has a single pod for each of `sourcegraph-frontend`, `searcher`, and `gitserver`. You can
increase the number of replicas of each of these services to handle higher scale.

We recommend setting the `sourcegraph-frontend`, `searcher`, and `gitserver` replica counts according to the following tables:

| Users      | Number of `sourcegraph-frontend` replicas |
| ---------- | ----------------------------------------- |
| 10-500     | 1                                         |
| 500-2000   | 2                                         |
| 2000-4000  | 6                                         |
| 4000-10000 | 18                                        |
| 10000+     | 28                                        |

_You can change the replica count of `sourcegraph-frontend` by editing [base/frontend/sourcegraph-frontend.Deployment.yaml](https://github.com/sourcegraph/deploy-sourcegraph/blob/master/base/frontend/sourcegraph-frontend.Deployment.yaml)._

| Repositories | Number of `searcher` replicas                                                  |
| ------------ | ------------------------------------------------------------------------------ |
| 1-20         | 1                                                                              |
| 20-50        | 2                                                                              |
| 50-200       | 3-5                                                                            |
| 200-1k       | 5-10                                                                           |
| 1k-5k        | 10-15                                                                          |
| 5k-25k       | 20-40                                                                          |
| 25k+         | 40+ ([contact us](https://about.sourcegraph.com/contact/) for scaling advice)  |
| Monorepo     | 1-25 ([contact us](https://about.sourcegraph.com/contact/) for scaling advice) |

_You can change the replica count of `searcher` by editing [base/searcher/searcher.Deployment.yaml](https://github.com/sourcegraph/deploy-sourcegraph/blob/master/base/searcher/searcher.Deployment.yaml)._

| Repositories | Number of `gitserver` replicas                                                |
| ------------ | ----------------------------------------------------------------------------- |
| 1-200        | 1                                                                             |
| 200-500      | 2                                                                             |
| 500-1000     | 3                                                                             |
| 1k-5k        | 4-8                                                                           |
| 5k-25k       | 8-20                                                                          |
| 25k+         | 20+ ([contact us](https://about.sourcegraph.com/contact/) for scaling advice) |
| Monorepo     | 1 ([contact us](https://about.sourcegraph.com/contact/) for scaling advice)   |

_Read [configure.md](configure.md#Configure-gitserver-replica-count) to learn about how to change
the replica count of `gitserver`._

---

## Improving performance with a large number of repositories

When you're using Sourcegraph with many repositories (100s-10,000s), the most important parameters to tune are:

- `sourcegraph-frontend` CPU/memory resource allocations
- `searcher` replica count
- `indexedSearch` CPU/memory resource allocations
- `gitserver` replica count
- `symbols` replica count and CPU/memory resource allocations
- `gitMaxConcurrentClones`, because `git clone` and `git fetch` operations are IO- and CPU-intensive
- `repoListUpdateInterval` (in minutes), because each interval triggers `git fetch` operations for all repositories

Consult the tables above for the recommended replica counts to use. **Note:** the `gitserver` replica count is specified
differently from the replica counts for other services; read [configure.md](configure.md#Configure-gitserver-replica-count) to learn about how to change
the replica count of `gitserver`.

Notes:

- If your change requires `gitserver` pods to be restarted and they are scheduled on another node
  when they restart, they may go offline for 60-90 seconds (and temporarily show a `Multi-Attach`
  error). This delay is caused by Kubernetes detaching and reattaching the volume. Mitigation
  steps depend on your cloud provider; [contact us](https://about.sourcegraph.com/contact/) for
  advice.

- For context on what each service does, see [Sourcegraph Architecture Overview](https://docs.sourcegraph.com/dev/architecture).

---

## Improving performance with large monorepos

When you're using Sourcegraph with a large monorepo (or several large monorepos), the most important parameters to tune
are:

- `sourcegraph-frontend` CPU/memory resource allocations
- `searcher` CPU/memory resource allocations (allocate enough memory to hold all non-binary files in your repositories)
- `indexedSearch` CPU/memory resource allocations (for the `zoekt-indexserver` pod, allocate enough memory to hold all non-binary files in your largest repository; for the `zoekt-webserver` pod, allocate enough memory to hold ~2.7x the size of all non-binary files in your repositories)
- `symbols` CPU/memory resource allocations
- `gitserver` CPU/memory resource allocations (allocate enough memory to hold your Git packed bare repositories)

---

## Configuring faster disk I/O for caches

Many parts of Sourcegraph's infrastructure benefit from using SSDs for caches. This is especially
important for search performance. By default, disk caches will use the
Kubernetes [hostPath](https://kubernetes.io/docs/concepts/storage/volumes/#hostpath) and will be the
same IO speed as the underlying node's disk. Even if the node's default disk is a SSD, however, it
is likely network-mounted rather than local.

See [configure/ssd/README.md](https://github.com/sourcegraph/deploy-sourcegraph/blob/master/configure/ssd/README.md) for instructions about configuring SSDs.

---

## Cluster resource allocation guidelines

For production environments, we recommend the following resource allocations for the entire
Kubernetes cluster, based on the number of users in your organization:

| Users        | vCPUs | Memory | Attached Storage | Root Storage |
| ------------ | ----- | ------ | ---------------- | ------------ |
| 10-500       | 10    | 24 GB  | 500 GB           | 50 GB        |
| 500-2,000    | 16    | 48 GB  | 500 GB           | 50 GB        |
| 2,000-4,000  | 32    | 72 GB  | 900 GB           | 50 GB        |
| 4,000-10,000 | 48    | 96 GB  | 900 GB           | 50 GB        |
| 10,000+      | 64    | 200 GB | 900 GB           | 50 GB        |

---

<a id="node-selector">

## Using heterogeneous node pools with `nodeSelector`

See ["Assign resource-hungry pods to larger nodes" in docs/configure.md](configure.md#assign-resource-hungry-pods-to-larger-nodes).
