package gitserver

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/store"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/gitserver"
	"github.com/sourcegraph/sourcegraph/internal/vcs/git"
)

type Client struct{}

var DefaultClient = &Client{}

// Archive retrieves a tar-formatted archive of the given commit.
func (c *Client) Archive(ctx context.Context, store store.Store, repositoryID int, commit string) (io.Reader, error) {
	repo, err := repositoryIDToRepo(ctx, store, repositoryID)
	if err != nil {
		return nil, err
	}

	if _, err := git.ResolveRevision(ctx, repo, nil, commit, git.ResolveRevisionOptions{}); err != nil {
		return nil, errors.Wrap(err, "git.ResolveRevision")
	}

	return gitserver.DefaultClient.Archive(ctx, repo, gitserver.ArchiveOptions{
		Treeish: commit,
		Format:  "tar",
	})
}

// Head determines the tip commit of the default branch for the given repository.
func (c *Client) Head(ctx context.Context, store store.Store, repositoryID int) (string, error) {
	return execGitCommand(ctx, store, repositoryID, "rev-parse", "HEAD")
}

type CommitGraphOptions struct {
	Commit string
	Limit  int
}

// CommitGraph returns the commit graph for the given repository as a mapping from a commit
// to its parents. If a commit is supplied, the returned graph will be rooted at the given
// commit. If a non-zero limit is supplied, at most that many commits will be returned.
func (c *Client) CommitGraph(ctx context.Context, store store.Store, repositoryID int, options CommitGraphOptions) (map[string][]string, error) {
	commands := []string{"log", "--all", "--pretty=%H %P"}
	if options.Commit != "" {
		commands = append(commands, options.Commit)
	}
	if options.Limit > 0 {
		commands = append(commands, fmt.Sprintf("-%d", options.Limit))
	}

	out, err := execGitCommand(ctx, store, repositoryID, commands...)
	if err != nil {
		return nil, err
	}

	return parseParents(strings.Split(out, "\n")), nil
}

// parseParents converts the output of git log into a map from commits to parent commits.
// If a commit is listed but has no ancestors then its parent slice is empty but is still
// present in the map.
func parseParents(pair []string) map[string][]string {
	commits := map[string][]string{}

	for _, pair := range pair {
		line := strings.TrimSpace(pair)
		if line == "" {
			continue
		}

		parts := strings.Split(line, " ")
		commits[parts[0]] = append(commits[parts[0]], parts[1:]...)

		for _, part := range parts[1:] {
			if _, ok := commits[part]; !ok {
				commits[part] = []string{}
			}
		}
	}

	return commits
}

// RawContents returns the contents of a file in a particular commit of a repository.
func (c *Client) RawContents(ctx context.Context, store store.Store, repositoryID int, commit, file string) ([]byte, error) {
	out, err := execGitCommand(ctx, store, repositoryID, "show", fmt.Sprintf("%s:%s", commit, file))
	if err != nil {
		return nil, err
	}

	return []byte(out), err
}

// DirectoryChildren determines all children known to git for the given directory names via an invocation
// of git ls-tree. The keys of the resulting map are the input (unsanitized) dirnames, and the value of
// that key are the files nested under that directory.
func (c *Client) DirectoryChildren(ctx context.Context, store store.Store, repositoryID int, commit string, dirnames []string) (map[string][]string, error) {
	out, err := execGitCommand(ctx, store, repositoryID, append([]string{"ls-tree", "--name-only", commit, "--"}, cleanDirectoriesForLsTree(dirnames)...)...)
	if err != nil {
		return nil, err
	}

	return parseDirectoryChildren(dirnames, strings.Split(out, "\n")), nil
}

// cleanDirectoriesForLsTree sanitizes the input dirnames to a git ls-tree command. There are a
// few peculiarities handled here:
//
//   1. The root of the tree must be indicated with `.`, and
//   2. In order for git ls-tree to return a directory's contents, the name must end in a slash.
func cleanDirectoriesForLsTree(dirnames []string) []string {
	var args []string
	for _, dir := range dirnames {
		if dir == "" {
			args = append(args, ".")
		} else {
			if !strings.HasSuffix(dir, "/") {
				dir += "/"
			}
			args = append(args, dir)
		}
	}

	return args
}

