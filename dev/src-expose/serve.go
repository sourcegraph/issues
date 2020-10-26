package main

import (
	"bytes"
	"encoding/json"
	"html/template"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/pkg/errors"
)

type Serve struct {
	Addr  string
	Root  string
	Info  *log.Logger
	Debug *log.Logger

	// updatingServerInfo is used to ensure we only have 1 goroutine running
	// git update-server-info.
	updatingServerInfo uint64
}

func (s *Serve) Start() error {
	ln, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return errors.Wrap(err, "listen")
	}

	// Update Addr to what listener actually used.
	s.Addr = ln.Addr().String()

	s.Info.Printf("listening on http://%s", s.Addr)
	h, err := s.handler()
	if err != nil {
		return errors.Wrap(err, "configuring server")
	}

	if err := (&http.Server{Handler: h}).Serve(ln); err != nil {
		return errors.Wrap(err, "serving")
	}

	return nil
}

var indexHTML = template.Must(template.New("").Parse(`<html>
<head><title>src-expose</title></head>
<body>
<h2>src-expose</h2>
<pre>
{{.Explain}}
<ul>{{range .Links}}
<li><a href="{{.}}">{{.}}</a></li>
{{- end}}
</ul>
</pre>
</body>
</html>`))

type Repo struct {
	Name string
	URI  string
}

func (s *Serve) handler() (http.Handler, error) {
	s.Info.Printf("serving git repositories from %s", s.Root)
	s.configureRepos()

	// Start the HTTP server.
	mux := &http.ServeMux{}

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		err := indexHTML.Execute(w, map[string]interface{}{
			"Explain": explainAddr(s.Addr),
			"Links": []string{
				"/v1/list-repos",
				"/repos/",
			},
		})
		if err != nil {
			log.Println(err)
		}
	})

	mux.HandleFunc("/v1/list-repos", func(w http.ResponseWriter, r *http.Request) {
		var repos []Repo
		var reposRootIsRepo bool
		for _, name := range s.configureRepos() {
			if name == "." {
				reposRootIsRepo = true
			}

			repos = append(repos, Repo{
				Name: name,
				URI:  path.Join("/repos", name),
			})
		}

		if reposRootIsRepo {
			// Update all names to be relative to the parent of
			// reposRoot. This is to give a better name than "." for repos
			// root
			abs, err := filepath.Abs(s.Root)
			if err != nil {
				http.Error(w, "failed to get the absolute path of reposRoot: "+err.Error(), http.StatusInternalServerError)
				return
			}
			rootName := filepath.Base(abs)
			for i := range repos {
				repos[i].Name = path.Join(rootName, repos[i].Name)
			}
		}

		resp := struct {
			Items []Repo
		}{
			Items: repos,
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		_ = enc.Encode(&resp)
	})

	mux.Handle("/repos/", http.StripPrefix("/repos/", http.FileServer(httpDir{http.Dir(s.Root)})))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/.git/objects/") { // exclude noisy path
			s.Info.Printf("%s %s", r.Method, r.URL.Path)
		}
		mux.ServeHTTP(w, r)
	}), nil
}

type httpDir struct {
	http.Dir
}

// Wraps the http.Dir to inject subdir "/.git" to the path.
func (d httpDir) Open(name string) (http.File, error) {
	// Backwards compatibility for old config, skip if name already contains "/.git/".
	if !strings.Contains(name, "/.git/") {
		// Loops over subpaths that are requested by Git client to find the insert point.
		// The order of slice matters, must try to match "/objects/" before "/info/"
		// because there is a path "/objects/info/" exists.
		for _, sp := range []string{"/objects/", "/info/", "/HEAD"} {
			if i := strings.LastIndex(name, sp); i > 0 {
				name = name[:i] + "/.git" + name[i:]
				break
			}
		}
	}
	return d.Dir.Open(name)
}

