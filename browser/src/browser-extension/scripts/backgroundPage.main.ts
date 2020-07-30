// We want to polyfill first.
import '../../shared/polyfills'

import { Endpoint } from 'comlink'
import { without } from 'lodash'
import { noop, Observable, Subscription } from 'rxjs'
import { bufferCount, filter, groupBy, map, mergeMap, switchMap, take, concatMap } from 'rxjs/operators'
import addDomainPermissionToggle from 'webext-domain-permission-toggle'
import { createExtensionHostWorker } from '../../../../shared/src/api/extension/worker'
import { GraphQLResult, requestGraphQL as requestGraphQLCommon } from '../../../../shared/src/graphql/graphql'
import { BackgroundMessageHandlers } from '../web-extension-api/types'
import { initializeOmniboxInterface } from '../../shared/cli'
import { initSentry } from '../../shared/sentry'
import { createBlobURLForBundle } from '../../shared/platform/worker'
import { getHeaders } from '../../shared/backend/headers'
import { fromBrowserEvent } from '../web-extension-api/fromBrowserEvent'
import { observeSourcegraphURL } from '../../shared/util/context'
import { assertEnvironment } from '../environmentAssertion'
import { observeStorageKey, storage } from '../web-extension-api/storage'
import { isDefined } from '../../../../shared/src/util/types'
import { browserPortToMessagePort, findMessagePorts } from '../../shared/platform/ports'
import { EndpointPair } from '../../../../shared/src/platform/context'

const IS_EXTENSION = true

assertEnvironment('BACKGROUND')

initSentry('background')

let customServerOrigins: string[] = []

const contentScripts = browser.runtime.getManifest().content_scripts

// jsContentScriptOrigins are the required URLs inside of the manifest. When checking for permissions to inject
// the content script on optional pages (inside browser.tabs.onUpdated) we need to skip manual injection of the
// script since the browser extension will automatically inject it.
const jsContentScriptOrigins: string[] = []
if (contentScripts) {
    for (const contentScript of contentScripts) {
        if (!contentScript || !contentScript.js || !contentScript.matches) {
            continue
        }
        jsContentScriptOrigins.push(...contentScript.matches)
    }
}

const configureOmnibox = (serverUrl: string): void => {
    browser.omnibox.setDefaultSuggestion({
        description: `Search code on ${serverUrl}`,
    })
}

const requestGraphQL = <T>({
    request,
    variables,
}: {
    request: string
    variables: {}
}): Observable<GraphQLResult<T>> =>
    observeSourcegraphURL(IS_EXTENSION).pipe(
        take(1),
        switchMap(sourcegraphURL =>
            requestGraphQLCommon<T>({
                request,
                variables,
                baseUrl: sourcegraphURL,
                headers: getHeaders(),
                credentials: 'include',
            })
        )
    )

initializeOmniboxInterface(requestGraphQL)

