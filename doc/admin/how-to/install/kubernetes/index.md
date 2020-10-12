# Install Sourcegraph with Kubernetes

Deploying Sourcegraph into a Kubernetes cluster is for organizations that need highly scalable and
available code search and code intelligence.

The Kubernetes manifests for a Sourcegraph on Kubernetes installation are in the repository
 [deploy-sourcegraph](https://github.com/sourcegraph/deploy-sourcegraph).

## Requirements

- [Kubernetes](https://kubernetes.io/) v1.9 or later with an SSD storage class
  - [Cluster role administrator access](https://kubernetes.io/docs/reference/access-authn-authz/rbac/)
- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) v1.9.7 or later
- Access to server infrastructure on which you can create a Kubernetes cluster (see
  [resource allocation guidelines](scale.md)).
- [Sourcegraph Enterprise license](configure.md#add-license-key). You can run through these instructions without one, but you must obtain a license for instances of more than 10 users.
- A valid domain name for your Sourcegraph instance ([to enable SSL/TLS](configure.md#configure-tlsssl))
- A valid TLS certificate (whether from a trusted certificate authority such as Comodo, RapidSSL, or others, a self-signed certificate that can be distributed and installed across all users' machines, or the ability to use an existing reverse proxy that provides SSL termination for the connection)
- Access tokens or other credentials to [connect to your code hosts of choice](../../external_service/index.md)
- [Administrative access to your single sign-on (SSO) provider of choice](../../index.md)

## Steps

- [Provision a Kubernetes cluster](k8s.md) on the infrastructure of your choice.
- Make sure you have configured `kubectl` to [access your cluster](https://kubernetes.io/docs/tasks/access-application-cluster/configure-access-multiple-clusters/).

   - If you are using GCP, you'll need to give your user the ability to create roles in Kubernetes [(see GCP's documentation)](https://cloud.google.com/kubernetes-engine/docs/how-to/role-based-access-control#prerequisites_for_using_role-based_access_control):

     ```bash
     kubectl create clusterrolebinding cluster-admin-binding --clusterrole cluster-admin --user $(gcloud config get-value account)
     ```

- Clone the [deploy-sourcegraph](https://github.com/sourcegraph/deploy-sourcegraph) repository and check out the version tag you wish to deploy.

   ```bash
   # Go to https://github.com/sourcegraph/deploy-sourcegraph/tags and select the latest version tag
   git clone https://github.com/sourcegraph/deploy-sourcegraph && cd deploy-sourcegraph && git checkout ${VERSION}
   ```

- Configure the `sourcegraph` storage class for the cluster by reading through ["Configure a storage class"](./configure.md#configure-a-storage-class).

- If you want to add a large number of repositories to your instance, you should [configure the number of gitserver replicas](configure.md#configure-gitserver-replica-count) and [the number of indexed-search replicas](configure.md#configure-indexed-search-replica-count) _before_ you continue with the next step. (See ["Tuning replica counts for horizontal scalability"](scale.md#tuning-replica-counts-for-horizontal-scalability) for guidelines.)

- Deploy the desired version of Sourcegraph to your cluster:

   ```bash
   ./kubectl-apply-all.sh
   ```

- Monitor the status of the deployment.

   ```bash
   watch kubectl get pods -o wide
   ```

- Once the deployment completes, verify Sourcegraph is running by temporarily making the frontend port accessible:

   kubectl 1.9.x:

   ```bash
   kubectl port-forward $(kubectl get pod -l app=sourcegraph-frontend -o template --template="{{(index .items 0).metadata.name}}") 3080
   ```

   kubectl 1.10.0 or later:

   ```
   kubectl port-forward svc/sourcegraph-frontend 3080:30080
   ```

   Open http://localhost:3080 in your browser and you will see a setup page. Congrats, you have Sourcegraph up and running!

- Now [configure your deployment](configure.md).

### Troubleshooting

See the [Troubleshooting docs](troubleshoot.md).

### Updating

See the [Upgrading Howto](update.md) on how to upgrade.
See the [Upgrading docs](../../updates/kubernetes.md) for details on what changed in a version and if manual migration steps
are necessary.

### Cluster-admin privileges

> Note: Not all organizations have this split in admin privileges. If your organization does not then you don't need to
> change anything and can ignore this section.

The default installation has a few manifests that require cluster-admin privileges to apply. We have labelled all resources
with a label indicating if they require cluster-admin privileges or not. This allows cluster admins to install the
manifests that cannot be installed otherwise.

- Manifests deployed by cluster-admin

   ```bash
   ./kubectl-apply-all.sh -l sourcegraph-resource-requires=cluster-admin
   ```

- Manifests deployed by non-cluster-admin

   ```bash
   ./kubectl-apply-all.sh -l sourcegraph-resource-requires=no-cluster-admin
   ```

We also provide an [overlay](configure.md#non-privileged-overlay) that generates a version of the manifests that does not
require cluster-admin privileges. 
