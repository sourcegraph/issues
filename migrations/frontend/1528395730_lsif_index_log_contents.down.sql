BEGIN;

ALTER TABLE lsif_indexes DROP COLUMN log_contents;

COMMIT;
