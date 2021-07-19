package run

import (
	"context"
	"math"
	"regexp"
	"runtime"

	otlog "github.com/opentracing/opentracing-go/log"

	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/search"
	"github.com/sourcegraph/sourcegraph/internal/search/query"
	"github.com/sourcegraph/sourcegraph/internal/search/result"
	"github.com/sourcegraph/sourcegraph/internal/search/streaming"
	"github.com/sourcegraph/sourcegraph/internal/search/unindexed"
	"github.com/sourcegraph/sourcegraph/internal/trace"
)

var MockSearchRepositories func(args *search.TextParameters) ([]result.Match, *streaming.Stats, error)

// SearchRepositories searches for repositories by name.
//
// For a repository to match a query, the repository's name must match all of the repo: patterns AND the
// default patterns (i.e., the patterns that are not prefixed with any search field).
func SearchRepositories(ctx context.Context, args *search.TextParameters, limit int32, stream streaming.Sender) (err error) {
	if MockSearchRepositories != nil {
		results, stats, err := MockSearchRepositories(args)
		stream.Send(streaming.SearchEvent{
			Results: results,
			Stats:   stats.Deref(),
		})
		return err
	}

	tr, ctx := trace.New(ctx, "run.SearchRepositories", "")
	defer func() {
		tr.SetError(err)
		tr.Finish()
	}()

	ctx, stream, cancel := streaming.WithLimit(ctx, stream, int(limit))
	defer cancel()

	fieldAllowlist := map[string]struct{}{
		query.FieldRepo:               {},
		query.FieldRepoGroup:          {},
		query.FieldContext:            {},
		query.FieldType:               {},
		query.FieldDefault:            {},
		query.FieldIndex:              {},
		query.FieldCount:              {},
		query.FieldTimeout:            {},
		query.FieldFork:               {},
		query.FieldArchived:           {},
		query.FieldVisibility:         {},
		query.FieldCase:               {},
		query.FieldRepoHasFile:        {},
		query.FieldRepoHasCommitAfter: {},
		query.FieldPatternType:        {},
		query.FieldSelect:             {},
	}
	// Don't return repo results if the search contains fields that aren't on the allowlist.
	// Matching repositories based whether they contain files at a certain path (etc.) is not yet implemented.
	for field := range args.Query.Fields() {
		if _, ok := fieldAllowlist[field]; !ok {
			tr.LazyPrintf("contains dissallowed field: %s", field)
			return nil
		}
	}

	patternRe := args.PatternInfo.Pattern
	if !args.Query.IsCaseSensitive() {
		patternRe = "(?i)" + patternRe
	}

	tr.LogFields(
		otlog.String("pattern", patternRe),
		otlog.Int32("limit", limit))

	pattern, err := regexp.Compile(patternRe)
	if err != nil {
		return err
	}

	// Filter args.Repos by matching their names against the query pattern.
	resolved, err := args.RepoPromise.Get(ctx)
	if err != nil {
		return err
	}

	tr.LogFields(otlog.Int("resolved.len", len(resolved)))

	results := make(chan []*search.RepositoryRevisions)
	go func() {
		defer close(results)
		matchRepos(pattern, resolved, results)
	}()

	// Filter the repos if there is a repohasfile: or -repohasfile field.
	if len(args.PatternInfo.FilePatternsReposMustExclude) > 0 || len(args.PatternInfo.FilePatternsReposMustInclude) > 0 {
		// Fallback to batch for reposToAdd
		var repos []*search.RepositoryRevisions
		for matched := range results {
			repos = append(repos, matched...)
		}
		repos, err = reposToAdd(ctx, args, repos)
		if err != nil {
			return err
		}
		stream.Send(streaming.SearchEvent{
			Results: repoRevsToRepoMatches(ctx, repos),
		})
		return nil
	}

	count := 0
	for repos := range results {
		count += len(repos)
		stream.Send(streaming.SearchEvent{
			Results: repoRevsToRepoMatches(ctx, repos),
		})
	}
	tr.LogFields(otlog.Int("matched.len", count))

	return nil
}

func repoRevsToRepoMatches(ctx context.Context, repos []*search.RepositoryRevisions) []result.Match {
	matches := make([]result.Match, 0, len(repos))
	for _, r := range repos {
		revs, err := r.ExpandedRevSpecs(ctx)
		if err != nil { // fallback to just return revspecs
			revs = r.RevSpecs()
		}
		for _, rev := range revs {
			matches = append(matches, &result.RepoMatch{
				Name: r.Repo.Name,
				ID:   r.Repo.ID,
				Rev:  rev,
			})
		}
	}
	return matches
}

