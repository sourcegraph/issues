# Updating Sourcegraph


A new version of Sourcegraph is released every month (with patch releases in between, released as needed). Check the [Sourcegraph blog](https://about.sourcegraph.com/blog) for release announcements.

> <br>
> ⚠️ IMPORTANT: Please check [Upgrading docs](../../updates/kubernetes.md) for critical version-specific requirements before upgrading to any particular
> version of Sourcegraph, to check if any manual migrations or changes are necessary.
> <br> 

## Steps

**These steps assume that you have created a `release` branch following the [forking instructions in docs/configure.md](configure.md#fork-this-repository)**

1. Merge the corresponding `upstream release` tag into your `release` branch _(created from the [forking instructions](#fork-this-repository) step)_. 

   ```bash
   # to add the upstream remote.
   git remote add upstream https://github.com/sourcegraph/deploy-sourcegraph
   ```

2. <a name="merge">Merge the new version of Sourcegraph into your release branch.</a>

   ```bash
   # Go to the directory for your fork of the sourcegraph/deploy-sourcegraph repository
   cd $DEPLOY_SOURCEGRAPH_FORK
   # Retrieve the updated list of releases available
   git fetch
   # Check out your release branch, ready to merge the upstream release tag into your release branch.
   git checkout release # Or use `git checkout -b release` to create the branch
   # Choose which version you want to deploy from https://github.com/sourcegraph/deploy-sourcegraph/releases
   git merge $NEW_VERSION # e.g. `git merge v3.28.0`
   
   ```

3. Deploy the updated version of Sourcegraph to your Kubernetes cluster:

   ```
   ./kubectl-apply-all.sh
   ```

4. Monitor the status of the deployment.

   ```
   kubectl get pods -o wide --watch
   ```

## Rollback

You can rollback by resetting your `release` branch to the old state and proceeding with [step 2 above](#merge).

To reset your branch to the default state:
```
git checkout -f release
```

> ⚠️ Warning: If an update you're rolling back _from_ included a database migration, rollback will require some manual DB modifications. 
> We plan to eliminate these in the near future, but for now, email <mailto:support@sourcegraph.com> if you have concerns before updating to a new release or have issues rolling back.

## Improving update reliability and latency with node selectors

Some of the services that comprise Sourcegraph require more resources than others, especially if the
default CPU or memory allocations have been overridden. During an update when many services restart,
you may observe that the more resource-hungry pods (e.g., `gitserver`, `indexed-search`) fail to
restart, because no single node has enough available CPU or memory to accommodate them. This may be
especially true if the cluster is heterogeneous (i.e., not all nodes have the same amount of
CPU/memory).

If this happens, do the following:

- Use `kubectl drain $NODE` to drain a node of existing pods, so it has enough allocation for the larger
  service.
- Run `watch kubectl get pods -o wide` and wait until the node has been drained. Run `kubectl get pods` to check that all pods except for the resource-hungry one(s) have been assigned to a node.
- Run `kubectl uncordon $NODE` to enable the larger pod(s) to be scheduled on the drained node.

Note that the need to run the above steps can be prevented altogether with [node
selectors](https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#nodeselector), which
tell Kubernetes to assign certain pods to specific nodes. See the [docs on enabling node
selectors](scale.md#node-selector) for Sourcegraph on Kubernetes.

## High-availability updates

Sourcegraph is designed to be a high-availability (HA) service, but upgrades by default require a 10m downtime
window. If you need zero-downtime upgrades, please contact us. Services employ health checks to test the health
of newly updated components before switching live traffic over to them by default. HA-enabling features include
the following:

- Replication: nearly all of the critical services within Sourcegraph are replicated. If a single instance of a
  service fails, that instance is restarted and removed from operation until it comes online again.
- Updates are applied in a rolling fashion to each service such that a subset of instances are updated first while
  traffic continues to flow to the old instances. Once the health check determines the set of new instances is
  healthy, traffic is directed to the new set and the old set is terminated. By default, some database operations
  may fail during this time as migrations occur so a scheduled 10m downtime window is required.
- Each service includes a health check that detects whether the service is in a healthy state. This check is specific to
  the service. These are used to check the health of new instances after an update and during regular operation to
  determine if an instance goes down.
- Database migrations are handled automatically on update when they are necessary.

### Updating blue-green deployments

Some users may wish to opt for running two separate Sourcegraph clusters running in a
[blue-green](https://martinfowler.com/bliki/BlueGreenDeployment.html) deployment. Such a setup makes
the update step more complex, but it can still be done with the `sourcegraph-server-gen snapshot`
command:

- **Preconditions:**
  - Suppose cluster A is currently live, and cluster B is in standby.
  - Clusters A and B should be running the same version of Sourcegraph.
  - Ensure `sourcegraph-server-gen` is upgraded to version 3.0.1 (`sourcegraph-server-gen update`)
- **Snapshot of A:** Configure `kubectl` to access cluster A and then run `sourcegraph-server-gen snapshot create`.
- **Restore A's snapshot to B:**
  - Configure `kubectl` to access B.
  - Spin down `sourcegraph-frontend` replicas to 0. (**Note:** this is very important, because
    otherwise `sourcegraph-frontend` may apply changes to the database that corrupt the snapshot
    restoration.)

    ```
    kubectl scale --replicas=0 deployment/sourcegraph-frontend
    ```

  - `sourcegraph-server-gen snapshot restore` from the same directory where you ran the snapshot creation earlier.
  - Spin up `sourcegraph-frontend` replicas to what it was before:

    ```
    kubectl scale --replicas=$N deployment/sourcegraph-frontend
    ```
- **Upgrade cluster B** to the new Sourcegraph version. Perform some quick checks to verify it is
  functioning.
- **Switch traffic over to B.** (B is now live.)
- **Upgrade cluster A** to the new Sourcegraph version.
- **Switch traffic back to A.** (A is now live again.)

After the update, cluster A will be live, cluster B will be in standby, and both will be running the
same new version of Sourcegraph. You may lose a few minutes of database updates while A is not live,
but that is generally acceptable.

To keep the database on B current, you may periodically wish to sync A's database over to B
(`sourcegraph-server-gen snapshot create` on A, `sourcegraph-server-gen snapshot restore` on B). It
is important that the versions of A and B are equivalent when this is done.

### Troubleshooting

See the [troubleshooting page](troubleshoot.md).
