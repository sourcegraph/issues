package git

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/avelino/slugify"
	"github.com/pkg/errors"
	"github.com/rainycape/unidecode"

	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/gitserver"
	"github.com/sourcegraph/sourcegraph/internal/lazyregexp"
	"github.com/sourcegraph/sourcegraph/internal/trace/ot"
	"github.com/sourcegraph/sourcegraph/internal/vcs"
)

// HumanReadableBranchName returns a human readable branch name from the
// given text. It replaces unicode characters with their ASCII equivalent
// or similar and connects each component with a dash.
//
// Example: "Change coördination mechanism" -> "change-coordination-mechanism"
func HumanReadableBranchName(text string) string {
	name := slugify.Slugify(unidecode.Unidecode(text))

	const length = 60
	if len(name) <= length {
		return name
	}

	// Find the last word separator so we don't cut in the middle of a word.
	// If the word separator is found in the very first part of the name we don't
	// cut there because it'd leave out too much of it.
	sep := strings.LastIndexByte(name[:length], '-')
	if sep >= 0 && float32(sep)/float32(length) >= 0.2 {
		return name[:sep]
	}

	return name[:length]
}

// EnsureRefPrefix checks whether the ref is a full ref and contains the
// "refs/heads" prefix (i.e. "refs/heads/master") or just an abbreviated ref
// (i.e. "master") and adds the "refs/heads/" prefix if the latter is the case.
func EnsureRefPrefix(ref string) string {
	return "refs/heads/" + strings.TrimPrefix(ref, "refs/heads/")
}

// AbbreviateRef removes the "refs/heads/" prefix from a given ref. If the ref
// doesn't have the prefix, it returns it unchanged.
func AbbreviateRef(ref string) string {
	return strings.TrimPrefix(ref, "refs/heads/")
}

// A Branch is a VCS branch.
type Branch struct {
	// Name is the name of this branch.
	Name string `json:"Name,omitempty"`
	// Head is the commit ID of this branch's head commit.
	Head api.CommitID `json:"Head,omitempty"`
	// Commit optionally contains commit information for this branch's head commit.
	// It is populated if IncludeCommit option is set.
	Commit *Commit `json:"Commit,omitempty"`
	// Counts optionally contains the commit counts relative to specified branch.
	Counts *BehindAhead `json:"Counts,omitempty"`
}

// BranchesOptions specifies options for the list of branches returned by
// (Repository).Branches.
type BranchesOptions struct {
	// MergedInto will cause the returned list to be restricted to only
	// branches that were merged into this branch name.
	MergedInto string `json:"MergedInto,omitempty" url:",omitempty"`
	// IncludeCommit controls whether complete commit information is included.
	IncludeCommit bool `json:"IncludeCommit,omitempty" url:",omitempty"`
	// BehindAheadBranch specifies a branch name. If set to something other than blank
	// string, then each returned branch will include a behind/ahead commit counts
	// information against the specified base branch. If left blank, then branches will
	// not include that information and their Counts will be nil.
	BehindAheadBranch string `json:"BehindAheadBranch,omitempty" url:",omitempty"`
	// ContainsCommit filters the list of branches to only those that
	// contain a specific commit ID (if set).
	ContainsCommit string `json:"ContainsCommit,omitempty" url:",omitempty"`
}

// A Tag is a VCS tag.
type Tag struct {
	Name         string `json:"Name,omitempty"`
	api.CommitID `json:"CommitID,omitempty"`
	CreatorDate  time.Time
}

// BehindAhead is a set of behind/ahead counts.
type BehindAhead struct {
	Behind uint32 `json:"Behind,omitempty"`
	Ahead  uint32 `json:"Ahead,omitempty"`
}

type Branches []*Branch

func (p Branches) Len() int           { return len(p) }
func (p Branches) Less(i, j int) bool { return p[i].Name < p[j].Name }
func (p Branches) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

// ByAuthorDate sorts by author date. Requires full commit information to be included.
type ByAuthorDate []*Branch

func (p ByAuthorDate) Len() int { return len(p) }
func (p ByAuthorDate) Less(i, j int) bool {
	return p[i].Commit.Author.Date.Before(p[j].Commit.Author.Date)
}
func (p ByAuthorDate) Swap(i, j int) { p[i], p[j] = p[j], p[i] }

type Tags []*Tag

