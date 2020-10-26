package externallink

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/db"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/repoupdater"
	"github.com/sourcegraph/sourcegraph/internal/repoupdater/protocol"
	"github.com/sourcegraph/sourcegraph/internal/vcs/git"
)

func TestRepository(t *testing.T) {
	t.Run("repo-updater info with repo.Name", func(t *testing.T) {
		resetMocks()
		repoName := "myrepo"
		repoupdater.MockRepoLookup = func(args protocol.RepoLookupArgs) (*protocol.RepoLookupResult, error) {
			if args.Repo != api.RepoName(repoName) {
				t.Errorf("got %+v, want %+v", args.Repo, repoName)
			}
			return &protocol.RepoLookupResult{
				Repo: &protocol.RepoInfo{
					Links: &protocol.RepoLinks{
						Root: "http://example.com/" + repoName,
					},
				},
			}, nil
		}
		db.Mocks.Phabricator.GetByName = func(repo api.RepoName) (*types.PhabricatorRepo, error) {
			return nil, errors.New("x")
		}
		links, err := Repository(context.Background(), &types.Repo{Name: api.RepoName(repoName)})
		if err != nil {
			t.Fatal(err)
		}
		if want := []*Resolver{
			{
				url:         "http://example.com/" + repoName,
				serviceType: "",
			},
		}; !reflect.DeepEqual(links, want) {
			t.Errorf("got %+v, want %+v", links, want)
		}
	})

	t.Run("phabricator", func(t *testing.T) {
		resetMocks()
		repoupdater.MockRepoLookup = func(protocol.RepoLookupArgs) (*protocol.RepoLookupResult, error) {
			return &protocol.RepoLookupResult{}, nil
		}
		db.Mocks.Phabricator.GetByName = func(repo api.RepoName) (*types.PhabricatorRepo, error) {
			if want := api.RepoName("myrepo"); repo != want {
				t.Errorf("got %q, want %q", repo, want)
			}
			return &types.PhabricatorRepo{URL: "http://phabricator.example.com/", Callsign: "MYREPO"}, nil
		}
		links, err := Repository(context.Background(), &types.Repo{Name: "myrepo"})
		if err != nil {
			t.Fatal(err)
		}
		if want := []*Resolver{
			{
				url:         "http://phabricator.example.com/diffusion/MYREPO",
				serviceType: extsvc.TypePhabricator,
			},
		}; !reflect.DeepEqual(links, want) {
			t.Errorf("got %+v, want %+v", links, want)
		}
	})

	t.Run("errors", func(t *testing.T) {
		resetMocks()
		repoupdater.MockRepoLookup = func(protocol.RepoLookupArgs) (*protocol.RepoLookupResult, error) {
			return nil, errors.New("x")
		}
		db.Mocks.Phabricator.GetByName = func(repo api.RepoName) (*types.PhabricatorRepo, error) {
			return nil, errors.New("x")
		}
		links, err := Repository(context.Background(), &types.Repo{Name: "myrepo"})
		if err != nil {
			t.Fatal(err)
		}
		if want := []*Resolver(nil); !reflect.DeepEqual(links, want) {
			t.Errorf("got %+v, want %+v", links, want)
		}
	})
}

