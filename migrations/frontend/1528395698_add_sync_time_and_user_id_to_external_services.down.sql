BEGIN;

ALTER TABLE external_services DROP COLUMN IF EXISTS last_sync_at;
ALTER TABLE external_services DROP COLUMN IF EXISTS next_sync_at;
ALTER TABLE external_services DROP COLUMN IF EXISTS namespace_user_id;

ALTER TABLE IF EXISTS ONLY external_services DROP CONSTRAINT IF EXISTS external_services_namepspace_user_id_fkey;

COMMIT;
