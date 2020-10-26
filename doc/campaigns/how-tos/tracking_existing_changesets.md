# Tracking existing changesets

You can track existing changests by adding them to the [campaign spec](#campaign-specs) under the `importChangesets` property.

The following example campaign spec tracks five existing changesets in different repositories on different code hosts:

```yaml
name: track-important-milestone
description: Track all changesets related to our important milestone

importChangesets:
- repository: github.com/sourcegraph/sourcegraph
  externalIDs: [12374, 11675]
- repository: bitbucket.sgdev.org/SOUR/vegeta
  externalIDs: [8]
- repository: gitlab.sgdev.org/sourcegraph/src-cli
  externalIDs: [113, 119]
```

1. Create a campaign from the campaign spec by running the following [Sourcegraph CLI (`src`)](https://github.com/sourcegraph/src-cli) command:

    <pre><code>src campaign preview -f <em>YOUR_CAMPAIGN_SPEC.campaign.yaml</em> -namespace USERNAME_OR_ORG</code></pre>

1. Open the preview URL that the command printed out.
1. Examine the preview. Confirm that the changesets are the ones you intended to track. If not, edit the campaign spec and then rerun the command above.
1. Click the **Create campaign** button.

You'll see the existing changeset in the list. The campaign will track the changeset's status and include it in the overall campaign progress (in the same way as if it had been created by the campaign). For more information, see ["Introduction to campaigns"](../explanations/introduction_to_campaigns.md).

> NOTE: You can combine the tracking of existing changesets and creating new ones by adding `importChangesets:` to your campaign specs that have `on:`, `steps:` and `changesetTemplate:` properties.

