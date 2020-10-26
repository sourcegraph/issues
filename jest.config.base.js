// @ts-check

const path = require('path')

// Use the same locale for test runs so that snapshots generated using code that
// uses Intl or toLocaleString() are consistent.
//
// We have to do this at this point instead of in the normal Jest setup hooks
// because ICU (and therefore Intl) only reads the environment once:
// specifically, at the point any Intl object is instantiated. Jest indirectly
// uses a locale-sensitive sort while setting up test suites _before_ invoking
// the setup hooks, so we have no opportunity to change $LANG outside of this
// ugly side effect. (This is especially evident when running tests in-band.)
process.env.LANG = 'en_US.UTF-8'

/** @type {jest.InitialOptions} */
const config = {
  // uses latest jsdom and exposes jsdom as a global,
  // for example to change the URL in window.location
  testEnvironment: __dirname + '/client/shared/dev/jest-environment.js',

  collectCoverage: !!process.env.CI,
  collectCoverageFrom: ['<rootDir>/src/**/*.{ts,tsx}'],
  coverageDirectory: '<rootDir>/coverage',
  coveragePathIgnorePatterns: [/\/node_modules\//.source, /\.(test|story)\.tsx?$/.source, /\.d\.ts$/.source],
  roots: ['<rootDir>/src'],

  // Transform packages that do not distribute CommonJS packages (typically because they only distribute ES6
  // modules). If you get an error from jest like "Jest encountered an unexpected token. ... SyntaxError:
  // unexpected token import/export", then add it here. See
  // https://github.com/facebook/create-react-app/issues/5241#issuecomment-426269242 for more information on why
  // this is necessary.
  transformIgnorePatterns: [
    '/node_modules/(?!abortable-rx|@sourcegraph/react-loading-spinner|@sourcegraph/codeintellify|@sourcegraph/comlink|monaco-editor)',
  ],

  moduleNameMapper: {
    '\\.s?css$': 'identity-obj-proxy',
    '\\.ya?ml$': 'identity-obj-proxy',
    '^worker-loader': 'identity-obj-proxy',
    // monaco-editor uses the "module" field in package.json, which isn't supported by Jest
    // https://github.com/facebook/jest/issues/2702
    // https://github.com/Microsoft/monaco-editor/issues/996
    '^monaco-editor': 'monaco-editor/esm/vs/editor/editor.main.js',
  },

  // By default, don't clutter `yarn test --watch` output with the full coverage table. To see it, use the
  // `--coverageReporters text` jest option.
  coverageReporters: ['json', 'lcov', 'text-summary'],

  setupFiles: [
    path.join(__dirname, 'client/shared/dev/mockDate.js'),
    // Needed for reusing API functions that use fetch
    // Neither NodeJS nor JSDOM have fetch + AbortController yet
    require.resolve('abort-controller/polyfill'),
    path.join(__dirname, 'client/shared/dev/fetch'),
    path.join(__dirname, 'client/shared/dev/setLinkComponentForTest.ts'),
    // Enzyme setup file
    path.join(__dirname, 'client/shared/dev/enzymeSetup.js'),
  ],
  setupFilesAfterEnv: [require.resolve('core-js/stable'), require.resolve('regenerator-runtime/runtime')],
  globalSetup: path.join(__dirname, 'client/shared/dev/jestGlobalSetup.js'),
  snapshotSerializers: [path.join(__dirname, 'client/shared/dev/enzymeSerializer.js')],
}

module.exports = config
