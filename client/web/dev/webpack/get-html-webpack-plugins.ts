import path from 'path'

import HtmlWebpackHarddiskPlugin from 'html-webpack-harddisk-plugin'
import HtmlWebpackPlugin, { TemplateParameter, Options } from 'html-webpack-plugin'
import { WebpackPluginInstance } from 'webpack'

import { createJsContext, environmentConfig, STATIC_ASSETS_PATH } from '../utils'

const { SOURCEGRAPH_HTTPS_PORT, NODE_ENV } = environmentConfig

export const getHTMLWebpackPlugins = (): WebpackPluginInstance[] => {
    const jsContext = createJsContext({ sourcegraphBaseUrl: `http://localhost:${SOURCEGRAPH_HTTPS_PORT}` })

    /**
     * To match `cmd/frontend/internal/app/ui/app.html` template used by our `frontend` server
     * script tags are injected after the <body> tag.
     */
    const templateContent = ({ htmlWebpackPlugin }: TemplateParameter): string => `
        <!DOCTYPE html>
        <html lang="en">
            <head>
                <title>Sourcegraph</title>
                ${htmlWebpackPlugin.tags.headTags.filter(tag => tag.tagName !== 'script').toString()}
            </head>
            <body>
                <div id="root"></div>
                <script>
                    // Optional value useful for checking if index.html is created by HtmlWebpackPlugin with the right NODE_ENV.
                    window.webpackBuildEnvironment = '${NODE_ENV}'

                    // Required mock of the JS context object.
                    window.context = ${JSON.stringify(jsContext)}
                </script>
                ${htmlWebpackPlugin.tags.headTags.filter(tag => tag.tagName === 'script').toString()}
            </body>
        </html>
        `

    const htmlWebpackPlugin = new HtmlWebpackPlugin({
        // `TemplateParameter` can be mutated. We need to tell TS that we didn't touch it.
        templateContent: templateContent as Options['templateContent'],
        meta: {
            charset: 'utf-8',
            viewport: 'width=device-width, viewport-fit=cover, initial-scale=1',
            referrer: 'origin-when-cross-origin',
            'color-scheme': 'light dark',
        },
        filename: path.resolve(STATIC_ASSETS_PATH, 'index.html'),
        alwaysWriteToDisk: true,
        inject: false,
    })

    // Write index.html to the disk so it can be served by dev/prod servers.
    return [htmlWebpackPlugin, new HtmlWebpackHarddiskPlugin()]
}
