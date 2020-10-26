# Creating a campaign

You can create a campaign from a campaign spec, which is a YAML file that describes your campaign.

The following example campaign spec adds "Hello World" to all `README.md` files:

```yaml
name: hello-world
description: Add Hello World to READMEs

# Find all repositories that contain a README.md file.
on:
  - repositoriesMatchingQuery: file:README.md

# In each repository, run this command. Each repository's resulting diff is captured.
steps:
  - run: echo Hello World | tee -a $(find -name README.md)
    container: alpine:3

# Describe the changeset (e.g., GitHub pull request) you want for each repository.
changesetTemplate:
  title: Hello World
  body: My first campaign!
  branch: hello-world # Push the commit to this branch.
  commit:
    message: Append Hello World to all README.md files
  published: false
```

1. Get started by running the following [Sourcegraph CLI (`src`)](https://github.com/sourcegraph/src-cli) command:

    <pre><code>src campaign preview -f <em>YOUR_CAMPAIGN_SPEC.campaign.yaml</em> -namespace <em>USERNAME_OR_ORG</em></code></pre>

    > **Don't worry!** Before any branches are pushed or changesets (e.g., GitHub pull requests) are created, you will see a preview of all changes and can confirm each one before proceeding.

    The `namespace` can be your Sourcegraph username or the name of a Sourcegraph organization under which you want to create the campaign.

1. Wait for it to run and compute the changes for each repository (using the repositories and commands in the campaign spec).
1. Open the preview URL that the command printed out.
1. Examine the preview. Confirm that the changes are what you intended. If not, edit the campaign spec and then rerun the command above.
1. Click the **Create campaign** button.

After you've applied a campaign spec, you can [publish changesets](publishing_changesets.md) to the code host when you're ready. This will turn the patches into commits, branches, and changesets (such as GitHub pull requests) for others to review and merge.

You can share the link to your campaign with other people if you want their help. Any person on your Sourcegraph instance can [view it in the campaigns list](viewing_campaigns.md).

If a person viewing the campaign lacks read access to a repository in the campaign, they can only see [limited information about the changes to that repository](../explanations/permissions_in_campaigns.md#repository-permissions-for-campaigns) (and not the repository name, file paths, or diff).

You can update a campaign's changes at any time, even after you've published changesets. For more information, see [Updating a campaign](updating_a_campaign.md).
