package backend

import (
	"bytes"
	"encoding/json"
	"sort"

	"github.com/google/zoekt"
	"github.com/sourcegraph/sourcegraph/schema"
)

// zoektIndexOptions are options which change what we index for a
// repository. Everytime a repository is indexed by zoekt this structure is
// fetched. See getIndexOptions in the zoekt codebase.
//
// We only specify a subset of the fields.
type zoektIndexOptions struct {
	// RepoID is the Sourcegraph Repository ID.
	RepoID int32

	// LargeFiles is a slice of glob patterns where matching file paths should
	// be indexed regardless of their size. The pattern syntax can be found
	// here: https://golang.org/pkg/path/filepath/#Match.
	LargeFiles []string

	// Symbols if true will make zoekt index the output of ctags.
	Symbols bool

	// Branches is a slice of branches to index.
	Branches []zoekt.RepositoryBranch `json:",omitempty"`

	// Error if non-empty indicates the request failed for the repo.
	Error string `json:",omitempty"`
}

// RepoIndexOptions are the options used by GetIndexOptions for a specific
// repository.
type RepoIndexOptions struct {
	// RepoID is the Sourcegraph Repository ID.
	RepoID int32

	// GetVersion is used to resolve revisions for a repo. If it fails, the
	// error is encoded in the body. If the revision is missing, an empty
	// string should be returned rather than an error.
	GetVersion func(branch string) (string, error)
}

// GetIndexOptions returns a json blob for consumption by
// sourcegraph-zoekt-indexserver. It is for repos based on site settings c.
func GetIndexOptions(c *schema.SiteConfiguration, getRepoIndexOptions func(repo string) (*RepoIndexOptions, error), repos ...string) []byte {
	// Limit concurrency to 32 to avoid too many active network requests and
	// strain on gitserver (as ported from zoekt-sourcegraph-indexserver). In
	// future we want a more intelligent global limit based on scale.
	sema := make(chan struct{}, 32)

	results := make([][]byte, len(repos))
	for i := range repos {
		sema <- struct{}{}
		go func(i int) {
			defer func() { <-sema }()
			results[i] = getIndexOptions(c, repos[i], getRepoIndexOptions)
		}(i)
	}

	// Wait for jobs to finish (acquire full semaphore)
	for i := 0; i < cap(sema); i++ {
		sema <- struct{}{}
	}

	return bytes.Join(results, []byte{'\n'})
}

func getIndexOptions(c *schema.SiteConfiguration, repoName string, getRepoIndexOptions func(repo string) (*RepoIndexOptions, error)) []byte {
	opts, err := getRepoIndexOptions(repoName)
	if err != nil {
		return marshal(&zoektIndexOptions{Error: err.Error()})
	}

	o := &zoektIndexOptions{
		RepoID:     opts.RepoID,
		LargeFiles: c.SearchLargeFiles,
		Symbols:    getBoolPtr(c.SearchIndexSymbolsEnabled, true),
	}

	// Set of branch names. Always index HEAD
	branches := map[string]struct{}{"HEAD": {}}

	if c.ExperimentalFeatures != nil {
		for _, vc := range c.ExperimentalFeatures.VersionContexts {
			for _, rev := range vc.Revisions {
				if rev.Repo == repoName && rev.Rev != "" {
					branches[rev.Rev] = struct{}{}
				}
			}
		}

		for _, rev := range c.ExperimentalFeatures.SearchIndexBranches[repoName] {
			branches[rev] = struct{}{}
		}
	}

	for branch := range branches {
		v, err := opts.GetVersion(branch)
		if err != nil {
			return marshal(&zoektIndexOptions{Error: err.Error()})
		}

		// If we failed to resolve a branch, skip it
		if v == "" {
			continue
		}

		o.Branches = append(o.Branches, zoekt.RepositoryBranch{
			Name:    branch,
			Version: v,
		})
	}

	sort.Slice(o.Branches, func(i, j int) bool {
		a, b := o.Branches[i].Name, o.Branches[j].Name
		// Zoekt treats first branch as default branch, so put HEAD first
		if a == "HEAD" || b == "HEAD" {
			return a == "HEAD"
		}
		return a < b
	})

	// If the first branch is not HEAD, do not index anything. This should
	// not happen, since HEAD should always exist if other branches exist.
	if len(o.Branches) == 0 || o.Branches[0].Name != "HEAD" {
		o.Branches = nil
	}

	// Zoekt has a limit of 64 branches
	if len(o.Branches) > 64 {
		o.Branches = o.Branches[:64]
	}

	return marshal(o)
}

func getBoolPtr(b *bool, default_ bool) bool {
	if b == nil {
		return default_
	}
	return *b
}

func marshal(o *zoektIndexOptions) []byte {
	b, _ := json.Marshal(o)
	return b
}
