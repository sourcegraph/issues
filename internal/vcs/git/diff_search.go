package git

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/cockroachdb/errors"

	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/gitserver"
	"github.com/sourcegraph/sourcegraph/internal/pathmatch"
	"github.com/sourcegraph/sourcegraph/internal/trace"
)

// TextSearchOptions contains common options for text search commands.
type TextSearchOptions struct {
	Pattern         string // the pattern to look for
	IsRegExp        bool   // whether the pattern is a regexp (if false, treated as exact string)
	IsCaseSensitive bool   // whether the pattern should be matched case-sensitively
}

// PathOptions contains common options for commands that can be limited
// to only certain paths.
type PathOptions struct {
	IncludePatterns []string // include paths matching all of these patterns
	ExcludePattern  string   // exclude paths matching any of these patterns
	IsRegExp        bool     // whether the pattern is a regexp (if false, treated as exact string)
	IsCaseSensitive bool     // whether the pattern should be matched case-sensitively
}

// CompilePathMatcher compiles the path options into a PathMatcher.
func CompilePathMatcher(options PathOptions) (pathmatch.PathMatcher, error) {
	return pathmatch.CompilePathPatterns(
		options.IncludePatterns, options.ExcludePattern,
		pathmatch.CompileOptions{CaseSensitive: options.IsCaseSensitive, RegExp: options.IsRegExp},
	)
}

// RawLogDiffSearchOptions specifies options to (Repository).RawLogDiffSearch.
type RawLogDiffSearchOptions struct {
	// Query specifies the search query to find.
	Query TextSearchOptions

	// MatchChangedOccurrenceCount makes the operation run `git log -S` not `git log -G`.
	// See `git log --help` for more information.
	MatchChangedOccurrenceCount bool

	// Diff is whether the diff should be computed and returned.
	Diff bool

	// OnlyMatchingHunks makes the diff only include hunks that match the query. If false,
	// all hunks from files that match the query are included.
	OnlyMatchingHunks bool

	// Paths specifies the paths to include/exclude.
	Paths PathOptions

	// FormatArgs is a list of format args that are passed to the `git log` command.
	// Because the output is parsed, it is expected to be in a known format. If the
	// FormatArgs does not match one of the server's expected values, the operation
	// will fail.
	//
	// If nil, the default format args are used.
	FormatArgs []string

	// RawArgs is a list of non-format args that are passed to the `git log` command.
	// It should not contain any "--" elements; those should be passed using the Paths
	// field.
	//
	// No arguments that affect the format of the output should be present in this
	// slice.
	Args []string
}

// LogCommitSearchResult describes a matching diff from (Repository).RawLogDiffSearch.
type LogCommitSearchResult struct {
	Commit         Commit      // the commit whose diff was matched
	Diff           *RawDiff    // the diff, with non-matching/irrelevant portions deleted (respecting diff syntax)
	DiffHighlights []Highlight // highlighted query matches in the diff

	// Refs is the list of ref names of this commit (from `git log --decorate`).
	Refs []string

	// SourceRefs is the list of ref names by which this commit was reached. (See
	// `git log --help` documentation on the `--source` flag.)
	SourceRefs []string

	// Incomplete indicates that this result may represent a subset of the actual data.
	// This can occur when the underlying command returns early due to an impending
	// timeout.
	Incomplete bool
}

// LogCommitSearchEvent are emitted by RawLogDiffSearchStream
type LogCommitSearchEvent struct {
	// Results are new commit results found.
	Results []*LogCommitSearchResult

	// Complete is false when the results may have been parsed from only
	// partial output from the underlying git command (because, e.g., it timed
	// out during execution and only returned partial output).
	//
	// Complete defaults to true, but once false will remain false.
	Complete bool

	// Error is non-nil if an error occurred. It will be the last event if
	// set.
	//
	// Note: Results will be empty if Error is set.
	Error error
}

// A RawDiff represents changes between two commits.
type RawDiff struct {
	Raw string // the raw diff output
}

