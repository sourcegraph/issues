package testing

import (
	"context"
	test "testing"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/authz"
	"github.com/sourcegraph/sourcegraph/internal/db"
)

// AuthzFilterRepos sets up a mock for the authzFilter in the db package that
// filters out the repositories with the given IDs.
func AuthzFilterRepos(t *test.T, ids ...api.RepoID) {
	toFilter := map[api.RepoID]struct{}{}
	for _, id := range ids {
		toFilter[id] = struct{}{}
	}
	db.MockAuthzFilter = func(ctx context.Context, repos []*types.Repo, p authz.Perms) ([]*types.Repo, error) {
		var result []*types.Repo
		for _, r := range repos {
			if _, ok := toFilter[r.ID]; ok {
				continue
			}
			result = append(result, r)
		}
		return result, nil
	}
	t.Cleanup(func() { db.MockAuthzFilter = nil })
}
