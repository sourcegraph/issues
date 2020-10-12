# Sourcegraph upgrades

All you need to do to upgrade Sourcegraph is to restart your Docker server with a new image tag.

We actively maintain the two most recent monthly releases of Sourcegraph.

Upgrades should happen across consecutive minor versions of Sourcegraph. For example, if you are
running Sourcegraph 3.1 and want to upgrade to 3.3, you should upgrade to 3.2 and then 3.3.

> The Docker server image tags follow SemVer semantics, so version `3.20.1` can be found at `sourcegraph/server:3.20.1`. You can see the full list of tags on our [Docker Hub page](https://hub.docker.com/r/sourcegraph/server/tags).
