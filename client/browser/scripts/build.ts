import signale from 'signale'
import webpack from 'webpack'
import config from '../config/webpack/production.config'
import * as tasks from './tasks'

const buildChrome = tasks.buildChrome('prod')
const buildFirefox = tasks.buildFirefox('prod')

tasks.copyAssets()

const compiler = webpack(config)

signale.await('Webpack compilation')

compiler.run((error, stats) => {
    console.log(stats.toString(tasks.WEBPACK_STATS_OPTIONS))

    if (stats.hasErrors()) {
        signale.error('Webpack compilation error')
        process.exit(1)
    }
    signale.success('Webpack compilation done')

    buildChrome()
    buildFirefox()
    tasks.copyIntegrationAssets()
    signale.success('Build done')
})
