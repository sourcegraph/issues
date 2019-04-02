package graphqlbackend

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/neelance/parallel"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	otlog "github.com/opentracing/opentracing-go/log"
	"github.com/pkg/errors"
	lsp "github.com/sourcegraph/go-lsp"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/internal/goroutine"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/internal/pkg/search"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/internal/pkg/search/query"
	"github.com/sourcegraph/sourcegraph/pkg/api"
	"github.com/sourcegraph/sourcegraph/pkg/errcode"
	"github.com/sourcegraph/sourcegraph/pkg/symbols"
	"github.com/sourcegraph/sourcegraph/pkg/symbols/protocol"
	"github.com/sourcegraph/sourcegraph/pkg/trace"
	"github.com/sourcegraph/sourcegraph/pkg/vcs/git"
	log15 "gopkg.in/inconshreveable/log15.v2"
)

var mockSearchSymbols func(ctx context.Context, args *search.Args, limit int32) (res []*searchResultResolver, common *searchResultsCommon, err error)

// searchSymbols searches the given repos in parallel for symbols matching the given search query
// it can be used for both search suggestions and search results
//
// May return partial results and an error
func searchSymbols(ctx context.Context, args *search.Args, limit int32) (res []*searchResultResolver, common *searchResultsCommon, err error) {
	if mockSearchSymbols != nil {
		return mockSearchSymbols(ctx, args, limit)
	}

	tr, ctx := trace.New(ctx, "Search symbols", fmt.Sprintf("query: %+v, numRepoRevs: %d", args.Pattern, len(args.Repos)))
	defer func() {
		tr.SetError(err)
		tr.Finish()
	}()

	if args.Pattern.Pattern == "" {
		return nil, nil, nil
	}

	ctx, cancelAll := context.WithCancel(ctx)
	defer cancelAll()

	common = &searchResultsCommon{}
	var (
		run = parallel.NewRun(20)
		mu  sync.Mutex
	)
	for _, repoRevs := range args.Repos {
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
			repoSymbols, repoErr := searchSymbolsInRepo(ctx, repoRevs, args.Pattern, args.Query, limit)
			if repoErr != nil {
				tr.LogFields(otlog.String("repo", string(repoRevs.Repo.Name)), otlog.String("repoErr", repoErr.Error()), otlog.Bool("timeout", errcode.IsTimeout(repoErr)), otlog.Bool("temporary", errcode.IsTemporary(repoErr)))
			}
			mu.Lock()
			defer mu.Unlock()
			limitHit := int32(len(res)) > limit
			repoErr = handleRepoSearchResult(common, *repoRevs, limitHit, false, repoErr)
			if repoErr != nil {
				if ctx.Err() == nil || errors.Cause(repoErr) != ctx.Err() {
					// Only record error if it's not directly caused by a context error.
					run.Error(repoErr)
				}
			} else {
				common.searched = append(common.searched, repoRevs.Repo)
			}
			if repoSymbols != nil {
				for _, fileMatch := range repoSymbols {
					res = append(res, &searchResultResolver{fileMatch: fileMatch})
				}
				if limitHit {
					cancelAll()
				}
			}
		})
	}
	err = run.Wait()

	if int32(len(res)) > limit {
		common.limitHit = true
		res = res[:limit]
	}
	return res, common, err
}

func searchSymbolsInRepo(ctx context.Context, repoRevs *search.RepositoryRevisions, patternInfo *search.PatternInfo, query *query.Query, limit int32) (res []*fileMatchResolver, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "Search symbols in repo")
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
	commitID, err := git.ResolveRevision(ctx, repoRevs.GitserverRepo(), nil, inputRev, nil)
	if err != nil {
		return nil, err
	}
	span.SetTag("commit", string(commitID))

	symbols, err := symbols.ComputeSymbols(ctx, symbols.ComputeSymbolsArgs{
		CommitID:        commitID,
		RepoName:        repoRevs.Repo.Name,
		Query:           &patternInfo.Pattern,
		IncludePatterns: &patternInfo.IncludePatterns,
		ExcludePattern:  &patternInfo.ExcludePattern,
		IsCaseSensitive: &patternInfo.IsCaseSensitive,
		IsRegExp:        &patternInfo.IsRegExp,
		First:           &limit,
	})
	if err != nil {
		return nil, err
	}

	fileMatchesByURI := make(map[string]*fileMatchResolver)
	fileMatches := make([]*fileMatchResolver, 0)
	for _, symbol := range symbols {
		commit := &gitCommitResolver{
			repo:     &repositoryResolver{repo: repoRevs.Repo},
			oid:      gitObjectID(commitID),
			inputRev: &inputRev,
			// NOTE: Not all fields are set, for performance.
		}
		if inputRev != "" {
			commit.inputRev = &inputRev
		}
		symbolRes := toSymbolResolver(symbol, commit)
		uri := makeFileMatchURIFromSymbol(symbolRes, inputRev)
		if fileMatch, ok := fileMatchesByURI[uri]; ok {
			fileMatch.symbols = append(fileMatch.symbols, symbolRes)
		} else {
			fileMatch := &fileMatchResolver{
				symbols: []*symbolResolver{symbolRes},
				uri:     uri,
				repo:    symbolRes.location.resource.commit.repo.repo,
				// Don't get commit from gitCommitResolver.OID() because we don't want to
				// slow search results down when they are coming from zoekt.
				commitID: api.CommitID(symbolRes.location.resource.commit.oid),
			}
			fileMatchesByURI[uri] = fileMatch
			fileMatches = append(fileMatches, fileMatch)
		}
	}
	return fileMatches, err
}

// makeFileMatchURIFromSymbol makes a git://repo?rev#path URI from a symbolResolver to use in a fileMatchResolver
func makeFileMatchURIFromSymbol(symbolResolver *symbolResolver, inputRev string) string {
	uri := "git:/" + string(symbolResolver.location.resource.commit.repo.URL())
	if inputRev != "" {
		uri += "?" + inputRev
	}
	uri += "#" + symbolResolver.location.resource.path
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
	switch kind {
	case "file":
		return lsp.SKFile
	case "module":
		return lsp.SKModule
	case "namespace":
		return lsp.SKNamespace
	case "package", "packageName", "subprogspec":
		return lsp.SKPackage
	case "class", "type", "service", "typedef", "union", "section", "subtype", "component":
		return lsp.SKClass
	case "method", "methodSpec":
		return lsp.SKMethod
	case "property":
		return lsp.SKProperty
	case "field", "member", "anonMember":
		return lsp.SKField
	case "constructor":
		return lsp.SKConstructor
	case "enum", "enumerator":
		return lsp.SKEnum
	case "interface":
		return lsp.SKInterface
	case "function", "func", "subroutine", "macro", "subprogram", "procedure", "command", "singletonMethod":
		return lsp.SKFunction
	case "variable", "var", "functionVar", "define", "alias":
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
	case "enum member", "enumConstant":
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
