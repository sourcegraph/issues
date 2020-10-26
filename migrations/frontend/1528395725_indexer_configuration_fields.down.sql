BEGIN;

DROP VIEW lsif_indexes_with_repository_name;

ALTER TABLE lsif_indexes
    DROP COLUMN docker_steps,
    DROP COLUMN root,
    DROP COLUMN indexer,
    DROP COLUMN indexer_args,
    DROP COLUMN outfile;

CREATE VIEW lsif_indexes_with_repository_name AS
    SELECT u.*, r.name as repository_name FROM lsif_indexes u
    JOIN repo r ON r.id = u.repository_id
    WHERE r.deleted_at IS NULL;

COMMIT;
