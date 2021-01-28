package graphqlbackend

import (
	"context"
	"errors"
	"fmt"
	"path"
	"regexp"
	rxsyntax "regexp/syntax"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/hashicorp/go-multierror"
	"github.com/inconshreveable/log15"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/envvar"
	searchrepos "github.com/sourcegraph/sourcegraph/cmd/frontend/internal/search/repos"
	"github.com/sourcegraph/sourcegraph/internal/comby"
	"github.com/sourcegraph/sourcegraph/internal/search"
	"github.com/sourcegraph/sourcegraph/internal/search/query"
	querytypes "github.com/sourcegraph/sourcegraph/internal/search/query/types"
	"github.com/sourcegraph/sourcegraph/internal/search/streaming"
)

type searchAlert struct {
	prometheusType  string
	title           string
	description     string
	proposedQueries []*searchQueryDescription
	// The higher the priority the more important is the alert.
	priority int
}

func (a searchAlert) Title() string { return a.title }

func (a searchAlert) Description() *string {
	if a.description == "" {
		return nil
	}
	return &a.description
}

func (a searchAlert) ProposedQueries() *[]*searchQueryDescription {
	if len(a.proposedQueries) == 0 {
		return nil
	}
	return &a.proposedQueries
}

func alertForCappedAndExpression() *searchAlert {
	return &searchAlert{
		prometheusType: "exceed_and_expression_search_limit",
		title:          "Too many files to search for and-expression",
		description:    "One and-expression in the query requires a lot of work! Try using the 'repo:' or 'file:' filters to narrow your search. We're working on improving this experience in https://github.com/sourcegraph/sourcegraph/issues/9824",
	}
}

func toSearchQueryDescription(proposed []*query.ProposedQuery) (result []*searchQueryDescription) {
	for _, p := range proposed {
		result = append(result, &searchQueryDescription{
			description: p.Description,
			query:       p.Query,
		})
	}
	return result
}

// alertForQuery converts errors in the query to search alerts.
func alertForQuery(queryString string, err error) *searchAlert {
	switch e := err.(type) {
	case *query.LegacyParseError:
		return &searchAlert{
			prometheusType:  "parse_syntax_error",
			title:           capFirst(e.Msg),
			description:     "Quoting the query may help if you want a literal match.",
			proposedQueries: toSearchQueryDescription(query.ProposedQuotedQueries(queryString)),
		}
	case *query.ValidationError:
		return &searchAlert{
			prometheusType: "validation_error",
			title:          "Invalid Query",
			description:    capFirst(e.Msg),
		}
	case *querytypes.TypeError:
		switch e := e.Err.(type) {
		case *rxsyntax.Error:
			return &searchAlert{
				prometheusType:  "typecheck_regex_syntax_error",
				title:           capFirst(e.Error()),
				description:     "Quoting the query may help if you want a literal match instead of a regular expression match.",
				proposedQueries: toSearchQueryDescription(query.ProposedQuotedQueries(queryString)),
			}
		}
	case *query.UnsupportedError, *query.ExpectedOperand:
		return &searchAlert{
			prometheusType: "unsupported_and_or_query",
			title:          "Unable To Process Query",
			description:    `I'm having trouble understanding that query. Your query contains "and" or "or" operators that make me think they apply to filters like "repo:" or "file:". We only support "and" or "or" operators on search patterns for file contents currently. You can help me by putting parentheses around the search pattern.`,
		}
	}
	return &searchAlert{
		prometheusType: "generic_invalid_query",
		title:          "Unable To Process Query",
		description:    capFirst(err.Error()),
	}
}

func alertForTimeout(usedTime time.Duration, suggestTime time.Duration, r *searchResolver) *searchAlert {
	return &searchAlert{
		prometheusType: "timed_out",
		title:          "Timed out while searching",
		description:    fmt.Sprintf("We weren't able to find any results in %s.", roundStr(usedTime.String())),
		proposedQueries: []*searchQueryDescription{
			{
				description: "query with longer timeout",
				query:       fmt.Sprintf("timeout:%v %s", suggestTime, query.OmitQueryField(r.Query.ParseTree(), query.FieldTimeout)),
				patternType: r.PatternType,
			},
		},
	}
}

