// @ts-check

const path = require('path')

const ReactRefreshWebpackPlugin = require('@pmmmwh/react-refresh-webpack-plugin')
const CssMinimizerWebpackPlugin = require('css-minimizer-webpack-plugin')
const logger = require('gulplog')
const MiniCssExtractPlugin = require('mini-css-extract-plugin')
const MonacoWebpackPlugin = require('monaco-editor-webpack-plugin')
const TerserPlugin = require('terser-webpack-plugin')
const webpack = require('webpack')
const { BundleAnalyzerPlugin } = require('webpack-bundle-analyzer')
const { WebpackManifestPlugin } = require('webpack-manifest-plugin')

const { getCSSLoaders } = require('./dev/webpack/get-css-loaders')
const { getHTMLWebpackPlugins } = require('./dev/webpack/get-html-webpack-plugins')
const { isHotReloadEnabled } = require('./src/integration/environment')

const mode = process.env.NODE_ENV === 'production' ? 'production' : 'development'
logger.info('Using mode', mode)

const isDevelopment = mode === 'development'
const isProduction = mode === 'production'
const isCI = process.env.CI === 'true'
const isCacheEnabled = isDevelopment && !isCI

const devtool = isProduction ? 'source-map' : 'eval-cheap-module-source-map'

const shouldServeIndexHTML = process.env.WEBPACK_SERVE_INDEX === 'true'
if (shouldServeIndexHTML) {
  logger.info('Serving index.html with HTMLWebpackPlugin')
}
const webServerEnvironmentVariables = {
  WEBPACK_SERVE_INDEX: JSON.stringify(process.env.WEBPACK_SERVE_INDEX),
  SOURCEGRAPH_API_URL: JSON.stringify(process.env.SOURCEGRAPH_API_URL),
}

const shouldAnalyze = process.env.WEBPACK_ANALYZER === '1'
if (shouldAnalyze) {
  logger.info('Running bundle analyzer')
}

const rootPath = path.resolve(__dirname, '..', '..')
const hotLoadablePaths = ['branded', 'shared', 'web', 'wildcard'].map(workspace =>
  path.resolve(rootPath, 'client', workspace, 'src')
)
const nodeModulesPath = path.resolve(rootPath, 'node_modules')
const monacoEditorPaths = [path.resolve(nodeModulesPath, 'monaco-editor')]

const isEnterpriseBuild = process.env.ENTERPRISE && Boolean(JSON.parse(process.env.ENTERPRISE))
const enterpriseDirectory = path.resolve(__dirname, 'src', 'enterprise')

const babelLoader = {
  loader: 'babel-loader',
  options: {
    cacheDirectory: true,
    configFile: path.join(__dirname, 'babel.config.js'),
  },
}

const extensionHostWorker = /main\.worker\.ts$/

