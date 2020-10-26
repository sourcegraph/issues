BEGIN;

ALTER TABLE campaigns RENAME COLUMN initial_applier_id TO author_id;

ALTER TABLE campaigns DROP COLUMN IF EXISTS last_applier_id;
ALTER TABLE campaigns DROP COLUMN IF EXISTS last_applied_at;

COMMIT;
