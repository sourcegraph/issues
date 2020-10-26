package gitlab

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/inconshreveable/log15"
	"github.com/pkg/errors"
	"golang.org/x/time/rate"

	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/httpcli"
	"github.com/sourcegraph/sourcegraph/internal/metrics"
	"github.com/sourcegraph/sourcegraph/internal/ratelimit"
	"github.com/sourcegraph/sourcegraph/internal/rcache"
	"github.com/sourcegraph/sourcegraph/internal/trace/ot"
)

var (
	requestCounter = metrics.NewRequestMeter("gitlab", "Total number of requests sent to the GitLab API.")

	// Whether debug logging is turned on
	traceEnabled int32 = 0
)

func init() {
	go func() {
		conf.Watch(func() {
			exp := conf.Get().ExperimentalFeatures
			if exp == nil {
				atomic.StoreInt32(&traceEnabled, 0)
				return
			}
			if debugLog := exp.DebugLog; debugLog == nil || !debugLog.ExtsvcGitlab {
				atomic.StoreInt32(&traceEnabled, 0)
				return
			}
			atomic.StoreInt32(&traceEnabled, 1)
		})
	}()
}

func trace(msg string, ctx ...interface{}) {
	if atomic.LoadInt32(&traceEnabled) == 1 {
		log15.Info(fmt.Sprintf("TRACE %s", msg), ctx...)
	}
}

// ClientProvider creates GitLab API clients. Each client has separate authentication creds and a
// separate cache, but they share an underlying HTTP client and rate limiter. Callers who want a simple
// unauthenticated API client should use `NewClientProvider(baseURL, transport).GetClient()`.
type ClientProvider struct {
	// baseURL is the base URL of GitLab; e.g., https://gitlab.com or https://gitlab.example.com
	baseURL *url.URL

	// httpClient is the underlying the HTTP client to use
	httpClient httpcli.Doer

	gitlabClients   map[string]*Client
	gitlabClientsMu sync.Mutex

	rateLimitMonitor *ratelimit.Monitor // the API rate limit monitor
}

type CommonOp struct {
	// NoCache, if true, will bypass any caching done in this package
	NoCache bool
}

func NewClientProvider(baseURL *url.URL, cli httpcli.Doer) *ClientProvider {
	if cli == nil {
		cli = httpcli.ExternalDoer()
	}
	cli = requestCounter.Doer(cli, func(u *url.URL) string {
		// The 3rd component of the Path (/api/v4/XYZ) mostly maps to the type of API
		// request we are making.
		var category string
		if parts := strings.SplitN(u.Path, "/", 3); len(parts) >= 4 {
			category = parts[3]
		}
		return category
	})

	return &ClientProvider{
		baseURL:          baseURL.ResolveReference(&url.URL{Path: path.Join(baseURL.Path, "api/v4") + "/"}),
		httpClient:       cli,
		gitlabClients:    make(map[string]*Client),
		rateLimitMonitor: &ratelimit.Monitor{},
	}
}

// GetPATClient returns a client authenticated by the personal access token.
func (p *ClientProvider) GetPATClient(personalAccessToken, sudo string) *Client {
	if personalAccessToken == "" {
		return p.getClient(getClientOp{})
	}
	return p.getClient(getClientOp{personalAccessToken: personalAccessToken, sudo: sudo})
}

// GetOAuthClient returns a client authenticated by the OAuth token.
func (p *ClientProvider) GetOAuthClient(oauthToken string) *Client {
	if oauthToken == "" {
		return p.getClient(getClientOp{})
	}
	return p.getClient(getClientOp{oauthToken: oauthToken})
}

// GetClient returns an unauthenticated client.
func (p *ClientProvider) GetClient() *Client {
	return p.getClient(getClientOp{})
}

type getClientOp struct {
	personalAccessToken string
	oauthToken          string
	sudo                string
}

func (p *ClientProvider) getClient(op getClientOp) *Client {
	if op.personalAccessToken != "" && op.oauthToken != "" {
		panic("op.personalAccessToken and op.oauthToken should never both be set")
	}
	p.gitlabClientsMu.Lock()
	defer p.gitlabClientsMu.Unlock()

	var key string
	if op.personalAccessToken != "" {
		key = fmt.Sprintf("pat::sudo:%s::%s", op.sudo, op.personalAccessToken)
	} else if op.oauthToken != "" {
		key = fmt.Sprintf("oauth::%s", op.oauthToken)
	}
	if c, ok := p.gitlabClients[key]; ok {
		return c
	}

	c := p.newClient(p.baseURL, op, p.httpClient, p.rateLimitMonitor)
	p.gitlabClients[key] = c
	return c
}