func (p Tags) Len() int           { return len(p) }
func (p Tags) Less(i, j int) bool { return p[i].Name < p[j].Name }
func (p Tags) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

// branchFilter is a filter for branch names.
// If not empty, only contained branch names are allowed. If empty, all names are allowed.
// The map should be made so it's not nil.
type branchFilter map[string]struct{}

// allows will return true if the current filter set-up validates against
// the passed string. If there are no filters, all strings pass.
func (f branchFilter) allows(name string) bool {
	if len(f) == 0 {
		return true
	}
	_, ok := f[name]
	return ok
}

// add adds a slice of strings to the filter.
func (f branchFilter) add(list []string) {
	for _, l := range list {
		f[l] = struct{}{}
	}
}

// ListBranches returns a list of all branches in the repository.
func ListBranches(ctx context.Context, repo gitserver.Repo, opt BranchesOptions) ([]*Branch, error) {
	span, ctx := ot.StartSpanFromContext(ctx, "Git: Branches")
	span.SetTag("Opt", opt)
	defer span.Finish()

	f := make(branchFilter)
	if opt.MergedInto != "" {
		b, err := branches(ctx, repo, "--merged", opt.MergedInto)
		if err != nil {
			return nil, err
		}
		f.add(b)
	}
	if opt.ContainsCommit != "" {
		b, err := branches(ctx, repo, "--contains="+opt.ContainsCommit)
		if err != nil {
			return nil, err
		}
		f.add(b)
	}

	refs, err := showRef(ctx, repo, "--heads")
	if err != nil {
		return nil, err
	}

	var branches []*Branch
	for _, ref := range refs {
		name := strings.TrimPrefix(ref.Name, "refs/heads/")
		if !f.allows(name) {
			continue
		}

		branch := &Branch{Name: name, Head: ref.CommitID}
		if opt.IncludeCommit {
			branch.Commit, err = getCommit(ctx, repo, nil, ref.CommitID, ResolveRevisionOptions{})
			if err != nil {
				return nil, err
			}
		}
		if opt.BehindAheadBranch != "" {
			branch.Counts, err = GetBehindAhead(ctx, repo, "refs/heads/"+opt.BehindAheadBranch, "refs/heads/"+name)
			if err != nil {
				return nil, err
			}
		}
		branches = append(branches, branch)
	}
	return branches, nil
}

// branches runs the `git branch` command followed by the given arguments and
// returns the list of branches if successful.
func branches(ctx context.Context, repo gitserver.Repo, args ...string) ([]string, error) {
	cmd := gitserver.DefaultClient.Command("git", append([]string{"branch"}, args...)...)
	cmd.Repo = repo
	out, err := cmd.Output(ctx)
	if err != nil {
		return nil, fmt.Errorf("exec %v in %s failed: %v (output follows)\n\n%s", cmd.Args, cmd.Repo, err, out)
	}
	lines := strings.Split(string(out), "\n")
	lines = lines[:len(lines)-1]
	branches := make([]string, len(lines))
	for i, line := range lines {
		branches[i] = line[2:]
	}
	return branches, nil
}

// GetBehindAhead returns the behind/ahead commit counts information for right vs. left (both Git
// revspecs).
func GetBehindAhead(ctx context.Context, repo gitserver.Repo, left, right string) (*BehindAhead, error) {
	span, ctx := ot.StartSpanFromContext(ctx, "Git: BehindAhead")
	defer span.Finish()

	if err := checkSpecArgSafety(left); err != nil {
		return nil, err
	}
	if err := checkSpecArgSafety(right); err != nil {
		return nil, err
	}

	cmd := gitserver.DefaultClient.Command("git", "rev-list", "--count", "--left-right", fmt.Sprintf("%s...%s", left, right))
	cmd.Repo = repo
	out, err := cmd.Output(ctx)
	if err != nil {
		return nil, err
	}
	behindAhead := strings.Split(strings.TrimSuffix(string(out), "\n"), "\t")
	b, err := strconv.ParseUint(behindAhead[0], 10, 0)
	if err != nil {
		return nil, err
	}
	a, err := strconv.ParseUint(behindAhead[1], 10, 0)
	if err != nil {
		return nil, err
	}
	return &BehindAhead{Behind: uint32(b), Ahead: uint32(a)}, nil
}

