package graphqlbackend

import (
	"context"
	"reflect"
	"testing"

	"github.com/graph-gophers/graphql-go/gqltesting"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/db"
	"github.com/sourcegraph/sourcegraph/internal/gitserver"
	"github.com/sourcegraph/sourcegraph/internal/repoupdater"
	"github.com/sourcegraph/sourcegraph/internal/repoupdater/protocol"
)

func TestCheckMirrorRepositoryConnection(t *testing.T) {
	resetMocks()

	const repoName = "my/repo"

	db.Mocks.Users.GetByCurrentAuthUser = func(context.Context) (*types.User, error) {
		return &types.User{SiteAdmin: true}, nil
	}

	t.Run("repository arg", func(t *testing.T) {
		backend.Mocks.Repos.Get = func(ctx context.Context, repoID api.RepoID) (*types.Repo, error) {
			return &types.Repo{Name: repoName}, nil
		}

		calledRepoLookup := false
		repoupdater.MockRepoLookup = func(args protocol.RepoLookupArgs) (*protocol.RepoLookupResult, error) {
			calledRepoLookup = true
			if args.Repo != repoName {
				t.Errorf("got %q, want %q", args.Repo, repoName)
			}
			return &protocol.RepoLookupResult{
				Repo: &protocol.RepoInfo{Name: repoName, VCS: protocol.VCSInfo{URL: "http://example.com/my/repo"}},
			}, nil
		}
		defer func() { repoupdater.MockRepoLookup = nil }()

		calledIsRepoCloneable := false
		gitserver.MockIsRepoCloneable = func(repo gitserver.Repo) error {
			calledIsRepoCloneable = true
			if want := (gitserver.Repo{Name: repoName, URL: "http://example.com/my/repo"}); !reflect.DeepEqual(repo, want) {
				t.Errorf("got %+v, want %+v", repo, want)
			}
			return nil
		}
		defer func() { gitserver.MockIsRepoCloneable = nil }()

		gqltesting.RunTests(t, []*gqltesting.Test{
			{
				Schema: mustParseGraphQLSchema(t),
				Query: `
				mutation {
					checkMirrorRepositoryConnection(repository: "UmVwb3NpdG9yeToxMjM=") {
					    error
					}
				}
			`,
				ExpectedResult: `
				{
					"checkMirrorRepositoryConnection": {
						"error": null
					}
				}
			`,
			},
		})

		if !calledRepoLookup {
			t.Error("!calledRepoLookup")
		}
		if !calledIsRepoCloneable {
			t.Error("!calledIsRepoCloneable")
		}
	})

	t.Run("name arg", func(t *testing.T) {
		backend.Mocks.Repos.GetByName = func(ctx context.Context, name api.RepoName) (*types.Repo, error) {
			t.Fatal("want GetByName to not be called")
			return nil, nil
		}

		calledRepoLookup := false
		repoupdater.MockRepoLookup = func(args protocol.RepoLookupArgs) (*protocol.RepoLookupResult, error) {
			calledRepoLookup = true
			if args.Repo != repoName {
				t.Errorf("got %q, want %q", args.Repo, repoName)
			}
			return &protocol.RepoLookupResult{
				Repo: &protocol.RepoInfo{Name: repoName, VCS: protocol.VCSInfo{URL: "http://example.com/my/repo"}},
			}, nil
		}
		defer func() { repoupdater.MockRepoLookup = nil }()

		calledIsRepoCloneable := false
		gitserver.MockIsRepoCloneable = func(repo gitserver.Repo) error {
			calledIsRepoCloneable = true
			if want := (gitserver.Repo{Name: repoName, URL: "http://example.com/my/repo"}); !reflect.DeepEqual(repo, want) {
				t.Errorf("got %+v, want %+v", repo, want)
			}
			return nil
		}
		defer func() { gitserver.MockIsRepoCloneable = nil }()

		gqltesting.RunTests(t, []*gqltesting.Test{
			{
				Schema: mustParseGraphQLSchema(t),
				Query: `
				mutation {
					checkMirrorRepositoryConnection(name: "my/repo") {
					    error
					}
				}
			`,
				ExpectedResult: `
				{
					"checkMirrorRepositoryConnection": {
						"error": null
					}
				}
			`,
			},
		})

		if !calledRepoLookup {
			t.Error("!calledRepoLookup")
		}
		if !calledIsRepoCloneable {
			t.Error("!calledIsRepoCloneable")
		}
	})
}

