BEGIN;

DROP INDEX IF EXISTS batch_spec_executions_rand_id;

ALTER TABLE IF EXISTS batch_spec_executions DROP COLUMN IF EXISTS rand_id;

COMMIT;
