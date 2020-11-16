import { SettingsCascade } from '../../settings/settings'
import { Remote, proxy } from 'comlink'
import * as sourcegraph from 'sourcegraph'
import { BehaviorSubject, Subject, of, Observable, from, concat } from 'rxjs'
import { FlatExtensionHostAPI, MainThreadAPI } from '../contract'
import { syncSubscription } from '../util'
import { switchMap, mergeMap, map, defaultIfEmpty, catchError, distinctUntilChanged } from 'rxjs/operators'
import { proxySubscribable, providerResultToObservable } from './api/common'
import { TextDocumentIdentifier, match } from '../client/types/textDocument'
import { getModeFromPath } from '../../languages'
import { parseRepoURI } from '../../util/url'
import { ExtensionDocuments } from './api/documents'
import { fromLocation, toPosition } from './api/types'
import { TextDocumentPositionParameters } from '../protocol'
import { LOADING, MaybeLoadingResult } from '@sourcegraph/codeintellify'
import { combineLatestOrDefault } from '../../util/rxjs/combineLatestOrDefault'
import * as extensionApiTypes from '@sourcegraph/extension-api-types'
import { castArray, isEqual } from 'lodash'
import { fromHoverMerged, HoverMerged } from '../client/types/hover'
import { isNot, isExactly, isDefined } from '../../util/types'
import { WorkspaceRoot } from './api/workspace'

/**
 * Holds the entire state exposed to the extension host
 * as a single object
 */
export interface ExtensionHostState {
    settings: Readonly<SettingsCascade<object>>

    // Workspace
    roots: WorkspaceRoot[]
    versionContext: string | undefined

    // Search
    queryTransformers: BehaviorSubject<sourcegraph.QueryTransformer[]>

    // Lang
    hoverProviders: BehaviorSubject<RegisteredProvider<sourcegraph.HoverProvider>[]>
    documentHighlightProviders: BehaviorSubject<RegisteredProvider<sourcegraph.DocumentHighlightProvider>[]>
    definitionProviders: BehaviorSubject<RegisteredProvider<sourcegraph.DefinitionProvider>[]>
}

export interface RegisteredProvider<T> {
    selector: sourcegraph.DocumentSelector
    provider: T
}

export interface InitResult {
    configuration: sourcegraph.ConfigurationService
    workspace: PartialWorkspaceNamespace
    exposedToMain: FlatExtensionHostAPI
    // todo this is needed as a temp solution for getter problem
    state: Readonly<ExtensionHostState>
    commands: typeof sourcegraph['commands']
    search: typeof sourcegraph['search']
    languages: Pick<
        typeof sourcegraph['languages'],
        'registerHoverProvider' | 'registerDocumentHighlightProvider' | 'registerDefinitionProvider'
    >
}

/**
 * mimics sourcegraph.workspace namespace without documents
 */
export type PartialWorkspaceNamespace = Omit<
    typeof sourcegraph['workspace'],
    'textDocuments' | 'onDidOpenTextDocument' | 'openedTextDocuments' | 'roots' | 'versionContext'
>

/**
 * Holds internally ExtState and manages communication with the Client
 * Returns the initialized public extension API pieces ready for consumption and the internal extension host API ready to be exposed to the main thread
 * NOTE that this function will slowly merge with the one in extensionHost.ts
 *
 * @param mainAPI
 */
