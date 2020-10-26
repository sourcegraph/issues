-- Note: CREATE INDEX CONCURRENTLY cannot run inside a transaction block

CREATE INDEX CONCURRENTLY IF NOT EXISTS repo_created_at ON repo(created_at);
