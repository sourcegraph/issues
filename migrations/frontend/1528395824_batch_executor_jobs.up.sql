BEGIN;

CREATE TABLE IF NOT EXISTS batch_executor_jobs (
  id              BIGSERIAL PRIMARY KEY,
  state           text DEFAULT 'queued',
  failure_message text,
  started_at      timestamp with time zone,
  finished_at     timestamp with time zone,
  process_after   timestamp with time zone,
  num_resets      integer not null default 0,
  num_failures    integer not null default 0,
  execution_logs  json[],

  created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

  job JSONB NOT NULL
);

COMMIT;
