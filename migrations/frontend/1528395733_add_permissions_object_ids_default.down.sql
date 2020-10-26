BEGIN;

ALTER TABLE repo_permissions ALTER COLUMN user_ids DROP DEFAULT;
ALTER TABLE user_permissions ALTER COLUMN object_ids DROP DEFAULT;
ALTER TABLE repo_pending_permissions ALTER COLUMN user_ids DROP DEFAULT;
ALTER TABLE user_pending_permissions ALTER COLUMN object_ids DROP DEFAULT;

COMMIT;
