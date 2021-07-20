package zoekt

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/cockroachdb/errors"
	"github.com/google/zoekt"
	zoektquery "github.com/google/zoekt/query"
	"github.com/inconshreveable/log15"
	"github.com/opentracing/opentracing-go/log"
	"go.uber.org/atomic"
	"golang.org/x/sync/errgroup"

	"github.com/sourcegraph/sourcegraph/internal/actor"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/search"
	"github.com/sourcegraph/sourcegraph/internal/search/backend"
	"github.com/sourcegraph/sourcegraph/internal/search/filter"
	"github.com/sourcegraph/sourcegraph/internal/search/query"
	"github.com/sourcegraph/sourcegraph/internal/search/result"
	"github.com/sourcegraph/sourcegraph/internal/search/streaming"
	"github.com/sourcegraph/sourcegraph/internal/trace"
	"github.com/sourcegraph/sourcegraph/internal/types"
)

type IndexedRequestType string

const (
	TextRequest   IndexedRequestType = "text"
	SymbolRequest IndexedRequestType = "symbol"
)

// indexedRepoRevs creates both the Sourcegraph and Zoekt representation of a
// list of repository and refs to search.
type IndexedRepoRevs struct {
	// repoRevs is the Sourcegraph representation of a the list of repoRevs
	// repository and revisions to search.
	repoRevs map[string]*search.RepositoryRevisions

	// repoBranches will be used when we query zoekt. The order of branches
	// must match that in a reporev such that we can map back results. IE this
	// invariant is maintained:
	//
	//  repoBranches[reporev.Repo.Name][i] <-> reporev.Revs[i]
	repoBranches map[string][]string
}

// headBranch is used as a singleton of the indexedRepoRevs.repoBranches to save
// common-case allocations within indexedRepoRevs.Add.
var headBranch = []string{"HEAD"}

// add will add reporev and repo to the list of repository and branches to
// search if reporev's refs are a subset of repo's branches. It will return
// the revision specifiers it can't add.
func (rb *IndexedRepoRevs) add(reporev *search.RepositoryRevisions, repo *zoekt.Repository) []search.RevisionSpecifier {
	// A repo should only appear once in revs. However, in case this
	// invariant is broken we will treat later revs as if it isn't
	// indexed.
	if _, ok := rb.repoBranches[string(reporev.Repo.Name)]; ok {
		return reporev.Revs
	}

	if !reporev.OnlyExplicit() {
		// Contains a RefGlob or ExcludeRefGlob so we can't do indexed
		// search on it.
		//
		// TODO we could only process the explicit revs and return the non
		// explicit ones as unindexed.
		return reporev.Revs
	}

	if len(reporev.Revs) == 1 && repo.Branches[0].Name == "HEAD" && (reporev.Revs[0].RevSpec == "" || reporev.Revs[0].RevSpec == "HEAD") {
		rb.repoRevs[string(reporev.Repo.Name)] = reporev
		rb.repoBranches[string(reporev.Repo.Name)] = headBranch
		return nil
	}

	// Assume for large searches they will mostly involve indexed
	// revisions, so just allocate that.
	var unindexed []search.RevisionSpecifier
	indexed := make([]search.RevisionSpecifier, 0, len(reporev.Revs))

	branches := make([]string, 0, len(reporev.Revs))
	for _, rev := range reporev.Revs {
		if rev.RevSpec == "" || rev.RevSpec == "HEAD" {
			// Zoekt convention that first branch is HEAD
			branches = append(branches, repo.Branches[0].Name)
			indexed = append(indexed, rev)
			continue
		}

		found := false
		for _, branch := range repo.Branches {
			if branch.Name == rev.RevSpec {
				branches = append(branches, branch.Name)
				found = true
				break
			}
			// Check if rev is an abbrev commit SHA
			if len(rev.RevSpec) >= 4 && strings.HasPrefix(branch.Version, rev.RevSpec) {
				branches = append(branches, branch.Name)
				found = true
				break
			}
		}

		if found {
			indexed = append(indexed, rev)
		} else {
			unindexed = append(unindexed, rev)
		}
	}

	// We found indexed branches! Track them.
	if len(indexed) > 0 {
		rb.repoRevs[string(reporev.Repo.Name)] = reporev
		rb.repoBranches[string(reporev.Repo.Name)] = branches
	}

	return unindexed
}

