BEGIN;
ALTER TABLE insights_query_runner_jobs
    DROP COLUMN priority;

ALTER TABLE insights_query_runner_jobs
    DROP COLUMN cost;
COMMIT;