/** @type {import('webpack').Configuration} */
const config = {
  context: __dirname, // needed when running `gulp webpackDevServer` from the root dir
  mode,
  stats: {
    // Minimize logging in case if Webpack is used along with multiple other services.
    // Use `normal` output preset in case of running standalone web server.
    preset: shouldServeIndexHTML || isProduction ? 'normal' : 'errors-warnings',
    errorDetails: true,
    timings: true,
  },
  infrastructureLogging: {
    // Controls webpack-dev-server logging level.
    level: 'warn',
  },
  target: 'browserslist',
  // Use cache only in `development` mode to speed up production build.
  cache: isCacheEnabled && {
    type: 'filesystem',
    buildDependencies: {
      // Invalidate cache on config change.
      config: [
        __filename,
        path.resolve(__dirname, 'babel.config.js'),
        path.resolve(rootPath, 'babel.config.js'),
        path.resolve(rootPath, 'postcss.config.js'),
      ],
    },
  },
  optimization: {
    minimize: isProduction,
    minimizer: [
      new TerserPlugin({
        terserOptions: {
          compress: {
            // Don't inline functions, which causes name collisions with uglify-es:
            // https://github.com/mishoo/UglifyJS2/issues/2842
            inline: 1,
          },
        },
      }),
      new CssMinimizerWebpackPlugin(),
    ],
    ...(isDevelopment && {
      // Running multiple entries on a single page that do not share a runtime chunk from the same compilation is not supported.
      // https://github.com/webpack/webpack-dev-server/issues/2792#issuecomment-808328432
      runtimeChunk: isHotReloadEnabled ? 'single' : false,
      removeAvailableModules: false,
      removeEmptyChunks: false,
      splitChunks: false,
    }),
  },
  entry: {
    // Enterprise vs. OSS builds use different entrypoints. The enterprise entrypoint imports a
    // strict superset of the OSS entrypoint.
    app: isEnterpriseBuild ? path.join(enterpriseDirectory, 'main.tsx') : path.join(__dirname, 'src', 'main.tsx'),
    'editor.worker': 'monaco-editor/esm/vs/editor/editor.worker.js',
    'json.worker': 'monaco-editor/esm/vs/language/json/json.worker',
  },
  output: {
    path: path.join(rootPath, 'ui', 'assets'),
    // Do not [hash] for development -- see https://github.com/webpack/webpack-dev-server/issues/377#issuecomment-241258405
    filename: mode === 'production' ? 'scripts/[name].[contenthash].bundle.js' : 'scripts/[name].bundle.js',
    chunkFilename: mode === 'production' ? 'scripts/[id]-[contenthash].chunk.js' : 'scripts/[id].chunk.js',
    publicPath: '/.assets/',
    globalObject: 'self',
    pathinfo: false,
  },
  devtool,
  plugins: [
    // Needed for React
    new webpack.DefinePlugin({
      'process.env': {
        NODE_ENV: JSON.stringify(mode),
        ...(shouldServeIndexHTML && webServerEnvironmentVariables),
      },
    }),
    new webpack.ProvidePlugin({
      process: 'process/browser',
      // Based on the issue: https://github.com/webpack/changelog-v5/issues/10
      Buffer: ['buffer', 'Buffer'],
    }),
    new MiniCssExtractPlugin({
      // Do not [hash] for development -- see https://github.com/webpack/webpack-dev-server/issues/377#issuecomment-241258405
      filename: mode === 'production' ? 'styles/[name].[contenthash].bundle.css' : 'styles/[name].bundle.css',
    }),
    new MonacoWebpackPlugin({
      languages: ['json'],
      features: [
        'bracketMatching',
        'clipboard',
        'coreCommands',
        'cursorUndo',
        'find',
        'format',
        'hover',
        'inPlaceReplace',
        'iPadShowKeyboard',
        'links',
        'suggest',
      ],
    }),
    !shouldServeIndexHTML &&
      new WebpackManifestPlugin({
        writeToFileEmit: true,
        fileName: 'webpack.manifest.json',
        // Only output files that are required to run the application
        filter: ({ isInitial }) => isInitial,
      }),
    ...(shouldServeIndexHTML ? getHTMLWebpackPlugins() : []),
    shouldAnalyze && new BundleAnalyzerPlugin(),
    isHotReloadEnabled && new webpack.HotModuleReplacementPlugin(),
    isHotReloadEnabled && new ReactRefreshWebpackPlugin({ overlay: false }),
  ].filter(Boolean),
  resolve: {
    extensions: ['.mjs', '.ts', '.tsx', '.js', '.json'],
    mainFields: ['es2015', 'module', 'browser', 'main'],
    alias: {
      // react-visibility-sensor's main field points to a UMD bundle instead of ESM
      // https://github.com/joshwnj/react-visibility-sensor/issues/148
      'react-visibility-sensor': path.resolve(rootPath, 'node_modules/react-visibility-sensor/visibility-sensor.js'),
    },
  },
  module: {
    rules: [
      // Run hot-loading-related Babel plugins on our application code only (because they'd be
      // slow to run on all JavaScript code).
      {
        test: /\.[jt]sx?$/,
        include: hotLoadablePaths,
        exclude: extensionHostWorker,
        use: [
          ...(isProduction ? ['thread-loader'] : []),
          {
            loader: 'babel-loader',
            options: {
              cacheDirectory: true,
              ...(isHotReloadEnabled && { plugins: ['react-refresh/babel'] }),
            },
          },
        ],
      },
      {
        test: /\.[jt]sx?$/,
        exclude: [...hotLoadablePaths, extensionHostWorker],
        use: [...(isProduction ? ['thread-loader'] : []), babelLoader],
      },
      {
        test: /\.(sass|scss)$/,
        // CSS Modules loaders are only applied when the file is explicitly named as CSS module stylesheet using the extension `.module.scss`.
        include: /\.module\.(sass|scss)$/,
        use: getCSSLoaders(rootPath, isDevelopment, {
          loader: 'css-loader',
          options: {
            sourceMap: isDevelopment,
            url: false,
            modules: {
              exportLocalsConvention: 'camelCase',
              localIdentName: '[name]__[local]_[hash:base64:5]',
            },
          },
        }),
      },
      {
        test: /\.(sass|scss)$/,
        exclude: /\.module\.(sass|scss)$/,
        use: getCSSLoaders(rootPath, isDevelopment, { loader: 'css-loader', options: { url: false } }),
      },
      {
        // CSS rule for monaco-editor and other external plain CSS (skip SASS and PostCSS for build perf)
        test: /\.css$/,
        include: monacoEditorPaths,
        use: ['style-loader', { loader: 'css-loader' }],
      },
      {
        // TTF rule for monaco-editor
        test: /\.ttf$/,
        include: monacoEditorPaths,
        type: 'asset/resource',
      },
      {
        test: extensionHostWorker,
        use: [{ loader: 'worker-loader', options: { inline: 'no-fallback' } }, babelLoader],
      },
      { test: /\.ya?ml$/, type: 'asset/source' },
    ],
  },
}

module.exports = config
