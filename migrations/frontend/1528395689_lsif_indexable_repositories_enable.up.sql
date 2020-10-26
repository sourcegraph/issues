BEGIN;

ALTER TABLE lsif_indexable_repositories ADD COLUMN enabled boolean;

COMMIT;
