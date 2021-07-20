package gitserver

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/opentracing/opentracing-go/log"

	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/gitserver"
	"github.com/sourcegraph/sourcegraph/internal/observation"
	"github.com/sourcegraph/sourcegraph/internal/vcs"
	"github.com/sourcegraph/sourcegraph/internal/vcs/git"
	"github.com/sourcegraph/sourcegraph/lib/codeintel/pathexistence"
)

type Client struct {
	dbStore    DBStore
	operations *operations
}

func New(dbStore DBStore, observationContext *observation.Context) *Client {
	return &Client{
		dbStore:    dbStore,
		operations: newOperations(observationContext),
	}
}

// Head determines the tip commit of the default branch for the given repository.
func (c *Client) CommitExists(ctx context.Context, repositoryID int, commit string) (_ bool, err error) {
	ctx, endObservation := c.operations.commitExists.With(ctx, &err, observation.Args{LogFields: []log.Field{
		log.Int("repositoryID", repositoryID),
		log.String("commit", commit),
	}})
	defer endObservation(1, observation.Args{})

	out, err := c.execGitCommand(ctx, repositoryID, "cat-file", "-t", commit)
	if err == nil {
		return true, nil
	}

	if strings.Contains(out, "Not a valid object name") {
		err = nil
	}
	return false, err
}

// Head determines the tip commit of the default branch for the given repository. If no HEAD revision exists
// for the given repository (which occurs with empty repositories), a false-valued flag is returned along with
// a nil error and empty revision.
func (c *Client) Head(ctx context.Context, repositoryID int) (_ string, revisionExists bool, err error) {
	ctx, endObservation := c.operations.head.With(ctx, &err, observation.Args{LogFields: []log.Field{
		log.Int("repositoryID", repositoryID),
	}})
	defer endObservation(1, observation.Args{})

	revision, err := c.execGitCommand(ctx, repositoryID, "rev-parse", "HEAD")
	if err != nil {
		if errors.HasType(err, &gitserver.RevisionNotFoundError{}) {
			err = nil
		}

		return "", false, err
	}

	return revision, true, nil
}

// CommitDate returns the time that the given commit was committed.
func (c *Client) CommitDate(ctx context.Context, repositoryID int, commit string) (_ time.Time, err error) {
	ctx, endObservation := c.operations.commitDate.With(ctx, &err, observation.Args{LogFields: []log.Field{
		log.Int("repositoryID", repositoryID),
		log.String("commit", commit),
	}})
	defer endObservation(1, observation.Args{})

	out, err := c.execResolveRevGitCommand(ctx, repositoryID, commit, "show", "-s", "--format=%cI", commit)
	if err != nil {
		return time.Time{}, err
	}

	return time.Parse(time.RFC3339, strings.TrimSpace(out))
}

type CommitGraph struct {
	graph map[string][]string
	order []string
}

func (c *CommitGraph) Graph() map[string][]string { return c.graph }
func (c *CommitGraph) Order() []string            { return c.order }

type CommitGraphOptions struct {
	Commit  string
	AllRefs bool
	Limit   int
	Since   *time.Time
}

// CommitGraph returns the commit graph for the given repository as a mapping from a commit
// to its parents. If a commit is supplied, the returned graph will be rooted at the given
// commit. If a non-zero limit is supplied, at most that many commits will be returned.
func (c *Client) CommitGraph(ctx context.Context, repositoryID int, opts CommitGraphOptions) (_ *CommitGraph, err error) {
	ctx, endObservation := c.operations.commitGraph.With(ctx, &err, observation.Args{LogFields: []log.Field{
		log.Int("repositoryID", repositoryID),
		log.String("commit", opts.Commit),
		log.Int("limit", opts.Limit),
	}})
	defer endObservation(1, observation.Args{})

	args := []string{"log", "--pretty=%H %P", "--topo-order"}
	if opts.AllRefs {
		args = append(args, "--all")
	}
	if opts.Commit != "" {
		args = append(args, opts.Commit)
	}
	if opts.Since != nil {
		args = append(args, fmt.Sprintf("--since=%s", opts.Since.Format(time.RFC3339)))
	}
	if opts.Limit > 0 {
		args = append(args, fmt.Sprintf("-%d", opts.Limit))
	}

	out, err := c.execResolveRevGitCommand(ctx, repositoryID, opts.Commit, args...)
	if err != nil {
		return nil, err
	}

	return ParseCommitGraph(strings.Split(out, "\n")), nil
}

