BEGIN;

DROP TRIGGER IF EXISTS trig_read_only_repo_sources_column ON repo;

DROP FUNCTION IF EXISTS make_repo_sources_column_read_only();

ALTER TABLE repo DROP COLUMN IF EXISTS sources;

COMMIT;
