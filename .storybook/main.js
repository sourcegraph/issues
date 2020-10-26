const path = require('path')
const { remove } = require('lodash')
const { DefinePlugin, ProgressPlugin } = require('webpack')
const MonacoWebpackPlugin = require('monaco-editor-webpack-plugin')

const monacoEditorPaths = [path.resolve(__dirname, '..', 'node_modules', 'monaco-editor')]

const config = {
  stories: ['../client/**/*.story.tsx'],
  addons: ['@storybook/addon-knobs', '@storybook/addon-actions', 'storybook-addon-designs'],
  /**
   * @param config {import('webpack').Configuration}
   * @returns {import('webpack').Configuration}
   */
  webpackFinal: config => {
    // Include sourcemaps
    config.mode = 'development'
    config.devtool = 'cheap-module-eval-source-map'
    const definePlugin = config.plugins.find(plugin => plugin instanceof DefinePlugin)
    // @ts-ignore
    definePlugin.definitions.NODE_ENV = JSON.stringify('development')
    // @ts-ignore
    definePlugin.definitions['process.env'].NODE_ENV = JSON.stringify('development')

    // We don't use Storybook's default Babel config for our repo, it doesn't include everything we need.
    config.module.rules.splice(0, 1)

    if (process.env.CI) {
      remove(config.plugins, plugin => plugin instanceof ProgressPlugin)
    }

    config.module.rules.push({
      test: /\.tsx?$/,
      loader: require.resolve('babel-loader'),
      options: {
        configFile: path.resolve(__dirname, '..', 'babel.config.js'),
      },
    })

    config.plugins.push(
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
      })
    )

    const storybookDirectory = path.resolve(__dirname, '../node_modules/@storybook')

    // Put our style rules at the beginning so they're processed by the time it
    // gets to storybook's style rules.
    config.module.rules.unshift({
      test: /\.(sass|scss)$/,
      use: [
        'to-string-loader',
        'css-loader',
        {
          loader: 'postcss-loader',
        },
        {
          loader: 'sass-loader',
          options: {
            sassOptions: {
              includePaths: [path.resolve(__dirname, '..', 'node_modules')],
            },
          },
        },
      ],
      // Make sure Storybook styles get handled by the Storybook config
      exclude: storybookDirectory,
    })

    // Make sure Storybook style loaders are only evaluated for Storybook styles.
    config.module.rules.find(rule => rule.test?.toString() === /\.css$/.toString()).include = storybookDirectory

    config.module.rules.unshift({
      // CSS rule for external plain CSS (skip SASS and PostCSS for build perf)
      test: /\.css$/,
      // Make sure Storybook styles get handled by the Storybook config
      exclude: [storybookDirectory, ...monacoEditorPaths],
      use: ['to-string-loader', 'css-loader'],
    })

    config.module.rules.unshift({
      // CSS rule for monaco-editor, it expects styles to be loaded with `style-loader`.
      test: /\.css$/,
      include: monacoEditorPaths,
      // Make sure Storybook styles get handled by the Storybook config
      exclude: [storybookDirectory],
      use: ['style-loader', 'css-loader'],
    })
    config.module.rules.unshift({
      test: /\.ya?ml$/,
      use: ['raw-loader'],
    })

    Object.assign(config.entry, {
      'editor.worker': 'monaco-editor/esm/vs/editor/editor.worker.js',
      'json.worker': 'monaco-editor/esm/vs/language/json/json.worker',
    })

    return config
  },
}
module.exports = config
