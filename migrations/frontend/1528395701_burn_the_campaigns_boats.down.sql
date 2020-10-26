BEGIN;

-- Refer to the up migration for full details on what we're doing: we're just
-- doing all of that in reverse.

ALTER TABLE campaign_specs
    DROP CONSTRAINT IF EXISTS campaign_specs_user_id_fkey,
    ADD CONSTRAINT campaign_specs_user_id_fkey
        FOREIGN KEY (user_id)
        REFERENCES users (id)
        DEFERRABLE,
    ALTER COLUMN user_id SET NOT NULL;

ALTER TABLE campaigns
    DROP CONSTRAINT IF EXISTS campaigns_initial_applier_id_fkey,
    DROP CONSTRAINT IF EXISTS campaigns_last_applier_id_fkey,
    ADD CONSTRAINT campaigns_author_id_fkey
        FOREIGN KEY (initial_applier_id)
        REFERENCES users (id)
        ON DELETE CASCADE
        DEFERRABLE,
    ADD CONSTRAINT campaigns_last_applier_id_fkey
        FOREIGN KEY (last_applier_id)
        REFERENCES users (id)
        DEFERRABLE;

ALTER TABLE campaigns
    ALTER COLUMN last_applied_at DROP NOT NULL,
    ALTER COLUMN campaign_spec_id DROP NOT NULL,
    ALTER COLUMN initial_applier_id SET NOT NULL;

-- The ON CONFLICT clause here is because changesets can be partially migrated:
-- if a changeset is attached to multiple campaigns and the repo-updater
-- migrator has already run, we may have a matching external changeset in the
-- changesets table that would violate the unique key on (repo_id,
-- external_id). Since we have a changeset, that's OK, and we can just ignore
-- those records.
INSERT INTO
    changesets
    (
        id,
        campaign_ids,
        repo_id,
        created_at,
        updated_at,
        metadata,
        external_id,
        external_service_type,
        external_deleted_at,
        external_branch,
        external_updated_at,
        external_state,
        external_review_state,
        external_check_state,
        created_by_campaign,
        added_to_campaign,
        diff_stat_added,
        diff_stat_changed,
        diff_stat_deleted,
        sync_state,
        current_spec_id,
        previous_spec_id,
        publication_state,
        owned_by_campaign_id,
        reconciler_state,
        failure_message,
        started_at,
        finished_at,
        process_after,
        num_resets
    )
SELECT
    id,
    campaign_ids,
    repo_id,
    created_at,
    updated_at,
    metadata,
    external_id,
    external_service_type,
    external_deleted_at,
    external_branch,
    external_updated_at,
    external_state,
    external_review_state,
    external_check_state,
    created_by_campaign,
    added_to_campaign,
    diff_stat_added,
    diff_stat_changed,
    diff_stat_deleted,
    sync_state,
    current_spec_id,
    previous_spec_id,
    publication_state,
    owned_by_campaign_id,
    reconciler_state,
    failure_message,
    started_at,
    finished_at,
    process_after,
    num_resets
FROM
    changesets_old
ON CONFLICT
    DO NOTHING;

INSERT INTO
    campaigns
SELECT
    *
FROM
    campaigns_old;

DROP TABLE IF EXISTS changesets_old;
DROP TABLE IF EXISTS campaigns_old;

COMMIT;
