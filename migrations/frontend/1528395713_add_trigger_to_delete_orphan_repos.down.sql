BEGIN;

DROP FUNCTION IF EXISTS soft_delete_orphan_repo_by_external_service_repos() CASCADE;

COMMIT;
