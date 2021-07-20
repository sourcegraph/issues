package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Masterminds/semver"
	"github.com/cockroachdb/errors"
	"github.com/graphql-go/graphql/language/ast"
	"github.com/graphql-go/graphql/language/parser"
	"github.com/graphql-go/graphql/language/visitor"
	"github.com/inconshreveable/log15"
	"golang.org/x/time/rate"

	"github.com/sourcegraph/sourcegraph/internal/extsvc/auth"
	"github.com/sourcegraph/sourcegraph/internal/httpcli"
	"github.com/sourcegraph/sourcegraph/internal/ratelimit"
)

// V4Client is a GitHub GraphQL API client.
type V4Client struct {
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

	// rateLimitMonitor is the API rate limit monitor.
	rateLimitMonitor *ratelimit.Monitor

	// rateLimit is our self imposed rate limiter.
	rateLimit *rate.Limiter
}

// NewV4Client creates a new GitHub GraphQL API client with an optional default
// authenticator.
//
// apiURL must point to the base URL of the GitHub API. See the docstring for
// V4Client.apiURL.
func NewV4Client(apiURL *url.URL, a auth.Authenticator, cli httpcli.Doer) *V4Client {
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
	rlm := ratelimit.DefaultMonitorRegistry.GetOrSet(apiURL.String(), tokenHash, "graphql", &ratelimit.Monitor{HeaderPrefix: "X-"})

	return &V4Client{
		apiURL:           apiURL,
		githubDotCom:     urlIsGitHubDotCom(apiURL),
		auth:             a,
		httpClient:       cli,
		rateLimit:        rl,
		rateLimitMonitor: rlm,
	}
}

// WithAuthenticator returns a new V4Client that uses the same configuration as
// the current V4Client, except authenticated as the GitHub user with the given
// authenticator instance (most likely a token).
func (c *V4Client) WithAuthenticator(a auth.Authenticator) *V4Client {
	return NewV4Client(c.apiURL, a, c.httpClient)
}

// RateLimitMonitor exposes the rate limit monitor.
func (c *V4Client) RateLimitMonitor() *ratelimit.Monitor {
	return c.rateLimitMonitor
}

func (c *V4Client) requestGraphQL(ctx context.Context, query string, vars map[string]interface{}, result interface{}) (err error) {
	reqBody, err := json.Marshal(struct {
		Query     string                 `json:"query"`
		Variables map[string]interface{} `json:"variables"`
	}{
		Query:     query,
		Variables: vars,
	})
	if err != nil {
		return err
	}

	// GitHub.com GraphQL endpoint is api.github.com/graphql. GitHub Enterprise is /api/graphql (the
	// REST endpoint is /api/v3, necessitating the "..").
	graphqlEndpoint := "/graphql"
	if !c.githubDotCom {
		graphqlEndpoint = "../graphql"
	}
	req, err := http.NewRequest("POST", graphqlEndpoint, bytes.NewReader(reqBody))
	if err != nil {
		return err
	}

	// Enable Checks API
	// https://developer.github.com/v4/previews/#checks
	req.Header.Add("Accept", "application/vnd.github.antiope-preview+json")
	var respBody struct {
		Data   json.RawMessage `json:"data"`
		Errors graphqlErrors   `json:"errors"`
	}

	cost, err := estimateGraphQLCost(query)
	if err != nil {
		return errors.Wrap(err, "estimating graphql cost")
	}

	if err := c.rateLimit.WaitN(ctx, cost); err != nil {
		return errors.Wrap(err, "rate limit")
	}

	time.Sleep(c.rateLimitMonitor.RecommendedWaitForBackgroundOp(cost))

	if _, err := doRequest(ctx, c.apiURL, c.auth, c.rateLimitMonitor, c.httpClient, req, &respBody); err != nil {
		return err
	}

	// If the GraphQL response has errors, still attempt to unmarshal the data portion, as some
	// requests may expect errors but have useful responses (e.g., querying a list of repositories,
	// some of which you expect to 404).
	if len(respBody.Errors) > 0 {
		err = respBody.Errors
	}
	if result != nil && respBody.Data != nil {
		if err0 := unmarshal(respBody.Data, result); err0 != nil && err == nil {
			return err0
		}
	}
	return err
}

