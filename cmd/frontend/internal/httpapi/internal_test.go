package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/gorilla/mux"
	"github.com/sourcegraph/sourcegraph/internal/api"

	apirouter "github.com/sourcegraph/sourcegraph/cmd/frontend/internal/httpapi/router"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/internal/db"
)

func TestGitServiceHandlers(t *testing.T) {
	m := apirouter.NewInternal(mux.NewRouter())

	gitService := &gitServiceHandler{
		Gitserver: mockAddrForRepo{},
	}
	m.Get(apirouter.GitInfoRefs).Handler(http.HandlerFunc(gitService.serveInfoRefs))
	m.Get(apirouter.GitUploadPack).Handler(http.HandlerFunc(gitService.serveGitUploadPack))

	cases := map[string]string{
		"/git/foo/bar/info/refs?service=git-upload-pack": "http://foo.bar.gitserver/git/foo/bar/info/refs?service=git-upload-pack",
		"/git/foo/bar/git-upload-pack":                   "http://foo.bar.gitserver/git/foo/bar/git-upload-pack",
	}

	for target, want := range cases {
		req := httptest.NewRequest("GET", target, nil)
		w := httptest.NewRecorder()
		m.ServeHTTP(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusTemporaryRedirect {
			body, _ := ioutil.ReadAll(resp.Body)
			t.Errorf("expected redirect for %q, got status %d. Body: %s", target, resp.StatusCode, body)
			continue
		}

		got := resp.Header.Get("Location")
		if got != want {
			t.Errorf("mismatched location for %q:\ngot:  %s\nwant: %s", target, got, want)
		}
	}
}

type mockAddrForRepo struct{}

func (mockAddrForRepo) AddrForRepo(_ context.Context, name api.RepoName) string {
	return strings.ReplaceAll(string(name), "/", ".") + ".gitserver"
}

func TestReposIndex(t *testing.T) {
	defaultRepos := []string{"github.com/popular/foo", "github.com/popular/bar"}
	allRepos := append(defaultRepos, "github.com/alice/foo", "github.com/alice/bar")

	cases := []struct {
		name string
		srv  *reposListServer
		body string
		want []string
	}{{
		name: "indexers",
		srv: &reposListServer{
			Repos: &mockRepos{
				defaultRepos: defaultRepos,
				repos:        allRepos,
			},
			Indexers: suffixIndexers(true),
		},
		body: `{"Hostname": "foo"}`,
		want: []string{"github.com/popular/foo", "github.com/alice/foo"},
	}, {
		name: "indexers",
		srv: &reposListServer{
			Repos: &mockRepos{
				defaultRepos: defaultRepos,
				repos:        allRepos,
			},
			Indexers: suffixIndexers(true),
		},
		body: `{"Hostname": "foo", "Indexed": ["github.com/alice/bar"]}`,
		want: []string{"github.com/popular/foo", "github.com/alice/foo", "github.com/alice/bar"},
	}, {
		name: "dot-com indexers",
		srv: &reposListServer{
			SourcegraphDotComMode: true,
			Repos: &mockRepos{
				defaultRepos: defaultRepos,
				repos:        allRepos,
			},
			Indexers: suffixIndexers(true),
		},
		body: `{"Hostname": "foo"}`,
		want: []string{"github.com/popular/foo"},
	}, {
		name: "none",
		srv: &reposListServer{
			Repos: &mockRepos{
				defaultRepos: defaultRepos,
				repos:        allRepos,
			},
			Indexers: suffixIndexers(true),
		},
		body: `{"Hostname": "baz"}`,
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/", bytes.NewReader([]byte(tc.body)))
			w := httptest.NewRecorder()
			if err := tc.srv.serveIndex(w, req); err != nil {
				t.Fatal(err)
			}

			resp := w.Result()
			body, _ := ioutil.ReadAll(resp.Body)

			if resp.StatusCode != http.StatusOK {
				t.Errorf("got status %v", resp.StatusCode)
			}

			var data struct {
				RepoNames []string
			}
			if err := json.Unmarshal(body, &data); err != nil {
				t.Fatal(err)
			}
			got := data.RepoNames

			if !cmp.Equal(tc.want, got) {
				t.Fatalf("mismatch (-want +got):\n%s", cmp.Diff(tc.want, got))
			}
		})
	}
}

type mockRepos struct {
	defaultRepos []string
	repos        []string
}

func (r *mockRepos) ListDefault(context.Context) ([]*types.Repo, error) {
	var repos []*types.Repo
	for _, name := range r.defaultRepos {
		repos = append(repos, &types.Repo{
			Name: api.RepoName(name),
		})
	}
	return repos, nil
}

func (r *mockRepos) List(ctx context.Context, opt db.ReposListOptions) ([]*types.Repo, error) {
	if opt.Index == nil || !*opt.Index {
		return nil, errors.New("reposList test expects Index=true options")
	}

	var repos []*types.Repo
	for _, name := range r.repos {
		repos = append(repos, &types.Repo{
			Name: api.RepoName(name),
		})
	}
	return repos, nil
}

// suffixIndexers mocks Indexers. ReposSubset will return all repoNames with
// the suffix of hostname.
type suffixIndexers bool

func (b suffixIndexers) ReposSubset(ctx context.Context, hostname string, indexed map[string]struct{}, repoNames []string) ([]string, error) {
	if !b.Enabled() {
		return nil, errors.New("indexers disabled")
	}
	if hostname == "" {
		return nil, errors.New("empty hostname")
	}

	var filter []string
	for _, name := range repoNames {
		if strings.HasSuffix(name, hostname) {
			filter = append(filter, name)
		} else if _, ok := indexed[name]; ok {
			filter = append(filter, name)
		}
	}
	return filter, nil
}

func (b suffixIndexers) Enabled() bool {
	return bool(b)
}