func alertForStalePermissions() *searchAlert {
	return &searchAlert{
		prometheusType: "no_resolved_repos__stale_permissions",
		title:          "Permissions syncing in progress",
		description:    "Permissions are being synced from your code host, please wait for a minute and try again.",
	}
}

// reposExist returns true if one or more repos resolve. If the attempt
// returns 0 repos or fails, it returns false. It is a helper function for
// raising NoResolvedRepos alerts with suggestions when we know the original
// query does not contain any repos to search.
func (r *searchResolver) reposExist(ctx context.Context, options searchrepos.Options) bool {
	options.UserSettings = r.UserSettings
	resolved, err := searchrepos.ResolveRepositories(ctx, options)
	return err == nil && len(resolved.RepoRevs) > 0
}

func (r *searchResolver) alertForNoResolvedRepos(ctx context.Context) *searchAlert {
	globbing := getBoolPtr(r.UserSettings.SearchGlobbing, false)

	repoFilters, minusRepoFilters := r.Query.RegexpPatterns(query.FieldRepo)
	repoGroupFilters, _ := r.Query.StringValues(query.FieldRepoGroup)
	fork, _ := r.Query.StringValue(query.FieldFork)
	onlyForks, noForks := fork == "only", fork == "no"
	forksNotSet := len(fork) == 0
	archived, _ := r.Query.StringValue(query.FieldArchived)
	archivedNotSet := len(archived) == 0

	// Handle repogroup-only scenarios.
	if len(repoFilters) == 0 && len(repoGroupFilters) == 0 {
		return &searchAlert{
			prometheusType: "no_resolved_repos__no_repositories",
			title:          "Add repositories or connect repository hosts",
			description:    "There are no repositories to search. Add an external service connection to your code host.",
		}
	}
	if len(repoFilters) == 0 && len(repoGroupFilters) == 1 {
		return &searchAlert{
			prometheusType: "no_resolved_repos__repogroup_empty",
			title:          fmt.Sprintf("Add repositories to repogroup:%s to see results", repoGroupFilters[0]),
			description:    fmt.Sprintf("The repository group %q is empty. See the documentation for configuration and troubleshooting.", repoGroupFilters[0]),
		}
	}
	if len(repoFilters) == 0 && len(repoGroupFilters) > 1 {
		return &searchAlert{
			prometheusType: "no_resolved_repos__repogroup_none_in_common",
			title:          "Repository groups have no repositories in common",
			description:    "No repository exists in all of the specified repository groups.",
		}
	}

	// TODO(sqs): handle -repo:foo fields.

	withoutRepoFields := query.OmitQueryField(r.Query.ParseTree(), query.FieldRepo)

	switch {
	case len(repoGroupFilters) > 1:
		// This is a rare case, so don't bother proposing queries.
		return &searchAlert{
			prometheusType: "no_resolved_repos__more_than_one_repogroup",
			title:          "No repository exists in all specified groups and satisfies all of your repo: filters.",
			description:    "Expand your repository filters to see results",
		}

	case len(repoGroupFilters) == 1 && len(repoFilters) > 1:
		if globbing {
			return &searchAlert{
				prometheusType: "no_resolved_repos__try_remove_filters_for_repogroup",
				title:          fmt.Sprintf("No repositories in repogroup:%s satisfied all of your repo: filters.", repoGroupFilters[0]),
				description:    "Remove repo: filters to see results",
			}
		}
		proposedQueries := []*searchQueryDescription{}
		tryRemoveRepoGroup := searchrepos.Options{
			RepoFilters:      repoFilters,
			MinusRepoFilters: minusRepoFilters,
			OnlyForks:        onlyForks,
			NoForks:          noForks,
		}
		if r.reposExist(ctx, tryRemoveRepoGroup) {
			proposedQueries = []*searchQueryDescription{
				{
					description: fmt.Sprintf("include repositories outside of repogroup:%s", repoGroupFilters[0]),
					query:       query.OmitQueryField(r.Query.ParseTree(), query.FieldRepoGroup),
					patternType: r.PatternType,
				},
			}
		}

		unionRepoFilter := searchrepos.UnionRegExps(repoFilters)
		tryAnyRepo := searchrepos.Options{
			RepoFilters:      []string{unionRepoFilter},
			MinusRepoFilters: minusRepoFilters,
			RepoGroupFilters: repoGroupFilters,
			OnlyForks:        onlyForks,
			NoForks:          noForks,
		}
		if r.reposExist(ctx, tryAnyRepo) {
			proposedQueries = append(proposedQueries, &searchQueryDescription{
				description: "include repositories satisfying any (not all) of your repo: filters",
				query:       withoutRepoFields + fmt.Sprintf(" repo:%s", unionRepoFilter),
				patternType: r.PatternType,
			})
		} else {
			// Fall back to removing repo filters.
			proposedQueries = append(proposedQueries, &searchQueryDescription{
				description: "remove repo: filters",
				query:       withoutRepoFields,
				patternType: r.PatternType,
			})
		}

		return &searchAlert{
			prometheusType:  "no_resolved_repos__try_remove_filters_for_repogroup",
			title:           fmt.Sprintf("No repositories in repogroup:%s satisfied all of your repo: filters.", repoGroupFilters[0]),
			description:     "Expand your repository filters to see results",
			proposedQueries: proposedQueries,
		}

	case len(repoGroupFilters) == 1 && len(repoFilters) == 1:
		if globbing {
			return &searchAlert{
				prometheusType: "no_resolved_repogroups",
				title:          fmt.Sprintf("No repositories in repogroup:%s satisfied all of your repo: filters.", repoGroupFilters[0]),
				description:    "Remove repo: filters to see results",
			}
		}
		proposedQueries := []*searchQueryDescription{}
		tryRemoveRepoGroup := searchrepos.Options{
			RepoFilters:      repoFilters,
			MinusRepoFilters: minusRepoFilters,
			OnlyForks:        onlyForks,
			NoForks:          noForks,
		}
		if r.reposExist(ctx, tryRemoveRepoGroup) {
			proposedQueries = []*searchQueryDescription{
				{
					description: fmt.Sprintf("include repositories outside of repogroup:%s", repoGroupFilters[0]),
					query:       query.OmitQueryField(r.Query.ParseTree(), query.FieldRepoGroup),
					patternType: r.PatternType,
				},
			}
		}

		proposedQueries = append(proposedQueries, &searchQueryDescription{
			description: "remove repo: filters",
			query:       withoutRepoFields,
			patternType: r.PatternType,
		})
		return &searchAlert{
			prometheusType:  "no_resolved_repogroups",
			title:           fmt.Sprintf("No repositories in repogroup:%s satisfied all of your repo: filters.", repoGroupFilters[0]),
			description:     "Expand your repository filters to see results",
			proposedQueries: proposedQueries,
		}

	case len(repoGroupFilters) == 0 && len(repoFilters) > 1:
		if globbing {
			return &searchAlert{
				prometheusType: "no_resolved_repos__suggest_add_remove_repos",
				title:          "No repositories satisfied all of your repo: filters.",
				description:    "Remove repo: filters to see results",
			}
		}
		proposedQueries := []*searchQueryDescription{}
		unionRepoFilter := searchrepos.UnionRegExps(repoFilters)
		tryAnyRepo := searchrepos.Options{
			RepoFilters:      []string{unionRepoFilter},
			MinusRepoFilters: minusRepoFilters,
			RepoGroupFilters: repoGroupFilters,
			OnlyForks:        onlyForks,
			NoForks:          noForks,
		}
		if r.reposExist(ctx, tryAnyRepo) {
			proposedQueries = append(proposedQueries, &searchQueryDescription{
				description: "include repositories satisfying any (not all) of your repo: filters",
				query:       withoutRepoFields + fmt.Sprintf(" repo:%s", unionRepoFilter),
				patternType: r.PatternType,
			})
		}

		proposedQueries = append(proposedQueries, &searchQueryDescription{
			description: "remove repo: filters",
			query:       withoutRepoFields,
		})
		return &searchAlert{
			prometheusType:  "no_resolved_repos__suggest_add_remove_repos",
			title:           "No repositories satisfied all of your repo: filters.",
			description:     "Expand your repo: filters to see results",
			proposedQueries: proposedQueries,
		}

	case len(repoGroupFilters) == 0 && len(repoFilters) == 1:
		isSiteAdmin := backend.CheckCurrentUserIsSiteAdmin(ctx) == nil
		if !envvar.SourcegraphDotComMode() {
			if needsRepoConfig, err := needsRepositoryConfiguration(ctx); err == nil && needsRepoConfig {
				if isSiteAdmin {
					return &searchAlert{
						title:       "No repositories or code hosts configured",
						description: "To start searching code, first go to site admin to configure repositories and code hosts.",
					}

				} else {
					return &searchAlert{
						title:       "No repositories or code hosts configured",
						description: "To start searching code, ask the site admin to configure and enable repositories.",
					}
				}
			}
		}

		if globbing {
			return &searchAlert{
				prometheusType: "no_resolved_repos__generic",
				title:          "No repositories satisfied your repo: filter",
				description:    "Modify your repo: filter to see results",
			}
		}

		proposedQueries := []*searchQueryDescription{}
		if forksNotSet {
			tryIncludeForks := searchrepos.Options{
				RepoFilters:      repoFilters,
				MinusRepoFilters: minusRepoFilters,
				NoForks:          false,
			}
			if r.reposExist(ctx, tryIncludeForks) {
				proposedQueries = append(proposedQueries, &searchQueryDescription{
					description: "include forked repositories in your query.",
					query:       r.OriginalQuery + " fork:yes",
					patternType: r.PatternType,
				})
			}
		}

		if archivedNotSet {
			tryIncludeArchived := searchrepos.Options{
				RepoFilters:      repoFilters,
				MinusRepoFilters: minusRepoFilters,
				OnlyForks:        onlyForks,
				NoForks:          noForks,
				OnlyArchived:     true,
			}
			if r.reposExist(ctx, tryIncludeArchived) {
				proposedQueries = append(proposedQueries, &searchQueryDescription{
					description: "include archived repositories in your query.",
					query:       r.OriginalQuery + " archived:yes",
					patternType: r.PatternType,
				})
			}
		}

		if strings.TrimSpace(withoutRepoFields) != "" {
			proposedQueries = append(proposedQueries, &searchQueryDescription{
				description: "remove repo: filter",
				query:       withoutRepoFields,
				patternType: r.PatternType,
			})
		}
		return &searchAlert{
			prometheusType:  "no_resolved_repos__generic",
			title:           "No repositories satisfied your repo: filter",
			description:     "Modify your repo: filter to see results",
			proposedQueries: proposedQueries,
		}
	}
	// Should be unreachable. Return a generic alert if reached.
	return &searchAlert{
		title:       "No repository results.",
		description: "There are no repositories to search.",
	}
}

