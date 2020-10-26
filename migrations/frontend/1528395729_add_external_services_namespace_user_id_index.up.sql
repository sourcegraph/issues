BEGIN;

CREATE INDEX external_services_namespace_user_id_idx ON external_services (namespace_user_id);

COMMIT;

