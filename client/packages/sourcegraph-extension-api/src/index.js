// This is the module that is bundled with extensions when they use `import ... from 'sourcegraph'` or
// `require('sourcegraph')`. It delegates to the extension host's runtime implementation of this module by calling
// `global.require` (which ensures that the extension host's `require` is called at runtime).
//
// This dummy file is used when the extension is bundled with a JavaScript bundler that lacks support for externals
// (or when `sourcegraph` is not configured as an external module). Parcel does not support externals
// (https://github.com/parcel-bundler/parcel/issues/144). Webpack, Rollup, and Microbundle support externals.
module.exports = global.require('sourcegraph')
