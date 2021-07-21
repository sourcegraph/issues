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
  <a target="_blank" href="https://apps.apple.com/us/app/sourcegraph-for-safari/id1543262193" style="display:flex;align-items:center">
  <img src="img/safari.svg" width="24" height="24" style="margin-right:5px" /> <strong>Install Sourcegraph for Safari</strong>
  </a>
</p>

<p>
  <a target="_blank" href="https://addons.mozilla.org/en-US/firefox/addon/sourcegraph-for-firefox/" style="display:flex;align-items:center">
  <img src="img/firefox.svg" width="24" height="24" style="margin-right:5px" /> <strong>Install Sourcegraph for Firefox</strong>
  </a>
</p>

>NOTE: If you were using our self-hosted version of Firefox Extension and are looking to upgrade, please check our [migration guide](migrating_firefox_extension.md).

<video class="theme-dark-only" width="1760" height="1060" autoplay loop muted playsinline style="width: 100%; height: auto; max-width: 50rem">
  <source src="https://storage.googleapis.com/sourcegraph-assets/code-host-integration/CodeIntelligenceOnCodeHostDark.webm" type="video/webm">
  <source src="https://storage.googleapis.com/sourcegraph-assets/code-host-integration/CodeIntelligenceOnCodeHostDark.mp4" type="video/mp4">
  <p>Sourcegraph browser extension adds code intelligence to your code host</p>
</video>
<video class="theme-light-only" width="1760" height="1060" autoplay loop muted playsinline style="width: 100%; height: auto; max-width: 50rem">
  <source src="https://storage.googleapis.com/sourcegraph-assets/code-host-integration/CodeIntelligenceOnCodeHostLight.webm" type="video/webm">
  <source src="https://storage.googleapis.com/sourcegraph-assets/code-host-integration/CodeIntelligenceOnCodeHostLight.mp4" type="video/mp4">
  <p>Sourcegraph browser extension adds code intelligence to your code host</p>
</video>

## Features

### Code intelligence

When you hover your mouse over code in files, diffs, pull requests, etc., the Sourcegraph extension displays a tooltip with:

- Documentation and the type signature for the hovered token
- **Go to definition** button
- **Find references** button

### Search shortcut in URL location bar

<video class="theme-dark-only" width="1196" height="720" autoplay loop muted playsinline style="width: 100%; height: auto; max-width: 50rem">
  <source src="https://storage.googleapis.com/sourcegraph-assets/code-host-integration/BrowserShortcutDark.webm" type="video/webm">
  <source src="https://storage.googleapis.com/sourcegraph-assets/code-host-integration/BrowserShortcutDark.mp4" type="video/mp4">
  <p>Sourcegraph search shortcut</p>
</video>
<video class="theme-light-only" width="1196" height="720" autoplay loop muted playsinline style="width: 100%; height: auto; max-width: 50rem">
  <source src="https://storage.googleapis.com/sourcegraph-assets/code-host-integration/BrowserShortcutLight.webm" type="video/webm">
  <source src="https://storage.googleapis.com/sourcegraph-assets/code-host-integration/BrowserShortcutLight.mp4" type="video/mp4">
  <p>Sourcegraph search shortcut</p>
</video>

The Sourcegraph extension adds a search engine shortcut to your web browser that performs a search on your Sourcegraph instance. After you've installed it (see above), use the search shortcut it provides to perform a search:

1. Place cursor in the URL location bar, then type <kbd>src</kbd> <kbd>Tab</kbd>.
1. Start typing your search query.
1. Select an instant search suggestion or press <kbd>Enter</kbd> to see all results.

To install this search engine shortcut manually, and for more information, see "[Browser search engine shortcuts](browser_search_engine.md)".

## Make it work on your code host

<video class="theme-dark-only" width="1762" height="1384" autoplay loop muted playsinline style="width: 100%; height: auto; max-width: 50rem">
  <source src="https://storage.googleapis.com/sourcegraph-assets/code-host-integration/GrantPermissionsDark.webm" type="video/webm">
  <source src="https://storage.googleapis.com/sourcegraph-assets/code-host-integration/GrantPermissionsDark.mp4" type="video/mp4">
  <p>Grant permissions</p>
</video>
<video class="theme-light-only" width="1762" height="1384" autoplay loop muted playsinline style="width: 100%; height: auto; max-width: 50rem">
  <source src="https://storage.googleapis.com/sourcegraph-assets/code-host-integration/GrantPermissionsLight.webm" type="video/webm">
  <source src="https://storage.googleapis.com/sourcegraph-assets/code-host-integration/GrantPermissionsLight.mp4" type="video/mp4">
  <p>Grant permissions</p>
</video>

