BEGIN;

CREATE TABLE IF NOT EXISTS pending_batch_specs (
  id              BIGSERIAL PRIMARY KEY,
  state           text DEFAULT 'queued',
  failure_message text,
  started_at      timestamp with time zone,
  finished_at     timestamp with time zone,
  process_after   timestamp with time zone,
  num_resets      integer not null default 0,
  num_failures    integer not null default 0,

  created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  creator_user_id INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  spec TEXT NOT NULL
);

COMMIT;
