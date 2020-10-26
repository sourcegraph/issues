# Repository permissions

Sourcegraph can be configured to enforce repository permissions from code hosts.

Currently, GitHub, GitHub Enterprise, GitLab and Bitbucket Server permissions are supported. Check our [product direction](https://about.sourcegraph.com/direction) for plans to support other code hosts. If your desired code host is not yet on the roadmap, please [open a feature request](https://github.com/sourcegraph/sourcegraph/issues/new?template=feature_request.md).

> NOTE: Site admin users bypass all permission checks and have access to every repository on Sourcegraph.

## GitHub

> WARNING: It takes time to complete mirroring repository permissions from the code host, please read about [background permissions syncing](#background-permissions-syncing) to know what to expect.

Prerequisite: [Add GitHub as an authentication provider.](../auth/index.md#github)

Then, [add or edit a GitHub connection](../external_service/github.md#repository-syncing) and include the `authorization` field:

```json
{
   "url": "https://github.com",
   "token": "$PERSONAL_ACCESS_TOKEN",
   "authorization": {}
}
```

## GitLab

> WARNING: It takes time to complete mirroring repository permissions from the code host, please read about [background permissions syncing](#background-permissions-syncing) to know what to expect.

GitLab permissions can be configured in three ways:

1. Set up GitLab as an OAuth sign-on provider for Sourcegraph (recommended)
2. Use a GitLab administrator (sudo-level) personal access token in conjunction with another SSO provider
   (recommended only if the first option is not possible)
3. Assume username equivalency between Sourcegraph and GitLab (warning: this is generally unsafe and
   should only be used if you are using strictly `http-header` authentication).

### OAuth application

Prerequisite: [Add GitLab as an authentication provider.](../auth/index.md#gitlab)

Then, [add or edit a GitLab connection](../external_service/gitlab.md#repository-syncing) and include the `authorization` field:

```json
{
  "url": "https://gitlab.com",
  "token": "$PERSONAL_ACCESS_TOKEN",
  "authorization": {
    "identityProvider": {
      "type": "oauth"
    }
  }
}
```

### Administrator (sudo-level) access token

This method requires administrator access to GitLab so that Sourcegraph can access the [admin GitLab Users API endpoint](https://docs.gitlab.com/ee/api/users.html#for-admins). For each GitLab user, this endpoint provides the user ID that comes from the authentication provider, so Sourcegraph can associate a user in its system to a user in GitLab.

Prerequisite: Add the [SAML](../auth/index.md#saml) or [OpenID Connect](../auth/index.md#openid-connect)
authentication provider you use to sign into GitLab.

Then, [add or edit a GitLab connection](../external_service/gitlab.md#repository-syncing) using an administrator (sudo-level) personal access token, and include the `authorization` field:

```json
{
  "url": "https://gitlab.com",
  "token": "$PERSONAL_ACCESS_TOKEN",
  "authorization": {
    "identityProvider": {
      "type": "external",
      "authProviderID": "$AUTH_PROVIDER_ID",
      "authProviderType": "$AUTH_PROVIDER_TYPE",
      "gitlabProvider": "$AUTH_PROVIDER_GITLAB_ID"
    }
  }
}
```

`$AUTH_PROVIDER_ID` and `$AUTH_PROVIDER_TYPE` identify the authentication provider to use and should
match the fields specified in the authentication provider config
(`auth.providers`). 

`$AUTH_PROVIDER_GITLAB_ID` should match the `identities.provider` returned by
[the admin GitLab Users API endpoint](https://docs.gitlab.com/ee/api/users.html#for-admins).

### Username

Prerequisite: Ensure that `http-header` is the *only* authentication provider type configured for
Sourcegraph. If this is not the case, then it will be possible for users to escalate privileges,
because Sourcegraph usernames are mutable.

[Add or edit a GitLab connection](../external_service/gitlab.md#repository-syncing) and include the `authorization` field:

```json
{
  "url": "https://gitlab.com",
  "token": "$PERSONAL_ACCESS_TOKEN",
  "authorization": {
    "identityProvider": {
      "type": "username"
    }
  }
}
```

## Bitbucket Server

> WARNING: It takes time to complete mirroring repository permissions from the code host, please read about [background permissions syncing](#background-permissions-syncing) to know what to expect.

Enforcing Bitbucket Server permissions can be configured via the `authorization` setting in its configuration.

### Prerequisites

1. You have the exact same user accounts, **with matching usernames**, in Sourcegraph and Bitbucket Server. This can be accomplished by configuring an [external authentication provider](../auth/index.md) that mirrors user accounts from a central directory like LDAP or Active Directory. The same should be done on Bitbucket Server with [external user directories](https://confluence.atlassian.com/bitbucketserver/external-user-directories-776640394.html).
1. Ensure you have set `auth.enableUsernameChanges` to **`false`** in the [site config](../config/site_config.md) to prevent users from changing their usernames and **escalating their privileges**.


### Setup

This section walks you through the process of setting up an *Application Link between Sourcegraph and Bitbucket Server* and configuring the Sourcegraph Bitbucket Server configuration with `authorization` settings. It assumes the above prerequisites are met.

As an admin user, go to the "Application Links" page. You can use the sidebar navigation in the admin dashboard, or go directly to [https://bitbucketserver.example.com/plugins/servlet/applinks/listApplicationLinks](https://bitbucketserver.example.com/plugins/servlet/applinks/listApplicationLinks).

<img src="https://imgur.com/Hg4bzOf.png" width="800">

---

Write Sourcegraph's external URL in the text area (e.g. `https://sourcegraph.example.com`) and click **Create new link**. Click **Continue** even if Bitbucket Server warns you about the given URL not responding.

<img src="https://imgur.com/x6vFKIL.png" width="800">

---

Write `Sourcegraph` as the *Application Name* and select `Generic Application` as the *Application Type*. Leave everything else unset and click **Continue**.

<img src="https://imgur.com/161rbB9.png" width="800">

---


Now click the edit button in the `Sourcegraph` Application Link that you just created and select the `Incoming Authentication` panel.

<img src="https://imgur.com/sMGmzhH.png" width="800">

---


Generate a *Consumer Key* in your terminal with `echo sourcegraph$(openssl rand -hex 16)`. Copy this command's output and paste it in the *Consumer Key* field. Write `Sourcegraph` in the *Consumer Name* field.

<img src="https://imgur.com/1kK2Y5x.png" width="800">

---

Generate an RSA key pair in your terminal with `openssl genrsa -out sourcegraph.pem 4096 && openssl rsa -in sourcegraph.pem -pubout > sourcegraph.pub`. Copy the contents of `sourcegraph.pub` and paste them in the *Public Key* field.

<img src="https://imgur.com/YHm1uSr.png" width="800">

---

Scroll to the bottom and check the *Allow 2-Legged OAuth* checkbox, then write your admin account's username in the *Execute as* field and, lastly, check the *Allow user impersonation through 2-Legged OAuth* checkbox. Press **Save**.

<img src="https://imgur.com/1qxEAye.png" width="800">

---

Go to your Sourcegraph's *Manage repositories* page (i.e. `https://sourcegraph.example.com/site-admin/external-services`) and either edit or create a new *Bitbucket Server* connection. Click on the *Enforce permissions* quick action on top of the configuration editor. Copy the *Consumer Key* you generated before to the `oauth.consumerKey` field and the output of the command `base64 sourcegraph.pem | tr -d '\n'` to the `oauth.signingKey` field.

<img src="https://imgur.com/ucetesA.png" width="800">

### Fast permission sync with Bitbucket Server plugin

By installing the [Bitbucket Server plugin](../../../integration/bitbucket_server.md), you can make use of the fast permission sync feature that allows using Bitbucket Server permissions on larger instances.

---

Finally, **save the configuration**. You're done!

## Background permissions syncing

Sourcegraph 3.17+ supports syncing permissions in the background by default to better handle repository permissions at scale for GitHub, GitLab, and Bitbucket Server code hosts, and has become the only permissions mirror option since Sourcegraph 3.19. Rather than syncing a user's permissions when they log in and potentially blocking them from seeing search results, Sourcegraph syncs these permissions asynchronously in the background, opportunistically refreshing them in a timely manner.

For older versions (Sourcegraph 3.14, 3.15, and 3.16), background permissions syncing is behind a feature flag in the [site configuration](../config/site_config.md):

```json
"permissions.backgroundSync": {
	"enabled": true
}
```

Benefits of background syncing:

1. More predictable load on the code host API due to maintaining a schedule of permission updates.
1. Permissions are quickly synced for new repositories added to the Sourcegraph instance.
1. Users who sign up on the Sourcegraph instance can immediately get search results from some repositories they have access to on the code host as we begin to incrementally sync their permissions.

Considerations when enabling for the first time:

1. While the initial sync for all repositories and users is happening, users can gradually see more and more search results from repositories they have access to.
1. It takes time to complete the first sync. Depending on how many private repositories and users you have on the Sourcegraph instance, it can take from a few minutes to several hours. This is generally not a problem for fresh installations, since admins should only make the instance available after it's ready, but for existing installations, active users may not see the repositories they expect in search results because the initial permissions syncing hasn't finished yet.
1. More requests to the code host API need to be done during the first sync, but their pace is controlled with rate limiting.

Please contact [support@sourcegraph.com](mailto:support@sourcegraph.com) if you have any concerns/questions about enabling this feature for your Sourcegraph instance.

### Complete sync vs incremental sync

A complete sync means a repository or user has done a repository-centric or user-centric syncing respectively, which presists the most accurate permissions from code hosts to Sourcegraph.

An incremental sync is in fact a side effect of a complete sync because a user may grant or lose access to repositories and we react to such changes as soon as we know to improve permissions accuracy.

## Explicit permissions API

Sourcegraph exposes a GraphQL API to explicitly set repository permissions. This will become the primary
way to specify permissions in the future and will eventually replace the other repository
permissions mechanisms.

To enable the permissions API, add the following to the [site configuration](../config/site_config.md):

```json
"permissions.userMapping": {
    "enabled": true,
    "bindID": "email"
}
```

The `bindID` value specifies how to uniquely identify users when setting permissions:

- `email`: You can [set permissions](#settings-repository-permissions-for-users) for users by specifying their email addresses (which must be verified emails associated with their Sourcegraph user account).
- `username`: You can [set permissions](#settings-repository-permissions-for-users) for users by specifying their Sourcegraph usernames.

If the permissions API is enabled, all other repository permissions mechanisms are disabled.

After you enable the permissions API, you must [set permissions](#settings-repository-permissions-for-users) to allow users to view repositories. (Site admins bypass all permissions checks and can always view all repositories.) 

> If you were previously using [background permissions syncing](#background-permissions-syncing), then those permissions are used as the initial state. Otherwise, the initial state is for all repositories to have an empty set of authorized users, so users will not be able to view any repositories.

### Setting repository permissions for users

Setting the permissions for a repository can be accomplished with 2 [GraphQL API](../../api/graphql.md) calls.

First, obtain the ID of the repository from its name:

```graphql
query {
  repository(name: "github.com/owner/repo") {
    id
  }
}
```

Next, set the list of users allowed to view the repository:

```graphql
mutation {
  setRepositoryPermissionsForUsers(
    repository: "<repo ID>", 
    userPermissions: [
      { bindID: "user@example.com" }
    ]) {
    alwaysNil
  }
}
```

Now, only the users specified in the `userPermissions` parameter will be allowed to view the repository. Sourcegraph automatically enforces these permissions for all operations. (Site admins bypass all permissions checks and can always view all repositories.)

You can call `setRepositoryPermissionsForUsers` repeatedly to set permissions for each repository, and whenever you want to change the list of authorized users.

### Listing a user's authorized repositories

You may query the set of repositories visible to a particular user with the `authorizedUserRepositories` [GraphQL API](../../api/graphql.md) mutation, which accepts a `username` or `email` parameter to specify the user:

```graphql
query {
  authorizedUserRepositories(email: "user@example.com", first: 100) {
    nodes {
      name
    }
    totalCount
  }
}
```
