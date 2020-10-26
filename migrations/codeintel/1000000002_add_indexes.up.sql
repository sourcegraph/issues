BEGIN;

ALTER TABLE lsif_data_metadata ADD PRIMARY KEY (dump_id);
ALTER TABLE lsif_data_documents ADD PRIMARY KEY (dump_id, path);
ALTER TABLE lsif_data_result_chunks ADD PRIMARY KEY (dump_id, idx);
ALTER TABLE lsif_data_definitions ADD PRIMARY KEY (dump_id, scheme, identifier);
ALTER TABLE lsif_data_references ADD PRIMARY KEY (dump_id, scheme, identifier);

COMMIT;
