BEGIN;

ALTER TABLE changesets ADD COLUMN IF NOT EXISTS diff_stat_added integer;
ALTER TABLE changesets ADD COLUMN IF NOT EXISTS diff_stat_changed integer;
ALTER TABLE changesets ADD COLUMN IF NOT EXISTS diff_stat_deleted integer;
ALTER TABLE changesets ADD COLUMN IF NOT EXISTS sync_state jsonb DEFAULT '{}'::jsonb NOT NULL;

COMMIT;
