# Publishing a Sourcegraph extension

Publishing a Sourcegraph extension is fast and easy. It involves building (compiling and bundling) one or more TypeScript files into a single JavaScript file.

When [setting up your development environment](development_environment.md), you'll already have:

1. Installed the [Sourcegraph CLI (`src`)](https://github.com/sourcegraph/src-cli#installation)
1. [Configured `src` with an access token](https://github.com/sourcegraph/src-cli#setup)

Now publish your extension by running:

```bash
src extensions publish
```

At this point, your extension has been built and sent to Sourcegraph. The output will include a link to a detail page where you can enable your extension and start using it.

## Private extensions

Any user can publish to the Sourcegraph.com extension registry, all Sourcegraph instances can use extensions from Sourcegraph.com, and all Sourcegraph.com extensions are visible to everyone. If you need to publish an extension privately, use a private extension registry on your own self-hosted Sourcegraph instance.

## WIP extensions

An extension with no published releases, or whose `package.json` extension manifest has a `"wip": true` property, is considered a work-in-progress (WIP) extension. WIP extensions:

- are sorted last on the list of extensions (unless the user has previously enabled the WIP extension);
- have a red "WIP" badge on the extension card in the list; and
- have a red "WIP" badge on the extension's page.

You can use WIP extensions for testing in-development extensions, as well as new versions of an existing extension.

Don't forget to delete your WIP extension when it's no longer needed (in the **Manage** tab on the extension's registry page). We reserve the right to periodically purge WIP extensions that are not in use, to avoid user confusion (to re-add an extension, just republish it, or contact us to restore it).