// parseDirectoryChildren converts the flat list of files from git ls-tree into a map. The keys of the
// resulting map are the input (unsanitized) dirnames, and the value of that key are the files nested
// under that directory. If dirnames contains a directory that encloses another, then the paths will
// be placed into the key sharing the longest path prefix.
func parseDirectoryChildren(dirnames []string, paths []string) map[string][]string {
	childrenMap := map[string][]string{}

	// Ensure each directory has an entry, even if it has no children
	// listed in the gitserver output.
	for _, dirname := range dirnames {
		childrenMap[dirname] = nil
	}

	// Order directory names by length (biggest first) so that we assign
	// paths to the most specific enclosing directory in the following loop.
	sort.Slice(dirnames, func(i, j int) bool {
		return len(dirnames[i]) > len(dirnames[j])
	})

	for _, path := range paths {
		if strings.Contains(path, "/") {
			for _, dirname := range dirnames {
				if strings.HasPrefix(path, dirname) {
					childrenMap[dirname] = append(childrenMap[dirname], path)
					break
				}
			}
		} else {
			// No need to loop here. If we have a root input directory it
			// will necessarily be the last element due to the previous
			// sorting step.
			if len(dirnames) > 0 && dirnames[len(dirnames)-1] == "" {
				childrenMap[""] = append(childrenMap[""], path)
			}
		}
	}

	return childrenMap
}

// FileExists determines whether a file exists in a particular commit of a repository.
func (c *Client) FileExists(ctx context.Context, store store.Store, repositoryID int, commit, file string) (bool, error) {
	repo, err := repositoryIDToRepo(ctx, store, repositoryID)
	if err != nil {
		return false, err
	}

	if _, err := git.ResolveRevision(ctx, repo, nil, commit, git.ResolveRevisionOptions{}); err != nil {
		return false, errors.Wrap(err, "git.ResolveRevision")
	}

	if _, err := git.Stat(ctx, repo, api.CommitID(commit), file); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

// ListFiles returns a list of root-relative file paths matching the given pattern in a particular
// commit of a repository.
func (c *Client) ListFiles(ctx context.Context, store store.Store, repositoryID int, commit string, pattern *regexp.Regexp) ([]string, error) {
	out, err := execGitCommand(ctx, store, repositoryID, "ls-tree", "--name-only", "-r", commit, "--")
	if err != nil {
		return nil, err
	}

	var matching []string
	for _, path := range strings.Split(out, "\n") {
		if pattern.MatchString(path) {
			matching = append(matching, path)
		}
	}

	return matching, nil
}

// Tags returns the git tags associated with the given commit along with a boolean indicating whether
// or not the tag was attached directly to the commit. If no tags exist at or before this commit, the
// tag is an empty string.
func (c *Client) Tags(ctx context.Context, store store.Store, repositoryID int, commit string) (string, bool, error) {
	tag, err := execGitCommand(ctx, store, repositoryID, "tag", "-l", "--points-at", commit)
	if err != nil {
		return "", false, err
	}
	if tag != "" {
		return tag, true, nil
	}

	// git describe --tags will exit with status 128 (fatal: No names found, cannot describe anything)
	// when there are no tags known to the given repo. In order to prevent a gitserver error from
	// occurring, we first check to see if there are any tags and early-exit.
	tags, err := execGitCommand(ctx, store, repositoryID, "tag")
	if err != nil {
		return "", false, err
	}
	if tags == "" {
		return "", false, nil
	}

	tag, err = execGitCommand(ctx, store, repositoryID, "describe", "--tags", "--abbrev=0", commit)
	if err != nil {
		return "", false, err
	}

	return tag, false, nil
}

// execGitCommand executes a git command for the given repository by identifier.
func execGitCommand(ctx context.Context, store store.Store, repositoryID int, args ...string) (string, error) {
	repo, err := repositoryIDToRepo(ctx, store, repositoryID)
	if err != nil {
		return "", err
	}

	cmd := gitserver.DefaultClient.Command("git", args...)
	cmd.Repo = repo
	out, err := cmd.CombinedOutput(ctx)
	return string(bytes.TrimSpace(out)), errors.Wrap(err, "gitserver.Command")
}

// repositoryIDToRepo creates a gitserver.Repo from a repository identifier.
func repositoryIDToRepo(ctx context.Context, store store.Store, repositoryID int) (gitserver.Repo, error) {
	repoName, err := store.RepoName(ctx, repositoryID)
	if err != nil {
		return gitserver.Repo{}, errors.Wrap(err, "store.RepoName")
	}

	return gitserver.Repo{Name: api.RepoName(repoName)}, nil
}
