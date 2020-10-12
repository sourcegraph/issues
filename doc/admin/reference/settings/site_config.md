# Site configuration reference

Site configuration defines how various Sourcegraph features behave and specify operational aspects of the instance.

See the [global, organization, and user settings](global_org_user_settings.md) for customizing the Sourcegraph UI and user-specific features.

## View and edit site configuration

Site admins can view and edit site configuration on a Sourcegraph instance:

1. Go to **User menu > Site admin**.
1. Open the **Configuration** page. (The URL is `https://sourcegraph.example.com/site-admin/configuration`.)

## Site config reference

All site configuration options and their default values are shown below.

<div markdown-func=jsonschemadoc jsonschemadoc:path="admin/reference/settings/site.schema.json">[View page on docs.sourcegraph.com](https://docs.sourcegraph.com/admin/config/site_config) to see rendered content.</div>

## Known bugs

The following site configuration options require the server to be restarted for the changes to take effect:

```
auth.accessTokens
auth.sessionExpiry
git.cloneURLToRepositoryName
searchScopes
extensions
disablePublicRepoRedirects
```
