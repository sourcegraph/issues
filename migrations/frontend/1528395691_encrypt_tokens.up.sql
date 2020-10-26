BEGIN;

CREATE TABLE secrets (
    id BIGSERIAL PRIMARY KEY,
    source_type varchar(50),
    source_id bigint,
    key_name varchar(100),
    value text NOT NULL
);

-- A source_type/source_id combination should always be unique, otherwise we
-- can have duplicate token entries
CREATE UNIQUE INDEX secret_sourcetype_idx ON secrets USING btree (source_type, source_id);

-- PostgreSQL treats NULL as distinct values
CREATE UNIQUE INDEX secret_key_idx ON secrets USING btree (key_name);

COMMIT;
