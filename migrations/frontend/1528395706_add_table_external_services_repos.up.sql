BEGIN;

CREATE TABLE IF NOT EXISTS external_service_repos (
    external_service_id bigint NOT NULL,
    repo_id integer NOT NULL,
    clone_url text NOT NULL
);

-- Lock the repo table before running the migration.
LOCK TABLE ONLY repo IN EXCLUSIVE MODE;


-- Mark the sources columns as read-only using a trigger.
CREATE FUNCTION make_repo_sources_column_read_only() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
    BEGIN
        IF (OLD.sources != NEW.sources) THEN
            RAISE EXCEPTION 'sources is read-only';
        END IF;

        RETURN OLD;
    END;
$$;
CREATE TRIGGER trig_read_only_repo_sources_column BEFORE UPDATE OF sources ON repo FOR EACH ROW EXECUTE PROCEDURE make_repo_sources_column_read_only();

-- Migrate repo.sources column content to the external_service_repos table.
-- Each repo.sources value is a jsonb containing one or more source.
-- Each source must be extracted as a single row in the external_service_repos table.

DO $$
DECLARE
   _key   text;
   _value text;
   _repo_id integer;
   _sources jsonb;
BEGIN
    FOR _repo_id, _sources IN
        SELECT id, sources FROM repo WHERE deleted_at IS NULL
    LOOP
        FOR _key, _value IN
            SELECT * FROM jsonb_each_text(_sources)
        LOOP
            INSERT INTO external_service_repos (external_service_id, repo_id, clone_url)
            VALUES (
                split_part((_value::jsonb->'ID'#>>'{}')::text, ':', 3)::bigint,
                _repo_id,
                _value::jsonb->'CloneURL'#>>'{}'
            );
        END LOOP;
    END LOOP;
END$$;

ALTER TABLE ONLY external_service_repos
    ADD CONSTRAINT external_service_repos_external_service_id_fkey FOREIGN KEY (external_service_id) REFERENCES external_services(id) ON DELETE CASCADE DEFERRABLE;

ALTER TABLE ONLY external_service_repos
    ADD CONSTRAINT external_service_repos_repo_id_fkey FOREIGN KEY (repo_id) REFERENCES repo(id) ON DELETE CASCADE DEFERRABLE;

CREATE FUNCTION delete_repo_ref_on_external_service_repos() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
    BEGIN
        -- if a repo is soft-deleted, delete every row that references that repo
        IF (OLD.deleted_at IS NULL AND NEW.deleted_at IS NOT NULL) THEN
        DELETE FROM
            external_service_repos
        WHERE
            repo_id = OLD.id;
        END IF;

        RETURN OLD;
    END;
$$;

CREATE FUNCTION delete_external_service_ref_on_external_service_repos() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
    BEGIN
        -- if an external service is soft-deleted, delete every row that references it
        IF (OLD.deleted_at IS NULL AND NEW.deleted_at IS NOT NULL) THEN
          DELETE FROM
            external_service_repos
          WHERE
            external_service_id = OLD.id;
        END IF;

        RETURN OLD;
    END;
$$;

CREATE TRIGGER trig_delete_repo_ref_on_external_service_repos AFTER UPDATE OF deleted_at ON repo FOR EACH ROW EXECUTE PROCEDURE delete_repo_ref_on_external_service_repos();
CREATE TRIGGER trig_delete_external_service_ref_on_external_service_repos AFTER UPDATE OF deleted_at ON external_services FOR EACH ROW EXECUTE PROCEDURE delete_external_service_ref_on_external_service_repos();

COMMIT;