// Highlight represents a highlighted region in a string.
type Highlight struct {
	Line      int // the 1-indexed line number
	Character int // the 1-indexed character on the line
	Length    int // the length of the highlight, in characters (on the same line)
}

var validRawLogDiffSearchFormatArgs = [][]string{
	{"--no-merges", "-z", "--decorate=full", "--patch", logFormatWithRefs},
	{"--no-merges", "-z", "--decorate=full", logFormatWithRefs},
}

func isValidRawLogDiffSearchFormatArgs(formatArgs []string) bool {
	for _, validArgs := range validRawLogDiffSearchFormatArgs {
		if reflect.DeepEqual(formatArgs, validArgs) {
			return true
		}
	}
	return false
}

// RawLogDiffSearch wraps RawLogDiffSearchStream providing a blocking API. See
// RawLogDiffSearchStream.
func RawLogDiffSearch(ctx context.Context, repo api.RepoName, opt RawLogDiffSearchOptions) (results []*LogCommitSearchResult, complete bool, err error) {
	for event := range RawLogDiffSearchStream(ctx, repo, opt) {
		results = append(results, event.Results...)
		complete = event.Complete
		err = event.Error
	}
	return results, complete, err
}

// RawLogDiffSearchStream runs a raw `git log` command that is expected to
// return logs with patches. It returns a subset of the output, including only
// hunks that actually match the given pattern.
//
// The returned channel must be read until closed, otherwise you may leak
// resources.
func RawLogDiffSearchStream(ctx context.Context, repo api.RepoName, opt RawLogDiffSearchOptions) <-chan LogCommitSearchEvent {
	c := make(chan LogCommitSearchEvent)
	go func() {
		defer close(c)
		_, _ = doLogDiffSearchStream(ctx, repo, opt, c)
	}()

	return c
}

// doLogDiffSearchStream is called by RawLogDiffSearchStream to send events
// down c. It uses named return values to simplify sending errors down the
// channel. The return values can be ignored.
func doLogDiffSearchStream(ctx context.Context, repo api.RepoName, opt RawLogDiffSearchOptions, c chan LogCommitSearchEvent) (complete bool, err error) {
	resultCount := 0
	tr, ctx := trace.New(ctx, "Git: RawLogDiffSearch", fmt.Sprintf("%+v, timeout=%s", opt, deadlineLabel(ctx)))
	defer func() {
		tr.LazyPrintf("%d results, complete=%v, err=%v", resultCount, complete, err)
		tr.SetError(err)
		tr.Finish()
	}()

	// This defer will read the named return values. This is a convenient way
	// to send errors down the channel, since we only want to do this once.
	empty := true
	defer func() {
		// We do best-effort in case of timeout. So we clear out the error and
		// indicate the results are incomplete.
		if errors.Is(err, context.DeadlineExceeded) {
			c <- LogCommitSearchEvent{Complete: false}
			complete = false
			err = nil
			return
		}

		// Send a final event if we had an error or if we hadn't sent down the
		// channel.
		if err != nil || empty {
			c <- LogCommitSearchEvent{
				Complete: complete,
				Error:    err,
			}
		}
	}()

	if Mocks.RawLogDiffSearch != nil {
		results, complete, err := Mocks.RawLogDiffSearch(opt)
		if len(results) > 0 {
			c <- LogCommitSearchEvent{
				Results:  results,
				Complete: complete,
			}
			empty = false
		}
		return complete, err
	}

	if opt.FormatArgs == nil {
		if opt.Diff {
			opt.FormatArgs = validRawLogDiffSearchFormatArgs[0] // with --patch
		} else {
			opt.FormatArgs = validRawLogDiffSearchFormatArgs[1] // without --patch
		}
	}
	if opt.FormatArgs != nil && !isValidRawLogDiffSearchFormatArgs(opt.FormatArgs) {
		return false, errors.Errorf("invalid FormatArgs: %q", opt.FormatArgs)
	}
	for _, arg := range opt.Args {
		if arg == "--" {
			return false, errors.Errorf("invalid Args (must not contain \"--\" element): %q", opt.Args)
		}
	}

	if opt.Query.IsCaseSensitive != opt.Paths.IsCaseSensitive {
		// These options can't be set separately in `git log`, so fail.
		return false, errors.Errorf("invalid options: Query.IsCaseSensitive != Paths.IsCaseSensitive")
	}

	// We do a search with git log returning just the commits (and source sha).
	onelineCmd, err := rawLogSearchCmd(ctx, repo, opt)
	if err != nil {
		return false, err
	}
	onelineReader, err := gitserver.StdoutReader(ctx, onelineCmd)
	if err != nil {
		return false, err
	}
	defer onelineReader.Close()

	// We batch up the commits from onelineReader to amortize the cost of
	// running rawShowSearch
	const (
		maxBatchSize = 100
		debounce     = 10 * time.Millisecond
	)
	nextCommits, nextCommitsClose := logOnelineBatchScanner(logOnelineScanner(onelineReader), maxBatchSize, debounce)
	defer nextCommitsClose()

	// We then search each commit in batches to further filter the results.
	complete = true
	var cache refResolveCache
	for {
		commits, err := nextCommits()
		if err == io.EOF {
			break
		} else if err != nil {
			return complete, err
		}

		results, complete2, err := rawShowSearch(ctx, repo, opt, &cache, commits)
		complete = complete && complete2

		if len(results) > 0 {
			c <- LogCommitSearchEvent{
				Results:  results,
				Complete: true,
			}
			empty = false
			resultCount += len(results)
		}

		if err != nil {
			return complete, err
		}
	}

	return complete, nil
}

