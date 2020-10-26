// @ts-check

const MiniCssExtractPlugin = require('mini-css-extract-plugin')
const MonacoWebpackPlugin = require('monaco-editor-webpack-plugin')
const OptimizeCssAssetsPlugin = require('optimize-css-assets-webpack-plugin')
const path = require('path')
const TerserPlugin = require('terser-webpack-plugin')
const webpack = require('webpack')
const logger = require('gulplog')
const { BundleAnalyzerPlugin } = require('webpack-bundle-analyzer')

const mode = process.env.NODE_ENV === 'production' ? 'production' : 'development'
logger.info('Using mode', mode)

const devtool = mode === 'production' ? 'source-map' : 'cheap-module-eval-source-map'

const shouldAnalyze = process.env.WEBPACK_ANALYZER === '1'
if (shouldAnalyze) {
  logger.info('Running bundle analyzer')
}

const rootDirectory = path.resolve(__dirname, '..', '..')
const nodeModulesPath = path.resolve(rootDirectory, 'node_modules')
const monacoEditorPaths = [path.resolve(nodeModulesPath, 'monaco-editor')]

const isEnterpriseBuild = !!process.env.ENTERPRISE
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
  optimization: {
    minimize: mode === 'production',
    minimizer: [
      new TerserPlugin({
        sourceMap: true,
        terserOptions: {
          compress: {
            // // Don't inline functions, which causes name collisions with uglify-es:
            // https://github.com/mishoo/UglifyJS2/issues/2842
            inline: 1,
          },
        },
      }),
    ],
    namedModules: false,

    ...(mode === 'development'
      ? {
          removeAvailableModules: false,
          removeEmptyChunks: false,
          splitChunks: false,
        }
      : {}),
  },
  entry: {
    // Enterprise vs. OSS builds use different entrypoints. The enterprise entrypoint imports a
    // strict superset of the OSS entrypoint.
    app: [
      'react-hot-loader/patch',
      isEnterpriseBuild ? path.join(enterpriseDirectory, 'main.tsx') : path.join(__dirname, 'src', 'main.tsx'),
    ],

    'editor.worker': 'monaco-editor/esm/vs/editor/editor.worker.js',
    'json.worker': 'monaco-editor/esm/vs/language/json/json.worker',
  },
  output: {
    path: path.join(rootDirectory, 'ui', 'assets'),
    filename: 'scripts/[name].bundle.js',
    chunkFilename: 'scripts/[id]-[contenthash].chunk.js',
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
      },
    }),
    new MiniCssExtractPlugin({ filename: 'styles/[name].bundle.css' }),
    new OptimizeCssAssetsPlugin(),
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
    new webpack.IgnorePlugin(/\.flow$/, /.*/),
    ...(shouldAnalyze ? [new BundleAnalyzerPlugin()] : []),
  ],
  resolve: {
    extensions: ['.mjs', '.ts', '.tsx', '.js'],
    mainFields: ['es2015', 'module', 'browser', 'main'],
    alias: {
      // react-visibility-sensor's main field points to a UMD bundle instead of ESM
      // https://github.com/joshwnj/react-visibility-sensor/issues/148
      'react-visibility-sensor': path.resolve(
        rootDirectory,
        'node_modules/react-visibility-sensor/visibility-sensor.js'
      ),
    },
  },
  module: {
    rules: [
      // Run hot-loading-related Babel plugins on our application code only (because they'd be
      // slow to run on all JavaScript code).
      {
        test: /\.[jt]sx?$/,
        include: path.join(__dirname, 'src'),
        exclude: extensionHostWorker,
        use: [
          ...(mode === 'production' ? ['thread-loader'] : []),
          {
            loader: 'babel-loader',
            options: {
              cacheDirectory: true,
              plugins: [
                'react-hot-loader/babel',
                [
                  '@sourcegraph/babel-plugin-transform-react-hot-loader-wrapper',
                  {
                    modulePattern: 'web/src/.*\\.tsx$',
                    componentNamePattern: '(Page|Area)$',
                  },
                ],
              ],
            },
          },
        ],
      },
      {
        test: /\.[jt]sx?$/,
        exclude: [path.join(__dirname, 'src'), extensionHostWorker],
        use: [...(mode === 'production' ? ['thread-loader'] : []), babelLoader],
      },
      {
        test: /\.(sass|scss)$/,
        use: [
          mode === 'production' ? MiniCssExtractPlugin.loader : 'style-loader',
          'css-loader',
          'postcss-loader',
          {
            loader: 'sass-loader',
            options: {
              sassOptions: {
                implementation: require('sass'),
                includePaths: [nodeModulesPath],
              },
            },
          },
        ],
      },
      {
        // CSS rule for monaco-editor and other external plain CSS (skip SASS and PostCSS for build perf)
        test: /\.css$/,
        include: monacoEditorPaths,
        use: ['style-loader', 'css-loader'],
      },
      {
        test: extensionHostWorker,
        use: [{ loader: 'worker-loader', options: { inline: 'no-fallback' } }, babelLoader],
      },
      { test: /\.ya?ml$/, use: ['raw-loader'] },
    ],
  },
}

module.exports = config
