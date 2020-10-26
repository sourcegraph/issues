package ui

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gorilla/mux"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/envvar"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/globals"
	uirouter "github.com/sourcegraph/sourcegraph/cmd/frontend/internal/app/ui/router"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/internal/siteid"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/db/globalstatedb"
	"github.com/sourcegraph/sourcegraph/internal/gitserver"
	"github.com/sourcegraph/sourcegraph/internal/repoupdater"
	"github.com/sourcegraph/sourcegraph/internal/vcs"
	"github.com/sourcegraph/sourcegraph/internal/vcs/git"
	"github.com/sourcegraph/sourcegraph/internal/vcs/util"
)

func TestRedirects(t *testing.T) {
	check := func(t *testing.T, path string, wantStatusCode int, wantRedirectLocation string, userAgent string) {
		t.Helper()

		rw := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", path, nil)
		req.Header.Set("User-Agent", userAgent)
		uirouter.Router.ServeHTTP(rw, req)
		if rw.Code != wantStatusCode {
			t.Errorf("got HTTP response code %d, want %d", rw.Code, wantStatusCode)
		}
		if got := rw.Header().Get("Location"); got != wantRedirectLocation {
			t.Errorf("got redirect location %q, want %q", got, wantRedirectLocation)
		}
	}

	t.Run("on Sourcegraph.com", func(t *testing.T) {
		orig := envvar.SourcegraphDotComMode()
		envvar.MockSourcegraphDotComMode(true)
		defer envvar.MockSourcegraphDotComMode(orig) // reset
		t.Run("root", func(t *testing.T) {
			check(t, "/", http.StatusTemporaryRedirect, "https://about.sourcegraph.com", "Mozilla/5.0")
		})
	})

	t.Run("on Sourcegraph.com from Cookiebot", func(t *testing.T) {
		orig := envvar.SourcegraphDotComMode()
		envvar.MockSourcegraphDotComMode(true)
		defer envvar.MockSourcegraphDotComMode(orig) // reset
		t.Run("root", func(t *testing.T) {
			check(t, "/", http.StatusTemporaryRedirect, "/search", "Mozilla/5.0 Cookiebot")
		})
	})

	t.Run("non-Sourcegraph.com", func(t *testing.T) {
		orig := envvar.SourcegraphDotComMode()
		envvar.MockSourcegraphDotComMode(false)
		defer envvar.MockSourcegraphDotComMode(orig) // reset
		t.Run("root", func(t *testing.T) {
			check(t, "/", http.StatusTemporaryRedirect, "/search", "Mozilla/5.0")
		})
	})
}

func TestRepoShortName(t *testing.T) {
	tests := []struct {
		input api.RepoName
		want  string
	}{
		{input: "repo", want: "repo"},
		{input: "github.com/foo/bar", want: "foo/bar"},
		{input: "mycompany.com/foo", want: "foo"},
	}
	for _, tst := range tests {
		t.Run(string(tst.input), func(t *testing.T) {
			got := repoShortName(tst.input)
			if got != tst.want {
				t.Fatalf("input %q got %q want %q", tst.input, got, tst.want)
			}
		})
	}
}

func TestNewCommon_repo_error(t *testing.T) {
	cases := []struct {
		name string
		rev  string
		err  error

		want string
		code int
	}{{
		name: "cloning",
		err:  &vcs.RepoNotExistError{CloneInProgress: true},
		code: 200,
	}, {
		name: "repo-404",
		err:  &vcs.RepoNotExistError{Repo: "repo-404"},
		want: "repository does not exist: repo-404",
		code: 404,
	}, {
		name: "rev-404",
		rev:  "@marco",
		err:  &gitserver.RevisionNotFoundError{Repo: "rev-404", Spec: "marco"},
		want: "revision not found: rev-404@marco",
		code: 404,
	}, {
		name: "repoupdater-not-found",
		err:  &repoupdater.ErrNotFound{Repo: "repo-404", IsNotFound: true},
		want: fmt.Sprintf("repository not found (name=%s notfound=%v)", "repo-404", true),
		code: 404,
	}, {
		name: "repoupdater-unauthorized",
		err:  &repoupdater.ErrUnauthorized{Repo: "repo-unauth", NoAuthz: true},
		want: fmt.Sprintf("not authorized (name=%s noauthz=%v)", "repo-unauth", true),
		code: 401,
	}, {
		name: "github.com/sourcegraphtest/Always500Test",
		want: "error caused by Always500Test repo name",
		code: 500,
	}}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			backend.Mocks.Repos.MockGetByName(t, api.RepoName(tt.name), 1)
			backend.Mocks.Repos.MockGet(t, 1)
			backend.Mocks.Repos.ResolveRev = func(context.Context, *types.Repo, string) (api.CommitID, error) {
				if tt.err != nil {
					return "", tt.err
				}
				return api.CommitID("deadbeef"), nil
			}

			req, err := http.NewRequest("GET", "/", nil)
			if err != nil {
				t.Fatal(err)
			}
			req = mux.SetURLVars(req, map[string]string{
				"Repo": tt.name,
				"Rev":  tt.rev,
			})

			code := 200
			got := ""
			serveError := func(w http.ResponseWriter, r *http.Request, err error, statusCode int) {
				got = err.Error()
				code = statusCode
			}

			_, err = newCommon(httptest.NewRecorder(), req, "test", serveError)
			if err != nil {
				if got != "" || code != 200 {
					t.Fatal("serveError called and error returned from newCommon")
				}
				code = 500
				got = err.Error()
			}

			if tt.want != got {
				t.Errorf("unexpected error.\ngot:  %s\nwant: %s", got, tt.want)
			}
			if tt.code != code {
				t.Errorf("unexpected status code: got=%d want=%d", code, tt.code)
			}
		})
	}
}

