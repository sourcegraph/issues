BEGIN;

ALTER TABLE IF EXISTS ONLY user_external_accounts DROP CONSTRAINT IF EXISTS user_external_accounts_user_id_fkey;
ALTER TABLE IF EXISTS ONLY user_emails DROP CONSTRAINT IF EXISTS user_emails_user_id_fkey;
ALTER TABLE IF EXISTS ONLY survey_responses DROP CONSTRAINT IF EXISTS survey_responses_user_id_fkey;
ALTER TABLE IF EXISTS ONLY settings DROP CONSTRAINT IF EXISTS settings_user_id_fkey;
ALTER TABLE IF EXISTS ONLY settings DROP CONSTRAINT IF EXISTS settings_references_orgs;
ALTER TABLE IF EXISTS ONLY settings DROP CONSTRAINT IF EXISTS settings_author_user_id_fkey;
ALTER TABLE IF EXISTS ONLY saved_searches DROP CONSTRAINT IF EXISTS saved_searches_user_id_fkey;
ALTER TABLE IF EXISTS ONLY saved_searches DROP CONSTRAINT IF EXISTS saved_searches_org_id_fkey;
ALTER TABLE IF EXISTS ONLY registry_extensions DROP CONSTRAINT IF EXISTS registry_extensions_publisher_user_id_fkey;
ALTER TABLE IF EXISTS ONLY registry_extensions DROP CONSTRAINT IF EXISTS registry_extensions_publisher_org_id_fkey;
ALTER TABLE IF EXISTS ONLY registry_extension_releases DROP CONSTRAINT IF EXISTS registry_extension_releases_registry_extension_id_fkey;
ALTER TABLE IF EXISTS ONLY registry_extension_releases DROP CONSTRAINT IF EXISTS registry_extension_releases_creator_user_id_fkey;
ALTER TABLE IF EXISTS ONLY product_subscriptions DROP CONSTRAINT IF EXISTS product_subscriptions_user_id_fkey;
ALTER TABLE IF EXISTS ONLY product_licenses DROP CONSTRAINT IF EXISTS product_licenses_product_subscription_id_fkey;
ALTER TABLE IF EXISTS ONLY org_members DROP CONSTRAINT IF EXISTS org_members_user_id_fkey;
ALTER TABLE IF EXISTS ONLY org_members DROP CONSTRAINT IF EXISTS org_members_references_orgs;
ALTER TABLE IF EXISTS ONLY org_invitations DROP CONSTRAINT IF EXISTS org_invitations_sender_user_id_fkey;
ALTER TABLE IF EXISTS ONLY org_invitations DROP CONSTRAINT IF EXISTS org_invitations_recipient_user_id_fkey;
ALTER TABLE IF EXISTS ONLY org_invitations DROP CONSTRAINT IF EXISTS org_invitations_org_id_fkey;
ALTER TABLE IF EXISTS ONLY names DROP CONSTRAINT IF EXISTS names_user_id_fkey;
ALTER TABLE IF EXISTS ONLY names DROP CONSTRAINT IF EXISTS names_org_id_fkey;
ALTER TABLE IF EXISTS ONLY lsif_references DROP CONSTRAINT IF EXISTS lsif_references_dump_id_fkey;
ALTER TABLE IF EXISTS ONLY lsif_packages DROP CONSTRAINT IF EXISTS lsif_packages_dump_id_fkey;
ALTER TABLE IF EXISTS ONLY discussion_threads_target_repo DROP CONSTRAINT IF EXISTS discussion_threads_target_repo_thread_id_fkey;
ALTER TABLE IF EXISTS ONLY discussion_threads_target_repo DROP CONSTRAINT IF EXISTS discussion_threads_target_repo_repo_id_fkey;
ALTER TABLE IF EXISTS ONLY discussion_threads DROP CONSTRAINT IF EXISTS discussion_threads_target_repo_id_fk;
ALTER TABLE IF EXISTS ONLY discussion_threads DROP CONSTRAINT IF EXISTS discussion_threads_author_user_id_fkey;
ALTER TABLE IF EXISTS ONLY discussion_mail_reply_tokens DROP CONSTRAINT IF EXISTS discussion_mail_reply_tokens_user_id_fkey;
ALTER TABLE IF EXISTS ONLY discussion_mail_reply_tokens DROP CONSTRAINT IF EXISTS discussion_mail_reply_tokens_thread_id_fkey;
ALTER TABLE IF EXISTS ONLY discussion_comments DROP CONSTRAINT IF EXISTS discussion_comments_thread_id_fkey;
ALTER TABLE IF EXISTS ONLY discussion_comments DROP CONSTRAINT IF EXISTS discussion_comments_author_user_id_fkey;
ALTER TABLE IF EXISTS ONLY default_repos DROP CONSTRAINT IF EXISTS default_repos_repo_id_fkey;
ALTER TABLE IF EXISTS ONLY changesets DROP CONSTRAINT IF EXISTS changesets_repo_id_fkey;
ALTER TABLE IF EXISTS ONLY changeset_jobs DROP CONSTRAINT IF EXISTS changeset_jobs_changeset_id_fkey;
ALTER TABLE IF EXISTS ONLY changeset_jobs DROP CONSTRAINT IF EXISTS changeset_jobs_campaign_job_id_fkey;
ALTER TABLE IF EXISTS ONLY changeset_jobs DROP CONSTRAINT IF EXISTS changeset_jobs_campaign_id_fkey;
ALTER TABLE IF EXISTS ONLY changeset_events DROP CONSTRAINT IF EXISTS changeset_events_changeset_id_fkey;
ALTER TABLE IF EXISTS ONLY campaigns DROP CONSTRAINT IF EXISTS campaigns_namespace_user_id_fkey;
ALTER TABLE IF EXISTS ONLY campaigns DROP CONSTRAINT IF EXISTS campaigns_namespace_org_id_fkey;
ALTER TABLE IF EXISTS ONLY campaigns DROP CONSTRAINT IF EXISTS campaigns_campaign_plan_id_fkey;
ALTER TABLE IF EXISTS ONLY campaigns DROP CONSTRAINT IF EXISTS campaigns_author_id_fkey;
ALTER TABLE IF EXISTS ONLY patch_sets DROP CONSTRAINT IF EXISTS campaign_plans_user_id_fkey;
ALTER TABLE IF EXISTS ONLY patches DROP CONSTRAINT IF EXISTS campaign_jobs_repo_id_fkey;
ALTER TABLE IF EXISTS ONLY patches DROP CONSTRAINT IF EXISTS campaign_jobs_campaign_plan_id_fkey;
ALTER TABLE IF EXISTS ONLY access_tokens DROP CONSTRAINT IF EXISTS access_tokens_subject_user_id_fkey;
ALTER TABLE IF EXISTS ONLY access_tokens DROP CONSTRAINT IF EXISTS access_tokens_creator_user_id_fkey;
DROP TRIGGER IF EXISTS trig_delete_changeset_reference_on_campaigns ON changesets;
DROP TRIGGER IF EXISTS trig_delete_campaign_reference_on_changesets ON campaigns;
DROP INDEX IF EXISTS users_username;
DROP INDEX IF EXISTS users_billing_customer_id;
DROP INDEX IF EXISTS user_external_accounts_user_id;
DROP INDEX IF EXISTS user_external_accounts_account;
DROP INDEX IF EXISTS saved_queries_query_unique;
DROP INDEX IF EXISTS repo_uri_idx;
DROP INDEX IF EXISTS repo_sources_gin_idx;
DROP INDEX IF EXISTS repo_private;
DROP INDEX IF EXISTS repo_name_trgm;
DROP INDEX IF EXISTS repo_metadata_gin_idx;
DROP INDEX IF EXISTS repo_fork;
DROP INDEX IF EXISTS repo_external_unique_idx;
DROP INDEX IF EXISTS repo_archived;
DROP INDEX IF EXISTS registry_extensions_uuid;
DROP INDEX IF EXISTS registry_extensions_publisher_name;
DROP INDEX IF EXISTS registry_extension_releases_version;
DROP INDEX IF EXISTS registry_extension_releases_registry_extension_id;
DROP INDEX IF EXISTS orgs_name;
DROP INDEX IF EXISTS org_invitations_singleflight;
DROP INDEX IF EXISTS org_invitations_recipient_user_id;
DROP INDEX IF EXISTS org_invitations_org_id;
DROP INDEX IF EXISTS lsif_uploads_visible_repository_id_commit;
DROP INDEX IF EXISTS lsif_uploads_uploaded_at;
DROP INDEX IF EXISTS lsif_uploads_state;
DROP INDEX IF EXISTS lsif_uploads_repository_id_commit_root_indexer;
DROP INDEX IF EXISTS lsif_references_package;
DROP INDEX IF EXISTS lsif_packages_scheme_name_version;
DROP INDEX IF EXISTS lsif_commits_repository_id_parent_commit;
DROP INDEX IF EXISTS lsif_commits_repository_id_commit_parent_commit_unique;
DROP INDEX IF EXISTS event_logs_user_id;
DROP INDEX IF EXISTS event_logs_timestamp_at_utc;
DROP INDEX IF EXISTS event_logs_timestamp;
DROP INDEX IF EXISTS event_logs_source;
DROP INDEX IF EXISTS event_logs_name;
DROP INDEX IF EXISTS event_logs_anonymous_user_id;
DROP INDEX IF EXISTS discussion_threads_target_repo_repo_id_path_idx;
DROP INDEX IF EXISTS discussion_threads_author_user_id_idx;
DROP INDEX IF EXISTS discussion_mail_reply_tokens_user_id_thread_id_idx;
DROP INDEX IF EXISTS discussion_comments_thread_id_idx;
DROP INDEX IF EXISTS discussion_comments_reports_array_length_idx;
DROP INDEX IF EXISTS discussion_comments_author_user_id_idx;
DROP INDEX IF EXISTS critical_and_site_config_unique;
DROP INDEX IF EXISTS changeset_jobs_started_at;
DROP INDEX IF EXISTS changeset_jobs_finished_at;
DROP INDEX IF EXISTS changeset_jobs_error_not_null;
DROP INDEX IF EXISTS changeset_jobs_campaign_job_id;
DROP INDEX IF EXISTS campaigns_namespace_user_id;
DROP INDEX IF EXISTS campaigns_namespace_org_id;
DROP INDEX IF EXISTS campaigns_changeset_ids_gin_idx;
DROP INDEX IF EXISTS access_tokens_lookup;
ALTER TABLE IF EXISTS ONLY versions DROP CONSTRAINT IF EXISTS versions_pkey;
ALTER TABLE IF EXISTS ONLY users DROP CONSTRAINT IF EXISTS users_pkey;
ALTER TABLE IF EXISTS ONLY user_permissions DROP CONSTRAINT IF EXISTS user_permissions_perm_object_unique;
ALTER TABLE IF EXISTS ONLY user_pending_permissions DROP CONSTRAINT IF EXISTS user_pending_permissions_service_perm_object_unique;
ALTER TABLE IF EXISTS ONLY user_external_accounts DROP CONSTRAINT IF EXISTS user_external_accounts_pkey;
ALTER TABLE IF EXISTS ONLY user_emails DROP CONSTRAINT IF EXISTS user_emails_unique_verified_email;
ALTER TABLE IF EXISTS ONLY user_emails DROP CONSTRAINT IF EXISTS user_emails_no_duplicates_per_user;
ALTER TABLE IF EXISTS ONLY survey_responses DROP CONSTRAINT IF EXISTS survey_responses_pkey;
ALTER TABLE IF EXISTS ONLY settings DROP CONSTRAINT IF EXISTS settings_pkey;
ALTER TABLE IF EXISTS ONLY schema_migrations DROP CONSTRAINT IF EXISTS schema_migrations_pkey;
ALTER TABLE IF EXISTS ONLY saved_searches DROP CONSTRAINT IF EXISTS saved_searches_pkey;
ALTER TABLE IF EXISTS ONLY repo DROP CONSTRAINT IF EXISTS repo_pkey;
ALTER TABLE IF EXISTS ONLY repo_permissions DROP CONSTRAINT IF EXISTS repo_permissions_perm_unique;
ALTER TABLE IF EXISTS ONLY repo_pending_permissions DROP CONSTRAINT IF EXISTS repo_pending_permissions_perm_unique;
ALTER TABLE IF EXISTS ONLY repo DROP CONSTRAINT IF EXISTS repo_name_unique;
ALTER TABLE IF EXISTS ONLY registry_extensions DROP CONSTRAINT IF EXISTS registry_extensions_pkey;
ALTER TABLE IF EXISTS ONLY registry_extension_releases DROP CONSTRAINT IF EXISTS registry_extension_releases_pkey;
ALTER TABLE IF EXISTS ONLY product_subscriptions DROP CONSTRAINT IF EXISTS product_subscriptions_pkey;
ALTER TABLE IF EXISTS ONLY product_licenses DROP CONSTRAINT IF EXISTS product_licenses_pkey;
ALTER TABLE IF EXISTS ONLY phabricator_repos DROP CONSTRAINT IF EXISTS phabricator_repos_repo_name_key;
ALTER TABLE IF EXISTS ONLY phabricator_repos DROP CONSTRAINT IF EXISTS phabricator_repos_pkey;
ALTER TABLE IF EXISTS ONLY orgs DROP CONSTRAINT IF EXISTS orgs_pkey;
ALTER TABLE IF EXISTS ONLY org_members DROP CONSTRAINT IF EXISTS org_members_pkey;
ALTER TABLE IF EXISTS ONLY org_members DROP CONSTRAINT IF EXISTS org_members_org_id_user_id_key;
ALTER TABLE IF EXISTS ONLY org_invitations DROP CONSTRAINT IF EXISTS org_invitations_pkey;
ALTER TABLE IF EXISTS ONLY names DROP CONSTRAINT IF EXISTS names_pkey;
ALTER TABLE IF EXISTS ONLY lsif_uploads DROP CONSTRAINT IF EXISTS lsif_uploads_pkey;
ALTER TABLE IF EXISTS ONLY lsif_references DROP CONSTRAINT IF EXISTS lsif_references_pkey;
ALTER TABLE IF EXISTS ONLY lsif_packages DROP CONSTRAINT IF EXISTS lsif_packages_pkey;
ALTER TABLE IF EXISTS ONLY lsif_indexes DROP CONSTRAINT IF EXISTS lsif_indexes_pkey;
ALTER TABLE IF EXISTS ONLY lsif_indexable_repositories DROP CONSTRAINT IF EXISTS lsif_indexable_repositories_repository_id_key;
ALTER TABLE IF EXISTS ONLY lsif_indexable_repositories DROP CONSTRAINT IF EXISTS lsif_indexable_repositories_pkey;
ALTER TABLE IF EXISTS ONLY lsif_commits DROP CONSTRAINT IF EXISTS lsif_commits_pkey;
ALTER TABLE IF EXISTS ONLY global_state DROP CONSTRAINT IF EXISTS global_state_pkey;
ALTER TABLE IF EXISTS ONLY external_services DROP CONSTRAINT IF EXISTS external_services_pkey;
ALTER TABLE IF EXISTS ONLY event_logs DROP CONSTRAINT IF EXISTS event_logs_pkey;
ALTER TABLE IF EXISTS ONLY discussion_threads_target_repo DROP CONSTRAINT IF EXISTS discussion_threads_target_repo_pkey;
ALTER TABLE IF EXISTS ONLY discussion_threads DROP CONSTRAINT IF EXISTS discussion_threads_pkey;
ALTER TABLE IF EXISTS ONLY discussion_mail_reply_tokens DROP CONSTRAINT IF EXISTS discussion_mail_reply_tokens_pkey;
ALTER TABLE IF EXISTS ONLY discussion_comments DROP CONSTRAINT IF EXISTS discussion_comments_pkey;
ALTER TABLE IF EXISTS ONLY default_repos DROP CONSTRAINT IF EXISTS default_repos_pkey;
ALTER TABLE IF EXISTS ONLY critical_and_site_config DROP CONSTRAINT IF EXISTS critical_and_site_config_pkey;
ALTER TABLE IF EXISTS ONLY changesets DROP CONSTRAINT IF EXISTS changesets_repo_external_id_unique;
ALTER TABLE IF EXISTS ONLY changesets DROP CONSTRAINT IF EXISTS changesets_pkey;
ALTER TABLE IF EXISTS ONLY changeset_jobs DROP CONSTRAINT IF EXISTS changeset_jobs_unique;
ALTER TABLE IF EXISTS ONLY changeset_jobs DROP CONSTRAINT IF EXISTS changeset_jobs_pkey;
ALTER TABLE IF EXISTS ONLY changeset_events DROP CONSTRAINT IF EXISTS changeset_events_pkey;
ALTER TABLE IF EXISTS ONLY changeset_events DROP CONSTRAINT IF EXISTS changeset_events_changeset_id_kind_key_unique;
ALTER TABLE IF EXISTS ONLY campaigns DROP CONSTRAINT IF EXISTS campaigns_pkey;
ALTER TABLE IF EXISTS ONLY patch_sets DROP CONSTRAINT IF EXISTS campaign_plans_pkey;
ALTER TABLE IF EXISTS ONLY patches DROP CONSTRAINT IF EXISTS campaign_jobs_pkey;
ALTER TABLE IF EXISTS ONLY patches DROP CONSTRAINT IF EXISTS campaign_jobs_campaign_plan_repo_rev_unique;
ALTER TABLE IF EXISTS ONLY access_tokens DROP CONSTRAINT IF EXISTS access_tokens_value_sha256_key;
ALTER TABLE IF EXISTS ONLY access_tokens DROP CONSTRAINT IF EXISTS access_tokens_pkey;
ALTER TABLE IF EXISTS users ALTER COLUMN id DROP DEFAULT;
ALTER TABLE IF EXISTS user_pending_permissions ALTER COLUMN id DROP DEFAULT;
ALTER TABLE IF EXISTS user_external_accounts ALTER COLUMN id DROP DEFAULT;
ALTER TABLE IF EXISTS survey_responses ALTER COLUMN id DROP DEFAULT;
ALTER TABLE IF EXISTS settings ALTER COLUMN id DROP DEFAULT;
ALTER TABLE IF EXISTS saved_searches ALTER COLUMN id DROP DEFAULT;
ALTER TABLE IF EXISTS repo ALTER COLUMN id DROP DEFAULT;
ALTER TABLE IF EXISTS registry_extensions ALTER COLUMN id DROP DEFAULT;
ALTER TABLE IF EXISTS registry_extension_releases ALTER COLUMN id DROP DEFAULT;
ALTER TABLE IF EXISTS phabricator_repos ALTER COLUMN id DROP DEFAULT;
ALTER TABLE IF EXISTS patches ALTER COLUMN id DROP DEFAULT;
ALTER TABLE IF EXISTS patch_sets ALTER COLUMN id DROP DEFAULT;
ALTER TABLE IF EXISTS orgs ALTER COLUMN id DROP DEFAULT;
ALTER TABLE IF EXISTS org_members ALTER COLUMN id DROP DEFAULT;
ALTER TABLE IF EXISTS org_invitations ALTER COLUMN id DROP DEFAULT;
ALTER TABLE IF EXISTS lsif_uploads ALTER COLUMN id DROP DEFAULT;
ALTER TABLE IF EXISTS lsif_references ALTER COLUMN id DROP DEFAULT;
ALTER TABLE IF EXISTS lsif_packages ALTER COLUMN id DROP DEFAULT;
ALTER TABLE IF EXISTS lsif_indexes ALTER COLUMN id DROP DEFAULT;
ALTER TABLE IF EXISTS lsif_indexable_repositories ALTER COLUMN id DROP DEFAULT;
ALTER TABLE IF EXISTS lsif_commits ALTER COLUMN id DROP DEFAULT;
ALTER TABLE IF EXISTS external_services ALTER COLUMN id DROP DEFAULT;
ALTER TABLE IF EXISTS event_logs ALTER COLUMN id DROP DEFAULT;
ALTER TABLE IF EXISTS discussion_threads_target_repo ALTER COLUMN id DROP DEFAULT;
ALTER TABLE IF EXISTS discussion_threads ALTER COLUMN id DROP DEFAULT;
ALTER TABLE IF EXISTS discussion_comments ALTER COLUMN id DROP DEFAULT;
ALTER TABLE IF EXISTS critical_and_site_config ALTER COLUMN id DROP DEFAULT;
ALTER TABLE IF EXISTS changesets ALTER COLUMN id DROP DEFAULT;
ALTER TABLE IF EXISTS changeset_jobs ALTER COLUMN id DROP DEFAULT;
ALTER TABLE IF EXISTS changeset_events ALTER COLUMN id DROP DEFAULT;
ALTER TABLE IF EXISTS campaigns ALTER COLUMN id DROP DEFAULT;
ALTER TABLE IF EXISTS access_tokens ALTER COLUMN id DROP DEFAULT;
DROP TABLE IF EXISTS versions;
DROP SEQUENCE IF EXISTS users_id_seq;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS user_permissions;
DROP SEQUENCE IF EXISTS user_pending_permissions_id_seq;
DROP TABLE IF EXISTS user_pending_permissions;
DROP SEQUENCE IF EXISTS user_external_accounts_id_seq;
DROP TABLE IF EXISTS user_external_accounts;
DROP TABLE IF EXISTS user_emails;
DROP SEQUENCE IF EXISTS survey_responses_id_seq;
DROP TABLE IF EXISTS survey_responses;
DROP VIEW IF EXISTS site_config;
DROP SEQUENCE IF EXISTS settings_id_seq;
DROP TABLE IF EXISTS settings_bkup_1514702776;
DROP TABLE IF EXISTS settings;
DROP TABLE IF EXISTS schema_migrations;
DROP SEQUENCE IF EXISTS saved_searches_id_seq;
DROP TABLE IF EXISTS saved_searches;
DROP TABLE IF EXISTS saved_queries;
DROP TABLE IF EXISTS repo_permissions;
DROP TABLE IF EXISTS repo_pending_permissions;
DROP SEQUENCE IF EXISTS repo_id_seq;
DROP TABLE IF EXISTS repo;
DROP SEQUENCE IF EXISTS registry_extensions_id_seq;
DROP TABLE IF EXISTS registry_extensions;
DROP SEQUENCE IF EXISTS registry_extension_releases_id_seq;
DROP TABLE IF EXISTS registry_extension_releases;
DROP TABLE IF EXISTS query_runner_state;
DROP TABLE IF EXISTS product_subscriptions;
DROP TABLE IF EXISTS product_licenses;
DROP SEQUENCE IF EXISTS phabricator_repos_id_seq;
DROP TABLE IF EXISTS phabricator_repos;
DROP SEQUENCE IF EXISTS orgs_id_seq;
DROP TABLE IF EXISTS orgs;
DROP SEQUENCE IF EXISTS org_members_id_seq;
DROP TABLE IF EXISTS org_members_bkup_1514536731;
DROP TABLE IF EXISTS org_members;
DROP SEQUENCE IF EXISTS org_invitations_id_seq;
DROP TABLE IF EXISTS org_invitations;
DROP TABLE IF EXISTS names;
DROP SEQUENCE IF EXISTS lsif_references_id_seq;
DROP TABLE IF EXISTS lsif_references;
DROP SEQUENCE IF EXISTS lsif_packages_id_seq;
DROP TABLE IF EXISTS lsif_packages;
DROP SEQUENCE IF EXISTS lsif_indexes_id_seq;
DROP TABLE IF EXISTS lsif_indexes;
DROP SEQUENCE IF EXISTS lsif_indexable_repositories_id_seq;
DROP TABLE IF EXISTS lsif_indexable_repositories;
DROP SEQUENCE IF EXISTS lsif_dumps_id_seq;
DROP VIEW IF EXISTS lsif_dumps;
DROP TABLE IF EXISTS lsif_uploads;
DROP SEQUENCE IF EXISTS lsif_commits_id_seq;
DROP TABLE IF EXISTS lsif_commits;
DROP TABLE IF EXISTS global_state;
DROP SEQUENCE IF EXISTS external_services_id_seq;
DROP TABLE IF EXISTS external_services;
DROP SEQUENCE IF EXISTS event_logs_id_seq;
DROP TABLE IF EXISTS event_logs;
DROP SEQUENCE IF EXISTS discussion_threads_target_repo_id_seq;
DROP TABLE IF EXISTS discussion_threads_target_repo;
DROP SEQUENCE IF EXISTS discussion_threads_id_seq;
DROP TABLE IF EXISTS discussion_threads;
DROP TABLE IF EXISTS discussion_mail_reply_tokens;
DROP SEQUENCE IF EXISTS discussion_comments_id_seq;
DROP TABLE IF EXISTS discussion_comments;
DROP TABLE IF EXISTS default_repos;
DROP SEQUENCE IF EXISTS critical_and_site_config_id_seq;
DROP TABLE IF EXISTS critical_and_site_config;
DROP SEQUENCE IF EXISTS changesets_id_seq;
DROP TABLE IF EXISTS changesets;
DROP SEQUENCE IF EXISTS changeset_jobs_id_seq;
DROP TABLE IF EXISTS changeset_jobs;
DROP SEQUENCE IF EXISTS changeset_events_id_seq;
DROP TABLE IF EXISTS changeset_events;
DROP SEQUENCE IF EXISTS campaigns_id_seq;
DROP TABLE IF EXISTS campaigns;
DROP SEQUENCE IF EXISTS campaign_plans_id_seq;
DROP TABLE IF EXISTS patch_sets;
DROP SEQUENCE IF EXISTS campaign_jobs_id_seq;
DROP TABLE IF EXISTS patches;
DROP SEQUENCE IF EXISTS access_tokens_id_seq;
DROP TABLE IF EXISTS access_tokens;
DROP FUNCTION IF EXISTS delete_changeset_reference_on_campaigns();
DROP FUNCTION IF EXISTS delete_campaign_reference_on_changesets();
DROP TYPE IF EXISTS lsif_upload_state;
DROP TYPE IF EXISTS lsif_index_state;
DROP TYPE IF EXISTS critical_or_site;

