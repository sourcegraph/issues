BEGIN;

ALTER TABLE IF EXISTS batch_spec_executions ADD COLUMN IF NOT EXISTS rand_id text NOT NULL;

CREATE INDEX batch_spec_executions_rand_id ON batch_spec_executions USING btree (rand_id);

COMMIT;