export const initNewExtensionAPI = (
    mainAPI: Remote<MainThreadAPI>,
    initialSettings: Readonly<SettingsCascade<object>>,
    textDocuments: ExtensionDocuments
): InitResult => {
    const state: ExtensionHostState = {
        roots: [],
        versionContext: undefined,
        settings: initialSettings,
        queryTransformers: new BehaviorSubject<sourcegraph.QueryTransformer[]>([]),
        hoverProviders: new BehaviorSubject<RegisteredProvider<sourcegraph.HoverProvider>[]>([]),
        documentHighlightProviders: new BehaviorSubject<RegisteredProvider<sourcegraph.DocumentHighlightProvider>[]>(
            []
        ),
        definitionProviders: new BehaviorSubject<RegisteredProvider<sourcegraph.DefinitionProvider>[]>([]),
    }

    // Most extensions never call `configuration.get()` synchronously in
    // `activate()` to get the initial settings data, and instead only subscribe
    // to configuration changes. In order for these extensions to be able to
    // access settings, make sure `configuration` emits on subscription by
    // making it a BehaviorSubject.
    const configChanges = new BehaviorSubject<void>(undefined)

    const rootChanges = new Subject<void>()

    const versionContextChanges = new Subject<string | undefined>()

    const exposedToMain: FlatExtensionHostAPI = {
        // Configuration
        syncSettingsData: data => {
            state.settings = Object.freeze(data)
            configChanges.next()
        },

        // Workspace
        addWorkspaceRoot: (root: extensionApiTypes.WorkspaceRoot) => {
            if (!state.roots.some(existingRoot => existingRoot.uri.toString() !== root.uri)) {
                state.roots = [...state.roots, new WorkspaceRoot(new URL(root.uri), root.inputRevision)]
            }
            rootChanges.next()
        },
        getWorkspaceRoots: () => state.roots.map(root => ({ ...root, uri: root.uri.toString() })),
        removeWorkspaceRoot: (uri: string) => {
            state.roots = state.roots.filter(root => root.uri.toString() !== uri)
            rootChanges.next()
        },
        setVersionContext: context => {
            state.versionContext = context
            versionContextChanges.next()
        },
        getVersionContext: () => state.versionContext, // TODO remove if this isn't needed

        // Search
        transformSearchQuery: query =>
            // TODO (simon) I don't enjoy the dark arts below
            // we return observable because of potential deferred addition of transformers
            // in this case we need to reissue the transformation and emit the resulting value
            // we probably won't need an Observable if we somehow coordinate with extensions activation
            proxySubscribable(
                state.queryTransformers.pipe(
                    switchMap(transformers =>
                        transformers.reduce(
                            (currentQuery: Observable<string>, transformer) =>
                                currentQuery.pipe(
                                    mergeMap(query => {
                                        const result = transformer.transformQuery(query)
                                        return result instanceof Promise ? from(result) : of(result)
                                    })
                                ),
                            of(query)
                        )
                    )
                )
            ),

        // Language
        getHover: (textParameters: TextDocumentPositionParameters) => {
            const document = textDocuments.get(textParameters.textDocument.uri)
            const position = toPosition(textParameters.position)

            return proxySubscribable(
                callProviders(
                    state.hoverProviders,
                    document,
                    provider => provider.provideHover(document, position),
                    mergeHoverResults
                )
            )
        },
        getDocumentHighlights: (textParameters: TextDocumentPositionParameters) => {
            const document = textDocuments.get(textParameters.textDocument.uri)
            const position = toPosition(textParameters.position)

            return proxySubscribable(
                callProviders(
                    state.documentHighlightProviders,
                    document,
                    provider => provider.provideDocumentHighlights(document, position),
                    mergeDocumentHighlightResults
                ).pipe(map(result => (result.isLoading ? [] : result.result)))
            )
        },
        getDefinition: (textParameters: TextDocumentPositionParameters) => {
            const document = textDocuments.get(textParameters.textDocument.uri)
            const position = toPosition(textParameters.position)

            return proxySubscribable(
                callProviders(
                    state.definitionProviders,
                    document,
                    provider => provider.provideDefinition(document, position),
                    mergeDefinition
                )
            )
        },
    }

    // Configuration
    const getConfiguration = <C extends object>(): sourcegraph.Configuration<C> => {
        const snapshot = state.settings.final as Readonly<C>

        const configuration: sourcegraph.Configuration<C> & { toJSON: any } = {
            value: snapshot,
            get: key => snapshot[key],
            update: (key, value) => mainAPI.applySettingsEdit({ path: [key as string | number], value }),
            toJSON: () => snapshot,
        }
        return configuration
    }

    // Workspace
    const workspace: PartialWorkspaceNamespace = {
        onDidChangeRoots: rootChanges.asObservable(),
        rootChanges: rootChanges.asObservable(),
        versionContextChanges: versionContextChanges.asObservable(),
    }

    // Commands
    const commands: typeof sourcegraph['commands'] = {
        executeCommand: (command, ...args) => mainAPI.executeCommand(command, args),
        registerCommand: (command, callback) => syncSubscription(mainAPI.registerCommand(command, proxy(callback))),
    }

    // Search
    const search: typeof sourcegraph['search'] = {
        registerQueryTransformer: transformer => addWithRollback(state.queryTransformers, transformer),
    }

    // Languages
    const registerHoverProvider = (
        selector: sourcegraph.DocumentSelector,
        provider: sourcegraph.HoverProvider
    ): sourcegraph.Unsubscribable => addWithRollback(state.hoverProviders, { selector, provider })
    const registerDocumentHighlightProvider = (
        selector: sourcegraph.DocumentSelector,
        provider: sourcegraph.DocumentHighlightProvider
    ): sourcegraph.Unsubscribable => addWithRollback(state.documentHighlightProviders, { selector, provider })
    // definition
    const registerDefinitionProvider = (
        selector: sourcegraph.DocumentSelector,
        provider: sourcegraph.DefinitionProvider
    ): sourcegraph.Unsubscribable => addWithRollback(state.definitionProviders, { selector, provider })

    return {
        configuration: Object.assign(configChanges.asObservable(), {
            get: getConfiguration,
        }),
        exposedToMain,
        workspace,
        state,
        commands,
        search,
        languages: {
            registerHoverProvider,
            registerDocumentHighlightProvider,
            registerDefinitionProvider,
        },
    }
}

