# Using external databases with Sourcegraph

> NOTE: Using Sourcegraph with an external database is a [paid feature](https://about.sourcegraph.com/pricing). [Contact us](https://about.sourcegraph.com/contact/sales) to get a trial license.

Sourcegraph by default provides its own PostgreSQL and Redis databases for data storage:

- A PostgreSQL instance for storing long-term information, such as user information when using Sourcegraph's built-in authentication provider instead of an external one.
- A second PostgreSQL instance for storing large-volume precise code intelligence data.
- A pair of redis instances for storing short-term information, such as session information and cache data.

## Using your own PostgreSQL server

You can use your own PostgreSQL v9.6+ server with Sourcegraph if you wish. For example, you may prefer this if you already have existing backup infrastructure around your own PostgreSQL server, wish to use Amazon RDS, etc.

The addition of `PG*` environment variables to your Sourcegraph deployment files will instruct Sourcegraph to target an external PostgreSQL server. To externalize the _frontend database_, use the following standard `PG*` variables:

- `PGHOST`
- `PGPORT`
- `PGUSER`
- `PGPASSWORD`
- `PGDATABASE`
- `PGSSLMODE`

To externalize the _code intelligence database_, use the following prefixed `CODEINTEL_PG*` variables:

- `CODEINTEL_PGHOST`
- `CODEINTEL_PGPORT`
- `CODEINTEL_PGUSER`
- `CODEINTEL_PGPASSWORD`
- `CODEINTEL_PGDATABASE`
- `CODEINTEL_PGSSLMODE`

:warning: If you have configured both the frontend (pgsql) and code intelligence (codeintel-db) databases with the same values, the Sourcegraph instance will refuse to start. Each database should either be configured to point to distinct hosts (recommended), or configured to point to distinct databases on the same host.

### sourcegraph/server

Add the following to your `docker run` command:

<!--
  DO NOT CHANGE THIS TO A CODEBLOCK.
  We want line breaks for readability, but backslashes to escape them do not work cross-platform.
  This uses line breaks that are rendered but not copy-pasted to the clipboard.
-->

<pre class="pre-wrap start-sourcegraph-command"><code>docker run [...]<span class="virtual-br"></span> -e PGHOST=psql1.mycompany.org<span class="virtual-br"></span> -e PGUSER=sourcegraph<span class="virtual-br"></span> -e PGPASSWORD=secret<span class="virtual-br"></span> -e PGDATABASE=sourcegraph<span class="virtual-br"></span> -e PGSSLMODE=require<span class="virtual-br"> -e CODEINTEL_PGHOST=psql2.mycompany.org<span class="virtual-br"></span> -e CODEINTEL_PGUSER=sourcegraph<span class="virtual-br"></span> -e CODEINTEL_PGPASSWORD=secret<span class="virtual-br"></span> -e CODEINTEL_PGDATABASE=sourcegraph-codeintel<span class="virtual-br"></span> -e CODEINTEL_PGSSLMODE=require<span class="virtual-br"></span> sourcegraph/server:3.21.2</code></pre>

### Docker Compose

1. Add/modify the following environment variables to all of the `sourcegraph-frontend-*` services and the `sourcegraph-frontend-internal` service in [docker-compose.yaml](https://github.com/sourcegraph/deploy-sourcegraph-docker/blob/v3.21.0/docker-compose/docker-compose.yaml):

    ```
    sourcegraph-frontend-0:
      # ...
      environment:
        # ...
        - 'PGHOST=psql1.mycompany.org'
        - 'PGUSER=sourcegraph'
        - 'PGPASSWORD=secret'
        - 'PGDATABASE=sourcegraph'
        - 'PGSSLMODE=require'
        - 'CODEINTEL_PGHOST=psql2.mycompany.org'
        - 'CODEINTEL_PGUSER=sourcegraph'
        - 'CODEINTEL_PGPASSWORD=secret'
        - 'CODEINTEL_PGDATABASE=sourcegraph-codeintel'
        - 'CODEINTEL_PGSSLMODE=require'
      # ...
    ```

    See ["Environment variables in Compose"](https://docs.docker.com/compose/environment-variables/) for other ways to pass these environment variables to the relevant services (including from the command line, a `.env` file, etc.).

1. Comment out / remove the internal `pgsql` and `codeintel-db` services in [docker-compose.yaml](https://github.com/sourcegraph/deploy-sourcegraph-docker/blob/v3.21.0/docker-compose/docker-compose.yaml) since Sourcegraph is using the external one now.

    ```
    # # Description: PostgreSQL database for various data.
    # #
    # # Disk: 128GB / persistent SSD
    # # Ports exposed to other Sourcegraph services: 5432/TCP 9187/TCP
    # # Ports exposed to the public internet: none
    # #
    # pgsql:
    # container_name: pgsql
    # image: 'index.docker.io/sourcegraph/postgres-11.4:19-11-14_b084311b@sha256:072481559d559cfd9a53ad77c3688b5cf583117457fd452ae238a20405923297'
    # cpus: 4
    # mem_limit: '2g'
    # healthcheck:
    #    test: '/liveness.sh'
    #    interval: 10s
    #    timeout: 1s
    #    retries: 3
    #    start_period: 15s
    # volumes:
    #    - 'pgsql:/data/'
    # networks:
    #     - sourcegraph
    # restart: always

    # # Description: PostgreSQL database for code intelligence data.
    # #
    # # Disk: 128GB / persistent SSD
    # # Ports exposed to other Sourcegraph services: 5432/TCP 9187/TCP
    # # Ports exposed to the public internet: none
    # #
    # codeintel-db:
    #   container_name: codeintel-db
    #   image: 'index.docker.io/sourcegraph/codeintel-db@sha256:63090799b34b3115a387d96fe2227a37999d432b774a1d9b7966b8c5d81b56ad'
    #   cpus: 4
    #   mem_limit: '2g'
    #   healthcheck:
    #     test: '/liveness.sh'
    #     interval: 10s
    #     timeout: 1s
    #     retries: 3
    #     start_period: 15s
    #   volumes:
    #     - 'codeintel-db:/data/'
    #   networks:
    #     - sourcegraph
    #   restart: always
    ```

### Kubernetes

Update the `PG*` and `CODEINTEL_PG*` environment variables in the `sourcegraph-frontend` deployment YAML file to point to the external frontend (`pgsql`) and code intelligence (`codeintel-db`) PostgreSQL instances, respectively. Again, these must not point to the same database or the Sourcegraph instance will refuse to start.

You are then free to remove the now unused `pgsql` and `codeintel-db` services and deployments from your cluster.

### Version requirements

Please refer to our [Postgres](https://docs.sourcegraph.com/admin/postgres) documentation to learn about version requirements.

### Caveats

> NOTE: If your PostgreSQL server does not support SSL, set `PGSSLMODE=disable` instead of `PGSSLMODE=require`. Note that this is potentially insecure.

Most standard PostgreSQL environment variables may be specified (`PGPORT`, etc). See http://www.postgresql.org/docs/current/static/libpq-envars.html for a full list.

> NOTE: On Mac/Windows, if trying to connect to a PostgreSQL server on the same host machine, remember that Sourcegraph is running inside a Docker container inside of the Docker virtual machine. You may need to specify your actual machine IP address and not `localhost` or `127.0.0.1` as that refers to the Docker VM itself.

## Using your own Redis server

**Version requirements**: We support any version *starting from 5.0*.

Generally, there is no reason to do this as Sourcegraph only stores ephemeral cache and session data in Redis. However, if you want to use an external Redis server with Sourcegraph, you can do the following:

Add the `REDIS_ENDPOINT` environment variable to your `docker run` command and Sourcegraph will use that Redis server instead of its built-in one. The string must either have the format `$HOST:PORT`
or follow the [IANA specification for Redis URLs](https://www.iana.org/assignments/uri-schemes/prov/redis) (e.g., `redis://:mypassword@host:6379/2`). For example:

<!--
  DO NOT CHANGE THIS TO A CODEBLOCK.
  We want line breaks for readability, but backslashes to escape them do not work cross-platform.
  This uses line breaks that are rendered but not copy-pasted to the clipboard.
-->
<pre class="pre-wrap"><code>docker run [...]<span class="virtual-br"></span>   -e REDIS_ENDPOINT=redis.mycompany.org:6379<span class="virtual-br"></span>   sourcegraph/server:3.21.2</code></pre>

> NOTE: On Mac/Windows, if trying to connect to a Redis server on the same host machine, remember that Sourcegraph is running inside a Docker container inside of the Docker virtual machine. You may need to specify your actual machine IP address and not `localhost` or `127.0.0.1` as that refers to the Docker VM itself.

If using Docker for Desktop, `host.docker.internal` will resolve to the host IP address.
