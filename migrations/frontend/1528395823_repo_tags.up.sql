BEGIN;

CREATE TABLE IF NOT EXISTS repo_tags (
    id BIGSERIAL PRIMARY KEY,
    repo_id INTEGER NOT NULL REFERENCES repo (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    tag VARCHAR(100) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT repo_tag UNIQUE (repo_id, tag, deleted_at)
);

CREATE INDEX IF NOT EXISTS
    repo_tags_deleted_at_idx
ON
    repo_tags (deleted_at);

CREATE INDEX IF NOT EXISTS
    repo_tags_tag_trgm
ON
    repo_tags
USING
    gin (lower((tag)::text) gin_trgm_ops);

COMMIT;
