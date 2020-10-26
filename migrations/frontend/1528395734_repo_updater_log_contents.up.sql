BEGIN;

DROP VIEW external_service_sync_jobs_with_next_sync_at;

ALTER TABLE external_service_sync_jobs ADD COLUMN log_contents text;

CREATE VIEW external_service_sync_jobs_with_next_sync_at AS SELECT
    j.id,
    j.state,
    j.failure_message,
    j.started_at,
    j.finished_at,
    j.process_after,
    j.num_resets,
    j.num_failures,
    j.log_contents,
    j.external_service_id,
    e.next_sync_at
FROM external_services e JOIN external_service_sync_jobs j ON e.id = j.external_service_id;

ALTER TABLE changesets ADD COLUMN log_contents text;

COMMIT;
