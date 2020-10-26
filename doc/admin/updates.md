# Upgrading Sourcegraph

## Updating to a new version of Sourcegraph

Please see the instructions for your deployment type:

- [Single-container `sourcegraph/server` deployments](updates/server.md)
- [Docker Compose single-machine deployments](updates/docker_compose.md)
- [Kubernetes cluster deployments](updates/kubernetes.md)
- [pure-Docker custom deployments](updates/pure_docker.md)

## Migrating to a new deployment type

See [this page](install.md) to get advice on which deployment type you should be running.

- [Migrate to Docker Compose](install/docker-compose/migrate.md) for improved stability and performance if you are using a single-container `sourcegraph/server` deployment.
- [Migrate to a Kubernetes cluster](https://docs.sourcegraph.com/admin/install/kubernetes) if you exceed the limits of a single machine Docker Compose deployment.
