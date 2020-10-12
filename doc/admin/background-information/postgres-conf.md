# PostgreSQL configuration

Sourcegraph Kubernetes cluster site admins can override the default PostgreSQL configuration by supplying their own `postgresql.conf` file contents. These are specified in [`pgsql.ConfigMap.yaml`](https://github.com/sourcegraph/deploy-sourcegraph/blob/master/base/pgsql/pgsql.ConfigMap.yaml).

There is no officially supported way of customizing the PostgreSQL configuration in the single Docker image.
