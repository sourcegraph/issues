# Creating a Sourcegraph extension

First, [set up your development environment](development_environment.md) so you're ready for creating and publishing.

## What is a Sourcegraph extension?

A Sourcegraph extension is a single JavaScript file that has an exported `activate` function. It is called by the extension runtime if the extension's [activation](activation.md) conditions are satisfied.

```javascript
// my-extension.js
export function activate() {
  console.log('my-extension activated');
}
```

A build tool, such as Parcel, bundles this code for module loading and puts the exported file into a `dist` directory. A `package.json` is required for dependencies, configuration, and metadata. Now the extension is ready for publishing.

You can use any build tool you wish, so long as it meets these requirements.

## Creating an extension the easy way

The easiest way to get an extension ready to publish is to use the [Sourcegraph extension creator](https://github.com/sourcegraph/create-extension):

```bash
mkdir my-extension
cd my-extension
npm init sourcegraph-extension
```

Follow the prompts, and when complete, you'll have the following files:

```bash
├── README.md
├── node_modules
├── package-lock.json
├── package.json
├── src
│   └── my-extension.ts
├── tsconfig.json
└── .eslintrc.json
```

### Description of generated files

#### The src directory and activate function

A `src` directory has been created; it contains a TypeScript file with an exported `activate` function.

For simplicity, the extension will always activate. See the [activation documentation](activation.md) to configure your extension's activation.

For code layout, a single TypeScript/JavaScript file is usually all you'll need. For larger projects, create multiples files in the `src` directory, and Parcel will bundle them into a single JavaScript file.

#### README.md

The `README.md` is the content for your extension page in the [extensions registry](https://sourcegraph.com/extensions). See the [Codecov extension](https://sourcegraph.com/extensions/sourcegraph/codecov) for a great example.

#### package.json

The Sourcegraph extension creator generates a minimal and production ready `package.json` used for [extension metadata and configuration](manifest.md).

#### .eslintrc.json and tsconfig.json

These are configuration files for linting and TypeScript compilation and will be sufficient for most extensions.

## Next steps

- [Local development](local_development.md)
- [Publishing an extension](publishing.md)
- [Extension activation](activation.md)
- [Extension manifest (configuration)](manifest.md)
