BEGIN;

CREATE SEQUENCE IF NOT EXISTS external_service_sync_jobs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

CREATE TABLE IF NOT EXISTS external_service_sync_jobs (
    -- Columns required by workerutil.Store
    id integer NOT NULL DEFAULT nextval('external_service_sync_jobs_id_seq'::regclass),
    state text NOT NULL DEFAULT 'queued'::text,
    failure_message text,
    started_at timestamp with time zone,
    finished_at timestamp with time zone,
    process_after timestamp with time zone,
    num_resets integer not null DEFAULT 0,
    -- Extra columns
    external_service_id bigint,
    -- Constraints
    CONSTRAINT external_services_id_fk
    FOREIGN KEY(external_service_id)
    REFERENCES external_services(id)
);

CREATE OR REPLACE VIEW external_service_sync_jobs_with_next_sync_at AS
    SELECT j.id,
            j.state,
            j.failure_message,
            j.started_at,
            j.finished_at,
            j.process_after,
            j.num_resets,
            j.external_service_id,
            e.next_sync_at
    FROM
    external_services e join external_service_sync_jobs j on e.id = j.external_service_id;

-- NOTE: No index on the state column was added as we expect the size of this table to stay fairly small

COMMIT;
