BEGIN;

ALTER TABLE changeset_specs
    ALTER COLUMN user_id DROP NOT NULL,
    DROP CONSTRAINT IF EXISTS changeset_specs_user_id_fkey,
    ADD CONSTRAINT changeset_specs_user_id_fkey
        FOREIGN KEY (user_id)
        REFERENCES users (id)
        ON DELETE SET NULL
        DEFERRABLE;

COMMIT;
