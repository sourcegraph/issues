BEGIN;

CREATE TABLE IF NOT EXISTS lsif_data_documentation_mappings (
    dump_id integer NOT NULL,
    path_id TEXT NOT NULL,
    result_id integer NOT NULL
);

ALTER TABLE lsif_data_documentation_mappings ADD PRIMARY KEY (dump_id, path_id);

CREATE UNIQUE INDEX lsif_data_documentation_mappings_inverse_unique_idx ON lsif_data_documentation_mappings(dump_id, result_id);

COMMENT ON TABLE lsif_data_documentation_mappings IS 'Maps documentation path IDs to their corresponding integral documentationResult vertex IDs, which are unique within a dump.';
COMMENT ON COLUMN lsif_data_documentation_mappings.dump_id IS 'The identifier of the associated dump in the lsif_uploads table (state=completed).';
COMMENT ON COLUMN lsif_data_documentation_mappings.path_id IS 'The documentation page path ID, see see GraphQL codeintel.schema:documentationPage for what this is.';
COMMENT ON COLUMN lsif_data_documentation_mappings.result_id IS 'The documentationResult vertex ID.';

COMMIT;
