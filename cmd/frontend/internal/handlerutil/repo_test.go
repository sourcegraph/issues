package handlerutil

import (
	"context"
	"testing"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/internal/api"
)

func TestGetRepo(t *testing.T) {
	t.Run("URLMovedError", func(t *testing.T) {
		backend.Mocks.Repos.GetByName = func(ctx context.Context, name api.RepoName) (*types.Repo, error) {
			return &types.Repo{Name: name + name}, nil
		}
		t.Cleanup(func() {
			backend.Mocks.Repos = backend.MockRepos{}
		})

		_, err := GetRepo(context.Background(), map[string]string{"Repo": "repo1"})
		if _, ok := err.(*URLMovedError); !ok {
			t.Fatalf("err: want type *URLMovedError but got %T", err)
		}
	})
}
