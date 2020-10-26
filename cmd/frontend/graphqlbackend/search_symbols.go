package graphqlbackend

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"

	"github.com/inconshreveable/log15"
	"github.com/neelance/parallel"
	"github.com/opentracing/opentracing-go/ext"
	otlog "github.com/opentracing/opentracing-go/log"
	"github.com/pkg/errors"
	"github.com/sourcegraph/go-lsp"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/errcode"
	"github.com/sourcegraph/sourcegraph/internal/gituri"
	"github.com/sourcegraph/sourcegraph/internal/goroutine"
	"github.com/sourcegraph/sourcegraph/internal/search"
	"github.com/sourcegraph/sourcegraph/internal/symbols/protocol"
	"github.com/sourcegraph/sourcegraph/internal/trace"
	"github.com/sourcegraph/sourcegraph/internal/trace/ot"
	"github.com/sourcegraph/sourcegraph/internal/vcs/git"
)

// searchSymbolResult is a result from symbol search.
type searchSymbolResult struct {
	symbol  protocol.Symbol
	baseURI *gituri.URI
	lang    string
	commit  *GitCommitResolver // TODO: change to utility type we create to remove git resolvers from search.
}

func (s *searchSymbolResult) uri() *gituri.URI {
	return s.baseURI.WithFilePath(s.symbol.Path)
}

var mockSearchSymbols func(ctx context.Context, args *search.TextParameters, limit int) (res []*FileMatchResolver, common *searchResultsCommon, err error)

// searchSymbols searches the given repos in parallel for symbols matching the given search query
// it can be used for both search suggestions and search results
//
// May return partial results and an error
func searchSymbols(ctx context.Context, args *search.TextParameters, limit int) (res []*FileMatchResolver, common *searchResultsCommon, err error) {
	if mockSearchSymbols != nil {
		return mockSearchSymbols(ctx, args, limit)
	}

	repos, err := getRepos(ctx, args.RepoPromise)
	if err != nil {
		return nil, nil, err
	}

	tr, ctx := trace.New(ctx, "Search symbols", fmt.Sprintf("query: %+v, numRepoRevs: %d", args.PatternInfo, len(repos)))
	defer func() {
		tr.SetError(err)
		tr.Finish()
	}()

	if args.PatternInfo.Pattern == "" {
		return nil, nil, nil
	}

	ctx, cancelAll := context.WithCancel(ctx)
	defer cancelAll()

	common = &searchResultsCommon{partial: make(map[api.RepoName]struct{})}

	indexed, err := newIndexedSearchRequest(ctx, args, symbolRequest)
	if err != nil {
		return nil, nil, err
	}

	common.repos = make([]*types.Repo, len(repos))
	for i, repo := range repos {
		common.repos[i] = repo.Repo
	}

	var searcherRepos []*search.RepositoryRevisions
	if indexed.DisableUnindexedSearch {
		tr.LazyPrintf("disabling unindexed search")
		common.missing = make([]*types.Repo, len(indexed.Unindexed))
		for i, r := range indexed.Unindexed {
			common.missing[i] = r.Repo
		}
	} else {
		// Limit the number of unindexed repositories searched for a single
		// query. Searching more than this will merely flood the system and
		// network with requests that will timeout.
		searcherRepos, common.missing = limitSearcherRepos(indexed.Unindexed, maxUnindexedRepoRevSearchesPerQuery)
		if len(common.missing) > 0 {
			tr.LazyPrintf("limiting unindexed repos searched to %d", maxUnindexedRepoRevSearchesPerQuery)
		}
	}

	var (
		run = parallel.NewRun(conf.SearchSymbolsParallelism())
		mu  sync.Mutex

		unflattened       [][]*FileMatchResolver
		flattenedSize     int
		overLimitCanceled bool
	)

	addMatches := func(matches []*FileMatchResolver) {
		if len(matches) > 0 {
			common.resultCount += int32(len(matches))
			sort.Slice(matches, func(i, j int) bool {
				a, b := matches[i].uri, matches[j].uri
				return a > b
			})
			unflattened = append(unflattened, matches)
			flattenedSize += len(matches)

			if flattenedSize > int(args.PatternInfo.FileMatchLimit) {
				tr.LazyPrintf("cancel due to result size: %d > %d", flattenedSize, args.PatternInfo.FileMatchLimit)
				overLimitCanceled = true
				common.limitHit = true
				cancelAll()
			}
		}
	}

	run.Acquire()
	goroutine.Go(func() {
		defer run.Release()
		matches, limitHit, reposLimitHit, searchErr := indexed.Search(ctx)
		mu.Lock()
		defer mu.Unlock()
		if ctx.Err() == nil {
			for _, repo := range indexed.Repos() {
				common.searched = append(common.searched, repo.Repo)
				common.indexed = append(common.indexed, repo.Repo)
			}
			for repo := range reposLimitHit {
				common.partial[api.RepoName(repo)] = struct{}{}
			}
		}
		if limitHit {
			common.limitHit = true
		}
		tr.LogFields(otlog.Object("searchErr", searchErr), otlog.Error(err), otlog.Bool("overLimitCanceled", overLimitCanceled))
		if searchErr != nil && err == nil && !overLimitCanceled {
			err = searchErr
			tr.LazyPrintf("cancel indexed symbol search due to error: %v", err)
		}
		addMatches(matches)
	})

	for _, repoRevs := range searcherRepos {
		repoRevs := repoRevs
		if ctx.Err() != nil {
			break
		}
		if len(repoRevs.RevSpecs()) == 0 {
			continue
		}
		run.Acquire()
		goroutine.Go(func() {
			defer run.Release()
			repoSymbols, repoErr := searchSymbolsInRepo(ctx, repoRevs, args.PatternInfo, limit)
			if repoErr != nil {
				tr.LogFields(otlog.String("repo", string(repoRevs.Repo.Name)), otlog.String("repoErr", repoErr.Error()), otlog.Bool("timeout", errcode.IsTimeout(repoErr)), otlog.Bool("temporary", errcode.IsTemporary(repoErr)))
			}
			mu.Lock()
			defer mu.Unlock()
			limitHit := symbolCount(res) > limit
			repoErr = handleRepoSearchResult(common, repoRevs, limitHit, false, repoErr)
			if repoErr != nil {
				if ctx.Err() == nil || errors.Cause(repoErr) != ctx.Err() {
					// Only record error if it's not directly caused by a context error.
					run.Error(repoErr)
				}
			} else {
				common.searched = append(common.searched, repoRevs.Repo)
			}
			if repoSymbols != nil {
				addMatches(repoSymbols)
			}
		})
	}
	err = run.Wait()
	flattened := flattenFileMatches(unflattened, int(args.PatternInfo.FileMatchLimit))
	res2 := limitSymbolResults(flattened, limit)
	common.limitHit = symbolCount(res2) < symbolCount(res)
	return res2, common, err
}