// estimateGraphQLCost estimates the cost of the query as described here:
// https://developer.github.com/v4/guides/resource-limitations/#calculating-a-rate-limit-score-before-running-the-call
func estimateGraphQLCost(query string) (int, error) {
	doc, err := parser.Parse(parser.ParseParams{
		Source: query,
	})
	if err != nil {
		return 0, errors.Wrap(err, "parsing query")
	}

	var totalCost int
	for _, def := range doc.Definitions {
		cost := calcDefinitionCost(def)
		totalCost += cost
	}

	// As per the calculation spec, cost should be divided by 100
	totalCost /= 100
	if totalCost < 1 {
		return 1, nil
	}
	return totalCost, nil
}

type limitDepth struct {
	// The 'first' or 'last' limit
	limit int
	// The depth at which it was added
	depth int
}

func calcDefinitionCost(def ast.Node) int {
	var cost int
	limitStack := make([]limitDepth, 0)

	v := &visitor.VisitorOptions{
		Enter: func(p visitor.VisitFuncParams) (string, interface{}) {
			switch node := p.Node.(type) {
			case *ast.IntValue:
				// We're looking for a 'first' or 'last' param indicating a limit
				parent, ok := p.Parent.(*ast.Argument)
				if !ok {
					return visitor.ActionNoChange, nil
				}
				if parent.Name == nil {
					return visitor.ActionNoChange, nil
				}
				if parent.Name.Value != "first" && parent.Name.Value != "last" {
					return visitor.ActionNoChange, nil
				}

				// Prune anything above our current depth as we may have started walking
				// back down the tree
				currentDepth := len(p.Ancestors)
				limitStack = filterInPlace(limitStack, currentDepth)

				limit, err := strconv.Atoi(node.Value)
				if err != nil {
					return "", errors.Wrap(err, "parsing limit")
				}
				limitStack = append(limitStack, limitDepth{limit: limit, depth: currentDepth})
				// The first item in the tree is always worth 1
				if len(limitStack) == 1 {
					cost++
					return visitor.ActionNoChange, nil
				}
				// The cost of the current item is calculated using the limits of
				// its children
				children := limitStack[:len(limitStack)-1]
				product := 1
				// Multiply them all together
				for _, n := range children {
					product = n.limit * product
				}
				cost += product
			}
			return visitor.ActionNoChange, nil
		},
	}

	_ = visitor.Visit(def, v, nil)

	return cost
}

func filterInPlace(limitStack []limitDepth, depth int) []limitDepth {
	n := 0
	for _, x := range limitStack {
		if depth > x.depth {
			limitStack[n] = x
			n++
		}
	}
	limitStack = limitStack[:n]
	return limitStack
}

// graphqlErrors describes the errors in a GraphQL response. It contains at least 1 element when returned by
// requestGraphQL. See https://graphql.github.io/graphql-spec/June2018/#sec-Errors.
type graphqlErrors []struct {
	Message   string        `json:"message"`
	Type      string        `json:"type"`
	Path      []interface{} `json:"path"`
	Locations []struct {
		Line   int `json:"line"`
		Column int `json:"column"`
	} `json:"locations,omitempty"`
}

const graphqlErrTypeNotFound = "NOT_FOUND"

func (e graphqlErrors) Error() string {
	return fmt.Sprintf("error in GraphQL response: %s", e[0].Message)
}

// unmarshal wraps json.Unmarshal, but includes extra context in the case of
// json.UnmarshalTypeError
func unmarshal(data []byte, v interface{}) error {
	err := json.Unmarshal(data, v)
	var e *json.UnmarshalTypeError
	if errors.As(err, &e) && e.Offset >= 0 {
		a := e.Offset - 100
		b := e.Offset + 100
		if a < 0 {
			a = 0
		}
		if b > int64(len(data)) {
			b = int64(len(data))
		}
		if e.Offset >= int64(len(data)) {
			return errors.Wrapf(err, "graphql: cannot unmarshal at offset %d: before %q", e.Offset, string(data[a:e.Offset]))
		}
		return errors.Wrapf(err, "graphql: cannot unmarshal at offset %d: before %q; after %q", e.Offset, string(data[a:e.Offset]), string(data[e.Offset:b]))
	}
	return err
}

