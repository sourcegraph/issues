BEGIN;

CREATE TABLE lsif_index_configuration (
    id bigserial NOT NULL PRIMARY KEY,
    repository_id integer UNIQUE NOT NULL REFERENCES repo(id) ON DELETE CASCADE,
    data bytea NOT NULL
);

COMMIT;
