package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/inconshreveable/log15"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
)

// SplitRepositoryNameWithOwner splits a GitHub repository's "owner/name" string into "owner" and "name", with
// validation.
func SplitRepositoryNameWithOwner(nameWithOwner string) (owner, repo string, err error) {
	parts := strings.SplitN(nameWithOwner, "/", 2)
	if len(parts) != 2 || strings.Contains(parts[1], "/") || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid GitHub repository \"owner/name\" string: %q", nameWithOwner)
	}
	return parts[0], parts[1], nil
}

// Repository is a GitHub repository.
type Repository struct {
	ID            string // ID of repository (GitHub GraphQL ID, not GitHub database ID)
	DatabaseID    int64  // The integer database id
	NameWithOwner string // full name of repository ("owner/name")
	Description   string // description of repository
	URL           string // the web URL of this repository ("https://github.com/foo/bar")
	IsPrivate     bool   // whether the repository is private
	IsFork        bool   // whether the repository is a fork of another repository
	IsArchived    bool   // whether the repository is archived on the code host
	// This field will always be blank on repos stored in our database because the value will be different
	// depending on which token was used to fetch it
	ViewerPermission string // ADMIN, WRITE, READ, or empty if unknown. Only the graphql api populates this. https://developer.github.com/v4/enum/repositorypermission/
}

// repositoryFieldsGraphQLFragment returns a GraphQL fragment that contains the fields needed to populate the
// Repository struct.
func (c *Client) repositoryFieldsGraphQLFragment() string {
	if c.githubDotCom {
		return `
fragment RepositoryFields on Repository {
	id
	databaseId
	nameWithOwner
	description
	url
	isPrivate
	isFork
	isArchived
	viewerPermission
}
	`
	}
	// Some fields are not yet available on GitHub Enterprise yet
	// or are available but too new to expect our customers to have updated:
	// - viewerPermission
	return `
fragment RepositoryFields on Repository {
	id
	databaseId
	nameWithOwner
	description
	url
	isPrivate
	isFork
	isArchived
}
	`
}

func ownerNameCacheKey(owner, name string) string       { return "0:" + owner + "/" + name }
func nameWithOwnerCacheKey(nameWithOwner string) string { return "0:" + nameWithOwner }
func nodeIDCacheKey(id string) string                   { return "1:" + id }

// GetRepositoryMock is set by tests to mock (*Client).GetRepository.
var GetRepositoryMock func(ctx context.Context, owner, name string) (*Repository, error)

// MockGetRepository_Return is called by tests to mock (*Client).GetRepository.
func MockGetRepository_Return(returns *Repository) {
	GetRepositoryMock = func(context.Context, string, string) (*Repository, error) {
		return returns, nil
	}
}

// GetRepository gets a repository from GitHub by owner and repository name.
func (c *Client) GetRepository(ctx context.Context, owner, name string) (*Repository, error) {
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

// GetRepositoryByNodeID gets a repository from GitHub by its GraphQL node ID using the specified user token.
func (c *Client) GetRepositoryByNodeID(ctx context.Context, id string) (*Repository, error) {
	return c.getRepositoryByNodeID(ctx, id, false)
}

// HACK: allows us to bypass the GitHub client cache (used for fetching repositories to determine
// permissions). TODO(beyang): merge this into a unified GetRepositoryByNodeID with a clean
// interface.
func (c *Client) GetRepositoryByNodeIDNoCache(ctx context.Context, id string) (*Repository, error) {
	return c.getRepositoryByNodeID(ctx, id, true)
}

func (c *Client) getRepositoryByNodeID(ctx context.Context, id string, nocache bool) (*Repository, error) {
	key := nodeIDCacheKey(id)

	return c.cachedGetRepository(ctx, key, func(ctx context.Context) (repo *Repository, keys []string, err error) {
		keys = append(keys, key)
		repo, err = c.getRepositoryByNodeIDFromAPI(ctx, id)
		if repo != nil {
			keys = append(keys, nameWithOwnerCacheKey(repo.NameWithOwner)) // also cache under "owner/name"
		}
		return repo, keys, err
	}, nocache)
}

// cachedGetRepository caches the getRepositoryFromAPI call.
func (c *Client) cachedGetRepository(ctx context.Context, key string, getRepositoryFromAPI func(ctx context.Context) (repo *Repository, keys []string, err error), nocache bool) (*Repository, error) {
	if !nocache {
		if cached := c.getRepositoryFromCache(ctx, key); cached != nil {
			reposGitHubCacheCounter.WithLabelValues("hit").Inc()
			if cached.NotFound {
				return nil, ErrNotFound
			}
			return &cached.Repository, nil
		}
	}

	repo, keys, err := getRepositoryFromAPI(ctx)
	if IsNotFound(err) {
		// Before we do anything, ensure we cache NotFound responses.
		// Do this if client is unauthed or authed, it's okay since we're only caching not found responses here.
		c.addRepositoryToCache(keys, &cachedRepo{NotFound: true})
		reposGitHubCacheCounter.WithLabelValues("notfound").Inc()
	}
	if err != nil {
		reposGitHubCacheCounter.WithLabelValues("error").Inc()
		return nil, err
	}

	c.addRepositoryToCache(keys, &cachedRepo{Repository: *repo})
	reposGitHubCacheCounter.WithLabelValues("miss").Inc()

	return repo, nil
}

var reposGitHubCacheCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: "src_repos_github_cache_hit",
	Help: "Counts cache hits and misses for GitHub repo metadata.",
}, []string{"type"})

