BEGIN;

ALTER TABLE IF EXISTS changesets DROP COLUMN IF EXISTS changeset_spec_id;
DROP TABLE IF EXISTS changeset_specs;

ALTER TABLE IF EXISTS campaigns DROP COLUMN IF EXISTS campaign_spec_id;
DROP TABLE IF EXISTS campaign_specs;

COMMIT;
