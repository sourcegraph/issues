BEGIN;
-- Add default values for git commit author (name and email)
UPDATE changeset_specs
SET spec = spec || json_build_object(
        'commits',
        json_build_array(
            spec->'commits'->0 || '{"authorName": "Sourcegraph", "authorEmail": "campaigns@sourcegraph.com"}'
        )
    )::jsonb
WHERE jsonb_array_length(spec->'commits') > 0;
COMMIT;