// getRepoInputRev returns the repo and inputRev associated with file.
func (rb *IndexedRepoRevs) getRepoInputRev(file *zoekt.FileMatch) (repo types.RepoName, inputRevs []string) {
	repoRev := rb.repoRevs[file.Repository]

	inputRevs = make([]string, 0, len(file.Branches))
	for _, branch := range file.Branches {
		for i, b := range rb.repoBranches[file.Repository] {
			if branch == b {
				// RevSpec is guaranteed to be explicit via zoektIndexedRepos
				inputRevs = append(inputRevs, repoRev.Revs[i].RevSpec)
			}
		}
	}

	if len(inputRevs) == 0 {
		// Did not find a match. This is unexpected, but we can fallback to
		// file.Version to generate correct links.
		inputRevs = append(inputRevs, file.Version)
	}

	return repoRev.Repo, inputRevs
}

// IndexedSearchRequest is responsible for translating a Sourcegraph search
// query into a Zoekt query and mapping the results from zoekt back to
// Sourcegraph result types.
type IndexedSearchRequest struct {
	// Unindexed is a slice of repository revisions that can't be searched by
	// Zoekt. The repository revisions should be searched by the searcher
	// service.
	//
	// If IndexUnavailable is true or the query specifies index:no then all
	// repository revisions will be listed. Otherwise it will just be
	// repository revisions not indexed.
	Unindexed []*search.RepositoryRevisions

	// IndexUnavailable is true if zoekt is offline or disabled.
	IndexUnavailable bool

	// DisableUnindexedSearch is true if the query specified that only index
	// search should be used.
	DisableUnindexedSearch bool

	// inputs
	Args *search.TextParameters
	Typ  IndexedRequestType

	// RepoRevs is the repository revisions that are indexed and will be
	// searched.
	RepoRevs *IndexedRepoRevs

	// since if non-nil will be used instead of time.Since. For tests
	since func(time.Time) time.Duration
}

// Repos is a map of repository revisions that are indexed and will be
// searched by Zoekt. Do not mutate.
func (s *IndexedSearchRequest) Repos() map[string]*search.RepositoryRevisions {
	if s.RepoRevs == nil {
		return nil
	}
	return s.RepoRevs.repoRevs
}

// Search streams 0 or more events to c.
func (s *IndexedSearchRequest) Search(ctx context.Context, c streaming.Sender) error {
	if s.Args == nil {
		return nil
	}
	if len(s.Repos()) == 0 && s.Args.Mode != search.ZoektGlobalSearch {
		return nil
	}

	since := time.Since
	if s.since != nil {
		since = s.since
	}

	return zoektSearch(ctx, s.Args, s.RepoRevs, s.Typ, since, c)
}

const maxUnindexedRepoRevSearchesPerQuery = 200

