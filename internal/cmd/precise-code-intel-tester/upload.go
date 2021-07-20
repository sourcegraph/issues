package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cockroachdb/errors"

	"github.com/sourcegraph/sourcegraph/internal/cmd/precise-code-intel-tester/util"
)

// uploadCommand runs the "upload" command.
func uploadCommand() error {
	ctx, cleanup := util.SignalSensitiveContext()
	defer cleanup()

	start := time.Now()

	if err := uploadIndexes(ctx); err != nil {
		return err
	}

	fmt.Printf("All uploads completed processing in %s\n", time.Since(start))
	return nil
}

// Upload represents a fully uploaded (but possibly unprocessed) LSIF index.
type Upload struct {
	Name     string
	Index    int
	Rev      string
	UploadID string
}

// uploadIndexes uploads each file in the index directory and blocks until each upload has
// been successfully processed.
func uploadIndexes(ctx context.Context) error {
	revsByRepo, err := readRevsByRepo()
	if err != nil {
		return err
	}

	total := countRevs(revsByRepo)
	uploaded := make(chan Upload, total)
	processedSignals := makeProcessedSignals(revsByRepo)
	refreshedSignals := makeRefreshedSignals(revsByRepo)

	// Watch API for changes in state, and inform workers when their upload has been processed
	go watchStateChanges(ctx, uploaded, processedSignals, refreshedSignals)

	limiter := util.NewLimiter(numConcurrentUploads)
	defer limiter.Close()

	var fns []util.ParallelFn
	for name, revs := range revsByRepo {
		fns = append(fns, makeTestUploadForRepositoryFunction(name, revs, uploaded, processedSignals, refreshedSignals, limiter))
	}

	return util.RunParallel(ctx, total, fns)
}

// indexFilenamePattern extracts a repo name and rev from the index filename. We assume that the
// index segment here (the non-captured `.\d+.`) occupies [0,n) without gaps for each repository.
var indexFilenamePattern = regexp.MustCompile(`^(.+)\.\d+\.([0-9A-Fa-f]{40})\.dump$`)

// readRevsByRepo returns a list of revisions by repository names for which there is an index file.
func readRevsByRepo() (map[string][]string, error) {
	infos, err := os.ReadDir(indexDir)
	if err != nil {
		return nil, err
	}

	revsByRepo := map[string][]string{}
	for _, info := range infos {
		matches := indexFilenamePattern.FindStringSubmatch(info.Name())
		if len(matches) > 0 {
			revsByRepo[matches[1]] = append(revsByRepo[matches[1]], matches[2])
		}
	}

	return revsByRepo, nil
}

// countRevs returns the total number of revision.
func countRevs(revsByRepo map[string][]string) int {
	total := 0
	for _, revs := range revsByRepo {
		total += len(revs)
	}

	return total
}

// makeProcessedSignals returns a map of error channels for each revision.
func makeProcessedSignals(revsByRepo map[string][]string) map[string]map[string]chan error {
	processedSignals := map[string]map[string]chan error{}
	for repo, revs := range revsByRepo {
		revMap := make(map[string]chan error, len(revs))
		for _, rev := range revs {
			revMap[rev] = make(chan error, 1)
		}

		processedSignals[repo] = revMap
	}

	return processedSignals
}

type refreshState struct {
	Stale bool
	Err   error
}

// refreshedSignals returns a map of error channels for each repository.
func makeRefreshedSignals(revsByRepo map[string][]string) map[string]chan refreshState {
	refreshedSignals := map[string]chan refreshState{}
	for repo, revs := range revsByRepo {
		// Each channel may receive two values for each revision: a value when
		// a new upload has been processed and the repository becomes stale by
		// definition, and a value when the repository's commit graph has been
		// refreshed (or an error occurs).
		refreshedSignals[repo] = make(chan refreshState, len(revs)*2)
	}

	return refreshedSignals
}

