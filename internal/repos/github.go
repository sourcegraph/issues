package repos

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/inconshreveable/log15"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/envvar"
	"github.com/sourcegraph/sourcegraph/internal/conf/reposource"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/auth"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/github"
	"github.com/sourcegraph/sourcegraph/internal/httpcli"
	"github.com/sourcegraph/sourcegraph/internal/jsonc"
	"github.com/sourcegraph/sourcegraph/internal/lazyregexp"
	"github.com/sourcegraph/sourcegraph/internal/ratelimit"
	"github.com/sourcegraph/sourcegraph/internal/types"
	"github.com/sourcegraph/sourcegraph/schema"
)

// A GithubSource yields repositories from a single Github connection configured
// in Sourcegraph via the external services configuration.
type GithubSource struct {
	svc             *types.ExternalService
	config          *schema.GitHubConnection
	exclude         excludeFunc
	excludeArchived bool
	excludeForks    bool
	githubDotCom    bool
	baseURL         *url.URL
	v3Client        *github.V3Client
	v4Client        *github.V4Client
	// searchClient is for using the GitHub search API, which has an independent
	// rate limit much lower than non-search API requests.
	searchClient *github.V3Client

	// originalHostname is the hostname of config.Url (differs from client APIURL, whose host is api.github.com
	// for an originalHostname of github.com).
	originalHostname string
}

var (
	_ Source                     = &GithubSource{}
	_ UserSource                 = &GithubSource{}
	_ AffiliatedRepositorySource = &GithubSource{}
	_ VersionSource              = &GithubSource{}
)

// NewGithubSource returns a new GithubSource from the given external service.
func NewGithubSource(svc *types.ExternalService, cf *httpcli.Factory) (*GithubSource, error) {
	var c schema.GitHubConnection
	if err := jsonc.Unmarshal(svc.Config, &c); err != nil {
		return nil, errors.Errorf("external service id=%d config error: %s", svc.ID, err)
	}
	return newGithubSource(svc, &c, cf)
}

var githubRemainingGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
	// _v2 since we have an older metric defined in github-proxy
	Name: "src_github_rate_limit_remaining_v2",
	Help: "Number of calls to GitHub's API remaining before hitting the rate limit.",
}, []string{"resource", "name"})

var githubRatelimitWaitCounter = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "src_github_rate_limit_wait_duration_seconds",
	Help: "The amount of time spent waiting on the rate limit",
}, []string{"resource", "name"})

func newGithubSource(svc *types.ExternalService, c *schema.GitHubConnection, cf *httpcli.Factory) (*GithubSource, error) {
	baseURL, err := url.Parse(c.Url)
	if err != nil {
		return nil, err
	}
	baseURL = extsvc.NormalizeBaseURL(baseURL)
	originalHostname := baseURL.Hostname()

	apiURL, githubDotCom := github.APIRoot(baseURL)

	if cf == nil {
		cf = httpcli.NewExternalHTTPClientFactory()
	}

	opts := []httpcli.Opt{
		// Use a 30s timeout to avoid running into EOF errors, because GitHub
		// closes idle connections after 60s
		httpcli.NewIdleConnTimeoutOpt(30 * time.Second),
	}

	if c.Certificate != "" {
		opts = append(opts, httpcli.NewCertPoolOpt(c.Certificate))
	}

	cli, err := cf.Doer(opts...)
	if err != nil {
		return nil, err
	}

	var (
		eb              excludeBuilder
		excludeArchived bool
		excludeForks    bool
	)

	for _, r := range c.Exclude {
		eb.Exact(r.Name)
		eb.Exact(r.Id)
		eb.Pattern(r.Pattern)

		if r.Archived {
			excludeArchived = true
		}

		if r.Forks {
			excludeForks = true
		}
	}

	exclude, err := eb.Build()
	if err != nil {
		return nil, err
	}
	token := &auth.OAuthBearerToken{Token: c.Token}

	var (
		v3Client     = github.NewV3Client(apiURL, token, cli)
		v4Client     = github.NewV4Client(apiURL, token, cli)
		searchClient = github.NewV3SearchClient(apiURL, token, cli)
	)

	if !envvar.SourcegraphDotComMode() || svc.CloudDefault {
		for resource, monitor := range map[string]*ratelimit.Monitor{
			"rest":    v3Client.RateLimitMonitor(),
			"graphql": v4Client.RateLimitMonitor(),
			"search":  searchClient.RateLimitMonitor(),
		} {
			// Need to copy the resource or func will use the last one seen while iterating
			// the map
			resource := resource
			monitor.SetCollector(&ratelimit.MetricsCollector{
				Remaining: func(n float64) {
					githubRemainingGauge.WithLabelValues(resource, svc.DisplayName).Set(n)
				},
				WaitDuration: func(n time.Duration) {
					githubRatelimitWaitCounter.WithLabelValues(resource, svc.DisplayName).Add(n.Seconds())
				},
			})
		}
	}

	return &GithubSource{
		svc:              svc,
		config:           c,
		exclude:          exclude,
		excludeArchived:  excludeArchived,
		excludeForks:     excludeForks,
		baseURL:          baseURL,
		githubDotCom:     githubDotCom,
		v3Client:         v3Client,
		v4Client:         v4Client,
		searchClient:     searchClient,
		originalHostname: originalHostname,
	}, nil
}

