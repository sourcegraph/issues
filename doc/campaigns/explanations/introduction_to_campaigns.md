# Introduction to campaigns

<style>

img.screenshot {
    display: block;
    margin: 1em auto;
    max-width: 600px;
    margin-bottom: 0.5em;
    border: 1px solid lightgrey;
    border-radius: 10px;
}

</style>

## Overview

Campaigns let you make large-scale code changes across many repositories and code hosts. The campaign lets you create pull requests on all affected repositories, and it tracks their progress until they're all merged. You can preview the changes and update them at any time.

People usually use campaigns to make the following kinds of changes:

- Cleaning up common problems using linters.
- Updating uses of deprecated library APIs.
- Upgrading dependencies.
- Patching critical security issues.
- Standardizing build, configuration, and deployment files.

A campaign tracks all of its changesets (a generic term for pull requests or merge requests) for updates to:

- Status: open, merged, or closed
- Checks: passed (green), failed (red), or pending (yellow)
- Review status: approved, changes requested, pending, or other statuses (depending on your code host or code review tool)

<img src="https://storage.googleapis.com/sourcegraph-assets/docs/images/campaigns/campaign_tracking_sourcegraph_prs.png" class="screenshot">

You can see the overall trend of a campaign in the burndown chart, which shows the proportion of changesets that have been merged over time since the campaign was created.

<img src="https://storage.googleapis.com/sourcegraph-assets/docs/images/campaigns/campaign_tracking_sourcegraph_prs_burndown.png" class="screenshot">

## Supported code hosts and changeset types

The generic term **changeset** is used to refer to any of the following:

- GitHub pull requests.
- Bitbucket Server pull requests.
- GitLab merge requests.
- Bitbucket Cloud pull requests (not yet supported).
- Phabricator diffs (not yet supported).
- Gerrit changes (not yet supported).

A single campaign can span many repositories and many code hosts.

## Concepts

- A **campaign** is group of related changes to code, along with a title and description.
- {#campaign-spec} A **campaign spec** is a YAML file describing the campaign: repositories to change, commands to run, and a template for changesets and commits. You describe your high-level intent in the campaign spec, such as "lint files in all repositories with a `package.json` file".
- The campaign has associated **changesets**, which is a generic term for pull requests, merge requests, or any other reviewable chunk of code. (Code hosts use different terms for this, which is why we chose a generic term.)
- A **published changeset** means the commit, branch, and changeset have been created on the code host. An **unpublished changeset** is just a preview that you can view in the campaign but does not exist on the code host yet.
- A **spec** (campaign spec or changeset spec) is a "record of intent". When you provide a spec for a thing, the system will continuously try to reconcile the actual thing with your desired intent (as described by the spec). This involves creating, updating, and deleting things as needed.
- A campaign has many **changeset specs**, which are produced by executing the campaign spec (i.e., running the commands on each selected repository) and then using its changeset template to produce a list of changesets, including the diffs, commit messages, changeset title, and changeset body. You don't need to view or edit the raw changeset specs; you will edit the campaign spec and view the changesets in the UI.
- The **campaign controller** reconciles the actual state of the campaign's changesets on the code host so that they match your desired intent (as described in the changeset specs).

To learn about the internals of campaigns, see [Campaigns](../../../dev/background-information/campaigns/index.md) in the developer documentation.

## Known issues

- Campaigns currently support **GitHub**, **GitLab** and **Bitbucket Server** repositories. If you're interested in using campaigns on other code hosts, [let us know](https://about.sourcegraph.com/contact).
- It is not yet possible for a campaign to create multiple changesets in a single repository (e.g., to make changes to multiple subtrees in a monorepo).
- Forking a repository and creating a pull request on the fork is not yet supported. Because of this limitation, you need write access to each repository that your campaign will change (in order to push a branch to it).
- Campaign steps are run locally (in the [Sourcegraph CLI](https://github.com/sourcegraph/src-cli)). Sourcegraph does not yet support executing campaign steps on the server. For this reason, the APIs for creating and updating a campaign require you to upload all of the changeset specs (which are produced by executing the campaign spec locally). {#server-execution}
- It is not yet possible for multiple users to edit the same campaign that was created under an organization.
- It is not yet possible to reuse a branch in a repository across multiple campaigns.