func TestRedirectTreeOrBlob(t *testing.T) {
	tests := []struct {
		name          string
		route         string
		path          string
		common        *Common
		mockStat      os.FileInfo
		expHandled    bool
		expStatusCode int
		expLocation   string
	}{
		{
			name:          "empty commit ID, no redirect",
			common:        &Common{},
			expStatusCode: http.StatusOK,
		},
		{
			name:  "empty path, no redirect",
			route: routeRepo,
			path:  "",
			common: &Common{
				Repo: &types.Repo{
					Name: "github.com/user/repo",
				},
				CommitID: "eca7e807356b887ee24b7a7497973bbfc5688dac",
			},
			expStatusCode: http.StatusOK,
		},
		{
			name:  "root path, no redirect",
			route: routeRepo,
			path:  "/",
			common: &Common{
				Repo: &types.Repo{
					Name: "github.com/user/repo",
				},
				CommitID: "eca7e807356b887ee24b7a7497973bbfc5688dac",
			},
			expStatusCode: http.StatusOK,
		},
		{
			name:  "view tree, no redirect",
			route: routeTree,
			path:  "/some/dir",
			common: &Common{
				Repo: &types.Repo{
					Name: "github.com/user/repo",
				},
				CommitID: "eca7e807356b887ee24b7a7497973bbfc5688dac",
			},
			mockStat:      &util.FileInfo{Mode_: os.ModeDir},
			expStatusCode: http.StatusOK,
		},
		{
			name:  "view blob, no redirect",
			route: routeBlob,
			path:  "/some/file.go",
			common: &Common{
				Repo: &types.Repo{
					Name: "github.com/user/repo",
				},
				CommitID: "eca7e807356b887ee24b7a7497973bbfc5688dac",
			},
			mockStat:      &util.FileInfo{}, // Not a directory
			expStatusCode: http.StatusOK,
		},

		// "/github.com/user/repo/-/tree/some/file.go" -> "/github.com/user/repo/-/blob/some/file.go"
		{
			name:  "redirct tree to blob",
			route: routeTree,
			path:  "/some/file.go",
			common: &Common{
				Repo: &types.Repo{
					Name: "github.com/user/repo",
				},
				CommitID: "eca7e807356b887ee24b7a7497973bbfc5688dac",
			},
			mockStat:      &util.FileInfo{}, // Not a directory
			expHandled:    true,
			expStatusCode: http.StatusTemporaryRedirect,
			expLocation:   "/github.com/user/repo/-/blob/some/file.go",
		},
		// "/github.com/user/repo/-/blob/some/dir" -> "/github.com/user/repo/-/tree/some/dir"
		{
			name:  "redirct blob to tree",
			route: routeBlob,
			path:  "/some/dir",
			common: &Common{
				Repo: &types.Repo{
					Name: "github.com/user/repo",
				},
				CommitID: "eca7e807356b887ee24b7a7497973bbfc5688dac",
			},
			mockStat:      &util.FileInfo{Mode_: os.ModeDir},
			expHandled:    true,
			expStatusCode: http.StatusTemporaryRedirect,
			expLocation:   "/github.com/user/repo/-/tree/some/dir",
		},
		// "/github.com/user/repo@master/-/tree/some/file.go" -> "/github.com/user/repo@master/-/blob/some/file.go"
		{
			name:  "redirct tree to blob on a revision",
			route: routeTree,
			path:  "/some/file.go",
			common: &Common{
				Repo: &types.Repo{
					Name: "github.com/user/repo",
				},
				Rev:      "@master",
				CommitID: "eca7e807356b887ee24b7a7497973bbfc5688dac",
			},
			mockStat:      &util.FileInfo{}, // Not a directory
			expHandled:    true,
			expStatusCode: http.StatusTemporaryRedirect,
			expLocation:   "/github.com/user/repo@master/-/blob/some/file.go",
		},
		// "/github.com/user/repo@master/-/blob/some/dir" -> "/github.com/user/repo@master/-/tree/some/dir"
		{
			name:  "redirct blob to tree on a revision",
			route: routeBlob,
			path:  "/some/dir",
			common: &Common{
				Repo: &types.Repo{
					Name: "github.com/user/repo",
				},
				Rev:      "@master",
				CommitID: "eca7e807356b887ee24b7a7497973bbfc5688dac",
			},
			mockStat:      &util.FileInfo{Mode_: os.ModeDir},
			expHandled:    true,
			expStatusCode: http.StatusTemporaryRedirect,
			expLocation:   "/github.com/user/repo@master/-/tree/some/dir",
		},

		// "/github.com/user/repo/-/tree" -> "/github.com/user/repo"
		{
			name:  "redirct tree to root",
			route: routeTree,
			path:  "",
			common: &Common{
				Repo: &types.Repo{
					Name: "github.com/user/repo",
				},
				CommitID: "eca7e807356b887ee24b7a7497973bbfc5688dac",
			},
			expHandled:    true,
			expStatusCode: http.StatusTemporaryRedirect,
			expLocation:   "/github.com/user/repo",
		},
		// "/github.com/user/repo/-/blob" -> "/github.com/user/repo"
		{
			name:  "redirct blob to root",
			route: routeBlob,
			path:  "",
			common: &Common{
				Repo: &types.Repo{
					Name: "github.com/user/repo",
				},
				CommitID: "eca7e807356b887ee24b7a7497973bbfc5688dac",
			},
			expHandled:    true,
			expStatusCode: http.StatusTemporaryRedirect,
			expLocation:   "/github.com/user/repo",
		},
		// "/github.com/user/repo@master/-/tree" -> "/github.com/user/repo"
		{
			name:  "redirct tree to root on a revision",
			route: routeTree,
			path:  "",
			common: &Common{
				Repo: &types.Repo{
					Name: "github.com/user/repo",
				},
				Rev:      "@master",
				CommitID: "eca7e807356b887ee24b7a7497973bbfc5688dac",
			},
			expHandled:    true,
			expStatusCode: http.StatusTemporaryRedirect,
			expLocation:   "/github.com/user/repo@master",
		},
		// "/github.com/user/repo@master/-/blob" -> "/github.com/user/repo"
		{
			name:  "redirct blob to root on a revision",
			route: routeBlob,
			path:  "",
			common: &Common{
				Repo: &types.Repo{
					Name: "github.com/user/repo",
				},
				Rev:      "@master",
				CommitID: "eca7e807356b887ee24b7a7497973bbfc5688dac",
			},
			expHandled:    true,
			expStatusCode: http.StatusTemporaryRedirect,
			expLocation:   "/github.com/user/repo@master",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			git.Mocks.Stat = func(commit api.CommitID, name string) (os.FileInfo, error) {
				return test.mockStat, nil
			}
			t.Cleanup(git.ResetMocks)

			w := httptest.NewRecorder()
			r, err := http.NewRequest("GET", test.path, nil)
			if err != nil {
				t.Fatal(err)
			}

			handled, err := redirectTreeOrBlob(test.route, test.path, test.common, w, r)
			if err != nil {
				t.Fatal(err)
			}

			if handled != test.expHandled {
				t.Fatalf("handled: want %v but got %v", test.expHandled, handled)
			} else if w.Code != test.expStatusCode {
				t.Fatalf("code: want %d but got %d", test.expStatusCode, w.Code)
			}

			if got := w.Header().Get("Location"); got != test.expLocation {
				t.Fatalf("redirect location: want %q but got %q", test.expLocation, got)
			}
		})
	}
}

func init() {
	globals.ConfigurationServerFrontendOnly = &conf.Server{}
	globalstatedb.Mock.Get = func(ctx context.Context) (*globalstatedb.State, error) {
		return &globalstatedb.State{SiteID: "a"}, nil
	}
	siteid.Init()
}
