import signale from 'signale'
import { DllPlugin, Configuration } from 'webpack'
import { WebpackManifestPlugin } from 'webpack-manifest-plugin'

import { getWebpackStats, getVendorModules } from './dllPlugin'
import {
    monacoEditorPath,
    dllPluginConfig,
    getMonacoCSSRule,
    getMonacoTTFRule,
    getMonacoWebpackPlugin,
    getBasicCSSLoader,
    dllBundleManifestPath,
} from './webpack.config.common'

const webpackStats = getWebpackStats()
signale.await('Waiting for Webpack to build DLL bundle based on vendor stats.')

const config: Configuration = {
    mode: 'development',
    stats: 'errors-warnings',
    entry: {
        dll: [...getVendorModules(webpackStats), 'monaco-editor'],
    },
    output: {
        filename: '[name].bundle.[contenthash].js',
        path: dllPluginConfig.context,
        library: dllPluginConfig.name,
        // Required to fix the `WebpackManifestPlugin` output: https://github.com/shellscape/webpack-manifest-plugin/issues/229
        publicPath: '',
    },
    module: {
        rules: [
            getMonacoCSSRule(),
            getMonacoTTFRule(),
            {
                test: /\.css$/,
                exclude: [monacoEditorPath],
                use: [getBasicCSSLoader()],
            },
        ],
    },
    plugins: [
        getMonacoWebpackPlugin(),
        new DllPlugin(dllPluginConfig),
        new WebpackManifestPlugin({ fileName: dllBundleManifestPath }),
    ],
}

module.exports = config
