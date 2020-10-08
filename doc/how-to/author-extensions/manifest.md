# Sourcegraph extension manifest - package.json

Sourcegraph extensions use a `package.json` file for metadata and configuration.

## Fields

Name | Required | Type | Details
---- |:--------:| ---- | -------
`name` | ✔️ | `string` | Extension identifier: all lowercase, alphanumeric with hyphens and underscores.
`description` | ✔️ | `string` | The extension's description, which summarizes the extension's purpose and features.
`version` | | `string` | [Semantic versioning](https://semver.org/) format.
`publisher` | ✔️ | `string` | Your [Sourcegraph username](development_environment.md#sourcegraph-com-account-and-the-sourcegraph-cli) (or the name of an organization you're a member of)
`license` | | `string` | The type of license chosen.
`main` | | `string` | Path to the transpiled JavaScript file for your extension.
`contributes` | | `object` | An object describing the contributions (features) this extension provides. See "[Extension contribution points](contributions.md)" for a full listing.
[`activationEvents`](activation.md) | ✔️ | `array` | A list of events that cause this extension to be activated.
`dependencies` | | `object` | npm dependencies.
`devDependencies` | | `object` | npm dependencies needed for development.
`scripts` | ✔️ | `object` | npm's scripts with Sourcegraph specific entries such as `sourcegraph:prepublish`.
`browserslist` | | `string` | Modern list of browsers for build tools to target when transpiling.
`repository` | | `object` | npm field for the repository location.
`categories` | | `string[]` | Categories that describe this extension, from the predefined set [`Programming languages`](https://sourcegraph.com/extensions?query=category%3A%22Programming+languages%22), `Linters`, `Code analysis`, `External services`, `Reports and stats`, `Other`.
`tags` | | `string[]` | Arbitrary tags that describe this extension.
`wip` | | `boolean`| Indicates that this is a [work-in-progress extension](publishing.md#wip-extensions).

See the [npm package.json documentation](https://docs.npmjs.com/creating-a-package-json-file) for other fields.

**Note:** Including the `repository` field is recommended so anyone can follow the link from the extension detail page to view the source code.

```json
"repository": {
  "type": "git",
  "url": "https://github.com/sourcegraph/sourcegraph-codecov.git"
}
```

## Example

Here is an example `package.json` created by the [Sourcegraph extension creator](creating.md#creating-an-extension-the-easy-way).

```json
{
  "name": "my-extension",
  "description": "An awesome Sourcegraph extension",
  "publisher": "your-sourcegraph-username",
  "repository": {
    "type": "git",
    "url": "https://github.com/example/repo"
  },
  "categories": ["Programming languages"],
  "tags": ["awesome"],
  "activationEvents": [
    "*"
  ],
  "contributes": {
    "actions": [
      {}
    ],
    "menus": {
      "editor/title": [],
      "commandPalette": []
    },
    "configuration": {}
  },
  "version": "0.0.0-DEVELOPMENT",
  "license": "MIT",
  "main": "dist/my-extension.js",
  "scripts": {
    "typecheck": "tsc -p tsconfig.json",
    "build": "parcel build --out-file dist/my-extension.js src/my-extension.ts",
    "serve": "parcel serve --no-hmr --out-file dist/my-extension.js src/my-extension.ts",
    "watch:typecheck": "tsc -p tsconfig.json -w",
    "watch:build": "tsc -p tsconfig.dist.json -w",
    "sourcegraph:prepublish": "npm run build"
  },
  "browserslist": [
    "last 1 Chrome versions",
    "last 1 Firefox versions",
    "last 1 Edge versions",
    "last 1 Safari versions"
  ],
  "devDependencies": {
    "@sourcegraph/tsconfig": "^4.0.1",
    "@sourcegraph/eslint-config": "^0.11.3",
    "parcel-bundler": "^1.12.4",
    "sourcegraph": "^24.0.0",
    "eslint": "^6.8.0",
    "typescript": "^3.8.3"
  }
}
```
