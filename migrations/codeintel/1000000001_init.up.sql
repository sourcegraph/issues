BEGIN;

CREATE TABLE IF NOT EXISTS lsif_data_metadata (dump_id integer NOT NULL, num_result_chunks integer);
CREATE TABLE IF NOT EXISTS lsif_data_documents (dump_id integer NOT NULL, path text NOT NULL, data bytea);
CREATE TABLE IF NOT EXISTS lsif_data_result_chunks (dump_id integer NOT NULL, idx integer NOT NULL, data bytea);
CREATE TABLE IF NOT EXISTS lsif_data_definitions (dump_id integer NOT NULL, scheme text NOT NULL, identifier text NOT NULL, data bytea);
CREATE TABLE IF NOT EXISTS lsif_data_references (dump_id integer NOT NULL, scheme text NOT NULL, identifier text NOT NULL, data bytea);

COMMIT;