func rawLogSearchCmd(ctx context.Context, repo api.RepoName, opt RawLogDiffSearchOptions) (*gitserver.Cmd, error) {
	args := []string{"log"}
	args = append(args, opt.Args...)
	if !isAllowedGitCmd(args) {
		return nil, errors.Errorf("command failed: %q is not a allowed git command", args)
	}

	// TODO(keegan 2021-02-04) Now that git log directly supports a format
	// string which includes '%S' (--source) we may be able to directly call
	// log, instead of piping log into show.
	onelineArgs := append([]string{}, args...)
	onelineArgs = append(onelineArgs,
		"-z",
		// %H :: commit hash
		// %S :: ref name given on the command line by which the commit was
		//       reached (like git log --source), only works with git
		//       log. Since Git 2.21 (Q1 2019)
		"--format=%H %S",
		"--no-color",
		"--no-patch",
		"--no-merges",
	)
	onelineArgs = append(onelineArgs, logDiffCommonArgs(opt)...)

	cmd := gitserver.DefaultClient.Command("git", onelineArgs...)
	cmd.Repo = repo
	return cmd, nil
}

// rawShowSearch runs git show on each commit in onelineCommits. We need to do
// this to further filter hunks.
func rawShowSearch(ctx context.Context, repo api.RepoName, opt RawLogDiffSearchOptions, cache *refResolveCache, onelineCommits []*onelineCommit) (results []*LogCommitSearchResult, complete bool, err error) {
	if len(onelineCommits) == 0 {
		return nil, true, nil
	}

	// Build a map of commit -> source ref.
	commitSourceRefs := make(map[string]string, len(onelineCommits))
	for _, c := range onelineCommits {
		commitSourceRefs[c.sha1] = c.sourceRef
	}

	// Even though we've already searched using the query, we need to
	// search the returned diff again to filter to only matching hunks
	// and to highlight matches.
	var query *regexp.Regexp
	if pattern := opt.Query.Pattern; pattern != "" {
		if !opt.Query.IsRegExp {
			pattern = regexp.QuoteMeta(pattern)
		}
		if !opt.Query.IsCaseSensitive {
			pattern = "(?i:" + pattern + ")"
		}
		var err error
		query, err = regexp.Compile(pattern)
		if err != nil {
			return nil, false, err
		}
	}

	pathMatcher, err := compilePathMatcher(opt.Paths)
	if err != nil {
		return nil, false, err
	}

	// Now fetch the full commit data for all of the commits.
	commitOIDs := make([]string, len(onelineCommits))
	for i, c := range onelineCommits {
		commitOIDs[i] = c.sha1
	}
	showArgs := append([]string{}, "show")
	showArgs = append(showArgs, "--no-patch")  // will be overridden if opt.FormatArgs has --patch
	showArgs = append(showArgs, "--no-prefix") // do not prefix file names with a/ and b/
	showArgs = append(showArgs, opt.FormatArgs...)
	showArgs = append(showArgs, commitOIDs...)
	// Need --patch (TODO(sqs): or just --raw, which is smaller) if we are filtering by file paths,
	// because we post-filter by path since we need to support regexps. Just the commit message
	// alone would be insufficient for our post-filtering.
	hasPathFilters := opt.Paths.ExcludePattern != "" || len(opt.Paths.IncludePatterns) > 0
	if hasPathFilters {
		showArgs = append(showArgs, "--patch")
	}
	if !isAllowedGitCmd(showArgs) {
		return nil, false, errors.Errorf("command failed: %q is not a allowed git command", showArgs)
	}
	showCmd := gitserver.DefaultClient.Command("git", showArgs...)
	showCmd.Repo = repo
	data, complete, err := readUntilTimeout(ctx, showCmd)
	if err != nil {
		return nil, complete, err
	}
	for len(data) > 0 {
		var commit *Commit
		var refs []string
		var err error
		commit, refs, data, err = parseCommitFromLog(data)
		if err != nil {
			if !complete {
				// Partial data can yield parse errors, but we still want to return what we have.
				// We know all of the results we already found are complete, so we can return
				// immediately instead of marking the last one as incomplete.
				return results, false, nil
			}
			return nil, complete, err
		}

		result := &LogCommitSearchResult{
			Commit:     *commit,
			Refs:       refs,
			SourceRefs: []string{commitSourceRefs[string(commit.ID)]},
		}
		result.Refs, err = filterAndResolveRefs(ctx, repo, result.Refs, cache)
		if err == nil {
			result.SourceRefs, err = filterAndResolveRefs(ctx, repo, result.SourceRefs, cache)
		}
		sort.Strings(result.Refs)
		sort.Strings(result.SourceRefs)
		if err != nil {
			if ctx.Err() != nil {
				// Return partial data.
				complete = false
				return results, complete, err
			}
			return nil, false, err
		}

		hasMatch := true
		if len(data) == 0 || (len(data) >= 1 && data[0] == '\x00') {
			// No diff patch.
			if len(data) >= 1 {
				data = data[1:]
			}
			if hasPathFilters {
				hasMatch = false // patch was empty for the filtered paths, don't add to results
			}
		} else if len(data) >= 1 && data[0] == '\n' {
			data = data[1:]

			// Next is the diff patch.
			patchEnd := bytes.Index(data, []byte("\n\x00"))
			var rawDiff []byte
			if patchEnd != -1 {
				rawDiff = data[:patchEnd+1]
				data = data[patchEnd+2:]
			} else {
				// Rest of data is the diff patch.
				rawDiff = data
				data = nil
			}

			var err error
			rawDiff, result.DiffHighlights, err = filterAndHighlightDiff(rawDiff, query, opt.OnlyMatchingHunks, pathMatcher)
			if err != nil {
				return nil, false, err
			}
			if rawDiff == nil {
				hasMatch = false // patch was empty (after applying filters), don't add to results
			} else {
				result.Diff = &RawDiff{Raw: string(rawDiff)}
			}
		}

		if hasMatch {
			results = append(results, result)
		}
	}

	if !complete && len(results) > 0 {
		// The last result may have been parsed from an incomplete output stream (e.g., stdout
		// cut off halfway through), so mark it as such.
		results[len(results)-1].Incomplete = true
	}

	return results, complete, nil
}

