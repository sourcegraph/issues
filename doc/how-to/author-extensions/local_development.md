# Local development (sideloading)

When developing an extension, you can sideload it from your local development machine's Parcel dev server (instead of re-publishing it after each code change). This speeds up the development cycle and avoids breaking the published version of your extension. This works on [Sourcegraph.com](https://sourcegraph.com/search) and self-hosted instances.

To set this up:

1. In your extension's directory, run `npm run serve` to run the Parcel dev server. Wait until it reports that it's listening.
2. Reveal the **Ext ▲** debug menu by running the following JavaScript code in your browser's devtools console on a Sourcegraph page: `localStorage.debug=true;location.reload()`.
3. In the **Ext ▲** debug menu, click **Sideload Extension -> Load Extension**.
3. Enter the URL the Parcel dev server is listening on.
4. Your extension should appear in the debug menu's "active extensions" list. If it doesn't, there may have been an error when activating your extension - check the debug console for error messages.

After doing this, the development cycle is as follows:

1. Make a change to your extension's code, then save the file.
2. Reload your browser window. (It will fetch the package.json and the newly compiled JavaScript bundle for your extension.)

When you're done, clear the sideload URL from the extensions debug menu.

**Note:** This workflow assumes that, when running the Parcel dev server, a symlink exists in the `dist/` directory pointing to your `package.json`. If you [created your extension the easy way](creating.md#creating-an-extension-the-easy-way), this is already set up for you. Otherwise, follow these steps:

1. Add `mkdirp` and `lnfs-cli` as dependencies (`npm install --save-dev mkdirp lnfs-cli`).
2. Add the following npm script to your `package.json`:

    ```
    "symlink-package": "mkdirp dist && lnfs ./package.json ./dist/package.json"
    ```

3. Edit the `serve` npm script to run `symlink-package`:

    ```
    "serve": "npm run symlink-package && parcel serve --no-hmr --out-file dist/your-extension.js src/your-extension.ts"
    ```

## Next steps

- [Publishing an extension](publishing.md)
- [Extension activation](activation.md)
- [Extension manifest (configuration)](manifest.md)
