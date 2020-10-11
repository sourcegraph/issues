package db

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/internal/db/dbconn"
)

// defaultReposMaxAge is how long we cache the list of default repos. The list
// changes very rarely, so we can cache for a while.
const defaultReposMaxAge = time.Minute

type cachedRepos struct {
	repos   []*types.Repo
	fetched time.Time
}

func (c *cachedRepos) Repos() []*types.Repo {
	if c == nil || time.Since(c.fetched) > defaultReposMaxAge {
		return nil
	}
	return append([]*types.Repo{}, c.repos...)
}

type defaultRepos struct {
	cache atomic.Value
}

func (s *defaultRepos) List(ctx context.Context) (results []*types.Repo, err error) {
	cached, _ := s.cache.Load().(*cachedRepos)
	if repos := cached.Repos(); repos != nil {
		return repos, nil
	}

	const q = `
-- source: internal/db/default_repos.go:defaultRepos.List
SELECT DISTINCT ON(repo.id) repo.id, repo.name
FROM repo
JOIN LATERAL (
  SELECT esr.repo_id
  FROM external_service_repos esr
  JOIN external_services es ON es.id = esr.external_service_id
  WHERE
    es.namespace_user_id IS NOT NULL AND
    es.deleted_at IS NULL
  UNION ALL
  SELECT repo_id
  FROM default_repos
) repo_ids ON repo.id = repo_ids.repo_id
WHERE
  repo.deleted_at IS NULL;
`
	rows, err := dbconn.Global.QueryContext(ctx, q)
	if err != nil {
		return nil, errors.Wrap(err, "querying default_repos table")
	}
	defer rows.Close()
	var repos []*types.Repo
	for rows.Next() {
		var r types.Repo
		if err := rows.Scan(&r.ID, &r.Name); err != nil {
			return nil, errors.Wrap(err, "scanning row from default_repos table")
		}
		repos = append(repos, &r)
	}
	if err = rows.Err(); err != nil {
		return nil, errors.Wrap(err, "scanning rows from default_repos table")
	}

	s.cache.Store(&cachedRepos{
		// Copy since repos will be mutated by the caller
		repos:   append([]*types.Repo{}, repos...),
		fetched: time.Now(),
	})

	return repos, nil
}

func (s *defaultRepos) resetCache() {
	s.cache.Store(&cachedRepos{})
}