func TestFileOrDir(t *testing.T) {
	const (
		rev  = "myrev"
		path = "mydir/myfile"
	)

	for _, which := range []string{"file", "dir"} {
		var (
			repoLinks protocol.RepoLinks
			isDir     bool
			wantURL   string
		)
		switch which {
		case "file":
			repoLinks = protocol.RepoLinks{Blob: "http://example.com/myrepo@{rev}/file/{path}"}
			isDir = false
			wantURL = "http://example.com/myrepo@myrev/file/mydir/myfile"
		case "dir":
			repoLinks = protocol.RepoLinks{Tree: "http://example.com/myrepo@{rev}/dir/{path}"}
			isDir = true
			wantURL = "http://example.com/myrepo@myrev/dir/mydir/myfile"
		}

		t.Run(which, func(t *testing.T) {
			t.Run("repo-updater info with no repo.ExternalRepo", func(t *testing.T) {
				resetMocks()
				repoName := "myrepo"
				repoupdater.MockRepoLookup = func(args protocol.RepoLookupArgs) (*protocol.RepoLookupResult, error) {
					if args.Repo != api.RepoName(repoName) {
						t.Errorf("got %+v, want %+v", args.Repo, repoName)
					}
					return &protocol.RepoLookupResult{
						Repo: &protocol.RepoInfo{
							Links: &repoLinks,
						},
					}, nil
				}
				db.Mocks.Phabricator.GetByName = func(repo api.RepoName) (*types.PhabricatorRepo, error) {
					return nil, errors.New("x")
				}
				links, err := FileOrDir(context.Background(), &types.Repo{Name: api.RepoName(repoName)}, rev, path, isDir)
				if err != nil {
					t.Fatal(err)
				}
				if want := []*Resolver{
					{
						url:         wantURL,
						serviceType: "",
					},
				}; !reflect.DeepEqual(links, want) {
					t.Errorf("got %+v, want %+v", links, want)
				}
			})
		})
	}

	t.Run("phabricator", func(t *testing.T) {
		resetMocks()
		repoupdater.MockRepoLookup = func(protocol.RepoLookupArgs) (*protocol.RepoLookupResult, error) {
			return &protocol.RepoLookupResult{}, nil
		}
		db.Mocks.Phabricator.GetByName = func(repo api.RepoName) (*types.PhabricatorRepo, error) {
			if want := api.RepoName("myrepo"); repo != want {
				t.Errorf("got %q, want %q", repo, want)
			}
			return &types.PhabricatorRepo{URL: "http://phabricator.example.com/", Callsign: "MYREPO"}, nil
		}
		git.Mocks.ExecSafe = func(params []string) ([]byte, []byte, int, error) {
			return []byte("mybranch"), nil, 0, nil
		}
		defer git.ResetMocks()
		links, err := FileOrDir(context.Background(), &types.Repo{Name: "myrepo"}, rev, path, true)
		if err != nil {
			t.Fatal(err)
		}
		if want := []*Resolver{
			{
				url:         "http://phabricator.example.com/source/MYREPO/browse/mybranch/mydir/myfile;myrev",
				serviceType: extsvc.TypePhabricator,
			},
		}; !reflect.DeepEqual(links, want) {
			t.Errorf("got %+v, want %+v", links, want)
		}
	})

	t.Run("errors", func(t *testing.T) {
		resetMocks()
		repoupdater.MockRepoLookup = func(protocol.RepoLookupArgs) (*protocol.RepoLookupResult, error) {
			return nil, errors.New("x")
		}
		db.Mocks.Phabricator.GetByName = func(repo api.RepoName) (*types.PhabricatorRepo, error) {
			return nil, errors.New("x")
		}
		links, err := FileOrDir(context.Background(), &types.Repo{Name: "myrepo"}, rev, path, true)
		if err != nil {
			t.Fatal(err)
		}
		if want := []*Resolver(nil); !reflect.DeepEqual(links, want) {
			t.Errorf("got %+v, want %+v", links, want)
		}
	})
}

func TestCommit(t *testing.T) {
	const commit = "mycommit"

	t.Run("repo-updater info with no repo.ExternalRepo", func(t *testing.T) {
		resetMocks()
		repoName := "myrepo"
		repoupdater.MockRepoLookup = func(args protocol.RepoLookupArgs) (*protocol.RepoLookupResult, error) {
			if args.Repo != api.RepoName(repoName) {
				t.Errorf("got %+v, want %+v", args.Repo, repoName)
			}
			return &protocol.RepoLookupResult{
				Repo: &protocol.RepoInfo{
					Links: &protocol.RepoLinks{
						Commit: "http://example.com/" + repoName + "/commit/{commit}",
					},
				},
			}, nil
		}
		db.Mocks.Phabricator.GetByName = func(repo api.RepoName) (*types.PhabricatorRepo, error) {
			return nil, errors.New("x")
		}
		links, err := Commit(context.Background(), &types.Repo{Name: api.RepoName(repoName)}, commit)
		if err != nil {
			t.Fatal(err)
		}
		if want := []*Resolver{
			{
				url:         "http://example.com/" + repoName + "/commit/mycommit",
				serviceType: "",
			},
		}; !reflect.DeepEqual(links, want) {
			t.Errorf("got %+v, want %+v", links, want)
		}
	})

	t.Run("phabricator", func(t *testing.T) {
		resetMocks()
		repoupdater.MockRepoLookup = func(protocol.RepoLookupArgs) (*protocol.RepoLookupResult, error) {
			return &protocol.RepoLookupResult{}, nil
		}
		db.Mocks.Phabricator.GetByName = func(repo api.RepoName) (*types.PhabricatorRepo, error) {
			if want := api.RepoName("myrepo"); repo != want {
				t.Errorf("got %q, want %q", repo, want)
			}
			return &types.PhabricatorRepo{URL: "http://phabricator.example.com/", Callsign: "MYREPO"}, nil
		}
		links, err := Commit(context.Background(), &types.Repo{Name: "myrepo"}, commit)
		if err != nil {
			t.Fatal(err)
		}
		if want := []*Resolver{
			{
				url:         "http://phabricator.example.com/rMYREPOmycommit",
				serviceType: extsvc.TypePhabricator,
			},
		}; !reflect.DeepEqual(links, want) {
			t.Errorf("got %+v, want %+v", links, want)
		}
	})

	t.Run("errors", func(t *testing.T) {
		resetMocks()
		repoupdater.MockRepoLookup = func(protocol.RepoLookupArgs) (*protocol.RepoLookupResult, error) {
			return nil, errors.New("x")
		}
		db.Mocks.Phabricator.GetByName = func(repo api.RepoName) (*types.PhabricatorRepo, error) {
			return nil, errors.New("x")
		}
		links, err := Commit(context.Background(), &types.Repo{Name: "myrepo"}, commit)
		if err != nil {
			t.Fatal(err)
		}
		if want := []*Resolver(nil); !reflect.DeepEqual(links, want) {
			t.Errorf("got %+v, want %+v", links, want)
		}
	})
}
