BEGIN;

ALTER TABLE changesets ADD COLUMN IF NOT EXISTS created_by_campaign boolean DEFAULT false NOT NULL;

UPDATE changesets SET created_by_campaign = true WHERE owned_by_campaign_id IS NOT NULL AND publication_state = 'PUBLISHED';

COMMIT;