func (r *searchResolver) alertForInvalidRevision(revision string) *searchAlert {
	revision = strings.TrimSuffix(revision, "^0")
	return &searchAlert{
		title:       "Invalid revision syntax",
		description: fmt.Sprintf("We don't know how to interpret the revision (%s) you specified. Learn more about the revision syntax in our documentation: https://docs.sourcegraph.com/code_search/reference/queries#repository-revisions.", revision),
	}
}

func (r *searchResolver) alertForOverRepoLimit(ctx context.Context) *searchAlert {
	// Try to suggest the most helpful repo: filters to narrow the query.
	//
	// For example, suppose the query contains "repo:kubern" and it matches > 30
	// repositories, and each one of the (clipped result set of) 30 repos has
	// "kubernetes" in their path. Then it's likely that the user would want to
	// search for "repo:kubernetes". If that still matches > 30 repositories,
	// then try to narrow it further using "/kubernetes/", etc.
	//
	// (In the above sample paragraph, we assume MAX_REPOS_TO_SEARCH is 30.)
	//
	// TODO(sqs): this logic can be significantly improved, but it's better than
	// nothing for now.

	var proposedQueries []*searchQueryDescription
	description := "Use a 'repo:' or 'repogroup:' filter to narrow your search and see results."
	if envvar.SourcegraphDotComMode() {
		description = "Use a 'repo:' or 'repogroup:' filter to narrow your search and see results or set up a self-hosted Sourcegraph instance to search an unlimited number of repositories."
	}
	if backend.CheckCurrentUserIsSiteAdmin(ctx) == nil {
		description += " As a site admin, you can increase the limit by changing maxReposToSearch in site config."
	}

	buildAlert := func(proposedQueries []*searchQueryDescription, description string) *searchAlert {
		return &searchAlert{
			prometheusType:  "over_repo_limit",
			title:           "Too many matching repositories",
			proposedQueries: proposedQueries,
			description:     description,
		}
	}

	// If globbing is active we return a simple alert for now. The alert is still
	// helpful but it doesn't contain any proposed queries.
	if getBoolPtr(r.UserSettings.SearchGlobbing, false) {
		return buildAlert(proposedQueries, description)
	}

	resolved, _ := r.resolveRepositories(ctx, nil)
	if len(resolved.RepoRevs) > 0 {
		paths := make([]string, len(resolved.RepoRevs))
		for i, repo := range resolved.RepoRevs {
			paths[i] = string(repo.Repo.Name)
		}

		// See if we can narrow it down by using filters like
		// repo:github.com/myorg/.
		const maxParentsToPropose = 4
		ctx, cancel := context.WithTimeout(ctx, 1500*time.Millisecond)
		defer cancel()
	outer:
		for i, repoParent := range pathParentsByFrequency(paths) {
			if i >= maxParentsToPropose || ctx.Err() != nil {
				break
			}
			repoParentPattern := "^" + regexp.QuoteMeta(repoParent) + "/"
			repoFieldValues, _ := r.Query.RegexpPatterns(query.FieldRepo)

			for _, v := range repoFieldValues {
				if strings.HasPrefix(v, strings.TrimSuffix(repoParentPattern, "/")) {
					continue outer // this repo: filter is already applied
				}
			}

			repoFieldValues = append(repoFieldValues, repoParentPattern)
			ctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
			defer cancel()
			resolved, err := r.resolveRepositories(ctx, repoFieldValues)
			if ctx.Err() != nil {
				continue
			} else if err != nil {
				return buildAlert([]*searchQueryDescription{}, description)
			}

			var more string
			if resolved.OverLimit {
				more = "(further filtering required)"
			}
			// We found a more specific repo: filter that may be narrow enough. Now
			// add it to the user's query, but be smart. For example, if the user's
			// query was "repo:foo" and the parent is "foobar/", then propose "repo:foobar/"
			// not "repo:foo repo:foobar/" (which are equivalent, but shorter is better).
			newExpr := query.AddRegexpField(r.Query.ParseTree(), query.FieldRepo, repoParentPattern)
			proposedQueries = append(proposedQueries, &searchQueryDescription{
				description: fmt.Sprintf("in repositories under %s %s", repoParent, more),
				query:       newExpr,
				patternType: r.PatternType,
			})
		}
		if len(proposedQueries) == 0 || ctx.Err() == context.DeadlineExceeded {
			// Propose specific repos' paths if we aren't able to propose
			// anything else.
			const maxReposToPropose = 4
			shortest := append([]string{}, paths...) // prefer shorter repo names
			sort.Slice(shortest, func(i, j int) bool {
				return len(shortest[i]) < len(shortest[j]) || (len(shortest[i]) == len(shortest[j]) && shortest[i] < shortest[j])
			})
			for i, pathToPropose := range shortest {
				if i >= maxReposToPropose {
					break
				}
				newExpr := query.AddRegexpField(r.Query.ParseTree(), query.FieldRepo, "^"+regexp.QuoteMeta(pathToPropose)+"$")
				proposedQueries = append(proposedQueries, &searchQueryDescription{
					description: fmt.Sprintf("in the repository %s", strings.TrimPrefix(pathToPropose, "github.com/")),
					query:       newExpr,
					patternType: r.PatternType,
				})
			}
		}
	}
	return buildAlert(proposedQueries, description)
}

