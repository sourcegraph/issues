package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

const testAddress = "test.local:3939"

var discardLogger = log.New(ioutil.Discard, "", log.LstdFlags)

func TestReposHandler(t *testing.T) {
	cases := []struct {
		name  string
		repos []string
	}{{
		name: "empty",
	}, {
		name:  "simple",
		repos: []string{"project1", "project2"},
	}, {
		name:  "nested",
		repos: []string{"project1", "project1/subproject", "project2", "dir/project3"},
	}}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := gitInitRepos(t, tc.repos...)

			h, err := (&Serve{
				Info:  testLogger(t),
				Debug: discardLogger,
				Addr:  testAddress,
				Root:  root,

				updatingServerInfo: 2, // disables background updates
			}).handler()
			if err != nil {
				t.Fatal(err)
			}

			var want []Repo
			for _, name := range tc.repos {
				want = append(want, Repo{Name: name, URI: path.Join("/repos", name)})
			}
			testReposHandler(t, h, want)
		})

		// Now do the same test, but we root it under a repo we serve. This is
		// to test we properly serve up the root repo as something other than
		// "."
		t.Run("rooted-"+tc.name, func(t *testing.T) {
			repos := []string{"project-root"}
			for _, name := range tc.repos {
				repos = append(repos, filepath.Join("project-root", name))
			}

			root := gitInitRepos(t, repos...)

			// This is the difference to above, we point our root at the git repo
			root = filepath.Join(root, "project-root")

			h, err := (&Serve{
				Info:  testLogger(t),
				Debug: discardLogger,
				Addr:  testAddress,
				Root:  root,

				updatingServerInfo: 2, // disables background updates
			}).handler()
			if err != nil {
				t.Fatal(err)
			}

			// project-root is served from /repos, etc
			want := []Repo{{Name: "project-root", URI: "/repos"}}
			for _, name := range tc.repos {
				want = append(want, Repo{Name: filepath.Join("project-root", name), URI: path.Join("/repos", name)})
			}
			testReposHandler(t, h, want)
		})
	}
}

func testReposHandler(t *testing.T, h http.Handler, repos []Repo) {
	ts := httptest.NewServer(h)
	t.Cleanup(ts.Close)

	get := func(path string) string {
		res, err := http.Get(ts.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		b, err := ioutil.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		if testing.Verbose() {
			t.Logf("GET %s:\n%s", path, b)
		}
		return string(b)
	}

	// Check we have some known strings on the index page
	index := get("/")
	for _, sub := range []string{"http://" + testAddress, "/v1/list-repos", "/repos/"} {
		if !strings.Contains(index, sub) {
			t.Errorf("index page does not contain substring %q", sub)
		}
	}

	// repos page will list the top-level dirs
	list := get("/repos/")
	for _, repo := range repos {
		if path.Dir(repo.URI) != "/repos" {
			continue
		}
		if !strings.Contains(repo.Name, "/") && !strings.Contains(list, repo.Name) {
			t.Errorf("repos page does not contain substring %q", repo.Name)
		}
	}

	// check our API response
	type Response struct{ Items []Repo }
	var want, got Response
	want.Items = repos
	if err := json.Unmarshal([]byte(get("/v1/list-repos")), &got); err != nil {
		t.Fatal(err)
	}
	opts := []cmp.Option{
		cmpopts.EquateEmpty(),
		cmpopts.SortSlices(func(a, b Repo) bool { return a.Name < b.Name }),
	}
	if !cmp.Equal(want, got, opts...) {
		t.Errorf("mismatch (-want +got):\n%s", cmp.Diff(want, got, opts...))
	}
}

func gitInitRepos(t *testing.T, names ...string) string {
	root, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(root) })
	root = filepath.Join(root, "repos-root")

	for _, name := range names {
		p := filepath.Join(root, name)
		if err := os.MkdirAll(p, 0755); err != nil {
			t.Fatal(err)
		}
		p = filepath.Join(p, ".git")
		if err := exec.Command("git", "init", "--bare", p).Run(); err != nil {
			t.Fatal(err)
		}
	}

	return root
}

func TestIgnoreGitSubmodules(t *testing.T) {
	root, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(root) })

	if err := os.MkdirAll(filepath.Join(root, "dir"), os.ModePerm); err != nil {
		t.Fatal(err)
	}

	if err := ioutil.WriteFile(filepath.Join(root, "dir", ".git"), []byte("ignore me please"), os.ModePerm); err != nil {
		t.Fatal(err)
	}

	repos := (&Serve{
		Info:  testLogger(t),
		Debug: discardLogger,
		Root:  root,

		updatingServerInfo: 2, // disables background updates
	}).configureRepos()
	if len(repos) != 0 {
		t.Fatalf("expected no repos, got %v", repos)
	}
}

func testLogger(t *testing.T) *log.Logger {
	return log.New(testWriter{t}, "testLogger ", log.LstdFlags)
}

type testWriter struct {
	*testing.T
}

func (tw testWriter) Write(p []byte) (n int, err error) {
	tw.T.Log(string(p))
	return len(p), nil
}
