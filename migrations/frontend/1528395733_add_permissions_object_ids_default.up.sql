BEGIN;

ALTER TABLE repo_permissions ALTER COLUMN user_ids SET DEFAULT '\x';
ALTER TABLE user_permissions ALTER COLUMN object_ids SET DEFAULT '\x';
ALTER TABLE repo_pending_permissions ALTER COLUMN user_ids SET DEFAULT '\x';
ALTER TABLE user_pending_permissions ALTER COLUMN object_ids SET DEFAULT '\x';

COMMIT;