func NewIndexedSearchRequest(ctx context.Context, args *search.TextParameters, typ IndexedRequestType, stream streaming.Sender) (_ *IndexedSearchRequest, err error) {
	tr, ctx := trace.New(ctx, "newIndexedSearchRequest", string(typ))
	tr.LogFields(trace.Stringer("global_search_mode", args.Mode))
	defer func() {
		tr.SetError(err)
		tr.Finish()
	}()
	repos, err := args.RepoPromise.Get(ctx)
	if err != nil {
		return nil, err
	}

	// If Zoekt is disabled just fallback to Unindexed.
	if !args.Zoekt.Enabled() {
		if args.PatternInfo.Index == query.Only {
			return nil, errors.Errorf("invalid index:%q (indexed search is not enabled)", args.PatternInfo.Index)
		}

		return &IndexedSearchRequest{
			Unindexed:        limitUnindexedRepos(repos, maxUnindexedRepoRevSearchesPerQuery, stream),
			IndexUnavailable: true,
		}, nil
	}

	// Fallback to Unindexed if the query contains ref-globs
	if query.ContainsRefGlobs(args.Query) {
		if args.PatternInfo.Index == query.Only {
			return nil, errors.Errorf("invalid index:%q (revsions with glob pattern cannot be resolved for indexed searches)", args.PatternInfo.Index)
		}
		return &IndexedSearchRequest{
			Unindexed: limitUnindexedRepos(repos, maxUnindexedRepoRevSearchesPerQuery, stream),
		}, nil
	}

	// Fallback to Unindexed if index:no
	if args.PatternInfo.Index == query.No {
		return &IndexedSearchRequest{
			Unindexed: limitUnindexedRepos(repos, maxUnindexedRepoRevSearchesPerQuery, stream),
		}, nil
	}

	// Only include indexes with symbol information if a symbol request.
	var filter func(repo *zoekt.Repository) bool
	if typ == SymbolRequest {
		filter = func(repo *zoekt.Repository) bool {
			return repo.HasSymbols
		}
	}

	// Consult Zoekt to find out which repository revisions can be searched.
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	indexedSet, err := args.Zoekt.ListAll(ctx)
	if err != nil {
		if ctx.Err() == nil {
			// Only hard fail if the user specified index:only
			if args.PatternInfo.Index == query.Only {
				return nil, errors.New("index:only failed since indexed search is not available yet")
			}

			log15.Warn("zoektIndexedRepos failed", "error", err)
		}

		return &IndexedSearchRequest{
			Unindexed:        limitUnindexedRepos(repos, maxUnindexedRepoRevSearchesPerQuery, stream),
			IndexUnavailable: true,
		}, ctx.Err()
	}

	tr.LogFields(log.Int("all_indexed_set.size", len(indexedSet)))

	// Split based on indexed vs unindexed
	indexed, searcherRepos := zoektIndexedRepos(indexedSet, repos, filter)

	tr.LogFields(
		log.Int("indexed.size", len(indexed.repoRevs)),
		log.Int("searcher_repos.size", len(searcherRepos)),
	)

	// Disable unindexed search
	if args.PatternInfo.Index == query.Only {
		searcherRepos = limitUnindexedRepos(searcherRepos, 0, stream)
	}

	return &IndexedSearchRequest{
		Args: args,
		Typ:  typ,

		Unindexed: limitUnindexedRepos(searcherRepos, maxUnindexedRepoRevSearchesPerQuery, stream),
		RepoRevs:  indexed,

		DisableUnindexedSearch: args.PatternInfo.Index == query.Only,
	}, nil
}

