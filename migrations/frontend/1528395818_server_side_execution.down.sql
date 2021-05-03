BEGIN;

DROP VIEW IF EXISTS batch_job_workspace_with_state;
DROP TYPE IF EXISTS batch_job_workspace_state;
DROP TABLE IF EXISTS batch_job_workspace;
DROP TABLE IF EXISTS batch_job_spec;
DROP TABLE IF EXISTS batch_worker;

COMMIT;
