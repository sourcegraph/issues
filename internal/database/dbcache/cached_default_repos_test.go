package dbcache

import (
	"context"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/database/dbtest"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/types"
)

func TestListDefaultRepos(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	tcs := []struct {
		name  string
		repos []types.RepoName
	}{
		{
			name:  "empty case",
			repos: nil,
		},
		{
			name: "one repo",
			repos: []types.RepoName{
				{
					ID:   api.RepoID(1),
					Name: "github.com/foo/bar",
				},
			},
		},
		{
			name: "two repos",
			repos: []types.RepoName{
				{
					ID:   api.RepoID(1),
					Name: "github.com/foo/bar",
				},
				{
					ID:   api.RepoID(2),
					Name: "github.com/baz/qux",
				},
			},
		},
	}

	createExternalService := func(ctx context.Context, db dbutil.DB) *types.ExternalService {
		confGet := func() *conf.Unified {
			return &conf.Unified{}
		}
		es := &types.ExternalService{
			Kind:        extsvc.KindGitHub,
			DisplayName: "GITHUB #1",
			Config:      `{"url": "https://github.com", "repositoryQuery": ["none"], "token": "abc"}`,
		}
		err := database.ExternalServices(db).Create(ctx, confGet, es)
		if err != nil {
			t.Fatal(err)
		}
		return es
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			db := dbtest.NewDB(t, "")
			ctx := context.Background()

			es := createExternalService(ctx, db)

			for _, r := range tc.repos {
				if _, err := db.ExecContext(ctx, `INSERT INTO repo(id, name) VALUES ($1, $2)`, r.ID, r.Name); err != nil {
					t.Fatal(err)
				}
				if _, err := db.ExecContext(ctx, `INSERT INTO default_repos(repo_id) VALUES ($1)`, r.ID); err != nil {
					t.Fatal(err)
				}
				if _, err := db.ExecContext(ctx, `INSERT INTO external_service_repos(repo_id, external_service_id, clone_url) VALUES ($1, $2, 'example.com')`, r.ID, es.ID); err != nil {
					t.Fatal(err)
				}

				if _, err := db.ExecContext(ctx, `INSERT INTO gitserver_repos(repo_id, clone_status, shard_id) VALUES ($1, $2, 'test');`, r.ID, types.CloneStatusCloned); err != nil {
					t.Fatal(err)
				}
			}

			repos, err := NewDefaultRepoLister(database.Repos(db)).List(ctx)
			if err != nil {
				t.Fatal(err)
			}

			sort.Sort(types.RepoNames(repos))
			sort.Sort(types.RepoNames(tc.repos))
			if diff := cmp.Diff(tc.repos, repos, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}

	t.Run("user-added repos", func(t *testing.T) {
		db := dbtest.NewDB(t, "")
		ctx := context.Background()

		es := createExternalService(ctx, db)
		if es.ID != 1 {
			// Since we depend on this in the test below
			t.Fatal("id should be 1")
		}
		_, err := db.ExecContext(ctx, `
			-- insert one public user-added repo, i.e. a repo added by an external service owned by a user
			INSERT INTO users(id, username) VALUES (1, 'foo');
			INSERT INTO repo(id, name, stars) VALUES (10, 'github.com/foo/bar10', 5);
			INSERT INTO external_services(id, kind, display_name, config, namespace_user_id) VALUES (100, 'github', 'github', '{}', 1);
			INSERT INTO external_service_repos(repo_id, external_service_id, clone_url, user_id) VALUES (10, 100, '', 1);

			-- insert one repo referenced in the default repo table
			INSERT INTO repo(id, name, stars) VALUES (11, 'github.com/foo/bar11', 4);
			INSERT INTO default_repos(repo_id) VALUES(11);
			INSERT INTO external_service_repos(repo_id, external_service_id, clone_url, user_id) VALUES (11, 1, '', NULL);

			-- insert a repo only referenced by a cloud_default external service
			INSERT INTO repo(id, name, stars) VALUES (13, 'github.com/foo/bar13', 3);
			INSERT INTO external_services(id, kind, display_name, config, cloud_default) VALUES (101, 'github', 'github', '{}', true);
			INSERT INTO external_service_repos(repo_id, external_service_id, clone_url, user_id) VALUES (13, 101, 'https://github.com/foo/bar13', NULL);

			-- insert a repo only referenced by a cloud_default external service, but also in user_public_repos
			INSERT INTO repo(id, name, stars) VALUES (14, 'github.com/foo/bar14', 2);
			INSERT INTO external_service_repos(repo_id, external_service_id, clone_url, user_id) VALUES (14, 101, 'https://github.com/foo/bar14', NULL);
			INSERT INTO user_public_repos(user_id, repo_id, repo_uri) VALUES (1, 14, 'github.com/foo/bar/14');

			-- insert one private user-added repo, i.e. a repo added by an external service owned by a user
			INSERT INTO repo(id, name, private) VALUES (15, 'github.com/foo/bar15', true);
			INSERT INTO external_service_repos(repo_id, external_service_id, clone_url, user_id) VALUES (15, 100, 'example.com', 1);
		`)
		if err != nil {
			t.Fatal(err)
		}

		t.Run("List ALL repos", func(t *testing.T) {
			repos, err := NewDefaultRepoLister(database.Repos(db)).List(ctx)
			if err != nil {
				t.Fatal(err)
			}
			want := []types.RepoName{
				{
					ID:   api.RepoID(10),
					Name: "github.com/foo/bar10",
				},
				{
					ID:   api.RepoID(11),
					Name: "github.com/foo/bar11",
				},
				{
					ID:   api.RepoID(14),
					Name: "github.com/foo/bar14",
				},
				{
					ID:   api.RepoID(15),
					Name: "github.com/foo/bar15",
				},
			}
			if diff := cmp.Diff(want, repos, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("List only public default repos", func(t *testing.T) {
			repos, err := NewDefaultRepoLister(database.Repos(db)).ListPublic(ctx)
			if err != nil {
				t.Fatal(err)
			}
			want := []types.RepoName{
				{
					ID:   api.RepoID(10),
					Name: "github.com/foo/bar10",
				},
				{
					ID:   api.RepoID(11),
					Name: "github.com/foo/bar11",
				},
				{
					ID:   api.RepoID(14),
					Name: "github.com/foo/bar14",
				},
			}
			if diff := cmp.Diff(want, repos, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	})
}

func BenchmarkDefaultRepos_List_Empty(b *testing.B) {
	db := dbtest.NewDB(b, "")

	ctx := context.Background()
	select {
	case <-ctx.Done():
		b.Fatal("context already canceled")
	default:
	}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_, err := NewDefaultRepoLister(database.Repos(db)).List(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}
