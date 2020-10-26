import { Remote } from 'comlink'
import { Subscription, Observable, combineLatest, zip, from } from 'rxjs'
import { asError, isErrorLike } from '../../../util/errors'
import { tryCatchPromise } from '../../util'
<<<<<<< HEAD:shared/src/api/extension/api/extensions.ts
import { ExecutableExtension } from '../../client/services/extensionsService'
import { wrapRemoteObservable } from '../../client/api/common'
import { startWith, bufferCount, map, switchMap, catchError } from 'rxjs/operators'
import { ConfiguredExtension, getScriptURLFromExtensionManifest } from '../../../extensions/extension'
import { isDefined } from '../../../util/types'
import { memoizeObservable } from '../../../util/memoizeObservable'
import { MainThreadAPI } from '../../contract'
=======

/** @internal */
export interface ExtensionExtensionsAPI extends ProxyMarked {
    $activateExtension(extensionID: string, bundleURL: string): Promise<void>
    $deactivateExtension(extensionID: string): Promise<void>
}
>>>>>>> main:client/shared/src/api/extension/api/extensions.ts

/** The WebWorker's global scope */
declare const self: any

<<<<<<< HEAD:shared/src/api/extension/api/extensions.ts
type ExtensionDeactivate = () => void

/**
 * Proxy method invoked by the client to load an extension and invoke its `activate` function to start running it.
 *
 * It also sets up global hooks so that when the extension's code uses `require('sourcegraph')` and
 * `import 'sourcegraph'`, it gets the extension API handle (the value specified in
 * sourcegraph.d.ts).
 *
 * @param extensionID The extension ID of the extension to activate.
 * @param bundleURL The URL to the JavaScript source file (that exports an `activate` function) for
 * the extension.
 * @returns A promise that resolves when the extension's activation finishes (i.e., when it returns if it's synchronous, or when the promise it returns resolves if it's async).
 */
