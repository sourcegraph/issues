# Configuring Sourcegraph

Configuring a Sourcegraph Kubernetes cluster is done by applying manifest files and with simple
`kubectl` commands. You can configure Sourcegraph as flexibly as you need to meet the requirements
of your deployment environment.  We provide simple instructions for common things like setting up
TLS, enabling code intelligence, and exposing Sourcegraph to external traffic below.

## Fork this repository

We **strongly** recommend you fork the [deploy-sourcegraph](https://github.com/sourcegraph/deploy-sourcegraph) repository to track your configuration changes in Git.
**This will make upgrades far easier** and is a good practice not just for Sourcegraph, but for any Kubernetes application.

- Create a fork of the [deploy-sourcegraph](https://github.com/sourcegraph/deploy-sourcegraph) repository.

> WARNING: Set it to **private** if you plan to store secrets (SSL certificates, external Postgres credentials, etc.) within the repository.

> NOTE: We do not recommend storing secrets in the repository itself and these instructions document how.

- Create a `release` branch to track all of your customizations to Sourcegraph.
> NOTE: When you upgrade Sourcegraph, you will merge upstream into this branch.

```bash
export SOURCEGRAPH_VERSION="v3.26.3"
git checkout $SOURCEGRAPH_VERSION -b release
```

If you followed the installation instructions, `$SOURCEGRAPH_VERSION` should point at the Git tag you've deployed to your running Kubernetes cluster.

### Commit customizations to your release branch:

- Commit manual modifications to Kubernetes YAML files.
> WARNING: Modifications to files inside the `base` increases the odds of encountering git merge conflicts when upgrading. Consider using [overlays](overlays.md) instead.

- Commit commands that should be run on every update (e.g. `kubectl apply`) to [kubectl-apply-all.sh](https://github.com/sourcegraph/deploy-sourcegraph/blob/master/kubectl-apply-all.sh).

- Commit commands that generally only need to be run once per cluster to (e.g. `kubectl create secret`, `kubectl expose`) to [create-new-cluster.sh](https://github.com/sourcegraph/deploy-sourcegraph/blob/master/create-new-cluster.sh).

## Upgrading with a forked repository

When you upgrade, merge the corresponding `upstream release` tag into your `release` branch _(created from the [Fork this repository](#fork-this-repository) step)_.

```bash
# to add the upstream remote.
git remote add upstream https://github.com/sourcegraph/deploy-sourcegraph
# to merge the upstream release tag into your release branch.
git checkout release && git merge v3.26.3
```

_See also [git strategies when using overlays](overlays.md#git-strategies-with-overlays)_



## Dependencies

Configuration steps in this file depend on [jq](https://stedolan.github.io/jq/),
[yj](https://github.com/sourcegraph/yj) and [jy](https://github.com/sourcegraph/jy).

Install the [kustomize](https://kustomize.io/) tool if you choose to use [overlays](overlays.md).



## Security - Configure network access

You need to make the main web server accessible over the network to external users.

There are a few approaches, but using an ingress controller is recommended.

### Ingress controller (recommended)

For production environments, we recommend using the [ingress-nginx](https://kubernetes.github.io/ingress-nginx/) [ingress](https://kubernetes.io/docs/concepts/services-networking/ingress/).

- As part of our base configuration, we install an ingress for [sourcegraph-frontend](https://github.com/sourcegraph/deploy-sourcegraph/blob/master/base/frontend/sourcegraph-frontend.Ingress.yaml). It installs rules for the default ingress, see comments to restrict it to a specific host.

- In addition to the sourcegraph-frontend ingress, you'll need to install the NGINX ingress controller (ingress-nginx).

- Follow the instructions at https://kubernetes.github.io/ingress-nginx/deploy/ to create the ingress controller.

- Add the files to [configure/ingress-nginx](https://github.com/sourcegraph/deploy-sourcegraph/blob/master/configure/ingress-nginx), including an [install.sh](https://github.com/sourcegraph/deploy-sourcegraph/blob/master/configure/ingress-nginx/install.sh) file which applies the relevant manifests.

- We include sample generic-cloud manifests as part of this repository, but please follow the official instructions for your cloud provider.

- Add the [configure/ingress-nginx/install.sh](https://github.com/sourcegraph/deploy-sourcegraph/blob/master/configure/ingress-nginx/install.sh) command to [create-new-cluster.sh](https://github.com/sourcegraph/deploy-sourcegraph/blob/master/create-new-cluster.sh) and commit the change:

```shell
echo ./configure/ingress-nginx/install.sh >> create-new-cluster.sh
```

- Once the ingress has acquired an external address, you should be able to access Sourcegraph using that.

- You can check the external address by running the following command and looking for the `LoadBalancer` entry:

```bash
kubectl -n ingress-nginx get svc
```

If you are having trouble accessing Sourcegraph, ensure ingress-nginx IP is accessible above. Otherwise see [Troubleshooting ingress-nginx](https://kubernetes.github.io/ingress-nginx/troubleshooting/). The namespace of the ingress-controller is `ingress-nginx`.

#### Configuration

`ingress-nginx` has extensive configuration documented at [NGINX Configuration](https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/). We expect most administrators to modify [ingress-nginx annotations](https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/annotations/) in [sourcegraph-frontend.Ingress.yaml](https://github.com/sourcegraph/deploy-sourcegraph/blob/master/base/frontend/sourcegraph-frontend.Ingress.yaml). Some settings are modified globally (such as HSTS). In that case we expect administrators to modify the [ingress-nginx configmap](https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/configmap/) in [configure/ingress-nginx/mandatory.yaml](https://github.com/sourcegraph/deploy-sourcegraph/blob/master/configure/ingress-nginx/mandatory.yaml).

### NGINX service

In cases where ingress controllers cannot be created, creating an explicit NGINX service is a viable
alternative. See the files in the [configure/nginx-svc](https://github.com/sourcegraph/deploy-sourcegraph/blob/master/configure/nginx-svc) folder for an
example of how to do this via a NodePort service (any other type of Kubernetes service will also
work):

- Modify [configure/nginx-svc/nginx.ConfigMap.yaml](https://github.com/sourcegraph/deploy-sourcegraph/blob/master/configure/nginx-svc/nginx.ConfigMap.yaml) to
   contain the TLS certificate and key for your domain.

- `kubectl apply -f configure/nginx-svc` to create the NGINX service.

- Update [create-new-cluster.sh](https://github.com/sourcegraph/deploy-sourcegraph/blob/master/create-new-cluster.sh) with the previous command.

   ```
   echo kubectl apply -f configure/nginx-svc >> create-new-cluster.sh
   ```

### Network rule

> NOTE: this setup path does not support TLS.

Add a network rule that allows ingress traffic to port 30080 (HTTP) on at least one node.

#### [Google Cloud Platform Firewall rules](https://cloud.google.com/compute/docs/vpc/using-firewalls).

- Expose the necessary ports.

```bash
gcloud compute --project=$PROJECT firewall-rules create sourcegraph-frontend-http --direction=INGRESS --priority=1000 --network=default --action=ALLOW --rules=tcp:30080
```

- Change the type of the `sourcegraph-frontend` service in [base/frontend/sourcegraph-frontend.Service.yaml](https://github.com/sourcegraph/deploy-sourcegraph/blob/master/base/frontend/sourcegraph-frontend.Service.yaml) from `ClusterIP` to `NodePort`:

```diff
spec:
  ports:
  - name: http
    port: 30080
+    nodePort: 30080
-  type: ClusterIP
+  type: NodePort
```

- Directly applying this change to the service [will fail](https://github.com/kubernetes/kubernetes/issues/42282). Instead, you must delete the old service and then create the new one (this will result in a few seconds of downtime):

```shell
kubectl delete svc sourcegraph-frontend
kubectl apply -f base/frontend/sourcegraph-frontend.Service.yaml
```

- Find a node name.

```bash
kubectl get pods -l app=sourcegraph-frontend -o=custom-columns=NODE:.spec.nodeName
```

- Get the EXTERNAL-IP address (will be ephemeral unless you [make it static](https://cloud.google.com/compute/docs/ip-addresses/reserve-static-external-ip-address#promote_ephemeral_ip)).

```bash
kubectl get node $NODE -o wide
```

#### [AWS Security Group rules](http://docs.aws.amazon.com/AmazonVPC/latest/UserGuide/VPC_SecurityGroups.html).

Sourcegraph should now be accessible at `$EXTERNAL_ADDR:30080`, where `$EXTERNAL_ADDR` is the address of _any_ node in the cluster.

### Using NetworkPolicy

Network policy is a Kubernetes resource that defines how pods are allowed to communicate with each other and with
other network endpoints. If the cluster administration requires an associated NetworkPolicy when doing an installation,
then we recommend running Sourcegraph in a namespace (as described in our [Overlays docs](overlays.md) or below in the
[Using NetworkPolicy with Namespaced Overlay Example](#using-networkpolicy-with-namespaced-overlay)).
You can then use the `namespaceSelector` to allow traffic between the Sourcegraph pods.
When you create the namespace you need to give it a label so it can be used in a `matchLabels` clause.

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: ns-sourcegraph
  labels:
    name: ns-sourcegraph
```

If the namespace already exists you can still label it like so

```shell script
kubectl label namespace ns-sourcegraph name=ns-sourcegraph
```

> NOTE: You will need to augment this example NetworkPolicy to allow traffic to external services
> you plan to use (like github.com) and ingress traffic from
> the outside to the frontend for the users of the Sourcegraph installation.
> Check out this [collection](https://github.com/ahmetb/kubernetes-network-policy-recipes) of NetworkPolicies to get started.

```yaml
kind: NetworkPolicy
apiVersion: networking.k8s.io/v1
metadata:
  name: np-sourcegraph
  namespace: ns-sourcegraph
spec:
  # For all pods with the label "deploy: sourcegraph"
  podSelector:
    matchLabels:
      deploy: sourcegraph
  policyTypes:
  - Ingress
  - Egress
  # Allow all traffic inside the ns-sourcegraph namespace
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: ns-sourcegraph
  egress:
  - to:
    - namespaceSelector:
        matchLabels:
          name: ns-sourcegraph
```

### Using NetworkPolicy with Namespaced Overlay Example

1. Create a yaml file (`networkPolicy.yaml` for example) in the root directory with the added namespace after
applying the [Namespaced Overlay](overlays.md#namespaced-overlay):

   ```yaml
   kind: NetworkPolicy
   apiVersion: networking.k8s.io/v1
   metadata:
     name: np-sourcegraph
     namespace: ns-<EXAMPLE NAMESPACE>
   spec:
     # For all pods with the label "deploy: sourcegraph"
     podSelector:
       matchLabels:
         deploy: sourcegraph
     policyTypes:
     - Ingress
     - Egress
     # Allow all traffic inside the ns-<EXAMPLE NAMESPACE> namespace
     ingress:
     - from:
       - namespaceSelector:
           matchLabels:
             name: ns-sourcegraph
             namespace: ns-<EXAMPLE NAMESPACE>
     egress:
     - to:
       - namespaceSelector:
           matchLabels:
             name: ns-<EXAMPLE NAMESPACE>
   ```

1. Run `kubectl apply -f networkPolicy.yaml` to apply changes from the `networkPolicy.yaml` file

1. Run `kubectl apply -f generated-cluster/networking.k8s.io_v1beta1_ingress_sourcegraph-frontend.yaml`  to apply changes from the `networkPolicy.yaml` file

1. Apply setting to all using `kubectl apply --prune -l deploy=sourcegraph -f generated-cluster --recursive`

1. Run `kubectl get pods -A` to check for the namespaces and their status --it should now be up and running

1. Access Sourcegraph on your local machine by temporarily making the frontend port accessible:

   ```
   kubectl port-forward svc/sourcegraph-frontend 3080:30080
   ```

1. Open http://localhost:3080 in your browser and you will see a setup page.

1. 🎉 Congrats, you have Sourcegraph up and running! Now [configure your deployment](configure.md).



## Update site configuration

Sourcegraph's application configuration is stored in the PostgreSQL database. For editing this configuration you may use the web UI. See [site configuration](../../config/site_config.md) for more information.



## Configure TLS/SSL

If you intend to make your Sourcegraph instance accessible on the Internet or another untrusted network, you should use TLS so that all traffic will be served over HTTPS.

### Ingress controller

If you exposed your Sourcegraph instance via an ingress controller as described in ["Ingress controller (recommended)"](#ingress-controller-recommended):

- Create a [TLS secret](https://kubernetes.io/docs/concepts/configuration/secret/) that contains your TLS certificate and private key.

   ```
   kubectl create secret tls sourcegraph-tls --key $PATH_TO_KEY --cert $PATH_TO_CERT
   ```

- Update [create-new-cluster.sh](https://github.com/sourcegraph/deploy-sourcegraph/blob/master/create-new-cluster.sh) with the previous command.

   ```
   echo kubectl create secret tls sourcegraph-tls --key $PATH_TO_KEY --cert $PATH_TO_CERT >> create-new-cluster.sh
   ```

- Add the tls configuration to [base/frontend/sourcegraph-frontend.Ingress.yaml](https://github.com/sourcegraph/deploy-sourcegraph/blob/master/base/frontend/sourcegraph-frontend.Ingress.yaml).

   ```yaml
   # base/frontend/sourcegraph-frontend.Ingress.yaml
   tls:
     - hosts:
         #  Replace 'sourcegraph.example.com' with the real domain that you want to use for your Sourcegraph instance.
         - sourcegraph.example.com
       secretName: sourcegraph-tls
   rules:
     - http:
         paths:
         - path: /
           backend:
             serviceName: sourcegraph-frontend
             servicePort: 30080
       # Replace 'sourcegraph.example.com' with the real domain that you want to use for your Sourcegraph instance.
       host: sourcegraph.example.com
   ```

- Change your `externalURL` in [the site configuration](https://docs.sourcegraph.com/admin/config/site_config) to e.g. `https://sourcegraph.example.com`.

- Update the ingress controller with the previous changes with the following command:

  ```bash
  kubectl apply -f base/frontend/sourcegraph-frontend.Ingress.yaml
  ```

> WARNING: Do NOT commit the actual TLS cert and key files to your fork (unless your fork is private **and** you are okay with storing secrets in it).

### NGINX service

If you exposed your Sourcegraph instance via the altenative nginx service as described in ["nginx service"](#nginx-service), those instructions already walked you through setting up TLS/SSL.



## Configure repository cloning via SSH

Sourcegraph will clone repositories using SSH credentials if they are mounted at `/home/sourcegraph/.ssh` in the `gitserver` deployment.

[Create a secret](https://kubernetes.io/docs/concepts/configuration/secret/#using-secrets-as-environment-variables) that contains the base64 encoded contents of your SSH private key (_make sure it doesn't require a password_) and known_hosts file.

   ```bash
   kubectl create secret generic gitserver-ssh \
    --from-file id_rsa=${HOME}/.ssh/id_rsa \
    --from-file known_hosts=${HOME}/.ssh/known_hosts
   ```

Update [create-new-cluster.sh](https://github.com/sourcegraph/deploy-sourcegraph/blob/master/create-new-cluster.sh) with the previous command.

   ```bash
   echo kubectl create secret generic gitserver-ssh \
    --from-file id_rsa=${HOME}/.ssh/id_rsa \
    --from-file known_hosts=${HOME}/.ssh/known_hosts >> create-new-cluster.sh
   ```

Mount the [secret as a volume](https://kubernetes.io/docs/concepts/configuration/secret/#using-secrets-as-files-from-a-pod) in [gitserver.StatefulSet.yaml](https://github.com/sourcegraph/deploy-sourcegraph/blob/master/base/gitserver/gitserver.StatefulSet.yaml).

   For example:

   ```yaml
   # base/gitserver/gitserver.StatefulSet.yaml
   spec:
     containers:
       volumeMounts:
         - mountPath: /root/.ssh
           name: ssh
     volumes:
       - name: ssh
         secret:
           defaultMode: 0644
           secretName: gitserver-ssh
   ```

   Convenience script:

   ```bash
   # This script requires https://github.com/sourcegraph/jy and https://github.com/sourcegraph/yj
   GS=base/gitserver/gitserver.StatefulSet.yaml
   cat $GS | yj | jq '.spec.template.spec.containers[].volumeMounts += [{mountPath: "/root/.ssh", name: "ssh"}]' | jy -o $GS
   cat $GS | yj | jq '.spec.template.spec.volumes += [{name: "ssh", secret: {defaultMode: 384, secretName:"gitserver-ssh"}}]' | jy -o $GS
   ```

   If you run your installation with non-root users (the non-root overlay) then use the mount path `/home/sourcegraph/.ssh` instead of `/root/.ssh`:

   ```yaml
   # base/gitserver/gitserver.StatefulSet.yaml
   spec:
     containers:
       volumeMounts:
         - mountPath: /home/sourcegraph/.ssh
           name: ssh
     volumes:
       - name: ssh
         secret:
           defaultMode: 0644
           secretName: gitserver-ssh
   ```

   Convenience script:

   ```bash
   # This script requires https://github.com/sourcegraph/jy and https://github.com/sourcegraph/yj
   GS=base/gitserver/gitserver.StatefulSet.yaml
   cat $GS | yj | jq '.spec.template.spec.containers[].volumeMounts += [{mountPath: "/home/sourcegraph/.ssh", name: "ssh"}]' | jy -o $GS
   cat $GS | yj | jq '.spec.template.spec.volumes += [{name: "ssh", secret: {defaultMode: 384, secretName:"gitserver-ssh"}}]' | jy -o $GS
   ```


3. Apply the updated `gitserver` configuration to your cluster.

  ```bash
  ./kubectl-apply-all.sh
  ```

**WARNING:** Do NOT commit the actual `id_rsa` and `known_hosts` files to your fork (unless
your fork is private **and** you are okay with storing secrets in it).



## Configure gitserver replica count

Increasing the number of `gitserver` replicas can improve performance when your instance contains a large number of repositories. Repository clones are consistently striped across all `gitserver` replicas. Other services need to be aware of how many `gitserver` replicas exist so they can resolve an individual repo.

To change the number of `gitserver` replicas:

- Update the `replicas` field in [gitserver.StatefulSet.yaml](https://github.com/sourcegraph/deploy-sourcegraph/blob/master/base/gitserver/gitserver.StatefulSet.yaml).
- Update the `SRC_GIT_SERVERS` environment variable in the [sourcegraph-frontend.Deployment.yaml](https://github.com/sourcegraph/deploy-sourcegraph/blob/master/base/frontend/sourcegraph-frontend.Deployment.yaml) to reflect the number of replicas.

   For example, if there are 2 gitservers then `SRC_GIT_SERVERS` should have the value `gitserver-0.gitserver:3178 gitserver-1.gitserver:3178`:

   ```yaml
   - env:
       - name: SRC_GIT_SERVERS
         value: gitserver-0.gitserver:3178 gitserver-1.gitserver:3178
   ```

- Recommended: Increase [indexed-search replica count](#configure-indexed-search-replica-count)

Here is a convenience script that performs all three steps:

```bash
# This script requires https://github.com/sourcegraph/jy and https://github.com/sourcegraph/yj

GS=base/gitserver/gitserver.StatefulSet.yaml

REPLICA_COUNT=2 # number of gitserver replicas

# Update gitserver replica count
cat $GS | yj | jq ".spec.replicas = $REPLICA_COUNT" | jy -o $GS

# Compute all gitserver names
GITSERVERS=$(for i in `seq 0 $(($REPLICA_COUNT-1))`; do echo -n "gitserver-$i.gitserver:3178 "; done)

# Update SRC_GIT_SERVERS environment variable in other services
find . -name "*yaml" -exec sed -i.sedibak -e "s/value: gitserver-0.gitserver:3178.*/value: $GITSERVERS/g" {} +

IDX_SEARCH=base/indexed-search/indexed-search.StatefulSet.yaml

# Update indexed-search replica count
cat $IDX_SEARCH | yj | jq ".spec.replicas = $REPLICA_COUNT" | jy -o $IDX_SEARCH

# Delete sed's backup files
find . -name "*.sedibak" -delete
```

Commit the outstanding changes.



## Configure indexed-search replica count

Increasing the number of `indexed-search` replicas can improve performance and reliability when your instance contains a large number of repositories. Repository indexes are distributed evenly across all `indexed-search` replicas.

By default `indexed-search` relies on kubernetes service discovery, so adjusting the number of replicas just requires updating the `replicas` field in [indexed-search.StatefulSet.yaml](https://github.com/sourcegraph/deploy-sourcegraph/blob/master/base/indexed-search/indexed-search.StatefulSet.yaml).

Not Recommended: To use a static list of indexed-search servers you can configure `INDEXED_SEARCH_SERVERS` on `sourcegraph-frontend`. It uses the same format as `SRC_GIT_SERVERS` above. Adjusting replica counts will require the same steps as gitserver.



## Assign resource-hungry pods to larger nodes

If you have a heterogeneous cluster where you need to ensure certain more resource-hungry pods are assigned to more powerful nodes (e.g. `indexedSearch`), you can [specify node constraints](https://kubernetes.io/docs/concepts/configuration/assign-pod-node) (such as `nodeSelector`, etc.).

This is useful if, for example, you have a very large monorepo that performs best when `gitserver`
and `searcher` are on very large nodes, but you want to use smaller nodes for
`sourcegraph-frontend`, `repo-updater`, etc. Node constraints can also be useful to ensure fast
updates by ensuring certain pods are assigned to specific nodes, preventing the need for manual pod
shuffling.

See [the official documentation](https://kubernetes.io/docs/concepts/configuration/assign-pod-node/) for instructions about applying node constraints.

## Configure a storage class

Sourcegraph expects there to be storage class named `sourcegraph` that it uses for all its persistent volume claims. This storage class must be configured before applying the base configuration to your cluster.

- Create `base/sourcegraph.StorageClass.yaml` with the appropriate configuration for your cloud provider and commit the file to your fork.

- The sourcegraph StorageClass will retain any persistent volumes created in the event of an accidental deletion of a persistent volume claim.

- The sourcegraph StorageClass also allows the persistent volumes to expand their storage capacity by increasing the
 size of the related persistent volume claim.

- This cannot be changed once the storage class has been created. Persistent volumes not created with the reclaimPolicy set to `Retain` can be patched with the following command:

```bash
kubectl patch pv <your-pv-name> -p '{"spec":{"persistentVolumeReclaimPolicy":"Retain"}}'
```

See [the official documentation](https://kubernetes.io/docs/tasks/administer-cluster/change-pv-reclaim-policy/#changing-the-reclaim-policy-of-a-persistentvolume) for more information about patching persistent volumes.

### Google Cloud Platform (GCP)

```yaml
# base/sourcegraph.StorageClass.yaml
kind: StorageClass
apiVersion: storage.k8s.io/v1
metadata:
  name: sourcegraph
  labels:
    deploy: sourcegraph
provisioner: kubernetes.io/gce-pd
parameters:
  type: pd-ssd # This configures SSDs (recommended).
reclaimPolicy: Retain
allowVolumeExpansion: true
```

[Additional documentation](https://kubernetes.io/docs/concepts/storage/storage-classes/#gce-pd).

### Amazon Web Services (AWS)

```yaml
# base/sourcegraph.StorageClass.yaml
kind: StorageClass
apiVersion: storage.k8s.io/v1
metadata:
  name: sourcegraph
  labels:
    deploy: sourcegraph
provisioner: kubernetes.io/aws-ebs
parameters:
  type: gp2 # This configures SSDs (recommended).
reclaimPolicy: Retain
allowVolumeExpansion: true
```

[Additional documentation](https://kubernetes.io/docs/concepts/storage/storage-classes/#aws-ebs).

### Azure

```yaml
# base/sourcegraph.StorageClass.yaml
kind: StorageClass
apiVersion: storage.k8s.io/v1
metadata:
  name: sourcegraph
  labels:
    deploy: sourcegraph
provisioner: kubernetes.io/azure-disk
parameters:
  storageaccounttype: Premium_LRS # This configures SSDs (recommended). A Premium VM is required.
reclaimPolicy: Retain
allowVolumeExpansion: true
```

[Additional documentation](https://kubernetes.io/docs/concepts/storage/storage-classes/#azure-disk).

### Other cloud providers

```yaml
# base/sourcegraph.StorageClass.yaml
kind: StorageClass
apiVersion: storage.k8s.io/v1
metadata:
  name: sourcegraph
  labels:
    deploy: sourcegraph
reclaimPolicy: Retain
allowVolumeExpansion: true
# Read https://kubernetes.io/docs/concepts/storage/storage-classes/ to configure the "provisioner" and "parameters" fields for your cloud provider.
# SSDs are highly recommended!
# provisioner:
# parameters:
```

### Using a storage class with an alternate name

If you wish to use a different storage class for Sourcegraph, then you need to update all persistent volume claims with the name of the desired storage class. Convenience script:

```bash
#!/usr/bin/env bash

# This script requires https://github.com/mikefarah/yq v4 or greater

# Set SC to your storage class name
SC=

PVC=()
STS=()
mapfile -t PVC < <(fd --absolute-path --extension yaml "PersistentVolumeClaim" base)
mapfile -t STS < <(fd --absolute-path --extension yaml "StatefulSet" base)

for p in "${PVC[@]}"; do yq eval -i ".spec.storageClassName|=\"$SC\"" "$p"; done
for s in "${STS[@]}"; do yq eval -i ".spec.volumeClaimTemplates.[].spec.storageClassName|=\"$SC\"" "$s"; done
```

## Configure custom Redis

Sourcegraph supports specifying a custom Redis server for:

- caching information (specified via the `REDIS_CACHE_ENDPOINT` environment variable)
- storing information (session data and job queues) (specified via the `REDIS_STORE_ENDPOINT` environment variable)

If you want to specify a custom Redis server, you'll need specify the corresponding environment variable for each of the following deployments:

- `sourcegraph-frontend`
- `repo-updater`

## Configure custom PostgreSQL

You can use your own PostgreSQL v12+ server with Sourcegraph if you wish. For example, you may prefer this if you already have existing backup infrastructure around your own PostgreSQL server, wish to use Amazon RDS, etc.

Simply edit the relevant PostgreSQL environment variables (e.g. PGHOST, PGPORT, PGUSER, [etc.](http://www.postgresql.org/docs/current/static/libpq-envars.html)) in [base/frontend/sourcegraph-frontend.Deployment.yaml](https://github.com/sourcegraph/deploy-sourcegraph/blob/master/base/frontend/sourcegraph-frontend.Deployment.yaml) to point to your existing PostgreSQL instance.

## Install without cluster-wide RBAC

Sourcegraph communicates with the Kubernetes API for service discovery. It also has some janitor DaemonSets that clean up temporary cache data. To do that we need to create RBAC resources.

If using cluster roles and cluster rolebinding RBAC is not an option, then you can use the [non-privileged](https://github.com/sourcegraph/deploy-sourcegraph/blob/master/overlays/non-privileged) overlay to generate modified manifests. Read the [Overlays](#overlays) section below about overlays.


## Add license key

Sourcegraph's Kubernetes deployment [requires an Enterprise license key](https://about.sourcegraph.com/pricing).

- Create an account on or sign in to sourcegraph.com, and go to https://sourcegraph.com/subscriptions/new to obtain a license key.

- Once you have a license key, add it to your [site configuration](https://docs.sourcegraph.com/admin/config/site_config).



## Overlays

An overlay specifies customizations for a base directory of Kubernetes manifests. It enables us to change parameters (number of replicas, namespace, etc) for Kubernetes components without affecting the base directory. Read the [Overlays docs](overlays.md) for more information about using overlays with Sourcegraph.

### Use non-default namespace

Modifying the base manifests to use a non-default namespace can be done using the [namespaced overlay](overlays.md#use-non-default-namespace).



## Pulling images locally

In some cases, a site admin may want to pull all Docker images used in the cluster locally. For
example, if your organization requires use of a private registry, you may need to do this as an
intermediate step to mirroring them on the private registry. The following script accomplishes this
for all images under `base/`:

```bash
for IMAGE in $(grep --include '*.yaml' -FR 'image:' base | awk '{ print $(NF) }'); do docker pull "$IMAGE"; done;
```

## Troubleshooting

See the [Troubleshooting docs](troubleshoot.md).
