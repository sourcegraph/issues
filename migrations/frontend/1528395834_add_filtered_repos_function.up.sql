BEGIN;

CREATE OR REPLACE FUNCTION filtered_repos(bypass BOOLEAN, usePermissionsUserMapping BOOLEAN, userID INT, perm TEXT)
	RETURNS SETOF public.repo
AS
$BODY$
	SELECT * FROM repo
	WHERE (
			$1                            -- TRUE or FALSE to indicate whether to bypass the check
		OR  (
			NOT $2                        -- Disregard unrestricted state when permissions user mapping is enabled
			AND (
				NOT repo.private          -- Happy path of non-private repositories
				OR  EXISTS (              -- Each external service defines if repositories are unrestricted
					SELECT
					FROM external_services AS es
					JOIN external_service_repos AS esr ON (
							esr.external_service_id = es.id
						AND esr.repo_id = repo.id
						AND es.unrestricted = TRUE
						AND es.deleted_at IS NULL
					)
					LIMIT 1
				)
			)
		)
		OR EXISTS (                       -- We assume that all repos added by the authenticated user should be shown
			SELECT 1
			FROM external_service_repos
			WHERE repo_id = repo.id
			AND user_id = $3
		)
		OR (                              -- Restricted repositories require checking permissions
			SELECT object_ids_ints @> INTSET(repo.id)
			FROM user_permissions
			WHERE
				user_id = $3
			AND permission = $4
			AND object_type = 'repos'
		)
	)
$BODY$
LANGUAGE SQL;

COMMIT;
