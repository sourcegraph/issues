# How to gather kubernetes logs for troubleshooting

This document will take you through how to gather kubernetes logs for problems like a failing upgrade.

## Prerequisites

* This document assumes that you have Sourcegraph installed and are running into an issue such as a failing upgrade
* Assumes you have sufficient privileges to run kubectl commands on your Sourcegraph instance

## Steps to gather logs

Gather the following output from each of the services that are failing and export to a .txt file if appropriate.

* For example, first run `kubectl get pods` to see which pods are failing to start.

Then grab the following output from each of the services. 

* Examples of services are: sourcegraph-frontend, pgsql, and codeintel-db.

1. Logs: `kubectl logs [POD_NAME]`
2. Describe output (deployment): `kubectl describe deployment [NAME]`
3. Describe output (pod): `kubectl describe pod [POD_NAME]`
4. YAML configuration for each of those services: `kubectl get deployment [NAME] -o yaml`

If you need further assistance after reviewing the logs then reach out to Sourcegraph Customer Support at support@sourcegraph.com.


## Further resources

* [Kubernetes command reference page](https://kubernetes.io/docs/reference/kubectl/cheatsheet/)
* [Sourcegraph - Contact page](https://about.sourcegraph.com/contact/)
