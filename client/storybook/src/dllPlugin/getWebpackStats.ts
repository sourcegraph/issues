import { spawnSync } from 'child_process'
import fs from 'fs'
import path from 'path'

import signale from 'signale'
import { StatsCompilation } from 'webpack'

import { readJsonFile, storybookWorkspacePath, rootPath } from '../webpack.config.common'

const webpackStatsPath = path.resolve(storybookWorkspacePath, 'storybook-static/preview-stats.json')

export const ensureWebpackStatsAreReady = (): void => {
    signale.start(`Checking if Webpack stats are available: ${path.relative(rootPath, webpackStatsPath)}`)

    // eslint-disable-next-line no-sync
    if (!fs.existsSync(webpackStatsPath)) {
        signale.warn('Webpack stats not found!')
        signale.await('Building Webpack stats with `yarn build:webpack-stats`')

        spawnSync('yarn', ['build:webpack-stats'], {
            stdio: 'inherit',
            cwd: storybookWorkspacePath,
        })
    }

    signale.success('Webpack stats are ready!')
}

// Read Webpack stats JSON file. If it's not available use `yarn build:webpack-stats` command to create it.
export function getWebpackStats(): StatsCompilation {
    ensureWebpackStatsAreReady()

    return readJsonFile(webpackStatsPath) as StatsCompilation
}
