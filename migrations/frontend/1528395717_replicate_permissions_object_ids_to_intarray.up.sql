BEGIN;

CREATE EXTENSION IF NOT EXISTS intarray;

ALTER TABLE repo_permissions ADD COLUMN user_ids_ints INT[] NOT NULL DEFAULT '{}';
ALTER TABLE user_permissions ADD COLUMN object_ids_ints INT[] NOT NULL DEFAULT '{}';
ALTER TABLE repo_pending_permissions ADD COLUMN user_ids_ints INT[] NOT NULL DEFAULT '{}';
ALTER TABLE user_pending_permissions ADD COLUMN object_ids_ints INT[] NOT NULL DEFAULT '{}';

COMMIT;