func (s GithubSource) WithAuthenticator(a auth.Authenticator) (Source, error) {
	switch a.(type) {
	case *auth.OAuthBearerToken,
		*auth.OAuthBearerTokenWithSSH:
		break

	default:
		return nil, newUnsupportedAuthenticatorError("GithubSource", a)
	}

	sc := s
	sc.v3Client = sc.v3Client.WithAuthenticator(a)
	sc.v4Client = sc.v4Client.WithAuthenticator(a)
	sc.searchClient = sc.searchClient.WithAuthenticator(a)

	return &sc, nil
}

type githubResult struct {
	err  error
	repo *github.Repository
}

func (s GithubSource) ValidateAuthenticator(ctx context.Context) error {
	_, err := s.v3Client.GetAuthenticatedUser(ctx)
	return err
}

func (s GithubSource) Version(ctx context.Context) (string, error) {
	return s.v3Client.GetVersion(ctx)
}

// ListRepos returns all Github repositories accessible to all connections configured
// in Sourcegraph via the external services configuration.
func (s GithubSource) ListRepos(ctx context.Context, results chan SourceResult) {
	unfiltered := make(chan *githubResult)
	go func() {
		s.listAllRepositories(ctx, unfiltered)
		close(unfiltered)
	}()

	seen := make(map[int64]bool)
	for res := range unfiltered {
		if res.err != nil {
			results <- SourceResult{Source: s, Err: res.err}
			continue
		}
		if !seen[res.repo.DatabaseID] && !s.excludes(res.repo) {
			results <- SourceResult{Source: s, Repo: s.makeRepo(res.repo)}
			seen[res.repo.DatabaseID] = true
		}
	}
}

// ExternalServices returns a singleton slice containing the external service.
func (s GithubSource) ExternalServices() types.ExternalServices {
	return types.ExternalServices{s.svc}
}

// GetRepo returns the Github repository with the given name and owner
// ("org/repo-name")
func (s GithubSource) GetRepo(ctx context.Context, nameWithOwner string) (*types.Repo, error) {
	r, err := s.getRepository(ctx, nameWithOwner)
	if err != nil {
		return nil, err
	}
	return s.makeRepo(r), nil
}

func (s GithubSource) makeRepo(r *github.Repository) *types.Repo {
	urn := s.svc.URN()
	metadata := *r
	// This field flip flops depending on which token was used to retrieve the repo
	// so we don't want to store it.
	metadata.ViewerPermission = ""
	return &types.Repo{
		Name: reposource.GitHubRepoName(
			s.config.RepositoryPathPattern,
			s.originalHostname,
			r.NameWithOwner,
		),
		URI: string(reposource.GitHubRepoName(
			"",
			s.originalHostname,
			r.NameWithOwner,
		)),
		ExternalRepo: github.ExternalRepoSpec(r, s.baseURL),
		Description:  r.Description,
		Fork:         r.IsFork,
		Archived:     r.IsArchived,
		Stars:        r.StargazerCount,
		Private:      r.IsPrivate,
		Sources: map[string]*types.SourceInfo{
			urn: {
				ID:       urn,
				CloneURL: s.remoteURL(r),
			},
		},
		Metadata: &metadata,
	}
}

