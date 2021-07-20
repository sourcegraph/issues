package githuboauth

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/dghubble/gologin/github"
	"golang.org/x/oauth2"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/envvar"
	"github.com/sourcegraph/sourcegraph/enterprise/cmd/frontend/internal/auth/oauth"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/lazyregexp"
	"github.com/sourcegraph/sourcegraph/schema"
)

const sessionKey = "githuboauth@0"

func parseProvider(p *schema.GitHubAuthProvider, db dbutil.DB, sourceCfg schema.AuthProviders) (provider *oauth.Provider, messages []string) {
	rawURL := p.Url
	if rawURL == "" {
		rawURL = "https://github.com/"
	}
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		messages = append(messages, fmt.Sprintf("Could not parse GitHub URL %q. You will not be able to login via this GitHub instance.", rawURL))
		return nil, messages
	}
	if !validateClientIDAndSecret(p.ClientID) {
		messages = append(messages, "GitHub client ID contains unexpected characters, possibly hidden")
	}
	if !validateClientIDAndSecret(p.ClientSecret) {
		messages = append(messages, "GitHub client secret contains unexpected characters, possibly hidden")
	}
	codeHost := extsvc.NewCodeHost(parsedURL, extsvc.TypeGitHub)

	return oauth.NewProvider(oauth.ProviderOp{
		AuthPrefix: authPrefix,
		OAuth2Config: func(extraScopes ...string) oauth2.Config {
			return oauth2.Config{
				ClientID:     p.ClientID,
				ClientSecret: p.ClientSecret,
				Scopes:       requestedScopes(p, extraScopes),
				Endpoint: oauth2.Endpoint{
					AuthURL:  codeHost.BaseURL.ResolveReference(&url.URL{Path: "/login/oauth/authorize"}).String(),
					TokenURL: codeHost.BaseURL.ResolveReference(&url.URL{Path: "/login/oauth/access_token"}).String(),
				},
			}
		},
		SourceConfig: sourceCfg,
		StateConfig:  getStateConfig(),
		ServiceID:    codeHost.ServiceID,
		ServiceType:  codeHost.ServiceType,
		Login: func(oauth2Cfg oauth2.Config) http.Handler {
			return github.LoginHandler(&oauth2Cfg, nil)
		},
		Callback: func(oauth2Cfg oauth2.Config) http.Handler {
			return github.CallbackHandler(
				&oauth2Cfg,
				oauth.SessionIssuer(&sessionIssuerHelper{
					CodeHost:    codeHost,
					db:          db,
					clientID:    p.ClientID,
					allowSignup: p.AllowSignup,
					allowOrgs:   p.AllowOrgs,
				}, sessionKey),
				http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
					http.Redirect(w, req, "/", http.StatusTemporaryRedirect)
				}),
			)
		},
	}), messages
}

var clientIDSecretValidator = lazyregexp.New("^[a-z0-9]*$")

func validateClientIDAndSecret(clientIDOrSecret string) (valid bool) {
	return clientIDSecretValidator.MatchString(clientIDOrSecret)
}

func requestedScopes(p *schema.GitHubAuthProvider, extraScopes []string) []string {
	scopes := []string{"user:email"}
	if !envvar.SourcegraphDotComMode() {
		scopes = append(scopes, "repo")
	}

	// Needs extra scope to check organization membership
	if len(p.AllowOrgs) > 0 {
		scopes = append(scopes, "read:org")
	}

	// Append extra scopes and ensure there are no duplicates
	for _, s := range extraScopes {
		var found bool
		for _, inner := range scopes {
			if inner == s {
				found = true
				break
			}
		}

		if !found {
			scopes = append(scopes, s)
		}
	}

	return scopes
}