// Client is a GitLab API client. Clients are associated with a particular user identity, which is
// defined by some combination of the following fields: OAuthToken, PersonalAccessToken, Sudo. If
// Sudo is non-empty, then either PersonalAccessToken or OAuthToken must contain a sudo-level token
// and the user identity will be the user ID specified by Sudo (rather than the user that owns the
// token).
//
// The Client's cache is keyed by the union of OAuthToken, PersonalAccessToken, and Sudo. It is NOT
// keyed by the actual user ID that is defined by these. So if an OAuth token and personal access
// token belong to the same user and there are two corresponding Client instances, those Client
// instances will NOT share the same cache. However, two Client instances sharing the exact same
// values for those fields WILL share a cache.
type Client struct {
	baseURL             *url.URL
	httpClient          httpcli.Doer
	projCache           *rcache.Cache
	PersonalAccessToken string // a personal access token to authenticate requests, if set
	OAuthToken          string // an OAuth bearer token, if set
	Sudo                string // Sudo user value, if set
	RateLimitMonitor    *ratelimit.Monitor
	RateLimiter         *rate.Limiter // Our internal rate limiter
}

// newClient creates a new GitLab API client with an optional personal access token to authenticate requests.
//
// The URL must point to the base URL of the GitLab instance. This is https://gitlab.com for GitLab.com and
// http[s]://[gitlab-hostname] for self-hosted GitLab instances.
//
// See the docstring of Client for the meaning of the parameters.
func (p *ClientProvider) newClient(baseURL *url.URL, op getClientOp, httpClient httpcli.Doer, rateLimit *ratelimit.Monitor) *Client {
	// Cache for GitLab project metadata.
	var cacheTTL time.Duration
	if isGitLabDotComURL(baseURL) && op.personalAccessToken == "" && op.oauthToken == "" {
		cacheTTL = 10 * time.Minute // cache for longer when unauthenticated
	} else {
		cacheTTL = 30 * time.Second
	}
	key := sha256.Sum256([]byte(op.personalAccessToken + ":" + op.oauthToken + ":" + baseURL.String()))
	projCache := rcache.NewWithTTL("gl_proj:"+base64.URLEncoding.EncodeToString(key[:]), int(cacheTTL/time.Second))

	rl := ratelimit.DefaultRegistry.Get(baseURL.String())

	return &Client{
		baseURL:             baseURL,
		httpClient:          httpClient,
		projCache:           projCache,
		PersonalAccessToken: op.personalAccessToken,
		OAuthToken:          op.oauthToken,
		Sudo:                op.sudo,
		RateLimitMonitor:    rateLimit,
		RateLimiter:         rl,
	}
}

func isGitLabDotComURL(baseURL *url.URL) bool {
	hostname := strings.ToLower(baseURL.Hostname())
	return hostname == "gitlab.com" || hostname == "www.gitlab.com"
}

func (c *Client) do(ctx context.Context, req *http.Request, result interface{}) (responseHeader http.Header, responseCode int, err error) {
	req.URL = c.baseURL.ResolveReference(req.URL)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	if c.PersonalAccessToken != "" {
		req.Header.Set("Private-Token", c.PersonalAccessToken) // https://docs.gitlab.com/ee/api/README.html#personal-access-tokens
	}
	if c.OAuthToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.OAuthToken))
	}
	if c.Sudo != "" {
		req.Header.Set("Sudo", c.Sudo)
	}

	var resp *http.Response

	span, ctx := ot.StartSpanFromContext(ctx, "GitLab")
	span.SetTag("URL", req.URL.String())
	defer func() {
		if err != nil {
			span.SetTag("error", err.Error())
		}
		if resp != nil {
			span.SetTag("status", resp.Status)
		}
		span.Finish()
	}()

	if c.RateLimiter != nil {
		err = c.RateLimiter.Wait(ctx)
		if err != nil {
			return nil, 0, errors.Wrap(err, "rate limit")
		}
	}

	resp, err = c.httpClient.Do(req.WithContext(ctx))
	if err != nil {
		trace("GitLab API error", "method", req.Method, "url", req.URL.String(), "err", err)
		return nil, 0, err
	}
	defer resp.Body.Close()
	trace("GitLab API", "method", req.Method, "url", req.URL.String(), "respCode", resp.StatusCode)

	c.RateLimitMonitor.Update(resp.Header)
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return nil, resp.StatusCode, errors.Wrap(HTTPError(resp.StatusCode), fmt.Sprintf("unexpected response from GitLab API (%s)", req.URL))
	}

	return resp.Header, resp.StatusCode, json.NewDecoder(resp.Body).Decode(result)
}

type HTTPError int

func (err HTTPError) Error() string {
	return fmt.Sprintf("HTTP error status %d", err)
}

// HTTPErrorCode returns err's HTTP status code, if it is an HTTP error from
// this package. Otherwise it returns 0.
func HTTPErrorCode(err error) int {
	e, ok := err.(HTTPError)
	if !ok {
		// Try one level deeper.
		err = errors.Cause(err)
		e, ok = err.(HTTPError)
	}
	if ok {
		return int(e)
	}
	return 0
}

// ErrNotFound is when the requested GitLab project is not found.
var ErrNotFound = errors.New("GitLab project not found")

// IsNotFound reports whether err is a GitLab API error of type NOT_FOUND, the equivalent cached
// response error, or HTTP 404.
func IsNotFound(err error) bool {
	if err == ErrNotFound || errors.Cause(err) == ErrNotFound {
		return true
	}
	if HTTPErrorCode(err) == http.StatusNotFound {
		return true
	}
	return false
}