// remoteURL returns the repository's Git remote URL
//
// note: this used to contain credentials but that is no longer the case
// if you need to get an authenticated clone url use repos.CloneURL
func (s *GithubSource) remoteURL(repo *github.Repository) string {
	if s.config.GitURLType == "ssh" {
		url := fmt.Sprintf("git@%s:%s.git", s.originalHostname, repo.NameWithOwner)
		return url
	}

	return repo.URL
}

func (s *GithubSource) excludes(r *github.Repository) bool {
	if r.IsLocked || r.IsDisabled {
		return true
	}

	if s.exclude(r.NameWithOwner) || s.exclude(r.ID) {
		return true
	}

	if s.excludeArchived && r.IsArchived {
		return true
	}

	if s.excludeForks && r.IsFork {
		return true
	}

	return false
}

// repositoryPager is a function that returns repositories on a given `page`.
// It also returns:
// - `hasNext` bool: if there is a next page
// - `cost` int: rate limit cost used to determine recommended wait before next call
// - `err` error: if something goes wrong
type repositoryPager func(page int) (repos []*github.Repository, hasNext bool, cost int, err error)

// paginate returns all the repositories from the given repositoryPager.
// It repeatedly calls `pager` with incrementing page count until it
// returns false for hasNext.
func (s *GithubSource) paginate(ctx context.Context, results chan *githubResult, pager repositoryPager) {
	hasNext := true
	for page := 1; hasNext; page++ {
		if err := ctx.Err(); err != nil {
			results <- &githubResult{err: err}
			return
		}

		var pageRepos []*github.Repository
		var cost int
		var err error
		pageRepos, hasNext, cost, err = pager(page)
		if err != nil {
			results <- &githubResult{err: err}
			return
		}

		for _, r := range pageRepos {
			results <- &githubResult{repo: r}
		}

		if hasNext && cost > 0 {
			time.Sleep(s.v3Client.RateLimitMonitor().RecommendedWaitForBackgroundOp(cost))
		}
	}
}

// listOrg handles the `org` config option.
// It returns all the repositories belonging to the given organization
// by hitting the /orgs/:org/repos endpoint.
//
// It returns an error if the request fails on the first page.
func (s *GithubSource) listOrg(ctx context.Context, org string, results chan *githubResult) {
	dedupC := make(chan *githubResult)

	// Currently, the Github API doesn't return internal repos
	// when calling it with the "all" type.
	// We need to call it twice, once with the "all" type and
	// once with the "internal" type.
	// However, since we don't have any guarantee that this behavior
	// will always remain the same and that Github will never fix this issue,
	// we need to deduplicate the results before sending them to the results channel.

	getReposByType := func(tp string) error {
		var oerr error

		s.paginate(ctx, dedupC, func(page int) (repos []*github.Repository, hasNext bool, cost int, err error) {
			defer func() {
				if page == 1 {
					var e *github.APIError
					if errors.As(err, &e) && e.Code == 404 {
						oerr = errors.Errorf("organisation %q not found", org)
						err = nil
					}
				}

				remaining, reset, retry, _ := s.v3Client.RateLimitMonitor().Get()
				log15.Debug(
					"github sync: ListOrgRepositories",
					"repos", len(repos),
					"rateLimitCost", cost,
					"rateLimitRemaining", remaining,
					"rateLimitReset", reset,
					"retryAfter", retry,
					"type", tp,
				)
			}()

			return s.v3Client.ListOrgRepositories(ctx, org, page, tp)
		})

		return oerr
	}

	go func() {
		defer close(dedupC)

		err := getReposByType("all")
		// Handle 404 from org repos endpoint by trying user repos endpoint
		if err != nil {
			if s.listUser(ctx, org, dedupC) != nil {
				dedupC <- &githubResult{
					err: err,
				}
			}
			return
		}

		// if the first call succeeded,
		// call the same endpoint with the "internal" type
		if err = getReposByType("internal"); err != nil {
			dedupC <- &githubResult{
				err: err,
			}
		}
	}()

	seen := make(map[string]bool)

	for res := range dedupC {
		if res.err == nil {
			if seen[res.repo.ID] {
				continue
			}

			seen[res.repo.ID] = true
		}

		results <- res
	}
}

