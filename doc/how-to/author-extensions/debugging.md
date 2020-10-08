# Debugging a Sourcegraph extension

You can use your browser's devtools to debug Sourcegraph extensions running on a Sourcegraph instance (including Sourcegraph.com). Most browsers ship with an advanced JavaScript debugger, so this is a great way to find the cause of bugs in your extension.

## Setup

Before debugging your extension, make sure of the following things:

- You've enabled the extension on your Sourcegraph instance.
- You're on a page that triggers your extension's `activationEvents`. (For example, extensions with `*` are always activated. Extensions with `onLanguage:python` are only activated when you view `.py` files.)
- You published a sourcemap along with your extension's JavaScript file. The `src extensions publish` command expects a `.map` file next to your extension's `.js` file (e.g., `extension.js` and `extension.map`). If you used the [extension creator](https://github.com/sourcegraph/create-extension), this is already set up for you. (If you can't use sourcemaps, debugging the transpiled JavaScript code is still possible.)

## Use console.log

For quick debugging, you can use `console.log` (or [other `console` methods](https://developer.mozilla.org/en-US/docs/Web/API/console)). Sourcegraph extensions just consist of JavaScript code running in your browser in a Web Worker, so all of your favorite JavaScript debugging tricks work.

If debugging on a code host e.g. GitHub, the extension runs as a background script. To view your `console.log` statements, go to the background page in chrome://extensions. If you are unable to see a "Background page" link under the Sourcegraph extension then you need to enable Developer mode.

## Use the JavaScript debugger

Because Sourcegraph extensions just consist of JavaScript code that runs in a Web Worker in your browser, you can use your browser's JavaScript debugger to set breakpoints and step through execution.

To set breakpoints and step through execution in your Sourcegraph extension:

1. Open your browser's devtools.
1. In the **Sources** tab (or wherever your browser's devtools shows source files), open a source file from your extension. There are 3 ways to do this:
   - Press <kbd>Ctrl</kbd><kbd>P</kbd> or <kbd>Cmd</kbd><kbd>P</kbd> and search for the file by name. If you used the [extension creator](https://github.com/sourcegraph/create-extension), the main source filename is the extension name followed by `.ts` (e.g., `my-extension.ts`).
   - Add a `console.log` to your extension code and reload. Look in the devtools console for the log message, and then click on the filename where the log line was emitted (which is usually shown on the far right).
   - Add a `debugger;` statement to your extension code and reload (with your browser devtools open). Execution will stop when your browser encounters that statement, and you'll be dropped into the source file.
1. Set breakpoints and step through execution as you would for any other JavaScript file in your browser's devtools.

## Enable trace logging

A Sourcegraph page uses an internal RPC protocol to communicate with extensions (which run in a Web Worker). This protocol consists of request-response sequences, such as "get the hover contents for `$FILE` at `$POSITION`" and the corresponding response with the hover message.

In rare cases, it helps to see the communication between Sourcegraph and your extension. This can help identify bugs in the Sourcegraph extension API itself.

To enable trace logging:

1. Reveal the **Ext ▲** debug menu by running the following JavaScript code in your browser's devtools console on a Sourcegraph page: `localStorage.debug=true;location.reload()`
1. In the bottom right corner, click **Ext ▲**.
1. Enable the **Log to devtools console** toggle.

Trace log messages are logged via `console.log` and appear in your browser's devtools console.
