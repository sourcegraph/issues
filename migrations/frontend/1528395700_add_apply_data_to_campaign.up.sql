BEGIN;

ALTER TABLE campaigns RENAME COLUMN author_id TO initial_applier_id;

ALTER TABLE campaigns ADD COLUMN IF NOT EXISTS last_applier_id bigint REFERENCES users(id) DEFERRABLE;
ALTER TABLE campaigns ADD COLUMN IF NOT EXISTS last_applied_at timestamp with time zone;

COMMIT;
