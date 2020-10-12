# Administration of Sourcegraph extensions and the extension registry

[Sourcegraph extensions](../../extensions/index.md) add features to Sourcegraph. Sourcegraph Free and Enterprise instances allow users to view and enable extensions from the [Sourcegraph.com extension registry](https://sourcegraph.com/extensions).

Site administrators can customize how Sourcegraph extensions are used on their instance, with options for:

- a private extension registry on their instance,
- allowing only specific extensions to be enabled by users, and
- preventing users from enabling any extension from Sourcegraph.com.

> WARNING: Sourcegraph does not verify the authenticity or security of extensions published to Sourcegraph.com. You (and your users) are should take care when enabling new extensions, just as you would for any other programs installed from the web (such as editor extensions or browser extensions). The configuration options on this page allow site admins to lock down usage of extensions.

## Enable an extension for all users

To enable an extension for all users, add it to the `extensions` object in global settings (as shown below).

```json
{
  ...,
  "extensions": {
    ...,
    "alice/myextension": true,
    ...
  },
  ...
}
```

(To enable an extension for a single user or organization as a site admin, edit the user or organization settings in the same way as shown above.)

If a user's organization settings or user settings explicitly disable the extension (by setting its `extensions` key to `false`), the extension will be disabled for that user (even if it is enabled in global settings). This is because organization and user settings take precedence over global settings. Site admins can edit any user's or organization's settings to remove overrides if needed.

## Publish extensions to a private extension registry

If you want to create extensions that are only visible to users on your Sourcegraph instance, you can use Sourcegraph Enterprise's private extension registry feature. This is enabled by default on Sourcegraph Enterprise.

To publish an extension to your instance's private extension registry:

1. Configure your [Sourcegraph CLI (`src`)](https://github.com/sourcegraph/src-cli) with the URL and an access token for your Sourcegraph instance.
1. Depending on whether or not the extension already exists on Sourcegraph.com:
  - If the extension already exists on Sourcegraph.com, you can copy it to your private extension registry with `src extensions copy -extension-id=... -current-user=...`
  - If this is a new extension, run `src extensions publish` in the extension directory.

On Sourcegraph Free, the only way to publish extensions is to publish them to the [Sourcegraph.com extension registry](https://sourcegraph.com/extensions), where anyone on the web can view them.

## Use extensions from Sourcegraph.com (or disable remote extensions)

Sourcegraph Free and Enterprise instances use extensions from Sourcegraph.com with [`extensions.remoteRegistry`](../config/site_config.md) set to `"https://sourcegraph.com/.api/registry"`. The OSS version of Sourcegraph has no dependencies on external services, and its `extensions.remoteRegistry` defaults to `false`.

You can disable extensions from Sourcegraph.com by setting [`extensions.remoteRegistry`](../config/site_config.md) to `false` in your site configuration:

```json
{
  "extensions": { "remoteRegistry": false }
}
```

## Allow only specific extensions from Sourcegraph.com

On Sourcegraph Enterprise, you can set [`extensions.allowRemoteExtensions`](../config/site_config.md) so that only the explicitly specified extensions can be used from Sourcegraph.com:

```json
{
  "extensions": { "allowRemoteExtensions": ["chris/token-highlights"] }
}
```

## [Client-side security and privacy](../../extensions/security.md)

See "[Security and privacy of Sourcegraph extensions](../../extensions/security.md)" for information on the client-side security and privacy implications of Sourcegraph extensions.
