# Builtin Sourcegraph extension commands

This document lists the predefined commands that extensions can execute to perform actions on Sourcegraph or the client application. In addition to these commands, extensions may also define their own commands.

The following is example extension code to execute a command (`open`):

```typescript
await sourcegraph.commands.executeCommand('open', 'https://example.com')
```

Extension actions defined in `package.json` can also invoke commands. The following is an example `package.json` that defines an action that executes a command (`open`):

``` json
{
  ...
  "contributes": {
    "actions": [
      {
        "id": "myextension.openHomepage",
        "command": "open",
        "commandArguments": ["https://example.com"]
      }
    ],
    "menus": {
      "editor/title": [
        { "action": "myextension.openHomepage" }
      ]
    }
  },
  ...
}
```

## open

- Parameters:
  1. `url` (string) - The URL to open. (Example: `https://example.com`)
- Returns: `void`

Opens a URL using the client's default URL handler.

When a button (or other extension action) specifies this command, the button behaves like a link. (The client application uses HTML `<a href>` to render the button, instead of using a `click` handler and a `window.open` call.)

When this command is executed by an extension, it is equivalent to `window.open(url, '_blank')`. (Extensions can't call `window.open` directly because they run in a Web Worker without access to the DOM.) Most browsers' pop-up blockers will block new windows opened in this way, so use this sparingly.

## updateConfiguration

- Parameters:
  1. `keyPath` (`(string | number)[]`) - The key path of the value in user settings to update. Each element indexes into the settings object. For example, the key path `["foo", 2]` refers to `"B"` in `{ "foo": ["A", "B"] }`.
  1. `value` (any) - The value to insert or update at the key path.
- Returns: `Promise<void>`, which resolves when the update is persisted.

Updates a specific value in the user's settings.

User settings are visible to all extensions that the user has enabled. Each extension can read and write settings according to their own schema. Extensions should take care to avoid property name conflicts with other extensions (such as by using their extension name as a prefix).

The following is example extension code to update the user's settings (for a hypothetical "lint" extension):

```typescript
await sourcegraph.commands.executeCommand('updateConfiguration', ['lint.ignoreRules'], ['noSemicolons', 'longLines'])
await sourcegraph.commands.executeCommand('updateConfiguration', ['lint.maxWarnings'], 25)
```

## Toggle settings actions

Many extensions need to let the user easily toggle a feature on/off, such as with a "Show/hide lint warnings" button. To implement this, extensions can define an action that executes `updateConfiguration` and passes it specially crafted arguments to toggle between `true` and `false` values in the user's settings.

Here is an example `package.json` that defines a toggle button:

```json
{
  ...,
  "contributes": {
    "actions": [
      {
        "id": "myext.toggle",
        "command": "updateConfiguration",
        "commandArguments": [
          ["myext.enabled"],
          "${!config.myext.enabled}",
          null,
          "json"
        ],
        "title": "Enable/disable myext",
        "actionItem": {
          "label": "Toggle myext",
          "description": "${config.myext.enabled && \"Hide\" || \"Show\"} myext"
        }
      }
    ],
    "menus": {
      "editor/title": [
        { "action": "myext.toggle" }
      ]
    }
  },
  ...
}
```

The specially crafted `commandArguments` make the action update the `myext.enabled` settings property to the opposite of its current value. To achieve this, it uses template interpolation (with `${...}`) and a special 4th argument of `"json"` that causes the argument to be parsed as JSON instead of treated as a string. (This is important, because we want the `myext.enabled` setting to have a boolean value: `{"myext.enabled": true}`, not `{"myext.enabled": "true"}`.

## queryGraphQL

- Parameters:
  1. `query` (string) - The GraphQL query. (Example: `query Foo($bar: Int) { baz(bar: $bar) { zap } }`)
  1. `variables` (object) - The GraphQL query variables. (Example: `{ bar: 123 }`)
- Returns: `Promise<GraphQLResult>`, which resolves to the result of the GraphQL query.

Executes a [Sourcegraph GraphQL API](../../api/graphql/index.md) query or mutation on the associated Sourcegraph instance and returns the result asynchronously. If the user is authenticated, the query or mutation is executed as the current user.

This can be used to identify the current user, perform a search, fetch a file's contents, list a repository's branches, and anything else that Sourcegraph itself can do. See [example Sourcegraph GraphQL API queries](../../api/graphql/examples.md) for more.

## openPanel

- Parameters:
  1. `viewID` (string) - The `id` of the panel view to open (as specified in the `sourcegraph.app.createPanelView(id)` call).
- Returns: `void`

Opens a view in the panel. The view must have been previously created by the extension using `sourcegraph.app.createPanelView`.

## executeLocationProvider

- Parameters:
  1. `id` (string) - The location provider ID. This is defined by the extension as the first argument to `sourcegraph.languages.registerLocationProvider`.
  1. `uri` (string) - The URI of a text document (usually the currently active document).
  1. `position` (Position, `{line: number, character: number}`) - A position in a text document (usually the cursor position).
- Returns: `Location[] | Promise<Location[]>`

Executes a location provider and returns the results (a list of locations). Location providers are registered by extensions calling `sourcegraph.languages.registerLocationProvider`. They are the general form of definition providers and reference providers; they accept a document position and return a list of related locations in other files.

The `executeLocationProvider` command returns results to the caller but does not display them to the user.

Known issues:

- If the location provider returns an `Observable` (stream of values), the `executeLocationProvider` only returns a promise that resolves with the first emission. It does not return an observable.

## Opening a panel with a list of file locations

This example shows how to open a panel view that displays the location results from a location provider.

In your extension code, create a location provider and panel view, and link them together:

```typescript
// Create the panel view.
const panelView = sourcegraph.app.createPanelView('fooPanel')
panelView.title = 'Foo'

// Create a location provider.
sourcegraph.languages.registerLocationProvider('fooLocations', ['*'], {
  provideLocations: () => [],
})

// Tell the panel view to display the location provider's results.
panelView.component = { locationProvider: 'fooLocations' }
```

Now, execute the [`openPanel`](builtin_commands.md#openPanel) command with the first argument `fooPanel`. The panel will display the results from the location provider for the currently active document position.
