BEGIN;
ALTER TABLE insights_query_runner_jobs
    ADD COLUMN priority INT NOT NULL DEFAULT 1;

ALTER TABLE insights_query_runner_jobs
    ADD COLUMN cost INT NOT NULL DEFAULT 500;

COMMENT ON COLUMN insights_query_runner_jobs.priority IS 'Integer representing a category of priority for this query. Priority in this context is ambiguously defined for consumers to decide an interpretation.';
COMMENT ON COLUMN insights_query_runner_jobs.cost IS 'Integer representing a cost approximation of executing this search query.';

CREATE INDEX insights_query_runner_jobs_priority_idx on insights_query_runner_jobs(priority);
CREATE INDEX insights_query_runner_jobs_cost_idx on insights_query_runner_jobs(cost);

COMMIT;

