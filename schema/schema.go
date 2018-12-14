// Code generated by go-jsonschema-compiler. DO NOT EDIT.

package schema

import (
	"encoding/json"
	"errors"
	"fmt"
)

type AWSCodeCommitConnection struct {
	AccessKeyID                 string `json:"accessKeyID"`
	InitialRepositoryEnablement bool   `json:"initialRepositoryEnablement,omitempty"`
	Region                      string `json:"region"`
	RepositoryPathPattern       string `json:"repositoryPathPattern,omitempty"`
	SecretAccessKey             string `json:"secretAccessKey"`
}

// AuthAccessTokens description: Settings for access tokens, which enable external tools to access the Sourcegraph API with the privileges of the user.
type AuthAccessTokens struct {
	Allow string `json:"allow,omitempty"`
}

// AuthProviderCommon description: Common properties for authentication providers.
type AuthProviderCommon struct {
	DisplayName string `json:"displayName,omitempty"`
}
type AuthProviders struct {
	Builtin       *BuiltinAuthProvider
	Saml          *SAMLAuthProvider
	Openidconnect *OpenIDConnectAuthProvider
	HttpHeader    *HTTPHeaderAuthProvider
	Github        *GitHubAuthProvider
	Gitlab        *GitLabAuthProvider
}

func (v AuthProviders) MarshalJSON() ([]byte, error) {
	if v.Builtin != nil {
		return json.Marshal(v.Builtin)
	}
	if v.Saml != nil {
		return json.Marshal(v.Saml)
	}
	if v.Openidconnect != nil {
		return json.Marshal(v.Openidconnect)
	}
	if v.HttpHeader != nil {
		return json.Marshal(v.HttpHeader)
	}
	if v.Github != nil {
		return json.Marshal(v.Github)
	}
	if v.Gitlab != nil {
		return json.Marshal(v.Gitlab)
	}
	return nil, errors.New("tagged union type must have exactly 1 non-nil field value")
}
func (v *AuthProviders) UnmarshalJSON(data []byte) error {
	var d struct {
		DiscriminantProperty string `json:"type"`
	}
	if err := json.Unmarshal(data, &d); err != nil {
		return err
	}
	switch d.DiscriminantProperty {
	case "builtin":
		return json.Unmarshal(data, &v.Builtin)
	case "github":
		return json.Unmarshal(data, &v.Github)
	case "gitlab":
		return json.Unmarshal(data, &v.Gitlab)
	case "http-header":
		return json.Unmarshal(data, &v.HttpHeader)
	case "openidconnect":
		return json.Unmarshal(data, &v.Openidconnect)
	case "saml":
		return json.Unmarshal(data, &v.Saml)
	}
	return fmt.Errorf("tagged union type must have a %q property whose value is one of %s", "type", []string{"builtin", "saml", "openidconnect", "http-header", "github", "gitlab"})
}

// AuthnProvider description: Identifies the authentication provider to use to identify users to GitLab.
type AuthnProvider struct {
	ConfigID       string `json:"configID"`
	GitlabProvider string `json:"gitlabProvider"`
	Type           string `json:"type"`
}
type BitbucketServerConnection struct {
	Certificate                 string `json:"certificate,omitempty"`
	ExcludePersonalRepositories bool   `json:"excludePersonalRepositories,omitempty"`
	GitURLType                  string `json:"gitURLType,omitempty"`
	InitialRepositoryEnablement bool   `json:"initialRepositoryEnablement,omitempty"`
	Password                    string `json:"password,omitempty"`
	RepositoryPathPattern       string `json:"repositoryPathPattern,omitempty"`
	Token                       string `json:"token,omitempty"`
	Url                         string `json:"url"`
	Username                    string `json:"username,omitempty"`
}

// BuiltinAuthProvider description: Configures the builtin username-password authentication provider.
type BuiltinAuthProvider struct {
	AllowSignup bool   `json:"allowSignup,omitempty"`
	Type        string `json:"type"`
}

