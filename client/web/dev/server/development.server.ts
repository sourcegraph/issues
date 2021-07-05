import chalk from 'chalk'
import { Application } from 'express'
import signale from 'signale'
import createWebpackCompiler, { Configuration } from 'webpack'
import WebpackDevServer, { ProxyConfigArrayItem } from 'webpack-dev-server'

import {
    getCSRFTokenCookieMiddleware,
    PROXY_ROUTES,
    environmentConfig,
    getAPIProxySettings,
    getCSRFTokenAndCookie,
    STATIC_ASSETS_PATH,
    STATIC_ASSETS_URL,
    WEB_SERVER_URL,
} from '../utils'

// TODO: migrate webpack.config.js to TS to use `import` in this file.
// eslint-disable-next-line @typescript-eslint/no-var-requires, @typescript-eslint/no-require-imports
const webpackConfig = require('../../webpack.config') as Configuration
const { SOURCEGRAPH_API_URL, SOURCEGRAPH_HTTPS_PORT, IS_HOT_RELOAD_ENABLED } = environmentConfig

export async function startDevelopmentServer(): Promise<void> {
    // Get CSRF token value from the `SOURCEGRAPH_API_URL`.
    const { csrfContextValue, csrfCookieValue } = await getCSRFTokenAndCookie(SOURCEGRAPH_API_URL)
    signale.await('Development server', { ...environmentConfig, csrfContextValue, csrfCookieValue })

    const proxyConfig = {
        context: PROXY_ROUTES,
        ...getAPIProxySettings({
            csrfContextValue,
            apiURL: SOURCEGRAPH_API_URL,
        }),
    }

    // It's not possible to use `WebpackDevServer.Configuration` here yet, because
    // type definitions for the `webpack-dev-server` are not updated to match v4.
    const developmentServerConfig = {
        hot: IS_HOT_RELOAD_ENABLED,
        // TODO: resolve https://github.com/webpack/webpack-dev-server/issues/2313 and enable HTTPS.
        https: false,
        historyApiFallback: {
            disableDotRule: true,
        },
        port: SOURCEGRAPH_HTTPS_PORT,
        client: {
            overlay: {
                errors: true,
                warnings: false,
            },
        },
        static: {
            directory: STATIC_ASSETS_PATH,
            publicPath: [STATIC_ASSETS_URL, '/'],
        },
        firewall: false,
        proxy: [proxyConfig as ProxyConfigArrayItem],
        onBeforeSetupMiddleware(app: Application) {
            app.use(getCSRFTokenCookieMiddleware(csrfCookieValue))
        },
    }

    const server = new WebpackDevServer(
        createWebpackCompiler(webpackConfig),
        developmentServerConfig as WebpackDevServer.Configuration
    )

    server.listen(SOURCEGRAPH_HTTPS_PORT, '0.0.0.0', () => {
        signale.success(`Development server is ready at ${chalk.blue.bold(WEB_SERVER_URL)}`)
    })
}

startDevelopmentServer().catch(error => signale.error(error))
