# User authentication (SSO)

Sourcegraph supports the following ways for users to sign in:

- [Builtin](#builtin-password-authentication)
- [GitHub OAuth](#github)
- [GitLab OAuth](#gitlab)
- [OpenID Connect](#openid-connect) (including [Google accounts on Google Workspace](#google-workspace-google-accounts))
- [SAML](saml/index.md)
- [HTTP authentication proxies](#http-authentication-proxies)

The authentication provider is configured in the [`auth.providers`](../config/site_config.md#authentication-providers) site configuration option.

### Guidance

If you are unsure which auth provider is right for you, we recommend applying the following rules in
order:

- If you have no external identity providers (i.e., not SSO) or are just trying to spin Sourcegraph
  up as quickly as possible to try, use [`builtin`](#builtin-password-authentication) authentication. You can
  always change the auth configuration later, and user identities from external providers will be
  linked automatically to existing Sourcegraph accounts using verified email addresses.
- If you are deploying Sourcegraph behind a HTTP authentication proxy service, use the
  [`http-header`](#http-authentication-proxies) provider type. The proxy service should handle
  authentication and session management and, in turn, set a HTTP header that indicates the user
  identity to Sourcegraph.
- If you are configuring Sourcegraph to index a GitHub or GitLab instance, we recommend using the
  OAuth provider for that code host. This applies even if the code host itself uses an external
  identity provider (e.g., SAML, OpenID Connect, LDAP, etc.). Sourcegraph will redirect to your code
  host on sign-in and the code host will perform the required sign-in flow before redirecting to
  Sourcegraph on success.
- If you are using an identity provider that supports SAML, use the [SAML auth provider](#saml).
- If you are using an identity provider that supports OpenID Connect (including Google accounts),
  use the [OpenID Connect provider](#openid-connect).
- If you wish to use LDAP and cannot use the GitHub/GitLab OAuth provider as described above, or if
  you wish to use another authentication mechanism that is not yet supported, please [contact
  us](https://github.com/sourcegraph/sourcegraph/issues/new?template=feature_request.md) (we respond
  promptly).

Most users will use only one auth provider, but you can use multiple auth providers if desired to
enable sign-in via multiple services. Identities from different providers will be mapped to a
Sourcegraph user by comparing the user's verified email address to the email address from the
external identity provider.

## Builtin password authentication

The [`builtin` auth provider](../config/site_config.md#builtin-password-authentication) manages user accounts internally in its own database. It supports user signup, login, and password reset (via email if configured, or else via a site admin).

Password reset links expire after 4 hours.

Site configuration example:

```json
{
  // ...,
  "auth.providers": [{ "type": "builtin", "allowSignup": true }]
}
```

## GitHub

[Create a GitHub OAuth
application](https://developer.github.com/apps/building-oauth-apps/creating-an-oauth-app/) (if using
GitHub Enterprise, create one on your instance, not GitHub.com). Set the following values, replacing
`sourcegraph.example.com` with the IP or hostname of your Sourcegraph instance:

- Homepage URL: `https://sourcegraph.example.com`
- Authorization callback URL: `https://sourcegraph.example.com/.auth/github/callback`

> NOTE: If you want to enable repository permissions, you should grant your OAuth app permission to
> your GitHub organization(s). You can do that either by creating the OAuth app under your GitHub
> organization (rather than your personal account) or by [following these
> instructions](https://help.github.com/articles/approving-oauth-apps-for-your-organization/).

Then add the following lines to your site configuration:

```json
{
  // ...
  "auth.providers": [
    {
      "type": "github",
      "url": "https://github.example.com",  // URL of your GitHub instance; can leave empty for github.com
      "displayName": "GitHub",
      "clientID": "replace-with-the-oauth-client-id",
      "clientSecret": "replace-with-the-oauth-client-secret",
      "allowSignup": false,  // Set to true to enable anyone with a GitHub account to sign up without invitation
      "allowOrgs": ["your-org-name"] // Restrict logins to members of these orgs.
    }
  ]
}
```

Replace the `clientID` and `clientSecret` values with the values from your GitHub OAuth app
configuration.

Leave the `url` field empty for GitHub.com.

Set `allowSignup` to `true` to enable anyone with a GitHub account to sign up without invitation
(typically done only for GitHub Enterprise). If `allowSignup` is `false`, a user can sign in through
GitHub only if an account with the same verified email already exists. If none exists, a site admin
must create one explicitly.

The `allowOrgs` fields restricts logins to members of the specified GitHub organizations. Existing user sessions are **not invalidated**. Only new logins after this setting is changed are affected.

Once you've configured GitHub as a sign-on provider, you may also want to [add GitHub repositories to Sourcegraph](../external_service/github.md#repository-syncing).

### Troubleshooting

Setting the env var `INSECURE_OAUTH2_LOG_TRACES=1` on the `sourcegraph/server` Docker container (or the `sourcegraph-frontend` deployment if you're using Kubernetes) causes all OAuth2 requests and responses to be logged.

## GitLab

[Create a GitLab OAuth application](https://docs.gitlab.com/ee/integration/oauth_provider.html). Set
the following values, replacing `sourcegraph.example.com` with the IP or hostname of your
Sourcegraph instance:

- Authorization callback URL: `https://sourcegraph.example.com/.auth/gitlab/callback`
- Scopes: `api`, `read_user`

Then add the following lines to your site configuration:

```json
{
    // ...
    "auth.providers": [
      {
        "type": "gitlab",
        "displayName": "GitLab",
        "clientID": "replace-with-the-oauth-application-id",
        "clientSecret": "replace-with-the-oauth-secret",
        "url": "https://gitlab.example.com"
      }
    ]
```

Replace the `clientID` and `clientSecret` values with the values from your GitLab OAuth app
configuration.

Once you've configured GitLab as a sign-on provider, you may also want to [add GitLab repositories
to Sourcegraph](../external_service/gitlab.md#repository-syncing).

### Troubleshooting

Setting the env var `INSECURE_OAUTH2_LOG_TRACES=1` on the `sourcegraph/server` Docker container (or the `sourcegraph-frontend` deployment if you're using Kubernetes) causes all OAuth2 requests and responses to be logged.

## OpenID Connect

The [`openidconnect` auth provider](../config/site_config.md#openid-connect-including-google-workspace) authenticates users via OpenID Connect, which is supported by many external services, including:

- [Google Workspace (Google accounts)](#google-workspace-google-accounts)
- [Okta](https://developer.okta.com/docs/api/resources/oidc.html)
- [Ping Identity](https://www.pingidentity.com/en/resources/client-library/articles/openid-connect.html)
- [Auth0](https://auth0.com/docs/protocols/oidc)
- [Salesforce Identity](https://developer.salesforce.com/page/Inside_OpenID_Connect_on_Force.com)
- [Microsoft Azure Active Directory](https://docs.microsoft.com/en-us/azure/active-directory/develop/active-directory-protocols-openid-connect-code)
- [Google Identity Platform](https://developers.google.com/identity/)
- Known issue: [OneLogin](https://www.onelogin.com/openid-connect) OpenID Connect is not supported (use SAML for OneLogin instead)

To configure Sourcegraph to authenticate users via OpenID Connect:

1. Create a new OpenID Connect client in the external service (such as one of those listed above).
    - **Redirect/callback URI:** `https://sourcegraph.example.com/.auth/callback` (replace `https://sourcegraph.example.com` with the value of the `externalURL` property in your config)
1. Provide the OpenID Connect client's issuer, client ID, and client secret in the Sourcegraph site configuration shown below.
1. (Optional) Require users to have a specific email domain name to authenticate (e.g., to limit users to only those from your organization).

Example [`openidconnect` auth provider](../config/site_config.md#openid-connect-including-google-workspace) configuration:

```json
{
  // ...
  "externalURL": "https://sourcegraph.example.com",
  "auth.providers": [
    {
      "type": "openidconnect",
      "issuer": "https://oidc.example.com",
      "clientID": "my-client-id",
      "clientSecret": "my-client-secret",
      "requireEmailDomain": "example.com"
    }
  ]
}
```

Sourcegraph supports the OpenID Connect Discovery standard for configuring the auth provider (using the document at, e.g., `https://oidc.example.com/.well-known/openid-configuration`).

See the [`openid` auth provider documentation](../config/site_config.md#openid-connect-including-google-workspace) for the full set of configuration options.

### Google Workspace (Google accounts)

Google's Workspace (formerly known as G Suite) supports OpenID Connect, which is the best way to enable Sourcegraph authentication using Google accounts. To set it up:

1. Create an **OAuth client ID** and client secret in the [Google API credentials console](https://console.developers.google.com/apis/credentials). [Google's interactive OpenID Connect documentation page](https://developers.google.com/identity/protocols/OpenIDConnect#getcredentials):
    - **Application type:** Web application
    - **Name:** Sourcegraph (or any other name your users will recognize)
    - **Authorized JavaScript origins:** (leave blank)
    - **Authorized redirect URIs:** `https://sourcegraph.example.com/.auth/callback` (replace `https://sourcegraph.example.com` with the value of the `externalURL` property in your config)
1. Use the **client ID** and **client secret** values in Sourcegraph site configuration (as shown in the example below).
1. Set your Google Workspace domain in `requireEmailDomain` to prevent users outside your organization from signing in.

Example [`openidconnect` auth provider](../config/site_config.md#openid-connect-including-google-workspace) configuration for Google Workspace:

```json
{
  // ...
  "externalURL": "https://sourcegraph.example.com",
  "auth.providers": [
    {
      "type": "openidconnect",
      "issuer": "https://accounts.google.com", // All Google Workspace domains use this issuer URI.
      "clientID": "my-client-id",
      "clientSecret": "my-client-secret",
      "requireEmailDomain": "example.com"
    }
  ]
}
```

## HTTP authentication proxies

You can wrap Sourcegraph in an authentication proxy that authenticates the user and passes the user's username to Sourcegraph via HTTP headers. The most popular such authentication proxy is [pusher/oauth2_proxy](https://github.com/pusher/oauth2_proxy). Another example is [Google Identity-Aware Proxy (IAP)](https://cloud.google.com/iap/). Both work well with Sourcegraph.

To use an authentication proxy to authenticate users to Sourcegraph, add the following lines to your site configuration:

```json
{
  // ...
  "auth.providers": [
    {
      "type": "http-header",
      "usernameHeader": "X-Forwarded-User"
    }
  ]
}
```

Replace `X-Forwarded-User` with the name of the HTTP header added by the authentication proxy that contains the user's username.

Ensure that the HTTP proxy is not setting its own `Authorization` header on the request. Sourcegraph rejects requests with unrecognized `Authorization` headers and prints the error log `lvl=eror msg="Invalid Authorization header." err="unrecognized HTTP Authorization request header scheme (supported values: token, token-sudo)"`.

For pusher/oauth2_proxy, use the `-pass-basic-auth false` option to prevent it from sending the `Authorization` header.

### Username header prefixes

Some proxies add a prefix to the username header value. For example, Google IAP sets the `x-goog-authenticated-user-id` to a value like `accounts.google.com:alice` rather than just `alice`. If this is the case, use the `stripUsernameHeaderPrefix` field. If using Google IAP, for example, add the following lines to your site configuration:

```json
{
  // ...
  "auth.providers": [
    {
      "type": "http-header",
      "usernameHeader": "x-goog-authenticated-user-email",
      "stripUsernameHeaderPrefix": "accounts.google.com:"
    }
  ]
}
```

## Username normalization

Usernames on Sourcegraph are normalized according to the following rules.

- Any characters not in `[a-zA-Z0-9-.]` are replaced with `-`
- Usernames with exactly one `@` character are interpreted as an email address, so the username will be extracted by truncating at the `@` character.
- Usernames with two or more `@` characters are not considered an email address, so the `@` will be treated as a non-standard character and be replaced with `-`
- Usernames with consecutive `-` or `.` characters are not allowed
- Usernames that start or end with `.` are not allowed
- Usernames that start with `-` are not allowed

Usernames from authentication providers are normalized before being used in Sourcegraph. Usernames chosen by users are rejected if they do not meet these criteria.

For example, a user whose external username (according the authentication provider) is `alice_smith@example.com` would have the Sourcegraph username `alice-smith`.

If multiple accounts normalize into the same username, only the first user account is created. Other users won't be able to sign in. This is a rare occurrence; contact support if this is a blocker.