// CloneURLToRepositoryName description: Describes a mapping from clone URL to repository name. The `from` field contains a regular expression with named capturing groups. The `to` field contains a template string that references capturing group names. For instance, if `from` is "^../(?P<name>\w+)$" and `to` is "github.com/user/{name}", the clone URL "../myRepository" would be mapped to the repository name "github.com/user/myRepository".
type CloneURLToRepositoryName struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// CriticalConfiguration description: Critical configuration for a Sourcegraph site.
type CriticalConfiguration struct {
	AuthProviders               []AuthProviders     `json:"auth.providers,omitempty"`
	AuthPublic                  bool                `json:"auth.public,omitempty"`
	AuthSessionExpiry           string              `json:"auth.sessionExpiry,omitempty"`
	AuthUserOrgMap              map[string][]string `json:"auth.userOrgMap,omitempty"`
	ExternalURL                 string              `json:"externalURL,omitempty"`
	HtmlBodyBottom              string              `json:"htmlBodyBottom,omitempty"`
	HtmlBodyTop                 string              `json:"htmlBodyTop,omitempty"`
	HtmlHeadBottom              string              `json:"htmlHeadBottom,omitempty"`
	HtmlHeadTop                 string              `json:"htmlHeadTop,omitempty"`
	HttpStrictTransportSecurity interface{}         `json:"httpStrictTransportSecurity,omitempty"`
	HttpToHttpsRedirect         interface{}         `json:"httpToHttpsRedirect,omitempty"`
	LicenseKey                  string              `json:"licenseKey,omitempty"`
	LightstepAccessToken        string              `json:"lightstepAccessToken,omitempty"`
	LightstepProject            string              `json:"lightstepProject,omitempty"`
	Log                         *Log                `json:"log,omitempty"`
	TlsLetsencrypt              string              `json:"tls.letsencrypt,omitempty"`
	TlsCert                     string              `json:"tlsCert,omitempty"`
	TlsKey                      string              `json:"tlsKey,omitempty"`
	UpdateChannel               string              `json:"update.channel,omitempty"`
	UseJaeger                   bool                `json:"useJaeger,omitempty"`
}

// Discussions description: Configures Sourcegraph code discussions.
type Discussions struct {
	AbuseEmails     []string `json:"abuseEmails,omitempty"`
	AbuseProtection bool     `json:"abuseProtection,omitempty"`
}

// ExperimentalFeatures description: Experimental features to enable or disable. Features that are now enabled by default are marked as deprecated.
type ExperimentalFeatures struct {
	CanonicalURLRedirect string `json:"canonicalURLRedirect,omitempty"`
	Discussions          string `json:"discussions,omitempty"`
	ExternalServices     string `json:"externalServices,omitempty"`
	GithubAuth           bool   `json:"githubAuth,omitempty"`
	GitlabAuth           bool   `json:"gitlabAuth,omitempty"`
	UpdateScheduler2     string `json:"updateScheduler2,omitempty"`
}

// Extensions description: Configures Sourcegraph extensions.
type Extensions struct {
	AllowRemoteExtensions []string    `json:"allowRemoteExtensions,omitempty"`
	Disabled              *bool       `json:"disabled,omitempty"`
	RemoteRegistry        interface{} `json:"remoteRegistry,omitempty"`
}

// GitHubAuthProvider description: Configures the GitHub (or GitHub Enterprise) OAuth authentication provider for SSO. In addition to specifying this configuration object, you must also create a OAuth App on your GitHub instance: https://developer.github.com/apps/building-oauth-apps/creating-an-oauth-app/. When a user signs into Sourcegraph or links their GitHub account to their existing Sourcegraph account, GitHub will prompt the user for the repo scope.
type GitHubAuthProvider struct {
	AllowSignup  bool   `json:"allowSignup,omitempty"`
	ClientID     string `json:"clientID"`
	ClientSecret string `json:"clientSecret"`
	DisplayName  string `json:"displayName,omitempty"`
	Type         string `json:"type"`
	Url          string `json:"url,omitempty"`
}

// GitHubAuthorization description: If non-null, enforces GitHub repository permissions. This requires that there is an item in the `auth.providers` field of type "github" with the same `url` field as specified in this `GitHubConnection`.
type GitHubAuthorization struct {
	Ttl string `json:"ttl,omitempty"`
}
type GitHubConnection struct {
	Authorization               *GitHubAuthorization `json:"authorization,omitempty"`
	Certificate                 string               `json:"certificate,omitempty"`
	GitURLType                  string               `json:"gitURLType,omitempty"`
	InitialRepositoryEnablement bool                 `json:"initialRepositoryEnablement,omitempty"`
	Repos                       []string             `json:"repos,omitempty"`
	RepositoryPathPattern       string               `json:"repositoryPathPattern,omitempty"`
	RepositoryQuery             []string             `json:"repositoryQuery,omitempty"`
	Token                       string               `json:"token"`
	Url                         string               `json:"url"`
}

