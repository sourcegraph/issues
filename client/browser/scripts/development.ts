import { noop } from 'lodash'
import signale from 'signale'
import webpack from 'webpack'
import config from '../config/webpack/development.config'
import * as autoReloading from './auto-reloading'
import * as tasks from './tasks'

signale.config({ displayTimestamp: true })

const triggerReload = process.env.AUTO_RELOAD === 'false' ? noop : autoReloading.initializeServer()

const buildChrome = tasks.buildChrome('dev')
const buildFirefox = tasks.buildFirefox('dev')

tasks.copyAssets()

const compiler = webpack(config)

signale.info('Running webpack')

compiler.hooks.watchRun.tap('Notify', () => signale.await('Compiling...'))

compiler.watch(
    {
        aggregateTimeout: 300,
    },
    (error, stats) => {
        signale.complete(stats.toString(tasks.WEBPACK_STATS_OPTIONS))

        if (error || stats.hasErrors()) {
            signale.error('Webpack compilation error')
            return
        }
        signale.success('Webpack compilation done')

        buildChrome()
        buildFirefox()
        tasks.copyIntegrationAssets()
        triggerReload()
    }
)