func alertForStructuralSearchNotSet(queryString string) *searchAlert {
	return &searchAlert{
		prometheusType: "structural_search_not_set",
		title:          "No results",
		description:    "It looks like you may have meant to run a structural search, but it is not toggled.",
		proposedQueries: []*searchQueryDescription{{
			description: "Activate structural search",
			query:       queryString,
			patternType: query.SearchTypeStructural,
		}},
	}
}

func alertForMissingRepoRevs(patternType query.SearchType, missingRepoRevs []*search.RepositoryRevisions) *searchAlert {
	var description string
	if len(missingRepoRevs) == 1 {
		if len(missingRepoRevs[0].RevSpecs()) == 1 {
			description = fmt.Sprintf("The repository %s matched by your repo: filter could not be searched because it does not contain the revision %q.", missingRepoRevs[0].Repo.Name, missingRepoRevs[0].RevSpecs()[0])
		} else {
			description = fmt.Sprintf("The repository %s matched by your repo: filter could not be searched because it has multiple specified revisions: @%s.", missingRepoRevs[0].Repo.Name, strings.Join(missingRepoRevs[0].RevSpecs(), ","))
		}
	} else {
		repoRevs := make([]string, 0, len(missingRepoRevs))
		for _, r := range missingRepoRevs {
			repoRevs = append(repoRevs, string(r.Repo.Name)+"@"+strings.Join(r.RevSpecs(), ","))
		}
		description = fmt.Sprintf("%d repositories matched by your repo: filter could not be searched because the following revisions do not exist, or differ but were specified for the same repository: %s.", len(missingRepoRevs), strings.Join(repoRevs, ", "))
	}
	return &searchAlert{
		prometheusType: "missing_repo_revs",
		title:          "Some repositories could not be searched",
		description:    description,
	}
}

