package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"golang.org/x/time/rate"

	"github.com/sourcegraph/sourcegraph/internal/extsvc/auth"
	"github.com/sourcegraph/sourcegraph/internal/httpcli"
	"github.com/sourcegraph/sourcegraph/internal/ratelimit"
	"github.com/sourcegraph/sourcegraph/internal/rcache"
)

// V3Client is a caching GitHub API client for GitHub's REST API v3.
//
// All instances use a map of rcache.Cache instances for caching (see the `repoCache` field). These
// separate instances have consistent naming prefixes so that different instances will share the
// same Redis cache entries (provided they were computed with the same API URL and access
// token). The cache keys are agnostic of the http.RoundTripper transport.
type V3Client struct {
	// apiURL is the base URL of a GitHub API. It must point to the base URL of the GitHub API. This
	// is https://api.github.com for GitHub.com and http[s]://[github-enterprise-hostname]/api for
	// GitHub Enterprise.
	apiURL *url.URL

	// githubDotCom is true if this client connects to github.com.
	githubDotCom bool

	// auth is used to authenticate requests. May be empty, in which case the
	// default behavior is to make unauthenticated requests.
	// 🚨 SECURITY: Should not be changed after client creation to prevent
	// unauthorized access to the repository cache. Use `WithAuthenticator` to
	// create a new client with a different authenticator instead.
	auth auth.Authenticator

	// httpClient is the HTTP client used to make requests to the GitHub API.
	httpClient httpcli.Doer

	// repoCache is the repository cache associated with the token.
	repoCache *rcache.Cache

	// rateLimitMonitor is the API rate limit monitor.
	rateLimitMonitor *ratelimit.Monitor

	// rateLimit is our self imposed rate limiter
	rateLimit *rate.Limiter

	// resource specifies which API this client is intended for.
	// One of 'rest' or 'search'.
	resource string
}

// NewV3Client creates a new GitHub API client with an optional default
// authenticator.
//
// apiURL must point to the base URL of the GitHub API. See the docstring for
// V3Client.apiURL.
func NewV3Client(apiURL *url.URL, a auth.Authenticator, cli httpcli.Doer) *V3Client {
	return newV3Client(apiURL, a, "rest", cli)
}

// NewV3SearchClient creates a new GitHub API client intended for use with the
// search API with an optional default authenticator.
//
// apiURL must point to the base URL of the GitHub API. See the docstring for
// V3Client.apiURL.
func NewV3SearchClient(apiURL *url.URL, a auth.Authenticator, cli httpcli.Doer) *V3Client {
	return newV3Client(apiURL, a, "search", cli)
}

func newV3Client(apiURL *url.URL, a auth.Authenticator, resource string, cli httpcli.Doer) *V3Client {
	apiURL = canonicalizedURL(apiURL)
	if gitHubDisable {
		cli = disabledClient{}
	}
	if cli == nil {
		cli = httpcli.ExternalDoer()
	}

	cli = requestCounter.Doer(cli, func(u *url.URL) string {
		// The first component of the Path mostly maps to the type of API
		// request we are making. See `curl https://api.github.com` for the
		// exact mapping
		var category string
		if parts := strings.SplitN(u.Path, "/", 3); len(parts) > 1 {
			category = parts[1]
		}
		return category
	})

	var tokenHash string
	if a != nil {
		tokenHash = a.Hash()
	}

	rl := ratelimit.DefaultRegistry.Get(apiURL.String())
	rlm := ratelimit.DefaultMonitorRegistry.GetOrSet(apiURL.String(), tokenHash, resource, &ratelimit.Monitor{HeaderPrefix: "X-"})

	return &V3Client{
		apiURL:           apiURL,
		githubDotCom:     urlIsGitHubDotCom(apiURL),
		auth:             a,
		httpClient:       cli,
		rateLimit:        rl,
		rateLimitMonitor: rlm,
		repoCache:        newRepoCache(apiURL, a),
		resource:         resource,
	}
}

// WithAuthenticator returns a new V3Client that uses the same configuration as
// the current V3Client, except authenticated as the GitHub user with the given
// authenticator instance (most likely a token).
func (c *V3Client) WithAuthenticator(a auth.Authenticator) *V3Client {
	return newV3Client(c.apiURL, a, c.resource, c.httpClient)
}

// RateLimitMonitor exposes the rate limit monitor.
func (c *V3Client) RateLimitMonitor() *ratelimit.Monitor {
	return c.rateLimitMonitor
}

