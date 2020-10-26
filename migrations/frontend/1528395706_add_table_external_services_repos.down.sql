BEGIN;

DROP TRIGGER IF EXISTS trig_delete_repo_ref_on_external_service_repos ON repo;
DROP TRIGGER IF EXISTS trig_delete_external_service_ref_on_external_service_repos ON external_services;
DROP TRIGGER IF EXISTS trig_read_only_repo_sources_column ON repo;

DROP FUNCTION IF EXISTS delete_repo_ref_on_external_service_repos();
DROP FUNCTION IF EXISTS delete_external_service_ref_on_external_service_repos();
DROP FUNCTION IF EXISTS make_repo_sources_column_read_only();

DROP TABLE IF EXISTS external_service_repos;

COMMIT;
