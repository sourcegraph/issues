# Using Sourcegraph extensions

[Sourcegraph extensions](index.md) add the following kinds of features to Sourcegraph and (using the [Chrome/Firefox browser extensions](../integration/browser_extension.md)) your code host and review tools:

- Go to definition
- Find references
- Hovers
- Line overlays, decorations, and annotations
- Toolbar buttons

The [Sourcegraph.com extension registry](https://sourcegraph.com/extensions) contains all publicly available Sourcegraph extensions. You can use these extensions on Sourcegraph.com and on your own self-hosted Sourcegraph instance.

If you've used Sourcegraph before, you've used a Sourcegraph extension. For example, try hovering over tokens or toggling Git blame on [`tuf_store.go`](https://sourcegraph.com/github.com/theupdateframework/notary/-/blob/server/storage/tuf_store.go). Or see a [demo video of code coverage overlays](https://www.youtube.com/watch?v=j1eWBa3rWH8). These features (and many others) are provided by extensions. You could improve these features, or add new features, just by improving or creating extensions to Sourcegraph.

Screenshots of test coverage and Git blame extensions:

<div style="text-align:center;margin:20px 0;display:flex">
<a href="https://github.com/sourcegraph/sourcegraph-codecov" target="_blank"><img src="https://user-images.githubusercontent.com/1976/45107396-53d56880-b0ee-11e8-96e9-ca83e991101c.png" style="padding:15px"></a>
<a href="https://github.com/sourcegraph/sourcegraph-git-extras" target="_blank"><img src="https://user-images.githubusercontent.com/1976/47624533-f3a1e800-dada-11e8-81d9-3d4bd67fc08a.png" style="padding:15px"></a>
</div>

## Usage

To view all available extensions on your Sourcegraph instance, click **User menu > Extensions** in the top navigation bar.

To enable/disable an extension for yourself, click **User menu > Extensions**, find the extension, and toggle the slider.

After enabling a Sourcegraph extension, it is immediately ready to use. Of course, some extensions only activate for certain files (e.g., the Python extension only adds code intelligence for `.py` files).

### On your code host

Install the [browser extension](../integration/browser_extension.md) and point it to your Sourcegraph instance (or Sourcegraph.com) in its options menu. It will consult your Sourcegraph user settings and activate the extensions you've enabled whenever you view code or diffs on your code host or review tool.

### For organizations

To enable/disable an extension for all organization members, add it to the `extensions` object in organization settings (as shown below).

```json
{
  ...,
  "extensions": {
    ...,
    "alice/myextension": true, // or false to disable
    ...
  },
  ...
}
```

### For all users

On a self-hosted Sourcegraph instance, add the same JSON above to global settings (in **Site admin > Global settings**).

<div style="text-align:center;margin:20px 0;display:flex">
<img src="https://sourcegraphstatic.com/docs/images/extensions/all-users-global-settings.png" style="padding:15px"></a>
</div>


