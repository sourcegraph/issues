# Updating a Kubernetes Sourcegraph instance

This document describes the exact changes needed to update a Kubernetes Sourcegraph instance.
Follow the [recommended method](../install/kubernetes/update.md) of upgrading a Kubernetes cluster.

A new version of Sourcegraph is released every month (with patch releases in between, released as needed). Check the [Sourcegraph blog](https://about.sourcegraph.com/blog) or the site admin updates page to learn about updates. We actively maintain the two most recent monthly releases of Sourcegraph.

Upgrades should happen across consecutive minor versions of Sourcegraph. For example, if you are running Sourcegraph 3.1 and want to upgrade to 3.3, you should upgrade to 3.2 and then 3.3.

**Always refer to this page before upgrading Sourcegraph,** as it comprehensively describes the steps needed to upgrade, and any manual migration steps you must perform.

## 3.20 -> 3.21

Follow the [standard upgrade method](../install/kubernetes/update.md) to upgrade your deployment.

This release introduces a second database instance, `codeintel-db`. If you have configured Sourcegraph with an external database, then update the `CODEINTEL_PG*` environment variables to point to a new external database  as described in the [external database documentation](../external_database.md). Again, these must not point to the same database or the Sourcegraph instance will refuse to start.

### If you wish to keep existing LSIF data

> Warning: **Do not upgrade out of the 3.21.x release branch** until you have seen the log message indicating the completion of the LSIF data migration, or verified that the `/lsif-storage/dbs` directory on the precise-code-intel-bundle-manager volume is empty. Otherwise, you risk data loss for precise code intelligence.

If you had LSIF data uploaded prior to upgrading to 3.21.0, there is a background migration that moves all existing LSIF data into the `codeintel-db` upon upgrade. Once this process completes, the `/lsif-storage/dbs` directory on the precise-code-intel-bundle-manager volume should be empty, and the bundle manager should print the following log message:

> Migration to Postgres has completed. All existing LSIF bundles have moved to the path /lsif-storage/db-backups and can be removed from the filesystem to reclaim space.

**Wait for the above message to be printed in `docker logs precise-code-intel-bundle-manager` before upgrading to the next Sourcegraph version**.

## 3.20

No manual migration is required, follow the [standard upgrade method](../install/kubernetes/update.md) to upgrade your deployment.

## 3.19

No manual migration is required, follow the [standard upgrade method](../install/kubernetes/update.md) to upgrade your deployment.

> Warning: If you use an overlay that does not reference one of the provided overlays, please add `- ../bases/pvcs` as an additional base
to your `kustomization.yaml` file. Otherwise the PVCs could be pruned if `kubectl apply -prune` is used.

## 3.18

No manual migration is required, follow the [standard upgrade method](../install/kubernetes/update.md) to upgrade your deployment.

## 3.17

No manual migration is required, follow the [standard upgrade method](../install/kubernetes/update.md) to upgrade your deployment.

## 3.16

No manual migration is required, follow the [standard upgrade method](../install/kubernetes/update.md) to upgrade your deployment.

Note: The following deployments have had their `strategy` changed from `rolling` to `recreate`:

- redis-cache
- redis-store
- pgsql
- precise-code-intel-bundle-manager
- prometheus
  
This change was made to avoid two pods writing to the same volume and causing corruption. No special action is needed to apply the change.

## 3.15

### Note: Prometheus and Grafana resource requirements increase

Resource _requests and limits_ for Grafana and Prometheus are now equal to the following:

- Grafana 100Mi -> 512Mi
- Prometheus: 500M -> 3G

This change was made to ensure that even if another Sourcegraph service starts consuming more memory than expected and the Kubernetes node has been over-provisioned, that Sourcegraph's monitoring will still have enough memory to run and monitor / send alerts to the site admin. For additional information see [#638](https://github.com/sourcegraph/deploy-sourcegraph/pull/638)

### (optional) Keep LSIF data through manual migration

If you have previously uploaded LSIF precise code intelligence data and wish to retain it after upgrading, you will need to perform this migration.

**Skipping the migration**

If you choose not to migrate the data, Sourcegraph will use basic code intelligence until you upload LSIF data again.

You may run the following commands to remove the now unused resources:

```shell script
kubectl delete svc lsif-server
kubectl delete deployment lsif-server
kubectl delete pvc lsif-server
```

**Migrating**

The lsif-server service has been replaced by a trio of services defined in [precise-code-intel](https://github.com/sourcegraph/deploy-sourcegraph/blob/master/base/precise-code-intel),
and the persistent volume claim in which lsif-server  stored converted LSIF uploads has been replaced by
[bundle storage](https://github.com/sourcegraph/deploy-sourcegraph/blob/master/base/precise-code-intel/bundle-storage.PersistentVolume.yaml).

Upgrading to 3.15 will create a new empty volume for LSIF data. Without any action, the LSIF data previously uploaded
to the instance will be lost. To retain old LSIF data, perform the following migration steps. This will cause some
temporary downtime for precise code intelligence.

**Migrating**

1. Deploy 3.15. This will create a `bundle-manager` persistent volume claim.
2. Release the claims to old and new persistent volumes by taking down `lsif-server` and `precise-code-intel-bundle-manager`.

```shell script
kubectl delete svc lsif-server
kubectl delete deployment lsif-server
kubectl delete deployment precise-code-intel-bundle-manager
```

3. Deploy the `lsif-server-migrator` deployment to transfer the data from the old volume to the new volume.

```shell script
kubectl apply -f configure/lsif-server-migrator/lsif-server-migrator.Deployment.yaml
```

4. Watch the output of the `lsif-server-migrator` until the copy completes (`'Copy complete!'`).

```shell script
kubectl logs lsif-server-migrator
```

5. Tear down the deployment and re-create the bundle manager deployment.

```shell script
kubectl delete deployment lsif-server-migrator
./kubectl-apply-all.sh
```

6. Remove the old persistent volume claim.

```shell script
kubectl delete pvc lsif-server
```

## 3.11

In 3.11 we removed the management console. If you make use of `CRITICAL_CONFIG_FILE` or `SITE_CONFIG_FILE`, please refer to the [migration notes for Sourcegraph 3.11+](https://docs.sourcegraph.com/admin/migration/3_11).

## 3.10

In 3.9 we migrated `indexed-search` to a StatefulSet. However, we didn't migrate the `indexed-search` service to a headless service. You can't mutate a service, so you will need to replace the service before running `kubectl-apply-all.sh`:

``` bash
# Replace since we can't mutate services
kubectl replace --force -f base/indexed-search/indexed-search.Service.yaml

# Now apply all so frontend knows how to speak to the new service address
# for indexed-search
./kubectl-apply-all.sh
```

## 3.9

In 3.9 `indexed-search` is migrated from a Kubernetes [Deployment](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/) to a [StatefulSet](https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/). By default Kubernetes will assign a new volume to `indexed-search`, leading to it being unavailable while it reindexes. To avoid that we need to update the [PersistentVolume](https://kubernetes.io/docs/concepts/storage/persistent-volumes/)'s claim to the new indexed-search pod (from `indexed-search` to `data-indexed-search-0`. This can be achieved by running the commands in the script below before upgrading. Please read the script closely to understand what it does before following it.

``` bash
# Set the reclaim policy to retain so when we delete the volume claim the volume is not deleted.
kubectl patch pv -p '{"spec":{"persistentVolumeReclaimPolicy":"Retain"}}' $(kubectl get pv -o json | jq -r '.items[] | select(.spec.claimRef.name == "indexed-search").metadata.name') 

# Stop indexed search so we can migrate it. This means indexed search will be down!
kubectl scale deploy/indexed-search --replicas=0

# Remove the existing claim on the volume
kubectl delete pvc indexed-search

# Move the claim to data-indexed-search-0, which is the name created by stateful set.
kubectl patch pv -p '{"spec":{"claimRef":{"name":"data-indexed-search-0","uuid":null}}}' $(kubectl get pv -o json | jq -r '.items[] | select(.spec.claimRef.name == "indexed-search").metadata.name') 

# Create the stateful set
kubectl apply -f base/indexed-search/indexed-search.StatefulSet.yaml
```

## 3.8

If you're deploying Sourcegraph into a non-default namespace, refer to ["Use non-default namespace" in docs/configure.md](../install/kubernetes/configure.md#use-non-default-namespace) for further configuration instructions.

## 3.7.2

Before upgrading or downgrading 3.7, please consult the [v3.7.2 migration guide](https://docs.sourcegraph.com/admin/migration/3_7) to ensure you have enough free disk space.

## 3.0

🚨 If you have not migrated off of helm yet, please refer to [helm.migrate.md](https://github.com/sourcegraph/deploy-sourcegraph/blob/v3.15.1/docs/helm.migrate.md) before reading the following notes for migrating to Sourcegraph 3.0.

🚨 Please upgrade your Sourcegraph instance to 2.13.x before reading the following notes for migrating to Sourcegraph 3.0.

### Configuration

In Sourcegraph 3.0 all site configuration has been moved out of the `config-file.ConfigMap.yaml` and into the PostgreSQL database. We have an automatic migration if you use version 3.2 or before. Please do not upgrade directly from 2.x to 3.3 or higher.

After running 3.0, you should visit the configuration page (`/site-admin/configuration`) and [the management console](https://docs.sourcegraph.com/admin/management_console) and ensure that your configuration is as expected. In some rare cases, automatic migration may not be able to properly carry over some settings and you may need to reconfigure them.

### `sourcegraph-frontend` service type 

The type of the `sourcegraph-frontend` service ([base/frontend/sourcegraph-frontend.Service.yaml](https://github.com/sourcegraph/deploy-sourcegraph/blob/master/base/frontend/sourcegraph-frontend.Service.yaml)) has changed
from `NodePort` to `ClusterIP`. Directly applying this change [will
fail](https://github.com/kubernetes/kubernetes/issues/42282). Instead, you must delete the old
service and then create the new one (this will result in a few seconds of downtime):

```shell
kubectl delete svc sourcegraph-frontend
kubectl apply -f base/frontend/sourcegraph-frontend.Service.yaml
```

### Language server deployment

Sourcegraph 3.0 removed lsp-proxy and automatic language server deployment in favor of [Sourcegraph extensions](https://docs.sourcegraph.com/extensions). As a consequence, Sourcegraph 3.0 does not automatically run or manage language servers. If you had code intelligence enabled in 2.x, you will need to follow the instructions for each language extension and deploy them individually. Read the [code intelligence documentation](https://docs.sourcegraph.com/user/code_intelligence).

### HTTPS / TLS

Sourcegraph 3.0 removed HTTPS / TLS features from Sourcegraph in favor of relying on [Kubernetes Ingress Resources](https://kubernetes.io/docs/concepts/services-networking/ingress/). As a consequence, Sourcegraph 3.0 does not expose TLS as the NodePort 30433. Instead you need to ensure you have setup and configured either an ingress controller (recommended) or an explicit NGINX service. See [ingress controller documentation](../install/kubernetes/configure.md#ingress-controller-recommended), [NGINX service documentation](../install/kubernetes/configure.md#nginx-service), and [configure TLS/SSL documentation](../install/kubernetes/configure.md#configure-tlsssl).

If you previously configured `TLS_KEY` and `TLS_CERT` environment variables, you can remove them from [base/frontend/sourcegraph-frontend.Deployment.yaml](https://github.com/sourcegraph/deploy-sourcegraph/blob/master/base/frontend/sourcegraph-frontend.Deployment.yaml)

### Postgres 11.1

Sourcegraph 3.0 ships with Postgres 11.1. The upgrade procedure is mostly automatic. Please read [this page](https://docs.sourcegraph.com/admin/postgres) for detailed information.

## 2.12

Beginning in version 2.12.0, Sourcegraph's Kubernetes deployment [requires an Enterprise license key](https://about.sourcegraph.com/pricing). Follow the steps in [docs/configure.md](../install/kubernetes/configure.md#add-a-license-key).
