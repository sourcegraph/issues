import { ProxyMarked, transferHandlers, releaseProxy, TransferHandler, Remote, proxy } from 'comlink'
import { Subscription, of, EMPTY } from 'rxjs'
import { Subscribable, Unsubscribable } from 'sourcegraph'
import { hasProperty, keyExistsIn } from '../util/types'
import { FlatExtHostAPI, MainThreadAPI } from './contract'
import { noop, identity } from 'lodash'
import { proxySubscribable } from './extension/api/common'

/**
 * Tests whether a value is a WHATWG URL object.
 */
export const isURL = (value: unknown): value is URL =>
    typeof value === 'object' &&
    value !== null &&
    hasProperty('href')(value) &&
    hasProperty('toString')(value) &&
    typeof value.toString === 'function' &&
    // eslint-disable-next-line @typescript-eslint/no-base-to-string
    value.href === value.toString()

/**
 * Registers global comlink transfer handlers.
 * This needs to be called before using comlink.
 * Idempotent.
 */
export function registerComlinkTransferHandlers(): void {
    const urlTransferHandler: TransferHandler<URL, string> = {
        canHandle: isURL,
        serialize: url => [url.href, []],
        deserialize: urlString => new URL(urlString),
    }
    transferHandlers.set('URL', urlTransferHandler)
}

/**
 * Creates a synchronous Subscription that will unsubscribe the given proxied Subscription asynchronously.
 *
 * @param subscriptionPromise A Promise for a Subscription proxied from the other thread
 */
export const syncSubscription = (subscriptionPromise: Promise<Remote<Unsubscribable & ProxyMarked>>): Subscription =>
    // We cannot pass the proxy subscription directly to Rx because it is a Proxy that looks like a function
    // eslint-disable-next-line @typescript-eslint/no-misused-promises
    new Subscription(async () => {
        const subscriptionProxy = await subscriptionPromise
        await subscriptionProxy.unsubscribe()
        subscriptionProxy[releaseProxy]()
    })

/**
 * Runs f and returns a resolved promise with its value or a rejected promise with its exception,
 * regardless of whether it returns a promise or not.
 */
export const tryCatchPromise = async <T>(function_: () => T | Promise<T>): Promise<T> => function_()

/**
 * Reports whether value is a Promise.
 */
export const isPromiseLike = (value: unknown): value is PromiseLike<unknown> =>
    typeof value === 'object' && value !== null && hasProperty('then')(value) && typeof value.then === 'function'

/**
 * Reports whether value is a {@link sourcegraph.Subscribable}.
 */
export const isSubscribable = (value: unknown): value is Subscribable<unknown> =>
    typeof value === 'object' &&
    value !== null &&
    hasProperty('subscribe')(value) &&
    typeof value.subscribe === 'function'

/**
 * Promisifies method calls and objects if specified, throws otherwise if there is no stub provided
 * NOTE: it does not handle ProxyMethods and callbacks yet
 * NOTE2: for testing purposes only!!
 */
export const pretendRemote = <T extends object>(object: T): Remote<T> =>
    new Proxy(object, {
        get: (target, key) => {
            if (!keyExistsIn(key, target)) {
                return undefined
            }
            const value = target[key]
            if (typeof value === 'function') {
                return (...args: unknown[]) => Promise.resolve(value(...args))
            }
            return Promise.resolve(value)
        },
    }) as Remote<T>

// TODO this object is better than just casting, but it is still not good.
// Calling these function will never work as expected and the test needs to override the things actually used,
// meaning it needs to have knowledge of the internals of what is being tested.
export const noopFlatExtensionHostAPI: FlatExtHostAPI = {
    getWorkspaceRoots: () => [],
    addWorkspaceRoot: noop,
    removeWorkspaceRoot: noop,
    syncVersionContext: noop,
    transformSearchQuery: (query: string) => proxySubscribable(of(query)),
    syncSettingsData: noop,
    addTextDocumentIfNotExists: noop,
    addViewerIfNotExists: () => {
        throw new Error('Not implemented')
    },
    removeViewer: noop,
    setEditorSelections: noop,
    getActiveCodeEditorPosition: () => proxySubscribable(EMPTY),
    getDecorations: () => proxySubscribable(of([])),
    // Languages
    getHover: () => proxySubscribable(of({ isLoading: false, result: null })),
    getDefinitions: () => proxySubscribable(of({ isLoading: false, result: [] })),
    getReferences: () => proxySubscribable(of({ isLoading: false, result: [] })),
    getLocations: () => proxySubscribable(of({ isLoading: false, result: [] })),
    hasReferenceProvider: () => proxySubscribable(of(false)),
}

export const noopMainThreadAPI: MainThreadAPI = {
    getActiveExtensions: () => proxySubscribable(EMPTY),
    getScriptURLForExtension: identity,
    applySettingsEdit: () => Promise.resolve(),
    executeCommand: () => Promise.resolve(),
    registerCommand: () => proxy(new Subscription()),
}
