import path from 'path'

import CssMinimizerWebpackPlugin from 'css-minimizer-webpack-plugin'
import MiniCssExtractPlugin from 'mini-css-extract-plugin'
import TerserPlugin from 'terser-webpack-plugin'
import webpack, { optimize } from 'webpack'

import { subtypeOf } from '../../../shared/src/util/types'

export const rootPath = path.resolve(__dirname, '../../../../')
export const browserWorkspacePath = path.resolve(rootPath, 'client/browser')
const browserSourcePath = path.resolve(browserWorkspacePath, 'src')

const contentEntry = path.resolve(browserSourcePath, 'config/content.entry.js')
const backgroundEntry = path.resolve(browserSourcePath, 'config/background.entry.js')
const optionsEntry = path.resolve(browserSourcePath, 'config/options.entry.js')
const pageEntry = path.resolve(browserSourcePath, 'config/page.entry.js')
const extensionEntry = path.resolve(browserSourcePath, 'config/extension.entry.js')

const babelLoader = {
    loader: 'babel-loader',
    options: {
        cacheDirectory: true,
        configFile: path.join(__dirname, '..', '..', 'babel.config.js'),
    },
}

const extensionHostWorker = /main\.worker\.ts$/

const getCSSLoaders = (...loaders: webpack.RuleSetUseItem[]): webpack.RuleSetUse => [
    MiniCssExtractPlugin.loader,
    ...loaders,
    {
        loader: 'postcss-loader',
    },
    {
        loader: 'sass-loader',
        options: {
            sassOptions: {
                includePaths: [path.resolve(rootPath, 'node_modules'), path.resolve(rootPath, 'client')],
            },
        },
    },
]

export const config = subtypeOf<webpack.Configuration>()({
    target: 'browserslist',
    entry: {
        // Browser extension
        background: [
            extensionEntry,
            backgroundEntry,
            path.resolve(browserSourcePath, 'browser-extension/scripts/backgroundPage.main.ts'),
        ],
        inject: [
            extensionEntry,
            contentEntry,
            path.resolve(browserSourcePath, 'browser-extension/scripts/contentPage.main.ts'),
        ],
        options: [
            extensionEntry,
            optionsEntry,
            path.resolve(browserSourcePath, 'browser-extension/scripts/optionsPage.main.tsx'),
        ],
        'after-install': path.resolve(browserSourcePath, 'browser-extension/scripts/afterInstallPage.main.tsx'),

        // Common native integration entry point (Gitlab, Bitbucket)
        integration: [pageEntry, path.resolve(browserSourcePath, 'native-integration/integration.main.ts')],
        // Phabricator-only native integration entry point
        phabricator: [pageEntry, path.resolve(browserSourcePath, 'native-integration/phabricator/integration.main.ts')],

        // Styles
        style: path.join(browserSourcePath, 'app.scss'),
        'branded-style': path.join(browserSourcePath, 'branded.scss'),
    },
    output: {
        path: path.join(browserWorkspacePath, 'build/dist/js'),
        filename: '[name].bundle.js',
        chunkFilename: '[id].chunk.js',
    },
    devtool: 'inline-cheap-module-source-map',
    optimization: {
        minimizer: [
            new TerserPlugin({
                terserOptions: {
                    compress: {
                        // Don't inline functions, which causes name collisions with uglify-es:
                        // https://github.com/mishoo/UglifyJS2/issues/2842
                        inline: 1,
                    },
                },
            }) as webpack.WebpackPluginInstance,
            new CssMinimizerWebpackPlugin(),
        ],
    },

    plugins: [
        new MiniCssExtractPlugin({ filename: '../css/[name].bundle.css' }),
        // Code splitting doesn't make sense/work in the browser extension, but we still want to use dynamic import()
        new optimize.LimitChunkCountPlugin({ maxChunks: 1 }),
    ],
    resolve: {
        extensions: ['.ts', '.tsx', '.js'],
    },
    module: {
        rules: [
            {
                test: /\.[jt]sx?$/,
                exclude: extensionHostWorker,
                use: [babelLoader],
            },
            {
                // SCSS rule for our own styles and Bootstrap
                test: /\.(css|sass|scss)$/,
                exclude: /\.module\.(sass|scss)$/,
                use: getCSSLoaders({ loader: 'css-loader', options: { url: false } }),
            },
            {
                test: /\.(css|sass|scss)$/,
                include: /\.module\.(sass|scss)$/,
                use: getCSSLoaders({
                    loader: 'css-loader',
                    options: {
                        sourceMap: false,
                        modules: {
                            exportLocalsConvention: 'camelCase',
                            localIdentName: '[name]__[local]_[hash:base64:5]',
                        },
                        url: false,
                    },
                }),
            },
            {
                test: extensionHostWorker,
                use: [
                    {
                        loader: 'worker-loader',
                        options: { filename: 'extensionHostWorker.bundle.js' },
                    },
                    babelLoader,
                ],
            },
        ],
    },
})