// determineGitHubVersion returns a *semver.Version for the targetted GitHub instance by this client. When an
// error occurs, we print a warning to the logs but don't fail and return the allMatchingSemver.
func (c *V4Client) determineGitHubVersion(ctx context.Context) *semver.Version {
	url := normalizeURL(c.apiURL.String())
	globalVersionCache.mu.Lock()
	defer globalVersionCache.mu.Unlock()

	if globalVersionCache.lastReset.IsZero() || time.Now().After(globalVersionCache.lastReset.Add(versionCacheResetTime)) {
		// Clear cache and set last expiry to now.
		globalVersionCache.lastReset = time.Now()
		globalVersionCache.versions = make(map[string]*semver.Version)
	}
	if version, ok := globalVersionCache.versions[url]; ok {
		return version
	}
	version := c.fetchGitHubVersion(ctx)
	globalVersionCache.versions[url] = version
	return version
}

func (c *V4Client) fetchGitHubVersion(ctx context.Context) *semver.Version {
	if c.githubDotCom {
		return allMatchingSemver
	}

	var resp struct {
		InstalledVersion string `json:"installed_version"`
	}
	req, err := http.NewRequest("GET", "/meta", nil)
	if err != nil {
		log15.Warn("Failed to fetch GitHub enterprise version", "build request", "apiURL", c.apiURL, "err", err)
		return allMatchingSemver
	}
	if _, err = doRequest(ctx, c.apiURL, c.auth, c.rateLimitMonitor, c.httpClient, req, &resp); err != nil {
		log15.Warn("Failed to fetch GitHub enterprise version", "doRequest", "apiURL", c.apiURL, "err", err)
		return allMatchingSemver
	}
	version, err := semver.NewVersion(resp.InstalledVersion)
	if err == nil {
		return version
	}
	log15.Warn("Failed to fetch GitHub enterprise version", "parse version", "apiURL", c.apiURL, "resp.InstalledVersion", resp.InstalledVersion, "err", err)
	return allMatchingSemver
}

func (c *V4Client) GetAuthenticatedUser(ctx context.Context) (*Actor, error) {
	var result struct {
		Viewer Actor `json:"viewer"`
	}
	err := c.requestGraphQL(ctx, `query GetAuthenticatedUser {
    viewer {
        login
        avatarUrl
        url
    }
}`, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result.Viewer, nil
}

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
func (c *V4Client) GetReposByNameWithOwner(ctx context.Context, namesWithOwners ...string) ([]*Repository, error) {
	if len(namesWithOwners) > 30 {
		return nil, ErrBatchTooLarge
	}

	query, err := c.buildGetReposBatchQuery(ctx, namesWithOwners)
	if err != nil {
		return nil, err
	}

	var result map[string]*Repository
	err = c.requestGraphQL(ctx, query, map[string]interface{}{}, &result)
	if err != nil {
		var e graphqlErrors
		if errors.As(err, &e) {
			for _, err2 := range e {
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

func (c *V4Client) buildGetReposBatchQuery(ctx context.Context, namesWithOwners []string) (string, error) {
	var b strings.Builder
	b.WriteString(c.repositoryFieldsGraphQLFragment(ctx))
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

// repositoryFieldsGraphQLFragment returns a GraphQL fragment that contains the fields needed to populate the
// Repository struct.
func (c *V4Client) repositoryFieldsGraphQLFragment(ctx context.Context) string {
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
	isLocked
	isDisabled
	viewerPermission
	stargazerCount
	forkCount
}
	`
	}
	ghe300Fields := []string{}
	version := c.determineGitHubVersion(ctx)
	if ghe300PlusOrDotComSemver.Check(version) {
		ghe300Fields = append(ghe300Fields, "stargazerCount")
	}
	// Some fields are not yet available on GitHub Enterprise yet
	// or are available but too new to expect our customers to have updated:
	// - viewerPermission
	return fmt.Sprintf(`
fragment RepositoryFields on Repository {
	id
	databaseId
	nameWithOwner
	description
	url
	isPrivate
	isFork
	isArchived
	isLocked
	isDisabled
	forkCount
	%s
}
	`, strings.Join(ghe300Fields, "\n	"))
}
