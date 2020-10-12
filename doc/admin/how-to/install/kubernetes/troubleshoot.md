# Troubleshooting

If Sourcegraph does not start up or shows unexpected behavior, there are a variety of ways you can determine the root
cause of the failure. The most useful commands are:

- `kubectl get pods -o=wide` — lists all pods in your cluster and the corresponding health status of each.
- `kubectl logs -f $POD_NAME` — tails the logs for the specified pod.

If Sourcegraph is unavailable and the `sourcegraph-frontend-*` pod(s) are not in status `Running`, then view their logs with `kubectl logs -f sourcegraph-frontend-$POD_ID` (filling in `$POD_ID` from the `kubectl get pods` output). Inspect both the log messages printed at startup (at the beginning of the log output) and recent log messages.

Less frequently used commands:

- `kubectl describe $POD_NAME` — shows detailed info about the status of a single pod.
- `kubectl get pvc` — lists all Persistent Volume Claims (PVCs) and the status of each.
- `kubectl get pv` — lists all Persistent Volumes (PVs) that have been provisioned. In a healthy cluster, there should
  be a one-to-one mapping between PVs and PVCs.
- `kubectl get events` — lists all events in the cluster's history.
- `kubectl delete pod $POD_NAME` — delete a failing pod so it gets recreated, possibly on a different node
- `kubectl drain --force --ignore-daemonsets --delete-local-data $NODE` — remove all pods from a node and mark it as unschedulable to prevent new pods from arriving

### Common errors

- `Error from server (Forbidden): error when creating "base/frontend/sourcegraph-frontend.Role.yaml": roles.rbac.authorization.k8s.io "sourcegraph-frontend" is forbidden: attempt to grant extra privileges`

  - The account you are using to apply the Kubernetes configuration doesn't have sufficient permissions to create roles.
  - GCP: `kubectl create clusterrolebinding cluster-admin-binding --clusterrole cluster-admin --user $YOUR_EMAIL`

- `kubectl get pv` shows no Persistent Volumes, and/or `kubectl get events` shows a `Failed to provision volume with StorageClass "default"` error.

  Check that a storage class named "default" exists via `kubectl get storageclass`. If one does exist, run `kubectl get storageclass default -o=yaml` and verify that the zone indicated in the output matches the zone of your cluster.
  Google Cloud Platform users may need to [request an increase in storage quota](https://cloud.google.com/compute/quotas).

- Many pods are stuck in Pending status. Use `kubectl cluster-info dump > dump.txt` to obtain a dump of
  the logs. One thing to check for is insufficient resources:

  ```
   "Reason": "FailedScheduling",
   "Message": "0/3 nodes are available: 1 Insufficient memory, 3 Insufficient cpu.",
  ```

  This means that your cluster is under provisioned (i.e. has too few nodes, or not enough CPU and memory).
  If you're using Google Cloud Platform, note that the default node type is `n1-standard-1`, a machine
  with only one CPU, and that some components request a 2-CPU node. When creating a cluster, use
  `--machine-type=n1-standard-16`.

- You can't access Sourcegraph. See [Troubleshooting ingress-nginx](https://kubernetes.github.io/ingress-nginx/troubleshooting/). If you followed our instructions the namespace of the ingress-controller is `ingress-nginx`.

Any other issues? Contact us at [@srcgraph](https://twitter.com/srcgraph)
or <mailto:support@sourcegraph.com>, or file issues on
our [public issue tracker](https://github.com/sourcegraph/issues/issues).
