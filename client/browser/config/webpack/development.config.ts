import * as path from 'path'

import * as webpack from 'webpack'

import { config as baseConfig, browserWorkspacePath, rootPath } from './base.config'
import { generateBundleUID } from './utils'

const { plugins, entry: entries, ...base } = baseConfig

const entriesWithAutoReload = {
    ...entries,
    background: [path.join(__dirname, '../../src/browser-extension/scripts/auto-reloading.ts'), ...entries.background],
}

export const config: webpack.Configuration = {
    ...base,
    entry: process.env.AUTO_RELOAD === 'false' ? entries : entriesWithAutoReload,
    mode: 'development',
    // Use cache only in `development` mode to speed up production build.
    cache: {
        type: 'filesystem',
        buildDependencies: {
            // Invalidate cache on config change.
            config: [
                __filename,
                path.resolve(browserWorkspacePath, 'babel.config.js'),
                path.resolve(rootPath, 'babel.config.js'),
                path.resolve(rootPath, 'postcss.config.js'),
            ],
        },
    },
    plugins: (plugins || []).concat(
        ...[
            new webpack.DefinePlugin({
                'process.env': {
                    NODE_ENV: JSON.stringify('development'),
                    BUNDLE_UID: JSON.stringify(generateBundleUID()),
                    USE_EXTENSIONS: JSON.stringify(process.env.USE_EXTENSIONS),
                },
            }),
        ]
    ),
}