// ParseCommitGraph converts the output of git log into a map from commits to parent commits,
// and a topological ordering of commits such that parents come before children. If a commit
// is listed but has no ancestors then its parent slice is empty, but is still present in
// the map and the ordering. If the ordering is to be correct, the git log output must be
// formatted with --topo-order.
func ParseCommitGraph(lines []string) *CommitGraph {
	// Process lines backwards so that we see all parents before children.
	// We get a topological ordering by simply scraping the keys off in this
	// order.

	n := len(lines) - 1
	for i := 0; i < len(lines)/2; i++ {
		lines[i], lines[n-i] = lines[n-i], lines[i]
	}

	graph := make(map[string][]string, len(lines))
	order := make([]string, 0, len(lines))

	var prefix []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Split(line, " ")

		if len(parts) == 1 {
			graph[parts[0]] = []string{}
		} else {
			graph[parts[0]] = parts[1:]
		}

		order = append(order, parts[0])

		for _, part := range parts[1:] {
			if _, ok := graph[part]; !ok {
				graph[part] = []string{}
				prefix = append(prefix, part)
			}
		}
	}

	return &CommitGraph{
		graph: graph,
		order: append(prefix, order...),
	}
}

// RefDescription describes a commit at the head of a branch or tag.
type RefDescription struct {
	Name            string
	Type            RefType
	IsDefaultBranch bool
	CreatedDate     time.Time
}

type RefType int

const (
	RefTypeUnknown RefType = iota
	RefTypeBranch
	RefTypeTag
)

var refPrefixes = map[string]RefType{
	"refs/heads/": RefTypeBranch,
	"refs/tags/":  RefTypeTag,
}

// RefDescriptions returns a map from commits to descriptions of the tip of each
// branch and tag of the given repository.
func (c *Client) RefDescriptions(ctx context.Context, repositoryID int) (_ map[string]RefDescription, err error) {
	ctx, endObservation := c.operations.refDescriptions.With(ctx, &err, observation.Args{LogFields: []log.Field{
		log.Int("repositoryID", repositoryID),
	}})
	defer endObservation(1, observation.Args{})

	args := []string{"for-each-ref", "--format=%(objectname):%(refname):%(HEAD):%(creatordate:iso8601-strict)"}
	for prefix := range refPrefixes {
		args = append(args, prefix)
	}

	out, err := c.execGitCommand(ctx, repositoryID, args...)
	if err != nil {
		return nil, err
	}

	return parseRefDescriptions(strings.Split(out, "\n"))
}

// parseRefDescriptions converts the output of the for-each-ref command in the RefDescriptions
// method to a map from commits to RefDescription objects. Each line should conform to the format
// string `%(objectname):%(refname):%(HEAD):%(creatordate)`, where
//
// - %(objectname) is the 40-character revhash
// - %(refname) is the name of the tag or branch (prefixed with refs/heads/ or ref/tags/)
// - %(HEAD) is `*` if the branch is the default branch (and whitesace otherwise)
// - %(creatordate) is the ISO-formatted date the object was created
func parseRefDescriptions(lines []string) (map[string]RefDescription, error) {
	refDescriptions := make(map[string]RefDescription, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 4)
		if len(parts) != 4 {
			return nil, errors.Errorf(`unexpected output from git for-each-ref "%s"`, line)
		}

		commit := parts[0]
		isDefaultBranch := parts[2] == "*"

		var name string
		var refType RefType
		for prefix, typ := range refPrefixes {
			if strings.HasPrefix(parts[1], prefix) {
				name = parts[1][len(prefix):]
				refType = typ
				break
			}
		}
		if refType == RefTypeUnknown {
			return nil, errors.Errorf(`unexpected output from git for-each-ref "%s"`, line)
		}

		createdDate, err := time.Parse(time.RFC3339, parts[3])
		if err != nil {
			return nil, errors.Errorf(`unexpected output from git for-each-ref (bad date format) "%s"`, line)
		}

		refDescriptions[commit] = RefDescription{
			Name:            name,
			Type:            refType,
			IsDefaultBranch: isDefaultBranch,
			CreatedDate:     createdDate,
		}
	}

	return refDescriptions, nil
}

// RawContents returns the contents of a file in a particular commit of a repository.
func (c *Client) RawContents(ctx context.Context, repositoryID int, commit, file string) (_ []byte, err error) {
	ctx, endObservation := c.operations.rawContents.With(ctx, &err, observation.Args{LogFields: []log.Field{
		log.Int("repositoryID", repositoryID),
		log.String("commit", commit),
		log.String("file", file),
	}})
	defer endObservation(1, observation.Args{})

	out, err := c.execResolveRevGitCommand(ctx, repositoryID, commit, "show", fmt.Sprintf("%s:%s", commit, file))
	if err != nil {
		return nil, err
	}

	return []byte(out), err
}

