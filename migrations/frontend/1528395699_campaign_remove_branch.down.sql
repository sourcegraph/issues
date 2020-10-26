BEGIN;

ALTER TABLE campaigns ADD COLUMN branch text;

COMMIT;
