# Building a language-specific extension tutorial

![Sourcegraph extension button](img/python-button.png)

Extensions can be configured to activate and contribute UI elements only upon the presence of a specific language, languages or glob pattern. This improves performance by only using the network and CPU for extensions that are needed.

In this tutorial, you'll build an extension that activates, displays a button, and hover content, but only if a Python file is present.

## Prerequisites

This tutorial presumes you have created and published an extension. If not, complete the [Hello world tutorial](hello_world.md) first.

It also presumes you have the [Sourcegraph browser extension](https://docs.sourcegraph.com/integration/browser_extension) installed.

## Set up

Create the extension you'll use for this tutorial:

```
mkdir py-button
cd py-button
npm init sourcegraph-extension
```

Then publish your extension:

```
src extension publish
```

Confirm your extension is enabled and working by:

- Opening the extension detail page.
- Viewing a file on Sourcegraph.com and confirming the hover for your extension appears.

## Controlling extension activation

To activate only for Python files, the contents of the `activationEvents` field in package.json changes to:

```json
"activationEvents": ["onLanguage:python"]
```

Publish your extension to apply this change and now, you will see hovers only when [viewing a Python file](https://sourcegraph.com/github.com/django/django/-/blob/django/core/wsgi.py).

## Conditional button display

This code defines an action that will be placed in `editor/title` section. It uses an icon and a small label to leave space for other other extension buttons.

Replace the contents of the [`contributes` object](../contributions.md) in `package.json`:

```json
{
  "actions": [
  {
    "id": "pybutton.open",
      "command": "open",
      "commandArguments": [
        "https://www.python.org/"
      ],
      "actionItem": {
        "description": "Go to python.org",
        "iconURL": "data:image/png;base64,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"
      }
    }
  ],
  "menus": {
    "editor/title": [
      {
        "action": "pybutton.open",
        "when": "resource"
      }
    ]
  }
}

```

The when field is an expression. If an expression returns a true or a truthy, the properties of the `action` it references are used to display an element in the chosen menu location, e.g. `editor/title`.

The code `"when": "resource"` means the extension will be active on code view pages only.

<!-- TODO (ryan): Link to template expression recipes and docs -->

Publish the extension to see the button when [viewing a Python file](https://sourcegraph.com/github.com/django/django/-/blob/django/core/wsgi.py).

![Sourcegraph extension button](img/python-button.png)

The button when clicked will call the built-in [`open command`](../builtin_commands.md#open) to load Python's home page. We're almost done, but we have one more change to do.

## Controlling language providers

Language providers e.g. the `HoverProvider`, control if they provide content by accepting an array of language filters when registered.

In the extension code, replace the contents of the array supplied to `registerHoverProvider` from `'*'` to `{ language: 'python' }`. It should now look like this:

```typescript
sourcegraph.languages.registerHoverProvider([{ language: 'python' }], {
  ...
}
```

Despite our extension activating for only Python files, we need to filter here too.

When an extension's `activeConditions` become not true (e.g, looking at a Ruby file), it's not immediately deactivated, meaning its hover registration(s) still exist. It's a safeguard so only Python files trigger hover content to be shown.

The [activation documentation](../activation.md) delves deeper into deactivation and the extension lifecycle.

## Summary

You can now conditionally activate, show a button, and hover content for a specific language!

## Next Steps

- [Extension activation](../builtin_commands.md)
- [Buttons and custom commands tutorial](button_custom_commands.md)
- [Extension contribution points](../contributions.md)
- [Builtin commands](../builtin_commands.md)
