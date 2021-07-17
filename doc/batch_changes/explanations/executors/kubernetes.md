# Running Sourcegraph executors in a Kubernetes cluster

<style>
@import url(draft.css);
</style>

<div id="draft"><span>DRAFT</span></div>

In this model, changesets are computed in Kubernetes jobs, with each step being run as a separate container.

## Pros

* Deployment may be able to reuse the same Kubernetes infrastructure that you already have.
* Scaling and resource limits can be defined on the Kubernetes namespace.

## Cons

* Not as simple to get up and running as a bare server deployment.

## Security considerations

As batch changes execute user-supplied code, care must be taken to place the Kubernetes namespace in an environment without access to sensitive services. This can be done with an appropriate network policy.

<!-- aharvey: I'd like to get Dax to provide a sample at some point, although we might be able to steal https://serverfault.com/questions/933230/how-to-isolate-kubernetes-namespaces-but-allow-access-from-outside for now. -->

Note that pods created when running Kubernetes jobs are created without access to the service account. Care must be taken to ensure that they will not have access to any other sensitive elements of the cluster.

## Installation

The Sourcegraph executor is provided as a container that can be deployed into the Kubernetes cluster; it uses the service account to create and remove jobs in Kubernetes.

For example, this manifest would start a deployment with one executor:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: sourcegraph-executor-deployment
  labels:
    app: sourcegraph-executor
spec:
  replicas: 1
  selector:
    matchLabels:
      app: sourcegraph-executor
  template:
    metadata:
      labels:
        app: sourcegraph-executor
    spec:
      containers:
      - name: sourcegraph-executor
        image: sourcegraph/executor
        env:
          - name: EXECUTOR_QUEUE_NAME
            value: "batches"
          - name: EXECUTOR_FRONTEND_URL
            value: "https://sourcegraph.at.my.company/"
          - name: EXECUTOR_FRONTEND_USERNAME
            value: "executor"
          - name: EXECUTOR_FRONTEND_PASSWORD
            value: "a-highly-secure-password"
          - name: EXECUTOR_BACKEND
            value: "kubernetes"
          - name: EXECUTOR_MAX_NUM_JOBS
            value: "4"
          - name: EXECUTOR_KUBE_NAMESPACE
            value: "sourcegraph-executor"
```

The environment variables required are documented on the [configuration page](configuration.md).

## Scaling

The `EXECUTOR_MAX_NUM_JOBS` environment variable controls how many concurrent jobs will be started from the executor container configured above. A single executor can manage many jobs at once; values of up to about 100 should work well.

Since the `sourcegraph/executor` container is stateless, you can also run more than one replica in the deployment to increase the amount of concurrency.
