package graphqlbackend

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/graph-gophers/graphql-go/gqltesting"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/db"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/vcs/git"
)

const exampleCommitSHA1 = "1234567890123456789012345678901234567890"

func TestRepository_Commit(t *testing.T) {
	resetMocks()
	db.Mocks.Repos.MockGetByName(t, "github.com/gorilla/mux", 2)
	backend.Mocks.Repos.ResolveRev = func(ctx context.Context, repo *types.Repo, rev string) (api.CommitID, error) {
		if repo.ID != 2 || rev != "abc" {
			t.Error("wrong arguments to ResolveRev")
		}
		return exampleCommitSHA1, nil
	}
	backend.Mocks.Repos.MockGetCommit_Return_NoCheck(t, &git.Commit{ID: exampleCommitSHA1})

	gqltesting.RunTests(t, []*gqltesting.Test{
		{
			Schema: mustParseGraphQLSchema(t),
			Query: `
				{
					repository(name: "github.com/gorilla/mux") {
						commit(rev: "abc") {
							oid
						}
					}
				}
			`,
			ExpectedResult: `
				{
					"repository": {
						"commit": {
							"oid": "` + exampleCommitSHA1 + `"
						}
					}
				}
			`,
		},
	})
}

func TestRepositoryHydration(t *testing.T) {
	makeRepos := func() (*types.Repo, *types.Repo) {
		const id = 42
		name := fmt.Sprintf("repo-%d", id)

		minimal := types.Repo{
			ID:   api.RepoID(id),
			Name: api.RepoName(name),
			ExternalRepo: api.ExternalRepoSpec{
				ID:          name,
				ServiceType: extsvc.TypeGitHub,
				ServiceID:   "https://github.com",
			},
		}

		hydrated := minimal
		hydrated.RepoFields = &types.RepoFields{
			URI:         fmt.Sprintf("github.com/foobar/%s", name),
			Description: "This is a description of a repository",
			Fork:        false,
		}

		return &minimal, &hydrated
	}

	ctx := context.Background()

	t.Run("hydrated without errors", func(t *testing.T) {
		minimalRepo, hydratedRepo := makeRepos()
		db.Mocks.Repos.Get = func(ctx context.Context, id api.RepoID) (*types.Repo, error) {
			return hydratedRepo, nil
		}
		defer func() { db.Mocks = db.MockStores{} }()

		repoResolver := &RepositoryResolver{repo: minimalRepo}
		assertRepoResolverHydrated(ctx, t, repoResolver, hydratedRepo)
	})

	t.Run("hydration results in errors", func(t *testing.T) {
		minimalRepo, _ := makeRepos()

		dbErr := errors.New("cannot load repo")

		db.Mocks.Repos.Get = func(ctx context.Context, id api.RepoID) (*types.Repo, error) {
			return nil, dbErr
		}
		defer func() { db.Mocks = db.MockStores{} }()

		repoResolver := &RepositoryResolver{repo: minimalRepo}
		_, err := repoResolver.Description(ctx)
		if err == nil {
			t.Fatal("err is unexpected nil")
		}

		if err != dbErr {
			t.Fatalf("wrong err. want=%q, have=%q", dbErr, err)
		}

		// Another call to make sure err does not disappear
		_, err = repoResolver.URI(ctx)
		if err == nil {
			t.Fatal("err is unexpected nil")
		}

		if err != dbErr {
			t.Fatalf("wrong err. want=%q, have=%q", dbErr, err)
		}
	})
}

func assertRepoResolverHydrated(ctx context.Context, t *testing.T, r *RepositoryResolver, hydrated *types.Repo) {
	t.Helper()

	description, err := r.Description(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if description != hydrated.Description {
		t.Fatalf("wrong Description. want=%q, have=%q", hydrated.Description, description)
	}

	uri, err := r.URI(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if uri != hydrated.URI {
		t.Fatalf("wrong URI. want=%q, have=%q", hydrated.URI, uri)
	}
}
