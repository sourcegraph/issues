package graphqlbackend

import (
	"context"
	"errors"
	"os"
	"sync"

	"github.com/neelance/parallel"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/internal/inventory"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/goroutine"
)

func (srs *searchResultsStats) Languages(ctx context.Context) ([]*languageStatisticsResolver, error) {
	srr, err := srs.getResults(ctx)
	if err != nil {
		return nil, err
	}

	langs, err := searchResultsStatsLanguages(ctx, srr.Results())
	if err != nil {
		return nil, err
	}

	wrapped := make([]*languageStatisticsResolver, len(langs))
	for i, lang := range langs {
		wrapped[i] = &languageStatisticsResolver{lang}
	}
	return wrapped, nil
}

func searchResultsStatsLanguages(ctx context.Context, results []SearchResultResolver) ([]inventory.Lang, error) {
	// Batch our operations by repo-commit.
	type repoCommit struct {
		repo     api.RepoID
		commitID api.CommitID
	}

	// Records the work necessary for a batch (repoCommit).
	type fileStatsWork struct {
		fullEntries  []os.FileInfo     // matched these full files
		partialFiles map[string]uint64 // file with line matches (path) -> count of lines matching
	}

	var (
		repos    = map[api.RepoID]*types.Repo{}
		filesMap = map[repoCommit]*fileStatsWork{}

		run = parallel.NewRun(16)

		allInventories   []inventory.Inventory
		allInventoriesMu sync.Mutex
	)

	// Track the mapping of repo ID -> repo object as we iterate.
	sawRepo := func(repo *types.Repo) {
		if _, ok := repos[repo.ID]; !ok {
			repos[repo.ID] = repo
		}
	}

	// Only count repo matches if all matches are repo matches. Otherwise, it would get confusing
	// because we might have a match of a repo *and* a file in the repo. We would need to avoid
	// double-counting. In this case, we will just count the matching files.
	hasNonRepoMatches := false
	for _, res := range results {
		if _, ok := res.ToRepository(); !ok {
			hasNonRepoMatches = true
		}
	}

	for _, res := range results {
		if fileMatch, ok := res.ToFileMatch(); ok {
			sawRepo(fileMatch.Repository().repo)
			key := repoCommit{repo: fileMatch.Repository().repo.ID, commitID: fileMatch.CommitID}

			if _, ok := filesMap[key]; !ok {
				filesMap[key] = &fileStatsWork{}
			}

			if len(fileMatch.LineMatches()) > 0 {
				// Only count matching lines. TODO(sqs): bytes are not counted for these files
				if filesMap[key].partialFiles == nil {
					filesMap[key].partialFiles = map[string]uint64{}
				}
				filesMap[key].partialFiles[fileMatch.path()] += uint64(len(fileMatch.LineMatches()))
			} else {
				// Count entire file.
				filesMap[key].fullEntries = append(filesMap[key].fullEntries, &fileInfo{
					path:  fileMatch.path(),
					isDir: fileMatch.File().IsDirectory(),
				})
			}
		} else if repo, ok := res.ToRepository(); ok && !hasNonRepoMatches {
			sawRepo(repo.repo)
			run.Acquire()
			goroutine.Go(func() {
				defer run.Release()

				branchRef, err := repo.DefaultBranch(ctx)
				if err != nil {
					run.Error(err)
					return
				}
				if branchRef == nil || branchRef.Target() == nil {
					return
				}
				target, err := branchRef.Target().OID(ctx)
				if err != nil {
					run.Error(err)
					return
				}
				inv, err := backend.Repos.GetInventory(ctx, repo.repo, api.CommitID(target), true)
				if err != nil {
					run.Error(err)
					return
				}
				allInventoriesMu.Lock()
				allInventories = append(allInventories, *inv)
				allInventoriesMu.Unlock()
			})
		} else if _, ok := res.ToCommitSearchResult(); ok {
			return nil, errors.New("language statistics do not support diff searches")
		}
	}

	for key_, work_ := range filesMap {
		key := key_
		work := work_
		run.Acquire()
		goroutine.Go(func() {
			defer run.Release()

			cachedRepo, err := backend.CachedGitRepo(ctx, repos[key.repo])
			if err != nil {
				run.Error(err)
				return
			}
			invCtx, err := backend.InventoryContext(*cachedRepo, key.commitID, true)
			if err != nil {
				run.Error(err)
				return
			}

			// Inventory all full-entry (files and trees) matches together.
			inv, err := invCtx.Entries(ctx, work.fullEntries...)
			if err != nil {
				run.Error(err)
				return
			}
			allInventoriesMu.Lock()
			allInventories = append(allInventories, inv)
			allInventoriesMu.Unlock()

			// Separately inventory each partial-file match because we only increment the language lines
			// by the number of matched lines in the file.
			for partialFile, lines := range work.partialFiles {
				inv, err := invCtx.Entries(ctx,
					fileInfo{path: partialFile, isDir: false},
				)
				if err != nil {
					run.Error(err)
					return
				}
				for i := range inv.Languages {
					inv.Languages[i].TotalLines = lines
				}
				allInventoriesMu.Lock()
				allInventories = append(allInventories, inv)
				allInventoriesMu.Unlock()
			}
		})
	}

	if err := run.Wait(); err != nil {
		return nil, err
	}
	return inventory.Sum(allInventories).Languages, nil
}
