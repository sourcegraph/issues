BEGIN;

DROP VIEW lsif_indexes_with_repository_name;

ALTER TABLE lsif_indexes
    ADD COLUMN docker_steps jsonb[],  -- root, image, commands
    ADD COLUMN root text,
    ADD COLUMN indexer text,
    ADD COLUMN indexer_args text[],
    ADD COLUMN outfile text;

UPDATE lsif_indexes SET
    docker_steps = '{}',
    root = '',
    indexer = 'sourcegraph/lsif-go:latest',
    indexer_args = '{}'::text[],
    outfile = '';

ALTER TABLE lsif_indexes
    ALTER COLUMN root SET NOT NULL,
    ALTER COLUMN indexer SET NOT NULL,
    ALTER COLUMN indexer_args SET NOT NULL,
    ALTER COLUMN outfile SET NOT NULL,
    ALTER COLUMN docker_steps SET NOT NULL;

CREATE VIEW lsif_indexes_with_repository_name AS
    SELECT u.*, r.name as repository_name FROM lsif_indexes u
    JOIN repo r ON r.id = u.repository_id
    WHERE r.deleted_at IS NULL;

COMMIT;
