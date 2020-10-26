BEGIN;

ALTER TABLE lsif_nearest_uploads ADD COLUMN ancestor_visible boolean;
ALTER TABLE lsif_nearest_uploads ADD COLUMN overwritten boolean;
UPDATE lsif_nearest_uploads SET ancestor_visible = false, overwritten = false;
ALTER TABLE lsif_nearest_uploads ALTER COLUMN ancestor_visible SET NOT NULL;
ALTER TABLE lsif_nearest_uploads ALTER COLUMN overwritten SET NOT NULL;

-- Mark all repositories as dirty so that we will refresh them
UPDATE lsif_dirty_repositories SET dirty_token = dirty_token + 1;

COMMIT;
