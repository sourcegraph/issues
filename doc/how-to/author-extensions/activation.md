# Sourcegraph extension activation

Sourcegraph selectively activates each extension based on the `activationEvents` array in its `package.json`. This improves performance by only using the network and CPU for extensions when necessary.

There are two types of activation events:

- `["*"]`: always activate
- `["onLanguage:typescript"]`: activate for files of a language (multiple languages supported)

For simplicity, the extension creator sets `activationEvents` to `["*"]`. Adjust this if your extension is language-specific.

## Determining the correct language value

Search this [list of languages](https://github.com/github/linguist/blob/master/lib/linguist/languages.yml) to find the value assigned to the `codemirror_mode` key for that language.

## Deactivation

When an extension is deactivated, it must unsubscribe its providers. Otherwise, its providers will continue to be invoked, leading to unintended behavior and resource leaks.

To support deactivation, an extension records its subscriptions in the `ExtensionContext` object passed to its `activate` function. For example:

```typescript
import * as sourcegraph from 'sourcegraph'

export function activate(ctx: sourcegraph.ExtensionContext): void {
    ctx.subscriptions.add(
        sourcegraph.languages.registerHoverProvider(['*'], () => ({ contents: { value: 'Hello, world!' } }))
    )
}
```

The `ctx.subscriptions.add` function accepts both `Unsubscribable` values (which are returned by functions such as `registerHoverProvider`) and arbitrary teardown functions (`() => void`). When the extension is deactivated, Sourcegraph invokes each entry to unsubscribe providers and tear down resources.

Tips:

- If your extension needs to support Sourcegraph versions prior to 3.0, see "[Backcompat for Sourcegraph versions prior to 3.0](activation.md#backcompat-for-sourcegraph-versions-prior-to-3-0)".
- It is safe to double-unsubscribe `Unsubscribable` values. Subsequent calls will be no-ops. There is no need to remove a subscription from `ctx.subscriptions` if your extension explicitly unsubscribed it already.
- Your extension can add subscriptions to `ctx.subscriptions` at any time, not just during initial activation.
- There is no guarantee that an extension will be deactivated, or that the deactivation process will finish. For example, if you close the browser tab where it was running, it may uncleanly terminate all extensions immediately or after deactivation has partially completed.
- If your extension requires asynchronous deactivation (which is rare), it can export a `deactivate` function, as in `export async function deactivate(): Promise<void> { /* ... */ }`.

### Deactivation triggers

An active extension is deactivated when:

- the user disables the extension (such as by navigating to the extension registry and using the slider to disable it); or
- none of the extension's `activationEvents` evaluate to true---and an arbitrary time period has passed. (The delay is intended to avoid frequent deactivation and reactivation when navigating between files of different languages, for example.)

If the extension was never activated, then it does not need to be deactivated.

### Why explicit deactivation is necessary

Extensions must support deactivation because there is no way for Sourcegraph to know (in general) which resources to free when an extension is deactivated. All extensions run in the same JavaScript execution context (usually a Web Worker), so Sourcegraph can't determine *which* extension called functions such as `registerHoverProvider`.

### Backcompat for Sourcegraph versions prior to 3.0

The `ctx: sourcegraph.ExtensionContext` parameter was [added in Sourcegraph 3.0 (#1120)](https://github.com/sourcegraph/sourcegraph/pull/1120). In prior Sourcegraph versions, the `activate` function is called with no parameters.

To avoid [`Uncaught ReferenceError: ctx is not defined`](#uncaught-referenceerror-ctx-is-not-defined) errors and support prior Sourcegraph versions in your extension, use the following workaround (which provides a default value for the `ctx` argument):

```typescript
import * as sourcegraph from 'sourcegraph'

// No-op for Sourcegraph versions prior to 3.0
const DUMMY_CTX = { subscriptions: { add: (_unsubscribable: any) => void 0 } }

export function activate(ctx: sourcegraph.ExtensionContext = DUMMY_CTX): void {
  ctx.subscriptions.add(/* ... */)
}
```

This makes deactivation a noop, which is OK because in these prior versions, each extension was executed in its own Web Worker and could be terminated (which would free all of its resources).

### Troubleshooting

#### Undesired or duplicate references, definitions, decorations, etc.

This occurs when a provider is not unsubscribed upon deactivation.

For example, with a hover provider:

- If a hover provider is not unsubscribed upon deactivation, its hovers will continue appearing.
- If the same extension is later reactivated, duplicate hovers from the extension will appear.

The following **incorrect** code example contains this bug:

```typescript
import * as sourcegraph from 'sourcegraph'

export function activate(ctx: sourcegraph.ExtensionContext): void {
    // ❌❌❌ INCORRECT USAGE (the hover provider will NOT be unsubscribed upon deactivation)
    sourcegraph.languages.registerHoverProvider(['*'], () => ({ contents: { value: 'Hello, world!' } }))
}
```

To fix this issue, ensure that the `Unsubscribable` value returned by `sourcegraph.languages.registerHoverProvider` is added to `ExtensionContext#subscriptions`.

The **correct** code is:

```typescript
import * as sourcegraph from 'sourcegraph'

export function activate(ctx: sourcegraph.ExtensionContext): void {
    // ✔️✔️✔️ CORRECT USAGE (the hover provider *will* unsubscribed upon deactivation)
    ctx.subscriptions.add(
        sourcegraph.languages.registerHoverProvider(['*'], () => ({ contents: { value: 'Hello, world!' } }))
    )
}
```

#### Uncaught ReferenceError: ctx is not defined

This occurs when an extension's `activate` function expects to be passed a `ctx: sourcegraph.ExtensionContext` argument, but it is used in a version of Sourcegraph prior to 3.0. To fix this issue, the extension author must republish the extension with the workaround described in "[Backcompat for Sourcegraph versions prior to 3.0](activation.md#backcompat-for-sourcegraph-versions-prior-to-3-0-preview)".
