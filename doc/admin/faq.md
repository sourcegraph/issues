# Administration FAQ

## How do I expose my Sourcegraph instance to a different host port when running locally?

Change the `docker` `--publish` argument to make it listen on the specific interface and port on your host machine. For example, `docker run ... --publish 0.0.0.0:80:7080 ...` would make it accessible on port 80 of your machine. For more information, see "[Publish or expose port](https://docs.docker.com/engine/reference/commandline/run/#publish-or-expose-port--p---expose)" in the Docker documentation.

The other option is to deploy and run Sourcegraph on a cloud provider. For an example, see documentation to [deploy to Google Cloud](install/docker/google_cloud.md).

## How do I access the Sourcegraph database?

> NOTE: To execute an SQL query against the database without first creating an interactive session (as below), append `--command "SELECT * FROM users;"` to the `docker container exec` command.

### For single-node deployments (`sourcegraph/server`)

Get the Docker container ID for Sourcegraph:

```bash
docker ps
CONTAINER ID        IMAGE
d039ec989761        sourcegraph/server:VERSION
```

Open a PostgreSQL interactive terminal:

```bash
docker container exec -it d039ec989761 psql -U postgres sourcegraph
```

Run your SQL query:

```sql
SELECT * FROM users;
```

### For Kubernetes cluster deployments

Get the id of one `pgsql` Pod:

```bash
kubectl get pods -l app=pgsql
NAME                     READY     STATUS    RESTARTS   AGE
pgsql-76a4bfcd64-rt4cn   2/2       Running   0          19m
```

Make sure you are operating under the correct namespace (i.e. add `-n prod` if your pod is under the `prod` namespace).

Open a PostgreSQL interactive terminal:

```bash
kubectl exec -it pgsql-76a4bfcd64-rt4cn -- psql -U sg
```

Run your SQL query:

```sql
SELECT * FROM users;
```

## How does Sourcegraph store repositories on disk?

Sourcegraph stores bare Git repositories (without a working tree), which is a complete mirror of the repository on your code host.

If you are keen for more details on what bare Git repositories are, [check out this discussion on StackOverflow](https://stackoverflow.com/q/5540883).

The directories should contain just a few files and directories, namely: HEAD, config, description, hooks, info, objects, packed-refs, refs

## Does Sourcegraph support svn?

For Subversion and other non-Git code hosts, the recommended way to make these accessible in
Sourcegraph is through [`src-expose`](external_service/other.md#experimental-src-expose).

Alternatively, you can use [`git-svn`](https://git-scm.com/docs/git-svn) or
[`svg2git`](https://github.com/svn-all-fast-export/svn2git) to convert Subversion repositories to
Git repositories. Unlike `src-expose`, this will preserve commit history, but is generally much
slower.

## How do I access Sourcegraph if my authentication provider experiences an outage?

If you are using an external authentication provider and the provider experiences an outage, users
will be unable to sign into Sourcegraph. A site administrator can configure an alternate sign-in
method by modifying the `auth.providers` field in site configuration. However, the site
administrator may themselves be unable to sign in. If this is the case, then a site administrator
can update the configuration if they have direct `docker exec` or `kubectl exec` access to the
Sourcegraph instance. Follow the [instructions to update the site config if the web UI is
inaccessible](config/site_config.md#editing-your-site-configuration-if-you-cannot-access-the-web-ui).

## How do I set up redirect URLs in Sourcegraph?

Sometimes URLs in Sourcegraph may change. For example, if a code host configuration is
updated to use a different `repositoryPathPattern`, this will change the repository URLs on
Sourcegraph. Users may wish to preserve links to the old URLs, and this requires adding redirects.

We recommend configuring redirects in a reverse proxy. If you are running Sourcegraph as a single
Docker image, you can deploy a reverse proxy such as [Caddy](https://caddyserver.com/) or
[NGINX](https://www.nginx.com) in front of it. Refer to the
[Caddy](https://github.com/caddyserver/caddy/wiki/v2:-Documentation#rewrite) or
[NGINX](https://www.nginx.com/blog/creating-nginx-rewrite-rules/) documentation for URL rewrites.

If you are running Sourcegraph as a Kubernetes cluster, you have two additional options:

1. If you are using [NGINX
   ingress](https://github.com/sourcegraph/deploy-sourcegraph/blob/master/docs/configure.md#ingress-controller-recommended)
   (`kubectl get ingress | grep sourcegraph-frontend`), modify
   [`sourcegraph-frontend.Ingress.yaml`](https://github.com/sourcegraph/deploy-sourcegraph/blob/master/base/frontend/sourcegraph-frontend.Ingress.yaml)
   by [adding a rewrite rule](https://kubernetes.github.io/ingress-nginx/examples/rewrite/).
1. If you are using the [NGINX
   service](https://github.com/sourcegraph/deploy-sourcegraph/blob/master/docs/configure.md#nginx-service),
   modify
   [`nginx.ConfigMap.yaml`](https://github.com/sourcegraph/deploy-sourcegraph/blob/master/configure/nginx-svc/nginx.ConfigMap.yaml).

## Troubleshooting

Content moved to a [dedicated troubleshooting page](troubleshooting.md).
