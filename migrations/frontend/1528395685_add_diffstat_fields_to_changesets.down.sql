BEGIN;

ALTER TABLE changesets DROP COLUMN IF EXISTS diff_stat_added;
ALTER TABLE changesets DROP COLUMN IF EXISTS diff_stat_changed;
ALTER TABLE changesets DROP COLUMN IF EXISTS diff_stat_deleted;
ALTER TABLE changesets DROP COLUMN IF EXISTS sync_state;

COMMIT;
