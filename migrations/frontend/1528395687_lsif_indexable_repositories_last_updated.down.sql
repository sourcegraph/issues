BEGIN;

ALTER TABLE lsif_indexable_repositories DROP COLUMN last_updated_at;

COMMIT;
