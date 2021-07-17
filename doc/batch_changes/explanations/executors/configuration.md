# Configuration

<style>
@import url(draft.css);
</style>

<div id="draft"><span>DRAFT</span></div>

Sourcegraph executors use environment variables to configure their credentials and behaviour. The required and optional variables are listed below:

| Variable | Required | Default value | Description |
|----------|----------|---------------|-------------|
| `EXECUTOR_QUEUE_NAME` | **Yes** | **No default; must be provided** | The name of the queue to listen to; `batches` for Batch Changes. |
| `EXECUTOR_FRONTEND_URL` |  **Yes** | **No default; must be provided** | The Sourcegraph URL; eg `https://sourcegraph.com`. |
| `EXECUTOR_FRONTEND_USERNAME` |  **Yes** | **No default; must be provided** | The executor username, as provided in the [Sourcegraph site settings](TODO). |
| `EXECUTOR_FRONTEND_PASSWORD` |  **Yes** | **No default; must be provided** | The executor password, as provided in the [Sourcegraph site settings](TODO). |
| `EXECUTOR_BACKEND` | No | `docker` | The backend that should be used to execute tasks; either `docker` or `kubernetes`. |
| `EXECUTOR_MAX_NUM_JOBS` | No | `1` | The maximum number of jobs (or tasks) that will be run concurrently by this executor. |
| `EXECUTOR_KUBE_NAMESPACE` | Yes for k8s | **No default** | If `EXECUTOR_BACKEND` is set to `kubernetes`, this configures the Kubernetes namespace that jobs will be created in. |

<!-- aharvey: There are lots of other options at https://sourcegraph.com/github.com/sourcegraph/sourcegraph@3b1fbde4e2207de103a6736706bbfd0adaa579b6/-/blob/enterprise/cmd/executor/config.go#L36-52; most aren't super relevant for user facing documentation and are omitted for brevity, although the final version of the docs will need them. -->
