# Building a "Hello, world!" Sourcegraph extension

[Sourcegraph extensions](https://docs.sourcegraph.com/extensions) lets you add features and show new kinds of information alongside your code on Sourcegraph.com, GitHub, and other code hosts.

This guide shows you how to create a simple Sourcegraph extension that:

- Shows a friendly "Hello, world! 🎉🎉🎉" message when you hover over code.
- Works on all code on GitHub (requires the [Sourcegraph browser extension](https://docs.sourcegraph.com/integration/browser_extension)).
- Works on all code on Sourcegraph.com.
- Runs entirely client-side in the browser (your code remains local and is not sent to any server).

## Prerequisites

Follow the instructions for [setting up your development environment](../development_environment.md) so you can build and publish the extension.

## Create the extension

Use the [Sourcegraph extension creator](https://github.com/sourcegraph/create-extension) to get started:

```bash
mkdir hello-world-extension
cd hello-world-extension
npm init sourcegraph-extension
```

Follow the prompts and once complete, view the code for the extension in the `src` directory. That's all it takes to create a simple extension!

Now let's publish it so you (and other people) can use it.

## Publishing the extension

Publish the extension by running:

```bash
src ext publish
```

Now that the extension is published let's use it.

## Use the extension

Open the URL found in the output from the publish command. This is the extension's listing page on the [Sourcegraph.com extension registry](https://sourcegraph.com/extensions). Anyone can visit this page to see more information about the extension and to enable it.

Toggle the slider to enable the extension for your account. Now you can:

- Visit any code file on Sourcegraph (such as [this file](https://sourcegraph.com/github.com/ReactiveX/rxjs/-/blob/src/internal/observable/SubscribeOnObservable.ts)) and hover over the code to see the "Hello, world! 🎉🎉🎉" message.
- Visit any code file on GitHub (such as [this file](https://github.com/ReactiveX/rxjs/blob/HEAD/src/internal/observable/SubscribeOnObservable.ts)) and hover over the code to see it say the same.

## Next steps

You've created your first Sourcegraph extension!

Now check out the [Sourcegraph extensions authoring documentation](../index.md) to see how to build more powerful extensions.
