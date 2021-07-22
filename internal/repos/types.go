package repos

import (
	"context"
	"encoding/hex"
	"hash/fnv"
	"strings"

	"github.com/cockroachdb/errors"

	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/ratelimit"
	"github.com/sourcegraph/sourcegraph/internal/types"
)

type externalServiceLister interface {
	List(context.Context, database.ExternalServicesListOptions) ([]*types.ExternalService, error)
}

// RateLimitSyncer syncs rate limits based on external service configuration
type RateLimitSyncer struct {
	registry      *ratelimit.Registry
	serviceLister externalServiceLister
	// How many services to fetch in each DB call
	limit int64
}

// NewRateLimitSyncer returns a new syncer
func NewRateLimitSyncer(registry *ratelimit.Registry, serviceLister externalServiceLister) *RateLimitSyncer {
	r := &RateLimitSyncer{
		registry:      registry,
		serviceLister: serviceLister,
		limit:         500,
	}
	return r
}

// SyncRateLimiters syncs all rate limiters using current config.
// We sync them all as we need to pick the most restrictive configured limit per code host
// and rate limits can be defined in multiple external services for the same host.
func (r *RateLimitSyncer) SyncRateLimiters(ctx context.Context) error {
	byURL := make(map[string]extsvc.RateLimitConfig)

	cursor := database.LimitOffset{
		Limit: int(r.limit),
	}

	for {
		services, err := r.serviceLister.List(ctx, database.ExternalServicesListOptions{
			LimitOffset: &cursor,
		})
		if err != nil {
			return errors.Wrap(err, "listing external services")
		}

		if len(services) == 0 {
			break
		}

		cursor.Offset += len(services)

		for _, svc := range services {
			rlc, err := extsvc.ExtractRateLimitConfig(svc.Config, svc.Kind, svc.DisplayName)
			if err != nil {
				if errors.HasType(err, extsvc.ErrRateLimitUnsupported{}) {
					continue
				}
				return errors.Wrap(err, "getting rate limit configuration")
			}

			current, ok := byURL[rlc.BaseURL]
			if !ok || (ok && current.IsDefault) {
				byURL[rlc.BaseURL] = rlc
				continue
			}
			// Use the lower limit, but a default value should not override
			// a limit that has been configured
			if rlc.Limit < current.Limit && !rlc.IsDefault {
				byURL[rlc.BaseURL] = rlc
			}
		}

		if len(services) < int(r.limit) {
			break
		}
	}

	for u, rl := range byURL {
		l := r.registry.Get(u)
		l.SetLimit(rl.Limit)
	}

	return nil
}

type ScopeCache interface {
	Get(string) ([]byte, bool)
	Set(string, []byte)
}

// GrantedScopes returns a slice of scopes granted by the service based on the token
// provided in the config. It makes a request to the code host but responses are cached
// in Redis based on the token.
//
// Currently only GitHub and GitLab user added external services are supported,
// other code hosts will simply return an empty slice
func GrantedScopes(ctx context.Context, cache ScopeCache, svc *types.ExternalService) ([]string, error) {
	if svc.NamespaceUserID == 0 || (svc.Kind != extsvc.KindGitHub && svc.Kind != extsvc.KindGitLab) {
		return nil, nil
	}
	src, err := NewSource(svc, nil)
	if err != nil {
		return nil, errors.Wrap(err, "creating source")
	}
	switch v := src.(type) {
	case *GithubSource:
		// Cached path
		token := v.config.Token
		if token == "" {
			return nil, errors.New("missing token")
		}
		key, err := hashToken(token)
		if err != nil {
			return nil, err
		}
		if result, ok := cache.Get(key); ok && len(result) > 0 {
			return strings.Split(string(result), ","), nil
		}

		// Slow path
		src, err := NewGithubSource(svc, nil)
		if err != nil {
			return nil, errors.Wrap(err, "creating source")
		}
		scopes, err := src.v3Client.GetAuthenticatedUserOAuthScopes(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "getting scopes")
		}
		cache.Set(key, []byte(strings.Join(scopes, ",")))
		return scopes, nil

	case *GitLabSource:
		// Cached path
		token := v.config.Token
		if v.config.TokenType != "oauth" {
			return nil, errors.New("not an oauth token")
		}
		if token == "" {
			return nil, errors.New("missing token")
		}
		key, err := hashToken(token)
		if err != nil {
			return nil, err
		}
		if result, ok := cache.Get(key); ok && len(result) > 0 {
			return strings.Split(string(result), ","), nil
		}

		// Slow path
		src, err := NewGitLabSource(svc, nil)
		if err != nil {
			return nil, errors.Wrap(err, "creating source")
		}
		scopes, err := src.client.GetAuthenticatedUserOAuthScopes(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "getting scopes")
		}
		cache.Set(key, []byte(strings.Join(scopes, ",")))
		return scopes, nil
	default:
		return nil, errors.Errorf("unsupported config type: %T", v)
	}
}

func hashToken(token string) (string, error) {
	h := fnv.New32()
	_, err := h.Write([]byte(token))
	if err != nil {
		return "", errors.Wrap(err, "hashing token")
	}
	b := h.Sum(nil)
	return hex.EncodeToString(b), nil
}
