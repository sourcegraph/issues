package search

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/endpoint"
	"github.com/sourcegraph/sourcegraph/internal/gitserver"
	searchbackend "github.com/sourcegraph/sourcegraph/internal/search/backend"
	"github.com/sourcegraph/sourcegraph/internal/search/query"
	"github.com/sourcegraph/sourcegraph/internal/vcs/git"
)

type TypeParameters interface {
	typeParametersValue()
}

func (c CommitParameters) typeParametersValue()  {}
func (d DiffParameters) typeParametersValue()    {}
func (s SymbolsParameters) typeParametersValue() {}
func (t TextParameters) typeParametersValue()    {}

type CommitParameters struct {
	RepoRevs           *RepositoryRevisions
	PatternInfo        *CommitPatternInfo
	Query              query.QueryInfo
	Diff               bool
	ExtraMessageValues []string
}

type DiffParameters struct {
	Repo    gitserver.Repo
	Options git.RawLogDiffSearchOptions
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

type GlobalSearchMode int

const (
	ZoektGlobalSearch GlobalSearchMode = iota + 1
	SearcherOnly
	NoFilePath
)

type Promise struct {
	getOnce sync.Once
	err     error

	initOnce sync.Once
	done     chan struct{}

	valueOnce sync.Once
	value     interface{}
}

func (p *Promise) init() {
	p.initOnce.Do(func() { p.done = make(chan struct{}) })
}

// Resolve returns a promise that is resolved with a given value.
func (p *Promise) Resolve(v interface{}) *Promise {
	p.valueOnce.Do(func() {
		p.init()
		p.value = v
		close(p.done)
	})
	return p
}

// Get returns the value. It blocks until the promise resolves or the context is
// canceled. Further calls to Get will always return the original results, IE err
// will stay nil even if the context expired between the first and the second
// call. Vice versa, if ctx finishes while resolving, then we will always return
// ctx.Err()
func (p *Promise) Get(ctx context.Context) (interface{}, error) {
	p.getOnce.Do(func() {
		p.init()
		select {
		case <-ctx.Done():
			p.err = ctx.Err()
		case <-p.done:
		}
	})
	return p.value, p.err
}

// TextParameters are the parameters passed to a search backend. It contains the Pattern
// to search for, as well as the hydrated list of repository revisions to
// search. It defines behavior for text search on repository names, file names, and file content.
type TextParameters struct {
	PatternInfo *TextPatternInfo

	// Performance optimization: For global queries, resolving repositories and
	// querying zoekt happens concurrently.
	RepoPromise *Promise
	Mode        GlobalSearchMode

	// Query is the parsed query from the user. You should be using Pattern
	// instead, but Query is useful for checking extra fields that are set and
	// ignored by Pattern, such as index:no
	Query query.QueryInfo

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
	Query       query.QueryInfo
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
