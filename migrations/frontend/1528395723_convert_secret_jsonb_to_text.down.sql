BEGIN;
--- Note: This rollback will fail if encryption has been enabled since the base64 value will not be valid jsonb
ALTER TABLE user_external_accounts ALTER COLUMN auth_data TYPE JSONB USING auth_data::JSON;
ALTER TABLE user_external_accounts ALTER COLUMN account_data TYPE JSONB USING account_data::JSON;

COMMIT;