CREATE EXTENSION IF NOT EXISTS citext;

CREATE EXTENSION IF NOT EXISTS hstore;

CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TYPE critical_or_site AS ENUM (
    'critical',
    'site'
);

CREATE TYPE lsif_index_state AS ENUM (
    'queued',
    'processing',
    'completed',
    'errored'
);

CREATE TYPE lsif_upload_state AS ENUM (
    'uploading',
    'queued',
    'processing',
    'completed',
    'errored'
);

CREATE FUNCTION delete_campaign_reference_on_changesets() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
    BEGIN
        UPDATE
          changesets
        SET
          campaign_ids = changesets.campaign_ids - OLD.id::text
        WHERE
          changesets.campaign_ids ? OLD.id::text;

        RETURN OLD;
    END;
$$;

CREATE FUNCTION delete_changeset_reference_on_campaigns() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
    BEGIN
        UPDATE
          campaigns
        SET
          changeset_ids = campaigns.changeset_ids - OLD.id::text
        WHERE
          campaigns.changeset_ids ? OLD.id::text;

        RETURN OLD;
    END;
$$;

CREATE TABLE access_tokens (
    id bigint NOT NULL,
    subject_user_id integer NOT NULL,
    value_sha256 bytea NOT NULL,
    note text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    last_used_at timestamp with time zone,
    deleted_at timestamp with time zone,
    creator_user_id integer NOT NULL,
    scopes text[] NOT NULL
);

