# Principles of extensibility for Sourcegraph

[Sourcegraph](https://sourcegraph.com) provides IDE-like code intelligence (definitions, references, hover tooltips, and search) anywhere you view and review code, such as in GitHub files and pull requests. To do this, we built a way to show information from [language servers](http://langserver.org) (for 20+ languages) in the places you view and review code.

Sometimes that info is enough. But sometimes you need to see other kinds of info: error logs, test coverage, performance, authorship, reachability, lint warnings, etc. For example, if you're reviewing a PR that changes code for which there is [OpenTracing](https://opentracing.io/) trace information or that adds calls to untested code, you should be aware of that while reviewing without having to check an external system.

Existing APIs for code hosts and editors don't make this easy or possible.

## The state of extension APIs for code hosts

The APIs of code hosts and review tools don't generally provide the necessary hooks to customize the code/diff view and other parts of the UI and workflow. Commit statuses, bot comments, and pipelines are fantastic, but they are solving other problems. Developers need more information and code actions while viewing and reviewing code (just as they do while editing code), and that means providing deeper extensibility.

Even if a single code host implemented a comprehensive extension API, it would be tied to that single code host. Extensions would only be usable by a fraction of developers unless the author created separate extensions for many popular code hosts: GitHub, GitHub Enterprise, Bitbucket, Bitbucket Server, GitLab, Phabricator, etc.

## The state of extension APIs for editors

Editors generally have comprehensive extension APIs, but they are each tied to a single editor. Initiatives like [Language Server Protocol (LSP)](https://microsoft.github.io/language-server-protocol/) are partial solutions, but LSP still requires an editor extension for each editor (such as [vscode-go](https://github.com/Microsoft/vscode-go)).

As a result, all of the great work that goes into building editor extensions is fragmented among 10+ editors. That makes it less likely that your favorite dev tools have great extensions for your editor of choice. It also makes it difficult to get your entire team using the same editor extensions and configuring them consistently (if your teammates use a variety of editors).

## How Sourcegraph extensions work

To solve this problem (of seeing important contextual info when you're viewing and reviewing code), we made a way for you to write something like an editor extension--except that it can be used on GitHub, Sourcegraph, and (soon) other tools. Check out the code for [an example extension that shows line and character position in hover tooltips](https://sourcegraph.com/github.com/sourcegraph/sourcegraph-extension-samples/-/blob/hello-world/src/extension.ts).

After writing and publishing an extension (using `src extensions publish` in the [`src` CLI tool](https://github.com/sourcegraph/src-cli)), anyone can add it and use it on GitHub or Sourcegraph. Behind the scenes (but in open source), 2 other components make this possible:

### Sourcegraph extension API

The Sourcegraph extension API (this repository) is the open-source extension API that Sourcegraph extensions are written against. It exposes concepts present in all clients, such as notifications of opened/closed documents, adding buttons to toolbars, changing the background or gutter color of lines, showing hover tooltips, etc.

The [sourcegraph](https://npmjs.com/package/sourcegraph) npm package (published from this repository) exposes this as a TypeScript/JavaScript API that is familiar to anyone who has written a [VS Code](https://code.visualstudio.com/) extension.

### Adapters to "polyfill" existing tools to run Sourcegraph extensions

Adapters make it so that Sourcegraph extensions can modify the UI of existing tools, such as GitHub. They run Sourcegraph extensions and know how to detect when the user hovers over a specific line and column in a file, how to insert buttons and actions seamlessly in the tool's UI, etc. All Sourcegraph client adapters are open source.

- The open-source [Sourcegraph for Chrome/Firefox](../integration/browser_extension.md) supports using Sourcegraph extensions on GitHub (support for other code hosts and browsers coming soon). It runs extensions by executing their JavaScript code on the client in a Web Worker.
- Support for GitHub PRs and diffs, other code hosts, other code review tools, and more browsers is coming soon.
- Sourcegraph for `$EDITOR` is coming soon (for VS Code, Vim, Emacs, Atom, Sublime, JetBrains, etc.).

## Acknowledgments

The Sourcegraph extension API and architecture is inspired by the [VS Code extension API](https://code.visualstudio.com/docs/extensions/overview) and the [Language Server Protocol (LSP)](https://microsoft.github.io/language-server-protocol/).
