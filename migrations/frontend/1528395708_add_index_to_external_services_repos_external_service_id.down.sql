-- Note: DROP INDEX CONCURRENTLY cannot run inside a transaction block

DROP INDEX CONCURRENTLY IF EXISTS external_service_repos_external_service_id;

