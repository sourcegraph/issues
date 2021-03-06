BEGIN;

DROP INDEX IF EXISTS repo_is_blocked_idx;
DROP INDEX IF EXISTS repo_is_not_blocked_idx;

DROP FUNCTION IF EXISTS repo_block;

ALTER TABLE IF EXISTS repo DROP COLUMN IF EXISTS blocked;

COMMIT;