// zoektSearchGlobal searches the entire universe of indexed repositories.
//
// We only have to search "HEAD", because global queries, per definition, don't
// have a repo: filter and consequently no rev: filter. This makes the code a bit
// simpler because we don't have to resolve revisions before sending off (global)
// requests to Zoekt.
//
// We send 2 queries to Zoekt. One query for public repos and one query for
// private repos.
func zoektSearchGlobal(ctx context.Context, args *search.TextParameters, typ IndexedRequestType, c streaming.Sender) error {
	if args == nil {
		return nil
	}

	if args.Mode != search.ZoektGlobalSearch {
		return fmt.Errorf("zoektSearchGlobal called with args.Mode %d instead of %d", args.Mode, search.ZoektGlobalSearch)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	queryExceptRepos, err := queryToZoektQuery(args.PatternInfo, typ)
	if err != nil {
		return err
	}

	var queries []zoektquery.Q

	// Public
	if !args.RepoOptions.OnlyPrivate {
		rc := zoektquery.RcOnlyPublic
		apply := func(f zoektquery.RawConfig, b bool) {
			if !b {
				return
			}
			rc |= f
		}
		apply(zoektquery.RcOnlyArchived, args.RepoOptions.OnlyArchived)
		apply(zoektquery.RcNoArchived, args.RepoOptions.NoArchived)
		apply(zoektquery.RcOnlyForks, args.RepoOptions.OnlyForks)
		apply(zoektquery.RcNoForks, args.RepoOptions.NoForks)

		queries = append(queries, zoektquery.NewAnd(&zoektquery.Branch{Pattern: "HEAD", Exact: true}, rc, queryExceptRepos))
	}

	// Private
	if !args.RepoOptions.OnlyPublic && len(args.UserPrivateRepos) > 0 {
		privateRepoSet := make(map[string][]string, len(args.UserPrivateRepos))
		head := []string{"HEAD"}
		for _, r := range args.UserPrivateRepos {
			privateRepoSet[string(r.Name)] = head
		}
		queries = append(queries, zoektquery.NewAnd(&zoektquery.RepoBranches{Set: privateRepoSet}, queryExceptRepos))
	}

	k := ResultCountFactor(0, args.PatternInfo.FileMatchLimit, true)
	searchOpts := SearchOpts(ctx, k, args.PatternInfo)

	if deadline, ok := ctx.Deadline(); ok {
		// If the user manually specified a timeout, allow zoekt to use all of the remaining timeout.
		searchOpts.MaxWallTime = time.Until(deadline)
		if searchOpts.MaxWallTime < 0 {
			return ctx.Err()
		}
		// We don't want our context's deadline to cut off zoekt so that we can get the results
		// found before the deadline.
		//
		// We'll create a new context that gets cancelled if the other context is cancelled for any
		// reason other than the deadline being exceeded. This essentially means the deadline for the new context
		// will be `deadline + time for zoekt to cancel + network latency`.
		var cancel context.CancelFunc
		ctx, cancel = contextWithoutDeadline(ctx)
		defer cancel()
	}

	g, ctx := errgroup.WithContext(ctx)

	for _, q := range queries {
		q2 := q
		g.Go(func() error {
			// PERF: if we are going to be selecting to repo results only anyways, we can
			// just ask zoekt for only results of type repo.
			if args.PatternInfo.Select.Root() == filter.Repository {
				repoList, err := args.Zoekt.Client.List(ctx, q2, nil)
				if err != nil {
					return err
				}

				matches := make([]result.Match, 0, len(repoList.Repos))
				for _, repo := range repoList.Repos {
					matches = append(matches, &result.RepoMatch{
						Name: api.RepoName(repo.Repository.Name),
						ID:   api.RepoID(repo.Repository.ID),
					})
				}

				c.Send(streaming.SearchEvent{
					Results: matches,
					Stats:   streaming.Stats{}, // TODO
				})
				return nil
			}

			return args.Zoekt.Client.StreamSearch(ctx, q2, &searchOpts, backend.ZoektStreamFunc(func(event *zoekt.SearchResult) {
				sendMatches(event, func(file *zoekt.FileMatch) (types.RepoName, []string, bool) {
					repo := types.RepoName{
						ID:   api.RepoID(file.RepositoryID),
						Name: api.RepoName(file.Repository),
					}
					return repo, []string{""}, true
				}, typ, c)
			}))
		})
	}

	return g.Wait()
}

// zoektSearch searches repositories using zoekt.
func zoektSearch(ctx context.Context, args *search.TextParameters, repos *IndexedRepoRevs, typ IndexedRequestType, since func(t time.Time) time.Duration, c streaming.Sender) error {
	if args == nil {
		return nil
	}

	if args.Mode == search.ZoektGlobalSearch {
		return zoektSearchGlobal(ctx, args, typ, c)
	}

	if len(repos.repoRevs) == 0 {
		return nil
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	queryExceptRepos, err := queryToZoektQuery(args.PatternInfo, typ)
	if err != nil {
		return err
	}

	finalQuery := zoektquery.NewAnd(&zoektquery.RepoBranches{Set: repos.repoBranches}, queryExceptRepos)

	k := ResultCountFactor(len(repos.repoBranches), args.PatternInfo.FileMatchLimit, false)
	searchOpts := SearchOpts(ctx, k, args.PatternInfo)

	// Start event stream.
	t0 := time.Now()

	if deadline, ok := ctx.Deadline(); ok {
		// If the user manually specified a timeout, allow zoekt to use all of the remaining timeout.
		searchOpts.MaxWallTime = time.Until(deadline)
		if searchOpts.MaxWallTime < 0 {
			return ctx.Err()
		}
		// We don't want our context's deadline to cut off zoekt so that we can get the results
		// found before the deadline.
		//
		// We'll create a new context that gets cancelled if the other context is cancelled for any
		// reason other than the deadline being exceeded. This essentially means the deadline for the new context
		// will be `deadline + time for zoekt to cancel + network latency`.
		var cancel context.CancelFunc
		ctx, cancel = contextWithoutDeadline(ctx)
		defer cancel()
	}

	// PERF: if we are going to be selecting to repo results only anyways, we can just ask
	// zoekt for only results of type repo.
	if args.PatternInfo.Select.Root() == filter.Repository {
		return zoektSearchReposOnly(ctx, args.Zoekt.Client, finalQuery, c, func() map[api.RepoID]*search.RepositoryRevisions {
			repoRevMap := make(map[api.RepoID]*search.RepositoryRevisions, len(repos.repoRevs))
			for _, r := range repos.repoRevs {
				repoRevMap[r.Repo.ID] = r
			}
			return repoRevMap
		})
	}

	foundResults := atomic.Bool{}
	err = args.Zoekt.Client.StreamSearch(ctx, finalQuery, &searchOpts, backend.ZoektStreamFunc(func(event *zoekt.SearchResult) {
		foundResults.CAS(false, event.FileCount != 0 || event.MatchCount != 0)
		sendMatches(event, func(file *zoekt.FileMatch) (types.RepoName, []string, bool) {
			repo, inputRevs := repos.getRepoInputRev(file)
			return repo, inputRevs, true
		}, typ, c)
	}))
	if err != nil {
		return err
	}

	mkStatusMap := func(mask search.RepoStatus) search.RepoStatusMap {
		var statusMap search.RepoStatusMap
		for _, r := range repos.repoRevs {
			statusMap.Update(r.Repo.ID, mask)
		}
		return statusMap
	}

	if !foundResults.Load() && since(t0) >= searchOpts.MaxWallTime {
		c.Send(streaming.SearchEvent{Stats: streaming.Stats{Status: mkStatusMap(search.RepoStatusTimedout)}})
	}
	return nil
}

func sendMatches(event *zoekt.SearchResult, getRepoInputRev repoRevFunc, typ IndexedRequestType, c streaming.Sender) {
	files := event.Files
	limitHit := event.FilesSkipped+event.ShardsSkipped > 0

	if len(files) == 0 {
		c.Send(streaming.SearchEvent{
			Stats: streaming.Stats{IsLimitHit: limitHit},
		})
		return
	}

	matches := make([]result.Match, 0, len(files))
	for _, file := range files {
		repo, inputRevs, ok := getRepoInputRev(&file)
		if !ok {
			continue
		}

		var lines []*result.LineMatch
		if typ != SymbolRequest {
			lines = zoektFileMatchToLineMatches(&file)
		}

		for _, inputRev := range inputRevs {
			inputRev := inputRev // copy so we can take the pointer

			var symbols []*result.SymbolMatch
			if typ == SymbolRequest {
				symbols = zoektFileMatchToSymbolResults(repo, inputRev, &file)
			}
			fm := result.FileMatch{
				LineMatches: lines,
				Symbols:     symbols,
				File: result.File{
					InputRev: &inputRev,
					CommitID: api.CommitID(file.Version),
					Repo:     repo,
					Path:     file.FileName,
				},
			}
			matches = append(matches, &fm)
		}
	}

	c.Send(streaming.SearchEvent{
		Results: matches,
		Stats: streaming.Stats{
			IsLimitHit: limitHit,
		},
	})
}

// bufferedSender returns a buffered Sender with capacity cap, and a cleanup
// function which blocks until the buffer is drained. The cleanup function may
// only be called once. For cap=0, bufferedSender returns the input sender.
func bufferedSender(cap int, sender zoekt.Sender) (zoekt.Sender, func()) {
	if cap == 0 {
		return sender, func() {}
	}
	buf := make(chan *zoekt.SearchResult, cap-1)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for e := range buf {
			sender.Send(e)
		}
	}()
	cleanup := func() {
		close(buf)
		<-done
	}
	return backend.ZoektStreamFunc(func(event *zoekt.SearchResult) {
		buf <- event
	}), cleanup
}

// zoektSearchReposOnly is used when select:repo is set, in which case we can ask zoekt
// only for the repos that contain matches for the query. This is a performance optimization,
// and not required for proper function of select:repo.
func zoektSearchReposOnly(ctx context.Context, client zoekt.Streamer, query zoektquery.Q, c streaming.Sender, getRepoRevMap func() map[api.RepoID]*search.RepositoryRevisions) error {
	repoList, err := client.List(ctx, query, &zoekt.ListOptions{Minimal: true})
	if err != nil {
		return err
	}

	repoRevMap := getRepoRevMap()
	if repoRevMap == nil {
		return nil
	}

	matches := make([]result.Match, 0, len(repoList.Minimal))
	for id := range repoList.Minimal {
		rev, ok := repoRevMap[api.RepoID(id)]
		if !ok {
			continue
		}

		matches = append(matches, &result.RepoMatch{
			Name: rev.Repo.Name,
			ID:   rev.Repo.ID,
		})
	}

	c.Send(streaming.SearchEvent{
		Results: matches,
		Stats:   streaming.Stats{}, // TODO
	})
	return nil
}

func zoektFileMatchToLineMatches(file *zoekt.FileMatch) []*result.LineMatch {
	lines := make([]*result.LineMatch, 0, len(file.LineMatches))

	for _, l := range file.LineMatches {
		if l.FileName {
			continue
		}

		offsets := make([][2]int32, len(l.LineFragments))
		for k, m := range l.LineFragments {
			offset := utf8.RuneCount(l.Line[:m.LineOffset])
			length := utf8.RuneCount(l.Line[m.LineOffset : m.LineOffset+m.MatchLength])
			offsets[k] = [2]int32{int32(offset), int32(length)}
		}
		lines = append(lines, &result.LineMatch{
			Preview:          string(l.Line),
			LineNumber:       int32(l.LineNumber - 1),
			OffsetAndLengths: offsets,
		})
	}

	return lines
}

func escape(s string) string {
	isSpecial := func(c rune) bool {
		switch c {
		case '\\', '/':
			return true
		default:
			return false
		}
	}

	// Avoid extra work by counting additions. regexp.QuoteMeta does the same,
	// but is more efficient since it does it via bytes.
	count := 0
	for _, c := range s {
		if isSpecial(c) {
			count++
		}
	}
	if count == 0 {
		return s
	}

	escaped := make([]rune, 0, len(s)+count)
	for _, c := range s {
		if isSpecial(c) {
			escaped = append(escaped, '\\')
		}
		escaped = append(escaped, c)
	}
	return string(escaped)
}

func zoektFileMatchToSymbolResults(repoName types.RepoName, inputRev string, file *zoekt.FileMatch) []*result.SymbolMatch {
	newFile := &result.File{
		Path:     file.FileName,
		Repo:     repoName,
		CommitID: api.CommitID(file.Version),
		InputRev: &inputRev,
	}

	symbols := make([]*result.SymbolMatch, 0, len(file.LineMatches))
	for _, l := range file.LineMatches {
		if l.FileName {
			continue
		}

		for _, m := range l.LineFragments {
			if m.SymbolInfo == nil {
				continue
			}

			symbols = append(symbols, &result.SymbolMatch{
				Symbol: result.Symbol{
					Name:       m.SymbolInfo.Sym,
					Kind:       m.SymbolInfo.Kind,
					Parent:     m.SymbolInfo.Parent,
					ParentKind: m.SymbolInfo.ParentKind,
					Path:       file.FileName,
					Line:       l.LineNumber,
					// symbolRange requires a Pattern /^...$/ containing the line of the symbol to compute the symbol's offsets.
					// This Pattern is directly accessible on the unindexed code path. But on the indexed code path, we need to
					// populate it, or we will always compute a 0 offset, which messes up API use (e.g., highlighting).
					// It must escape `/` or `\` in the line.
					Pattern:  fmt.Sprintf("/^%s$/", escape(string(l.Line))),
					Language: file.Language,
				},
				File: newFile,
			})
		}
	}

	return symbols
}

// contextWithoutDeadline returns a context which will cancel if the cOld is
// canceled.
func contextWithoutDeadline(cOld context.Context) (context.Context, context.CancelFunc) {
	cNew, cancel := context.WithCancel(context.Background())

	// Set trace context so we still get spans propagated
	cNew = trace.CopyContext(cNew, cOld)

	// Copy actor from cOld to cNew.
	cNew = actor.WithActor(cNew, actor.FromContext(cOld))

	go func() {
		select {
		case <-cOld.Done():
			// cancel the new context if the old one is done for some reason other than the deadline passing.
			if cOld.Err() != context.DeadlineExceeded {
				cancel()
			}
		case <-cNew.Done():
		}
	}()

	return cNew, cancel
}

func queryToZoektQuery(query *search.TextPatternInfo, typ IndexedRequestType) (zoektquery.Q, error) {
	var and []zoektquery.Q

	var q zoektquery.Q
	var err error
	if query.IsRegExp {
		fileNameOnly := query.PatternMatchesPath && !query.PatternMatchesContent
		contentOnly := !query.PatternMatchesPath && query.PatternMatchesContent
		q, err = parseRe(query.Pattern, fileNameOnly, contentOnly, query.IsCaseSensitive)
		if err != nil {
			return nil, err
		}
	} else {
		q = &zoektquery.Substring{
			Pattern:       query.Pattern,
			CaseSensitive: query.IsCaseSensitive,

			FileName: true,
			Content:  true,
		}
	}

	if query.IsNegated {
		q = &zoektquery.Not{Child: q}
	}

	if typ == SymbolRequest {
		// Tell zoekt q must match on symbols
		q = &zoektquery.Symbol{
			Expr: q,
		}
	}

	and = append(and, q)

	// zoekt also uses regular expressions for file paths
	// TODO PathPatternsAreCaseSensitive
	// TODO whitespace in file path patterns?
	for _, p := range query.IncludePatterns {
		q, err := FileRe(p, query.IsCaseSensitive)
		if err != nil {
			return nil, err
		}
		and = append(and, q)
	}
	if query.ExcludePattern != "" {
		q, err := FileRe(query.ExcludePattern, query.IsCaseSensitive)
		if err != nil {
			return nil, err
		}
		and = append(and, &zoektquery.Not{Child: q})
	}

	// For conditionals that happen on a repo we can use type:repo queries. eg
	// (type:repo file:foo) (type:repo file:bar) will match all repos which
	// contain a filename matching "foo" and a filename matchinb "bar".
	//
	// Note: (type:repo file:foo file:bar) will only find repos with a
	// filename containing both "foo" and "bar".
	for _, p := range query.FilePatternsReposMustInclude {
		q, err := FileRe(p, query.IsCaseSensitive)
		if err != nil {
			return nil, err
		}
		and = append(and, &zoektquery.Type{Type: zoektquery.TypeRepo, Child: q})
	}
	for _, p := range query.FilePatternsReposMustExclude {
		q, err := FileRe(p, query.IsCaseSensitive)
		if err != nil {
			return nil, err
		}
		and = append(and, &zoektquery.Not{Child: &zoektquery.Type{Type: zoektquery.TypeRepo, Child: q}})
	}

	return zoektquery.Simplify(zoektquery.NewAnd(and...)), nil
}

// zoektIndexedRepos splits the revs into two parts: (1) the repository
// revisions in indexedSet (indexed) and (2) the repositories that are
// unindexed.
func zoektIndexedRepos(indexedSet map[string]*zoekt.Repository, revs []*search.RepositoryRevisions, filter func(*zoekt.Repository) bool) (indexed *IndexedRepoRevs, unindexed []*search.RepositoryRevisions) {
	// PERF: If len(revs) is large, we expect to be doing an indexed
	// search. So set indexed to the max size it can be to avoid growing.
	indexed = &IndexedRepoRevs{
		repoRevs:     make(map[string]*search.RepositoryRevisions, len(revs)),
		repoBranches: make(map[string][]string, len(revs)),
	}
	unindexed = make([]*search.RepositoryRevisions, 0)

	for _, reporev := range revs {
		repo, ok := indexedSet[string(reporev.Repo.Name)]
		if !ok || (filter != nil && !filter(repo)) {
			unindexed = append(unindexed, reporev)
			continue
		}

		unindexedRevs := indexed.add(reporev, repo)
		if len(unindexedRevs) > 0 {
			copy := reporev.Copy()
			copy.Revs = unindexedRevs
			unindexed = append(unindexed, copy)
		}
	}

	return indexed, unindexed
}

// limitUnindexedRepos limits the number of repo@revs searched by the
// unindexed searcher codepath.  Sending many requests to searcher would
// otherwise cause a flood of system and network requests that result in
// timeouts or long delays.
//
// It returns the new repositories destined for the unindexed searcher code
// path, and sends an event to stream for any repositories that are limited /
// excluded.
//
// A slice to the input list is returned, it is not copied.
func limitUnindexedRepos(unindexed []*search.RepositoryRevisions, limit int, stream streaming.Sender) []*search.RepositoryRevisions {
	var missing []*search.RepositoryRevisions

	for i, repoRevs := range unindexed {
		limit -= len(repoRevs.Revs)
		if limit < 0 {
			missing = unindexed[i:]
			unindexed = unindexed[:i]
			break
		}
	}

	if len(missing) > 0 {
		var status search.RepoStatusMap
		for _, r := range missing {
			status.Update(r.Repo.ID, search.RepoStatusMissing)
		}
		stream.Send(streaming.SearchEvent{
			Stats: streaming.Stats{
				Status: status,
			},
		})
	}

	return unindexed
}
