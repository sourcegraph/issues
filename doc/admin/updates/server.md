# Updating a single-image Sourcegraph instance (`sourcegraph/server`)

This document describes the exact changes needed to update a single-node Sourcegraph instance.

A new version of Sourcegraph is released every month (with patch releases in between, released as needed). Check the [Sourcegraph blog](https://about.sourcegraph.com/blog) or the site admin updates page to learn about updates. We actively maintain the two most recent monthly releases of Sourcegraph.

Upgrades should happen across consecutive minor versions of Sourcegraph. For example, if you are running Sourcegraph 3.1 and want to upgrade to 3.3, you should upgrade to 3.2 and then 3.3.

**Always refer to this page before upgrading Sourcegraph,** as it comprehensively describes the steps needed to upgrade, and any manual migration steps you must perform.

## 3.20 -> 3.21

This release introduces a second database instance, `codeintel-db`. If you have configured Sourcegraph with an external database, then update the `CODEINTEL_PG*` environment variables to point to a new external database  as described in the [external database documentation](../external_database.md). Again, these must not point to the same database or the Sourcegraph instance will refuse to start.

### If you wish to keep existing LSIF data

> WARNING: **Do not upgrade out of the 3.21.x release branch** until you have seen the log message indicating the completion of the LSIF data migration, or verified that the `/lsif-storage/dbs` directory on the precise-code-intel-bundle-manager volume is empty. Otherwise, you risk data loss for precise code intelligence.

If you had LSIF data uploaded prior to upgrading to 3.21.0, there is a background migration that moves all existing LSIF data into the `codeintel-db` upon upgrade. Once this process completes, the `/lsif-storage/dbs` directory on the precise-code-intel-bundle-manager volume should be empty, and the bundle manager should print the following log message:

> Migration to Postgres has completed. All existing LSIF bundles have moved to the path /lsif-storage/db-backups and can be removed from the filesystem to reclaim space.

**Wait for the above message to be printed in `docker logs precise-code-intel-bundle-manager` before upgrading to the next Sourcegraph version**.

### Turn off database secrets encryption

> WARNING: Please delete the secret key file `/var/lib/sourcegraph/token` inside the container before attempting to upgrade to 3.21.x.

In Sourcegraph version 3.20, we would automatically generate a secret key file (`/var/lib/sourcegraph/token`) inside the container for encrypting secrets stored in the database. However, it is not yet ready for general use and format of the secret key file might change. Therefore, it is best to delete the secret key file (`/var/lib/sourcegraph/token`) and turn off the database secrets encryption.

### Standard upgrade procedure

To update, just use the newer `sourcegraph/server:N.N.N` Docker image (where `N.N.N` is the version number) in place of the older one, using the same Docker volumes. Your server's data will be migrated automatically if needed.

You can always find the version number of the latest release at [docs.sourcegraph.com](https://docs.sourcegraph.com) in the `docker run` command's image tag.

- As a precaution, before updating, we recommend backing up the contents of the Docker volumes used by Sourcegraph.
- If you need a HA deployment, use the [Kubernetes cluster deployment option](https://github.com/sourcegraph/deploy-sourcegraph).
- There is currently no automated way to downgrade to an older version after you have updated. [Contact support](https://about.sourcegraph.com/contact) for help.
