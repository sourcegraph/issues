BEGIN;

-- Drop references to old tables.
ALTER TABLE campaigns DROP COLUMN IF EXISTS patch_set_id;

-- Now drop all old tables.
DROP TABLE changeset_jobs;
DROP TABLE patches;
DROP TABLE patch_sets;

COMMIT;