// listUser returns all the repositories belonging to the given user
// by hitting the /users/:user/repos endpoint.
//
// It returns an error if the request fails on the first page.
func (s *GithubSource) listUser(ctx context.Context, user string, results chan *githubResult) (fail error) {
	s.paginate(ctx, results, func(page int) (repos []*github.Repository, hasNext bool, cost int, err error) {
		defer func() {
			if err != nil && page == 1 {
				fail, err = err, nil
			}

			remaining, reset, retry, _ := s.v3Client.RateLimitMonitor().Get()
			log15.Debug(
				"github sync: ListUserRepositories",
				"repos", len(repos),
				"rateLimitCost", cost,
				"rateLimitRemaining", remaining,
				"rateLimitReset", reset,
				"retryAfter", retry,
			)
		}()
		return s.v3Client.ListUserRepositories(ctx, user, page)
	})
	return
}

// listRepos returns the valid repositories from the given list of repository names.
// This is done by hitting the /repos/:owner/:name endpoint for each of the given
// repository names.
func (s *GithubSource) listRepos(ctx context.Context, repos []string, results chan *githubResult) {
	if err := s.fetchAllRepositoriesInBatches(ctx, results); err == nil {
		return
	} else {
		// The way we fetch repositories in batches through the GraphQL API -
		// using aliases to query multiple repositories in one query - is
		// currently "undefined behaviour". Very rarely but unreproducibly it
		// resulted in EOF errors while testing. And since we rely on fetching
		// to work, we fall back to the (slower) sequential fetching in case we
		// run into an GraphQL API error
		log15.Warn("github sync: fetching in batches failed. falling back to sequential fetch", "error", err)
	}

	// Admins normally add to end of lists, so end of list most likely has new
	// repos => stream them first.
	for i := len(repos) - 1; i >= 0; i-- {
		nameWithOwner := repos[i]
		if err := ctx.Err(); err != nil {
			results <- &githubResult{err: err}
			return
		}

		owner, name, err := github.SplitRepositoryNameWithOwner(nameWithOwner)
		if err != nil {
			results <- &githubResult{err: errors.New("Invalid GitHub repository: nameWithOwner=" + nameWithOwner)}
			return
		}
		var repo *github.Repository
		repo, err = s.v3Client.GetRepository(ctx, owner, name)
		if err != nil {
			// TODO(tsenart): When implementing dry-run, reconsider alternatives to return
			// 404 errors on external service config validation.
			if github.IsNotFound(err) {
				log15.Warn("skipping missing github.repos entry:", "name", nameWithOwner, "err", err)
				continue
			}

			results <- &githubResult{err: errors.Wrapf(err, "Error getting GitHub repository: nameWithOwner=%s", nameWithOwner)}
			break
		}
		log15.Debug("github sync: GetRepository", "repo", repo.NameWithOwner)

		results <- &githubResult{repo: repo}

		time.Sleep(s.v3Client.RateLimitMonitor().RecommendedWaitForBackgroundOp(1)) // 0-duration sleep unless nearing rate limit exhaustion
	}
}

// listPublic handles the `public` keyword of the `repositoryQuery` config option.
// It returns the public repositories listed on the /repositories endpoint.
func (s *GithubSource) listPublic(ctx context.Context, results chan *githubResult) {
	if s.githubDotCom {
		results <- &githubResult{err: errors.New(`unsupported configuration "public" for "repositoryQuery" for github.com`)}
		return
	}
	var sinceRepoID int64
	for {
		if err := ctx.Err(); err != nil {
			results <- &githubResult{err: err}
			return
		}

		repos, err := s.v3Client.ListPublicRepositories(ctx, sinceRepoID)
		if err != nil {
			results <- &githubResult{err: errors.Wrapf(err, "failed to list public repositories: sinceRepoID=%d", sinceRepoID)}
			return
		}
		if len(repos) == 0 {
			return
		}
		log15.Debug("github sync public", "repos", len(repos), "error", err)
		for _, r := range repos {
			results <- &githubResult{repo: r}
			if sinceRepoID < r.DatabaseID {
				sinceRepoID = r.DatabaseID
			}
		}
	}
}

