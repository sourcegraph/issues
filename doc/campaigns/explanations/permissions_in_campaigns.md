# Permissions in campaigns

> NOTE: This documentation describes the current work-in-progress version of campaigns. [Click here](https://docs.sourcegraph.com/@3.18/user/campaigns) to read the documentation for campaigns in Sourcegraph 3.18.

You can customize access to a campaign and propose changes to repositories with varying permission levels. Other people see the campaign's proposed changes to a repository if they can view that repository; otherwise, they can see only limited, non-identifying information about the change.

## Permission levels for campaigns

The permission levels for a campaign are:

- **Read:** For people who need to view the campaign.
- **Admin:** For people who need full access to the campaign, including editing, closing, and deleting it.

To see the campaign's proposed changes on a repository, a person *also* needs read access to that specific repository. Read or admin access on the campaign does not (by itself) entitle a person to viewing all of the campaign's changes. For more information, see "[Repository permissions for campaigns](#repository-permissions-for-campaigns)".

Site admins have admin permissions on all campaigns.

For now, users only have read access to campaigns. In the future, users will have admin permissions on their own campaigns too.

### Campaign access for each permission level

Campaign action | Read | Admin
--------------- | :--: | :----:
View campaign name and description<br/><small class="text-muted">Also applies to viewing the input branch name, created/updated date, and campaign status</small> | ⬤ | ⬤
View burndown chart (aggregate changeset statuses over time) | ⬤ | ⬤
View list of patches and changesets | ⬤ | ⬤
View diffstat (aggregate count of added/changed/deleted lines) | ⬤ | ⬤
View error messages (related to creating or syncing changesets) |  | ⬤
Edit campaign name, description, and branch name | ⬤ | ⬤
Update campaign patches (and changesets on code hosts) |  | ⬤
Publish changesets to code host |  | ⬤
Add/remove existing changesets to/from campaign |  | ⬤
Refresh changeset statuses |  | ⬤
Close campaign |  | ⬤
Delete campaign |  | ⬤

Authorization for all actions is also subject to [repository permissions](#repository-permissions-for-campaigns).

## Setting and viewing permissions for a campaign

When you create a campaign, you are given admin permissions on the campaign.

All users are automatically given read permissions to a campaign. Granular permissions are not yet supported. Assigning admin permissions on a campaign to another person or to all organization members is not yet supported. Transferring ownership of a campaign is not yet supported.

## Code host interactions in campaigns

All interactions with the code host are performed by Sourcegraph with the token with which you configured the code host. These operations include:

- Pushing a branch with the changes (the Git author and committer will be you, and the Git push will be authenticated with your credentials)
- Creating a changeset (e.g., on GitHub, the pull request author will be you)
- Updating a changeset
- Closing a changeset

See these code host specific pages for which permissions and scopes the tokens require:

- [GitHub](../../../admin/external_service/github.md#github-api-token-and-access)
- [GitLab](../../../admin/external_service/gitlab.md#access-token-scopes)
- [Bitbucket Server](../../../admin/external_service/gitlab.md#access-token-permissions)

In the future you'll be able to perform all code host interactions with a separate access token or your personal code host account.

## Repository permissions for campaigns

Your [repository permissions](../../../admin/repo/permissions.md) determine what information in a campaign you can view. You can only see a campaign's proposed changes to a repository if you have read access to that repository. Read or admin permissions on a campaign does not (by itself) permit you to view all of the campaign's changes.

When you view a campaign, you can see a list of patches and changesets. For each patch and changeset:

- **If you have read access** to the repository for a patch or changeset, you can see the diff, changeset title, changeset link, detailed status, and other information.
- **If you do not have read access** to the repository for a patch or changeset, you can only see the status, last-updated date, and whether an error occurred (but not the error message). You can't see the diff, changeset title, changeset link, repository name, or any other information.

When you perform any campaign operation that involves repositories or code host interaction, your current repository permissions are taken into account.

- Creating, updating, or publishing a campaign; or publishing a single changeset: You must have access to view the repositories and the configured token must have the rights to push a branch and create the changesets on your code host (e.g., push branches to and open pull requests on the GitHub repositories).
- Adding existing changesets to a campaign: You must have read access to the existing changeset's repository.
- Closing or deleting a campaign: If you choose to also close associated changesets on the code host, you must have access to do so on the code host. If you do not have access to close a changeset on the code host, the changeset will remain in its current state. A person with repository permissions for the remaining changesets can view them and manually close them.

Your repository permissions can change at any time:

- If you've already published a changeset to a repository that you no longer have access to, then you won't be able to view its details or update it in your campaign. The changeset on the code host will remain in its current state. A person with permissions to the changeset on the code host will need to manually manage or close it. When possible, you'll be informed of this when updating a campaign that contains changesets you've lost access to.
- You need access to all repositories mentioned in a campaign plan to use it when updating a campaign.

If you are not permitted to view a repository on Sourcegraph, then you won't be able to perform any operations on it, even if you are authorized on the code host.

## Disabling campaigns

A site admin can disable campaigns for the entire site by setting the [site configuration](../../../admin/config/site_config.md) property `"campaigns.enabled"` to `false`.
