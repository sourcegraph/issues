BEGIN;

ALTER TABLE lsif_data_documentation_mappings DROP COLUMN file_path;

COMMIT;