- [GitHub.com](https://github.com/) - no action required.
- GitHub Enterprise, GitLab, Bitbucket Server and Phabricator - you need to grant the extension permissions first.

To grant these permissions:

1. Navigate to any page on your code host.
1. Right-click the Sourcegraph icon in the browser extension toolbar.
1. Click "Enable Sourcegraph on this domain".
1. Click "Allow" in the permissions request popup.

## Make it work for private code

<video class="theme-dark-only" width="1764" height="1390" autoplay loop muted playsinline style="width: 100%; height: auto; max-width: 50rem">
  <source src="https://storage.googleapis.com/sourcegraph-assets/code-host-integration/PrivateInstanceDark.webm" type="video/webm">
  <source src="https://storage.googleapis.com/sourcegraph-assets/code-host-integration/PrivateInstanceDark.mp4" type="video/mp4">
  <p>Configure browser extension for your private Sourcegraph instance</p>
</video>
<video class="theme-light-only" width="1764" height="1390" autoplay loop muted playsinline style="width: 100%; height: auto; max-width: 50rem">
  <source src="https://storage.googleapis.com/sourcegraph-assets/code-host-integration/PrivateInstanceLight.webm" type="video/webm">
  <source src="https://storage.googleapis.com/sourcegraph-assets/code-host-integration/PrivateInstanceLight.mp4" type="video/mp4">
  <p>Configure browser extension for your private Sourcegraph instance</p>
</video>

By default, the browser extension communicates with [sourcegraph.com](https://sourcegraph.com), which has only public code.

To use the browser extension with your private repositories, you need to set up a private Sourcegraph instance and connect the extension to it.

Follow these instructions:

1. [Install Sourcegraph](https://docs.sourcegraph.com/admin/install). Skip this step if you already have a private Sourcegraph instance.
2. Click the Sourcegraph extension icon in the browser toolbar to open the settings page.
3. Enter the URL (including the protocol) of your Sourcegraph instance (such as `https://sourcegraph.example.com`)
4. Make sure the connection status shows "Looks good!"

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

### Viewing browser extension console and network activity in Chrome and Safari

If you are still experiencing issues, the next step is to inspect the browser extension console output and network activity, often revealing subtle configuration errors.

![Chrome extension console and network activity](https://sourcegraphstatic.com/ChromeExtensionConsoleNetworkActivity.gif)

In Chrome:

1. Right click the Sourcegraph browser extension icon
2. Select Manage Extensions
3. Under Inspect Views select background page, this will open a dev console to the extension background page
4. In the developer console select the network tab

In Safari:

1. Ensure you have access to the develop tab by selecting Safari > Preferences > Advanced, at the bottom of the preference UI check the box labelled `Show Develop menu in menu bar`
2. In Develop select Web Extension Background Pages > Sourcegraph
3. Select the Network tab

If that still doesn't help, take a screenshot of the console and network activity and attach it [to a new issue](https://github.com/sourcegraph/sourcegraph/issues/new?assignees=&labels=&template=bug_report.md&title=Browser%20extension%20-%20) so we can investigate further.

### Unable to connect to `http://...` Ensure the URL is correct and you are logged in

Since `v3.14.0+`, the Sourcegraph browser extension can only authenticate with Sourcegraph instances that have [HTTPS](../admin/tls_ssl.md) configured.

Previously, the Sourcegraph browser extension was able to authenticate with instances that hadn't enabled tls / ssl. However, modern web browsers have started to adopt and implement [an IETF proposal](https://web.dev/samesite-cookies-explained/) that removes the deprecated logic that allowed this behavior. Please configure [HTTPS](../admin/tls_ssl.md) in order to continue using the browser extension with your private instance.

### `The string did not match the expected pattern` error in Safari

The above validation error occurs in Safari when the browser tries to parse a non-JSON text as JSON. Communication between the graphql server & browser extension is always assumed to be JSON. However, when a proxy/saml is used with Sourcegraph, sometimes the proxy can send text/html instead.

If you see this error displayed beneath the Url field of the sourcegraph extension, make sure that your current browser session is not in private mode. 

## Privacy

Sourcegraph integrations will only connect to Sourcegraph.com as required to provide code intelligence or other functionality on public code. As a result, no private code, private repository names, usernames, or any other specific data is sent to Sourcegraph.com.

If connected to a **private, self-hosted Sourcegraph instance**, Sourcegraph integrations never send any logs, pings, usage statistics, or telemetry to Sourcegraph.com. They will send notifications of usage to that private Sourcegraph instance only. This allows the site admins to see usage statistics.

If connected to the **public Sourcegraph.com instance**, Sourcegraph integrations will send notifications of usage on public repositories to Sourcegraph.com.  

The browser extension also does not store sensitive data locally. The information stored is restricted to:

- AnonymousUid
-  Feature flags
  - Hover alerts
- Client settings
  - Enable/disable status
  - Sourcegraph URL
