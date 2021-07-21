
BEGIN;

CREATE TABLE IF NOT EXISTS codeintel_dependency_repo_adding_jobs (
    id serial PRIMARY KEY,
    state text DEFAULT 'queued'::text NOT NULL,
    failure_message text,
    started_at timestamp with time zone,
    finished_at timestamp with time zone,
    process_after timestamp with time zone,
    last_heartbeat_at timestamp with time zone,
    num_resets integer DEFAULT 0 NOT NULL,
    num_failures integer DEFAULT 0 NOT NULL,
    worker_hostname text NOT NULL DEFAULT '',
    execution_logs json[],
    upload_id integer REFERENCES lsif_uploads(id) ON DELETE CASCADE
);

COMMENT ON TABLE codeintel_dependency_repo_adding_jobs IS 'Tracks jobs that scan imports of indexes to add repositories that are referenced in LSIF uploads.';
COMMENT ON COLUMN codeintel_dependency_repo_adding_jobs.upload_id IS 'The identifier of the triggering upload record.';

CREATE TABLE IF NOT EXISTS codeintel_dependency_repos (
    identifier text NOT NULL,
    version text NOT NULL,
    scheme text NOT NULL,
    PRIMARY KEY (identifier, version, scheme)
);

COMMIT;
