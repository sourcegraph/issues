// @ts-check

const gulp = require('gulp')
const {
  graphQlSchema,
  graphQlOperations,
  schema,
  watchGraphQlSchema,
  watchGraphQlOperations,
  watchSchema,
} = require('./client/shared/gulpfile')
const { webpack: webWebpack, webpackDevServer: webWebpackDevServer } = require('./client/web/gulpfile')

/**
 * Generates files needed for builds.
 */
const generate = gulp.parallel(schema, graphQlSchema, graphQlOperations)

/**
 * Generates files needed for builds whenever files change.
 */
const watchGenerate = gulp.series(generate, gulp.parallel(watchSchema, watchGraphQlSchema, watchGraphQlOperations))

/**
 * Builds everything.
 */
const build = gulp.series(generate, webWebpack)

/**
 * Watches everything and rebuilds on file changes.
 */
const dev = gulp.parallel(watchGenerate, webWebpackDevServer)

module.exports = {
  generate,
  watchGenerate,
  build,
  dev,
  schema,
  graphQlSchema,
  watchGraphQlSchema,
  graphQlOperations,
  watchGraphQlOperations,
}