func (c *V3Client) requestGet(ctx context.Context, requestURI string, result interface{}) error {
	_, err := c.get(ctx, requestURI, result)
	return err
}

func (c *V3Client) requestGetWithHeader(ctx context.Context, requestURI string, result interface{}) (http.Header, error) {
	return c.get(ctx, requestURI, result)
}

func (c *V3Client) get(ctx context.Context, requestURI string, result interface{}) (http.Header, error) {
	req, err := http.NewRequest("GET", requestURI, nil)
	if err != nil {
		return nil, err
	}

	// Include node_id (GraphQL ID) in response. See
	// https://developer.github.com/changes/2017-12-19-graphql-node-id/.
	//
	// Enable the repository topics API. See
	// https://developer.github.com/v3/repos/#list-all-topics-for-a-repository
	req.Header.Add("Accept", "application/vnd.github.jean-grey-preview+json,application/vnd.github.mercy-preview+json")

	// Enable the GitHub App API. See
	// https://developer.github.com/v3/apps/installations/#list-repositories
	req.Header.Add("Accept", "application/vnd.github.machine-man-preview+json")

	err = c.rateLimit.Wait(ctx)
	if err != nil {
		return nil, errInternalRateLimitExceeded
	}

	return doRequest(ctx, c.apiURL, c.auth, c.rateLimitMonitor, c.httpClient, req, result)
}

// newRepoCache creates a new cache for GitHub repository metadata. The backing
// store is Redis. A checksum of the authenticator and API URL are used as a
// Redis key prefix to prevent collisions with caches for different
// authentication and API URLs.
func newRepoCache(apiURL *url.URL, a auth.Authenticator) *rcache.Cache {
	var cacheTTL time.Duration
	if urlIsGitHubDotCom(apiURL) {
		cacheTTL = 10 * time.Minute
	} else {
		// GitHub Enterprise
		cacheTTL = 30 * time.Second
	}

	key := ""
	if a != nil {
		key = a.Hash()
	}
	return rcache.NewWithTTL("gh_repo:"+key, int(cacheTTL/time.Second))
}

