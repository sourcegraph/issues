BEGIN;

ALTER TABLE repo
  ADD COLUMN IF NOT EXISTS sources jsonb DEFAULT '{}'::jsonb NOT NULL;

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

COMMIT;
