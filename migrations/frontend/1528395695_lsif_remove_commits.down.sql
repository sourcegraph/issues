BEGIN;

CREATE TABLE lsif_commits (
    id SERIAL PRIMARY KEY,
    commit text NOT NULL,
    parent_commit text,
    repository_id integer NOT NULL,
    CONSTRAINT lsif_commits_commit_valid_chars CHECK ((commit ~ '^[a-z0-9]{40}$'::text)),
    CONSTRAINT lsif_commits_parent_commit_valid_chars CHECK ((parent_commit ~ '^[a-z0-9]{40}$'::text))
);

-- Drop dependent views
DROP VIEW lsif_dumps_with_repository_name;
DROP VIEW lsif_uploads_with_repository_name;
DROP VIEW lsif_dumps;

ALTER TABLE lsif_uploads ADD COLUMN visible_at_tip boolean NOT NULL DEFAULT false;

-- Recreate views
CREATE VIEW lsif_dumps AS SELECT u.*, u.finished_at as processed_at FROM lsif_uploads u WHERE state = 'completed';

CREATE VIEW lsif_dumps_with_repository_name AS
    SELECT u.*, r.name as repository_name FROM lsif_dumps u
    JOIN repo r ON r.id = u.repository_id
    WHERE r.deleted_at IS NULL;

CREATE VIEW lsif_uploads_with_repository_name AS
    SELECT u.*, r.name as repository_name FROM lsif_uploads u
    JOIN repo r ON r.id = u.repository_id
    WHERE r.deleted_at IS NULL;

COMMIT;
