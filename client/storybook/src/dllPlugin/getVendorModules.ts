import { StatsCompilation } from 'webpack'

import { nodeModulesPath } from '../webpack.config.common'

// Do not include Webpack related modules into a DLL bundle. It breaks development build.
const SKIP_VENDOR_MODULES = ['webpack']

// Get a list of files to include into a DLL bundle based on Webpack stats provided.
export function getVendorModules(webpackStats: StatsCompilation): Set<string> {
    const vendorsChunk = webpackStats.chunks?.find(
        chunk => typeof chunk.id === 'string' && chunk.id.includes('vendors')
    )

    if (!vendorsChunk || !vendorsChunk.modules) {
        throw new Error('Vendors chunk not found in preview.stats.json!')
    }

    const vendorModules = vendorsChunk.modules
        .map(module => {
            if (!module.identifier) {
                return ''
            }

            // `identifier` contains loaders prefix, so `path.relative()` doesn't work for all cases.
            const [relativePathToModule] = module.identifier.split(`${nodeModulesPath}/`).slice(-1)

            // Remove suffix generated for some Storybook modules.
            return relativePathToModule.replace('-generated-other-entry.js', '')
        })
        .filter(modulePath => !SKIP_VENDOR_MODULES.some(ignoreModule => modulePath.includes(ignoreModule)))

    return new Set(vendorModules)
}
