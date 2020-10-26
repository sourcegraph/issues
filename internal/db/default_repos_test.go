package db

import (
	"context"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/db/dbconn"
	"github.com/sourcegraph/sourcegraph/internal/db/dbtesting"
)

func Test_defaultRepos_List(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	tcs := []struct {
		name  string
		repos []*types.Repo
	}{
		{
			name:  "empty case",
			repos: nil,
		},
		{
			name: "one repo",
			repos: []*types.Repo{
				{
					ID:   api.RepoID(0),
					Name: "github.com/foo/bar",
				},
			},
		},
		{
			name: "a few repos",
			repos: []*types.Repo{
				{
					ID:   api.RepoID(0),
					Name: "github.com/foo/bar",
				},
				{
					ID:   api.RepoID(1),
					Name: "github.com/baz/qux",
				},
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			dbtesting.SetupGlobalTestDB(t)
			ctx := context.Background()
			for _, r := range tc.repos {
				if _, err := dbconn.Global.ExecContext(ctx, `INSERT INTO repo(id, name) VALUES ($1, $2)`, r.ID, r.Name); err != nil {
					t.Fatal(err)
				}
				if _, err := dbconn.Global.ExecContext(ctx, `INSERT INTO default_repos(repo_id) VALUES ($1)`, r.ID); err != nil {
					t.Fatal(err)
				}
			}
			DefaultRepos.resetCache()

			repos, err := DefaultRepos.List(ctx)
			if err != nil {
				t.Fatal(err)
			}

			sort.Sort(types.Repos(repos))
			sort.Sort(types.Repos(tc.repos))
			if diff := cmp.Diff(repos, tc.repos, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}

	t.Run("user-added repos", func(t *testing.T) {
		dbtesting.SetupGlobalTestDB(t)
		ctx := context.Background()
		_, err := dbconn.Global.ExecContext(ctx, `
			-- insert one user-added repo, i.e. a repo added by an external service owned by a user
			INSERT INTO users(id, username) VALUES (1, 'foo');
			INSERT INTO repo(id, name) VALUES (10, 'github.com/foo/bar10');
			INSERT INTO external_services(id, kind, display_name, config, namespace_user_id) VALUES (100, 'github', 'github', '{}', 1);
			INSERT INTO external_service_repos VALUES (100, 10, 'https://github.com/foo/bar10');

			-- insert one repo referenced in the default repo table
			INSERT INTO repo(id, name) VALUES (11, 'github.com/foo/bar11');
			INSERT INTO default_repos(repo_id) VALUES(11);

			-- insert one repo not referenced in the default repo table;
			INSERT INTO repo(id, name) VALUES (12, 'github.com/foo/bar12');
		`)
		if err != nil {
			t.Fatal(err)
		}

		DefaultRepos.resetCache()

		repos, err := DefaultRepos.List(ctx)
		if err != nil {
			t.Fatal(err)
		}

		want := []*types.Repo{
			{
				ID:   api.RepoID(10),
				Name: "github.com/foo/bar10",
			},
			{
				ID:   api.RepoID(11),
				Name: "github.com/foo/bar11",
			},
		}
		// expect 2 repos, the user added repo and the one that is referenced in the default repos table
		if diff := cmp.Diff(want, repos, cmpopts.EquateEmpty()); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
	})
}

func BenchmarkDefaultRepos_List_Empty(b *testing.B) {
	dbtesting.SetupGlobalTestDB(b)
	ctx := context.Background()
	select {
	case <-ctx.Done():
		b.Fatal("context already canceled")
	default:
	}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_, err := DefaultRepos.List(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}
