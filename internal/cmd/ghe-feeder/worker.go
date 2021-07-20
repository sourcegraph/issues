package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/google/go-github/v31/github"
	"github.com/inconshreveable/log15"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/schollz/progressbar/v3"
	"golang.org/x/oauth2"
	"golang.org/x/time/rate"
)

func newGHEClient(ctx context.Context, baseURL, uploadURL, token string) (*github.Client, error) {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	return github.NewEnterpriseClient(baseURL, uploadURL, tc)
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

// randomOrgNameAndSize returns a random, unique name for an org and a random size of repos it should have
func randomOrgNameAndSize() (string, int) {
	size := rand.Intn(500)
	if size < 5 {
		size = 5
	}
	name := fmt.Sprintf("%s-%d", getRandomName(0), size)
	return name, size
}

// feederError is an error while processing an ownerRepo line. errType partitions the errors in 4 major categories
// to use in metrics in logging: api, clone, push and unknown.
type feederError struct {
	// one of: api, clone, push, unknown
	errType string
	// underlying error
	err error
}

func (e *feederError) Error() string {
	return fmt.Sprintf("%v: %v", e.errType, e.err)
}

func (e *feederError) Unwrap() error {
	return e.err
}

// worker processes ownerRepo strings, feeding them to GHE instance. it declares orgs if needed, clones from
// github.com, adds GHE as a remote, declares repo in GHE through API and does a git push to the GHE.
// there's many workers working at the same time, taking work from a work channel fed by a pump that reads lines
// from the input.
type worker struct {
	// used in logs and metrics
	name string
	// index of the worker (which one in range [0, numWorkers)
	index int
	// directory to use for cloning from github.com
	scratchDir string

	// GHE API client
	client *github.Client
	admin  string
	token  string

	// gets the lines of work from this channel (each line has a owner/repo string in some format)
	work <-chan string
	// wait group to decrement when this worker is done working
	wg *sync.WaitGroup
	// terminal UI progress bar
	bar *progressbar.ProgressBar

	// some stats
	numFailed    int64
	numSucceeded int64

	// feeder DB is a sqlite DB, worker marks processed ownerRepos as successfully processed or failed
	fdr *feederDB
	// keeps track of org to which to add repos
	// (when currentNumRepos reaches currentMaxRepos, it generates a new random triple of these)
	currentOrg      string
	currentNumRepos int
	currentMaxRepos int

	// logger has worker name inprinted
	logger log15.Logger

	// rate limiter for the GHE API calls
	rateLimiter *rate.Limiter
	// how many simultaneous `git push` operations to the GHE
	pushSem chan struct{}
	// how many simultaneous `git clone` operations from github.com
	cloneSem chan struct{}
	// how many times to try to clone from github.com
	numCloningAttempts int
	// how long to wait before cutting short a cloning from github.com
	cloneRepoTimeout time.Duration

	// host to add as a remote to a cloned repo pointing to GHE instance
	host string
}

// run spins until work channel closes or context cancels
func (wkr *worker) run(ctx context.Context) {
	defer wkr.wg.Done()

	wkr.currentOrg, wkr.currentMaxRepos = randomOrgNameAndSize()

	wkr.logger.Debug("switching to org", "org", wkr.currentOrg)

	// declare the first org to start the worker processing
	err := wkr.addGHEOrg(ctx)
	if err != nil {
		wkr.logger.Error("failed to create org", "org", wkr.currentOrg, "error", err)
		// add it to default org then
		wkr.currentOrg = ""
	} else {
		err = wkr.fdr.declareOrg(wkr.currentOrg)
		if err != nil {
			wkr.logger.Error("failed to declare org", "org", wkr.currentOrg, "error", err)
		}
	}

	for line := range wkr.work {
		_ = wkr.bar.Add(1)

		if ctx.Err() != nil {
			return
		}

		xs := strings.Split(line, "/")
		if len(xs) != 2 {
			wkr.logger.Error("failed tos split line", "line", line)
			continue
		}
		owner, repo := xs[0], xs[1]

		// process one owner/repo
		err := wkr.process(ctx, owner, repo)
		reposProcessedCounter.With(prometheus.Labels{"worker": wkr.name}).Inc()
		remainingWorkGauge.Add(-1.0)
		if err != nil {
			wkr.numFailed++
			errType := "unknown"
			var e *feederError
			if errors.As(err, &e) {
				errType = e.errType
			}
			reposFailedCounter.With(prometheus.Labels{"worker": wkr.name, "err_type": errType}).Inc()
			_ = wkr.fdr.failed(line, errType)
		} else {
			reposSucceededCounter.Inc()
			wkr.numSucceeded++
			wkr.currentNumRepos++

			err = wkr.fdr.succeeded(line, wkr.currentOrg)
			if err != nil {
				wkr.logger.Error("failed to mark succeeded repo", "ownerRepo", line, "error", err)
			}

			// switch to a new org
			if wkr.currentNumRepos >= wkr.currentMaxRepos {
				wkr.currentOrg, wkr.currentMaxRepos = randomOrgNameAndSize()
				wkr.currentNumRepos = 0
				wkr.logger.Debug("switching to org", "org", wkr.currentOrg)
				err := wkr.addGHEOrg(ctx)
				if err != nil {
					wkr.logger.Error("failed to create org", "org", wkr.currentOrg, "error", err)
					// add it to default org then
					wkr.currentOrg = ""
				} else {
					err = wkr.fdr.declareOrg(wkr.currentOrg)
					if err != nil {
						wkr.logger.Error("failed to declare org", "org", wkr.currentOrg, "error", err)
					}
				}
			}
		}
		ownerDir := filepath.Join(wkr.scratchDir, owner)

		// clean up clone on disk
		err = os.RemoveAll(ownerDir)
		if err != nil {
			wkr.logger.Error("failed to clean up cloned repo", "ownerRepo", line, "error", err, "ownerDir", ownerDir)
		}
	}
}

// process does the necessary work for one ownerRepo string: clone, declare repo in GHE through API, add remote and push
func (wkr *worker) process(ctx context.Context, owner, repo string) error {
	err := wkr.cloneRepo(ctx, owner, repo)
	if err != nil {
		wkr.logger.Error("failed to clone repo", "owner", owner, "repo", repo, "error", err)
		return &feederError{"clone", err}
	}

	gheRepo, err := wkr.addGHERepo(ctx, owner, repo)
	if err != nil {
		wkr.logger.Error("failed to create GHE repo", "owner", owner, "repo", repo, "error", err)
		return &feederError{"api", err}
	}

	err = wkr.addRemote(ctx, gheRepo, owner, repo)
	if err != nil {
		wkr.logger.Error("failed to add GHE as a remote in cloned repo", "owner", owner, "repo", repo, "error", err)
		return &feederError{"api", err}
	}

	for attempt := 0; attempt < wkr.numCloningAttempts && ctx.Err() == nil; attempt++ {
		err = wkr.pushToGHE(ctx, owner, repo)
		if err == nil {
			return nil
		}
		wkr.logger.Error("failed to push cloned repo to GHE", "attempt", attempt+1, "owner", owner, "repo", repo, "error", err)
	}

	if ctx.Err() != nil {
		return ctx.Err()
	}
	return &feederError{"push", err}
}

// cloneRepo clones the specified repo from github.com into the scratchDir
func (wkr *worker) cloneRepo(ctx context.Context, owner, repo string) error {
	select {
	case wkr.cloneSem <- struct{}{}:
		defer func() {
			<-wkr.cloneSem
		}()

		ownerDir := filepath.Join(wkr.scratchDir, owner)
		err := os.MkdirAll(ownerDir, 0777)
		if err != nil {
			wkr.logger.Error("failed to create owner dir", "ownerDir", ownerDir, "error", err)
			return err
		}

		ctx, cancel := context.WithTimeout(ctx, wkr.cloneRepoTimeout)
		defer cancel()

		cmd := exec.CommandContext(ctx, "git", "clone",
			fmt.Sprintf("https://github.com/%s/%s", owner, repo))
		cmd.Dir = ownerDir
		cmd.Env = append(cmd.Env, "GIT_ASKPASS=/bin/echo")

		return cmd.Run()
	case <-ctx.Done():
		return ctx.Err()
	}
}

// addRemote declares the GHE as a remote to the cloned repo
func (wkr *worker) addRemote(ctx context.Context, gheRepo *github.Repository, owner, repo string) error {
	repoDir := filepath.Join(wkr.scratchDir, owner, repo)

	remoteURL := fmt.Sprintf("https://%s@%s/%s.git", wkr.token, wkr.host, *gheRepo.FullName)
	cmd := exec.CommandContext(ctx, "git", "remote", "add", "ghe", remoteURL)
	cmd.Dir = repoDir

	return cmd.Run()
}

// pushToGHE does a `git push` command to the GHE remote
func (wkr *worker) pushToGHE(ctx context.Context, owner, repo string) error {
	select {
	case wkr.pushSem <- struct{}{}:
		defer func() {
			<-wkr.pushSem
		}()
		repoDir := filepath.Join(wkr.scratchDir, owner, repo)

		ctx, cancel := context.WithTimeout(ctx, wkr.cloneRepoTimeout)
		defer cancel()

		cmd := exec.CommandContext(ctx, "git", "push", "ghe", "master")
		cmd.Dir = repoDir

		return cmd.Run()
	case <-ctx.Done():
		return ctx.Err()
	}
}

// addGHEOrg uses the GHE API to declare the org at the GHE
func (wkr *worker) addGHEOrg(ctx context.Context) error {
	err := wkr.rateLimiter.Wait(ctx)
	if err != nil {
		wkr.logger.Error("failed to get a request spot from rate limiter", "error", err)
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()

	gheOrg := &github.Organization{
		Login: github.String(wkr.currentOrg),
	}

	_, _, err = wkr.client.Admin.CreateOrg(ctx, gheOrg, wkr.admin)
	return err
}

// addGHEOrg uses the GHE API to declare the repo at the GHE
func (wkr *worker) addGHERepo(ctx context.Context, owner, repo string) (*github.Repository, error) {
	err := wkr.rateLimiter.Wait(ctx)
	if err != nil {
		wkr.logger.Error("failed to get a request spot from rate limiter", "error", err)
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()

	gheRepo := &github.Repository{
		Name: github.String(fmt.Sprintf("%s-%s", owner, repo)),
	}

	gheReturnedRepo, _, err := wkr.client.Repositories.Create(ctx, wkr.currentOrg, gheRepo)
	return gheReturnedRepo, err
}