CREATE SEQUENCE access_tokens_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE access_tokens_id_seq OWNED BY access_tokens.id;

CREATE TABLE patches (
    id bigint NOT NULL,
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
    CONSTRAINT campaign_jobs_base_ref_check CHECK ((base_ref <> ''::text))
);

CREATE SEQUENCE campaign_jobs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE campaign_jobs_id_seq OWNED BY patches.id;

CREATE TABLE patch_sets (
    id bigint NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    user_id integer NOT NULL
);

CREATE SEQUENCE campaign_plans_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE campaign_plans_id_seq OWNED BY patch_sets.id;

CREATE TABLE campaigns (
    id bigint NOT NULL,
    name text NOT NULL,
    description text,
    author_id integer NOT NULL,
    namespace_user_id integer,
    namespace_org_id integer,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    changeset_ids jsonb DEFAULT '{}'::jsonb NOT NULL,
    patch_set_id integer,
    closed_at timestamp with time zone,
    branch text,
    CONSTRAINT campaigns_changeset_ids_check CHECK ((jsonb_typeof(changeset_ids) = 'object'::text)),
    CONSTRAINT campaigns_has_1_namespace CHECK (((namespace_user_id IS NULL) <> (namespace_org_id IS NULL))),
    CONSTRAINT campaigns_name_not_blank CHECK ((name <> ''::text))
);

