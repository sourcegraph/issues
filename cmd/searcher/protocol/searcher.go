// Package protocol contains structures used by the searcher API.
package protocol

import (
	"fmt"
	"strings"

	"github.com/sourcegraph/sourcegraph/internal/api"
)

// Request represents a request to searcher
type Request struct {
	// Repo is the name of the repository to search. eg "github.com/gorilla/mux"
	Repo api.RepoName

	// URL specifies the repository's Git remote URL (for gitserver). It is optional. See
	// (gitserver.ExecRequest).URL for documentation on what it is used for.
	URL string

	// Commit is which commit to search. It is required to be resolved,
	// not a ref like HEAD or master. eg
	// "599cba5e7b6137d46ddf58fb1765f5d928e69604"
	Commit api.CommitID

	PatternInfo

	// The amount of time to wait for a repo archive to fetch.
	// It is parsed with time.ParseDuration.
	//
	// This timeout should be low when searching across many repos
	// so that unfetched repos don't delay the search, and because we are likely
	// to get results from the repos that have already been fetched.
	//
	// This timeout should be high when searching across a single repo
	// because returning results slowly is better than returning no results at all.
	//
	// This only times out how long we wait for the fetch request;
	// the fetch will still happen in the background so future requests don't have to wait.
	FetchTimeout string

	// The deadline for the search request.
	// It is parsed with time.Time.UnmarshalText.
	Deadline string
}

// PatternInfo describes a search request on a repo. Most of the fields
// are based on PatternInfo used in vscode.
type PatternInfo struct {
	// Pattern is the search query. It is a regular expression if IsRegExp
	// is true, otherwise a fixed string. eg "route variable"
	Pattern string

	// IsNegated if true will invert the matching logic for regexp searches. IsNegated=true is
	// not supported for structural searches.
	IsNegated bool

	// IsRegExp if true will treat the Pattern as a regular expression.
	IsRegExp bool

	// IsStructuralPat if true will treat the pattern as a Comby structural search pattern.
	IsStructuralPat bool

	// IsWordMatch if true will only match the pattern at word boundaries.
	IsWordMatch bool

	// IsCaseSensitive if false will ignore the case of text and pattern
	// when finding matches.
	IsCaseSensitive bool

	// ExcludePattern is a pattern that may not match the returned files' paths.
	// eg '**/node_modules'
	ExcludePattern string

	// IncludePatterns is a list of patterns that must *all* match the returned
	// files' paths.
	// eg '**/node_modules'
	//
	// The patterns are ANDed together; a file's path must match all patterns
	// for it to be kept. That is also why it is a list (unlike the singular
	// ExcludePattern); it is not possible in general to construct a single
	// glob or Go regexp that represents multiple such patterns ANDed together.
	IncludePatterns []string

	// IncludeExcludePatternAreRegExps indicates that ExcludePattern, IncludePattern,
	// and IncludePatterns are regular expressions (not globs).
	PathPatternsAreRegExps bool

	// IncludeExcludePatternAreCaseSensitive indicates that ExcludePattern, IncludePattern,
	// and IncludePatterns are case sensitive.
	PathPatternsAreCaseSensitive bool

	// FileMatchLimit limits the number of files with matches that are returned.
	FileMatchLimit int

	// PatternMatchesPath is whether the pattern should be matched against the content
	// of files.
	PatternMatchesContent bool

	// PatternMatchesPath is whether a file whose path matches Pattern (but whose contents don't) should be
	// considered a match.
	PatternMatchesPath bool

	// Languages is the languages passed via the lang filters (e.g., "lang:c")
	Languages []string

	// CombyRule is a rule that constrains matching for structural search. It only applies when IsStructuralPat is true.
	CombyRule string
}

func (p *PatternInfo) String() string {
	args := []string{fmt.Sprintf("%q", p.Pattern)}
	if p.IsRegExp {
		args = append(args, "re")
	}
	if p.IsStructuralPat {
		if p.CombyRule != "" {
			args = append(args, fmt.Sprintf("comby:%s", p.CombyRule))
		} else {
			args = append(args, "comby")
		}
	}
	if p.IsWordMatch {
		args = append(args, "word")
	}
	if p.IsCaseSensitive {
		args = append(args, "case")
	}
	if !p.PatternMatchesContent {
		args = append(args, "nocontent")
	}
	if !p.PatternMatchesPath {
		args = append(args, "nopath")
	}
	if p.FileMatchLimit > 0 {
		args = append(args, fmt.Sprintf("filematchlimit:%d", p.FileMatchLimit))
	}
	for _, lang := range p.Languages {
		args = append(args, fmt.Sprintf("lang:%s", lang))
	}

	path := "glob"
	if p.PathPatternsAreRegExps {
		path = "f"
	}
	if p.PathPatternsAreCaseSensitive {
		path = "F"
	}
	if p.ExcludePattern != "" {
		args = append(args, fmt.Sprintf("-%s:%q", path, p.ExcludePattern))
	}
	for _, inc := range p.IncludePatterns {
		args = append(args, fmt.Sprintf("%s:%q", path, inc))
	}

	return fmt.Sprintf("PatternInfo{%s}", strings.Join(args, ","))
}

// Response represents the response from a Search request.
type Response struct {
	Matches []FileMatch

	// LimitHit is true if Matches may not include all FileMatches because a match limit was hit.
	LimitHit bool

	// DeadlineHit is true if Matches may not include all FileMatches because a deadline was hit.
	DeadlineHit bool
}

// FileMatch is the struct used by vscode to receive search results
type FileMatch struct {
	Path        string
	LineMatches []LineMatch
	// MatchCount is the number of matches. Different from len(LineMatches), as multiple lines may correspond to one logical match.
	MatchCount int

	// LimitHit is true if LineMatches may not include all LineMatches.
	LimitHit bool
}

// LineMatch is the struct used by vscode to receive search results for a line.
type LineMatch struct {
	// Preview is the matched line.
	Preview string

	// LineNumber is the 0-based line number. Note: Our editors present
	// 1-based line numbers, but internally vscode uses 0-based.
	LineNumber int

	// OffsetAndLengths is a slice of 2-tuples (Offset, Length)
	// representing each match on a line.
	// Offsets and lengths are measured in characters, not bytes.
	OffsetAndLengths [][2]int

	// LimitHit is true if OffsetAndLengths may not include all OffsetAndLengths.
	LimitHit bool
}
