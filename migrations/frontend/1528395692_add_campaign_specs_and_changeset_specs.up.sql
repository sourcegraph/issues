BEGIN;

CREATE TABLE campaign_specs (
    id bigserial NOT NULL PRIMARY KEY,
    rand_id text NOT NULL,

    raw_spec text NOT NULL,
    spec jsonb DEFAULT '{}'::jsonb NOT NULL,

    namespace_user_id integer,
    namespace_org_id integer,

    user_id integer NOT NULL,

    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,

    CONSTRAINT campaign_specs_has_1_namespace CHECK (((namespace_user_id IS NULL) <> (namespace_org_id IS NULL))),

    FOREIGN KEY (user_id) REFERENCES users(id) DEFERRABLE
);

CREATE INDEX IF NOT EXISTS campaign_specs_rand_id ON campaign_specs(rand_id);

ALTER TABLE campaigns
  ADD COLUMN IF NOT EXISTS campaign_spec_id bigint REFERENCES campaign_specs(id) DEFERRABLE;

CREATE TABLE changeset_specs (
  id bigserial NOT NULL PRIMARY KEY,
  rand_id text NOT NULL,

  raw_spec text NOT NULL,
  spec jsonb DEFAULT '{}'::jsonb NOT NULL,

  campaign_spec_id bigint,
  repo_id integer NOT NULL,

  user_id integer NOT NULL,

  diff_stat_added integer,
  diff_stat_changed integer,
  diff_stat_deleted integer,

  created_at timestamp with time zone DEFAULT now() NOT NULL,
  updated_at timestamp with time zone DEFAULT now() NOT NULL,

  FOREIGN KEY (campaign_spec_id) REFERENCES campaign_specs(id) DEFERRABLE,
  FOREIGN KEY (repo_id)          REFERENCES repo(id)           DEFERRABLE,
  FOREIGN KEY (user_id)          REFERENCES users(id)          DEFERRABLE
);

CREATE INDEX IF NOT EXISTS changeset_specs_rand_id ON changeset_specs(rand_id);

ALTER TABLE changesets
  ADD COLUMN IF NOT EXISTS changeset_spec_id bigint REFERENCES changeset_specs(id) DEFERRABLE;

COMMIT;