CREATE SEQUENCE campaigns_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE campaigns_id_seq OWNED BY campaigns.id;

CREATE TABLE changeset_events (
    id bigint NOT NULL,
    changeset_id bigint NOT NULL,
    kind text NOT NULL,
    key text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT changeset_events_key_check CHECK ((key <> ''::text)),
    CONSTRAINT changeset_events_kind_check CHECK ((kind <> ''::text)),
    CONSTRAINT changeset_events_metadata_check CHECK ((jsonb_typeof(metadata) = 'object'::text))
);

CREATE SEQUENCE changeset_events_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE changeset_events_id_seq OWNED BY changeset_events.id;

CREATE TABLE changeset_jobs (
    id bigint NOT NULL,
    campaign_id bigint NOT NULL,
    patch_id bigint NOT NULL,
    changeset_id bigint,
    error text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    started_at timestamp with time zone,
    finished_at timestamp with time zone,
    branch text
);

CREATE SEQUENCE changeset_jobs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE changeset_jobs_id_seq OWNED BY changeset_jobs.id;

CREATE TABLE changesets (
    id bigint NOT NULL,
    campaign_ids jsonb DEFAULT '{}'::jsonb NOT NULL,
    repo_id integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL,
    external_id text NOT NULL,
    external_service_type text NOT NULL,
    external_deleted_at timestamp with time zone,
    external_branch text,
    external_updated_at timestamp with time zone,
    external_state text,
    external_review_state text,
    external_check_state text,
    created_by_campaign boolean DEFAULT false NOT NULL,
    added_to_campaign boolean DEFAULT false NOT NULL,
    CONSTRAINT changesets_campaign_ids_check CHECK ((jsonb_typeof(campaign_ids) = 'object'::text)),
    CONSTRAINT changesets_external_id_check CHECK ((external_id <> ''::text)),
    CONSTRAINT changesets_external_service_type_not_blank CHECK ((external_service_type <> ''::text)),
    CONSTRAINT changesets_metadata_check CHECK ((jsonb_typeof(metadata) = 'object'::text))
);

CREATE SEQUENCE changesets_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE changesets_id_seq OWNED BY changesets.id;