// limitSymbolResults returns a new version of res containing no more than limit symbol matches.
func limitSymbolResults(res []*FileMatchResolver, limit int) []*FileMatchResolver {
	res2 := make([]*FileMatchResolver, 0, len(res))
	nsym := 0
	for _, r := range res {
		r2 := *r
		if nsym+len(r.symbols) > limit {
			r2.symbols = r2.symbols[:limit-nsym]
		}
		if len(r2.symbols) > 0 {
			res2 = append(res2, &r2)
		}
		nsym += len(r2.symbols)
		if nsym >= limit {
			return res2
		}
	}
	return res2
}

// symbolCount returns the total number of symbols in a slice of fileMatchResolvers.
func symbolCount(fmrs []*FileMatchResolver) int {
	nsym := 0
	for _, fmr := range fmrs {
		nsym += len(fmr.symbols)
	}
	return nsym
}

func searchSymbolsInRepo(ctx context.Context, repoRevs *search.RepositoryRevisions, patternInfo *search.TextPatternInfo, limit int) (res []*FileMatchResolver, err error) {
	span, ctx := ot.StartSpanFromContext(ctx, "Search symbols in repo")
	defer func() {
		if err != nil {
			ext.Error.Set(span, true)
			span.LogFields(otlog.Error(err))
		}
		span.Finish()
	}()
	span.SetTag("repo", string(repoRevs.Repo.Name))

	inputRev := repoRevs.RevSpecs()[0]
	span.SetTag("rev", inputRev)
	// Do not trigger a repo-updater lookup (e.g.,
	// backend.{GitRepo,Repos.ResolveRev}) because that would slow this operation
	// down by a lot (if we're looping over many repos). This means that it'll fail if a
	// repo is not on gitserver.
	commitID, err := git.ResolveRevision(ctx, repoRevs.GitserverRepo(), nil, inputRev, git.ResolveRevisionOptions{})
	if err != nil {
		return nil, err
	}
	span.SetTag("commit", string(commitID))
	baseURI, err := gituri.Parse("git://" + string(repoRevs.Repo.Name) + "?" + url.QueryEscape(inputRev))
	if err != nil {
		return nil, err
	}

	repoResolver := NewRepositoryResolver(repoRevs.Repo)
	commitResolver := &GitCommitResolver{
		repoResolver: repoResolver,
		oid:          GitObjectID(commitID),
		inputRev:     &inputRev,
		// NOTE: Not all fields are set, for performance.
	}

	symbols, err := backend.Symbols.ListTags(ctx, search.SymbolsParameters{
		Repo:            repoRevs.Repo.Name,
		CommitID:        commitID,
		Query:           patternInfo.Pattern,
		IsCaseSensitive: patternInfo.IsCaseSensitive,
		IsRegExp:        patternInfo.IsRegExp,
		IncludePatterns: patternInfo.IncludePatterns,
		ExcludePattern:  patternInfo.ExcludePattern,
		// Ask for limit + 1 so we can detect whether there are more results than the limit.
		First: limit + 1,
	})
	fileMatchesByURI := make(map[string]*FileMatchResolver)
	fileMatches := make([]*FileMatchResolver, 0)

	for _, symbol := range symbols {
		symbolRes := &searchSymbolResult{
			symbol:  symbol,
			baseURI: baseURI,
			lang:    strings.ToLower(symbol.Language),
			commit:  commitResolver,
		}
		uri := makeFileMatchURIFromSymbol(symbolRes, inputRev)
		if fileMatch, ok := fileMatchesByURI[uri]; ok {
			fileMatch.symbols = append(fileMatch.symbols, symbolRes)
		} else {
			fileMatch := &FileMatchResolver{
				JPath:   symbolRes.symbol.Path,
				symbols: []*searchSymbolResult{symbolRes},
				uri:     uri,
				Repo:    repoResolver,
				// Don't get commit from GitCommitResolver.OID() because we don't want to
				// slow search results down when they are coming from zoekt.
				CommitID: api.CommitID(symbolRes.commit.oid),
			}
			fileMatchesByURI[uri] = fileMatch
			fileMatches = append(fileMatches, fileMatch)
		}
	}
	return fileMatches, err
}

