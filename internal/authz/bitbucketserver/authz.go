package bitbucketserver

import (
	"fmt"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/internal/authz"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/bitbucketserver"
	"github.com/sourcegraph/sourcegraph/schema"
)

// NewAuthzProviders returns the set of Bitbucket Server authz providers derived from the connections.
// It also returns any validation problems with the config, separating these into "serious problems" and
// "warnings". "Serious problems" are those that should make Sourcegraph set authz.allowAccessByDefault
// to false. "Warnings" are all other validation problems.
func NewAuthzProviders(
	conns []*types.BitbucketServerConnection,
) (ps []authz.Provider, problems []string, warnings []string) {
	// Authorization (i.e., permissions) providers
	for _, c := range conns {
		pluginPerm := conf.BitbucketServerPluginPerm() || (c.Plugin != nil && c.Plugin.Permissions == "enabled")
		p, err := newAuthzProvider(c, pluginPerm)
		if err != nil {
			problems = append(problems, err.Error())
		} else if p != nil {
			ps = append(ps, p)
		}
	}

	for _, p := range ps {
		for _, problem := range p.Validate() {
			warnings = append(warnings, fmt.Sprintf("BitbucketServer config for %s was invalid: %s", p.ServiceID(), problem))
		}
	}

	return ps, problems, warnings
}

func newAuthzProvider(
	c *types.BitbucketServerConnection,
	pluginPerm bool,
) (authz.Provider, error) {
	if c.Authorization == nil {
		return nil, nil
	}

	errs := new(multierror.Error)

	cli, err := bitbucketserver.NewClient(c.BitbucketServerConnection, nil)
	if err != nil {
		errs = multierror.Append(errs, err)
		return nil, errs.ErrorOrNil()
	}

	var p authz.Provider
	switch idp := c.Authorization.IdentityProvider; {
	case idp.Username != nil:
		p = NewProvider(cli, c.URN, pluginPerm)
	default:
		errs = multierror.Append(errs, errors.Errorf("No identityProvider was specified"))
	}

	return p, errs.ErrorOrNil()
}

// ValidateAuthz validates the authorization fields of the given BitbucketServer external
// service config.
func ValidateAuthz(c *schema.BitbucketServerConnection) error {
	_, err := newAuthzProvider(&types.BitbucketServerConnection{BitbucketServerConnection: c}, false)
	return err
}
