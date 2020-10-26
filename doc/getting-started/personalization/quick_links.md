# Quick links

It is sometimes desirable to display links that are quickly accessible to users on the homepage and search page, e.g. links to:

- Important repositories in your organization
- Internal documentation you have about Sourcegraph
- Other links you think your Sourcegraph users would often want quick access to

Sourcegraph supports setting these `quicklinks` at 3 different levels:

- By site admins for all users: in the **Global settings** in the site admin area.
- By organization admins for all organization members: in the organization profile **Settings** section
- By users for themselves only: in the user profile **Settings** section

You can display these links by setting the `quicklinks` property to a JSON array of `{name, url}` objects. The `url` may be any valid URL or URL path.

For example, this JSON will create two quick links:

```json
{
  // ...
  "quicklinks": [
    {
      "name": "ACMECorp main repo",
      "url": "/github.com/acmecorp/main-repository"
    },
    {
      "name": "ACMECorp Sourcegraph docs",
      "url": "https://dev.acme.com/docs/sourcegraph"
    }
  ]
  // ...
}
```

After editing and saving the settings JSON, you can view the new links on the homepage or search results page.
