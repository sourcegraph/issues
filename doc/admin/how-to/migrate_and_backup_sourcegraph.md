# Backing up or migrating a Sourcegraph instance

In some circumstances it may be necessary or advantageous to migrate from one Sourcegraph instance or deployment to another. Below are three migration options.

## Option 1: Configuration only

The easiest option is to simply back up or migrate [configuration JSON data](../background-information/data_storage.md#configuration-json). See the guide on [how to backup configuration files](backup_configuration_files.md) for step-by-step instructions.

## Option 2: All Postgres data

This option provides a more complete backup, and ensures that almost all state will be restored. Repositories will have to be recloned and reindexed, so some downtime will be required while these oprations complete.

Follow the instructions in our [Docker to Docker Compose migration guide](../install/docker-compose/migrate.md#backup-single-docker-image-database) to generate a dump of Sourcegraph's Postgres database. [Contact us](https://about.sourcegraph.com/contact/sales) for specific recommendations for your deployment type.

## Option 3: All data

Backing up all persistent volumes is the most complete option. Instructions for doing this depend on the deployment method and the cloud host. [Contact us](https://about.sourcegraph.com/contact/sales) to discuss more.
