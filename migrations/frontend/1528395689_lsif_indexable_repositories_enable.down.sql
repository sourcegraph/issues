BEGIN;

ALTER TABLE lsif_indexable_repositories DROP COLUMN enabled;

COMMIT;
