# Installing Sourcegraph

| Deployment Type                                       | Suggested for                                       | Setup time | Multi-machine? | Auto healing? | Monitoring? |
|-------------------------------------------------------|-----------------------------------------------------|------------|----------------|---------------|-------------|
| [Single-container server](../install/docker/index.md) | Local testing                                       | 60 seconds | No             | No            | No          |
| [Docker Compose](../install/docker-compose/index.md)  | Small & medium production deployments               | 5 minutes  | Not Supported  | No            | Yes         |
| [Kubernetes](../install/kubernetes/index.md)          | Medium & large highly-available cluster deployments | 30 minutes | Yes            | Yes           | Yes         |


* If you're just starting out, we recommend [running Sourcegraph locally](docker/index.md). It takes only a few minutes and lets you try out all of the features.
* If you need scalability and high-availability beyond what a single-node [Docker Compose](https://docs.docker.com/compose/) can offer, use the [Kubernetes cluster deployment option](https://github.com/sourcegraph/deploy-sourcegraph), instead.

## Resource estimator

Use the [resource estimator](resource_estimator.md) to find a good starting point for your deployment.
