# Updating a campaign

Campaigns are identified by their name. It must be unique within a single namespace (your user account on Sourcegraph, or an organization you are a member of).
Updating a campaign works by targeting an **existing** campaign in the namespace by specifying the name in the spec. If the name matches an existing campaign, it will update the campaign. (Otherwise, a new campaign will be created.)

You can edit a the campaign's description, and any other part of its campaign spec at any time.

To update a campaign, you need [admin access to the campaign](../explanations/permissions_in_campaigns.md#campaign-access-for-each-permission-level), and [write access to all affected repositories](../explanations/permissions_in_campaigns.md#repository-permissions-for-campaigns) with published changesets.

1. Update the [campaign spec](../explanations/introduction_to_campaigns.md#concepts) to include the changes you want to make to the campaign. For example, change the `description` of the campaign or change the commit message in the `changesetTemplate`.
1. In your terminal, run the [Sourcegraph CLI (`src`)](https://github.com/sourcegraph/src-cli) command shown. The command will execute your campaign spec to generate changes and then upload them to the campaign for you to preview and accept.

    <pre><code>src campaign preview -f <em>YOUR_CAMPAIGN_SPEC.campaign.yaml</em> -namespace USERNAME_OR_ORG</code></pre>

    > **Don't worry!** Before any branches or changesets are modified, you will see a preview of all changes and can confirm before proceeding.

1. Open the preview URL that the command printed out.
1. Examine the preview. Confirm that the changes are what you intended. If not, edit your campaign spec and then rerun the command above.
1. Click the **Update campaign** button.

All of the changesets on your code host will be updated to the desired state that was shown in the preview.

> NOTE: If you are sure about the changes you want to make and don't need to preview them, you can run `src campaign apply` to apply the campaign spec directly:
> <pre><code>src campaign apply -f <em>YOUR_CAMPAIGN_SPEC.campaign.yaml</em> -namespace USERNAME_OR_ORG</code></pre>