// DirectoryChildren determines all children known to git for the given directory names via an invocation
// of git ls-tree. The keys of the resulting map are the input (unsanitized) dirnames, and the value of
// that key are the files nested under that directory.
func (c *Client) DirectoryChildren(ctx context.Context, repositoryID int, commit string, dirnames []string) (_ map[string][]string, err error) {
	ctx, endObservation := c.operations.directoryChildren.With(ctx, &err, observation.Args{LogFields: []log.Field{
		log.Int("repositoryID", repositoryID),
		log.String("commit", commit),
	}})
	defer endObservation(1, observation.Args{})

	return pathexistence.GitGetChildren(
		func(args ...string) (string, error) {
			return c.execResolveRevGitCommand(ctx, repositoryID, commit, args...)
		},
		commit,
		dirnames,
	)
}

// FileExists determines whether a file exists in a particular commit of a repository.
func (c *Client) FileExists(ctx context.Context, repositoryID int, commit, file string) (_ bool, err error) {
	ctx, endObservation := c.operations.fileExists.With(ctx, &err, observation.Args{LogFields: []log.Field{
		log.Int("repositoryID", repositoryID),
		log.String("commit", commit),
		log.String("file", file),
	}})
	defer endObservation(1, observation.Args{})

	repo, err := c.repositoryIDToRepo(ctx, repositoryID)
	if err != nil {
		return false, err
	}

	if _, err := git.ResolveRevision(ctx, repo, commit, git.ResolveRevisionOptions{}); err != nil {
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
func (c *Client) ListFiles(ctx context.Context, repositoryID int, commit string, pattern *regexp.Regexp) (_ []string, err error) {
	ctx, endObservation := c.operations.listFiles.With(ctx, &err, observation.Args{LogFields: []log.Field{
		log.Int("repositoryID", repositoryID),
		log.String("commit", commit),
		log.String("pattern", pattern.String()),
	}})
	defer endObservation(1, observation.Args{})

	out, err := c.execResolveRevGitCommand(ctx, repositoryID, commit, "ls-tree", "--name-only", "-r", commit, "--")
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

// ResolveRevision returns the absolute commit for a commit-ish spec.
func (c *Client) ResolveRevision(ctx context.Context, repositoryID int, versionString string) (commitID api.CommitID, err error) {
	ctx, endObservation := c.operations.resolveRevision.With(ctx, &err, observation.Args{LogFields: []log.Field{
		log.Int("repositoryID", repositoryID),
		log.String("versionString", versionString),
	}})
	defer endObservation(1, observation.Args{})

	repoName, err := c.repositoryIDToRepo(ctx, repositoryID)
	if err != nil {
		return "", err
	}
	commitID, err = git.ResolveRevision(ctx, repoName, versionString, git.ResolveRevisionOptions{})

	if err != nil {
		return "", errors.Wrap(err, "git.ResolveRevision")
	}

	return commitID, nil
}

// execGitCommand executes a git command for the given repository by identifier.
func (c *Client) execGitCommand(ctx context.Context, repositoryID int, args ...string) (string, error) {
	return c.execResolveRevGitCommand(ctx, repositoryID, "", args...)
}

// execResolveRevGitCommand executes a git command for the given repository by identifier if the
// given revision is resolvable prior to running the command.
func (c *Client) execResolveRevGitCommand(ctx context.Context, repositoryID int, revision string, args ...string) (string, error) {
	repo, err := c.repositoryIDToRepo(ctx, repositoryID)
	if err != nil {
		return "", err
	}

	cmd := gitserver.DefaultClient.Command("git", args...)
	cmd.Repo = repo

	out, err := cmd.CombinedOutput(ctx)
	if err == nil {
		return string(bytes.TrimSpace(out)), nil
	}

	// If the repo doesn't exist don't bother trying to resolve the commit. Otherwise,
	// if we're returning an error, try to resolve revision that was the target of the
	// command (if any). If the revision fails to resolve, we return an instance of a
	// RevisionNotFoundError error instead of an "exit 128".
	if revision != "" && !vcs.IsRepoNotExist(err) {
		if _, err := git.ResolveRevision(ctx, repo, revision, git.ResolveRevisionOptions{}); err != nil {
			return "", errors.Wrap(err, "git.ResolveRevision")
		}
	}

	// If we didn't expect a particular revision to exist, or we did but it
	// resolved without error, return the original error as the command had
	// failed for another reason.
	return "", errors.Wrap(err, "gitserver.Command")
}

// repositoryIDToRepo creates a api.RepoName from a repository identifier.
func (c *Client) repositoryIDToRepo(ctx context.Context, repositoryID int) (api.RepoName, error) {
	repoName, err := c.dbStore.RepoName(ctx, repositoryID)
	if err != nil {
		return "", errors.Wrap(err, "dbstore.RepoName")
	}

	return api.RepoName(repoName), nil
}
