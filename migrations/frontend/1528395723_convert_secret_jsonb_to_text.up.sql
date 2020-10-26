BEGIN;

ALTER TABLE IF EXISTS user_external_accounts ALTER COLUMN auth_data TYPE TEXT;
ALTER TABLE IF EXISTS user_external_accounts ALTER COLUMN account_data TYPE TEXT;

COMMIT;
