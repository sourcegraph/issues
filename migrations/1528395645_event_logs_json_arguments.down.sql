BEGIN;

-- Convert nullable jsonb back to non-nullable text (conversion is safe)
ALTER TABLE event_logs ALTER COLUMN argument TYPE text;
UPDATE event_logs SET argument = '' WHERE argument IS NULL;
ALTER TABLE event_logs ALTER COLUMN argument SET NOT NULL;

COMMIT;
