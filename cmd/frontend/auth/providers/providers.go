package providers

import (
	"context"
	"encoding/json"
	"sort"
	"sync"

	"github.com/inconshreveable/log15"
	"github.com/sourcegraph/sourcegraph/schema"
)

// A Provider represents a user authentication provider (which provides functionality related to
// signing in and signing up, user identity, etc.) that is present in the site configuration
// "auth.providers" array.
//
// An authentication provider implementation can have multiple Provider instances. For example, a
// site may support OpenID Connect authentication either via Google Workspace or Okta, each of which
// would be represented by its own Provider instance.
type Provider interface {
	// ConfigID returns the identifier for this provider's config in the auth.providers site
	// configuration array.
	//
	// 🚨 SECURITY: This MUST NOT contain secret information because it is shown to unauthenticated
	// and anonymous clients.
	ConfigID() ConfigID

	// Config is the entry in the site configuration "auth.providers" array that this provider
	// represents.
	//
	// 🚨 SECURITY: This value contains secret information that must not be shown to
	// non-site-admins.
	Config() schema.AuthProviders

	// CachedInfo returns cached information about the provider.
	CachedInfo() *Info

	// Refresh refreshes the provider's information with an external service, if any.
	Refresh(ctx context.Context) error
}

// ConfigID identifies a provider config object in the auth.providers site configuration
// array.
//
// 🚨 SECURITY: This MUST NOT contain secret information because it is shown to unauthenticated and
// anonymous clients.
type ConfigID struct {
	// Type is the type of this auth provider (equal to its "type" property in its entry in the
	// auth.providers array in site configuration).
	Type string

	// ID is an identifier that uniquely represents a provider's config among all other provider
	// configs of the same type.
	//
	// This value MUST NOT be persisted or used to associate accounts with this provider because it
	// can change when any property in this provider's config changes, even when those changes are
	// not material for identification (such as changing the display name).
	//
	// 🚨 SECURITY: This MUST NOT contain secret information because it is shown to unauthenticated
	// and anonymous clients.
	ID string
}

// Info contains information about an authentication provider.
type Info struct {
	// ServiceID identifies the external service that this authentication provider represents. It is
	// a stable identifier.
	ServiceID string

	// ClientID identifies the external service client used when communicating with the external
	// service. It is a stable identifier.
	ClientID string

	// DisplayName is the name to use when displaying the provider in the UI.
	DisplayName string

	// AuthenticationURL is the URL to visit in order to initiate authenticating via this provider.
	//
	// TODO(sqs): Support "return-to" post-authentication-redirect destinations so newly authed
	// users aren't dumped back onto the homepage.
	AuthenticationURL string
}

var (
	// curProviders is a map (package name -> (config string -> Provider)). The first key is the
	// package name under which the provider was registered (this should be unique among
	// packages). The second key is the normalized JSON serialization of Provider.Config().  We keep
	// track of providers by package, so that when a given package updates its set of registered
	// providers, we can easily remove its providers that are no longer present.
	curProviders   = map[string]map[string]Provider{}
	curProvidersMu sync.RWMutex

	MockProviders []Provider
)

// Update updates the set of active authentication provider instances. It replaces the
// current set of Providers under the specified pkgName with the new set.
func Update(pkgName string, providers []Provider) {
	curProvidersMu.Lock()
	defer curProvidersMu.Unlock()

	if providers == nil {
		delete(curProviders, pkgName)
		return
	}

	newPkgProviders := map[string]Provider{}
	for _, p := range providers {
		k, err := json.Marshal(p.Config())
		if err != nil {
			log15.Error("Omitting auth provider (failed to marshal its JSON config)", "error", err, "configID", p.ConfigID())
			continue
		}
		newPkgProviders[string(k)] = p
	}
	curProviders[pkgName] = newPkgProviders
}

// Providers returns the set of currently registered authentication providers. When no providers are
// registered, returns nil (and sign-in is effectively disabled).
func Providers() []Provider {
	if MockProviders != nil {
		return MockProviders
	}

	curProvidersMu.RLock()
	defer curProvidersMu.RUnlock()

	if curProviders == nil {
		return nil
	}

	ct := 0
	for _, pkgProviders := range curProviders {
		ct += len(pkgProviders)
	}
	providers := make([]Provider, 0, ct)
	for _, pkgProviders := range curProviders {
		for _, p := range pkgProviders {
			providers = append(providers, p)
		}
	}

	// Sort the providers to ensure a stable ordering (this is for the UI display order).
	sort.Sort(sortProviders(providers))

	return providers
}

func BuiltinAuthEnabled() bool {
	for _, p := range Providers() {
		if p.Config().Builtin != nil {
			return true
		}
	}
	return false
}

type sortProviders []Provider

func (p sortProviders) Len() int {
	return len(p)
}

// Less puts the builtin provider first and sorts the others alphabetically by type and then ID.
func (p sortProviders) Less(i, j int) bool {
	if p[i].ConfigID().Type == "builtin" && p[j].ConfigID().Type != "builtin" {
		return true
	}
	if p[j].ConfigID().Type == "builtin" && p[i].ConfigID().Type != "builtin" {
		return false
	}
	if p[i].ConfigID().Type != p[j].ConfigID().Type {
		return p[i].ConfigID().Type < p[j].ConfigID().Type
	}
	return p[i].ConfigID().ID < p[j].ConfigID().ID
}

func (p sortProviders) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func GetProviderByConfigID(id ConfigID) Provider {
	if MockProviders != nil {
		for _, p := range MockProviders {
			if p.ConfigID() == id {
				return p
			}
		}
		return nil
	}

	curProvidersMu.RLock()
	defer curProvidersMu.RUnlock()

	for _, pkgProviders := range curProviders {
		for _, p := range pkgProviders {
			if p.ConfigID() == id {
				return p
			}
		}
	}
	return nil
}
