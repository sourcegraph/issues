BEGIN;

ALTER TABLE changeset_specs
    DROP CONSTRAINT IF EXISTS changeset_specs_user_id_fkey,
    ADD CONSTRAINT changeset_specs_user_id_fkey
        FOREIGN KEY (user_id)
        REFERENCES users (id)
        DEFERRABLE,
    ALTER COLUMN user_id SET NOT NULL;

COMMIT;