const activateExtension = async (extensionID: string, bundleURL: string): Promise<ExtensionDeactivate> => {
    // Load the extension bundle and retrieve the extension entrypoint module's exports on
    // the global `module` property.
    try {
        const exports = {}
        self.exports = exports
        self.module = { exports }
        self.importScripts(bundleURL)
    } catch (error) {
        throw new Error(
            `error thrown while executing extension ${JSON.stringify(
                extensionID
            )} bundle (in importScripts of ${bundleURL}): ${String(error)}`
        )
    }
    const extensionExports = self.module.exports
    delete self.exports
    delete self.module
=======
/** @internal */
export class Extensions implements ExtensionExtensionsAPI, Unsubscribable, ProxyMarked {
    public readonly [proxyMarker] = true
>>>>>>> main:client/shared/src/api/extension/api/extensions.ts

    if (!('activate' in extensionExports)) {
        throw new Error(
            `extension bundle for ${JSON.stringify(extensionID)} has no exported activate function (in ${bundleURL})`
        )
    }

    // During extension deactivation, we first call the extension's `deactivate` function and then unsubscribe
    // the Subscription passed to the `activate` function.
    const extensionSubscriptions = new Subscription()
    const extensionDeactivate = typeof extensionExports.deactivate === 'function' ? extensionExports.deactivate : null
    const deactivate: ExtensionDeactivate = () => {
        try {
            if (extensionDeactivate) {
                extensionDeactivate()
            }
        } finally {
            extensionSubscriptions.unsubscribe()
        }
    }

    // The behavior should be consistent for both sync and async activate functions that throw
    // errors or reject. Both cases should yield a rejected promise.
    //
    // TODO(sqs): Add timeouts to prevent long-running activate or deactivate functions from
    // significantly delaying other extensions.
    await tryCatchPromise<void>(() => extensionExports.activate({ subscriptions: extensionSubscriptions })).catch(
        error => {
            error = asError(error)
            throw Object.assign(
                new Error(
                    `error during extension ${JSON.stringify(extensionID)} activate function: ${String(
                        error.stack || error
                    )}`
                ),
                { error }
            )
        }
    )

    return deactivate
}

function extensionsWithMatchedActivationEvent(
    enabledExtensions: ConfiguredExtension[],
    visibleTextDocumentLanguages: ReadonlySet<string>
): ConfiguredExtension[] {
    const languageActivationEvents = new Set(
        [...visibleTextDocumentLanguages].map(language => `onLanguage:${language}`)
    )
    return enabledExtensions.filter(extension => {
        try {
            if (!extension.manifest) {
                const match = /^sourcegraph\/lang-(.*)$/.exec(extension.id)
                if (match) {
                    console.warn(
                        `Extension ${extension.id} has been renamed to sourcegraph/${match[1]}. It's safe to remove ${extension.id} from your settings.`
                    )
                } else {
                    console.warn(
                        `Extension ${extension.id} was not found. Remove it from settings to suppress this warning.`
                    )
                }
                return false
            }
            if (isErrorLike(extension.manifest)) {
                console.warn(extension.manifest)
                return false
            }
            if (!extension.manifest.activationEvents) {
                console.warn(`Extension ${extension.id} has no activation events, so it will never be activated.`)
                return false
            }
            return extension.manifest.activationEvents.some(
                event => event === '*' || languageActivationEvents.has(event)
            )
        } catch (error) {
            console.error(error)
        }
        return false
    })
}

export const handleExtensionActivation = (
    {
        getActiveExtensions,
        getScriptURLForExtension,
    }: Pick<Remote<MainThreadAPI>, 'getActiveExtensions' | 'getScriptURLForExtension'>,
    activeLanguages: Observable<ReadonlySet<string>>
): Subscription => {
    const subscriptions = new Subscription()
    const extensionDeactivateFunctions = new Map<string, ExtensionDeactivate>()
    const memoizedGetScriptURL = memoizeObservable<string, string | null>(
        url =>
            from(getScriptURLForExtension(url)).pipe(
                catchError(error => {
                    console.error(`Error fetching extension script URL ${url}`, error)
                    return [null]
                })
            ),
        url => url
    )
    subscriptions.add(
        combineLatest([wrapRemoteObservable(getActiveExtensions(), subscriptions), activeLanguages])
            .pipe(
                map(([enabledExtensions, activeLanguages]) =>
                    extensionsWithMatchedActivationEvent(enabledExtensions, activeLanguages)
                ),
                switchMap(activeExtensions =>
                    zip(
                        activeExtensions.map(extension =>
                            memoizedGetScriptURL(getScriptURLFromExtensionManifest(extension)).pipe(
                                map((scriptURL): ExecutableExtension | null =>
                                    scriptURL === null
                                        ? null
                                        : {
                                              id: extension.id,
                                              manifest: extension.manifest,
                                              scriptURL,
                                          }
                                )
                            )
                        )
                    )
                ),
                map(extensions => extensions.filter(isDefined)),
                startWith([] as ExecutableExtension[]),
                bufferCount(2, 1)
            )
            .subscribe(([oldExtensions, newExtensions]) => {
                // Diff next state's activated extensions vs. current state's.
                if (!newExtensions) {
                    newExtensions = oldExtensions
                }
                const toActivate = [...newExtensions] // clone to avoid mutating state stored by bufferCount
                const toDeactivate: ExecutableExtension[] = []
                const next: ExecutableExtension[] = []
                if (oldExtensions) {
                    for (const extension of oldExtensions) {
                        const newIndex = toActivate.findIndex(({ id }) => extension.id === id)
                        if (newIndex === -1) {
                            // Extension is no longer activated
                            toDeactivate.push(extension)
                        } else {
                            // Extension is already activated.
                            toActivate.splice(newIndex, 1)
                            next.push(extension)
                        }
                    }
                }

                // Deactivate extensions that are no longer in use.
                for (const extension of toDeactivate) {
                    const deactivate = extensionDeactivateFunctions.get(extension.id)
                    if (deactivate) {
                        tryCatchPromise(deactivate).catch(error => {
                            console.warn(`Error deactivating extension ${JSON.stringify(extension.id)}:`, error)
                        })
                        extensionDeactivateFunctions.delete(extension.id)
                    }
                }

                // Activate extensions that haven't yet been activated.
                for (const extension of toActivate) {
                    console.log('Activating Sourcegraph extension:', extension.id)
                    activateExtension(extension.id, extension.scriptURL)
                        .then(deactivate => extensionDeactivateFunctions.set(extension.id, deactivate))
                        .catch(error => {
                            console.error(`Error activating extension ${JSON.stringify(extension.id)}:`, error)
                        })
                }
            })
    )
    return subscriptions
}