// configureRepos finds all .git directories and configures them to be served.
// It returns a slice of all the git directories it finds. The paths are
// relative to root.
func (s *Serve) configureRepos() []string {
	var gitDirs []string

	err := filepath.Walk(s.Root, func(path string, fi os.FileInfo, fileErr error) error {
		if fileErr != nil {
			s.Info.Printf("WARN: ignoring error searching %s: %v", path, fileErr)
			return nil
		}
		if !fi.IsDir() {
			return nil
		}

		// We recurse into bare repositories to find subprojects. Prevent
		// recursing into .git
		if filepath.Base(path) == ".git" {
			return filepath.SkipDir
		}

		// Check whether a particular directory is a repository or not.
		//
		// A directory which also is a repository (have .git folder inside it)
		// will contain nil error. If it does, proceed to configure.
		gitdir := filepath.Join(path, ".git")
		if fi, err := os.Stat(gitdir); err != nil || !fi.IsDir() {
			s.Debug.Printf("not a repository root: %s", path)
			return nil
		}

		if err := configurePostUpdateHook(s.Info, gitdir); err != nil {
			s.Info.Printf("failed configuring repo at %s: %v", gitdir, err)
			return nil
		}

		subpath, err := filepath.Rel(s.Root, path)
		if err != nil {
			// According to WalkFunc docs, path is always filepath.Join(root,
			// subpath). So Rel should always work.
			s.Info.Fatalf("filepath.Walk returned %s which is not relative to %s: %v", path, s.Root, err)
		}
		gitDirs = append(gitDirs, filepath.ToSlash(subpath))

		// Check whether a repository is a bare repository or not.
		//
		// If it yields false, which means it is a non-bare repository,
		// skip the directory so that it will not recurse to the subdirectories.
		// If it is a bare repository, proceed to recurse.
		c := exec.Command("git", "rev-parse", "--is-bare-repository")
		c.Dir = gitdir
		out, _ := c.CombinedOutput()

		if string(out) == "false\n" {
			return filepath.SkipDir
		}

		return nil
	})

	if err != nil {
		// Our WalkFunc doesn't return any errors, so neither should filepath.Walk
		panic(err)
	}

	go s.allUpdateServerInfo(gitDirs)

	return gitDirs
}

// allUpdateServerInfo will run updateServerInfo on each gitDirs. To prevent
// too many of these processes running, it will only run one at a time.
func (s *Serve) allUpdateServerInfo(gitDirs []string) {
	if !atomic.CompareAndSwapUint64(&s.updatingServerInfo, 0, 1) {
		return
	}

	for _, dir := range gitDirs {
		gitdir := filepath.Join(s.Root, dir)
		if err := updateServerInfo(gitdir); err != nil {
			s.Info.Printf("git update-server-info failed for %s: %v", gitdir, err)
		}
	}

	atomic.StoreUint64(&s.updatingServerInfo, 0)
}

var postUpdateHook = []byte("#!/bin/sh\nexec git update-server-info\n")

// configureOneRepos tweaks a .git repo such that it can be git cloned.
// See https://theartofmachinery.com/2016/07/02/git_over_http.html
// for background.
func configurePostUpdateHook(logger *log.Logger, gitDir string) error {
	postUpdatePath := filepath.Join(gitDir, "hooks", "post-update")
	if b, _ := ioutil.ReadFile(postUpdatePath); bytes.Equal(b, postUpdateHook) {
		return nil
	}

	logger.Printf("configuring git post-update hook for %s", gitDir)

	if err := updateServerInfo(gitDir); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(postUpdatePath), 0755); err != nil {
		return errors.Wrap(err, "create git hooks dir")
	}
	if err := ioutil.WriteFile(postUpdatePath, postUpdateHook, 0755); err != nil {
		return errors.Wrap(err, "setting post-update hook")
	}

	return nil
}

func updateServerInfo(gitDir string) error {
	c := exec.Command("git", "update-server-info")
	c.Dir = gitDir
	out, err := c.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "updating server info: %s", out)
	}
	return nil
}
