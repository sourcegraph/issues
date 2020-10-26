BEGIN;

ALTER TABLE lsif_indexes ADD COLUMN log_contents TEXT;

COMMIT;
