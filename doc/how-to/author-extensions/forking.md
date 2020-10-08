---
ignoreDisconnectedPageCheck: true
---

# Publishing a local copy of an extension

<!-- TODO(sqs): WIP -->

If your Sourcegraph instance is unable to connect to Sourcegraph.com (due to a firewall), or if you want to customize an extension, you need to publish a local copy to your Sourcegraph instance. To do so, follow these steps:

1.  Download and install the latest [src](https://github.com/sourcegraph/src-cli) (Sourcegraph CLI).
1.  [Configure and authenticate `src`](https://github.com/sourcegraph/src-cli#setup) with the URL and an access token for your Sourcegraph instance.
1.  Clone the repository of the extension you want to publish: [sourcegraph-codecov](https://github.com/sourcegraph/sourcegraph-codecov) or [sourcegraph-typescript](https://github.com/sourcegraph/sourcegraph-typescript).
1.  Run `npm install` in the clone directory to install dependencies.
1.  Run `src extensions publish -extension-id $USER/$NAME` in the clone directory to publish the extension locally to your Sourcegraph instance. Replace `$USER` with your Sourcegraph username and `$NAME` with with `codecov` or `typescript`.
1.  Enable the extension for your Sourcegraph user account by clicking on **User menu > Extensions** in the top navigation bar and then toggling the slider to on.