// listAffiliated handles the `affiliated` keyword of the `repositoryQuery` config option.
// It returns the repositories affiliated with the client token by hitting the /user/repos
// endpoint.
//
// Affiliation is present if the user: (1) owns the repo, (2) is apart of an org that
// the repo belongs to, or (3) is a collaborator.
func (s *GithubSource) listAffiliated(ctx context.Context, results chan *githubResult) {
	s.paginate(ctx, results, func(page int) (repos []*github.Repository, hasNext bool, cost int, err error) {
		defer func() {
			remaining, reset, retry, _ := s.v3Client.RateLimitMonitor().Get()
			log15.Debug(
				"github sync: ListAffiliated",
				"repos", len(repos),
				"rateLimitCost", cost,
				"rateLimitRemaining", remaining,
				"rateLimitReset", reset,
				"retryAfter", retry,
			)
		}()
		return s.v3Client.ListAffiliatedRepositories(ctx, github.VisibilityAll, page)
	})
}

// listSearch handles the `repositoryQuery` config option when a keyword is not present.
// It returns the repositories resulting from from GitHub's advanced repository search
// by hitting the /search/repositories endpoint.
func (s *GithubSource) listSearch(ctx context.Context, query string, results chan *githubResult) {
	s.paginate(ctx, results, func(page int) ([]*github.Repository, bool, int, error) {
		reposPage, err := s.searchClient.ListRepositoriesForSearch(ctx, query, page)
		if err != nil {
			return nil, false, 0, errors.Wrapf(err, "failed to list GitHub repositories for search: page=%d, searchString=%q", page, query)
		}

		if reposPage.TotalCount > 1000 {
			// GitHub's advanced repository search will only
			// return 1000 results. We specially handle this case
			// to ensure the admin gets a detailed error
			// message. https://github.com/sourcegraph/sourcegraph/issues/2562
			return nil, false, 0, errors.Errorf(`repositoryQuery %q would return %d results. GitHub's Search API only returns up to 1000 results. Please adjust your repository query into multiple queries such that each returns less than 1000 results. For example: {"repositoryQuery": %s}`, query, reposPage.TotalCount, exampleRepositoryQuerySplit(query))
		}

		repos, hasNext := reposPage.Repos, reposPage.HasNextPage
		remaining, reset, retry, ok := s.searchClient.RateLimitMonitor().Get()
		log15.Debug(
			"github sync: ListRepositoriesForSearch",
			"searchString", query,
			"repos", len(repos),
			"rateLimitRemaining", remaining,
			"rateLimitReset", reset,
			"retryAfter", retry,
		)

		// GitHub search has vastly different rate limits to
		// the normal GitHub API (30req/m vs
		// 5000req/h). RecommendedWaitForBackgroundOp has
		// heuristics tuned for the normal API, part of which
		// is to not sleep if we have ample rate limit left.
		//
		// So we only let the heuristic kick in if we have
		// less than 5 requests left.
		var cost int
		if retry > 0 || (ok && remaining < 5) {
			cost = 1
		}

		return repos, hasNext, cost, nil
	})
}

// regOrg is a regular expression that matches the pattern `org:<org-name>`
// `<org-name>` follows the GitHub username convention:
// - only single hyphens and alphanumeric characters allowed.
// - cannot begin/end with hyphen.
// - up to 38 characters.
var regOrg = lazyregexp.New(`^org:([a-zA-Z0-9](?:-?[a-zA-Z0-9]){0,38})$`)

// matchOrg extracts the org name from the pattern `org:<org-name>` if it exists.
func matchOrg(q string) string {
	match := regOrg.FindStringSubmatch(q)
	if len(match) != 2 {
		return ""
	}
	return match[1]
}

// listRepositoryQuery handles the `repositoryQuery` config option.
// The supported keywords to select repositories are:
// - `public`: public repositories (from endpoint: /repositories)
// - `affiliated`: repositories affiliated with client token (from endpoint: /user/repos)
// - `none`: disables `repositoryQuery`
// Inputs other than these three keywords will be queried using
// GitHub advanced repository search (endpoint: /search/repositories)
func (s *GithubSource) listRepositoryQuery(ctx context.Context, query string, results chan *githubResult) {
	switch query {
	case "public":
		s.listPublic(ctx, results)
		return
	case "affiliated":
		s.listAffiliated(ctx, results)
		return
	case "none":
		// nothing
		return
	}

	// Special-casing for `org:<org-name>`
	// to directly use GitHub's org repo
	// list API instead of the limited
	// search API.
	//
	// If the org repo list API fails, we
	// try the user repo list API.
	if org := matchOrg(query); org != "" {
		s.listOrg(ctx, org, results)
		return
	}

	// Run the query as a GitHub advanced repository search
	// (https://github.com/search/advanced).
	s.listSearch(ctx, query, results)
}

