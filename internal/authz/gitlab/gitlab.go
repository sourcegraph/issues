package gitlab

import (
	"fmt"
	"net/url"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/auth/providers"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/internal/authz"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/schema"
)

// NewAuthzProviders returns the set of GitLab authz providers derived from the connections.
// It also returns any validation problems with the config, separating these into "serious problems" and
// "warnings". "Serious problems" are those that should make Sourcegraph set authz.allowAccessByDefault
// to false. "Warnings" are all other validation problems.
func NewAuthzProviders(
	cfg *conf.Unified,
	conns []*types.GitLabConnection,
) (ps []authz.Provider, problems []string, warnings []string) {
	// Authorization (i.e., permissions) providers
	for _, c := range conns {
		p, err := newAuthzProvider(c.URN, c.Authorization, c.Url, c.Token, cfg.AuthProviders)
		if err != nil {
			problems = append(problems, err.Error())
		} else if p != nil {
			ps = append(ps, p)
		}
	}
	for _, p := range ps {
		for _, problem := range p.Validate() {
			warnings = append(warnings, fmt.Sprintf("GitLab config for %s was invalid: %s", p.ServiceID(), problem))
		}
	}

	return ps, problems, warnings
}

func newAuthzProvider(urn string, a *schema.GitLabAuthorization, instanceURL, token string, ps []schema.AuthProviders) (authz.Provider, error) {
	if a == nil {
		return nil, nil
	}

	glURL, err := url.Parse(instanceURL)
	if err != nil {
		return nil, fmt.Errorf("Could not parse URL for GitLab instance %q: %s", instanceURL, err)
	}

	switch idp := a.IdentityProvider; {
	case idp.Oauth != nil:
		// Check that there is a GitLab authn provider corresponding to this GitLab instance
		foundAuthProvider := false
		for _, authnProvider := range ps {
			if authnProvider.Gitlab == nil {
				continue
			}
			authnURL := authnProvider.Gitlab.Url
			if authnURL == "" {
				authnURL = "https://gitlab.com"
			}
			authProviderURL, err := url.Parse(authnURL)
			if err != nil {
				// Ignore the error here, because the authn provider is responsible for its own validation
				continue
			}
			if authProviderURL.Hostname() == glURL.Hostname() {
				foundAuthProvider = true
				break
			}
		}
		if !foundAuthProvider {
			return nil, fmt.Errorf("Did not find authentication provider matching %q. Check the [**site configuration**](/site-admin/configuration) to verify an entry in [`auth.providers`](https://docs.sourcegraph.com/admin/auth) exists for %s.", instanceURL, instanceURL)
		}

		return NewOAuthProvider(OAuthProviderOp{
			URN:     urn,
			BaseURL: glURL,
			Token:   token,
		}), nil
	case idp.Username != nil:
		return NewSudoProvider(SudoProviderOp{
			URN:               urn,
			BaseURL:           glURL,
			SudoToken:         token,
			UseNativeUsername: true,
		}), nil
	case idp.External != nil:
		ext := idp.External
		for _, authProvider := range ps {
			saml := authProvider.Saml
			foundMatchingSAML := saml != nil && saml.ConfigID == ext.AuthProviderID && ext.AuthProviderType == saml.Type
			oidc := authProvider.Openidconnect
			foundMatchingOIDC := oidc != nil && oidc.ConfigID == ext.AuthProviderID && ext.AuthProviderType == oidc.Type
			if foundMatchingSAML || foundMatchingOIDC {
				return NewSudoProvider(SudoProviderOp{
					URN:     urn,
					BaseURL: glURL,
					AuthnConfigID: providers.ConfigID{
						Type: ext.AuthProviderType,
						ID:   ext.AuthProviderID,
					},
					GitLabProvider:    ext.GitlabProvider,
					SudoToken:         token,
					UseNativeUsername: false,
				}), nil
			}
		}
		return nil, fmt.Errorf("Did not find authentication provider matching type %s and configID %s. Check the [**site configuration**](/site-admin/configuration) to verify that an entry in [`auth.providers`](https://docs.sourcegraph.com/admin/auth) matches the type and configID.", ext.AuthProviderType, ext.AuthProviderID)
	default:
		return nil, fmt.Errorf("No identityProvider was specified")
	}
}

// NewOAuthProvider is a mockable constructor for new OAuthProvider instances.
var NewOAuthProvider = func(op OAuthProviderOp) authz.Provider {
	return newOAuthProvider(op, nil)
}

// NewSudoProvider is a mockable constructor for new SudoProvider instances.
var NewSudoProvider = func(op SudoProviderOp) authz.Provider {
	return newSudoProvider(op, nil)
}

// ValidateAuthz validates the authorization fields of the given GitLab external
// service config.
func ValidateAuthz(cfg *schema.GitLabConnection, ps []schema.AuthProviders) error {
	_, err := newAuthzProvider("", cfg.Authorization, cfg.Url, cfg.Token, ps)
	return err
}
