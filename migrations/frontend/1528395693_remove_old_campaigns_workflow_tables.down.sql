BEGIN;

CREATE TABLE patch_sets (
    id bigserial NOT NULL PRIMARY KEY,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    user_id integer NOT NULL
);
ALTER TABLE patch_sets ADD CONSTRAINT patch_sets_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) DEFERRABLE;

ALTER TABLE campaigns
  ADD COLUMN IF NOT EXISTS patch_set_id bigint REFERENCES patch_sets(id) DEFERRABLE;

CREATE TABLE patches (
    id bigserial NOT NULL PRIMARY KEY,
    patch_set_id bigint NOT NULL,
    repo_id bigint NOT NULL,
    rev text NOT NULL,
    diff text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    base_ref text NOT NULL,
    diff_stat_added integer,
    diff_stat_changed integer,
    diff_stat_deleted integer,
    CONSTRAINT patches_base_ref_check CHECK ((base_ref <> ''::text)),

    FOREIGN KEY (patch_set_id) REFERENCES patch_sets(id) ON DELETE CASCADE DEFERRABLE,
    FOREIGN KEY (repo_id) REFERENCES repo(id) ON DELETE CASCADE DEFERRABLE
);

ALTER TABLE patches ADD CONSTRAINT patches_patch_set_repo_rev_unique UNIQUE (patch_set_id, repo_id, rev) DEFERRABLE;

CREATE TABLE changeset_jobs (
    id bigserial NOT NULL PRIMARY KEY,
    campaign_id bigint NOT NULL,
    patch_id bigint NOT NULL,
    changeset_id bigint,
    error text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    started_at timestamp with time zone,
    finished_at timestamp with time zone,
    branch text,

    FOREIGN KEY (campaign_id) REFERENCES campaigns(id) DEFERRABLE,
    FOREIGN KEY (patch_id) REFERENCES patches(id) DEFERRABLE,
    FOREIGN KEY (changeset_id) REFERENCES changesets(id) DEFERRABLE
);
ALTER TABLE changeset_jobs ADD CONSTRAINT changeset_jobs_unique UNIQUE (campaign_id, patch_id);

CREATE INDEX changeset_jobs_campaign_job_id ON changeset_jobs USING btree (patch_id);
CREATE INDEX changeset_jobs_error_not_null ON changeset_jobs USING btree (((error IS NOT NULL)));
CREATE INDEX changeset_jobs_finished_at ON changeset_jobs USING btree (finished_at);
CREATE INDEX changeset_jobs_started_at ON changeset_jobs USING btree (started_at);

COMMIT;
