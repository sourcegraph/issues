import path from 'path'

import { Options } from '@storybook/core-common'
import CaseSensitivePathsPlugin from 'case-sensitive-paths-webpack-plugin'
import { remove } from 'lodash'
import signale from 'signale'
import SpeedMeasurePlugin from 'speed-measure-webpack-plugin'
import TerserPlugin from 'terser-webpack-plugin'
import webpack, {
    DllReferencePlugin,
    Configuration,
    DefinePlugin,
    ProgressPlugin,
    RuleSetUseItem,
    RuleSetUse,
    RuleSetRule,
} from 'webpack'
import { BundleAnalyzerPlugin } from 'webpack-bundle-analyzer'

import { ensureDllBundleIsReady } from './dllPlugin'
import { environment } from './environment-config'
import {
    rootPath,
    monacoEditorPath,
    dllPluginConfig,
    dllBundleManifestPath,
    getMonacoCSSRule,
    getMonacoTTFRule,
    getMonacoWebpackPlugin,
    nodeModulesPath,
    getBasicCSSLoader,
    readJsonFile,
    storybookWorkspacePath,
} from './webpack.config.common'

const getStoriesGlob = (): string[] => {
    if (process.env.STORIES_GLOB) {
        return [path.resolve(rootPath, process.env.STORIES_GLOB)]
    }

    // Stories in `Chromatic.story.tsx` are guarded by the `isChromatic()` check. It will result in noop in all other environments.
    const chromaticStoriesGlob = path.resolve(rootPath, 'client/storybook/src/chromatic-story/Chromatic.story.tsx')

    // Due to an issue with constant recompiling (https://github.com/storybookjs/storybook/issues/14342)
    // we need to make the globs more specific (`(web|shared..)` also doesn't work). Once the above issue
    // is fixed, this can be removed and watched for `client/**/*.story.tsx` again.
    const directoriesWithStories = ['branded', 'browser', 'shared', 'web', 'wildcard']
    const storiesGlobs = directoriesWithStories.map(packageDirectory =>
        path.resolve(rootPath, `client/${packageDirectory}/src/**/*.story.tsx`)
    )

    return [...storiesGlobs, chromaticStoriesGlob]
}

const getCSSLoaders = (...loaders: RuleSetUseItem[]): RuleSetUse => [
    ...loaders,
    'postcss-loader',
    {
        loader: 'sass-loader',
        options: {
            sassOptions: {
                includePaths: [path.resolve(rootPath, 'node_modules'), path.resolve(rootPath, 'client')],
            },
        },
    },
]

const getDllScriptTag = (): string => {
    ensureDllBundleIsReady()
    signale.await('Waiting for Webpack to compile Storybook preview.')

    const dllManifest = readJsonFile(dllBundleManifestPath) as Record<string, string>

    return `
        <!-- Load JS bundle created by DLL_PLUGIN  -->
        <script src="/dll-bundle/${dllManifest['dll.js']}"></script>
    `
}

