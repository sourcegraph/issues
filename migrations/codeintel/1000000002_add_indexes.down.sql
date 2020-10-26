BEGIN;

ALTER TABLE lsif_data_metadata DROP CONSTRAINT lsif_data_metadata_pkey;
ALTER TABLE lsif_data_documents DROP CONSTRAINT lsif_data_documents_pkey;
ALTER TABLE lsif_data_result_chunks DROP CONSTRAINT lsif_data_result_chunks_pkey;
ALTER TABLE lsif_data_definitions DROP CONSTRAINT lsif_data_definitions_pkey;
ALTER TABLE lsif_data_references DROP CONSTRAINT lsif_data_references_pkey;

COMMIT;
