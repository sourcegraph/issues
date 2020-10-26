BEGIN;

CREATE INDEX IF NOT EXISTS settings_org_id_idx ON settings(org_id);

COMMIT;
