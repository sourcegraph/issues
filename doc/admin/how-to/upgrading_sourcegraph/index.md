# Upgrading Sourcegraph

## For single-node deployments (`sourcegraph/server`)

A new version of Sourcegraph is released every month (with patch releases in between, released as needed). Check the [Sourcegraph blog](https://about.sourcegraph.com/blog) or the site admin updates page to learn about updates.

To update, just use the newer `sourcegraph/server:N.N.N` Docker image (where `N.N.N` is the version number) in place of the older one, using the same Docker volumes. Your server's data will be migrated automatically if needed.

You can always find the version number of the latest release at [docs.sourcegraph.com](https://docs.sourcegraph.com) in the `docker run` command's image tag.

- As a precaution, before updating, we recommend backing up the contents of the Docker volumes used by Sourcegraph.
- If you need a HA deployment, use the [Kubernetes cluster deployment option](https://github.com/sourcegraph/deploy-sourcegraph).
- There is currently no automated way to downgrade to an older version after you have updated. [Contact support](https://about.sourcegraph.com/contact) for help.

## For Kubernetes cluster deployments

See "[Updating Sourcegraph](install/kubernetes/update.md)" in the Kubernetes cluster administrator guide.

## For Docker Compose deployments

Please see: [updating a Docker Compose Sourcegraph instance](updates/docker_compose.md)

## For pure-Docker cluster deployments

Please see: [updating a pure-Docker Sourcegraph cluster](updates/pure_docker.md)