func logDiffCommonArgs(opt RawLogDiffSearchOptions) []string {
	var args []string
	if opt.Query.Pattern != "" && opt.Diff {
		var queryArg string
		if opt.MatchChangedOccurrenceCount {
			queryArg = "-S"
		} else {
			queryArg = "-G"
		}
		args = append(args, queryArg+opt.Query.Pattern)
		if !opt.Query.IsCaseSensitive {
			args = append(args, "--regexp-ignore-case")
		}
		if opt.Query.IsRegExp {
			args = append(args, "--pickaxe-regex")
		}
	}
	if opt.Paths.IsRegExp {
		args = append(args, "--extended-regexp")
	}

	// If we have exclude paths, we need to effectively unset the --max-count because we can't
	// filter out changes that match the exclude path (because there's no way to use full
	// regexps in git pathspecs).
	//
	// TODO(sqs): use git pathspec %(...) extensions to reduce the number of cases where this is
	// necessary; see https://git-scm.com/docs/gitglossary.html#def_pathspec.
	var addMaxCount500 bool
	if opt.Paths.ExcludePattern != "" {
		addMaxCount500 = true
	}

	// Args we append after this don't need to be checked for allowlisting because "--"
	// precedes them.
	var pathspecs []string
	for _, p := range opt.Paths.IncludePatterns {
		// Roughly try to convert IncludePatterns (regexps) to git pathspecs (globs).
		glob, equiv := regexpToGlobBestEffort(p)
		if !opt.Paths.IsCaseSensitive && glob != "" {
			// This relies on regexpToGlobBestEffort not returning `:`-prefixed globs.
			glob = ":(icase)" + glob
		}
		if !equiv {
			addMaxCount500 = true
		}
		if glob != "" {
			pathspecs = append(pathspecs, glob)
		}
	}

	if addMaxCount500 {
		args = append(args, "--max-count=500") // TODO(sqs): 500 is arbitrary high number
	}
	args = append(args, "--")
	args = append(args, pathspecs...)

	return args
}

