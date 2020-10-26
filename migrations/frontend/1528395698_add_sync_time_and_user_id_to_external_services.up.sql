BEGIN;

ALTER TABLE external_services ADD COLUMN last_sync_at timestamp with time zone;
ALTER TABLE external_services ADD COLUMN next_sync_at timestamp with time zone;
ALTER TABLE external_services ADD COLUMN namespace_user_id integer;

ALTER TABLE ONLY external_services
    ADD CONSTRAINT external_services_namepspace_user_id_fkey FOREIGN KEY (namespace_user_id) REFERENCES users(id) ON DELETE CASCADE DEFERRABLE;

COMMIT;
