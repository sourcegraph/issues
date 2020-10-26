BEGIN;

DROP FUNCTION IF EXISTS soft_deleted_repository_name(text);

CREATE FUNCTION soft_deleted_repository_name(name TEXT) RETURNS TEXT AS $$
BEGIN
    RETURN 'DELETED-' || extract(epoch from transaction_timestamp()) || '-' || name;
END;
$$ LANGUAGE plpgsql VOLATILE STRICT;

COMMIT;
