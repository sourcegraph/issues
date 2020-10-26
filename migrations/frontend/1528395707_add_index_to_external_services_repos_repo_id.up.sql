-- Note: CREATE INDEX CONCURRENTLY cannot run inside a transaction block

CREATE INDEX CONCURRENTLY IF NOT EXISTS external_service_repos_repo_id ON external_service_repos(repo_id);

