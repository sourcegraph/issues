package github

import (
	"fmt"
	"net/url"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/internal/authz"
	"github.com/sourcegraph/sourcegraph/schema"
)

// NewAuthzProviders returns the set of GitHub authz providers derived from the connections.
// It also returns any validation problems with the config, separating these into "serious problems" and
// "warnings". "Serious problems" are those that should make Sourcegraph set authz.allowAccessByDefault
// to false. "Warnings" are all other validation problems.
func NewAuthzProviders(
	conns []*types.GitHubConnection,
) (ps []authz.Provider, problems []string, warnings []string) {
	// Authorization (i.e., permissions) providers
	for _, c := range conns {
		p, err := newAuthzProvider(c.URN, c.Authorization, c.Url, c.Token)
		if err != nil {
			problems = append(problems, err.Error())
		} else if p != nil {
			ps = append(ps, p)
		}
	}

	for _, p := range ps {
		for _, problem := range p.Validate() {
			warnings = append(warnings, fmt.Sprintf("GitHub config for %s was invalid: %s", p.ServiceID(), problem))
		}
	}

	return ps, problems, warnings
}

func newAuthzProvider(urn string, a *schema.GitHubAuthorization, instanceURL, token string) (authz.Provider, error) {
	if a == nil {
		return nil, nil
	}

	ghURL, err := url.Parse(instanceURL)
	if err != nil {
		return nil, fmt.Errorf("Could not parse URL for GitHub instance %q: %s", instanceURL, err)
	}

	return NewProvider(urn, ghURL, token, nil), nil
}

// ValidateGitHubAuthz validates the authorization fields of the given GitHub external
// service config.
func ValidateAuthz(cfg *schema.GitHubConnection) error {
	_, err := newAuthzProvider("", cfg.Authorization, cfg.Url, cfg.Token)
	return err
}
