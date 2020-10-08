# Hello World Campaign

> NOTE: This documentation describes the current work-in-progress version of campaigns. [Click here](https://docs.sourcegraph.com/@3.18/user/campaigns) to read the documentation for campaigns in Sourcegraph 3.18.

Have you ever needed to make the same kind of change to many repositories at once? Campaigns make this much easier. To get you started, let's run a very simple campaign: adding the line `Hello World` to all of your repositories' `README.md` files. After completing this exercise, you'll be able to create your own campaigns to make useful changes, fixes, refactors, and more.

You'll learn how to:

- Create a new campaign
- Select the repositories to work with
- Write the script to make changes to each repository
- Publish and monitor the status of your proposed changes

For more detailed information, see "[Campaigns](index.md)" in Sourcegraph documentation and watch the [campaigns demo video](https://www.youtube.com/watch?v=EfKwKFzOs3E).

## What is a campaign?

A campaign lets you make many related code changes, creating many branches and changesets (such as GitHub pull requests) across many repositories. You can track the progress as they are reviewed and merged. See "[About campaigns](index.md#about-campaigns)" for more information.

To use campaigns, you need to:

- [Set up a Sourcegraph instance](../../index.md#quickstart) and add some repositories to it.
- [Install Sourcegraph CLI](https://github.com/sourcegraph/src-cli) (`src`).

## Step 1. Write a campaign spec

A **campaign spec** is a YAML file that defines a campaign, including:

- The name and description of the campaign
- The set of repositories to change
- Commands to run in each repository to make the changes
- The commit message and branch name

Save the following campaign spec as `hello-world.campaign.yaml`:

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

## Step 2. Create the campaign

Let's see the changes that will be made. Don't worry---no commits, branches, or changesets will be published yet (the repositories on your code host will be untouched).

1. In your terminal, run this command:

    <pre><code>src campaign preview -f hello-world.campaign.yaml -namespace <em>USERNAME_OR_ORG</em></code></pre>

    > The `namespace` can be your Sourcegraph username or the name of a Sourcegraph organisation under which you want to create the campaign.
1. Wait for it to run and compute the changes for each repository.
1. When it's done, click the displayed link to see all of the changes that will be made.
1. Make sure the changes look right.

    > If you want to run the campaign on fewer repositories, change the roots query in `hello-world.campaign.yaml` to something like `file:README.md repo:myproject` (to only match repositories whose name contains `myproject`).
1. Click the **Create campaign** button.

You created your first campaign! The campaign's changesets are still unpublished, which means they exist only on Sourcegraph and haven't been pushed to your code host yet.

## Step 3. Publish the changes (optional)

Publishing causes commits, branches, and changesets to be created on your code host.

You probably don't want to publish these toy "Hello World" changesets to actively developed repositories, because that might confuse people ("Why did you add this line to our READMEs?").

On a real campaign, you would change the `published: false` in the `hello-world.campaign.yaml` to `published: true` and run the `src campaign preview` command again. The changesets will then be published when you create or update the campaign from the preview. You can also [publish only some of the changesets by setting `published` to an array](campaign_spec_yaml_reference.md#publishing-only-specific-changesets).

> NOTE: You can also create or update a campaign by running `src campaign apply`! This skips the preview stage, and is especially useful when updating an existing campaign.

## Congratulations!

You've created your first campaign! 🎉🎉

You can customize your campaign spec and experiment with making other types of changes. To update your campaign, edit `hello-world.campaign.yaml` and run `src campaign preview` again. (As before, you'll see a preview before any changes are applied.)

To learn what else you can do with campaigns, see "[Campaigns](index.md)" in Sourcegraph documentation.
