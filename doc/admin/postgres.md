# Upgrading PostgreSQL

Sourcegraph uses PostgreSQL as its main internal database and this documentation describes how to upgrade PostgreSQL between major versions.

> NOTE: ⚠️ Upgrading the PostgreSQL database requires stopping your Sourcegraph deployment which will result in **downtime**.

## Version requirements

We support any version **starting from 9.6**.

## Upgrading single node Docker deployments

> NOTE: If you running PostgreSQL externally, see [Upgrading external PostgreSQL instances](postgres.md#upgrading-external-postgresql-instances)

When running a new version of Sourcegraph, it will check if the PostgreSQL data needs upgrading upon initialization.

There are two ways that the PostgreSQL data can be updated:

- On start-up with access to the Docker socket.
- Running a script that uses the PostgreSQL upgrade container directly.

### Option 1. Upgrading on start-up using the Docker socket

<p class="container">
  <div style="padding:56.25% 0 0 0;position:relative;">
    <iframe src="https://player.vimeo.com/video/315980428?color=0CB6F4&title=0&byline=0&portrait=0" style="position:absolute;top:0;left:0;width:100%;height:100%;" frameborder="0" webkitallowfullscreen mozallowfullscreen allowfullscreen></iframe>
  </div>
</p>

**1.** Stop the `sourcegraph/server` container.

**2.** Add the volume mount code to your existing `docker run` command: `-v /var/run/docker.sock:/var/run/docker.sock:ro`.

See a complete example below:

```bash
# Add "--env=SRC_LOG_LEVEL=dbug" below for verbose logging.
docker run -p 7080:7080 -p 2633:2633 --rm \
  -v ~/.sourcegraph/config:/etc/sourcegraph \
  -v ~/.sourcegraph/data:/var/opt/sourcegraph \
  -v /var/run/docker.sock:/var/run/docker.sock:ro \
  sourcegraph/server:3.21.2
```

**3.** When the upgrade has been completed, stop the Sourcegraph container, then run again using the original `docker run` command (without mounting the Docker socket).

### Option 2. Upgrading with a script that uses the PostgreSQL upgrade container directly

<p class="container">
  <div style="padding:56.25% 0 0 0;position:relative;">
    <iframe src="https://player.vimeo.com/video/315980439?color=0CB6F4&title=0&byline=0&portrait=0" style="position:absolute;top:0;left:0;width:100%;height:100%;" frameborder="0" webkitallowfullscreen mozallowfullscreen allowfullscreen></iframe>
  </div>
</p>

You may need to manually upgrade the PostgreSQL data, e.g, if mounting the Docker socket isn't an option.

**1.** Stop the `sourcegraph/server` container.

**2.** Save this script and give it executable permissions (`chmod + x`).

> NOTE: The script presumes your data is being upgraded from `9.6` to `11` and your Sourcegraph directory is at `~/.sourcegraph/`. Change the values in the code below if that's not the case.

```bash
#!/usr/bin/env bash

set -xeuo pipefail

export OLD=${OLD:-"9.6"}
export NEW=${NEW:-"11"}
export SRC_DIR=${SRC_DIR:-"$HOME/.sourcegraph"}

docker run \
  -w /tmp/upgrade \
  -v "$SRC_DIR/data/postgres-$NEW-upgrade:/tmp/upgrade" \
  -v "$SRC_DIR/data/postgresql:/var/lib/postgresql/$OLD/data" \
  -v "$SRC_DIR/data/postgresql-$NEW:/var/lib/postgresql/$NEW/data" \
  "tianon/postgres-upgrade:$OLD-to-$NEW"

mv "$SRC_DIR/data/"{postgresql,postgresql-$OLD}
mv "$SRC_DIR/data/"{postgresql-$NEW,postgresql}

curl -fsSL -o "$SRC_DIR/data/postgres-$NEW-upgrade/optimize.sh" https://raw.githubusercontent.com/sourcegraph/sourcegraph/master/cmd/server/rootfs/postgres-optimize.sh

docker run \
  --entrypoint "/bin/bash" \
  -w /tmp/upgrade \
  -v "$SRC_DIR/data/postgres-$NEW-upgrade:/tmp/upgrade" \
  -v "$SRC_DIR/data/postgresql:/var/lib/postgresql/data" \
  "postgres:$NEW" \
  -c 'chown -R postgres $PGDATA . && gosu postgres bash ./optimize.sh $PGDATA'
```

**3.** Execute the script.

**4.** Start the `sourcegraph/server` container.

## Upgrading Kubernetes PostgreSQL instances

The upgrade process is different for [Sourcegraph cluster deployments](https://github.com/sourcegraph/deploy-sourcegraph) because [by default](https://github.com/sourcegraph/deploy-sourcegraph/blob/7edcadbc3ebf46cb1bc1198f8a3e359a2380e22a/base/pgsql/pgsql.Deployment.yaml#L29), it uses `sourcegraph/postgres-11.1:19-02-07_17a4376e` which can be [customized with environment variables](https://github.com/sourcegraph/deploy-sourcegraph/blob/7edcadb/docs/configure.md#configure-custom-postgresql).

If you have changed `PGUSER`, `PGDATABASE` or `PGDATA`, then the `PG*OLD` and `PG*NEW` environment variables are required. Below are the defaults and documentation on what each variable is used for:

- `POSTGRES_PASSWORD=''`: Password of `PGUSERNEW` if it is newly created (i.e when `PGUSERNEW` didn't exist in the old database).
- `PGUSEROLD=sg`: A user that exists in the old database that can be used to authenticate intermediate upgrade operations.
- `PGUSERNEW=sg`: A user that must exist in the new database after the upgrade is done (i.e. it'll be created if it didn't exist already).
- `PGDATABASEOLD=sg`: A database that exists in the old database that can be used to authenticate intermediate upgrade operations. (e.g `psql -d`)
- `PGDATABASENEW=sg`: A database that must exist in the new database after the upgrade is done (i.e. it'll be created if it didn't exist already).
- `PGDATAOLD=/data/pgdata`: The data directory containing the files of the old PostgreSQL database to be upgraded.
- `PGDATANEW=/data/pgdata-11`: The data directory containing the upgraded PostgreSQL data files, used by the new version of PostgreSQL.

Additionally the upgrade process assumes it can write to the parent directory of `PGDATAOLD`.

## Upgrading external PostgreSQL instances

When running an external PostgreSQL instance, please refer to your provider documentation for upgrade procedures.

- [AWS RDS](https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/USER_UpgradeDBInstance.PostgreSQL.html)
- [AWS Aurora](https://docs.aws.amazon.com/AmazonRDS/latest/AuroraUserGuide/USER_UpgradeDBInstance.Upgrading.html)
- [GCP CloudSQL](https://cloud.google.com/sql/docs/postgres/db-versions)
- [Azure DB](https://docs.microsoft.com/en-us/azure/postgresql/concepts-supported-versions#managing-updates-and-upgrades)
- [Heroku](https://devcenter.heroku.com/articles/upgrading-heroku-postgres-databases)
- [EnterpriseDB](https://www.enterprisedb.com/docs/en/9.6/pg/upgrading.html)
- [Citus](http://docs.citusdata.com/en/v8.1/admin_guide/upgrading_citus.html)
- [Aiven PostgreSQL](https://help.aiven.io/postgresql/operations/how-to-perform-a-postgresql-in-place-major-version-upgrade)
- [Your own PostgreSQL](https://www.postgresql.org/docs/11/pgupgrade.html)
