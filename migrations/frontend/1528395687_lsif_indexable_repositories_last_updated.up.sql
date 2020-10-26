BEGIN;

ALTER TABLE lsif_indexable_repositories ADD COLUMN last_updated_at timestamp with time zone DEFAULT now() NOT NULL;
UPDATE lsif_indexable_repositories SET last_updated_at = NOW();
ALTER TABLE lsif_indexable_repositories ALTER COLUMN last_updated_at SET NOT NULL;

COMMIT;
