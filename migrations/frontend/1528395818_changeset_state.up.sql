BEGIN;

ALTER TABLE changesets ADD COLUMN IF NOT EXISTS changeset_state text;


COMMIT;
