
BEGIN;

alter table if exists batch_spec_executions add column if not exists cancel bool NOT NULL default false;

COMMIT;
