# Configure PostgreSQL

TODO: Can this be deprecated and/or combined in favor of [quickstart step 2](quickstart_2_initialize_database.md)?

## Initialize

### 1. Install PostgreSQL

Install PostgreSQL `9.6`. See [PostgreSQL](../background-information/postgresql.md) for details.

### 2. Set up the user and database

Set up a `sourcegraph` user and database:

```
sudo su - postgres # this line only needed for Linux
createdb
createuser --superuser sourcegraph
psql -c "ALTER USER sourcegraph WITH PASSWORD 'sourcegraph';"
createdb --owner=sourcegraph --encoding=UTF8 --template=template0 sourcegraph
```

### 3. Set the timezone

Then update your `postgresql.conf` default timezone to UTC. Determine the location
of your `postgresql.conf` by running `psql -c 'show config_file;'`. Update the line beginning
with `timezone =` to the following:

```
timezone = 'UTC'
```

### 4. Restart your database server

Finally, restart your database server (mac: `brew services restart postgresql`, recent linux
probably `service postgresql restart`)

## Configure

### 1. Set your environment variables

Add the following to your `~/.bashrc`:

```
export PGPORT=5432
export PGHOST=localhost
export PGUSER=sourcegraph
export PGPASSWORD=sourcegraph
export PGDATABASE=sourcegraph
export PGSSLMODE=disable
```

#### Alternative `ENVAR` management

You can also use a tool like [`envdir`][s] or [a `.dotenv` file][dotenv] to
source these env vars on demand when you start the server.

[envdir]: https://cr.yp.to/daemontools/envdir.html
[dotenv]: https://github.com/joho/godotenv

### 2. Test your credentials

To test the environment's credentials, run `psql` (the PostgreSQL CLI
client) with the `PG*` environment variables set. If you see a
database prompt, then the environment's credentials are valid.

If you get an error message about "peer authentication", you are
probably connecting over the Unix domain socket, rather than over TCP.
Make sure you've set `PGHOST`. (Postgres can do peer authentication
on local sockets, which provides reliable identification but must
be specially configured to authenticate you as a user with a name
different from your account name.)
