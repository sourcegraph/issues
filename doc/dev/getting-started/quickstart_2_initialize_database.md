# Quickstart step 2: Initialize your database

## Without Docker

You need a fresh Postgres database and a database user that has full ownership of that database.

1. Create a database for the current Unix user

    ```
    # For Linux users, first access the postgres user shell
    sudo su - postgres
    # For Mac OS users
    sudo su - _postgres
    ```

    ```
    createdb
    ```

2. Create the Sourcegraph user and password

    ```
    createuser --superuser sourcegraph
    psql -c "ALTER USER sourcegraph WITH PASSWORD 'sourcegraph';"
    ```

3. Create the Sourcegraph database

    ```
    createdb --owner=sourcegraph --encoding=UTF8 --template=template0 sourcegraph
    ```

4. Configure database settings in your environment

    The Sourcegraph server reads PostgreSQL connection configuration from the [`PG*` environment variables](http://www.postgresql.org/docs/current/static/libpq-envars.html).

    Add these, for example, in your `~/.bashrc`:

    ```
    export PGPORT=5432
    export PGHOST=localhost
    export PGUSER=sourcegraph
    export PGPASSWORD=sourcegraph
    export PGDATABASE=sourcegraph
    export PGSSLMODE=disable
    ```

    You can also use a tool like [`envdir`][envdir] or [a `.dotenv` file][dotenv] to
    source these env vars on demand when you start the server.

    [envdir]: https://cr.yp.to/daemontools/envdir.html
    [dotenv]: https://github.com/joho/godotenv

## With Docker

You may also want to run Postgres within a docker container instead of as a system service. Running within a container provides some advantages such as storing the data seperately from the container, you do not need to run it as a system service and its easy to use different database versions or multiple databases.

1. Create a directory to store and mount the database from for persistence:

    ```
    # Create a seperate dir to store the database
    mkdir PGDATA_DIR

   # Also add this to your '~/.bashrc'
    export PGDATA_DIR=/path/to/PGDATA_DIR/
    ```

2. Run the container:

  ```
   docker run -d  -p 5432:5432 -e POSTGRES_PASSWORD=sourcegraph \
   -e POSTGRES_USER=sourcegraph -e POSTGRES_INITDB_ARGS=" --encoding=UTF8 " \
   -v $PGDATA_DIR:/var/lib/postgresql/data postgres
   ```

3. Ensure you can connect to the database using `psql -U sourcegraph` and enter password `sourcegraph`.

4. Configure database settings in your environment:

    The Sourcegraph server reads PostgreSQL connection configuration from the [`PG*` environment variables](http://www.postgresql.org/docs/current/static/libpq-envars.html).

    Add these, for example, in your `~/.bashrc`:

    ```
    export PGPORT=5432
    export PGHOST=localhost
    export PGUSER=sourcegraph
    export PGPASSWORD=sourcegraph
    export PGDATABASE=sourcegraph
    export PGSSLMODE=disable
    ```

    You can also use a tool like [`envdir`][envdir] or [a `.dotenv` file][dotenv] to source these env vars on demand when you start the server.

    [envdir]: https://cr.yp.to/daemontools/envdir.html
    [dotenv]: https://github.com/joho/godotenv

5. On restarting docker, you may need to start the container again. Find the image with `docker ps --all` and then `docker run <$containerID>` to start again.

## More info

For more information about data storage, [read our full PostgreSQL page](../background-information/postgresql.md).

Migrations are applied automatically.

[< Previous](quickstart_1_install_dependencies.md) | [Next >](quickstart_3_start_docker.md)
