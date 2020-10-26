# Telemetry

> NOTE: This document is a work-in-progress.

Telemetry describes the logging of user events, such as a page view or search. Telemetry data is collected by each Sourcegraph instance and is not sent to Sourcegraph.com (except in aggregate form as documented in "[Pings](../../admin/pings.md)").

## Action telemetry

An [action](../../extensions/authoring/contributions.md#actions) consists of an action ID (such as `linecounter.displayLineCountAction`), a command to invoke, and other presentation information. An action is executed when the user clicks on a button or command palette menu item for the action. Actions may be defined either by extensions or by the main application (these are called "builtin actions"). For telemetry purposes, the two types of actions are treated identically.

A telemetry event is logged for every action that is executed by the user. By default, only the action ID is logged (not the arguments). To include the arguments for an action in the telemetry event data, the caller will need to explicitly opt-in (for privacy/security). Action telemetry is implemented in the `ActionItem` React component.

Examples of builtin actions include `goToDefinition`, `goToDefinition.preloaded`, `findReferences`, and (soon) `toggleLineWrap`, `goToPermalink`, `toggleColorTheme`, etc. (i.e., the builtin buttons in the file header and global nav).

Examples of extension actions include [`git.blame.toggle` from sourcegraph/git-extras](https://sourcegraph.com/extensions/sourcegraph/git-extras/-/contributions). An extension's action IDs are specified in its extension manifest and can be viewed on the **Contributions** tab of its extension registry listing.

## Browser extension telemetry

> NOTE: This section is incomplete.
