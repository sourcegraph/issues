# Sourcegraph extension contribution points

Sourcegraph extensions contribute actions and menu items to the user interface of Sourcegraph (and other applications, such as code hosts, using the [browser extension](../../integration/browser_extension.md)). An extension's contributions are defined in its [`package.json` extension manifest](./manifest.md).

The [`extension.schema.json` JSON Schema](https://sourcegraph.com/github.com/sourcegraph/sourcegraph/-/blob/shared/src/schema/extension.schema.json) contains the full reference for all extension contribution points.

For an example extension that defines contributions, see the tutorial "[Sourcegraph extension buttons and custom commands](tutorials/button_custom_commands.md)".

## Actions

An action consists of an identifier (action ID), a command to invoke, and information about how the action should be shown to the user (title, description, icon, etc.). Most buttons and menu items you see in Sourcegraph are defined as actions.

To add a button or menu item, an extension must define an action in its `package.json` and specify in which [menu](contributions.md#menus) to display it.

See the `actions` property in [`extension.schema.json`](https://sourcegraph.com/github.com/sourcegraph/sourcegraph/-/blob/shared/src/schema/extension.schema.json) for a full reference on what fields can be set on an action.

## Menus

A menu is an existing part of the user interface (of Sourcegraph or any other integrated application, such as a code host) where actions can be shown. The available menus are:

* `editor/title`: The toolbar for a file or diff, which usually contains the filename on the left and actions on the right.
* `commandPalette`: The command palette, which is a searchable list of actions that the user can open by clicking the <kbd>≡</kbd> icon in the top navigation bar.
* `directory/page`: A section on all pages showing a directory listing. Sometimes known as a "tree page" on code hosts.
* `global/nav`: The global navigation bar, shown at the top of every page.
* `panel/toolbar`: The toolbar on the panel, which is used to show references, definitions, commit history, and other information related to a file or a token/position in a file.
* `search/results/toolbar`: The toolbar on search results pages.
* `help`: The help menu or page.

The set of available menus is defined in the `menus` property in [`extension.schema.json`](https://sourcegraph.com/github.com/sourcegraph/sourcegraph/-/blob/shared/src/schema/extension.schema.json).

To display an action in a menu, an extension's `package.json` must define the action *and* specify which menu it's displayed in. The menu entry can also specify a `when` condition to selectively show/hide the action depending on a condition.

For example, the following partial extension manifest would add an action to the command palette whenever the current file is a Python file:

``` json
{
  ...,
  "contributes": {
    "actions": [
      {
        "id": "pythonLint.toggle",
        "title": "Toggle Python lint warnings",
        "command": "pythonLint.toggleWarnings"
      }
    ],
    "menus": {
      "editor/title": [
        {"action": "pythonLint.toggle", "when": "resource.language == 'python'"}
      ]
    }
  }
}
```

See the tutorial "[Sourcegraph extension buttons and custom commands](tutorials/button_custom_commands.md)" for a full example.

## Configuration

Extensions can define JSON configuration properties that can be set at the global, organization, and user levels. The effective (final) settings for a user are created by merging settings at these levels, from lowest to highest precedence (i.e., a property defined in user settings overrides any values for the property from organization or global settings).

To help users edit their JSON settings, extensions can define a [JSON Schema](https://json-schema.org/) for their configuration. This makes it so that users get validation, completion, documentation, and hovers when editing their JSON settings on Sourcegraph. Extensions define the schema in the `configuration` property in their `contributions`.

For example, the following partial extension manifest would define a settings property for an extension:

``` json
{
  ...,
  "contributes": {
    "configuration": [
      "title": "Python lint settings",
      "properties": {
        "pythonLint.showWarnings": {
          "description": "Whether to report problems from the Python linter.",
          "type": "boolean"
        }
      }
    ]
  }
}
```

The entire value of the `configuration` object is itself a JSON Schema. See the [JSON Schema specification](https://json-schema.org/) and the [JSON Schema tutorial](https://json-schema.org/learn/getting-started-step-by-step.html) for more information.

## Search filters

> NOTE: This feature is not yet released.

Extensions will be able to contribute search filters, which are displayed on the Sourcegraph search results page and allow users to refine their query.

See the `searchFilters` property in [`extension.schema.json`](https://sourcegraph.com/github.com/sourcegraph/sourcegraph/-/blob/shared/src/schema/extension.schema.json).