// watchStateChanges maintains a list of uploaded but nonterminal upload records. This function
// polls the API and signals the worker when their upload has been processed. If an upload fails
// to process, the error will be sent to the worker.
func watchStateChanges(
	ctx context.Context,
	uploaded chan Upload,
	processedSignals map[string]map[string]chan error,
	refreshedSignals map[string]chan refreshState,
) {
	send := func(err error) {
		// Send err to everybody and exit
		for name, revs := range processedSignals {
			for rev, ch := range revs {
				if err != nil {
					ch <- err
				}

				close(ch)
				delete(processedSignals[name], rev)
			}
		}

		for name, ch := range refreshedSignals {
			if err != nil {
				ch <- refreshState{Err: err}
			}

			close(ch)
			delete(refreshedSignals, name)
		}
	}

	var uploads []Upload
	repositoryMap := map[string]struct{}{}

	for {
		select {
		case upload := <-uploaded:
			// Upload complete, add to process watch list
			uploads = append(uploads, upload)

		case <-time.After(time.Millisecond * 500):
			// Check states

		case <-ctx.Done():
			send(nil)
			return
		}

		var ids []string
		for _, upload := range uploads {
			ids = append(ids, upload.UploadID)
		}
		sort.Strings(ids)

		var names []string
		for name := range repositoryMap {
			names = append(names, name)
		}
		sort.Strings(names)

		stateByUpload, staleCommitGraphByRepo, err := uploadStates(ctx, ids, names)
		if err != nil {
			send(err)
			return
		}

		for name, stale := range staleCommitGraphByRepo {
			if !stale {
				// Repository is now up to date! Stop listening for updates.
				// If another upload is processed for this repository, we will
				// perform the same set of actions all over again; see below
				// when when the upload state is COMPLETED.
				refreshedSignals[name] <- refreshState{Stale: false}
				delete(repositoryMap, name)
			}
		}

		temp := uploads
		uploads = uploads[:0]

		for _, upload := range temp {
			var err error
			switch stateByUpload[upload.UploadID] {
			case "ERRORED":
				err = ErrProcessingFailed
				fallthrough

			case "COMPLETED":
				// Add repository to list of repositories with a stale
				// commit graph and watch until it becomes fresh again.
				repositoryMap[upload.Name] = struct{}{}
				refreshedSignals[upload.Name] <- refreshState{Stale: true}

				// Signal to listeners that this rev has been processed
				ch := processedSignals[upload.Name][upload.Rev]
				delete(processedSignals[upload.Name], upload.Rev)
				ch <- err
				close(ch)

			default:
				uploads = append(uploads, upload)
			}
		}
	}
}

// ErrProcessingFailed occurs when an upload enters the ERRORED state.
var ErrProcessingFailed = errors.New("processing failed")

const uploadQueryFragment = `
	u%d: node(id: "%s") {
		... on LSIFUpload { state }
	}
`

const repositoryQueryFragment = `
	r%d: repository(name: "%s") {
		codeIntelligenceCommitGraph {
			stale
		}
	}
`

// uploadStates returns a map from upload identifier to its current state.
func uploadStates(ctx context.Context, ids, names []string) (stateByUpload map[string]string, staleCommitGraphByRepo map[string]bool, _ error) {
	var fragments []string
	for i, id := range ids {
		fragments = append(fragments, fmt.Sprintf(uploadQueryFragment, i, id))
	}
	for i, name := range names {
		fullName := fmt.Sprintf("github.com/%s/%s", "sourcegraph-testing", name)
		fragments = append(fragments, fmt.Sprintf(repositoryQueryFragment, i, fullName))
	}
	query := fmt.Sprintf("{%s}", strings.Join(fragments, "\n"))

	payload := struct {
		Data map[string]struct {
			State       string `json:"state"`
			CommitGraph struct {
				Stale bool `json:"stale"`
			} `json:"codeIntelligenceCommitGraph"`
		} `json:"data"`
	}{}
	if err := util.QueryGraphQL(ctx, endpoint, "CodeIntelTesterUploadStates", token, query, nil, &payload); err != nil {
		return nil, nil, err
	}

	stateByUpload = map[string]string{}
	for i, id := range ids {
		stateByUpload[id] = payload.Data[fmt.Sprintf("u%d", i)].State
	}

	staleCommitGraphByRepo = map[string]bool{}
	for i, name := range names {
		staleCommitGraphByRepo[name] = payload.Data[fmt.Sprintf("r%d", i)].CommitGraph.Stale
	}

	return stateByUpload, staleCommitGraphByRepo, nil
}

