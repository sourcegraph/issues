BEGIN;

CREATE TABLE IF NOT EXISTS batch_worker (
    id BIGSERIAL PRIMARY KEY,
    -- name is only needed for an eventual admin UI.
    name CITEXT NOT NULL,
    -- If NULL, this worker can no longer connect, but is kept for historical
    -- reasons.
    token TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    last_seen_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL
);

CREATE TABLE IF NOT EXISTS batch_job_spec (
    id BIGSERIAL PRIMARY KEY,
    namespace_user_id INTEGER REFERENCES users (id) ON DELETE SET NULL,
    namespace_org_id INTEGER REFERENCES orgs (id) ON DELETE SET NULL,
    creator_id INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    raw_spec TEXT NOT NULL,
    spec JSONB NOT NULL,

    CONSTRAINT batch_job_spec_user_org_check
        CHECK (((namespace_user_id IS NULL) OR (namespace_org_id IS NULL)))
);

CREATE TABLE IF NOT EXISTS batch_job_workspace (
    id BIGSERIAL PRIMARY KEY,
    worker BIGINT REFERENCES batch_worker (id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,

    -- Input fields: anything not listed here is shared across all batch spec
    -- workspaces, such as the steps. (At least until Thorsten's done
    -- implementing conditional execution.)
    batch_job_spec_id BIGINT NOT NULL,
    repo INTEGER NOT NULL,
    path TEXT,
    only_fetch_workspace BOOLEAN NOT NULL,

    -- Output fields: what do we get back once execution is complete?
    error TEXT,
    diff BYTEA,
    outputs BYTEA
);

-- Probably won't use this in practice, but it helps reason about the implied
-- state of the workspace.
CREATE TYPE batch_job_workspace_state AS ENUM (
    'pending',
    'progress',
    'complete',
    'error'
);

CREATE OR REPLACE VIEW batch_job_workspace_with_state AS
    SELECT
        *,
        CASE
            WHEN worker IS NULL THEN 'pending'
            WHEN error IS NOT NULL then 'error'
            WHEN diff IS NOT NULL THEN 'complete'
            ELSE 'progress'
        END AS state
    FROM
        batch_job_workspace;

COMMIT;
