# Sourcegraph derivative Docker images

This directory contains Sourcegraph docker images which are derivatives of an existing Docker image, but with better defaults for our use cases. For example:

- `sourcegraph/alpine` handles setting up a `sourcegraph` user account, installing common packages.
- `sourcegraph/postgres-11.4` is `postgres-11.4` but with some Sourcegraph defaults.

If you are looking for our non-derivative Docker images, see e.g. `/cmd/.../Dockerfile` and `/enterprise/cmd/.../Dockerfile` instead.

## Building

All images in this directory are built and published automatically on CI:

- See [the handbook](https://about.sourcegraph.com/handbook/engineering/deployments) for more information
- Or see [how to build a test image](https://about.sourcegraph.com/handbook/engineering/deployments#building-docker-images-for-a-specific-branch) if you need to build a test image without merging your change to `master` first.

### Exception: `docker-images/alpine` is manually built and pushed as needed.

```sh
git checkout master
cd docker-images/alpine
IMAGE=sourcegraph/alpine:$MY_VERSION ./build.sh
VERSION=$MY_VERSION ./release.sh
```

Note: `$MY_VERSION` above should reflect the underlying Alpine version. If changes are made without altering the underlying Alpine version, then bump the suffix. For example, use 3.10-1, 3.10-2, and so on. To find the current version, consult https://hub.docker.com/r/sourcegraph/alpine

## Adding a new image

1. Create a `build.sh` and add your publishing script to it - the script should end with `docker tag ... "$IMAGE"`. See the scripts in this directory for examples.
2. Ensure your new script is executable with `chmod +x build.sh` (you can try it via e.g. `IMAGE=fake-repo/cadvisor:latest docker-images/$SERVICE/build.sh`, or by [building a test image](https://about.sourcegraph.com/handbook/engineering/deployments#building-docker-images-for-a-specific-branch))
3. Add an image to the automated builds pipeline by adding it to [`SourcegraphDockerImages`](https://sourcegraph.com/search?q=repo:%5Egithub%5C.com/sourcegraph/sourcegraph%24+file:%5Eenterprise/dev/ci/images/images%5C.go+SourcegraphDockerImages&patternType=literal).

(c) Copyright Sourcegraph 2012-2023.