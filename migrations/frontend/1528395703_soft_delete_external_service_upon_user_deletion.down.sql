BEGIN;

DROP TRIGGER IF EXISTS trig_soft_delete_user_reference_on_external_service ON users;
DROP FUNCTION IF EXISTS soft_delete_user_reference_on_external_service();

COMMIT;
