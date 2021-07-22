# How to troubleshoot a Sourcegraph extension

This guide gives specific instructions for troubleshooting [extensions](https://docs.sourcegraph.com/extensions) developed by Sourcegraph.

## FAQs

#### How do I know if a Sourcegraph extension is running?

1. Right click on the Sourcegraph website and click Inspect (Chrome/Firefox) / Inspect Element (Safari) to open `Developer Tools`
2. You should see a console message `Activating Sourcegraph extension:` follows by the names of all the running extensions in the `Console` tab
3. If you don't see the expected extension running, please go to the User Menu on your Sourcegraph instance and click on Extensions to make sure the extension is enabled

#### A Sourcegraph extension is not working, what should I do?

1. First of all, please make sure the extension in question is running by following the steps from above
2. Look for error messages in your browser's `Developer Console`
3. Look for error messages in your browser's  `Network panel`

#### Why is the extension icon on my sidebar is shown as inactive / greyed out?

This happens if the extension is disabled or if you're visiting a page where an extension can't have any actions. For example, Open-in-Editor extensions do not work on top level folders [(example)](https://sourcegraph.com/github.com/sourcegraph/sourcegraph) because you cannot open a repo.


#### How do I upgrade the extensions in my private extension registry?

You can upgrade an extension in your private extension registry by simpily running the same `src extensions copy -extension-id=... -current-user=...` command as you would when you first [publish the extension](https://docs.sourcegraph.com/admin/extensions#publish-extensions-to-a-private-extension-registry).


#### What does it mean when a red dot shows up on the Sourcegraph browser extension icon?

The red dot indicates that either the Sourcegraph URL entered is invalid, or you are currently on a private repository. Visit our [browser extension docs](https://docs.sourcegraph.com/integration/browser_extension#make-it-work-for-private-code) for more information about enabling Sourcegraph to work with private repositories.

## Extension Specific FAQs

#### Sonarqube: Error fetching Sonarqube data: Error: Forbidden
1. Look for error messages in your browser's  `Network panel`
2. If there is an error message indicates that the cors-anywhere request was being denied and you are using `"sonarqube.corsAnywhereUrl": "https://cors-anywhere.herokuapp.com"` in your configuration, please visit [https://cors-anywhere.herokuapp.com/corsdemo](https://cors-anywhere.herokuapp.com/corsdemo) to opt-in for temporary access by clicking on the `Request temporary access to the demo server` button
3. Alternatively you may remove this configuration option to use the default Sourcegraph's CORS proxy

#### Git-extras: Git blame is not working even though it is displayed as enabled
The extension is running if you can locate the Author in the bottom status bar. The plugin has 3 modes which can be activated by clicking on the extension icon on the extension sidebar on the right:

1. All but status bar are hidden
2. Author will be shown on the selected line
3. Author will show up on all lines (where changes are made)

#### Git-extras: Git blame is slow to load
The extension is expected to work slow when there are issues with the gitserver (eg. running out of resources) because the extension is dependent on the gitserver.

#### ESlint: The extension is not working on an instance.
The ESlint extension requires the eslint.insight.repository and eslint.insight.step to be configured in either the global settings or in the user settings for each insight using ESLint.

#### Open-in-intellij: Sourcegraph fails to load a file when trying to open the file from intellij Plugin.
This is most likely due to the file being opened in a Sourcegraph instance that does not have access to your files. You must first configure the plugin in order to use it with your private instance. See the plugin docs for more information.

#### Search-export: Can I export search results?
1. You can export search results by enabling the [Sourcegraph search results CSV export extension](https://sourcegraph.com/extensions/sourcegraph/search-export)
2. Once it is enabled, you will find an `Export to CSV` button in the Search-Results page
 
#### Search-export: Network Error when downloading CSV

It's likely that the CSV file exceeds the browser's limit for data URI size. Users can limit the size of search match preview size through their user settings (see ["contributions"](https://sourcegraph.com/extensions/sourcegraph/search-export/-/contributions) for search-export). If decreasing the size of search match previews doesn't resolve the issue, users can decrease the amount of search results exported with the `count:` filter in their search query.

#### Search-export: The number of exported results does not match the number of results displayed on Sourcegraph

This is expected, as all instances that match for a single file will be listed in the same entry column under the Search matches row.  

___

See the [Sourcegraph browser extension docs](https://docs.sourcegraph.com/integration/browser_extension#troubleshooting) for more troubleshooting tips.