// pathParentsByFrequency returns the most common path parents of the given paths.
// For example, given paths [a/b a/c x/y], it would return [a x] because "a"
// is a parent to 2 paths and "x" is a parent to 1 path.
func pathParentsByFrequency(paths []string) []string {
	var parents []string
	parentFreq := map[string]int{}
	for _, p := range paths {
		parent := path.Dir(p)
		if _, seen := parentFreq[parent]; !seen {
			parents = append(parents, parent)
		}
		parentFreq[parent]++
	}

	sort.Slice(parents, func(i, j int) bool {
		pi, pj := parents[i], parents[j]
		fi, fj := parentFreq[pi], parentFreq[pj]
		return fi > fj || (fi == fj && pi < pj) // freq desc, alpha asc
	})
	return parents
}

// Wrap an alert in a SearchResultsResolver. ElapsedMilliseconds() will
// calculate a very large value for duration if start takes on the nil-value of
// year 1. As a workaround, wrap instantiates start with time.now().
// TODO(rvantonder): #10801.
func (a searchAlert) wrap() *SearchResultsResolver {
	return &SearchResultsResolver{alert: &a, start: time.Now()}
}

// capFirst capitalizes the first rune in the given string. It can be safely
// used with UTF-8 strings.
func capFirst(s string) string {
	i := 0
	return strings.Map(func(r rune) rune {
		i++
		if i == 1 {
			return unicode.ToTitle(r)
		}
		return r
	}, s)
}

