FROM postgres:12.6@sha256:26ce5e265fbbf120b4a89a9b26770bad757a7dfa4f329038d93770edbe5aae00

# Install our previous version of Postgres so we can perform a data upgrade on
# container startup.
RUN sed -i 's/$/ 11/' /etc/apt/sources.list.d/pgdg.list

# Libnss-sss is required depedency, see comment on this line in infrastructure
RUN apt-get update && apt-get install -y --no-install-recommends \
    libnss-sss=1.16.3-3.2 \
    postgresql-11=11.12-0+deb10u1 \
    postgresql-contrib=11+200+deb10u4 \
    && rm -rf /var/lib/apt/lists/*

RUN mkdir -p /data && chown postgres:postgres /data

# PGUSEROLD: A user that exists in the old database that can be used
#            to authenticate intermediate upgrade operations.
# PGUSERNEW: A user that must exist in the new database (upgraded or freshly created).
#
# PGDATABASEOLD: A database that exists in the old database that can be used
#                to authenticate intermediate upgrade operations. (e.g `psql -d`)
# PGDATABASENEW: A database that must exist in the new database (upgraded or freshly created).
#
# PGDATAOLD: The data directory containing the files of the old Postgres database to be upgraded.
# PGDATANEW: The data directory containing the upgraded Postgres data files, used by the new version of Postgres
#
# PGBINOLD: The directory containing the executables of the old version of Postgres.
# PGBINNEW: The directort containing the executables of the new version of Postgres we're running.
#
# POSTGRES_PASSWORD: Password of PGUSERNEW if it is newly created (i.e. it didn't exist in the old database)
#
ENV POSTGRES_PASSWORD='' \
    PGUSEROLD=sg \
    PGUSERNEW=sg \
    PGDATABASEOLD=sg \
    PGDATABASENEW=sg \
    PGDATAOLD=/data/pgdata-11 \
    PGDATANEW=/data/pgdata-12 \
    PGVERSIONOLD="11" \
    PGVERSIONNEW="12" \
    PGBINOLD=/usr/lib/postgresql/11/bin \
    PGBINNEW=/usr/lib/postgresql/12/bin
COPY rootfs /
USER postgres
ENTRYPOINT ["/postgres.sh"]