// listAllRepositories returns the repositories from the given `orgs`, `repos`, and
// `repositoryQuery` config options excluding the ones specified by `exclude`.
func (s *GithubSource) listAllRepositories(ctx context.Context, results chan *githubResult) {
	s.listRepos(ctx, s.config.Repos, results)

	// Admins normally add to end of lists, so end of list most likely has new
	// repos => stream them first.
	for i := len(s.config.RepositoryQuery) - 1; i >= 0; i-- {
		s.listRepositoryQuery(ctx, s.config.RepositoryQuery[i], results)
	}

	for i := len(s.config.Orgs) - 1; i >= 0; i-- {
		s.listOrg(ctx, s.config.Orgs[i], results)
	}
}

func (s *GithubSource) getRepository(ctx context.Context, nameWithOwner string) (*github.Repository, error) {
	owner, name, err := github.SplitRepositoryNameWithOwner(nameWithOwner)
	if err != nil {
		return nil, errors.Wrapf(err, "Invalid GitHub repository: nameWithOwner="+nameWithOwner)
	}

	repo, err := s.v3Client.GetRepository(ctx, owner, name)
	if err != nil {
		return nil, err
	}

	return repo, nil
}

// fetchAllRepositoriesInBatches fetches the repositories configured in
// config.Repos in batches and adds them to the supplied set
func (s *GithubSource) fetchAllRepositoriesInBatches(ctx context.Context, results chan *githubResult) error {
	const batchSize = 30

	// Admins normally add to end of lists, so end of list most likely has new
	// repos => stream them first.
	for end := len(s.config.Repos); end > 0; end -= batchSize {
		if err := ctx.Err(); err != nil {
			return err
		}

		start := end - batchSize
		if start < 0 {
			start = 0
		}
		batch := s.config.Repos[start:end]

		repos, err := s.v4Client.GetReposByNameWithOwner(ctx, batch...)
		if err != nil {
			return err
		}

		log15.Debug("github sync: GetReposByNameWithOwner", "repos", batch)
		for _, r := range repos {
			results <- &githubResult{repo: r}
		}
	}

	return nil
}

func exampleRepositoryQuerySplit(q string) string {
	var qs []string
	for _, suffix := range []string{"created:>=2019", "created:2018", "created:2016..2017", "created:<2016"} {
		qs = append(qs, fmt.Sprintf("%s %s", q, suffix))
	}
	// Avoid escaping < and >
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(qs)
	return strings.TrimSpace(b.String())
}

func (s *GithubSource) AffiliatedRepositories(ctx context.Context) ([]types.CodeHostRepository, error) {
	var (
		repos []*github.Repository
		page  = 1
		cost  int
		err   error
	)
	defer func() {
		remaining, reset, retry, _ := s.v3Client.RateLimitMonitor().Get()
		log15.Debug(
			"github sync: ListAffiliated",
			"repos", len(repos),
			"rateLimitCost", cost,
			"rateLimitRemaining", remaining,
			"rateLimitReset", reset,
			"retryAfter", retry,
		)
	}()
	out := make([]types.CodeHostRepository, 0)
	hasNextPage := true
	for hasNextPage {
		select {
		case <-ctx.Done():
			return nil, errors.Errorf("context canceled")
		default:
		}
		repos, hasNextPage, _, err = s.v3Client.ListAffiliatedRepositories(ctx, github.VisibilityAll, page)
		if err != nil {
			return nil, err
		}
		for _, repo := range repos {
			out = append(out, types.CodeHostRepository{
				Name:       repo.NameWithOwner,
				Private:    repo.IsPrivate,
				CodeHostID: s.svc.ID,
			})
		}
		page++
	}
	return out, nil
}
