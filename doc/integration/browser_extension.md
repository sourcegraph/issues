# Sourcegraph browser extension

The [open-source](https://github.com/sourcegraph/sourcegraph/tree/main/client/browser) Sourcegraph
browser extension adds code intelligence to files and diffs on GitHub, GitHub
Enterprise, GitLab, Phabricator, and Bitbucket Server.

<p>
  <a target="_blank" href="https://chrome.google.com/webstore/detail/sourcegraph/dgjhfomjieaadpoljlnidmbgkdffpack" style="display:flex;align-items:center">
  <img src="img/chrome.svg" width="24" height="24" style="margin-right:5px" /> <strong>Install Sourcegraph for Chrome</strong>
  </a>
</p>

<p>
  <a target="_blank" href="https://storage.googleapis.com/sourcegraph-for-firefox/latest.xpi" style="display:flex;align-items:center">
  <img src="img/firefox.svg" width="24" height="24" style="margin-right:5px" /> <strong>Install Sourcegraph for Firefox</strong>
  </a>
</p>

>NOTE: The Firefox extension may need to be manually enabled from `about:addons`, you can find more information in [Firefox add-on security](firefox_security.md).

![Sourcegraph browser extension](https://sourcegraphstatic.com/BrowserExtension.gif)

## Features

### Code intelligence

When you hover your mouse over code in files, diffs, pull requests, etc., the Sourcegraph extension displays a tooltip with:

- Documentation and the type signature for the hovered token
- **Go to definition** button
- **Find references** button

### Search shortcut in URL location bar

![Sourcegraph search shortcut](https://sourcegraphstatic.com/SearchShortcut2.gif)

The Sourcegraph extension adds a search engine shortcut to your web browser that performs a search on your Sourcegraph instance. After you've installed it (see above), use the search shortcut it provides to perform a search:

1. Place cursor in the URL location bar, then type <kbd>src</kbd> <kbd>Space</kbd>.
1. Start typing your search query.
1. Select an instant search suggestion or press <kbd>Enter</kbd> to see all results.

To install this search engine shortcut manually, and for more information, see "[Browser search engine shortcuts](browser_search_engine.md)".

## Configuring the browser extension to use a private Sourcegraph instance

![Sourcegraph search shortcut](https://sourcegraphstatic.com/ConfigureSourcegraphInstanceUse.gif)

By default, the browser extension communicates with [Sourcegraph.com](https://sourcegraph.com), which has only public code.

To use the browser extension with a different Sourcegraph instance:

1. Click the Sourcegraph extension icon in the browser toolbar to open the settings page.
1. Enter the URL (including the protocol) of your Sourcegraph instance (such as `https://sourcegraph.example.com`).
1. Hit return/enter and the extension will indicate its connection status.

## Enabling the browser extension on your code host

By default, the Sourcegraph browser extension will only provide code intelligence on [github.com](https://github.com/). It needs additional permissions in order to run on other code hosts.

To grant these permissions:

1. Navigate to any page on your code host.
1. Right-click the Sourcegraph icon in the browser extension toolbar.
1. Click "Enable Sourcegraph on this domain".
1. Click "Allow" in the permissions request popup.

## Troubleshooting

The most common problem is:

### No code intelligence or buttons ("View repository", "View file", etc.) are displayed on the code host

![Browser extension not working on code host](https://sourcegraphstatic.com/BrowserExtensionNotWorkingCodeHost.gif)

Try the following:

1. Click the Sourcegraph extension icon in the browser toolbar to open the settings page.
    - Ensure that the Sourcegraph URL is correct. It must point to your own Sourcegraph instance to work on private code.
    - Check whether any permissions must be granted. If so, the settings page will offer you to "grant the Sourcegraph browser extension additional permissions".
1. On some code hosts, you need to be signed in (to the code host) to use the browser extension. Try signing in.

### The *Enable Sourcegraph on this domain* option is not available

In rare cases, Chrome can get into the state where the option to **Enable Sourcegraph on this domain** is not available when right-clicking on the extension icon. One fix we've observed is to toggle the site access from on, to off, then on again (see below).

![Toggle site access for browser extension ](https://sourcegraphstatic.com/ToggleSiteAccess.gif)

If that still doesn't work, viewing the console and network activity of the extension is the next step.

### Viewing browser extension console and network activity in Chrome

If still experiencing issues, the next step is to inspect the browser extension console output and network activity, often revealing subtle configuration errors.

![Chrome extension console and network activity](https://sourcegraphstatic.com/ChromeExtensionConsoleNetworkActivity.gif)

If that still doesn't help, take a screenshot of the console and network activity and attach it [to a new issue](https://github.com/sourcegraph/sourcegraph/issues/new?assignees=&labels=&template=bug_report.md&title=Browser%20extension%20-%20) so we can investigate further.

### Unable to connect to `http://...` Ensure the URL is correct and you are logged in

Since `v3.14.0+`, the Sourcegraph browser extension can only authenticate with Sourcegraph instances that have [HTTPS](../admin/tls_ssl.md) configured.

Previously, the Sourcegraph browser extension was able to authenticate with instances that hadn't enabled tls / ssl. However, modern web browsers have started to adopt and implement [an IETF proposal](https://web.dev/samesite-cookies-explained/) that removes the deprecated logic that allowed this behavior. Please configure [HTTPS](../admin/tls_ssl.md) in order to continue using the browser extension with your private instance.

## Privacy

Sourcegraph integrations will only connect to Sourcegraph.com as required to provide code intelligence or other functionality on public code. As a result, no private code, private repository names, usernames, or any other specific data is sent to Sourcegraph.com.

If connected to a **private, self-hosted Sourcegraph instance**, Sourcegraph integrations never send any logs, pings, usage statistics, or telemetry to Sourcegraph.com. They will send notifications of usage to that private Sourcegraph instance only. This allows the site admins to see usage statistics.

If connected to the **public Sourcegraph.com instance**, Sourcegraph integrations will send notifications of usage on public repositories to Sourcegraph.com.