// makeFileMatchURIFromSymbol makes a git://repo?rev#path URI from a symbol
// search result to use in a fileMatchResolver
func makeFileMatchURIFromSymbol(symbolResult *searchSymbolResult, inputRev string) string {
	uri := "git:/" + string(symbolResult.commit.repoResolver.URL())
	if inputRev != "" {
		uri += "?" + inputRev
	}
	uri += "#" + symbolResult.uri().Fragment
	return uri
}

func symbolRange(s protocol.Symbol) lsp.Range {
	ch := ctagsSymbolCharacter(s)
	return lsp.Range{
		Start: lsp.Position{Line: s.Line - 1, Character: ch},
		End:   lsp.Position{Line: s.Line - 1, Character: ch + len(s.Name)},
	}
}

// ctagsSymbolCharacter only outputs the line number, not the character (or range). Use the regexp it provides to
// guess the character.
func ctagsSymbolCharacter(s protocol.Symbol) int {
	if s.Pattern == "" {
		return 0
	}
	pattern := strings.TrimPrefix(s.Pattern, "/^")
	i := strings.Index(pattern, s.Name)
	if i >= 0 {
		return i
	}
	return 0
}

func ctagsKindToLSPSymbolKind(kind string) lsp.SymbolKind {
	// Ctags kinds are determined by the parser and do not (in general) match LSP symbol kinds.
	switch strings.ToLower(kind) {
	case "file":
		return lsp.SKFile
	case "module":
		return lsp.SKModule
	case "namespace":
		return lsp.SKNamespace
	case "package", "packagename", "subprogspec":
		return lsp.SKPackage
	case "class", "type", "service", "typedef", "union", "section", "subtype", "component":
		return lsp.SKClass
	case "method", "methodspec":
		return lsp.SKMethod
	case "property":
		return lsp.SKProperty
	case "field", "member", "anonmember", "recordfield":
		return lsp.SKField
	case "constructor":
		return lsp.SKConstructor
	case "enum", "enumerator":
		return lsp.SKEnum
	case "interface":
		return lsp.SKInterface
	case "function", "func", "subroutine", "macro", "subprogram", "procedure", "command", "singletonmethod":
		return lsp.SKFunction
	case "variable", "var", "functionvar", "define", "alias", "val":
		return lsp.SKVariable
	case "constant", "const":
		return lsp.SKConstant
	case "string", "message", "heredoc":
		return lsp.SKString
	case "number":
		return lsp.SKNumber
	case "bool", "boolean":
		return lsp.SKBoolean
	case "array":
		return lsp.SKArray
	case "object", "literal", "map":
		return lsp.SKObject
	case "key", "label", "target", "selector", "id", "tag":
		return lsp.SKKey
	case "null":
		return lsp.SKNull
	case "enum member", "enumconstant":
		return lsp.SKEnumMember
	case "struct":
		return lsp.SKStruct
	case "event":
		return lsp.SKEvent
	case "operator":
		return lsp.SKOperator
	case "type parameter", "annotation":
		return lsp.SKTypeParameter
	}
	log15.Debug("Unknown ctags kind", "kind", kind)
	return 0
}
