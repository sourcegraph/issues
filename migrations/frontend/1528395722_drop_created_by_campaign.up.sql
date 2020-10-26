BEGIN;

ALTER TABLE changesets DROP COLUMN IF EXISTS created_by_campaign;

COMMIT;