// makeTestUploadForRepositoryFunction constructs a function for RunParallel that uploads the index files
// for the given repo, then blocks until the upload records enter a terminal state. If any upload fails to
// process, an error is returned.
func makeTestUploadForRepositoryFunction(
	name string,
	revs []string,
	uploaded chan Upload,
	processedSignals map[string]map[string]chan error,
	refreshedSignals map[string]chan refreshState,
	limiter *util.Limiter,
) util.ParallelFn {
	var numUploaded uint32
	var numProcessed uint32

	return util.ParallelFn{
		Fn: func(ctx context.Context) error {
			var wg sync.WaitGroup
			ch := make(chan error, len(revs))

			for i, rev := range revs {
				id, err := upload(ctx, name, i, rev, limiter)
				if err != nil {
					return err
				}
				atomic.AddUint32(&numUploaded, 1)

				wg.Add(1)
				go func() {
					defer wg.Done()
					ch <- <-processedSignals[name][rev]
				}()

				select {
				// send id to monitor
				case uploaded <- Upload{Name: name, Index: i, Rev: rev, UploadID: id}:

				case <-ctx.Done():
					return ctx.Err()
				}
			}

			go func() {
				wg.Wait()
				close(ch)
			}()

			// wait for all uploads to process
		processLoop:
			for {
				select {
				case err, ok := <-ch:
					if err != nil {
						return err
					}
					if !ok {
						break processLoop
					}
					atomic.AddUint32(&numProcessed, 1)

				case <-ctx.Done():
					return ctx.Err()
				}
			}

			// consume all values from the refreshedSignals channel that have already
			// been written. If the last one written is nil, then there will be no more
			// updates to the commit graph. If the last one written indicates that the
			// commit graph is stale, we'll continue to wait on the channel for an
			// additional nil value indicating the refresh.

			var lastValue refreshState
		refreshLoop:
			for {
				select {
				case err, ok := <-refreshedSignals[name]:
					if !ok {
						return nil
					}

					lastValue = err
				default:
					// no more values already in the channel, jump down
					break refreshLoop
				}
			}

			for {
				if !lastValue.Stale {
					return lastValue.Err
				}

				select {
				case err, ok := <-refreshedSignals[name]:
					if !ok {
						return nil
					}

					lastValue = err

				case <-ctx.Done():
					return ctx.Err()
				}
			}
		},
		Description: func() string {
			if n := atomic.LoadUint32(&numUploaded); n < uint32(len(revs)) {
				return fmt.Sprintf("Uploading index %d of %d for %s...", n+1, len(revs), name)
			}

			if n := atomic.LoadUint32(&numProcessed); n < uint32(len(revs)) {
				return fmt.Sprintf("Waiting for %d remaining uploads to process for %s...", len(revs)-int(n), name)
			}

			return fmt.Sprintf("Waiting for commit graph to update for %s...", name)
		},
		Total:    func() int { return len(revs) },
		Finished: func() int { return int(atomic.LoadUint32(&numProcessed)) },
	}
}

// uploadIDPattern extracts a GraphQL identifier from the output of the `src lsif upload` command.
var uploadIDPattern = regexp.MustCompile(`/settings/code-intelligence/lsif-uploads/([0-9A-Za-z=]+)`)

// upload invokes the `src lsif upload` command. This requires that src is installed on the
// current user's $PATH and is relatively up to date.
func upload(ctx context.Context, name string, index int, rev string, limiter *util.Limiter) (string, error) {
	if err := limiter.Acquire(ctx); err != nil {
		return "", err
	}
	defer limiter.Release()

	args := []string{
		fmt.Sprintf("-endpoint=%s", endpoint),
		"lsif",
		"upload",
		"-root=/",
		fmt.Sprintf("-repo=%s", fmt.Sprintf("github.com/%s/%s", "sourcegraph-testing", name)),
		fmt.Sprintf("-commit=%s", rev),
		fmt.Sprintf("-file=%s", filepath.Join(fmt.Sprintf("%s.%d.%s.dump", name, index, rev))),
	}

	cmd := exec.CommandContext(ctx, "src", args...)
	cmd.Dir = indexDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", errors.Wrap(err, fmt.Sprintf("error running 'src %s':\n%s\n", strings.Join(args, " "), output))
	}

	match := uploadIDPattern.FindSubmatch(output)
	if len(match) == 0 {
		return "", errors.Errorf("failed to extract URL:\n%s", output)
	}

	return string(match[1]), nil
}
