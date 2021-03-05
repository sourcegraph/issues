import * as comlink from 'comlink'
import { Location, MarkupKind, Position, Range, Selection } from '@sourcegraph/extension-api-classes'
import { Subscription, Unsubscribable } from 'rxjs'
import * as sourcegraph from 'sourcegraph'
import { EndpointPair } from '../../platform/context'
import { ClientAPI } from '../client/api/api'
import { ExtensionHostAPI, ExtensionHostAPIFactory } from './api/api'
import { createDecorationType } from './api/decorations'
import { ExtensionDocuments } from './api/documents'
import { DocumentHighlightKind } from './api/documentHighlights'
import { Extensions } from './api/extensions'
import { ExtensionViewsApi } from './api/views'
import { registerComlinkTransferHandlers } from '../util'
import { initNewExtensionAPI } from './flatExtensionApi'
import { SettingsCascade } from '../../settings/settings'
import { NotificationType } from '../contract'

/**
 * Required information when initializing an extension host.
 */
export interface InitData {
    /** @see {@link module:sourcegraph.internal.sourcegraphURL} */
    sourcegraphURL: string

    /** @see {@link module:sourcegraph.internal.clientApplication} */
    clientApplication: 'sourcegraph' | 'other'

    /** fetched initial settings object */
    initialSettings: Readonly<SettingsCascade<object>>
}
/**
 * Starts the extension host, which runs extensions. It is a Web Worker or other similar isolated
 * JavaScript execution context. There is exactly 1 extension host, and it has zero or more
 * extensions activated (and running).
 *
 * It expects to receive a message containing {@link InitData} from the client application as the
 * first message.
 *
 * @param endpoints The endpoints to the client.
 * @returns An unsubscribable to terminate the extension host.
 */
export function startExtensionHost(
    endpoints: EndpointPair
): Unsubscribable & { extensionAPI: Promise<typeof sourcegraph> } {
    const subscription = new Subscription()

    // Wait for "initialize" message from client application before proceeding to create the
    // extension host.
    let initialized = false
    const extensionAPI = new Promise<typeof sourcegraph>(resolve => {
        const factory: ExtensionHostAPIFactory = initData => {
            if (initialized) {
                throw new Error('extension host is already initialized')
            }
            initialized = true
            const { subscription: extensionHostSubscription, extensionAPI, extensionHostAPI } = initializeExtensionHost(
                endpoints,
                initData
            )
            subscription.add(extensionHostSubscription)
            resolve(extensionAPI)
            return extensionHostAPI
        }
        comlink.expose(factory, endpoints.expose)
    })

    return { unsubscribe: () => subscription.unsubscribe(), extensionAPI }
}

/**
 * Initializes the extension host using the {@link InitData} from the client application. It is
 * called by {@link startExtensionHost} after the {@link InitData} is received.
 *
 * The extension API is made globally available to all requires/imports of the "sourcegraph" module
 * by other scripts running in the same JavaScript context.
 *
 * @param endpoints The endpoints to the client.
 * @param initData The information to initialize this extension host.
 * @returns An unsubscribable to terminate the extension host.
 */
function initializeExtensionHost(
    endpoints: EndpointPair,
    initData: InitData
): { extensionHostAPI: ExtensionHostAPI; extensionAPI: typeof sourcegraph; subscription: Subscription } {
    const subscription = new Subscription()

    const { extensionAPI, extensionHostAPI, subscription: apiSubscription } = createExtensionAPI(initData, endpoints)
    subscription.add(apiSubscription)

    // Make `import 'sourcegraph'` or `require('sourcegraph')` return the extension API.
    globalThis.require = ((modulePath: string): any => {
        if (modulePath === 'sourcegraph') {
            return extensionAPI
        }
        // All other requires/imports in the extension's code should not reach here because their JS
        // bundler should have resolved them locally.
        throw new Error(`require: module not found: ${modulePath}`)
    }) as any
    subscription.add(() => {
        globalThis.require = (() => {
            // Prevent callers from attempting to access the extension API after it was
            // unsubscribed.
            throw new Error('require: Sourcegraph extension API was unsubscribed')
        }) as any
    })

    return { subscription, extensionAPI, extensionHostAPI }
}

function createExtensionAPI(
    initData: InitData,
    endpoints: Pick<EndpointPair, 'proxy'>
): { extensionHostAPI: ExtensionHostAPI; extensionAPI: typeof sourcegraph; subscription: Subscription } {
    const subscription = new Subscription()

    // EXTENSION HOST WORKER

    registerComlinkTransferHandlers()

    /** Proxy to main thread */
    const proxy = comlink.wrap<ClientAPI>(endpoints.proxy)

    // For debugging/tests. TODO(tj): can deprecate
    const sync = async (): Promise<void> => {
        await proxy.ping()
    }
    // const context = new ExtensionContext(proxy.context)

    // TODO(tj): ready to remove
    const documents = new ExtensionDocuments(sync)

    const extensions = new Extensions()
    subscription.add(extensions)
    // TODO(tj): remember to add new extension activation sub to bag

    const views = new ExtensionViewsApi(proxy.views)

    const {
        configuration,
        exposedToMain,
        workspace,
        commands,
        search,
        languages,
        graphQL,
        content,
        app,
        internal,
    } = initNewExtensionAPI(proxy, initData)

    // Expose the extension host API to the client (main thread)
    const extensionHostAPI: ExtensionHostAPI = {
        [comlink.proxyMarker]: true,

        ping: () => 'pong',

        documents,
        extensions,
        ...exposedToMain,
    }

    // Expose the extension API to extensions
    // "redefines" everything instead of exposing internal Ext* classes directly so as to:
    // - Avoid exposing private methods to extensions
    // - Avoid exposing proxy.* to extensions, which gives access to the main thread
    const extensionAPI: typeof sourcegraph & {
        // Backcompat definitions that were removed from sourcegraph.d.ts but are still defined (as
        // noops with a log message), to avoid completely breaking extensions that use them.
        languages: {
            registerTypeDefinitionProvider: any
            registerImplementationProvider: any
        }
    } = {
        URI: URL,
        Position,
        Range,
        Selection,
        Location,
        MarkupKind,
        NotificationType,
        DocumentHighlightKind,
        app: {
            // TODO(tj): deprecate the following 3, implement get window()
            activeWindowChanges: app.activeWindowChanges,
            get activeWindow(): sourcegraph.Window | undefined {
                return app.activeWindow
            },
            get windows(): sourcegraph.Window[] {
                return app.windows
            },
            registerFileDecorationProvider: app.registerFileDecorationProvider,
            createPanelView: app.createPanelView,
            createDecorationType,
            registerViewProvider: (id, provider) => views.registerViewProvider(id, provider),
        },

        workspace,
        configuration,

        languages,

        search,
        commands,
        graphQL,
        content,

        internal: {
            sync: () => {
                console.warn('Sourcegraph extensions no longer need to call sync to ensure intended behavior') // TODO(tj): better error
                return sync()
            },
            updateContext: (updates: sourcegraph.ContextValues) => internal.updateContext(updates),
            sourcegraphURL: new URL(initData.sourcegraphURL),
            clientApplication: initData.clientApplication,
        },
    }
    return { extensionHostAPI, extensionAPI, subscription }
}
