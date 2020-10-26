BEGIN;

-- Default to now, since we can't have null times
ALTER TABLE IF EXISTS users ADD COLUMN IF NOT EXISTS invalidated_sessions_at timestamp with time zone DEFAULT now() NOT NULL;
-- Update the invalidated sessions at to be when the user was created
UPDATE users SET invalidated_sessions_at = created_at;

-- Create a procedure that invalidates sessions for the user that can be used for our trigger
-- Invalidates if the password is updated
-- For the reasoning behind adding one second, see security issue #93
CREATE OR REPLACE FUNCTION invalidate_session_for_userid_on_password_change() RETURNS trigger
LANGUAGE plpgsql
    AS $$
    BEGIN
        IF OLD.passwd != NEW.passwd THEN
            NEW.invalidated_sessions_at = now() + (1 * interval '1 second');
            RETURN NEW;
        END IF;
    RETURN NEW;
    END;
$$;

-- Need to drop and create, since we can't create if not exstis
DROP TRIGGER IF EXISTS trig_invalidate_session_on_password_change ON users;
-- Create a trigger to to invalidate sessions if the user's password is ever changed
CREATE TRIGGER trig_invalidate_session_on_password_change
    BEFORE UPDATE OF passwd ON users 
    FOR EACH ROW EXECUTE PROCEDURE invalidate_session_for_userid_on_password_change();

COMMIT;