func matchRepos(pattern *regexp.Regexp, resolved []*search.RepositoryRevisions, results chan<- []*search.RepositoryRevisions) {
	/*
		goos: linux
		goarch: amd64
		pkg: github.com/sourcegraph/sourcegraph/internal/search/run
		cpu: Intel(R) Core(TM) i9-8950HK CPU @ 2.90GHz
		BenchmarkSearchRepositories-8   	      39	  31514267 ns/op
		BenchmarkSearchRepositories-8   	      26	  38898255 ns/op
		BenchmarkSearchRepositories-8   	      36	  31482727 ns/op
		BenchmarkSearchRepositories-8   	      33	  30513691 ns/op
		BenchmarkSearchRepositories-8   	      30	  37038388 ns/op
		BenchmarkSearchRepositories-8   	      30	  38095363 ns/op
		BenchmarkSearchRepositories-8   	      36	  39347784 ns/op
		BenchmarkSearchRepositories-8   	      28	  41431416 ns/op
		BenchmarkSearchRepositories-8   	      30	  41695426 ns/op
		BenchmarkSearchRepositories-8   	      28	  39782412 ns/op
		PASS
		ok  	github.com/sourcegraph/sourcegraph/internal/search/run	18.729s
	*/
	workers := runtime.NumCPU()
	limit := len(resolved) / workers

	last := make(chan struct{})
	close(last)

	for i := 0; i < workers; i++ {
		page := resolved[i*limit : i*limit+limit]
		if i == workers-1 {
			page = resolved[i*limit:]
		}

		wait := last
		done := make(chan struct{})
		last = done

		go func() {
			defer close(done)

			var matched []*search.RepositoryRevisions
			for _, r := range page {
				if pattern.MatchString(string(r.Repo.Name)) {
					matched = append(matched, r)
				}
			}

			// Wait for the previous chunk to send its matches
			// before sending ours.
			<-wait
			results <- matched
		}()
	}

	<-last
}

// reposToAdd determines which repositories should be included in the result set based on whether they fit in the subset
// of repostiories specified in the query's `repohasfile` and `-repohasfile` fields if they exist.
func reposToAdd(ctx context.Context, args *search.TextParameters, repos []*search.RepositoryRevisions) ([]*search.RepositoryRevisions, error) {
	// matchCounts will contain the count of repohasfile patterns that matched.
	// For negations, we will explicitly set this to -1 if it matches.
	matchCounts := make(map[api.RepoID]int)
	if len(args.PatternInfo.FilePatternsReposMustInclude) > 0 {
		for _, pattern := range args.PatternInfo.FilePatternsReposMustInclude {
			// The high FileMatchLimit here is to make sure we get all the repo matches we can. Setting it to
			// len(repos) could mean we miss some repos since there could be for example len(repos) file matches in
			// the first repo and some more in other repos.
			p := search.TextPatternInfo{IsRegExp: true, FileMatchLimit: math.MaxInt32, IncludePatterns: []string{pattern}, PathPatternsAreCaseSensitive: false, PatternMatchesContent: true, PatternMatchesPath: true}
			q, err := query.ParseLiteral("file:" + pattern)
			if err != nil {
				return nil, err
			}
			newArgs := *args
			newArgs.PatternInfo = &p
			newArgs.RepoPromise = (&search.RepoPromise{}).Resolve(repos)
			newArgs.Query = q
			newArgs.UseFullDeadline = true

			if newArgs.PatternInfo.IsEmpty() {
				// Empty query isn't an error, but it has no results.
				return nil, nil
			}
			matches, _, err := unindexed.SearchFilesInReposBatch(ctx, &newArgs)
			if err != nil {
				return nil, err
			}

			// deduplicate repo results
			matchedIDs := make(map[api.RepoID]struct{})
			for _, m := range matches {
				matchedIDs[m.Repo.ID] = struct{}{}
			}

			// increment the count for all seen repos
			for id := range matchedIDs {
				matchCounts[id] += 1
			}
		}
	} else {
		// Default to including all the repos, then excluding some of them below.
		for _, r := range repos {
			matchCounts[r.Repo.ID] = 0
		}
	}

	if len(args.PatternInfo.FilePatternsReposMustExclude) > 0 {
		for _, pattern := range args.PatternInfo.FilePatternsReposMustExclude {
			p := search.TextPatternInfo{IsRegExp: true, FileMatchLimit: math.MaxInt32, IncludePatterns: []string{pattern}, PathPatternsAreCaseSensitive: false, PatternMatchesContent: true, PatternMatchesPath: true}
			q, err := query.ParseLiteral("file:" + pattern)
			if err != nil {
				return nil, err
			}
			newArgs := *args
			newArgs.PatternInfo = &p
			rp := (&search.RepoPromise{}).Resolve(repos)
			newArgs.RepoPromise = rp
			newArgs.Query = q
			newArgs.UseFullDeadline = true
			if newArgs.PatternInfo.IsEmpty() {
				// Empty query isn't an error, but it has no results.
				return nil, nil
			}

			matches, _, err := unindexed.SearchFilesInReposBatch(ctx, &newArgs)
			if err != nil {
				return nil, err
			}
			for _, m := range matches {
				matchCounts[m.Repo.ID] = -1
			}
		}
	}

	var rsta []*search.RepositoryRevisions
	for _, r := range repos {
		if count, ok := matchCounts[r.Repo.ID]; ok && count == len(args.PatternInfo.FilePatternsReposMustInclude) {
			rsta = append(rsta, r)
		}
	}

	return rsta, nil
}