func TestCheckMirrorRepositoryRemoteURL(t *testing.T) {
	resetMocks()

	const repoName = "my/repo"

	cases := []struct {
		repoURL string
		want    string
	}{
		{
			repoURL: "git@github.com:gorilla/mux.git",
			want:    `{"repository":{"mirrorInfo":{"remoteURL":"github.com:gorilla/mux.git"}}}`,
		},
		{
			repoURL: "git+https://github.com/gorilla/mux.git",
			want:    `{"repository":{"mirrorInfo":{"remoteURL":"git+https://github.com/gorilla/mux.git"}}}`,
		},
		{
			repoURL: "https://github.com/gorilla/mux.git",
			want:    `{"repository":{"mirrorInfo":{"remoteURL":"https://github.com/gorilla/mux.git"}}}`,
		},
		{
			repoURL: "https://github.com/gorilla/mux",
			want:    `{"repository":{"mirrorInfo":{"remoteURL":"https://github.com/gorilla/mux"}}}`,
		},
		{
			repoURL: "ssh://git@github.com/gorilla/mux",
			want:    `{"repository":{"mirrorInfo":{"remoteURL":"ssh://github.com/gorilla/mux"}}}`,
		},
		{
			repoURL: "ssh://github.com/gorilla/mux.git",
			want:    `{"repository":{"mirrorInfo":{"remoteURL":"ssh://github.com/gorilla/mux.git"}}}`,
		},
		{
			repoURL: "ssh://git@github.com:/my/repo.git",
			want:    `{"repository":{"mirrorInfo":{"remoteURL":"ssh://github.com:/my/repo.git"}}}`,
		},
		{
			repoURL: "git://git@github.com:/my/repo.git",
			want:    `{"repository":{"mirrorInfo":{"remoteURL":"git://github.com:/my/repo.git"}}}`,
		},
		{
			repoURL: "user@host.xz:/path/to/repo.git/",
			want:    `{"repository":{"mirrorInfo":{"remoteURL":"host.xz:/path/to/repo.git/"}}}`,
		},
		{
			repoURL: "host.xz:/path/to/repo.git/",
			want:    `{"repository":{"mirrorInfo":{"remoteURL":"host.xz:/path/to/repo.git/"}}}`,
		},
		{
			repoURL: "ssh://user@host.xz:1234/path/to/repo.git/",
			want:    `{"repository":{"mirrorInfo":{"remoteURL":"ssh://host.xz:1234/path/to/repo.git/"}}}`,
		},
		{
			repoURL: "host.xz:~user/path/to/repo.git/",
			want:    `{"repository":{"mirrorInfo":{"remoteURL":"host.xz:~user/path/to/repo.git/"}}}`,
		},
		{
			repoURL: "ssh://host.xz/~/path/to/repo.git",
			want:    `{"repository":{"mirrorInfo":{"remoteURL":"ssh://host.xz/~/path/to/repo.git"}}}`,
		},
		{
			repoURL: "git://host.xz/~user/path/to/repo.git/",
			want:    `{"repository":{"mirrorInfo":{"remoteURL":"git://host.xz/~user/path/to/repo.git/"}}}`,
		},
		{
			repoURL: "file:///path/to/repo.git/",
			want:    `{"repository":{"mirrorInfo":{"remoteURL":"file:///path/to/repo.git/"}}}`,
		},
		{
			repoURL: "file://~/path/to/repo.git/",
			want:    `{"repository":{"mirrorInfo":{"remoteURL":"file://~/path/to/repo.git/"}}}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.repoURL, func(t *testing.T) {
			db.Mocks.Users.GetByCurrentAuthUser = func(context.Context) (*types.User, error) {
				return &types.User{SiteAdmin: true}, nil
			}

			backend.Mocks.Repos.GetByName = func(ctx context.Context, name api.RepoName) (*types.Repo, error) {
				return &types.Repo{Name: repoName}, nil
			}

			repoupdater.MockRepoLookup = func(args protocol.RepoLookupArgs) (*protocol.RepoLookupResult, error) {
				return &protocol.RepoLookupResult{
					Repo: &protocol.RepoInfo{Name: repoName, VCS: protocol.VCSInfo{URL: tc.repoURL}},
				}, nil
			}
			defer func() { repoupdater.MockRepoLookup = nil }()

			gqltesting.RunTests(t, []*gqltesting.Test{
				{
					Schema: mustParseGraphQLSchema(t),
					Query: `
					{
						repository(name: "my/repo") {
							mirrorInfo {
								remoteURL
							}
						}
					}
				`,
					ExpectedResult: tc.want,
				},
			})
		})
	}
}
