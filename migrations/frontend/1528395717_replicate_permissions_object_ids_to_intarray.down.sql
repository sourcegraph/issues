BEGIN;

ALTER TABLE repo_permissions DROP COLUMN IF EXISTS user_ids_ints;
ALTER TABLE user_permissions DROP COLUMN IF EXISTS object_ids_ints;
ALTER TABLE repo_pending_permissions DROP COLUMN IF EXISTS user_ids_ints;
ALTER TABLE user_pending_permissions DROP COLUMN IF EXISTS object_ids_ints;

DROP EXTENSION IF EXISTS intarray;

COMMIT;
