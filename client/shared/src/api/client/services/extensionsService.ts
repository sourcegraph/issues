import { isEqual } from 'lodash'
import { combineLatest, from, Observable, ObservableInput, of, Subscribable } from 'rxjs'
import { fromFetch } from 'rxjs/fetch'
import { catchError, distinctUntilChanged, map, switchMap, tap } from 'rxjs/operators'
import {
    ConfiguredExtension,
    getScriptURLFromExtensionManifest,
    isExtensionEnabled,
} from '../../../extensions/extension'
import { viewerConfiguredExtensions } from '../../../extensions/helpers'
import { PlatformContext } from '../../../platform/context'
import { isErrorLike } from '../../../util/errors'
import { memoizeObservable } from '../../../util/memoizeObservable'
import { combineLatestOrDefault } from '../../../util/rxjs/combineLatestOrDefault'
import { isDefined } from '../../../util/types'
import { ModelService } from './modelService'
import { checkOk } from '../../../backend/fetch'
import { ExtensionManifest } from '../../../schema/extensionSchema'

/**
 * The information about an extension necessary to execute and activate it.
 */
export interface ExecutableExtension extends Pick<ConfiguredExtension, 'id' | 'manifest'> {
    /** The URL to the JavaScript bundle of the extension. */
    scriptURL: string
}

/**
 * The manifest of an extension sideloaded during local development.
 *
 * Doesn't include {@link ExtensionManifest#url}, as this is added when
 * publishing an extension to the registry.
 * Instead, the bundle URL is computed from the manifest's `main` field.
 */
interface SideloadedExtensionManifest extends Omit<ExtensionManifest, 'url'> {
    name: string
    main: string
}

const getConfiguredSideloadedExtension = (baseUrl: string): Observable<ConfiguredExtension> =>
    fromFetch(`${baseUrl}/package.json`, { selector: response => checkOk(response).json() }).pipe(
        map(
            (response: SideloadedExtensionManifest): ConfiguredExtension => ({
                id: response.name,
                manifest: {
                    ...response,
                    url: `${baseUrl}/${response.main.replace('dist/', '')}`,
                },
                rawManifest: null,
            })
        )
    )

export interface PartialContext
    extends Pick<PlatformContext, 'requestGraphQL' | 'getScriptURLForExtension' | 'settings'> {
    sideloadedExtensionURL: Subscribable<string | null>
}

export interface IExtensionsService {
    activeExtensions: Subscribable<ExecutableExtension[]>
}

/**
 * Manages the set of extensions that are available and activated.
 *
 * @internal This is an internal implementation detail and is different from the product feature called the
 * "extension registry" (where users can search for and enable extensions).
 */
export class ExtensionsService implements IExtensionsService {
    constructor(
        private platformContext: PartialContext,
        private modelService: Pick<ModelService, 'activeLanguages'>,
        private extensionActivationFilter = extensionsWithMatchedActivationEvent,
        private fetchSideloadedExtension: (
            baseUrl: string
        ) => Subscribable<ConfiguredExtension | null> = getConfiguredSideloadedExtension
    ) {}

    protected configuredExtensions: Subscribable<ConfiguredExtension[]> = viewerConfiguredExtensions({
        settings: this.platformContext.settings,
        requestGraphQL: this.platformContext.requestGraphQL,
    })

    /**
     * Returns an observable that emits the set of enabled extensions upon subscription and whenever it changes.
     *
     * Most callers should use {@link ExtensionsService#activeExtensions}.
     */
    private get enabledExtensions(): Subscribable<ConfiguredExtension[]> {
        return combineLatest([
            from(this.platformContext.settings),
            from(this.configuredExtensions),
            this.sideloadedExtension,
        ]).pipe(
            map(([settings, configuredExtensions, sideloadedExtension]) => {
                let enabled = configuredExtensions.filter(extension => isExtensionEnabled(settings.final, extension.id))

                if (sideloadedExtension) {
                    if (!isErrorLike(sideloadedExtension.manifest) && sideloadedExtension.manifest?.publisher) {
                        // Disable extension with the same ID while this extension is sideloaded
                        const constructedID = `${sideloadedExtension.manifest.publisher}/${sideloadedExtension.id}`
                        enabled = enabled.filter(extension => extension.id !== constructedID)
                    }

                    enabled.push(sideloadedExtension)
                }

                return enabled
            })
        )
    }

    private get sideloadedExtension(): Subscribable<ConfiguredExtension | null> {
        return from(this.platformContext.sideloadedExtensionURL).pipe(
            switchMap(url => (url ? this.fetchSideloadedExtension(url) : of(null))),
            catchError(error => {
                console.error('Error sideloading extension', error)
                return of(null)
            })
        )
    }

    /**
     * Returns an observable that emits the set of extensions that should be active, based on the previous and
     * current state and each available extension's activationEvents.
     *
     * An extension is activated when one or more of its activationEvents is true. After an extension has been
     * activated, it remains active for the rest of the session (i.e., for as long as the browser tab remains open)
     * as long as it remains enabled. If it is disabled, it is deactivated. (I.e., "activationEvents" are
     * retrospective/sticky.)
     *
     * @todo Consider whether extensions should be deactivated if none of their activationEvents are true (or that
     * plus a certain period of inactivity).
     */
    public get activeExtensions(): Subscribable<ExecutableExtension[]> {
        // Extensions that have been activated (including extensions with zero "activationEvents" that evaluate to
        // true currently).
        const activatedExtensionIDs = new Set<string>()
        return combineLatest([from(this.modelService.activeLanguages), this.enabledExtensions]).pipe(
            tap(([activeLanguages, enabledExtensions]) => {
                const activeExtensions = this.extensionActivationFilter(enabledExtensions, activeLanguages)
                for (const extension of activeExtensions) {
                    if (!activatedExtensionIDs.has(extension.id)) {
                        activatedExtensionIDs.add(extension.id)
                    }
                }
            }),
            map(([, extensions]) =>
                extensions ? extensions.filter(extension => activatedExtensionIDs.has(extension.id)) : []
            ),
            distinctUntilChanged((a, b) =>
                isEqual(new Set(a.map(extension => extension.id)), new Set(b.map(extension => extension.id)))
            ),
            switchMap(extensions =>
                combineLatestOrDefault(
                    extensions.map(extension =>
                        this.memoizedGetScriptURLForExtension(getScriptURLFromExtensionManifest(extension)).pipe(
                            map(scriptURL =>
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
            distinctUntilChanged((a, b) =>
                isEqual(new Set(a.map(extension => extension.id)), new Set(b.map(extension => extension.id)))
            )
        )
    }

    private memoizedGetScriptURLForExtension = memoizeObservable<string, string | null>(
        url =>
            asObservable(this.platformContext.getScriptURLForExtension(url)).pipe(
                catchError(error => {
                    console.error(`Error fetching extension script URL ${url}`, error)
                    return [null]
                })
            ),
        url => url
    )
}

function asObservable(input: string | ObservableInput<string>): Observable<string> {
    return typeof input === 'string' ? of(input) : from(input)
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
