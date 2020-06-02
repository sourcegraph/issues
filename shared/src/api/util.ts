import { ProxyMarked, transferHandlers, releaseProxy, TransferHandler, Remote } from 'comlink'
import { Subscription } from 'rxjs'
import { Subscribable, Unsubscribable } from 'sourcegraph'
import { hasProperty } from '../util/types'

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
 * When calling a method that returns a proxied `Subscription` through comlink, we don't get a proxy for the
 * `Subscription` object directly, but a `Promise` for the proxy of the `Subscription` object. However, it is
 * easier to work with a synchronous `Subscription` object, and the proxy can also not always be passed directly to
 * rxjs. This function wraps the Promise for the proxy into a synchronous Subscription. If the Subscription is
 * unsubscribed before the Promise resolves, it will wait until the Promise resolves, then immediately unsubscribe
 * the Subscription proxy.
 *
 * Since a Subscription can only be unsubscribed once and the proxy is therefor no longer needed afterwards, it
 * will then also release the proxy.
 *
 * @param subscriptionPromise A Promise for a Subscription proxied from the other thread
 */
export const synchronousSubscription = (
    subscriptionPromise: Promise<Remote<Unsubscribable & ProxyMarked>>
): Subscription =>
    // eslint-disable-next-line @typescript-eslint/no-misused-promises
    new Subscription(async function (this: any) {
        // Wait to retrieve the proxy
        const subscriptionProxy = await subscriptionPromise

        // Forward the unsubscribe
        await subscriptionProxy.unsubscribe()

        // Release the proxy, since it's no longer needed.
        subscriptionProxy[releaseProxy]()

        // Workaround for https://github.com/ReactiveX/rxjs/issues/5464
        // remove when fixed
        this._unsubscribe = null
    })

/**
 * Runs f and returns a resolved promise with its value or a rejected promise with its exception,
 * regardless of whether it returns a promise or not.
 */
export const tryCatchPromise = async <T>(f: () => T | Promise<T>): Promise<T> => f()

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
export const pretendRemote = <T>(obj: Partial<T>): Remote<T> =>
    // eslint-disable-next-line @typescript-eslint/no-unsafe-return
    (new Proxy(obj, {
        get: (a, prop) => {
            if (prop in a) {
                if (typeof (a as any)[prop] !== 'function') {
                    return Promise.resolve((a as any)[prop])
                }

                return (...args: any[]) => Promise.resolve((a as any)[prop](...args))
            }
            throw new Error(`unspecified property in the stub ${prop.toString()}`)
        },
    }) as unknown) as Remote<T>