func init() {
	prometheus.MustRegister(reposGitHubCacheCounter)
}

type cachedRepo struct {
	Repository

	// NotFound indicates that the GitHub API reported that the repository was not found.
	NotFound bool
}

// getRepositoryFromCache attempts to get a response from the redis cache.
// It returns nil error for cache-hit condition and non-nil error for cache-miss.
func (c *Client) getRepositoryFromCache(ctx context.Context, key string) *cachedRepo {
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
func (c *Client) addRepositoryToCache(keys []string, repo *cachedRepo) {
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
func (c *Client) addRepositoriesToCache(repos []*Repository) {
	for _, repo := range repos {
		keys := []string{nameWithOwnerCacheKey(repo.NameWithOwner), nodeIDCacheKey(repo.ID)} // cache under multiple
		c.addRepositoryToCache(keys, &cachedRepo{Repository: *repo})
	}
}

type restRepositoryPermissions struct {
	Admin bool `json:"admin"`
	Push  bool `json:"push"`
	Pull  bool `json:"pull"`
}

type restRepository struct {
	ID          string `json:"node_id"` // GraphQL ID
	DatabaseID  int64  `json:"id"`
	FullName    string `json:"full_name"` // same as nameWithOwner
	Description string
	HTMLURL     string `json:"html_url"` // web URL
	Private     bool
	Fork        bool
	Archived    bool
	Permissions restRepositoryPermissions `json:"permissions"`
}

// getRepositoryFromAPI attempts to fetch a repository from the GitHub API without use of the redis cache.
func (c *Client) getRepositoryFromAPI(ctx context.Context, owner, name string) (*Repository, error) {
	// If no token, we must use the older REST API, not the GraphQL API. See
	// https://platform.github.community/t/anonymous-access/2093/2. This situation occurs on (for
	// example) a server with autoAddRepos and no GitHub connection configured when someone visits
	// http://[sourcegraph-hostname]/github.com/foo/bar.
	var result restRepository
	if err := c.requestGet(ctx, fmt.Sprintf("/repos/%s/%s", owner, name), &result); err != nil {
		if HTTPErrorCode(err) == http.StatusNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return convertRestRepo(result), nil
}

// convertRestRepo converts repo information returned by the rest API
// to a standard format.
func convertRestRepo(restRepo restRepository) *Repository {
	return &Repository{
		ID:               restRepo.ID,
		DatabaseID:       restRepo.DatabaseID,
		NameWithOwner:    restRepo.FullName,
		Description:      restRepo.Description,
		URL:              restRepo.HTMLURL,
		IsPrivate:        restRepo.Private,
		IsFork:           restRepo.Fork,
		IsArchived:       restRepo.Archived,
		ViewerPermission: convertRestRepoPermissions(restRepo.Permissions),
	}
}

// convertRestRepoPermissions converts repo information returned by the rest API
// to a standard format.
func convertRestRepoPermissions(restRepoPermissions restRepositoryPermissions) string {
	if restRepoPermissions.Admin {
		return "ADMIN"
	}
	if restRepoPermissions.Push {
		return "WRITE"
	}
	if restRepoPermissions.Pull {
		return "READ"
	}
	return ""
}

// getPublicRepositories returns a page of public repositories that were created
// after the repository identified by sinceRepoID.
// An empty sinceRepoID returns the first page of results.
// This is only intended to be called for GitHub Enterprise, so no rate limit information is returned.
// https://developer.github.com/v3/repos/#list-all-public-repositories
func (c *Client) getPublicRepositories(ctx context.Context, sinceRepoID int64) ([]*Repository, error) {
	path := "repositories"
	if sinceRepoID > 0 {
		path += "?per_page=100&since=" + strconv.FormatInt(sinceRepoID, 10)
	}
	return c.listRepositories(ctx, path)
}

// getRepositoryByNodeIDFromAPI attempts to fetch a repository by GraphQL node ID from the GitHub
// API without use of the redis cache.
func (c *Client) getRepositoryByNodeIDFromAPI(ctx context.Context, id string) (*Repository, error) {
	var result struct {
		Node *Repository `json:"node"`
	}
	if err := c.requestGraphQL(ctx, `
query Repository($id: ID!) {
	node(id: $id) {
		... on Repository {
			...RepositoryFields
		}
	}
}`+c.repositoryFieldsGraphQLFragment(),
		map[string]interface{}{"id": id},
		&result,
	); err != nil {
		if gqlErrs, ok := err.(graphqlErrors); ok {
			for _, err2 := range gqlErrs {
				if err2.Type == graphqlErrTypeNotFound {
					return nil, ErrNotFound
				}
			}
		}
		return nil, err
	}
	if result.Node == nil {
		return nil, ErrNotFound
	}
	return result.Node, nil
}

// MaxNodeIDs is the maximum number of repository nodes that can be queried in one call to the
// GitHub GraphQL API.
var MaxNodeIDs = 100

// GetRepositoriesByNodeIDFromAPI fetches the specified repositories (nodeIDs) and returns a map
// from node ID to repository metadata. If a repository is not found, it will not be present in the
// return map. The caller should respect the max nodeID count limit of the GitHub API, which at the
// time of writing, is 100 (if the caller does not respect this match, this method will return an
// error). This method does not cache.
func (c *Client) GetRepositoriesByNodeIDFromAPI(ctx context.Context, nodeIDs []string) (map[string]*Repository, error) {
	var result struct {
		Nodes []*Repository
	}
	err := c.requestGraphQL(ctx, `
query Repositories($ids: [ID!]!) {
	nodes(ids: $ids) {
		... on Repository {
			...RepositoryFields
		}
	}
}
`+c.repositoryFieldsGraphQLFragment(), map[string]interface{}{"ids": nodeIDs}, &result)
	if err != nil {
		if gqlErrs, ok := err.(graphqlErrors); ok {
			for _, err2 := range gqlErrs {
				if err2.Type == graphqlErrTypeNotFound {
					continue
				}
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	repos := make(map[string]*Repository)
	for _, r := range result.Nodes {
		if r != nil {
			repos[r.ID] = r
		}
	}
	return repos, nil
}

// ErrBatchTooLarge is when the requested batch of GitHub repositories to fetch
// is too large and goes over the limit of what can be requested in a single
// GraphQL call
var ErrBatchTooLarge = errors.New("requested batch of GitHub repositories too large")

// GetReposByNameWithOwner fetches the specified repositories (namesWithOwners)
// from the GitHub GraphQL API and returns a slice of repositories.
// If a repository is not found, it will return an error.
//
// The maximum number of repositories to be fetched is 30. If more
// namesWithOwners are given, the method returns an error. 30 is not a official
// limit of the API, but based on the observation that the GitHub GraphQL does
// not return results when more than 37 aliases are specified in a query. 30 is
// the conservative step back from 37.
//
// This method does not cache.
func (c *Client) GetReposByNameWithOwner(ctx context.Context, namesWithOwners ...string) ([]*Repository, error) {
	if len(namesWithOwners) > 30 {
		return nil, ErrBatchTooLarge
	}

	query, err := c.buildGetReposBatchQuery(namesWithOwners)
	if err != nil {
		return nil, err
	}

	var result map[string]*Repository
	err = c.requestGraphQL(ctx, query, map[string]interface{}{}, &result)
	if err != nil {
		if gqlErrs, ok := err.(graphqlErrors); ok {
			for _, err2 := range gqlErrs {
				if err2.Type == graphqlErrTypeNotFound {
					log15.Warn("GitHub repository not found", "error", err2)
					continue
				}
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	repos := make([]*Repository, 0, len(result))
	for _, r := range result {
		if r != nil {
			repos = append(repos, r)
		}
	}
	return repos, nil
}

func (c *Client) buildGetReposBatchQuery(namesWithOwners []string) (string, error) {
	var b strings.Builder
	b.WriteString(c.repositoryFieldsGraphQLFragment())
	b.WriteString("query {\n")

	for i, pair := range namesWithOwners {
		owner, name, err := SplitRepositoryNameWithOwner(pair)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&b, "repo%d: repository(owner: %q, name: %q) { ", i, owner, name)
		b.WriteString("... on Repository { ...RepositoryFields } }\n")
	}

	b.WriteString("}")

	return b.String(), nil
}

func (c *Client) ListPublicRepositories(ctx context.Context, sinceRepoID int64) ([]*Repository, error) {
	repos, err := c.getPublicRepositories(ctx, sinceRepoID)
	if err != nil {
		return nil, err
	}
	c.addRepositoriesToCache(repos)
	return repos, nil
}

// Visibility is the visibility filter for listing repositories.
type Visibility string

const (
	VisibilityAll     Visibility = "all"
	VisibilityPublic  Visibility = "public"
	VisibilityPrivate Visibility = "private"
)

// ListAffiliatedRepositories lists GitHub repositories affiliated with the client
// token. page is the page of results to return. Pages are 1-indexed (so the
// first call should be for page 1).
func (c *Client) ListAffiliatedRepositories(ctx context.Context, visibility Visibility, page int) (
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
func (c *Client) ListOrgRepositories(ctx context.Context, org string, page int) (repos []*Repository, hasNextPage bool, rateLimitCost int, err error) {
	path := fmt.Sprintf("orgs/%s/repos?sort=created&page=%d&per_page=100", org, page)
	repos, err = c.listRepositories(ctx, path)
	return repos, len(repos) > 0, 1, err
}

// ListUserRepositories lists GitHub repositories from the specified user.
// Pages are 1-indexed (so the first call should be for page 1)
func (c *Client) ListUserRepositories(ctx context.Context, user string, page int) (repos []*Repository, hasNextPage bool, rateLimitCost int, err error) {
	path := fmt.Sprintf("users/%s/repos?sort=created&type=owner&page=%d&per_page=100", user, page)
	repos, err = c.listRepositories(ctx, path)
	return repos, len(repos) > 0, 1, err
}

type restSearchResponse struct {
	TotalCount        int              `json:"total_count"`
	IncompleteResults bool             `json:"incomplete_results"`
	Items             []restRepository `json:"items"`
}

// RepositoryListPage is a page of repositories returned from the GitHub Search API.
type RepositoryListPage struct {
	TotalCount  int
	Repos       []*Repository
	HasNextPage bool
}

func (c *Client) ListRepositoriesForSearch(ctx context.Context, searchString string, page int) (RepositoryListPage, error) {
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

type restTopicsResponse struct {
	Names []string `json:"names"`
}

// ListTopicsOnRepository lists topics on the given repository.
func (c *Client) ListTopicsOnRepository(ctx context.Context, ownerAndName string) ([]string, error) {
	owner, name, err := SplitRepositoryNameWithOwner(ownerAndName)
	if err != nil {
		return nil, err
	}

	var result restTopicsResponse
	if err := c.requestGet(ctx, fmt.Sprintf("/repos/%s/%s/topics", owner, name), &result); err != nil {
		if HTTPErrorCode(err) == http.StatusNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return result.Names, nil
}
