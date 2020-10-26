BEGIN;

DROP FUNCTION IF EXISTS soft_delete_user_reference_on_external_service();

CREATE FUNCTION soft_delete_user_reference_on_external_service() RETURNS trigger
    LANGUAGE plpgsql
AS $$
BEGIN
    -- If a user is soft-deleted, delete every row that references that user
    IF (OLD.deleted_at IS NULL AND NEW.deleted_at IS NOT NULL) THEN
        UPDATE external_services
        SET deleted_at = NOW()
        WHERE namespace_user_id = OLD.id;
    END IF;

    RETURN OLD;
END;
$$;

CREATE TRIGGER trig_soft_delete_user_reference_on_external_service
    AFTER UPDATE OF deleted_at ON users
    FOR EACH ROW EXECUTE PROCEDURE soft_delete_user_reference_on_external_service();

COMMIT;