CREATE TABLE critical_and_site_config (
    id integer NOT NULL,
    type critical_or_site NOT NULL,
    contents text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE SEQUENCE critical_and_site_config_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE critical_and_site_config_id_seq OWNED BY critical_and_site_config.id;

CREATE TABLE default_repos (
    repo_id integer NOT NULL
);

CREATE TABLE discussion_comments (
    id bigint NOT NULL,
    thread_id bigint NOT NULL,
    author_user_id integer NOT NULL,
    contents text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone,
    reports text[] DEFAULT '{}'::text[] NOT NULL
);

CREATE SEQUENCE discussion_comments_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE discussion_comments_id_seq OWNED BY discussion_comments.id;

CREATE TABLE discussion_mail_reply_tokens (
    token text NOT NULL,
    user_id integer NOT NULL,
    thread_id bigint NOT NULL,
    deleted_at timestamp with time zone
);

CREATE TABLE discussion_threads (
    id bigint NOT NULL,
    author_user_id integer NOT NULL,
    title text,
    target_repo_id bigint,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    archived_at timestamp with time zone,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone
);

CREATE SEQUENCE discussion_threads_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE discussion_threads_id_seq OWNED BY discussion_threads.id;

CREATE TABLE discussion_threads_target_repo (
    id bigint NOT NULL,
    thread_id bigint NOT NULL,
    repo_id integer NOT NULL,
    path text,
    branch text,
    revision text,
    start_line integer,
    end_line integer,
    start_character integer,
    end_character integer,
    lines_before text,
    lines text,
    lines_after text
);

CREATE SEQUENCE discussion_threads_target_repo_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE discussion_threads_target_repo_id_seq OWNED BY discussion_threads_target_repo.id;

CREATE TABLE event_logs (
    id bigint NOT NULL,
    name text NOT NULL,
    url text NOT NULL,
    user_id integer NOT NULL,
    anonymous_user_id text NOT NULL,
    source text NOT NULL,
    argument jsonb NOT NULL,
    version text NOT NULL,
    "timestamp" timestamp with time zone NOT NULL,
    CONSTRAINT event_logs_check_has_user CHECK ((((user_id = 0) AND (anonymous_user_id <> ''::text)) OR ((user_id <> 0) AND (anonymous_user_id = ''::text)) OR ((user_id <> 0) AND (anonymous_user_id <> ''::text)))),
    CONSTRAINT event_logs_check_name_not_empty CHECK ((name <> ''::text)),
    CONSTRAINT event_logs_check_source_not_empty CHECK ((source <> ''::text)),
    CONSTRAINT event_logs_check_version_not_empty CHECK ((version <> ''::text))
);

CREATE SEQUENCE event_logs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE event_logs_id_seq OWNED BY event_logs.id;

CREATE TABLE external_services (
    id bigint NOT NULL,
    kind text NOT NULL,
    display_name text NOT NULL,
    config text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone,
    CONSTRAINT check_non_empty_config CHECK ((btrim(config) <> ''::text))
);

CREATE SEQUENCE external_services_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE external_services_id_seq OWNED BY external_services.id;

CREATE TABLE global_state (
    site_id uuid NOT NULL,
    initialized boolean DEFAULT false NOT NULL,
    mgmt_password_plaintext text DEFAULT ''::text NOT NULL,
    mgmt_password_bcrypt text DEFAULT ''::text NOT NULL
);

CREATE TABLE lsif_commits (
    id integer NOT NULL,
    commit text NOT NULL,
    parent_commit text,
    repository_id integer NOT NULL,
    CONSTRAINT lsif_commits_commit_valid_chars CHECK ((commit ~ '^[a-z0-9]{40}$'::text)),
    CONSTRAINT lsif_commits_parent_commit_valid_chars CHECK ((parent_commit ~ '^[a-z0-9]{40}$'::text))
);

CREATE SEQUENCE lsif_commits_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE lsif_commits_id_seq OWNED BY lsif_commits.id;

CREATE TABLE lsif_uploads (
    id integer NOT NULL,
    commit text NOT NULL,
    root text DEFAULT ''::text NOT NULL,
    visible_at_tip boolean DEFAULT false NOT NULL,
    uploaded_at timestamp with time zone DEFAULT now() NOT NULL,
    state lsif_upload_state DEFAULT 'queued'::lsif_upload_state NOT NULL,
    failure_message text,
    started_at timestamp with time zone,
    finished_at timestamp with time zone,
    repository_id integer NOT NULL,
    indexer text NOT NULL,
    num_parts integer NOT NULL,
    uploaded_parts integer[] NOT NULL,
    process_after timestamp with time zone,
    num_resets integer DEFAULT 0 NOT NULL,
    CONSTRAINT lsif_uploads_commit_valid_chars CHECK ((commit ~ '^[a-z0-9]{40}$'::text))
);

CREATE VIEW lsif_dumps AS
 SELECT u.id,
    u.commit,
    u.root,
    u.visible_at_tip,
    u.uploaded_at,
    u.state,
    u.failure_message,
    u.started_at,
    u.finished_at,
    u.repository_id,
    u.indexer,
    u.num_parts,
    u.uploaded_parts,
    u.process_after,
    u.num_resets,
    u.finished_at AS processed_at
   FROM lsif_uploads u
  WHERE (u.state = 'completed'::lsif_upload_state);

CREATE SEQUENCE lsif_dumps_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE lsif_dumps_id_seq OWNED BY lsif_uploads.id;

CREATE TABLE lsif_indexable_repositories (
    id integer NOT NULL,
    repository_id integer NOT NULL,
    search_count integer DEFAULT 0 NOT NULL,
    precise_count integer DEFAULT 0 NOT NULL,
    last_index_enqueued_at timestamp with time zone
);

CREATE SEQUENCE lsif_indexable_repositories_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE lsif_indexable_repositories_id_seq OWNED BY lsif_indexable_repositories.id;

CREATE TABLE lsif_indexes (
    id bigint NOT NULL,
    commit text NOT NULL,
    queued_at timestamp with time zone DEFAULT now() NOT NULL,
    state lsif_index_state DEFAULT 'queued'::lsif_index_state NOT NULL,
    failure_message text,
    started_at timestamp with time zone,
    finished_at timestamp with time zone,
    repository_id integer NOT NULL,
    process_after timestamp with time zone,
    num_resets integer DEFAULT 0 NOT NULL,
    CONSTRAINT lsif_uploads_commit_valid_chars CHECK ((commit ~ '^[a-z0-9]{40}$'::text))
);

CREATE SEQUENCE lsif_indexes_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE lsif_indexes_id_seq OWNED BY lsif_indexes.id;

CREATE TABLE lsif_packages (
    id integer NOT NULL,
    scheme text NOT NULL,
    name text NOT NULL,
    version text,
    dump_id integer NOT NULL
);

CREATE SEQUENCE lsif_packages_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE lsif_packages_id_seq OWNED BY lsif_packages.id;

CREATE TABLE lsif_references (
    id integer NOT NULL,
    scheme text NOT NULL,
    name text NOT NULL,
    version text,
    filter bytea NOT NULL,
    dump_id integer NOT NULL
);

CREATE SEQUENCE lsif_references_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE lsif_references_id_seq OWNED BY lsif_references.id;

CREATE TABLE names (
    name citext NOT NULL,
    user_id integer,
    org_id integer,
    CONSTRAINT names_check CHECK (((user_id IS NOT NULL) OR (org_id IS NOT NULL)))
);

CREATE TABLE org_invitations (
    id bigint NOT NULL,
    org_id integer NOT NULL,
    sender_user_id integer NOT NULL,
    recipient_user_id integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    notified_at timestamp with time zone,
    responded_at timestamp with time zone,
    response_type boolean,
    revoked_at timestamp with time zone,
    deleted_at timestamp with time zone,
    CONSTRAINT check_atomic_response CHECK (((responded_at IS NULL) = (response_type IS NULL))),
    CONSTRAINT check_single_use CHECK ((((responded_at IS NULL) AND (response_type IS NULL)) OR (revoked_at IS NULL)))
);

CREATE SEQUENCE org_invitations_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE org_invitations_id_seq OWNED BY org_invitations.id;

CREATE TABLE org_members (
    id integer NOT NULL,
    org_id integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    user_id integer NOT NULL
);

CREATE TABLE org_members_bkup_1514536731 (
    id integer,
    org_id integer,
    user_id_old text,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    user_id integer
);

CREATE SEQUENCE org_members_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE org_members_id_seq OWNED BY org_members.id;

CREATE TABLE orgs (
    id integer NOT NULL,
    name citext NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    display_name text,
    slack_webhook_url text,
    deleted_at timestamp with time zone,
    CONSTRAINT orgs_display_name_max_length CHECK ((char_length(display_name) <= 255)),
    CONSTRAINT orgs_name_max_length CHECK ((char_length((name)::text) <= 255)),
    CONSTRAINT orgs_name_valid_chars CHECK ((name OPERATOR(~) '^[a-zA-Z0-9](?:[a-zA-Z0-9]|[-.](?=[a-zA-Z0-9]))*-?$'::citext))
);

CREATE SEQUENCE orgs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE orgs_id_seq OWNED BY orgs.id;

CREATE TABLE phabricator_repos (
    id integer NOT NULL,
    callsign citext NOT NULL,
    repo_name citext NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone,
    url text DEFAULT ''::text NOT NULL
);

CREATE SEQUENCE phabricator_repos_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE phabricator_repos_id_seq OWNED BY phabricator_repos.id;

CREATE TABLE product_licenses (
    id uuid NOT NULL,
    product_subscription_id uuid NOT NULL,
    license_key text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE TABLE product_subscriptions (
    id uuid NOT NULL,
    user_id integer NOT NULL,
    billing_subscription_id text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    archived_at timestamp with time zone
);

CREATE TABLE query_runner_state (
    query text,
    last_executed timestamp with time zone,
    latest_result timestamp with time zone,
    exec_duration_ns bigint
);

CREATE TABLE registry_extension_releases (
    id bigint NOT NULL,
    registry_extension_id integer NOT NULL,
    creator_user_id integer NOT NULL,
    release_version citext,
    release_tag citext NOT NULL,
    manifest jsonb NOT NULL,
    bundle text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone,
    source_map text
);

CREATE SEQUENCE registry_extension_releases_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE registry_extension_releases_id_seq OWNED BY registry_extension_releases.id;

CREATE TABLE registry_extensions (
    id integer NOT NULL,
    uuid uuid NOT NULL,
    publisher_user_id integer,
    publisher_org_id integer,
    name citext NOT NULL,
    manifest text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone,
    CONSTRAINT registry_extensions_name_length CHECK (((char_length((name)::text) > 0) AND (char_length((name)::text) <= 128))),
    CONSTRAINT registry_extensions_name_valid_chars CHECK ((name OPERATOR(~) '^[a-zA-Z0-9](?:[a-zA-Z0-9]|[_.-](?=[a-zA-Z0-9]))*$'::citext)),
    CONSTRAINT registry_extensions_single_publisher CHECK (((publisher_user_id IS NULL) <> (publisher_org_id IS NULL)))
);

CREATE SEQUENCE registry_extensions_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE registry_extensions_id_seq OWNED BY registry_extensions.id;

CREATE TABLE repo (
    id integer NOT NULL,
    name citext NOT NULL,
    description text,
    language text,
    fork boolean,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone,
    external_id text,
    external_service_type text,
    external_service_id text,
    archived boolean DEFAULT false NOT NULL,
    uri citext,
    deleted_at timestamp with time zone,
    sources jsonb DEFAULT '{}'::jsonb NOT NULL,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL,
    private boolean DEFAULT false NOT NULL,
    CONSTRAINT check_name_nonempty CHECK ((name OPERATOR(<>) ''::citext)),
    CONSTRAINT repo_metadata_check CHECK ((jsonb_typeof(metadata) = 'object'::text)),
    CONSTRAINT repo_sources_check CHECK ((jsonb_typeof(sources) = 'object'::text))
);

CREATE SEQUENCE repo_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE repo_id_seq OWNED BY repo.id;

CREATE TABLE repo_pending_permissions (
    repo_id integer NOT NULL,
    permission text NOT NULL,
    user_ids bytea NOT NULL,
    updated_at timestamp with time zone NOT NULL
);

CREATE TABLE repo_permissions (
    repo_id integer NOT NULL,
    permission text NOT NULL,
    user_ids bytea NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    synced_at timestamp with time zone
);

CREATE TABLE saved_queries (
    query text NOT NULL,
    last_executed timestamp with time zone NOT NULL,
    latest_result timestamp with time zone NOT NULL,
    exec_duration_ns bigint NOT NULL
);

CREATE TABLE saved_searches (
    id integer NOT NULL,
    description text NOT NULL,
    query text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    notify_owner boolean NOT NULL,
    notify_slack boolean NOT NULL,
    user_id integer,
    org_id integer,
    slack_webhook_url text,
    CONSTRAINT user_or_org_id_not_null CHECK ((((user_id IS NOT NULL) AND (org_id IS NULL)) OR ((org_id IS NOT NULL) AND (user_id IS NULL))))
);

CREATE SEQUENCE saved_searches_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE saved_searches_id_seq OWNED BY saved_searches.id;

CREATE TABLE schema_migrations (
    version bigint NOT NULL,
    dirty boolean NOT NULL
);

CREATE TABLE settings (
    id integer NOT NULL,
    org_id integer,
    contents text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    user_id integer,
    author_user_id integer
);

CREATE TABLE settings_bkup_1514702776 (
    id integer,
    org_id integer,
    author_user_id_old text,
    contents text,
    created_at timestamp with time zone,
    user_id integer,
    author_user_id integer
);

CREATE SEQUENCE settings_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE settings_id_seq OWNED BY settings.id;

CREATE VIEW site_config AS
 SELECT global_state.site_id,
    global_state.initialized
   FROM global_state;

CREATE TABLE survey_responses (
    id bigint NOT NULL,
    user_id integer,
    email text,
    score integer NOT NULL,
    reason text,
    better text,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE SEQUENCE survey_responses_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE survey_responses_id_seq OWNED BY survey_responses.id;

CREATE TABLE user_emails (
    user_id integer NOT NULL,
    email citext NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    verification_code text,
    verified_at timestamp with time zone,
    last_verification_sent_at timestamp with time zone
);

CREATE TABLE user_external_accounts (
    id integer NOT NULL,
    user_id integer NOT NULL,
    service_type text NOT NULL,
    service_id text NOT NULL,
    account_id text NOT NULL,
    auth_data jsonb,
    account_data jsonb,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone,
    client_id text NOT NULL
);

CREATE SEQUENCE user_external_accounts_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE user_external_accounts_id_seq OWNED BY user_external_accounts.id;

CREATE TABLE user_pending_permissions (
    id integer NOT NULL,
    bind_id text NOT NULL,
    permission text NOT NULL,
    object_type text NOT NULL,
    object_ids bytea NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    service_type text NOT NULL,
    service_id text NOT NULL
);

CREATE SEQUENCE user_pending_permissions_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE user_pending_permissions_id_seq OWNED BY user_pending_permissions.id;

CREATE TABLE user_permissions (
    user_id integer NOT NULL,
    permission text NOT NULL,
    object_type text NOT NULL,
    object_ids bytea NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    synced_at timestamp with time zone
);

CREATE TABLE users (
    id integer NOT NULL,
    username citext NOT NULL,
    display_name text,
    avatar_url text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone,
    invite_quota integer DEFAULT 15 NOT NULL,
    passwd text,
    passwd_reset_code text,
    passwd_reset_time timestamp with time zone,
    site_admin boolean DEFAULT false NOT NULL,
    page_views integer DEFAULT 0 NOT NULL,
    search_queries integer DEFAULT 0 NOT NULL,
    tags text[] DEFAULT '{}'::text[],
    billing_customer_id text,
    CONSTRAINT users_display_name_max_length CHECK ((char_length(display_name) <= 255)),
    CONSTRAINT users_username_max_length CHECK ((char_length((username)::text) <= 255)),
    CONSTRAINT users_username_valid_chars CHECK ((username OPERATOR(~) '^[a-zA-Z0-9](?:[a-zA-Z0-9]|[-.](?=[a-zA-Z0-9]))*-?$'::citext))
);

CREATE SEQUENCE users_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE users_id_seq OWNED BY users.id;

CREATE TABLE versions (
    service text NOT NULL,
    version text NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

ALTER TABLE ONLY access_tokens ALTER COLUMN id SET DEFAULT nextval('access_tokens_id_seq'::regclass);

ALTER TABLE ONLY campaigns ALTER COLUMN id SET DEFAULT nextval('campaigns_id_seq'::regclass);

ALTER TABLE ONLY changeset_events ALTER COLUMN id SET DEFAULT nextval('changeset_events_id_seq'::regclass);

ALTER TABLE ONLY changeset_jobs ALTER COLUMN id SET DEFAULT nextval('changeset_jobs_id_seq'::regclass);

ALTER TABLE ONLY changesets ALTER COLUMN id SET DEFAULT nextval('changesets_id_seq'::regclass);

ALTER TABLE ONLY critical_and_site_config ALTER COLUMN id SET DEFAULT nextval('critical_and_site_config_id_seq'::regclass);

ALTER TABLE ONLY discussion_comments ALTER COLUMN id SET DEFAULT nextval('discussion_comments_id_seq'::regclass);

ALTER TABLE ONLY discussion_threads ALTER COLUMN id SET DEFAULT nextval('discussion_threads_id_seq'::regclass);

ALTER TABLE ONLY discussion_threads_target_repo ALTER COLUMN id SET DEFAULT nextval('discussion_threads_target_repo_id_seq'::regclass);

ALTER TABLE ONLY event_logs ALTER COLUMN id SET DEFAULT nextval('event_logs_id_seq'::regclass);

ALTER TABLE ONLY external_services ALTER COLUMN id SET DEFAULT nextval('external_services_id_seq'::regclass);

ALTER TABLE ONLY lsif_commits ALTER COLUMN id SET DEFAULT nextval('lsif_commits_id_seq'::regclass);

ALTER TABLE ONLY lsif_indexable_repositories ALTER COLUMN id SET DEFAULT nextval('lsif_indexable_repositories_id_seq'::regclass);

ALTER TABLE ONLY lsif_indexes ALTER COLUMN id SET DEFAULT nextval('lsif_indexes_id_seq'::regclass);

ALTER TABLE ONLY lsif_packages ALTER COLUMN id SET DEFAULT nextval('lsif_packages_id_seq'::regclass);

ALTER TABLE ONLY lsif_references ALTER COLUMN id SET DEFAULT nextval('lsif_references_id_seq'::regclass);

ALTER TABLE ONLY lsif_uploads ALTER COLUMN id SET DEFAULT nextval('lsif_dumps_id_seq'::regclass);

ALTER TABLE ONLY org_invitations ALTER COLUMN id SET DEFAULT nextval('org_invitations_id_seq'::regclass);

ALTER TABLE ONLY org_members ALTER COLUMN id SET DEFAULT nextval('org_members_id_seq'::regclass);

ALTER TABLE ONLY orgs ALTER COLUMN id SET DEFAULT nextval('orgs_id_seq'::regclass);

ALTER TABLE ONLY patch_sets ALTER COLUMN id SET DEFAULT nextval('campaign_plans_id_seq'::regclass);

ALTER TABLE ONLY patches ALTER COLUMN id SET DEFAULT nextval('campaign_jobs_id_seq'::regclass);

ALTER TABLE ONLY phabricator_repos ALTER COLUMN id SET DEFAULT nextval('phabricator_repos_id_seq'::regclass);

ALTER TABLE ONLY registry_extension_releases ALTER COLUMN id SET DEFAULT nextval('registry_extension_releases_id_seq'::regclass);

ALTER TABLE ONLY registry_extensions ALTER COLUMN id SET DEFAULT nextval('registry_extensions_id_seq'::regclass);

ALTER TABLE ONLY repo ALTER COLUMN id SET DEFAULT nextval('repo_id_seq'::regclass);

ALTER TABLE ONLY saved_searches ALTER COLUMN id SET DEFAULT nextval('saved_searches_id_seq'::regclass);

ALTER TABLE ONLY settings ALTER COLUMN id SET DEFAULT nextval('settings_id_seq'::regclass);

ALTER TABLE ONLY survey_responses ALTER COLUMN id SET DEFAULT nextval('survey_responses_id_seq'::regclass);

ALTER TABLE ONLY user_external_accounts ALTER COLUMN id SET DEFAULT nextval('user_external_accounts_id_seq'::regclass);

ALTER TABLE ONLY user_pending_permissions ALTER COLUMN id SET DEFAULT nextval('user_pending_permissions_id_seq'::regclass);

ALTER TABLE ONLY users ALTER COLUMN id SET DEFAULT nextval('users_id_seq'::regclass);

ALTER TABLE ONLY access_tokens
    ADD CONSTRAINT access_tokens_pkey PRIMARY KEY (id);

ALTER TABLE ONLY access_tokens
    ADD CONSTRAINT access_tokens_value_sha256_key UNIQUE (value_sha256);

ALTER TABLE ONLY patches
    ADD CONSTRAINT campaign_jobs_campaign_plan_repo_rev_unique UNIQUE (patch_set_id, repo_id, rev) DEFERRABLE;

ALTER TABLE ONLY patches
    ADD CONSTRAINT campaign_jobs_pkey PRIMARY KEY (id);

ALTER TABLE ONLY patch_sets
    ADD CONSTRAINT campaign_plans_pkey PRIMARY KEY (id);

ALTER TABLE ONLY campaigns
    ADD CONSTRAINT campaigns_pkey PRIMARY KEY (id);

ALTER TABLE ONLY changeset_events
    ADD CONSTRAINT changeset_events_changeset_id_kind_key_unique UNIQUE (changeset_id, kind, key);

ALTER TABLE ONLY changeset_events
    ADD CONSTRAINT changeset_events_pkey PRIMARY KEY (id);

ALTER TABLE ONLY changeset_jobs
    ADD CONSTRAINT changeset_jobs_pkey PRIMARY KEY (id);

ALTER TABLE ONLY changeset_jobs
    ADD CONSTRAINT changeset_jobs_unique UNIQUE (campaign_id, patch_id);

ALTER TABLE ONLY changesets
    ADD CONSTRAINT changesets_pkey PRIMARY KEY (id);

ALTER TABLE ONLY changesets
    ADD CONSTRAINT changesets_repo_external_id_unique UNIQUE (repo_id, external_id);

ALTER TABLE ONLY critical_and_site_config
    ADD CONSTRAINT critical_and_site_config_pkey PRIMARY KEY (id);

ALTER TABLE ONLY default_repos
    ADD CONSTRAINT default_repos_pkey PRIMARY KEY (repo_id);

ALTER TABLE ONLY discussion_comments
    ADD CONSTRAINT discussion_comments_pkey PRIMARY KEY (id);

ALTER TABLE ONLY discussion_mail_reply_tokens
    ADD CONSTRAINT discussion_mail_reply_tokens_pkey PRIMARY KEY (token);

ALTER TABLE ONLY discussion_threads
    ADD CONSTRAINT discussion_threads_pkey PRIMARY KEY (id);

ALTER TABLE ONLY discussion_threads_target_repo
    ADD CONSTRAINT discussion_threads_target_repo_pkey PRIMARY KEY (id);

ALTER TABLE ONLY event_logs
    ADD CONSTRAINT event_logs_pkey PRIMARY KEY (id);

ALTER TABLE ONLY external_services
    ADD CONSTRAINT external_services_pkey PRIMARY KEY (id);

ALTER TABLE ONLY global_state
    ADD CONSTRAINT global_state_pkey PRIMARY KEY (site_id);

ALTER TABLE ONLY lsif_commits
    ADD CONSTRAINT lsif_commits_pkey PRIMARY KEY (id);

ALTER TABLE ONLY lsif_indexable_repositories
    ADD CONSTRAINT lsif_indexable_repositories_pkey PRIMARY KEY (id);

ALTER TABLE ONLY lsif_indexable_repositories
    ADD CONSTRAINT lsif_indexable_repositories_repository_id_key UNIQUE (repository_id);

ALTER TABLE ONLY lsif_indexes
    ADD CONSTRAINT lsif_indexes_pkey PRIMARY KEY (id);

ALTER TABLE ONLY lsif_packages
    ADD CONSTRAINT lsif_packages_pkey PRIMARY KEY (id);

ALTER TABLE ONLY lsif_references
    ADD CONSTRAINT lsif_references_pkey PRIMARY KEY (id);

ALTER TABLE ONLY lsif_uploads
    ADD CONSTRAINT lsif_uploads_pkey PRIMARY KEY (id);

ALTER TABLE ONLY names
    ADD CONSTRAINT names_pkey PRIMARY KEY (name);

ALTER TABLE ONLY org_invitations
    ADD CONSTRAINT org_invitations_pkey PRIMARY KEY (id);

ALTER TABLE ONLY org_members
    ADD CONSTRAINT org_members_org_id_user_id_key UNIQUE (org_id, user_id);

ALTER TABLE ONLY org_members
    ADD CONSTRAINT org_members_pkey PRIMARY KEY (id);

ALTER TABLE ONLY orgs
    ADD CONSTRAINT orgs_pkey PRIMARY KEY (id);

ALTER TABLE ONLY phabricator_repos
    ADD CONSTRAINT phabricator_repos_pkey PRIMARY KEY (id);

ALTER TABLE ONLY phabricator_repos
    ADD CONSTRAINT phabricator_repos_repo_name_key UNIQUE (repo_name);

ALTER TABLE ONLY product_licenses
    ADD CONSTRAINT product_licenses_pkey PRIMARY KEY (id);

ALTER TABLE ONLY product_subscriptions
    ADD CONSTRAINT product_subscriptions_pkey PRIMARY KEY (id);

ALTER TABLE ONLY registry_extension_releases
    ADD CONSTRAINT registry_extension_releases_pkey PRIMARY KEY (id);

ALTER TABLE ONLY registry_extensions
    ADD CONSTRAINT registry_extensions_pkey PRIMARY KEY (id);

ALTER TABLE ONLY repo
    ADD CONSTRAINT repo_name_unique UNIQUE (name) DEFERRABLE;

ALTER TABLE ONLY repo_pending_permissions
    ADD CONSTRAINT repo_pending_permissions_perm_unique UNIQUE (repo_id, permission);

ALTER TABLE ONLY repo_permissions
    ADD CONSTRAINT repo_permissions_perm_unique UNIQUE (repo_id, permission);

ALTER TABLE ONLY repo
    ADD CONSTRAINT repo_pkey PRIMARY KEY (id);

ALTER TABLE ONLY saved_searches
    ADD CONSTRAINT saved_searches_pkey PRIMARY KEY (id);

ALTER TABLE ONLY schema_migrations
    ADD CONSTRAINT schema_migrations_pkey PRIMARY KEY (version);

ALTER TABLE ONLY settings
    ADD CONSTRAINT settings_pkey PRIMARY KEY (id);

ALTER TABLE ONLY survey_responses
    ADD CONSTRAINT survey_responses_pkey PRIMARY KEY (id);

ALTER TABLE ONLY user_emails
    ADD CONSTRAINT user_emails_no_duplicates_per_user UNIQUE (user_id, email);

ALTER TABLE ONLY user_emails
    ADD CONSTRAINT user_emails_unique_verified_email EXCLUDE USING btree (email WITH OPERATOR(=)) WHERE ((verified_at IS NOT NULL));

ALTER TABLE ONLY user_external_accounts
    ADD CONSTRAINT user_external_accounts_pkey PRIMARY KEY (id);

ALTER TABLE ONLY user_pending_permissions
    ADD CONSTRAINT user_pending_permissions_service_perm_object_unique UNIQUE (service_type, service_id, permission, object_type, bind_id);

ALTER TABLE ONLY user_permissions
    ADD CONSTRAINT user_permissions_perm_object_unique UNIQUE (user_id, permission, object_type);

ALTER TABLE ONLY users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);

ALTER TABLE ONLY versions
    ADD CONSTRAINT versions_pkey PRIMARY KEY (service);

CREATE INDEX access_tokens_lookup ON access_tokens USING hash (value_sha256) WHERE (deleted_at IS NULL);

CREATE INDEX campaigns_changeset_ids_gin_idx ON campaigns USING gin (changeset_ids);

CREATE INDEX campaigns_namespace_org_id ON campaigns USING btree (namespace_org_id);

CREATE INDEX campaigns_namespace_user_id ON campaigns USING btree (namespace_user_id);

CREATE INDEX changeset_jobs_campaign_job_id ON changeset_jobs USING btree (patch_id);

CREATE INDEX changeset_jobs_error_not_null ON changeset_jobs USING btree (((error IS NOT NULL)));

CREATE INDEX changeset_jobs_finished_at ON changeset_jobs USING btree (finished_at);

CREATE INDEX changeset_jobs_started_at ON changeset_jobs USING btree (started_at);

CREATE UNIQUE INDEX critical_and_site_config_unique ON critical_and_site_config USING btree (id, type);

CREATE INDEX discussion_comments_author_user_id_idx ON discussion_comments USING btree (author_user_id);

CREATE INDEX discussion_comments_reports_array_length_idx ON discussion_comments USING btree (array_length(reports, 1));

CREATE INDEX discussion_comments_thread_id_idx ON discussion_comments USING btree (thread_id);

CREATE INDEX discussion_mail_reply_tokens_user_id_thread_id_idx ON discussion_mail_reply_tokens USING btree (user_id, thread_id);

CREATE INDEX discussion_threads_author_user_id_idx ON discussion_threads USING btree (author_user_id);

CREATE INDEX discussion_threads_target_repo_repo_id_path_idx ON discussion_threads_target_repo USING btree (repo_id, path);

CREATE INDEX event_logs_anonymous_user_id ON event_logs USING btree (anonymous_user_id);

CREATE INDEX event_logs_name ON event_logs USING btree (name);

CREATE INDEX event_logs_source ON event_logs USING btree (source);

CREATE INDEX event_logs_timestamp ON event_logs USING btree ("timestamp");

CREATE INDEX event_logs_timestamp_at_utc ON event_logs USING btree (date(timezone('UTC'::text, "timestamp")));

CREATE INDEX event_logs_user_id ON event_logs USING btree (user_id);

CREATE UNIQUE INDEX lsif_commits_repository_id_commit_parent_commit_unique ON lsif_commits USING btree (repository_id, commit, parent_commit);

CREATE INDEX lsif_commits_repository_id_parent_commit ON lsif_commits USING btree (repository_id, parent_commit);

CREATE INDEX lsif_packages_scheme_name_version ON lsif_packages USING btree (scheme, name, version);

CREATE INDEX lsif_references_package ON lsif_references USING btree (scheme, name, version);

CREATE UNIQUE INDEX lsif_uploads_repository_id_commit_root_indexer ON lsif_uploads USING btree (repository_id, commit, root, indexer) WHERE (state = 'completed'::lsif_upload_state);

CREATE INDEX lsif_uploads_state ON lsif_uploads USING btree (state);

CREATE INDEX lsif_uploads_uploaded_at ON lsif_uploads USING btree (uploaded_at);

CREATE INDEX lsif_uploads_visible_repository_id_commit ON lsif_uploads USING btree (repository_id, commit) WHERE visible_at_tip;

CREATE INDEX org_invitations_org_id ON org_invitations USING btree (org_id) WHERE (deleted_at IS NULL);

CREATE INDEX org_invitations_recipient_user_id ON org_invitations USING btree (recipient_user_id) WHERE (deleted_at IS NULL);

CREATE UNIQUE INDEX org_invitations_singleflight ON org_invitations USING btree (org_id, recipient_user_id) WHERE ((responded_at IS NULL) AND (revoked_at IS NULL) AND (deleted_at IS NULL));

CREATE UNIQUE INDEX orgs_name ON orgs USING btree (name) WHERE (deleted_at IS NULL);

CREATE INDEX registry_extension_releases_registry_extension_id ON registry_extension_releases USING btree (registry_extension_id, release_tag, created_at DESC) WHERE (deleted_at IS NULL);

CREATE UNIQUE INDEX registry_extension_releases_version ON registry_extension_releases USING btree (registry_extension_id, release_version) WHERE (release_version IS NOT NULL);

CREATE UNIQUE INDEX registry_extensions_publisher_name ON registry_extensions USING btree ((COALESCE(publisher_user_id, 0)), (COALESCE(publisher_org_id, 0)), name) WHERE (deleted_at IS NULL);

CREATE UNIQUE INDEX registry_extensions_uuid ON registry_extensions USING btree (uuid);

CREATE INDEX repo_archived ON repo USING btree (archived);

CREATE UNIQUE INDEX repo_external_unique_idx ON repo USING btree (external_service_type, external_service_id, external_id);

CREATE INDEX repo_fork ON repo USING btree (fork);

CREATE INDEX repo_metadata_gin_idx ON repo USING gin (metadata);

CREATE INDEX repo_name_trgm ON repo USING gin (lower((name)::text) gin_trgm_ops);

CREATE INDEX repo_private ON repo USING btree (private);

CREATE INDEX repo_sources_gin_idx ON repo USING gin (sources);

CREATE INDEX repo_uri_idx ON repo USING btree (uri);

CREATE UNIQUE INDEX saved_queries_query_unique ON saved_queries USING btree (query);

CREATE UNIQUE INDEX user_external_accounts_account ON user_external_accounts USING btree (service_type, service_id, client_id, account_id) WHERE (deleted_at IS NULL);

CREATE INDEX user_external_accounts_user_id ON user_external_accounts USING btree (user_id) WHERE (deleted_at IS NULL);

CREATE UNIQUE INDEX users_billing_customer_id ON users USING btree (billing_customer_id) WHERE (deleted_at IS NULL);

CREATE UNIQUE INDEX users_username ON users USING btree (username) WHERE (deleted_at IS NULL);

CREATE TRIGGER trig_delete_campaign_reference_on_changesets AFTER DELETE ON campaigns FOR EACH ROW EXECUTE PROCEDURE delete_campaign_reference_on_changesets();

CREATE TRIGGER trig_delete_changeset_reference_on_campaigns AFTER DELETE ON changesets FOR EACH ROW EXECUTE PROCEDURE delete_changeset_reference_on_campaigns();

ALTER TABLE ONLY access_tokens
    ADD CONSTRAINT access_tokens_creator_user_id_fkey FOREIGN KEY (creator_user_id) REFERENCES users(id);

ALTER TABLE ONLY access_tokens
    ADD CONSTRAINT access_tokens_subject_user_id_fkey FOREIGN KEY (subject_user_id) REFERENCES users(id);

ALTER TABLE ONLY patches
    ADD CONSTRAINT campaign_jobs_campaign_plan_id_fkey FOREIGN KEY (patch_set_id) REFERENCES patch_sets(id) ON DELETE CASCADE DEFERRABLE;

ALTER TABLE ONLY patches
    ADD CONSTRAINT campaign_jobs_repo_id_fkey FOREIGN KEY (repo_id) REFERENCES repo(id) ON DELETE CASCADE DEFERRABLE;

ALTER TABLE ONLY patch_sets
    ADD CONSTRAINT campaign_plans_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) DEFERRABLE;

ALTER TABLE ONLY campaigns
    ADD CONSTRAINT campaigns_author_id_fkey FOREIGN KEY (author_id) REFERENCES users(id) ON DELETE CASCADE DEFERRABLE;

ALTER TABLE ONLY campaigns
    ADD CONSTRAINT campaigns_campaign_plan_id_fkey FOREIGN KEY (patch_set_id) REFERENCES patch_sets(id) DEFERRABLE;

ALTER TABLE ONLY campaigns
    ADD CONSTRAINT campaigns_namespace_org_id_fkey FOREIGN KEY (namespace_org_id) REFERENCES orgs(id) ON DELETE CASCADE DEFERRABLE;

ALTER TABLE ONLY campaigns
    ADD CONSTRAINT campaigns_namespace_user_id_fkey FOREIGN KEY (namespace_user_id) REFERENCES users(id) ON DELETE CASCADE DEFERRABLE;

ALTER TABLE ONLY changeset_events
    ADD CONSTRAINT changeset_events_changeset_id_fkey FOREIGN KEY (changeset_id) REFERENCES changesets(id) ON DELETE CASCADE DEFERRABLE;

ALTER TABLE ONLY changeset_jobs
    ADD CONSTRAINT changeset_jobs_campaign_id_fkey FOREIGN KEY (campaign_id) REFERENCES campaigns(id) ON DELETE CASCADE DEFERRABLE;

ALTER TABLE ONLY changeset_jobs
    ADD CONSTRAINT changeset_jobs_campaign_job_id_fkey FOREIGN KEY (patch_id) REFERENCES patches(id) ON DELETE CASCADE DEFERRABLE;

ALTER TABLE ONLY changeset_jobs
    ADD CONSTRAINT changeset_jobs_changeset_id_fkey FOREIGN KEY (changeset_id) REFERENCES changesets(id) ON DELETE CASCADE DEFERRABLE;

ALTER TABLE ONLY changesets
    ADD CONSTRAINT changesets_repo_id_fkey FOREIGN KEY (repo_id) REFERENCES repo(id) ON DELETE CASCADE DEFERRABLE;

ALTER TABLE ONLY default_repos
    ADD CONSTRAINT default_repos_repo_id_fkey FOREIGN KEY (repo_id) REFERENCES repo(id) ON DELETE CASCADE;

ALTER TABLE ONLY discussion_comments
    ADD CONSTRAINT discussion_comments_author_user_id_fkey FOREIGN KEY (author_user_id) REFERENCES users(id) ON DELETE RESTRICT;

ALTER TABLE ONLY discussion_comments
    ADD CONSTRAINT discussion_comments_thread_id_fkey FOREIGN KEY (thread_id) REFERENCES discussion_threads(id) ON DELETE CASCADE;

ALTER TABLE ONLY discussion_mail_reply_tokens
    ADD CONSTRAINT discussion_mail_reply_tokens_thread_id_fkey FOREIGN KEY (thread_id) REFERENCES discussion_threads(id) ON DELETE CASCADE;

ALTER TABLE ONLY discussion_mail_reply_tokens
    ADD CONSTRAINT discussion_mail_reply_tokens_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE RESTRICT;

ALTER TABLE ONLY discussion_threads
    ADD CONSTRAINT discussion_threads_author_user_id_fkey FOREIGN KEY (author_user_id) REFERENCES users(id) ON DELETE RESTRICT;

ALTER TABLE ONLY discussion_threads
    ADD CONSTRAINT discussion_threads_target_repo_id_fk FOREIGN KEY (target_repo_id) REFERENCES discussion_threads_target_repo(id) ON DELETE CASCADE;

ALTER TABLE ONLY discussion_threads_target_repo
    ADD CONSTRAINT discussion_threads_target_repo_repo_id_fkey FOREIGN KEY (repo_id) REFERENCES repo(id) ON DELETE CASCADE;

ALTER TABLE ONLY discussion_threads_target_repo
    ADD CONSTRAINT discussion_threads_target_repo_thread_id_fkey FOREIGN KEY (thread_id) REFERENCES discussion_threads(id) ON DELETE CASCADE;

ALTER TABLE ONLY lsif_packages
    ADD CONSTRAINT lsif_packages_dump_id_fkey FOREIGN KEY (dump_id) REFERENCES lsif_uploads(id) ON DELETE CASCADE;

ALTER TABLE ONLY lsif_references
    ADD CONSTRAINT lsif_references_dump_id_fkey FOREIGN KEY (dump_id) REFERENCES lsif_uploads(id) ON DELETE CASCADE;

ALTER TABLE ONLY names
    ADD CONSTRAINT names_org_id_fkey FOREIGN KEY (org_id) REFERENCES orgs(id) ON UPDATE CASCADE ON DELETE CASCADE;

ALTER TABLE ONLY names
    ADD CONSTRAINT names_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON UPDATE CASCADE ON DELETE CASCADE;

ALTER TABLE ONLY org_invitations
    ADD CONSTRAINT org_invitations_org_id_fkey FOREIGN KEY (org_id) REFERENCES orgs(id);

ALTER TABLE ONLY org_invitations
    ADD CONSTRAINT org_invitations_recipient_user_id_fkey FOREIGN KEY (recipient_user_id) REFERENCES users(id);

ALTER TABLE ONLY org_invitations
    ADD CONSTRAINT org_invitations_sender_user_id_fkey FOREIGN KEY (sender_user_id) REFERENCES users(id);

ALTER TABLE ONLY org_members
    ADD CONSTRAINT org_members_references_orgs FOREIGN KEY (org_id) REFERENCES orgs(id) ON DELETE RESTRICT;

ALTER TABLE ONLY org_members
    ADD CONSTRAINT org_members_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE RESTRICT;

ALTER TABLE ONLY product_licenses
    ADD CONSTRAINT product_licenses_product_subscription_id_fkey FOREIGN KEY (product_subscription_id) REFERENCES product_subscriptions(id);

ALTER TABLE ONLY product_subscriptions
    ADD CONSTRAINT product_subscriptions_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id);

ALTER TABLE ONLY registry_extension_releases
    ADD CONSTRAINT registry_extension_releases_creator_user_id_fkey FOREIGN KEY (creator_user_id) REFERENCES users(id);

ALTER TABLE ONLY registry_extension_releases
    ADD CONSTRAINT registry_extension_releases_registry_extension_id_fkey FOREIGN KEY (registry_extension_id) REFERENCES registry_extensions(id) ON UPDATE CASCADE ON DELETE CASCADE;

ALTER TABLE ONLY registry_extensions
    ADD CONSTRAINT registry_extensions_publisher_org_id_fkey FOREIGN KEY (publisher_org_id) REFERENCES orgs(id);

ALTER TABLE ONLY registry_extensions
    ADD CONSTRAINT registry_extensions_publisher_user_id_fkey FOREIGN KEY (publisher_user_id) REFERENCES users(id);

ALTER TABLE ONLY saved_searches
    ADD CONSTRAINT saved_searches_org_id_fkey FOREIGN KEY (org_id) REFERENCES orgs(id);

ALTER TABLE ONLY saved_searches
    ADD CONSTRAINT saved_searches_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id);

ALTER TABLE ONLY settings
    ADD CONSTRAINT settings_author_user_id_fkey FOREIGN KEY (author_user_id) REFERENCES users(id) ON DELETE RESTRICT;

ALTER TABLE ONLY settings
    ADD CONSTRAINT settings_references_orgs FOREIGN KEY (org_id) REFERENCES orgs(id) ON DELETE RESTRICT;

ALTER TABLE ONLY settings
    ADD CONSTRAINT settings_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE RESTRICT;

ALTER TABLE ONLY survey_responses
    ADD CONSTRAINT survey_responses_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id);

ALTER TABLE ONLY user_emails
    ADD CONSTRAINT user_emails_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id);

ALTER TABLE ONLY user_external_accounts
    ADD CONSTRAINT user_external_accounts_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id);

COMMIT;
