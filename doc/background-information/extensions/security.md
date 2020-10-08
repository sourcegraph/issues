# Security and privacy of Sourcegraph extensions

You can use Sourcegraph extensions without a Sourcegraph.com account or self-hosted Sourcegraph instance.<sup><a href="#note-1">1</a></sup> To do so, just install the [browser extension](../integration/browser_extension.md).

We designed Sourcegraph extensions with security and privacy in mind:

- Extensions do not send your code to Sourcegraph to operate. They run entirely on the client in your web browser.
- Extensions don't have direct access to private code. Extensions needing code access, such as to provide find-references in a project, must fetch code from the API of your code host or self-hosted Sourcegraph instance. This requires you to explicitly permit access (e.g., by creating a GitHub access token and configuring the extension to use it).
- Extensions run in isolation from your code host's web pages. They don't have direct DOM access (because they run in a Web Worker) and can only contribute actions and behavior allowed by the Sourcegraph extension API.
- Extensions are sandboxed by your web browser. Because *Sourcegraph extensions* run inside of the Sourcegraph for Chrome/Firefox *browser extension*, they are limited by the permissions you granted to the browser extension.
- Sourcegraph development is open source, so these claims are verifiable.

<a name="note-1"><sup>1</sup></a> To use extensions other than the default set of language extensions, you currently do need an account or self-hosted instance. We plan to remove this limitation soon.
