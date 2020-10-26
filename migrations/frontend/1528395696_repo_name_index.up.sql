-- Note: CREATE INDEX CONCURRENTLY cannot run inside a transaction block

--- Aug 20, 2020: This migration was redacted as it caused upgrade deadlocks in v3.19.0. See also 1528395705_remove_bad_migration.up.sql
--- CREATE INDEX CONCURRENTLY IF NOT EXISTS repo_name_idx ON public.repo USING btree (lower(name::text) COLLATE pg_catalog."C");
