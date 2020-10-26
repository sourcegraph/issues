BEGIN;

-- We need two references to changeset_specs: one current spec and the previous one.
-- We use the already-existing column, changeset_spec_id, and rename it to current_spec_id.
ALTER TABLE changesets RENAME COLUMN changeset_spec_id TO current_spec_id;
ALTER TABLE changesets ADD COLUMN IF NOT EXISTS previous_spec_id bigint REFERENCES changeset_specs(id) DEFERRABLE;

-- Now we add the 'publication_state' field to changesets.
-- See ./internal/campaigns/types.go for the possible values here:
--   - UNPUBLISHED
--   - PUBLISHING
-- We use UNPUBLISHED was the default value
ALTER TABLE changesets ADD COLUMN IF NOT EXISTS publication_state text DEFAULT 'UNPUBLISHED';

-- Before switching to the new flow every changeset we had has been created
-- on the code host.
UPDATE changesets SET publication_state = 'PUBLISHED';

-- Since changesets can now be created in an "unpublished" state, we need to
-- make the following columns nullable:
ALTER TABLE changesets ALTER COLUMN external_id DROP NOT NULL;
ALTER TABLE changesets ALTER COLUMN metadata DROP NOT NULL;

-- We also need this field to make it easier to keep track of which campaign
-- "owns" which changeset: a campaign that owns a changeset can create and
-- close it on the code host.
-- Other campaigns that don't own a changeset can merely import/track it.
ALTER TABLE changesets ADD COLUMN IF NOT EXISTS owned_by_campaign_id bigint REFERENCES campaigns(id) DEFERRABLE;

-- These columns are necessary to make the reconciler work using the workerutils package.
ALTER TABLE changesets ADD COLUMN IF NOT EXISTS reconciler_state text DEFAULT 'queued';
ALTER TABLE changesets ADD COLUMN IF NOT EXISTS failure_message text;
ALTER TABLE changesets ADD COLUMN IF NOT EXISTS started_at timestamp with time zone;
ALTER TABLE changesets ADD COLUMN IF NOT EXISTS finished_at timestamp with time zone;
ALTER TABLE changesets ADD COLUMN IF NOT EXISTS process_after timestamp with time zone;
ALTER TABLE changesets ADD COLUMN IF NOT EXISTS num_resets integer NOT NULL DEFAULT 0;

-- Every changeset we have so far has been completed.
UPDATE changesets
SET reconciler_state = 'completed',
    started_at = created_at,
    finished_at = created_at;

COMMIT;
