BEGIN;

-- Undoes the bad migration 1528395696_repo_name_index.up.sql in deployments that had it rolled out before
-- v3.19.1 was released (primarily sourcegraph.com, k8s.sgdev.org, and other internal deployments - but
-- also to keep consistency in other deployments that may have ran v3.19.0 which had this bug like server
-- deployments.)
DROP INDEX IF EXISTS repo_name_idx;

COMMIT;
