package graphqlbackend

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/db"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/vcs/git"
)

func TestGitTreeEntry_RawZipArchiveURL(t *testing.T) {
	got := (&GitTreeEntryResolver{
		commit: &GitCommitResolver{
			repoResolver: &RepositoryResolver{
				repo: &types.Repo{Name: "my/repo"},
			},
		},
		stat: CreateFileInfo("a/b", true),
	}).RawZipArchiveURL()
	want := "http://example.com/my/repo/-/raw/a/b?format=zip"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestGitTreeEntry_Content(t *testing.T) {
	wantPath := "foobar.md"
	wantContent := "foobar"

	git.Mocks.ReadFile = func(commit api.CommitID, name string) ([]byte, error) {
		if name != wantPath {
			t.Fatalf("wrong name in ReadFile call. want=%q, have=%q", wantPath, name)
		}
		return []byte(wantContent), nil
	}
	t.Cleanup(func() { git.Mocks.ReadFile = nil })

	gitTree := &GitTreeEntryResolver{
		commit: &GitCommitResolver{
			repoResolver: &RepositoryResolver{
				repo: &types.Repo{Name: "my/repo"},
			},
		},
		stat: CreateFileInfo(wantPath, true),
	}

	newFileContent, err := gitTree.Content(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(newFileContent, wantContent); diff != "" {
		t.Fatalf("wrong newFileContent: %s", diff)
	}

	newByteSize, err := gitTree.ByteSize(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if have, want := newByteSize, int32(len([]byte(wantContent))); have != want {
		t.Fatalf("wrong file size, want=%d have=%d", want, have)
	}
}

func TestReposourceCloneURLToRepoName(t *testing.T) {
	ctx := context.Background()

	db.Mocks.ExternalServices.List = func(db.ExternalServicesListOptions) ([]*types.ExternalService, error) {
		return []*types.ExternalService{
			{
				ID:          1,
				Kind:        extsvc.KindGitHub,
				DisplayName: "GITHUB #1",
				Config:      `{"url": "https://github.example.com", "repositoryQuery": ["none"], "token": "abc"}`,
			},
		}, nil
	}
	defer func() { db.Mocks.ExternalServices = db.MockExternalServices{} }()

	tests := []struct {
		name         string
		cloneURL     string
		wantRepoName api.RepoName
	}{
		{
			name:     "no match",
			cloneURL: "https://gitlab.com/user/repo",
		},
		{
			name:         "match existing external service",
			cloneURL:     "https://github.example.com/user/repo.git",
			wantRepoName: api.RepoName("github.example.com/user/repo"),
		},
		{
			name:         "fallback for github.com",
			cloneURL:     "https://github.com/user/repo",
			wantRepoName: api.RepoName("github.com/user/repo"),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repoName, err := reposourceCloneURLToRepoName(ctx, test.cloneURL)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(test.wantRepoName, repoName); diff != "" {
				t.Fatalf("RepoName mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
