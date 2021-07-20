package search

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/endpoint"
	searchbackend "github.com/sourcegraph/sourcegraph/internal/search/backend"
	"github.com/sourcegraph/sourcegraph/internal/search/filter"
	"github.com/sourcegraph/sourcegraph/internal/search/query"
	"github.com/sourcegraph/sourcegraph/internal/search/result"
	"github.com/sourcegraph/sourcegraph/internal/types"
	"github.com/sourcegraph/sourcegraph/internal/vcs/git"
	"github.com/sourcegraph/sourcegraph/schema"
)

type TypeParameters interface {
	typeParametersValue()
}

func (CommitParameters) typeParametersValue()  {}
func (DiffParameters) typeParametersValue()    {}
func (SymbolsParameters) typeParametersValue() {}
func (TextParameters) typeParametersValue()    {}

type CommitParameters struct {
	RepoRevs           *RepositoryRevisions
	PatternInfo        *CommitPatternInfo
	Query              query.Q
	Diff               bool
	ExtraMessageValues []string
}

type DiffParameters struct {
	Repo    api.RepoName
	Options git.RawLogDiffSearchOptions
}

// CommitPatternInfo is the data type that describes the properties of
// a pattern used for commit search.
type CommitPatternInfo struct {
	Pattern         string
	IsRegExp        bool
	IsCaseSensitive bool
	FileMatchLimit  int32

	IncludePatterns []string
	ExcludePattern  string

	PathPatternsAreRegExps       bool
	PathPatternsAreCaseSensitive bool
}

type SymbolsParameters struct {
	// Repo is the name of the repository to search in.
	Repo api.RepoName `json:"repo"`

	// CommitID is the commit to search in.
	CommitID api.CommitID `json:"commitID"`

	// Query is the search query.
	Query string

	// IsRegExp if true will treat the Pattern as a regular expression.
	IsRegExp bool

	// IsCaseSensitive if false will ignore the case of query and file pattern
	// when finding matches.
	IsCaseSensitive bool

	// IncludePatterns is a list of regexes that symbol's file paths
	// need to match to get included in the result
	//
	// The patterns are ANDed together; a file's path must match all patterns
	// for it to be kept. That is also why it is a list (unlike the singular
	// ExcludePattern); it is not possible in general to construct a single
	// glob or Go regexp that represents multiple such patterns ANDed together.
	IncludePatterns []string

	// ExcludePattern is an optional regex that symbol's file paths
	// need to match to get included in the result
	ExcludePattern string

	// First indicates that only the first n symbols should be returned.
	First int
}

// GlobalSearchMode designates code paths which optimize performance for global
// searches, i.e., literal or regexp, indexed searches without repo: filter.
type GlobalSearchMode int

const (
	DefaultMode GlobalSearchMode = iota

	// ZoektGlobalSearch designates a performance optimised code path for indexed
	// searches. For a global search we don't need to resolve repos before searching
	// shards on Zoekt, instead we can resolve repos and call Zoekt concurrently.
	//
	// Note: Even for a global search we have to resolve repos to filter search results
	// returned by Zoekt.
	ZoektGlobalSearch

	// SearcherOnly designated a code path on which we skip indexed search, even if
	// the user specified index:yes. SearcherOnly is used in conjunction with
	// ZoektGlobalSearch and designates the non-indexed part of the performance
	// optimised code path.
	SearcherOnly

	// Disables file/path search. Used only in conjunction with ZoektGlobalSearch on
	// Sourcegraph.com.
	NoFilePath
)

var globalSearchModeStrings = map[GlobalSearchMode]string{
	ZoektGlobalSearch: "ZoektGlobalSearch",
	SearcherOnly:      "SearcherOnly",
	NoFilePath:        "NoFilePath",
}

func (m GlobalSearchMode) String() string {
	if s, ok := globalSearchModeStrings[m]; ok {
		return s
	}
	return "None"
}

// TextParameters are the parameters passed to a search backend. It contains the Pattern
// to search for, as well as the hydrated list of repository revisions to
// search. It defines behavior for text search on repository names, file names, and file content.
type TextParameters struct {
	PatternInfo *TextPatternInfo
	RepoOptions RepoOptions
	ResultTypes result.Types
	Timeout     time.Duration

	// deprecated
	RepoPromise *RepoPromise

	// perf: For global queries, we only resolve private repos.
	UserPrivateRepos []types.RepoName
	Mode             GlobalSearchMode

	// Query is the parsed query from the user. You should be using Pattern
	// instead, but Query is useful for checking extra fields that are set and
	// ignored by Pattern, such as index:no
	Query query.Q

	// UseFullDeadline indicates that the search should try do as much work as
	// it can within context.Deadline. If false the search should try and be
	// as fast as possible, even if a "slow" deadline is set.
	//
	// For example searcher will wait to full its archive cache for a
	// repository if this field is true. Another example is we set this field
	// to true if the user requests a specific timeout or maximum result size.
	UseFullDeadline bool

	Zoekt        *searchbackend.Zoekt
	SearcherURLs *endpoint.Map
}