// APIError is an error type returned by Client when the GitHub API responds with
// an error.
type APIError struct {
	URL              string
	Code             int
	Message          string
	DocumentationURL string `json:"documentation_url"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("request to %s returned status %d: %s", e.URL, e.Code, e.Message)
}

func (e *APIError) Unauthorized() bool {
	return e.Code == http.StatusUnauthorized
}

func (e *APIError) AccountSuspended() bool {
	return e.Code == http.StatusForbidden && strings.Contains(e.Message, "account was suspended")
}

// HTTPErrorCode returns err's HTTP status code, if it is an HTTP error from
// this package. Otherwise it returns 0.
func HTTPErrorCode(err error) int {
	var e *APIError
	if errors.As(err, &e) {
		return e.Code
	}

	return 0
}

func (c *V3Client) GetVersion(ctx context.Context) (string, error) {
	if c.githubDotCom {
		return "unknown", nil
	}

	var empty interface{}

	header, err := c.requestGetWithHeader(ctx, "/", &empty)
	if err != nil {
		return "", err
	}
	v := header.Get("X-GitHub-Enterprise-Version")
	return v, nil
}

func (c *V3Client) GetAuthenticatedUser(ctx context.Context) (*User, error) {
	var u User
	err := c.requestGet(ctx, "/user", &u)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

var MockGetAuthenticatedUserEmails func(ctx context.Context) ([]*UserEmail, error)

// GetAuthenticatedUserEmails returns the first 100 emails associated with the currently
// authenticated user.
func (c *V3Client) GetAuthenticatedUserEmails(ctx context.Context) ([]*UserEmail, error) {
	if MockGetAuthenticatedUserEmails != nil {
		return MockGetAuthenticatedUserEmails(ctx)
	}

	var emails []*UserEmail
	err := c.requestGet(ctx, "/user/emails?per_page=100", &emails)
	if err != nil {
		return nil, err
	}
	return emails, nil
}

var MockGetAuthenticatedUserOrgs func(ctx context.Context) ([]*Org, error)

// GetAuthenticatedUserOrgs returns the first 100 organizations associated with the currently
// authenticated user.
func (c *V3Client) GetAuthenticatedUserOrgs(ctx context.Context) ([]*Org, error) {
	if MockGetAuthenticatedUserOrgs != nil {
		return MockGetAuthenticatedUserOrgs(ctx)
	}

	var orgs []*Org
	err := c.requestGet(ctx, "/user/orgs?per_page=100", &orgs)
	if err != nil {
		return nil, err
	}
	return orgs, nil
}

var MockGetAuthenticatedUserOAuthScopes func(ctx context.Context) ([]string, error)

// GetAuthenticatedUserOAuthScopes gets the list of OAuth scopes granted to the
// currently authenticate user.
func (c *V3Client) GetAuthenticatedUserOAuthScopes(ctx context.Context) ([]string, error) {
	if MockGetAuthenticatedUserOAuthScopes != nil {
		return MockGetAuthenticatedUserOAuthScopes(ctx)
	}
	// We only care about headers
	var dest struct{}
	header, err := c.requestGetWithHeader(ctx, "/user", &dest)
	if err != nil {
		return nil, err
	}
	scope := header.Get("x-oauth-scopes")
	if scope == "" {
		return []string{}, nil
	}
	return strings.Split(scope, ", "), nil
}

// ListRepositoryCollaborators lists all GitHub users that has access to the repository.
// The page is the page of results to return, and is 1-indexed (so the first call should
// be for page 1).
func (c *V3Client) ListRepositoryCollaborators(ctx context.Context, owner, repo string, page int) (users []*Collaborator, hasNextPage bool, _ error) {
	path := fmt.Sprintf("/repos/%s/%s/collaborators?page=%d&per_page=100", owner, repo, page)
	err := c.requestGet(ctx, path, &users)
	if err != nil {
		return nil, false, err
	}
	return users, len(users) > 0, nil
}

// GetRepository gets a repository from GitHub by owner and repository name.
func (c *V3Client) GetRepository(ctx context.Context, owner, name string) (*Repository, error) {
	if GetRepositoryMock != nil {
		return GetRepositoryMock(ctx, owner, name)
	}

	key := ownerNameCacheKey(owner, name)
	return c.cachedGetRepository(ctx, key, func(ctx context.Context) (repo *Repository, keys []string, err error) {
		keys = append(keys, key)
		repo, err = c.getRepositoryFromAPI(ctx, owner, name)
		if repo != nil {
			keys = append(keys, nodeIDCacheKey(repo.ID)) // also cache under GraphQL node ID
		}
		return repo, keys, err
	}, false)
}

// getRepositoryFromCache attempts to get a response from the redis cache.
// It returns nil error for cache-hit condition and non-nil error for cache-miss.
func (c *V3Client) getRepositoryFromCache(ctx context.Context, key string) *cachedRepo {
	b, ok := c.repoCache.Get(strings.ToLower(key))
	if !ok {
		return nil
	}

	var cached cachedRepo
	if err := json.Unmarshal(b, &cached); err != nil {
		return nil
	}

	return &cached
}

// addRepositoryToCache will cache the value for repo. The caller can provide multiple cache keys
// for the multiple ways that this repository can be retrieved (e.g., both "owner/name" and the
// GraphQL node ID).
func (c *V3Client) addRepositoryToCache(keys []string, repo *cachedRepo) {
	b, err := json.Marshal(repo)
	if err != nil {
		return
	}
	for _, key := range keys {
		c.repoCache.Set(strings.ToLower(key), b)
	}
}

// addRepositoriesToCache will cache repositories that exist
// under relevant cache keys.
func (c *V3Client) addRepositoriesToCache(repos []*Repository) {
	for _, repo := range repos {
		keys := []string{nameWithOwnerCacheKey(repo.NameWithOwner), nodeIDCacheKey(repo.ID)} // cache under multiple
		c.addRepositoryToCache(keys, &cachedRepo{Repository: *repo})
	}
}

// getPublicRepositories returns a page of public repositories that were created
// after the repository identified by sinceRepoID.
// An empty sinceRepoID returns the first page of results.
// This is only intended to be called for GitHub Enterprise, so no rate limit information is returned.
// https://developer.github.com/v3/repos/#list-all-public-repositories
func (c *V3Client) getPublicRepositories(ctx context.Context, sinceRepoID int64) ([]*Repository, error) {
	path := "repositories"
	if sinceRepoID > 0 {
		path += "?per_page=100&since=" + strconv.FormatInt(sinceRepoID, 10)
	}
	return c.listRepositories(ctx, path)
}

func (c *V3Client) ListPublicRepositories(ctx context.Context, sinceRepoID int64) ([]*Repository, error) {
	repos, err := c.getPublicRepositories(ctx, sinceRepoID)
	if err != nil {
		return nil, err
	}
	c.addRepositoriesToCache(repos)
	return repos, nil
}

// ListAffiliatedRepositories lists GitHub repositories affiliated with the client
// token. page is the page of results to return. Pages are 1-indexed (so the
// first call should be for page 1).
func (c *V3Client) ListAffiliatedRepositories(ctx context.Context, visibility Visibility, page int) (
	repos []*Repository,
	hasNextPage bool,
	rateLimitCost int,
	err error,
) {
	path := fmt.Sprintf("user/repos?sort=created&visibility=%s&page=%d&per_page=100", visibility, page)
	repos, err = c.listRepositories(ctx, path)
	if err == nil {
		c.addRepositoriesToCache(repos)
	}

	return repos, len(repos) > 0, 1, err
}

// ListOrgRepositories lists GitHub repositories from the specified organization.
// org is the name of the organization. page is the page of results to return.
// Pages are 1-indexed (so the first call should be for page 1).
func (c *V3Client) ListOrgRepositories(ctx context.Context, org string, page int, repoType string) (repos []*Repository, hasNextPage bool, rateLimitCost int, err error) {
	path := fmt.Sprintf("orgs/%s/repos?sort=created&page=%d&per_page=100&type=%s", org, page, repoType)
	repos, err = c.listRepositories(ctx, path)
	return repos, len(repos) > 0, 1, err
}

// ListUserRepositories lists GitHub repositories from the specified user.
// Pages are 1-indexed (so the first call should be for page 1)
func (c *V3Client) ListUserRepositories(ctx context.Context, user string, page int) (repos []*Repository, hasNextPage bool, rateLimitCost int, err error) {
	path := fmt.Sprintf("users/%s/repos?sort=created&type=owner&page=%d&per_page=100", user, page)
	repos, err = c.listRepositories(ctx, path)
	return repos, len(repos) > 0, 1, err
}

func (c *V3Client) ListRepositoriesForSearch(ctx context.Context, searchString string, page int) (RepositoryListPage, error) {
	urlValues := url.Values{
		"q":        []string{searchString},
		"page":     []string{strconv.Itoa(page)},
		"per_page": []string{"100"},
	}
	path := "search/repositories?" + urlValues.Encode()
	var response restSearchResponse
	if err := c.requestGet(ctx, path, &response); err != nil {
		return RepositoryListPage{}, err
	}
	if response.IncompleteResults {
		return RepositoryListPage{}, ErrIncompleteResults
	}
	repos := make([]*Repository, 0, len(response.Items))
	for _, restRepo := range response.Items {
		repos = append(repos, convertRestRepo(restRepo))
	}
	c.addRepositoriesToCache(repos)

	return RepositoryListPage{
		TotalCount:  response.TotalCount,
		Repos:       repos,
		HasNextPage: page*100 < response.TotalCount,
	}, nil
}

// ListTopicsOnRepository lists topics on the given repository.
func (c *V3Client) ListTopicsOnRepository(ctx context.Context, ownerAndName string) ([]string, error) {
	owner, name, err := SplitRepositoryNameWithOwner(ownerAndName)
	if err != nil {
		return nil, err
	}

	var result restTopicsResponse
	if err := c.requestGet(ctx, fmt.Sprintf("/repos/%s/%s/topics", owner, name), &result); err != nil {
		if HTTPErrorCode(err) == http.StatusNotFound {
			return nil, ErrRepoNotFound
		}
		return nil, err
	}
	return result.Names, nil
}

// ListInstallationRepositories lists repositories on which the authenticated
// GitHub App has been installed.
func (c *V3Client) ListInstallationRepositories(ctx context.Context) ([]*Repository, error) {
	type response struct {
		Repositories []restRepository `json:"repositories"`
	}
	var resp response
	if err := c.requestGet(ctx, "installation/repositories", &resp); err != nil {
		return nil, err
	}
	repos := make([]*Repository, 0, len(resp.Repositories))
	for _, restRepo := range resp.Repositories {
		repos = append(repos, convertRestRepo(restRepo))
	}
	return repos, nil
}

// listRepositories is a generic method that unmarshals the given
// JSON HTTP endpoint into a []restRepository. It will return an
// error if it fails.
//
// This is used to extract repositories from the GitHub API endpoints:
// - /users/:user/repos
// - /orgs/:org/repos
// - /user/repos
func (c *V3Client) listRepositories(ctx context.Context, requestURI string) ([]*Repository, error) {
	var restRepos []restRepository
	if err := c.requestGet(ctx, requestURI, &restRepos); err != nil {
		return nil, err
	}
	repos := make([]*Repository, 0, len(restRepos))
	for _, restRepo := range restRepos {
		repos = append(repos, convertRestRepo(restRepo))
	}
	return repos, nil
}
