BEGIN;

DROP FUNCTION IF EXISTS soft_delete_orphan_repo_by_external_service_repos() CASCADE;

CREATE FUNCTION soft_delete_orphan_repo_by_external_service_repos() RETURNS trigger
    LANGUAGE plpgsql
AS $$
BEGIN
    -- When an external service is soft or hard-deleted,
    -- performs a clean up to soft-delete orphan repositories.
    UPDATE
        repo
    SET
        name = soft_deleted_repository_name(name),
        deleted_at = transaction_timestamp()
    WHERE
        deleted_at IS NULL
        AND id = OLD.repo_id
        AND id NOT IN (
            SELECT DISTINCT(repo_id) FROM external_service_repos
        );

    RETURN OLD;
END;
$$;

CREATE TRIGGER trig_soft_delete_orphan_repo_by_external_service_repo
    AFTER DELETE ON external_service_repos
    FOR EACH ROW EXECUTE PROCEDURE soft_delete_orphan_repo_by_external_service_repos();

COMMIT;
