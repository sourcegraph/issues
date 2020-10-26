import * as path from 'path'
import * as webpack from 'webpack'
import baseConfig from './base.config'
import { generateBundleUID } from './utils'

const { plugins, entry, ...base } = baseConfig

const entries = entry as webpack.Entry

const entriesWithAutoReload = {
    ...entries,
    background: [path.join(__dirname, '../../src/browser-extension/scripts/auto-reloading.ts'), ...entries.background],
}

const config: webpack.Configuration = {
    ...base,
    entry: process.env.AUTO_RELOAD === 'false' ? entries : entriesWithAutoReload,
    mode: 'development',
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
export default config
