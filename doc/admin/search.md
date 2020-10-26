# Search configuration

See "[Code search overview](../code_search/index.md)" for general information about Sourcegraph's code search.

## Indexed search

Sourcegraph can index the code on the default branch of each repository. This speeds up searches that hit many repositories at once.

For large deployments we recommend horizontally scaling indexed search. You can do this by [adjusting the number of replicas](https://github.com/sourcegraph/deploy-sourcegraph/blob/master/docs/configure.md#configure-indexed-search-replica-count). Sourcegraph shards repository indexes across replicas. When the replica count changes Sourcegraph will slowly rebalance indexes to ensure availability of existing indexes.

Indexed search increases the memory and storage requirements for Sourcegraph. The resource requirements vary considerably based on the text contents of your repositories, but a good estimate is that the node should have enough memory to hold the entire text contents of the default branch of each repository. To disable indexed search when running Sourcegraph on a single node, set the `search.index.enabled` [site configuration](config/site_config.md) property to `false`.
