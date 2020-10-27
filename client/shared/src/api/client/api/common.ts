import { Remote, proxyMarker, releaseProxy, ProxyMethods, ProxyMarked, proxy, UnproxyOrClone } from 'comlink'
import { noop } from 'lodash'
import { from, Observable, observable as symbolObservable, Subscription, Unsubscribable, of } from 'rxjs'
import { mergeMap, finalize } from 'rxjs/operators'
import { Subscribable } from 'sourcegraph'
import { ProxySubscribable } from '../../extension/api/common'
import { syncSubscription, isPromiseLike } from '../../util'
import { asError } from '../../../util/errors'
import { FeatureProviderRegistry } from '../services/registry'

// We subclass because rxjs checks instanceof Subscription.
// By exposing a Subscription as the interface to release the proxy,
// the released/not released state is inspectable and other Subcriptions
// can be smart about releasing references when this Subscription is closed.
// Subscriptions notify parent Subscriptions when they are unsubscribed.

/**
 * A `Subscription` representing the `MessagePort` used by a comlink proxy.
 * Unsubscribing will send a RELEASE message over the MessagePort, then close it and remove all event listeners from it.
 */
export class ProxySubscription extends Subscription {
    constructor(proxy: Pick<ProxyMethods, typeof releaseProxy>) {
        super(() => {
            proxy[releaseProxy]()
        })
    }
}

/**
 * An object that is backed by a comlink Proxy and exposes its Subscription so consumers can release it.
 */
export interface ProxySubscribed {
    readonly proxySubscription: Subscription
}

/**
 * An ordinary Observable linked to an Observable in another thread through a comlink Proxy.
 */
export interface RemoteObservable<T> extends Observable<T>, ProxySubscribed {}

/**
 * When a Subscribable is returned from the other thread (wrapped with `proxySubscribable()`),
 * this thread gets a `Promise` for a `Subscribable` _proxy_ where `subscribe()` returns a `Promise<Unsubscribable>`.
 * This function wraps that proxy in a real Rx Observable where `subscribe()` returns a `Subscription` directly as expected.
 *
 * The returned Observable is augmented with the `releaseProxy` method from comlink to release the underlying `MessagePort`.
 *
 * @param proxyPromise The proxy to the `ProxyObservable` in the other thread
 * @param addToSubscription If provided, directly adds the `ProxySubscription` to this Subscription.
 */
export const wrapRemoteObservable = <T>(
    proxyPromise: Promise<Remote<ProxySubscribable<T>>> | Remote<ProxySubscribable<T>>,
    addToSubscription?: Subscription
): RemoteObservable<T> => {
    const proxySubscription = new Subscription()
    if (addToSubscription) {
        addToSubscription.add(proxySubscription)
    }
    const observable = (isPromiseLike(proxyPromise) ? from(proxyPromise) : of(proxyPromise)).pipe(
        mergeMap(
            (proxySubscribable): Subscribable<T> => {
                proxySubscription.add(new ProxySubscription(proxySubscribable))
                return {
                    // Needed for Rx type check
                    [symbolObservable](): Subscribable<T> {
                        return this
                    },
                    subscribe(...args: any[]): Subscription {
                        // Always subscribe with an object because the other side
                        // is unable to tell if a Proxy is a function or an observer object
                        // (they always appear as functions)
                        let proxyObserver: Parameters<typeof proxySubscribable['subscribe']>[0]
                        if (typeof args[0] === 'function') {
                            proxyObserver = {
                                [proxyMarker]: true,
                                next: args[0] || noop,
                                error: args[1] ? error => args[1](asError(error)) : noop,
                                complete: args[2] || noop,
                            }
                        } else {
                            const partialObserver = args[0] || {}
                            proxyObserver = {
                                [proxyMarker]: true,
                                next: partialObserver.next ? value => partialObserver.next(value) : noop,
                                error: partialObserver.error ? error => partialObserver.error(asError(error)) : noop,
                                complete: partialObserver.complete ? () => partialObserver.complete() : noop,
                            }
                        }
                        return syncSubscription(proxySubscribable.subscribe(proxyObserver))
                    },
                }
            }
        )
    )
    return Object.assign(observable, { proxySubscription })
}

/**
 * Releases the underlying MessagePort of a remote Observable when it completes or is unsubscribed from.
 *
 * Important: This will prevent resubscribing to the Observable. Only use this operator in a scope where it is known
 * that no resubscriptions can happen after completion, e.g. in a `switchMap()` callback.
 *
 * Must be used as the first parameter to `pipe()`, because the source must be a `RemoteObservable`.
 */
// needed for the type parameter
// eslint-disable-next-line unicorn/consistent-function-scoping
export const finallyReleaseProxy = <T>() => (source: Observable<T> & Partial<ProxySubscribed>): Observable<T> => {
    const { proxySubscription } = source
    if (!proxySubscription) {
        console.warn('finallyReleaseProxy() used on Observable without proxy subscription')
        return source
    }
    return source.pipe(finalize(() => proxySubscription.unsubscribe()))
}

/**
 * Helper function to register a remote provider returning an Observable, proxied by comlink, in a provider registry.
 *
 * @param registry The registry to register the provider on.
 * @param registrationOptions The registration options to pass to `registerProvider()`
 * @param remoteProviderFunction The provider function in a remote thread, proxied by `comlink`.
 *
 * @returns A Subscription that can be proxied through comlink which will unregister the provider.
 */
export function registerRemoteProvider<
    TRegistrationOptions,
    TLocalProviderParameters extends UnproxyOrClone<TProviderParameters>,
    TProviderParameters,
    TProviderResult
>(
    registry: FeatureProviderRegistry<
        TRegistrationOptions,
        (parameters: TLocalProviderParameters) => Observable<TProviderResult>
    >,
    registrationOptions: TRegistrationOptions,
    remoteProviderFunction: Remote<
        ((parameters: TProviderParameters) => ProxySubscribable<TProviderResult>) & ProxyMarked
    >
): Unsubscribable & ProxyMarked {
    // This subscription will unregister the provider when unsubscribed.
    const subscription = new Subscription()

    subscription.add(
        registry.registerProvider(registrationOptions, parameters =>
            // Wrap the remote, proxied Observable in an ordinary Observable
            // and add its underlying proxy subscription to our subscription
            // to release the proxy when the provider gets unregistered.
            wrapRemoteObservable(remoteProviderFunction(parameters), subscription)
        )
    )

    // Track the underlying proxy subscription of the provider in our subscription
    // so that the proxy gets released when the provider gets unregistered.
    subscription.add(new ProxySubscription(remoteProviderFunction))

    // Prepare the subscription to be proxied to the remote side.
    return proxy(subscription)
}
