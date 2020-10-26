# sourcegraph/ignite-ubuntu

We produce a version of Ubuntu 20.04 (Focal Fossa) based on [weaveworks/ignite-ubuntu:20.04-amd64](https://github.com/weaveworks/ignite/blob/46bdd5d48425c4245fbe895e7da3621f491c3660/images/ubuntu/Dockerfile) that also contains the Docker distribution.

This image serves as the base image for [Firecracker](https://github.com/firecracker-microvm/firecracker) virtual machines in which we run user configured containers and code.
