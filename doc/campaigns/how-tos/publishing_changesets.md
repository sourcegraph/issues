# Publishing changesets to the code host

After you've added patches, you can see a preview of the changesets (e.g., GitHub pull requests) that will be created from the patches. Publishing the changesets will, for each repository:

- Create a commit with the changes from the patches for that repository.
- Push a branch using the branch name you chose when creating the campaign.
- Create a changeset (e.g., GitHub pull request) on the code host for review and merging.

> NOTE: When pushing the branch Sourcegraph will use a force push. Make sure that the branch names are unused, otherwise previous commits will be overwritten.

When you're ready, you can publish all of a campaign's changesets by changing the `published: false` in your campaign spec to `true`:

```yaml
name: hello-world

# ...

changesetTemplate:
  # ...
  published: true
```

> NOTE: You can also [publish some of a campaign's changesets](../campaign_spec_yaml_reference.md#publishing-only-specific-changesets).

Then run the `src campaign preview` command again, or `src campaign apply` to immediately publish the changesets.

In the Sourcegraph web UI you'll see a progress indicator for the changesets that are being published. Any errors will be shown, and you can retry publishing after you've resolved the problem by running `src campaign apply` again. You don't need to worry about multiple branches or pull requests being created when you retry, because the same branch name will be used.

To publish a changeset, you need admin access to the campaign and write access to the changeset's repository (on the code host). For more information, see [Code host interactions in campaigns](../explanations/permissions_in_campaigns.md#code-host-interactions-in-campaigns). [Forking the repository](../explanations/introduction_to_campaigns.md#known-issues) is not yet supported.

> NOTE: Set the Git commit author details with the [`changesetTemplate.commit.author`](../campaign_spec_yaml_reference.md#changesettemplate-commit-author) fields in the campaign spec.

