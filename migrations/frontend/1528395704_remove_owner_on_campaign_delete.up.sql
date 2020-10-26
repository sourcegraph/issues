BEGIN;

ALTER TABLE changesets
    DROP CONSTRAINT IF EXISTS changesets_owned_by_campaign_id_fkey,
    ADD CONSTRAINT changesets_owned_by_campaign_id_fkey
        FOREIGN KEY (owned_by_campaign_id)
        REFERENCES campaigns (id)
        ON DELETE SET NULL
        DEFERRABLE;

COMMIT;
