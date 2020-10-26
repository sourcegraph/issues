BEGIN;

CREATE TABLE lsif_nearest_uploads (
    repository_id integer NOT NULL,
    "commit" text NOT NULL,
    upload_id integer NOT NULL,
    distance integer NOT NULL
);

CREATE TABLE lsif_uploads_visible_at_tip (
    repository_id integer NOT NULL,
    upload_id integer NOT NULL
);

CREATE TABLE lsif_dirty_repositories (
    repository_id integer PRIMARY KEY,
    dirty_token int NOT NULL,
    update_token int NOT NULL
);

CREATE INDEX lsif_nearest_uploads_repository_id_commit ON lsif_nearest_uploads(repository_id, "commit");
CREATE INDEX lsif_uploads_visible_at_tip_repository_id ON lsif_uploads_visible_at_tip(repository_id);

COMMIT;