// cachedRefResolver is a short-lived cache for ref resolutions. Only use it for the lifetime of a
// single request and for a single repo.
type refResolveCache struct {
	mu      sync.Mutex
	results map[string]struct {
		target string
		err    error
	}
}

func (r *refResolveCache) resolveHEADSymbolicRef(ctx context.Context, repo api.RepoName) (target string, err error) {
	resolve := func() (string, error) {
		cmd := gitserver.DefaultClient.Command("git", "rev-parse", "--symbolic-full-name", "HEAD")
		cmd.Repo = repo
		stdout, err := cmd.Output(ctx)
		return string(bytes.TrimSpace(stdout)), err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.results == nil {
		r.results = map[string]struct {
			target string
			err    error
		}{}
	}
	const name = "HEAD" // only needed for HEAD right now
	e, ok := r.results[name]
	if !ok {
		e.target, e.err = resolve()
		r.results[name] = e
	}
	return e.target, e.err
}

// filterAndResolveRefs replaces "HEAD" entries with the names of the ref they refer to,
// and it omits "HEAD -> ..." entries.
func filterAndResolveRefs(ctx context.Context, repo api.RepoName, refs []string, cache *refResolveCache) ([]string, error) {
	filtered := refs[:0]
	for _, ref := range refs {
		if strings.HasPrefix(ref, "HEAD -> ") {
			continue
		}
		if ref == "HEAD" {
			var err error
			ref, err = cache.resolveHEADSymbolicRef(ctx, repo)
			if err != nil {
				return nil, err
			}
		} else if strings.HasPrefix(ref, "tag: ") {
			ref = strings.TrimPrefix(ref, "tag: ")
		}
		filtered = append(filtered, ref)
	}
	return filtered, nil
}

func deadlineLabel(ctx context.Context) string {
	if deadline, ok := ctx.Deadline(); ok {
		return time.Until(deadline).String()
	}
	return "unlimited"
}
