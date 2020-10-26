BEGIN;

-- We're pruning pre-spec campaigns and changesets so that we can tighten up
-- the schema to match the new spec-based way campaigns are represented.
-- However, we need to keep the data so that we can provide a somewhat graceful
-- migration, so the first thing we'll do is to create tables to retain old
-- campaigns and changesets.

CREATE TABLE IF NOT EXISTS
    campaigns_old
AS
    SELECT
        *
    FROM
        campaigns
    WHERE
        campaign_spec_id IS NULL;

-- This query's a bit tricky: what we want is changesets that are _only_
-- attached to campaigns without specs. Changesets that are (somehow) attached
-- to both a spec and non-spec campaign should be left untouched.

CREATE TABLE IF NOT EXISTS
    changesets_old
AS
    SELECT
        *
    FROM
        changesets
    WHERE
        campaign_ids ?| array(
            SELECT
                id::VARCHAR
            FROM
                campaigns
            WHERE campaign_spec_id IS NULL
        )
        AND NOT campaign_ids ?| array(
            SELECT
                id::VARCHAR
            FROM
                campaigns
            WHERE campaign_spec_id IS NOT NULL
        );

-- Now we've set up the tables, we can take the next step, which is to delete
-- the old campaigns and changesets.

DELETE FROM
    changesets
WHERE
    campaign_ids ?| array(
        SELECT
            id::VARCHAR
        FROM
            campaigns
        WHERE campaign_spec_id IS NULL
    )
    AND NOT campaign_ids ?| array(
        SELECT
            id::VARCHAR
        FROM
            campaigns
        WHERE campaign_spec_id IS NOT NULL
    );

DELETE FROM
    campaigns
WHERE
    campaign_spec_id IS NULL;

-- It was theoretically possible to end up with a NULL last_applied_at while
-- having a NOT NULL campaign_spec_id, as the migrations were not done at the
-- same time. While the likelihood of this happening for anyone other than
-- developers actively working on campaigns during the 3.19 cycle is
-- essentially zero, let's take care of it just in case.
UPDATE campaigns
    SET last_applied_at = created_at
    WHERE last_applied_at IS NULL;

-- Now we can alter our fields.

-- Set up the new NOT NULL constraints on the last applied at and campaign spec
-- ID fields.

ALTER TABLE campaigns
    ALTER COLUMN last_applied_at SET NOT NULL,
    ALTER COLUMN campaign_spec_id SET NOT NULL,
    ALTER COLUMN initial_applier_id DROP NOT NULL;

-- When a user is hard deleted, we don't want campaigns and specs to be deleted
-- just because their metadata is affected. We need to tweak their constraints
-- accordingly.

ALTER TABLE campaign_specs
    ALTER COLUMN user_id DROP NOT NULL,
    DROP CONSTRAINT IF EXISTS campaign_specs_user_id_fkey,
    ADD CONSTRAINT campaign_specs_user_id_fkey
        FOREIGN KEY (user_id)
        REFERENCES users (id)
        ON DELETE SET NULL
        DEFERRABLE;

ALTER TABLE campaigns
    DROP CONSTRAINT IF EXISTS campaigns_author_id_fkey,
    DROP CONSTRAINT IF EXISTS campaigns_last_applier_id_fkey,
    ADD CONSTRAINT campaigns_initial_applier_id_fkey
        FOREIGN KEY (initial_applier_id)
        REFERENCES users (id)
        ON DELETE SET NULL
        DEFERRABLE,
    ADD CONSTRAINT campaigns_last_applier_id_fkey
        FOREIGN KEY (last_applier_id)
        REFERENCES users (id)
        ON DELETE SET NULL
        DEFERRABLE;

COMMIT;
