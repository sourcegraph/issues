# Paid subscriptions for Sourcegraph Enterprise

> NOTE: Sourcegraph is [open source](https://github.com/sourcegraph/sourcegraph). Pricing documentation below applies to [Sourcegraph Enterprise](https://about.sourcegraph.com/pricing).

Organizations using Sourcegraph can [upgrade to Sourcegraph Enterprise](https://about.sourcegraph.com/pricing) to get the features that large organizations need (single sign-on, backups and recovery, cluster deployment, access to code navigation and intelligence and other Sourcegraph extensions, etc.). These additional features in Sourcegraph Enterprise are paid and not open source.

You can [contact Sourcegraph](https://about.sourcegraph.com/contact/sales) to purchase a subscription. This entitles you to a license key (provided immediately after your purchase), which activates Enterprise features on your Sourcegraph instance.

## Volume discounts

[Contact us](https://about.sourcegraph.com/contact) to ask about volume discounts for Sourcegraph Enterprise. 

## How active users are counted

[Sourcegraph's pricing](https://about.sourcegraph.com/pricing) is based on **monthly active users**. This count is maintained on your Sourcegraph instance, viewable and auditable on the **Site admin > Usage statistics** page, and is reported back in aggregate to Sourcegraph.com via [pings](https://docs.sourcegraph.com/admin/pings).

A Sourcegraph active user is a person (associated with a single user account) who visits or uses your Sourcegraph while signed in over the course of the month.

This includes:

- Successfully signing in to your instance (via https://sourcegraph.example.com/sign-in).
- Running a search on your instance.
- Clicking a link to a page on your instance.
- Using a Sourcegraph integration connected to your Sourcegraph instance (such as a browser extension or native code host integration) while signed in.
- Receiving a saved search notification from your Sourcegraph instance.

This does not include:

- Viewing the sign-up page (at https://sourcegraph.example.com/sign-up) without signing up
- Viewing the sign-in page (at https://sourcegraph.example.com/sign-in) without signing in.
- Viewing and submitting a forgotten password form.

## How user accounts are counted

Some customers have contracts based on **total user accounts**, rather than on monthly active users. This count is maintained on your Sourcegraph instance, viewable and auditable on the **Site admin > Users** page, and is reported back in aggregate to Sourcegraph.com via [pings](https://docs.sourcegraph.com/admin/pings).

A Sourcegraph user account is created when a user signs up or signs in for the first time. Sourcegraph user accounts can be deleted by administrators via the **Site admin > Users** page, or using the GraphQL API (including with the [Sourcegraph CLI](https://github.com/sourcegraph/src-cli), if needed).

## Updating your license key

- Navigate to Site admin > Configuration and update the `licenseKey` field with the new value and save.
