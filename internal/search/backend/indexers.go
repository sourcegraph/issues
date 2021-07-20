package backend

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/cockroachdb/errors"
)

// EndpointMap is the subset of endpoint.Map (consistent hashmap) methods we
// use. Declared as an interface for testing.
type EndpointMap interface {
	// Endpoints returns a set of all addresses. Do not modify the returned value.
	Endpoints() (map[string]struct{}, error)
	// GetMany returns the endpoint for each key. (consistent hashing).
	GetMany(...string) ([]string, error)
}

// Indexers provides methods over the set of indexed-search servers in a
// Sourcegraph cluster.
type Indexers struct {
	// Map is the desired mapping from repository names to endpoints.
	Map EndpointMap

	// Indexed returns a set of repository names currently indexed on
	// endpoint. If indexed fails, it is expected to return an empty set.
	Indexed func(ctx context.Context, endpoint string) map[string]struct{}
}

// ReposSubset returns the subset of repoNames that hostname should index.
//
// ReposSubset reuses the underlying array of repoNames.
//
// indexed is the set of repositories currently indexed by hostname.
//
// An error is returned if hostname is not part of the Indexers endpoints.
func (c *Indexers) ReposSubset(ctx context.Context, hostname string, indexed map[string]struct{}, repoNames []string) ([]string, error) {
	if !c.Enabled() {
		return repoNames, nil
	}

	eps, err := c.Map.Endpoints()
	if err != nil {
		return nil, err
	}

	endpoint, err := findEndpoint(eps, hostname)
	if err != nil {
		return nil, err
	}

	assigned, err := c.Map.GetMany(repoNames...)
	if err != nil {
		return nil, err
	}

	// Rebalancing: Other contains all repositories endpoint has indexed which
	// it should drop. We will only drop them if the assigned endpoint has
	// indexed it. This is to prevent dropping a computed index until
	// rebalancing is finished.
	other := map[string][]string{}

	subset := repoNames[:0]
	for i, name := range repoNames {
		if assigned[i] == endpoint {
			subset = append(subset, name)
		} else if _, ok := indexed[name]; ok {
			other[assigned[i]] = append(other[assigned[i]], name)
		}
	}

	// Only include repos from other if the assigned endpoint has not yet
	// indexed the repository.
	for assignedEndpoint, repoNames := range other {
		drop := c.Indexed(ctx, assignedEndpoint)
		for _, name := range repoNames {
			if _, ok := drop[name]; !ok {
				subset = append(subset, name)
			}
		}
	}

	return subset, nil
}

// Enabled returns true if this feature is enabled. At first horizontal
// sharding will be disabled, if so the functions here fallback to single
// shard behaviour.
func (c *Indexers) Enabled() bool {
	return c.Map != nil
}

// findEndpoint returns the endpoint in eps which matches hostname.
func findEndpoint(eps map[string]struct{}, hostname string) (string, error) {
	// The hostname can be a less qualified hostname. For example in k8s
	// $HOSTNAME will be "indexed-search-0", but to access the pod you will
	// need to specify the endpoint address
	// "indexed-search-0.indexed-search".
	//
	// Additionally an endpoint can also specify a port, which a hostname
	// won't.
	//
	// Given this looser matching, we ensure we don't match more than one
	// endpoint.
	endpoint := ""
	for ep := range eps {
		if !strings.HasPrefix(ep, hostname) {
			continue
		}
		if len(hostname) < len(ep) {
			if c := ep[len(hostname)]; c != '.' && c != ':' {
				continue
			}
		}

		if endpoint != "" {
			return "", errors.Errorf("hostname %q matches multiple in %s", hostname, endpointsString(eps))
		}
		endpoint = ep
	}
	if endpoint != "" {
		return endpoint, nil
	}

	return "", errors.Errorf("hostname %q not found in %s", hostname, endpointsString(eps))
}

// endpointsString creates a user readable String for an endpoint map.
func endpointsString(m map[string]struct{}) string {
	eps := make([]string, 0, len(m))
	for k := range m {
		eps = append(eps, k)
	}
	sort.Strings(eps)

	var b strings.Builder
	b.WriteString("Endpoints{")
	for i, k := range eps {
		if i != 0 {
			b.WriteByte(' ')
		}
		_, _ = fmt.Fprintf(&b, "%q", k)
	}
	b.WriteByte('}')
	return b.String()
}
