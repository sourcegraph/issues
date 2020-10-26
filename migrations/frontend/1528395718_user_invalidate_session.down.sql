BEGIN;

DROP FUNCTION IF EXISTS invalidate_session_for_userid_on_password_change() CASCADE;
ALTER TABLE IF EXISTS users DROP COLUMN IF EXISTS invalidated_sessions_at;

COMMIT;