// GitLabAuthProvider description: Configures the GitLab OAuth authentication provider for SSO. In addition to specifying this configuration object, you must also create a OAuth App on your GitLab instance: https://docs.gitlab.com/ee/integration/oauth_provider.html. The application should have `api` and `read_user` scopes and the callback URL set to the concatenation of your Sourcegraph instance URL and "/.auth/gitlab/callback".
type GitLabAuthProvider struct {
	ClientID     string `json:"clientID"`
	ClientSecret string `json:"clientSecret"`
	DisplayName  string `json:"displayName,omitempty"`
	Type         string `json:"type"`
	Url          string `json:"url,omitempty"`
}

// GitLabAuthorization description: If non-null, enforces GitLab repository permissions. This requires that the value of `token` be an access token with "sudo" and "api" scopes.
type GitLabAuthorization struct {
	AuthnProvider AuthnProvider `json:"authnProvider"`
	Ttl           string        `json:"ttl,omitempty"`
}
type GitLabConnection struct {
	Authorization               *GitLabAuthorization `json:"authorization,omitempty"`
	Certificate                 string               `json:"certificate,omitempty"`
	GitURLType                  string               `json:"gitURLType,omitempty"`
	InitialRepositoryEnablement bool                 `json:"initialRepositoryEnablement,omitempty"`
	ProjectQuery                []string             `json:"projectQuery,omitempty"`
	RepositoryPathPattern       string               `json:"repositoryPathPattern,omitempty"`
	Token                       string               `json:"token"`
	Url                         string               `json:"url"`
}
type GitoliteConnection struct {
	Blacklist                  string `json:"blacklist,omitempty"`
	Host                       string `json:"host"`
	PhabricatorMetadataCommand string `json:"phabricatorMetadataCommand,omitempty"`
	Prefix                     string `json:"prefix"`
}

// HTTPHeaderAuthProvider description: Configures the HTTP header authentication provider (which authenticates users by consulting an HTTP request header set by an authentication proxy such as https://github.com/bitly/oauth2_proxy).
type HTTPHeaderAuthProvider struct {
	StripUsernameHeaderPrefix string `json:"stripUsernameHeaderPrefix,omitempty"`
	Type                      string `json:"type"`
	UsernameHeader            string `json:"usernameHeader"`
}

// IMAPServerConfig description: Optional. The IMAP server used to retrieve emails (such as code discussion reply emails).
type IMAPServerConfig struct {
	Host     string `json:"host"`
	Password string `json:"password,omitempty"`
	Port     int    `json:"port"`
	Username string `json:"username,omitempty"`
}

// Log description: Configuration for logging and alerting, including to external services.
type Log struct {
	Sentry *Sentry `json:"sentry,omitempty"`
}

// OpenIDConnectAuthProvider description: Configures the OpenID Connect authentication provider for SSO.
type OpenIDConnectAuthProvider struct {
	ClientID           string `json:"clientID"`
	ClientSecret       string `json:"clientSecret"`
	ConfigID           string `json:"configID,omitempty"`
	DisplayName        string `json:"displayName,omitempty"`
	Issuer             string `json:"issuer"`
	RequireEmailDomain string `json:"requireEmailDomain,omitempty"`
	Type               string `json:"type"`
}

// ParentSourcegraph description: URL to fetch unreachable repository details from. Defaults to "https://sourcegraph.com"
type ParentSourcegraph struct {
	Url string `json:"url,omitempty"`
}
type PhabricatorConnection struct {
	Repos []*Repos `json:"repos,omitempty"`
	Token string   `json:"token,omitempty"`
	Url   string   `json:"url,omitempty"`
}
type Repos struct {
	Callsign string `json:"callsign"`
	Path     string `json:"path"`
}
type ReviewBoard struct {
	Url string `json:"url,omitempty"`
}

// SAMLAuthProvider description: Configures the SAML authentication provider for SSO.
//
// Note: if you are using IdP-initiated login, you must have *at most one* SAMLAuthProvider in the `auth.providers` array.
type SAMLAuthProvider struct {
	ConfigID                                 string `json:"configID,omitempty"`
	DisplayName                              string `json:"displayName,omitempty"`
	IdentityProviderMetadata                 string `json:"identityProviderMetadata,omitempty"`
	IdentityProviderMetadataURL              string `json:"identityProviderMetadataURL,omitempty"`
	InsecureSkipAssertionSignatureValidation bool   `json:"insecureSkipAssertionSignatureValidation,omitempty"`
	NameIDFormat                             string `json:"nameIDFormat,omitempty"`
	ServiceProviderCertificate               string `json:"serviceProviderCertificate,omitempty"`
	ServiceProviderIssuer                    string `json:"serviceProviderIssuer,omitempty"`
	ServiceProviderPrivateKey                string `json:"serviceProviderPrivateKey,omitempty"`
	SignRequests                             *bool  `json:"signRequests,omitempty"`
	Type                                     string `json:"type"`
}

