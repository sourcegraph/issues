// For comlink
// TODO comlink v4 seems to not need this
globalThis.self = globalThis as any
import 'message-port-polyfill'

import puppeteer, { Browser, ConsoleMessageType, Page } from 'puppeteer'
import { PlatformContext } from '../../shared/src/platform/context'
import { createExtensionHostClientConnection, ExtensionHostClientConnection } from '../../shared/src/api/client/connection'
import { Services } from '../../shared/src/api/client/services'
import { of, from, fromEvent, BehaviorSubject, zip, Observable } from 'rxjs'
import got from 'got'
import { Settings, SettingsCascadeOrError } from '../../shared/src/settings/settings'
import { take, concatMap, concatAll, filter, map, tap } from 'rxjs/operators'
import express from 'express'
import WebSocket, * as ws from 'ws'
import { IncomingMessage } from 'http'
import * as MessageChannelAdapter from '@sourcegraph/comlink/messagechanneladapter'
import html from 'tagged-template-noop'
import { Signale, DefaultMethods } from 'signale'
import { inspect } from 'util'

inspect.defaultOptions.colors = true

const logger = new Signale({ scope: 'Extension Runner' })

async function main(): Promise<void> {
    const app = express()
    app.get('/index.html', (req, res) =>
        res.send(html`
            <html>
                <head>
                    <script>
                        // Load extension host in web worker
                        new Worker('/extensionHost.bundle.js')
                    </script>
                </head>
            </html>
        `)
    )
    // Serve extension host bundle for the web worker
    app.use(express.static(__dirname))
    const httpServer = app.listen(11000)

    const wsServer = new ws.Server({ port: 11001 })

    const browser = await puppeteer.launch({ headless: true })

    try {
        const ext = await runExtension('danielevan/sourcegraph-hey-everyone', browser, wsServer)

        const uri = 'git://github.com/sourcegraph/sourcegraph#test.ts'

        logger.await('Asking for hover')

        const hover = await ext.services.textDocumentHover
            .getHover({ textDocument: { uri }, position: { line: 0, character: 0 } })
            .pipe(take(1))
            .toPromise()

        logger.success('Result:', hover)

    } finally {
        await browser.close()
        wsServer.close()
        httpServer.close()
    }
}

interface ExtensionEnvironment {
    page: Page
    connection: ExtensionHostClientConnection
    services: Services
}

async function runExtension(extensionID: string, browser: Browser, webSocketServer: ws.Server): Promise<ExtensionEnvironment> {
    logger.await('Opening page')
    const page = await browser.newPage()

    // Forward logs
    const extHostLogger = new Signale({ scope: 'Extension Host  ' })
    page.on('console', message => {
        const mapping: Partial<Record<ConsoleMessageType, DefaultMethods>> = {
            log: 'log',
            info: 'info',
            warning: 'warn',
            error: 'error',
        }
        const method = mapping[message.type()] || 'info'
        extHostLogger[method](message.text())
    })

    const connections = fromEvent<[WebSocket, IncomingMessage]>(webSocketServer, 'connection')
    const connectionWithPath = (path: string): Observable<WebSocket> =>
        connections.pipe(
            // TODO pass the page a UUID to identify itself, check for that too
            // Right now this would break with multiple environments running at the same time
            filter(([, request]) => request.url === path),
            map(([connection]) => connection),
            tap(connection => connection.setMaxListeners(1000)),
            take(1)
        )
    const connectionPromise = zip(connectionWithPath('/proxy'), connectionWithPath('/expose')).toPromise()

    await page.goto('http://localhost:11000/index.html')

    const [proxyConn, exposeConn] = await connectionPromise
    logger.success('Received proxy and expose connections')

    const clientApplication = 'other'
    const sourcegraphURL = 'http://localhost:3080'

    const requestGraphQL: PlatformContext['requestGraphQL'] = ({ request, variables }) =>
        from(
            got
                .post('http://localhost:3080/.api/graphql', {
                    headers: {
                        Authorization: 'token 6aa9ec9365511c717689b379facda1859c8ede21',
                    },
                    json: true,
                    body: { query: request, variables },
                })
                .then(res => res.body)
        )

    const settings: Settings = {
        extensions: {
            [extensionID]: true,
        },
    }

    const services = new Services({
        clientApplication,
        getScriptURLForExtension: bundleURL => bundleURL,
        settings: of<SettingsCascadeOrError<Settings>>({
            subjects: [
                {
                    subject: {
                        __typename: 'Client',
                        id: 'clientSettings',
                        displayName: 'Extension Runner Client Settings',
                        viewerCanAdminister: false,
                    },
                    lastID: 1,
                    settings,
                },
            ],
            final: settings,
        }),
        sideloadedExtensionURL: new BehaviorSubject<string | null>(null),
        requestGraphQL,
        updateSettings: () => Promise.reject(new Error('Cannot update settings in the backend extension runner')),
    })
    const endpoints = {
        proxy: MessageChannelAdapter.wrap(proxyConn as MessageChannelAdapter.StringMessageChannel),
        expose: MessageChannelAdapter.wrap(exposeConn as MessageChannelAdapter.StringMessageChannel),
    }
    const connection = await createExtensionHostClientConnection(endpoints, services, {
        clientApplication,
        sourcegraphURL,
    })
    logger.await('Sending ping to extension host')
    logger.success('Response:', await connection.proxy.ping())

    // TODO make configurable
    const uri = 'git://github.com/sourcegraph/sourcegraph#test.ts'
    services.model.addModel({
        languageId: 'typescript',
        text: 'Hello',
        uri,
    })
    services.editor.addEditor({
        resource: uri,
        isActive: true,
        selections: [],
        type: 'CodeEditor',
    })

    logger.await('Activating extension', extensionID)
    await from(services.extensions.activeExtensions)
        .pipe(
            concatAll(),
            take(1),
            concatMap(extension => connection.proxy.extensions.$activateExtension(extension.id, extension.scriptURL))
        )
        .toPromise()
    logger.success('Extension activated')

    return {
        services,
        page,
        connection,
    }
}

main().catch(err => {
    logger.error(err)
    process.exit(1)
})