// ListTags returns a list of all tags in the repository.
func ListTags(ctx context.Context, repo gitserver.Repo) ([]*Tag, error) {
	span, ctx := ot.StartSpanFromContext(ctx, "Git: Tags")
	defer span.Finish()

	// Support both lightweight tags and tag objects. For creatordate, use an %(if) to prefer the
	// taggerdate for tag objects, otherwise use the commit's committerdate (instead of just always
	// using committerdate).
	cmd := gitserver.DefaultClient.Command("git", "tag", "--list", "--sort", "-creatordate", "--format", "%(if)%(*objectname)%(then)%(*objectname)%(else)%(objectname)%(end)%00%(refname:short)%00%(if)%(creatordate:unix)%(then)%(creatordate:unix)%(else)%(*creatordate:unix)%(end)")
	cmd.Repo = repo
	out, err := cmd.CombinedOutput(ctx)
	if err != nil {
		if vcs.IsRepoNotExist(err) {
			return nil, err
		}
		return nil, errors.WithMessage(err, fmt.Sprintf("git command %v failed (output: %q)", cmd.Args, out))
	}

	return parseTags(out)
}

func parseTags(in []byte) ([]*Tag, error) {
	in = bytes.TrimSuffix(in, []byte("\n")) // remove trailing newline
	if len(in) == 0 {
		return nil, nil // no tags
	}
	lines := bytes.Split(in, []byte("\n"))
	tags := make([]*Tag, len(lines))
	for i, line := range lines {
		parts := bytes.SplitN(line, []byte("\x00"), 3)
		if len(parts) != 3 {
			return nil, fmt.Errorf("invalid git tag list output line: %q", line)
		}

		tag := &Tag{
			Name:     string(parts[1]),
			CommitID: api.CommitID(parts[0]),
		}

		date, err := strconv.ParseInt(string(parts[2]), 10, 64)
		if err == nil {
			tag.CreatorDate = time.Unix(date, 0).UTC()
		}

		tags[i] = tag
	}
	return tags, nil
}

type byteSlices [][]byte

func (p byteSlices) Len() int           { return len(p) }
func (p byteSlices) Less(i, j int) bool { return bytes.Compare(p[i], p[j]) < 0 }
func (p byteSlices) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

// ListRefs returns a list of all refs in the repository.
func ListRefs(ctx context.Context, repo gitserver.Repo) ([]Ref, error) {
	span, ctx := ot.StartSpanFromContext(ctx, "Git: ListRefs")
	defer span.Finish()
	return showRef(ctx, repo)
}

// Ref describes a Git ref.
type Ref struct {
	Name     string // the full name of the ref (e.g., "refs/heads/mybranch")
	CommitID api.CommitID
}

func showRef(ctx context.Context, repo gitserver.Repo, args ...string) ([]Ref, error) {
	cmd := gitserver.DefaultClient.Command("git", "show-ref")
	cmd.Args = append(cmd.Args, args...)
	cmd.Repo = repo
	out, err := cmd.CombinedOutput(ctx)
	if err != nil {
		if vcs.IsRepoNotExist(err) {
			return nil, err
		}
		// Exit status of 1 and no output means there were no
		// results. This is not a fatal error.
		if cmd.ExitStatus == 1 && len(out) == 0 {
			return nil, nil
		}
		return nil, errors.WithMessage(err, fmt.Sprintf("git command %v failed (output: %q)", cmd.Args, out))
	}

	out = bytes.TrimSuffix(out, []byte("\n")) // remove trailing newline
	lines := bytes.Split(out, []byte("\n"))
	sort.Sort(byteSlices(lines)) // sort for consistency
	refs := make([]Ref, len(lines))
	for i, line := range lines {
		if len(line) <= 41 {
			return nil, errors.New("unexpectedly short (<=41 bytes) line in `git show-ref ...` output")
		}
		id := line[:40]
		name := line[41:]
		refs[i] = Ref{Name: string(name), CommitID: api.CommitID(id)}
	}
	return refs, nil
}

var invalidBranch = lazyregexp.New(`\.\.|/\.|\.lock$|[\000-\037\177 ~^:?*[]+|^/|/$|//|\.$|@{|^@$|\\`)

// ValidateBranchName returns false if the given string is not a valid branch name.
// It follows the rules here: https://git-scm.com/docs/git-check-ref-format
// NOTE: It does not require a slash as mentioned in point 2.
func ValidateBranchName(branch string) bool {
	return !(invalidBranch.MatchString(branch) || strings.EqualFold(branch, "head"))
}
