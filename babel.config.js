// @ts-check
const logger = require('gulplog')
const semver = require('semver')

/** @type {import('@babel/core').ConfigFunction} */
module.exports = api => {
  const isTest = api.env('test')
  api.cache.forever()

  /**
   * Whether to instrument files with istanbul for code coverage.
   * This is needed for e2e test coverage.
   */
  const instrument = Boolean(process.env.COVERAGE_INSTRUMENT && JSON.parse(process.env.COVERAGE_INSTRUMENT))
  if (instrument) {
    logger.info('Instrumenting code for coverage tracking')
  }

  return {
    presets: [
      // Can't put this in plugins because it needs to run as the last plugin.
      ...(instrument ? [{ plugins: [['babel-plugin-istanbul', { exclude: ['node_modules/**'] }]] }] : []),
      [
        '@babel/preset-env',
        {
          // Node (used for testing) doesn't support modules, so compile to CommonJS for testing.
          modules: isTest ? 'commonjs' : false,
          bugfixes: true,
          useBuiltIns: 'entry',
          include: [
            // Polyfill URL because Chrome and Firefox are not spec-compliant
            // Hostnames of URIs with custom schemes (e.g. git) are not parsed out
            'web.url',
            // URLSearchParams.prototype.keys() is not iterable in Firefox
            'web.url-search-params',
            // Commonly needed by extensions (used by vscode-jsonrpc)
            'web.immediate',
            // Avoids issues with RxJS interop
            'esnext.symbol.observable',
            // Webpack v4 chokes on optional chaining and nullish coalescing syntax, fix will be released with webpack v5.
            '@babel/plugin-proposal-optional-chaining',
            '@babel/plugin-proposal-nullish-coalescing-operator',
          ],
          // See https://github.com/zloirock/core-js#babelpreset-env
          corejs: semver.minVersion(require('./package.json').dependencies['core-js']),
        },
      ],
      '@babel/preset-typescript',
      '@babel/preset-react',
    ],
    plugins: [
      'babel-plugin-lodash',
      // Node 12 (released 2019 Apr 23) supports these natively, but there seem to be issues when used with TypeScript.
      ['@babel/plugin-proposal-class-properties', { loose: true }],
    ],
  }
}
