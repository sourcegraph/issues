# Serving local repositories

The [Sourcegraph CLI (`src)`](https://github.com/sourcegraph/src-cli) provides the command `src serve-git` which will recursively serve up the git repositories it finds in the current directory. It exposes the repositories over HTTP as well as an API for Sourcegraph to query.

>NOTE: If using Perforce, see the [Perforce repositories with Sourcegraph guide](../repo/perforce.md).

The most common use-case for `src serve-git` is to create git repos that do not exist on a code host for Sourcegraph. For example:

- Splitting up a monorepo into a repository per project.
- Post-processing repositories for search. eg: extracting JAR files or committing files generated in the build process.
- Using `git p4` to serve up Perforce repositories.
- Serve up local repositories to Sourcegraph while trialling it.

## Quickstart

1. [Install Sourcegraph CLI](https://github.com/sourcegraph/src-cli#installation) (`src`).
1. Run `src serve-git` in a directory with git repositories. Ensure the address is reachable by Sourcegraph.
1. Go to **Site admin > Manage repositories > Add repositories**
1. Select **Sourcegraph CLI Serve-Git**.
1. Configure the URL field to the address for `src serve-git`.
1. Press **Add repositories**.

**IMPORTANT:** If you are running Sourcegraph in docker and are using a Linux host machine, replace `host.docker.internal` in the above with the IP address of your actual host machine because `host.docker.internal` [does not work on Linux](https://github.com/docker/for-linux/issues/264). You should use the network-accessible IP shown by `ifconfig` (rather than 127.0.0.1 or localhost).

## Docker

src-cli publishes Docker images which can be used instead of the binary. For example to publish your current directory run:

```
docker run \
  --rm=true \
  --publish 3434:3434 \
  --volume $PWD:/data/repos:ro \
  sourcegraph/src-cli:latest serve-git /data/repos
```

To confirm this is working visit http://localhost:3434

## src-expose

Before Sourcegraph 3.19 we recommend users to still use [`src-expose`](non-git.md).