func (a searchAlert) Results(context.Context) (*SearchResultsResolver, error) {
	alert := &searchAlert{
		prometheusType:  a.prometheusType,
		title:           a.title,
		description:     a.description,
		proposedQueries: a.proposedQueries,
	}
	return alert.wrap(), nil
}

func (searchAlert) Suggestions(context.Context, *searchSuggestionsArgs) ([]*searchSuggestionResolver, error) {
	return nil, nil
}
func (searchAlert) Stats(context.Context) (*searchResultsStats, error) { return nil, nil }

func alertForError(err error, inputs *SearchInputs) *searchAlert {
	var (
		alert *searchAlert
		rErr  *RepoLimitError
		tErr  *TimeLimitError
		mErr  *searchrepos.MissingRepoRevsError
	)

	if false {
	} else if errors.As(err, &mErr) {
		alert = alertForMissingRepoRevs(inputs.PatternType, mErr.Missing)
		alert.priority = 6
	} else if strings.Contains(err.Error(), "Worker_oomed") || strings.Contains(err.Error(), "Worker_exited_abnormally") {
		alert = &searchAlert{
			prometheusType: "structural_search_needs_more_memory",
			title:          "Structural search needs more memory",
			description:    "Running your structural search may require more memory. If you are running the query on many repositories, try reducing the number of repositories with the `repo:` filter.",
		}
		alert.priority = 5
	} else if strings.Contains(err.Error(), "Out of memory") {
		a := searchAlert{
			prometheusType: "structural_search_needs_more_memory__give_searcher_more_memory",
			title:          "Structural search needs more memory",
			description:    `Running your structural search requires more memory. You could try reducing the number of repositories with the "repo:" filter. If you are an administrator, try double the memory allocated for the "searcher" service. If you're unsure, reach out to us at support@sourcegraph.com.`,
		}
		a.priority = 4
	} else if strings.Contains(err.Error(), "no indexed repositories for structural search") {
		var msg string
		if envvar.SourcegraphDotComMode() {
			msg = "The good news is you can index any repository you like in a self-install. It takes less than 5 minutes to set up: https://docs.sourcegraph.com/#quickstart"
		} else {
			msg = "Learn more about managing indexed repositories in our documentation: https://docs.sourcegraph.com/admin/search#indexed-search."
		}
		alert = &searchAlert{
			prometheusType: "structural_search_on_zero_indexed_repos",
			title:          "Unindexed repositories or repository revisions with structural search",
			description:    fmt.Sprintf("Structural search currently only works on indexed repositories or revisions. Some of the repositories or revisions to search are not indexed, so we can't return results for them. %s", msg),
		}
		alert.priority = 3
	} else if errors.As(err, &rErr) {
		alert = &searchAlert{
			prometheusType: "exceeded_diff_commit_search_limit",
			title:          fmt.Sprintf("Too many matching repositories for %s search to handle", rErr.ResultType),
			description:    fmt.Sprintf(`%s search can currently only handle searching over %d repositories at a time. Try using the "repo:" filter to narrow down which repositories to search, or using 'after:"1 week ago"'. Tracking issue: https://github.com/sourcegraph/sourcegraph/issues/6826`, strings.Title(rErr.ResultType), rErr.Max),
		}
		alert.priority = 2
	} else if errors.As(err, &tErr) {
		alert = &searchAlert{
			prometheusType: "exceeded_diff_commit_with_time_search_limit",
			title:          fmt.Sprintf("Too many matching repositories for %s search to handle", tErr.ResultType),
			description:    fmt.Sprintf(`%s search can currently only handle searching over %d repositories at a time. Try using the "repo:" filter to narrow down which repositories to search. Tracking issue: https://github.com/sourcegraph/sourcegraph/issues/6826`, strings.Title(tErr.ResultType), tErr.Max),
		}
		alert.priority = 1
	}
	return alert
}