// TextParametersForCommitParameters is an intermediate type based on
// TextParameters that encodes parameters exclusively for a commit search. The
// commit search internals converts this type to CommitParameters. The
// commitParameter type definitions will be merged in future.
type TextParametersForCommitParameters struct {
	PatternInfo *CommitPatternInfo
	Repos       []*RepositoryRevisions
	Query       query.Q
}

// TextPatternInfo is the struct used by vscode pass on search queries. Keep it in
// sync with pkg/searcher/protocol.PatternInfo.
type TextPatternInfo struct {
	Pattern         string
	IsNegated       bool
	IsRegExp        bool
	IsStructuralPat bool
	CombyRule       string
	IsWordMatch     bool
	IsCaseSensitive bool
	FileMatchLimit  int32
	Index           query.YesNoOnly
	Select          filter.SelectPath

	// We do not support IsMultiline
	// IsMultiline     bool
	IncludePatterns []string
	ExcludePattern  string

	FilePatternsReposMustInclude []string
	FilePatternsReposMustExclude []string

	PathPatternsAreCaseSensitive bool

	PatternMatchesContent bool
	PatternMatchesPath    bool

	Languages []string
}

func (p *TextPatternInfo) String() string {
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

	for _, inc := range p.FilePatternsReposMustInclude {
		args = append(args, fmt.Sprintf("repositoryPathPattern:%s", inc))
	}
	for _, dec := range p.FilePatternsReposMustExclude {
		args = append(args, fmt.Sprintf("-repositoryPathPattern:%s", dec))
	}

	path := "f"
	if p.PathPatternsAreCaseSensitive {
		path = "F"
	}
	if p.ExcludePattern != "" {
		args = append(args, fmt.Sprintf("-%s:%q", path, p.ExcludePattern))
	}
	for _, inc := range p.IncludePatterns {
		args = append(args, fmt.Sprintf("%s:%q", path, inc))
	}

	return fmt.Sprintf("TextPatternInfo{%s}", strings.Join(args, ","))
}

type RepoOptions struct {
	RepoFilters        []string
	MinusRepoFilters   []string
	RepoGroupFilters   []string
	SearchContextSpec  string
	VersionContextName string
	UserSettings       *schema.Settings
	NoForks            bool
	OnlyForks          bool
	NoArchived         bool
	OnlyArchived       bool
	CommitAfter        string
	OnlyPrivate        bool
	OnlyPublic         bool
	Ranked             bool // Return results ordered by rank
	Limit              int
	CacheLookup        bool
	Query              query.Q
}

func (op *RepoOptions) String() string {
	var b strings.Builder
	if len(op.RepoFilters) == 0 {
		b.WriteString("r=[]")
	}
	for i, r := range op.RepoFilters {
		if i != 0 {
			b.WriteByte(' ')
		}
		b.WriteString(strconv.Quote(r))
	}

	if len(op.MinusRepoFilters) > 0 {
		_, _ = fmt.Fprintf(&b, " -r=%v", op.MinusRepoFilters)
	}
	if len(op.RepoGroupFilters) > 0 {
		_, _ = fmt.Fprintf(&b, " groups=%v", op.RepoGroupFilters)
	}
	if op.VersionContextName != "" {
		_, _ = fmt.Fprintf(&b, " versionContext=%q", op.VersionContextName)
	}
	if op.CommitAfter != "" {
		_, _ = fmt.Fprintf(&b, " CommitAfter=%q", op.CommitAfter)
	}

	if op.NoForks {
		b.WriteString(" NoForks")
	}
	if op.OnlyForks {
		b.WriteString(" OnlyForks")
	}
	if op.NoArchived {
		b.WriteString(" NoArchived")
	}
	if op.OnlyArchived {
		b.WriteString(" OnlyArchived")
	}
	if op.OnlyPrivate {
		b.WriteString(" OnlyPrivate")
	}
	if op.OnlyPublic {
		b.WriteString(" OnlyPublic")
	}

	return b.String()
}
