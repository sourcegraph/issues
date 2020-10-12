# Federation

Federation refers to using multiple Sourcegraph servers together, each of which is responsible for a subset of repositories.

## Redirecting to Sourcegraph.com for public repositories

Currently the only supported federation use case is [redirecting your Sourcegraph instance's users to Sourcegraph.com for _public_ repositories](../how-to/redirecting_public_repositories.md) (instead of mirroring, analyzing, and indexing public repositories on your Sourcegraph instance).

Example: If federation is enabled, a user on your Sourcegraph instance who tries to access a public repository at `https://sourcegraph.example.com/github.com/my/publicrepo` will be redirected to the same repository on Sourcegraph.com `https://sourcegraph.com/github.com/my/publicrepo`.

Regardless of whether federation is enabled, private repositories are always handled entirely by your own Sourcegraph instance.

## Benefits

Enabling federation has the following benefits:

- It offloads the work of mirroring, analyzing, and indexing public code to Sourcegraph.com, so your own instance's performance and resource consumption are unaffected.
- Users get full code intelligence for all supported languages on Sourcegraph.com, even if your instance only has code intelligence enabled for a subset of languages.
- Sourcegraph.com will show users more cross-repository references to code (via the "Find references" feature) than your instance because Sourcegraph.com's index is already very large. Building a comparable index of public code on your instance would require a lot of time and resources.
- It eliminates the risk of cloning and building public, untrusted code on your own instance (which may be running inside your private network). Sourcegraph.com applies strict isolation and resource quotas to mitigate this risk on our own infrastructure.

## Future plans

We plan to enhance federation in the future to support merging data (such as cross-references) from multiple Sourcegraph instances, sharing user accounts, etc. [Post an issue](https://github.com/sourcegraph/sourcegraph/issues) if you have a specific feature request for federation.