type AlertObserver struct {
	// alert is the current alert to show.
	alert      *searchAlert
	err        error
	hasResults bool
}

// Next returns a non-nil alert if there is a new alert to show to the user.
func (o *AlertObserver) Next(event SearchEvent, inputs *SearchInputs) *searchAlert {
	if len(event.Results) > 0 {
		o.hasResults = true
	}

	if event.Error == nil {
		return nil
	}

	alert := alertForError(event.Error, inputs)
	if alert == nil {
		o.err = multierror.Append(o.err, event.Error)
		return nil
	}
	return o.update(alert)
}

func (o *AlertObserver) update(alert *searchAlert) *searchAlert {
	if o.alert == nil || alert.priority > o.alert.priority {
		o.alert = alert
		return o.alert
	}
	return nil
}

//  Done returns the highest priority alert and a multierror.Error containing all
//  errors that could not be converted to alerts.
func (o *AlertObserver) Done(stats *streaming.Stats, s *SearchInputs) (*searchAlert, error) {
	if !o.hasResults && s.PatternType != query.SearchTypeStructural && comby.MatchHoleRegexp.MatchString(s.OriginalQuery) {
		o.update(alertForStructuralSearchNotSet(s.OriginalQuery))
	}

	if o.hasResults && o.err != nil {
		log15.Error("Errors during search", "error", o.err)
		return o.alert, nil
	}

	return o.alert, o.err
}

func (searchAlert) SetStream(c SearchStream) {}
func (searchAlert) Inputs() *SearchInputs {
	return nil
}