const config = {
    stories: getStoriesGlob(),
    addons: [
        '@storybook/addon-knobs',
        '@storybook/addon-actions',
        'storybook-addon-designs',
        'storybook-dark-mode',
        '@storybook/addon-a11y',
        '@storybook/addon-toolbars',
    ],

    core: {
        builder: 'webpack5',
    },

    features: {
        // Explicitly disable the deprecated, not used postCSS support,
        // so no warning is rendered on each start of storybook.
        postcss: false,
    },

    typescript: {
        check: false,
        reactDocgen: false,
    },

    // Include DLL bundle script tag into preview-head.html if DLLPlugin is enabled.
    previewHead: (head: string) => `
        ${head}
        ${environment.isDLLPluginEnabled ? getDllScriptTag() : ''}
    `,

    webpackFinal: (config: Configuration, options: Options) => {
        config.stats = 'errors-warnings'
        config.mode = environment.shouldMinify ? 'production' : 'development'

        // Check the default config is in an expected shape.
        if (!config.module?.rules || !config.plugins) {
            throw new Error(
                'The format of the default storybook webpack config changed, please check if the config in ./src/main.ts is still valid'
            )
        }

        config.plugins.push(
            new DefinePlugin({
                NODE_ENV: JSON.stringify(config.mode),
                'process.env.NODE_ENV': JSON.stringify(config.mode),
            }),
            new webpack.ProvidePlugin({
                process: 'process/browser',
                // Based on the issue: https://github.com/webpack/changelog-v5/issues/10
                Buffer: ['buffer', 'Buffer'],
            })
        )

        if (environment.shouldMinify) {
            if (!config.optimization) {
                throw new Error('The structure of the config changed, expected config.optimization to be not-null')
            }
            config.optimization.minimize = true
            config.optimization.minimizer = [
                new TerserPlugin({
                    terserOptions: {
                        compress: {
                            // Don't inline functions, which causes name collisions with uglify-es:
                            // https://github.com/mishoo/UglifyJS2/issues/2842
                            inline: 1,
                        },
                    },
                }),
            ]
        } else {
            // Use cache only in `development` mode to speed up production build.
            config.cache = {
                type: 'filesystem',
                buildDependencies: {
                    // Invalidate cache on config change.
                    config: [
                        __filename,
                        path.resolve(storybookWorkspacePath, 'babel.config.js'),
                        path.resolve(rootPath, 'babel.config.js'),
                        path.resolve(rootPath, 'postcss.config.js'),
                        path.resolve(__dirname, './webpack.config.dll.ts'),
                    ],
                },
            }
        }

        // We don't use Storybook's default Babel config for our repo, it doesn't include everything we need.
        config.module.rules.splice(0, 1)
        config.module.rules.unshift({
            test: /\.tsx?$/,
            loader: require.resolve('babel-loader'),
            options: {
                cacheDirectory: true,
                configFile: path.resolve(rootPath, 'babel.config.js'),
            },
        })

        const storybookPath = path.resolve(nodeModulesPath, '@storybook')

        // Put our style rules at the beginning so they're processed by the time it
        // gets to storybook's style rules.
        config.module.rules.unshift({
            test: /\.(sass|scss)$/,
            // Make sure Storybook styles get handled by the Storybook config
            exclude: [/\.module\.(sass|scss)$/, storybookPath],
            use: getCSSLoaders('@terminus-term/to-string-loader', getBasicCSSLoader()),
        })

        config.module?.rules.unshift({
            test: /\.(sass|scss)$/,
            include: /\.module\.(sass|scss)$/,
            exclude: storybookPath,
            use: getCSSLoaders('style-loader', {
                loader: 'css-loader',
                options: {
                    sourceMap: !environment.shouldMinify,
                    modules: {
                        exportLocalsConvention: 'camelCase',
                        localIdentName: '[name]__[local]_[hash:base64:5]',
                    },
                    url: false,
                },
            }),
        })

        // Make sure Storybook style loaders are only evaluated for Storybook styles.
        const cssRule = config.module.rules.find(
            (rule): rule is RuleSetRule => typeof rule !== 'string' && rule.test?.toString() === /\.css$/.toString()
        )
        if (!cssRule) {
            throw new Error('Cannot find original CSS rule')
        }
        cssRule.include = storybookPath

        config.module.rules.push({
            // CSS rule for external plain CSS (skip SASS and PostCSS for build perf)
            test: /\.css$/,
            // Make sure Storybook styles get handled by the Storybook config
            exclude: [storybookPath, monacoEditorPath],
            use: ['@terminus-term/to-string-loader', getBasicCSSLoader()],
        })

        config.module.rules.push({
            test: /\.ya?ml$/,
            type: 'asset/source',
        })

        // Disable `CaseSensitivePathsPlugin` by default to speed up development build.
        // Similar discussion: https://github.com/vercel/next.js/issues/6927#issuecomment-480579191
        remove(config.plugins, plugin => plugin instanceof CaseSensitivePathsPlugin)

        // Disable `ProgressPlugin` by default to speed up development build.
        // Can be re-enabled by setting `WEBPACK_PROGRESS_PLUGIN` env variable.
        if (!environment.isProgressPluginEnabled) {
            remove(config.plugins, plugin => plugin instanceof ProgressPlugin)
        }

        if (environment.isDLLPluginEnabled && !options.webpackStatsJson) {
            config.plugins.unshift(
                new DllReferencePlugin({
                    context: dllPluginConfig.context,
                    manifest: dllPluginConfig.path,
                })
            )
        } else {
            config.plugins.push(getMonacoWebpackPlugin())
            config.module.rules.push(getMonacoCSSRule(), getMonacoTTFRule())

            Object.assign(config.entry, {
                'editor.worker': 'monaco-editor/esm/vs/editor/editor.worker.js',
                'json.worker': 'monaco-editor/esm/vs/language/json/json.worker',
            })
        }

        if (environment.isBundleAnalyzerEnabled) {
            config.plugins.push(new BundleAnalyzerPlugin())
        }

        if (environment.isSpeedAnalyzerEnabled) {
            const speedMeasurePlugin = new SpeedMeasurePlugin({
                outputFormat: 'human',
            })

            config.plugins.push(speedMeasurePlugin)

            return speedMeasurePlugin.wrap(config)
        }

        return config
    },
}

module.exports = config
