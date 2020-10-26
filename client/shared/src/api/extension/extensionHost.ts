import * as comlink from 'comlink'
import { Location, MarkupKind, Position, Range, Selection } from '@sourcegraph/extension-api-classes'
import { Subscription, Unsubscribable, of } from 'rxjs'
import * as sourcegraph from 'sourcegraph'
import { EndpointPair } from '../../platform/context'
import { ClientAPI } from '../client/api/api'
import { NotificationType } from '../client/services/notifications'
import { ExtensionHostAPI, ExtensionHostAPIFactory } from './api/api'
import { ExtensionContent } from './api/content'
import { ExtensionContext } from './api/context'
import { createDecorationType } from './api/decorations'
import { DocumentHighlightKind } from './api/documentHighlights'
import { ExtensionViewsApi } from './api/views'
import { ExtensionWindows } from './api/windows'

import { registerComlinkTransferHandlers } from '../util'
import { initNewExtensionAPI } from './flatExtensionApi'
import { SettingsCascade } from '../../settings/settings'
import { ExtensionDocuments } from './api/documents'

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

    // For debugging/tests.
    const sync = async (): Promise<void> => {
        await proxy.ping()
    }
    const context = new ExtensionContext(proxy.context)
    const documents = new ExtensionDocuments(sync)

    const windows = new ExtensionWindows(proxy, documents)
    const views = new ExtensionViewsApi(proxy.views)
    const content = new ExtensionContent(proxy.content)

    const { configuration, exposedToMain, workspace, commands, search, languages } = initNewExtensionAPI(
        proxy,
        initData.initialSettings
    )

    // Expose the extension host API to the client (main thread)
    const extensionHostAPI: ExtensionHostAPI = {
        [comlink.proxyMarker]: true,

        ping: () => 'pong',

        windows,
        ...exposedToMain,
    }

    // Expose the extension API to extensions
    // "redefines" everything instead of exposing internal Ext* classes directly so as to:
    // - Avoid exposing private methods to extensions
    // - Avoid exposing proxy.* to extensions, which gives access to the main thread
    const extensionAPI: typeof sourcegraph = {
        URI: URL,
        Position,
        Range,
        Selection,
        Location,
        MarkupKind,
        NotificationType,
        DocumentHighlightKind,
        app: {
            activeWindowChanges: windows.activeWindowChanges,
            get activeWindow(): sourcegraph.Window | undefined {
                return windows.activeWindow
            },
            get windows(): sourcegraph.Window[] {
                return windows.getAll()
            },
            createPanelView: (id: string) => views.createPanelView(id),
            createDecorationType,
            registerViewProvider: (id, provider) => views.registerViewProvider(id, provider),
        },

        workspace,

        configuration,

        languages,

        search,
        commands,
        content: {
            registerLinkPreviewProvider: (urlMatchPattern: string, provider: sourcegraph.LinkPreviewProvider) =>
                content.registerLinkPreviewProvider(urlMatchPattern, provider),
        },

        internal: {
            sync,
            updateContext: (updates: sourcegraph.ContextValues) => context.updateContext(updates),
            sourcegraphURL: new URL(initData.sourcegraphURL),
            clientApplication: initData.clientApplication,
        },
    }

    subscription.add(handleExtensionActivation(proxy, of(new Set())))

    return { extensionHostAPI, extensionAPI, subscription }
}
