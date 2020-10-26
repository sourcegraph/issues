BEGIN;

ALTER TABLE lsif_nearest_uploads DROP COLUMN ancestor_visible;
ALTER TABLE lsif_nearest_uploads DROP COLUMN overwritten;

COMMIT;