// SMTPServerConfig description: The SMTP server used to send transactional emails (such as email verifications, reset-password emails, and notifications).
type SMTPServerConfig struct {
	Authentication string `json:"authentication"`
	Domain         string `json:"domain,omitempty"`
	Host           string `json:"host"`
	Password       string `json:"password,omitempty"`
	Port           int    `json:"port"`
	Username       string `json:"username,omitempty"`
}
type SearchSavedQueries struct {
	Description    string `json:"description"`
	Key            string `json:"key"`
	Notify         bool   `json:"notify,omitempty"`
	NotifySlack    bool   `json:"notifySlack,omitempty"`
	Query          string `json:"query"`
	ShowOnHomepage bool   `json:"showOnHomepage,omitempty"`
}
type SearchScope struct {
	Description string `json:"description,omitempty"`
	Id          string `json:"id,omitempty"`
	Name        string `json:"name"`
	Value       string `json:"value"`
}

// Sentry description: Configuration for Sentry
type Sentry struct {
	Dsn string `json:"dsn,omitempty"`
}

// Settings description: Configuration settings for users and organizations on Sourcegraph.
type Settings struct {
	Extensions             map[string]bool           `json:"extensions,omitempty"`
	Motd                   []string                  `json:"motd,omitempty"`
	NotificationsSlack     *SlackNotificationsConfig `json:"notifications.slack,omitempty"`
	SearchRepositoryGroups map[string][]string       `json:"search.repositoryGroups,omitempty"`
	SearchSavedQueries     []*SearchSavedQueries     `json:"search.savedQueries,omitempty"`
	SearchScopes           []*SearchScope            `json:"search.scopes,omitempty"`
}

// SiteConfiguration description: Configuration for a Sourcegraph site.
type SiteConfiguration struct {
	AuthAccessTokens                  *AuthAccessTokens           `json:"auth.accessTokens,omitempty"`
	AuthDisableAccessTokens           bool                        `json:"auth.disableAccessTokens,omitempty"`
	CorsOrigin                        string                      `json:"corsOrigin,omitempty"`
	DisableAutoGitUpdates             bool                        `json:"disableAutoGitUpdates,omitempty"`
	DisableBrowserExtension           bool                        `json:"disableBrowserExtension,omitempty"`
	DisableBuiltInSearches            bool                        `json:"disableBuiltInSearches,omitempty"`
	DisablePublicRepoRedirects        bool                        `json:"disablePublicRepoRedirects,omitempty"`
	Discussions                       *Discussions                `json:"discussions,omitempty"`
	DontIncludeSymbolResultsByDefault bool                        `json:"dontIncludeSymbolResultsByDefault,omitempty"`
	EmailAddress                      string                      `json:"email.address,omitempty"`
	EmailImap                         *IMAPServerConfig           `json:"email.imap,omitempty"`
	EmailSmtp                         *SMTPServerConfig           `json:"email.smtp,omitempty"`
	ExperimentalFeatures              *ExperimentalFeatures       `json:"experimentalFeatures,omitempty"`
	Extensions                        *Extensions                 `json:"extensions,omitempty"`
	GitCloneURLToRepositoryName       []*CloneURLToRepositoryName `json:"git.cloneURLToRepositoryName,omitempty"`
	GitMaxConcurrentClones            int                         `json:"gitMaxConcurrentClones,omitempty"`
	GithubClientID                    string                      `json:"githubClientID,omitempty"`
	GithubClientSecret                string                      `json:"githubClientSecret,omitempty"`
	MaxReposToSearch                  int                         `json:"maxReposToSearch,omitempty"`
	ParentSourcegraph                 *ParentSourcegraph          `json:"parentSourcegraph,omitempty"`
	RepoListUpdateInterval            int                         `json:"repoListUpdateInterval,omitempty"`
	ReviewBoard                       []*ReviewBoard              `json:"reviewBoard,omitempty"`
	SearchIndexEnabled                *bool                       `json:"search.index.enabled,omitempty"`
}

// SlackNotificationsConfig description: Configuration for sending notifications to Slack.
type SlackNotificationsConfig struct {
	WebhookURL string `json:"webhookURL"`
}