async function main(): Promise<void> {
    const subscriptions = new Subscription()

    // Mirror the managed sourcegraphURL to sync storage
    subscriptions.add(
        observeStorageKey('managed', 'sourcegraphURL')
            .pipe(
                filter(isDefined),
                concatMap(sourcegraphURL => storage.sync.set({ sourcegraphURL }))
            )
            .subscribe()
    )
    // Configure the omnibox when the sourcegraphURL changes.
    subscriptions.add(
        observeSourcegraphURL(IS_EXTENSION).subscribe(sourcegraphURL => {
            configureOmnibox(sourcegraphURL)
        })
    )

    const permissions = await browser.permissions.getAll()
    if (!permissions.origins) {
        customServerOrigins = []
        return
    }
    customServerOrigins = without(permissions.origins, ...jsContentScriptOrigins)

    // Not supported in Firefox
    // https://developer.mozilla.org/en-US/docs/Mozilla/Add-ons/WebExtensions/API/permissions/onAdded#Browser_compatibility
    if (browser.permissions.onAdded) {
        browser.permissions.onAdded.addListener(permissions => {
            if (!permissions.origins) {
                return
            }
            const origins = without(permissions.origins, ...jsContentScriptOrigins)
            customServerOrigins.push(...origins)
        })
    }
    if (browser.permissions.onRemoved) {
        browser.permissions.onRemoved.addListener(permissions => {
            if (!permissions.origins) {
                return
            }
            customServerOrigins = without(customServerOrigins, ...permissions.origins)
            const urlsToRemove: string[] = []
            for (const url of permissions.origins) {
                urlsToRemove.push(url.replace('/*', ''))
            }
        })
    }

    // Inject content script whenever a new tab was opened with a URL that we have permissions for
    browser.tabs.onUpdated.addListener(async (tabId, changeInfo, tab) => {
        if (
            changeInfo.status === 'complete' &&
            customServerOrigins.some(
                origin => origin === '<all_urls>' || (!!tab.url && tab.url.startsWith(origin.replace('/*', '')))
            )
        ) {
            await browser.tabs.executeScript(tabId, { file: 'js/inject.bundle.js', runAt: 'document_end' })
        }
    })

    const handlers: BackgroundMessageHandlers = {
        async openOptionsPage(): Promise<void> {
            await browser.runtime.openOptionsPage()
        },

        async createBlobURL(bundleUrl: string): Promise<string> {
            return createBlobURLForBundle(bundleUrl)
        },

        async requestGraphQL<T>({
            request,
            variables,
        }: {
            request: string
            variables: {}
        }): Promise<GraphQLResult<T>> {
            return requestGraphQL<T>({ request, variables }).toPromise()
        },
    }

    // Handle calls from other scripts
    browser.runtime.onMessage.addListener(async (message: { type: keyof BackgroundMessageHandlers; payload: any }) => {
        const method = message.type
        if (!handlers[method]) {
            throw new Error(`Invalid RPC call for "${method}"`)
        }
        return handlers[method](message.payload)
    })

    await browser.runtime.setUninstallURL('https://about.sourcegraph.com/uninstall/')

    browser.browserAction.onClicked.addListener(noop)
    browser.browserAction.setBadgeText({ text: '' })
    browser.browserAction.setPopup({ popup: 'options.html?popup=true' })

    // Add "Enable Sourcegraph on this domain" context menu item
    addDomainPermissionToggle()

    const ENDPOINT_KIND_REGEX = /^(proxy|expose)-/

    const portKind = (port: browser.runtime.Port): string | undefined => {
        const match = port.name.match(ENDPOINT_KIND_REGEX)
        return match?.[1]
    }

    /**
     * A stream of EndpointPair created from Port objects emitted by browser.runtime.onConnect.
     *
     * On initialization, the content script creates a pair of browser.runtime.Port objects
     * using browser.runtime.connect(). The two ports are named 'proxy-{uuid}' and 'expose-{uuid}',
     * and wrapped using {@link endpointFromPort} to behave like comlink endpoints on the content script side.
     *
     * This listens to events on browser.runtime.onConnect, pairs emitted ports using their naming pattern,
     * and emits pairs. Each pair of ports represents a connection with an instance of the content script.
     */
    const browserPortPairs: Observable<Record<keyof EndpointPair, browser.runtime.PortWithSender>> = fromBrowserEvent(
        browser.runtime.onConnect
    ).pipe(
        map(([port]) => port),
        groupBy(
            port => (port.name || 'other').replace(ENDPOINT_KIND_REGEX, ''),
            port => port,
            group => group.pipe(bufferCount(2))
        ),
        filter(group => group.key !== 'other'),
        mergeMap(group =>
            group.pipe(
                bufferCount(2),
                map(ports => {
                    const proxyPort = ports.find(port => portKind(port) === 'proxy')
                    if (!proxyPort) {
                        throw new Error('No proxy port')
                    }
                    const exposePort = ports.find(port => portKind(port) === 'expose')
                    if (!exposePort) {
                        throw new Error('No expose port')
                    }
                    return {
                        proxy: proxyPort,
                        expose: exposePort,
                    }
                })
            )
        )
    )

    // Extension Host Connection
    // When an Port pair is emitted, create an extension host worker.
    // Messages from the ports are forwarded to the endpoints returned by {@link createExtensionHostWorker}, and vice-versa.
    // The lifetime of the extension host worker is tied to that of the content script instance:
    // when a port disconnects, the worker is terminated. This means there should always be exactly one
    // extension host worker per active instance of the content script.
    subscriptions.add(
        browserPortPairs.subscribe({
            next: browserPortPair => {
                subscriptions.add(handleBrowserPortPair(browserPortPair))
            },
            error: error => {
                console.error('Error handling extension host client connection', error)
            },
        })
    )

    console.log('Sourcegraph background page initialized')
}

const workerBundleURL = browser.runtime.getURL('js/extensionHostWorker.bundle.js')

/**
 * Handle an incoming browser port pair coming from a content script.
 */
function handleBrowserPortPair(
    browserPortPair: Record<keyof EndpointPair, browser.runtime.PortWithSender>
): Subscription {
    /** Subscriptions for this browser port pair */
    const subscriptions = new Subscription()

    console.log('Extension host client connected')
    const { worker, clientEndpoints } = createExtensionHostWorker(workerBundleURL)
    subscriptions.add(() => worker.terminate())

    /** Forwards all messages between two endpoints (in one direction) */
    const forwardEndpoint = (from: Endpoint, to: Endpoint): void => {
        const messageListener = (event: Event): void => {
            const { data } = event as MessageEvent
            to.postMessage(data, [...findMessagePorts(data)])
        }
        from.addEventListener('message', messageListener)
        subscriptions.add(() => from.removeEventListener('message', messageListener))

        // False positive https://github.com/eslint/eslint/issues/12822
        // eslint-disable-next-line no-unused-expressions
        from.start?.()
    }

    const linkPortAndEndpoint = (role: keyof EndpointPair): void => {
        const browserPort = browserPortPair[role]
        const endpoint = clientEndpoints[role]
        const tabId = browserPort.sender.tab?.id
        if (!tabId) {
            throw new Error('Expected Port to come from tab')
        }
        const link = browserPortToMessagePort(browserPort, `comlink-${role}-`, name =>
            browser.tabs.connect(tabId, { name })
        )
        subscriptions.add(link.subscription)

        forwardEndpoint(link.messagePort, endpoint)
        forwardEndpoint(endpoint, link.messagePort)

        // Clean up when the port disconnects
        const disconnectListener = subscriptions.unsubscribe.bind(subscriptions)
        browserPort.onDisconnect.addListener(disconnectListener)
        subscriptions.add(() => browserPort.onDisconnect.removeListener(disconnectListener))
    }

    // Connect proxy client endpoint
    linkPortAndEndpoint('proxy')
    // Connect expose client endpoint
    linkPortAndEndpoint('expose')

    return subscriptions
}

// Browsers log this unhandled Promise automatically (and with a better stack trace through console.error)
// eslint-disable-next-line @typescript-eslint/no-floating-promises
main()
