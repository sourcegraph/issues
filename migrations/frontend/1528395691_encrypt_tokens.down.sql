BEGIN;

DROP INDEX if exists secret_sourcetype_idx;
DROP INDEX if exists secret_key_idx;
DROP table if exists secrets;

COMMIT;
