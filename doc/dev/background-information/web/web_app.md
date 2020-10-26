# Developing the Sourcegraph web app

Guide to contribute to the Sourcegraph webapp. Please also see our general [TypeScript documentation](https://about.sourcegraph.com/handbook/engineering/languages/typescript).

## Naming files

If the file only contains one main export (e.g. a component class + some interfaces), name the file like the main export.
This name is PascalCase if the main export is a class.
This makes it easy to find it with file search.
If the file has no main export (e.g. a file with utility functions), give the file a name that groups the exports semantically.
This name is usually short and lowercase or kebab-case, e.g. `util/errors.ts` contains error utilities.
Avoid adding utilities into a `util.ts` file, it is doomed to become a mess over time.

### `.ts` vs `.tsx`

You must use the `tsx` file extension if the file contains TSX (React) syntax.
You should use the normal `ts` extension if it does not.
The `tsx` extension makes certain generic syntax impossible and also enables emmet suggestions in editors, which are annoying in normal TypeScript code.

### `index.*` files

Index files should not never contain declarations on their own.
Their purpose is to reexport symbols from a number of other files to make imports easier and define the the public API.

## Components

- Try to do one component per file. This makes it easy to encapsulate corresponding styles.
- Don't pass arrow functions as React bindings unless unavoidable

## Styles

- Styles are written in SCSS
- Every component .tsx file should have a corresponding stylesheet named like the .tsx file
  - The stylesheet must contain a top-level selector to scope it to a class that is the kebab-case version of the component name.
    The component must apply that class to its top-level element.
- Use [BEM](http://getbem.com/). "Block" here is the component name, element a non-component child of the component.
- Only use descendent/child selectors where unavoidable. Prefer BEM-style class names that are nested in SCSS through the `&` operator
- Create utility classes for styles that should be shared horizontally between components
- Always use `rem` units (when converting designs, `1rem` = `16px`). This allows us to scale the whole UI by modifying the root font size.
- Avoid hardcoding colors, use SCSS variables if they are available / the color makes sense to share.
- Try to _minimize_ the usage of advanced SCSS features. They can lead to bugs and complicate styles.
  - Encouraged features are nesting and imports (which is the intersection of Less', SCSS' and PostCSS' feature set)
- Think about mobile at least so much that no feature breaks when the browser window is resized
- Don't couple the styles to the DOM structure. It should be possible to change the structure without changing the styles and vice versa.
- Prefer flexbox over absolute positioning
- Avoid styling the children of your components. This couples your component to the implementation of the child
- Order your rules so that layout rules (that describe how the component is laid out to its parents) come first, then rules that describe the layout of its children, and finally visual details.

## Code splitting

[Code splitting](https://reactjs.org/docs/code-splitting.html) refers to the practice of bundling the frontend code into multiple JavaScript files, each with a subset of the frontend code, so that the client only needs to download and parse/run the code necessary for the page they are viewing. We use the [react-router code splitting technique](https://reactjs.org/docs/code-splitting.html#route-based-code-splitting) to accomplish this.

When adding a new route (and new components), you can opt into code splitting for it by referring to a lazy-loading reference to the component (instead of a static import binding of the component). To create the lazy-loading reference to the component, use `React.lazy`, as in:

``` typescript
const MyComponent = React.lazy(async () => ({
    default: (await import('./path/to/MyComponent)).MyComponent,
}))
```

If you don't do this (and just use a normal `import`), it will still work, but it will increase the initial bundle size for all users.

(It is necessary to return the component as the `default` property of an object because `React.lazy` is hard-coded to look there for it. We could avoid this extra work by using default exports, but we chose not to use default exports ([reasons](https://blog.neufund.org/why-we-have-banned-default-exports-and-you-should-do-the-same-d51fdc2cf2ad)).)

### Theming

Theming is done through toggling top-level CSS classes `theme-light` and `theme-dark`.
Any style can be made different on either theme by scoping it to one of those two classes.
Where possible, we use CSS variables, but unfortunately they don't work with compile-time color manipulation (`darken()` etc)
and runtime color manipulation is not yet implemented in CSS (coming in CSS Color Level 4).

## Formatting

We use [Prettier](https://github.com/prettier/prettier) so you never have to worry about how to format your code.
`yarn run prettier` will check & autoformat all code.

## Tests

We write unit tests and e2e tests.

### Unit tests

Unit tests are for things that can be tested in isolation; you provide inputs and make assertion on the outputs and/or side effects.

React component snapshot tests are a special kind of unit test that we use to test React components. See "[React component snapshot tests](../testing.md#react-component-snapshot-tests)" for more information.

You can run unit tests via `yarn test` (to run all) or `yarn test --watch` (to run only tests changed since the last commit). See "[Testing](../testing.md)" for more information.

### E2E tests

See [testing.md](../testing.md).
