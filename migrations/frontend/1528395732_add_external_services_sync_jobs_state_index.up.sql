BEGIN;

CREATE INDEX IF NOT EXISTS external_service_sync_jobs_state_idx ON external_service_sync_jobs (state);

COMMIT;