// TODO (loic, felix) it might make sense to port tests with the rest of provider registries.
/**
 * Filters a list of Providers (P type) based on their selectors and a document
 *
 * @param document to use for filtering
 * @param entries array of providers (P[])
 * @param selector a way to get a selector from a Provider
 * @returns a filtered array of providers
 */
function providersForDocument<P>(
    document: TextDocumentIdentifier,
    entries: P[],
    selector: (p: P) => sourcegraph.DocumentSelector
): P[] {
    return entries.filter(provider =>
        match(selector(provider), {
            uri: document.uri,
            languageId: getModeFromPath(parseRepoURI(document.uri).filePath || ''),
        })
    )
}

/**
 * calls next() on behaviorSubject with a immutably added element ([...old, value])
 *
 * @param behaviorSubject subject that holds a collection
 * @param value to add to a collection
 * @returns Unsubscribable that will remove that element from the behaviorSubject.value and call next() again
 */
function addWithRollback<T>(behaviorSubject: BehaviorSubject<T[]>, value: T): sourcegraph.Unsubscribable {
    behaviorSubject.next([...behaviorSubject.value, value])
    return {
        unsubscribe: () => behaviorSubject.next(behaviorSubject.value.filter(item => item !== value)),
    }
}

/**
 * Helper function to abstract common logic of invoking language providers.
 *
 * 1. filters providers based on document
 * 2. invokes filtered providers via invokeProvider function
 * 3. adds [LOADING] state for each provider result stream
 * 4. omits errors from provider results with potential logging
 * 5. aggregates latests results from providers based on mergeResult function
 *
 * @param providersObservable observable of provider collection (expected to emit if a provider was added or removed)
 * @param document used for filtering providers
 * @param invokeProvider specifies how to get results from a provider (usually a closure over provider arguments)
 * @param mergeResult specifies how providers results should be aggregated
 * @param logErrors if console.error should be used for reporting errors from providers
 * @returns observable of aggregated results from all providers based on mergeResults function
 */
export function callProviders<TProvider, TProviderResult, TMergedResult>(
    providersObservable: Observable<RegisteredProvider<TProvider>[]>,
    document: TextDocumentIdentifier,
    invokeProvider: (provider: TProvider) => sourcegraph.ProviderResult<TProviderResult>,
    mergeResult: (providerResults: (TProviderResult | 'loading' | null | undefined)[]) => TMergedResult,
    logErrors: boolean = true
): Observable<MaybeLoadingResult<TMergedResult>> {
    return providersObservable
        .pipe(
            map(providers => providersForDocument(document, providers, ({ selector }) => selector)),
            switchMap(providers =>
                combineLatestOrDefault(
                    providers.map(provider =>
                        concat(
                            [LOADING],
                            providerResultToObservable(invokeProvider(provider.provider)).pipe(
                                defaultIfEmpty<typeof LOADING | TProviderResult | null | undefined>(null),
                                catchError(error => {
                                    if (logErrors) {
                                        console.error('Provider errored:', error)
                                    }
                                    return [null]
                                })
                            )
                        )
                    )
                )
            )
        )
        .pipe(
            defaultIfEmpty<(typeof LOADING | TProviderResult | null | undefined)[]>([]),
            map(results => ({
                isLoading: results.some(hover => hover === LOADING),
                result: mergeResult(results),
            })),
            distinctUntilChanged((a, b) => isEqual(a, b))
        )
}

/**
 * merges latest results from hover providers into a form that is convenient to show
 *
 * @param results latests results from hover providers
 * @returns a {@link HoverMerged} results if there are any actual Hover results or null in case of no results or loading
 */
export function mergeHoverResults(
    results: (typeof LOADING | extensionApiTypes.Hover | null | undefined)[]
): HoverMerged | null {
    return fromHoverMerged(results.filter(isNot(isExactly(LOADING))))
}

/**
 * Merges definition result and converts it to client types for sending it to the main thread.
 *
 * @param results Results from all definition providers.
 */
export function mergeDefinition(
    results: (typeof LOADING | sourcegraph.Definition | null | undefined)[]
): extensionApiTypes.Location[] {
    return results
        .filter(isNot(isExactly(LOADING)))
        .flatMap(castArray)
        .filter(isDefined)
        .map(fromLocation)
}

/**
 * @param results latests results from document highlight providers
 */
export function mergeDocumentHighlightResults(
    results: (typeof LOADING | sourcegraph.DocumentHighlight[] | null | undefined)[]
): sourcegraph.DocumentHighlight[] {
    return results.filter(isNot(isExactly(LOADING))).flatMap(highlights => highlights || [])
}
