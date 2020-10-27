import { SettingsCascade } from '../../settings/settings'
import { Remote, proxy } from 'comlink'
<<<<<<< HEAD:shared/src/api/extension/flatExtensionApi.ts
import type * as sourcegraph from 'sourcegraph'
import { BehaviorSubject, Subject, of, Observable, from, concat, OperatorFunction, Subscription } from 'rxjs'
import { FlatExtHostAPI, MainThreadAPI } from '../contract'
=======
import * as sourcegraph from 'sourcegraph'
import { BehaviorSubject, Subject, of, Observable, from, concat } from 'rxjs'
import { FlatExtensionHostAPI, MainThreadAPI } from '../contract'
>>>>>>> main:client/shared/src/api/extension/flatExtensionApi.ts
import { syncSubscription } from '../util'
import { switchMap, mergeMap, map, defaultIfEmpty, catchError, distinctUntilChanged, mapTo } from 'rxjs/operators'
import { proxySubscribable, providerResultToObservable } from './api/common'
import { TextDocumentIdentifier, match } from '../client/types/textDocument'
import { getModeFromPath } from '../../languages'
import { parseRepoURI } from '../../util/url'
<<<<<<< HEAD:shared/src/api/extension/flatExtensionApi.ts
import { toPosition, fromLocation, fromDocumentHighlight } from './api/types'
import { TextDocumentPositionParams, ReferenceParams } from '../protocol'
=======
import { ExtensionDocuments } from './api/documents'
import { toPosition } from './api/types'
import { TextDocumentPositionParameters } from '../protocol'
>>>>>>> main:client/shared/src/api/extension/flatExtensionApi.ts
import { LOADING, MaybeLoadingResult } from '@sourcegraph/codeintellify'
import { combineLatestOrDefault } from '../../util/rxjs/combineLatestOrDefault'
import { isEqual, castArray } from 'lodash'
import { fromHoverMerged } from '../client/types/hover'
import { isNot, isExactly, allOf, isDefined } from '../../util/types'
import { ViewerId, ExtensionViewer } from '../viewerTypes'
import { ReferenceCounter } from '../../util/ReferenceCounter'
import { ExtensionDocument } from './api/textDocument'
import { ExtensionCodeEditor } from './api/codeEditor'
import { ExtensionDirectoryViewer } from './api/directoryViewer'
import { ExtensionWorkspaceRoot } from './api/workspaceRoot'

/**
 * Holds the entire state exposed to the extension host
 * as a single object
 */
<<<<<<< HEAD:shared/src/api/extension/flatExtensionApi.ts
export interface ExtState {
    settings: BehaviorSubject<Readonly<SettingsCascade<object>>>
=======
export interface ExtensionHostState {
    settings: Readonly<SettingsCascade<object>>
>>>>>>> main:client/shared/src/api/extension/flatExtensionApi.ts

    // Workspace
    roots: BehaviorSubject<readonly ExtensionWorkspaceRoot[]>
    versionContext: BehaviorSubject<string | undefined>

    // Search
    queryTransformers: BehaviorSubject<readonly sourcegraph.QueryTransformer[]>

    // Lang
    hoverProviders: BehaviorSubject<readonly RegisteredProvider<sourcegraph.HoverProvider>[]>
    definitionProviders: BehaviorSubject<readonly RegisteredProvider<sourcegraph.DefinitionProvider>[]>
    referencesProviders: BehaviorSubject<readonly RegisteredProvider<sourcegraph.ReferenceProvider>[]>
    locationProviders: BehaviorSubject<readonly (RegisteredProvider<sourcegraph.LocationProvider> & { id: string })[]>
    documentHighlightProviders: BehaviorSubject<readonly RegisteredProvider<sourcegraph.DocumentHighlightProvider>[]>
}

export interface RegisteredProvider<T> {
    selector: sourcegraph.DocumentSelector
    provider: T
}

<<<<<<< HEAD:shared/src/api/extension/flatExtensionApi.ts
const matchProvider = (textDocument: TextDocumentIdentifier) => <T>(provider: RegisteredProvider<T>): boolean =>
    match(provider.selector, {
        uri: textDocument.uri,
        languageId: getModeFromPath(parseRepoURI(textDocument.uri).filePath || ''),
    })

export interface InitResult
    extends Pick<typeof sourcegraph, 'commands' | 'search' | 'languages' | 'workspace' | 'configuration'> {
    exposedToMain: FlatExtHostAPI
    state: Readonly<ExtState>
=======
export interface InitResult {
    configuration: sourcegraph.ConfigurationService
    workspace: PartialWorkspaceNamespace
    exposedToMain: FlatExtensionHostAPI
    // todo this is needed as a temp solution for getter problem
    state: Readonly<ExtensionHostState>
    commands: typeof sourcegraph['commands']
    search: typeof sourcegraph['search']
    languages: Pick<typeof sourcegraph['languages'], 'registerHoverProvider' | 'registerDocumentHighlightProvider'>
>>>>>>> main:client/shared/src/api/extension/flatExtensionApi.ts
}

/**
 * mimics sourcegraph.workspace namespace without documents
 */
export type PartialWorkspaceNamespace = Omit<typeof sourcegraph['workspace'], 'roots' | 'versionContext'>

const VIEWER_NOT_FOUND_ERROR_NAME = 'ViewerNotFoundError'
class ViewerNotFoundError extends Error {
    public readonly name = VIEWER_NOT_FOUND_ERROR_NAME
    constructor(viewerId: string) {
        super(`Viewer not found: ${viewerId}`)
    }
}

function assertViewerType<T extends ExtensionViewer['type']>(
    viewer: ExtensionViewer,
    type: T
): asserts viewer is ExtensionViewer & { type: T } {
    if (viewer.type !== type) {
        throw new Error(`Viewer ID ${viewer.viewerId} is type ${viewer.type}, expected ${type}`)
    }
}

/**
 * Holds internally ExtState and manages communication with the Client
 * Returns the initialized public extension API pieces ready for consumption and the internal extension host API ready to be exposed to the main thread
 * NOTE that this function will slowly merge with the one in extensionHost.ts
 *
 * @param mainAPI
 */
export const initNewExtensionAPI = (
    mainAPI: Remote<MainThreadAPI>,
    initialSettings: Readonly<SettingsCascade<object>>
): InitResult => {
<<<<<<< HEAD:shared/src/api/extension/flatExtensionApi.ts
    const state: ExtState = {
        // Most extensions never call `configuration.get()` synchronously in `activate()` to get
        // the initial settings data, and instead only subscribe to configuration changes.
        // In order for these extensions to be able to access settings, make sure `configuration` emits on subscription.
        settings: new BehaviorSubject<Readonly<SettingsCascade<object>>>(initialSettings),
        roots: new BehaviorSubject<readonly ExtensionWorkspaceRoot[]>([]),
        versionContext: new BehaviorSubject<string | undefined>(undefined),
        queryTransformers: new BehaviorSubject<readonly sourcegraph.QueryTransformer[]>([]),
        hoverProviders: new BehaviorSubject<readonly RegisteredProvider<sourcegraph.HoverProvider>[]>([]),
        definitionProviders: new BehaviorSubject<readonly RegisteredProvider<sourcegraph.DefinitionProvider>[]>([]),
        referencesProviders: new BehaviorSubject<readonly RegisteredProvider<sourcegraph.ReferenceProvider>[]>([]),
        locationProviders: new BehaviorSubject<
            readonly (RegisteredProvider<sourcegraph.LocationProvider> & { id: string })[]
        >([]),
        documentHighlightProviders: new BehaviorSubject<
            readonly RegisteredProvider<sourcegraph.DocumentHighlightProvider>[]
        >([]),
    }

    let lastViewerId = 0

    /** Mutable map of viewer ID to viewer. */
    const viewComponents = new Map<string, ExtensionCodeEditor | ExtensionDirectoryViewer>()
    const activeViewComponentChanges = new BehaviorSubject<ExtensionViewer | undefined>(undefined)

    /** A map of URIs to text documents */
    const textDocuments = new Map<string, ExtensionDocument>()
    const openedTextDocuments = new Subject<ExtensionDocument>()
    const activeLanguages = new BehaviorSubject<ReadonlySet<string>>(new Set())
    const languageReferences = new ReferenceCounter<string>()

    const getTextDocument = (uri: string): ExtensionDocument => {
        const textDocument = textDocuments.get(uri)
        if (!textDocument) {
            throw new Error(`Text document does not exist with URI ${uri}`)
        }
        return textDocument
=======
    const state: ExtensionHostState = {
        roots: [],
        versionContext: undefined,
        settings: initialSettings,
        queryTransformers: new BehaviorSubject<sourcegraph.QueryTransformer[]>([]),
        hoverProviders: new BehaviorSubject<RegisteredProvider<sourcegraph.HoverProvider>[]>([]),
        documentHighlightProviders: new BehaviorSubject<RegisteredProvider<sourcegraph.DocumentHighlightProvider>[]>(
            []
        ),
>>>>>>> main:client/shared/src/api/extension/flatExtensionApi.ts
    }

    /**
     * Returns the Viewer with the given viewerId.
     * Throws if no viewer exists with the given viewerId.
     */
    const getViewer = (viewerId: ViewerId['viewerId']): ExtensionViewer => {
        const viewer = viewComponents.get(viewerId)
        if (!viewer) {
            throw new ViewerNotFoundError(viewerId)
        }
        return viewer
    }

    /**
     * Removes a model.
     *
     * @param uri The URI of the model to remove.
     */
    const removeTextDocument = (uri: string): void => {
        const model = getTextDocument(uri)
        textDocuments.delete(uri)
        if (languageReferences.decrement(model.languageId)) {
            activeLanguages.next(new Set<string>(languageReferences.keys()))
        }
    }

    const modelReferences = new ReferenceCounter()

    const exposedToMain: FlatExtensionHostAPI = {
        // Configuration
        syncSettingsData: data => {
            state.settings.next(Object.freeze(data))
        },

        // Workspace
        getWorkspaceRoots: () => state.roots.value.map(({ uri, inputRevision }) => ({ uri: uri.href, inputRevision })),
        addWorkspaceRoot: root => {
            state.roots.next(Object.freeze([...state.roots.value, new ExtensionWorkspaceRoot(root)]))
        },
        removeWorkspaceRoot: uri => {
            state.roots.next(Object.freeze(state.roots.value.filter(workspace => workspace.uri.href !== uri)))
        },

        syncVersionContext: context => {
            state.versionContext.next(context)
        },

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
<<<<<<< HEAD:shared/src/api/extension/flatExtensionApi.ts
        getHover: (parameters: TextDocumentPositionParams) => {
            const textDocument = getTextDocument(parameters.textDocument.uri)
            const position = toPosition(parameters.position)

            return proxySubscribable(
                state.hoverProviders.pipe(
                    map(providers => providers.filter(matchProvider(textDocument))),
                    callProviders(provider => provider.provideHover(textDocument, position), fromHoverMerged)
                )
            )
        },
        getDefinitions: (parameters: TextDocumentPositionParams) => {
            const textDocument = getTextDocument(parameters.textDocument.uri)
            const position = toPosition(parameters.position)

            return proxySubscribable(
                state.definitionProviders.pipe(
                    map(providers => providers.filter(matchProvider(textDocument))),
                    callProviders(
                        provider => provider.provideDefinition(textDocument, position),
                        definitions => definitions.flatMap(castArray).map(fromLocation)
                    )
                )
            )
        },
        getReferences: (parameters: ReferenceParams) => {
            const textDocument = getTextDocument(parameters.textDocument.uri)
            const position = toPosition(parameters.position)
            const context = parameters.context

            return proxySubscribable(
                state.referencesProviders.pipe(
                    map(providers => providers.filter(matchProvider(textDocument))),
                    callProviders(
                        provider => provider.provideReferences(textDocument, position, context),
                        locations => locations.flat().map(fromLocation)
                    )
                )
            )
        },
        hasReferenceProvider: textDocument =>
            proxySubscribable(
                state.referencesProviders.pipe(
                    map(providers => providers.filter(matchProvider(textDocument)).length > 0)
                )
            ),
        getLocations: (id: string, parameters: TextDocumentPositionParams) => {
            const textDocument = getTextDocument(parameters.textDocument.uri)
            const position = toPosition(parameters.position)
=======
        getHover: (textParameters: TextDocumentPositionParameters) => {
            const document = textDocuments.get(textParameters.textDocument.uri)
            const position = toPosition(textParameters.position)
>>>>>>> main:client/shared/src/api/extension/flatExtensionApi.ts

            return proxySubscribable(
                state.locationProviders.pipe(
                    map(providers =>
                        providers.filter(provider => id === provider.id && matchProvider(textDocument)(provider))
                    ),
                    callProviders(
                        provider => provider.provideLocations(textDocument, position),
                        locations => locations.flat().map(fromLocation)
                    )
                )
            )
        },
<<<<<<< HEAD:shared/src/api/extension/flatExtensionApi.ts
        getDocumentHighlights: (textParameters: TextDocumentPositionParams) => {
            const document = getTextDocument(textParameters.textDocument.uri)
=======
        getDocumentHighlights: (textParameters: TextDocumentPositionParameters) => {
            const document = textDocuments.get(textParameters.textDocument.uri)
>>>>>>> main:client/shared/src/api/extension/flatExtensionApi.ts
            const position = toPosition(textParameters.position)

            return proxySubscribable(
                state.documentHighlightProviders.pipe(
                    callProviders(
                        provider => provider.provideDocumentHighlights(document, position),
                        highlights => highlights.flat().map(fromDocumentHighlight)
                    ),
                    map(result => (result.isLoading ? [] : result.result))
                )
            )
        },
        // Viewer
        addViewerIfNotExists: viewerData => {
            const viewerId = `viewer#${++lastViewerId}`
            if (viewerData.type === 'CodeEditor') {
                modelReferences.increment(viewerData.resource)
            }
            let viewComponent: ExtensionViewer
            switch (viewerData.type) {
                case 'CodeEditor': {
                    const textDocument = getTextDocument(viewerData.resource)
                    viewComponent = new ExtensionCodeEditor({ ...viewerData, viewerId }, textDocument)
                    break
                }
                case 'DirectoryViewer': {
                    viewComponent = new ExtensionDirectoryViewer({ ...viewerData, viewerId })
                    break
                }
            }
            viewComponents.set(viewerId, viewComponent)
            if (viewerData.isActive) {
                activeViewComponentChanges.next(viewComponent)
            }
            if (viewerData.isActive) {
                activeViewComponentChanges.next(viewComponent)
            }
            return { viewerId }
        },
        removeViewer: ({ viewerId }) => {
            const viewer = getViewer(viewerId)
            viewComponents.delete(viewerId)
            // Check if this was the active viewer
            if (activeViewComponentChanges.value?.viewerId === viewerId) {
                activeViewComponentChanges.next(undefined)
            }
            if (viewer.type === 'CodeEditor' && modelReferences.decrement(viewer.resource)) {
                removeTextDocument(viewer.resource)
            }
        },
        getActiveCodeEditorPosition: () =>
            proxySubscribable(
                activeViewComponentChanges.pipe(
                    map(activeViewer => {
                        if (activeViewer?.type !== 'CodeEditor') {
                            return null
                        }
                        const sel = activeViewer.selections[0]
                        if (!sel) {
                            return null
                        }
                        // TODO(sqs): Return null for empty selections (but currently all selected tokens are treated as an empty
                        // selection at the beginning of the token, so this would break a lot of things, so we only do this for empty
                        // selections when the start character is -1). HACK(sqs): Character === -1 means that the whole line is
                        // selected (this is a bug in the caller, but it is useful here).
                        const isEmpty =
                            sel.start.line === sel.end.line &&
                            sel.start.character === sel.end.character &&
                            sel.start.character === -1
                        if (isEmpty) {
                            return null
                        }
                        return {
                            textDocument: { uri: activeViewer.resource },
                            position: sel.start,
                        }
                    })
                )
            ),
        setEditorSelections: ({ viewerId }, selections) => {
            const viewer = getViewer(viewerId)
            assertViewerType(viewer, 'CodeEditor')
            viewer.update({ selections })
        },
        getDecorations: ({ viewerId }) => {
            const viewer = getViewer(viewerId)
            assertViewerType(viewer, 'CodeEditor')
            return proxySubscribable(viewer.mergedDecorations)
        },

        // Text documents
        addTextDocumentIfNotExists: textDocumentData => {
            if (textDocuments.has(textDocumentData.uri)) {
                return
            }
            const textDocument = new ExtensionDocument(textDocumentData)
            textDocuments.set(textDocumentData.uri, textDocument)
            openedTextDocuments.next(textDocument)
            // Update activeLanguages if no other existing model has the same language.
            if (languageReferences.increment(textDocumentData.languageId)) {
                activeLanguages.next(new Set<string>(languageReferences.keys()))
            }
        },
    }

    // Configuration
    const getConfiguration = <C extends object>(): sourcegraph.Configuration<C> => {
        const snapshot = state.settings.value.final as Readonly<C>

        const configuration: sourcegraph.Configuration<C> & { toJSON: any } = {
            value: snapshot,
            get: key => snapshot[key],
            update: (key, value) => mainAPI.applySettingsEdit({ path: [key as string | number], value }),
            toJSON: () => snapshot,
        }
        return configuration
    }

    // Workspace
    const workspace: typeof sourcegraph.workspace = {
        get roots() {
            return state.roots.value
        },
        get versionContext() {
            return state.versionContext.value
        },
        get textDocuments() {
            return [...textDocuments.values()]
        },
        openedTextDocuments: openedTextDocuments.asObservable(),
        onDidOpenTextDocument: openedTextDocuments.asObservable(),
        onDidChangeRoots: state.roots.pipe(mapTo(undefined)),
        rootChanges: state.roots.pipe(mapTo(undefined)),
        versionContextChanges: state.versionContext.pipe(mapTo(undefined)),
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

    return {
        exposedToMain,
        configuration: Object.assign(state.settings.pipe(mapTo(undefined)), {
            get: getConfiguration,
        }),
        workspace,
        state,
        commands,
        search,
        languages: {
            registerHoverProvider: (selector, provider) =>
                addWithRollback(state.hoverProviders, { selector, provider }),
            registerDefinitionProvider: (selector, provider) =>
                addWithRollback(state.definitionProviders, { selector, provider }),
            registerReferenceProvider: (selector, provider) =>
                addWithRollback(state.referencesProviders, { selector, provider }),
            registerLocationProvider: (id, selector, provider) =>
                addWithRollback(state.locationProviders, { id, selector, provider }),
            registerDocumentHighlightProvider: (selector, provider) =>
                addWithRollback(state.documentHighlightProviders, { selector, provider }),
            registerCompletionItemProvider: () => {
                console.warn(
                    'sourcegraph.languages.registerCompletionProvider was removed for the time being. It has no effect.'
                )
                return new Subscription()
            },
        },
    }
}

// TODO (loic, felix) it might make sense to port tests with the rest of provider registries.

/**
 * calls next() on behaviorSubject with a immutably added element ([...old, value])
 *
 * @param behaviorSubject subject that holds a collection
 * @param value to add to a collection
 * @returns Unsubscribable that will remove that element from the behaviorSubject.value and call next() again
 */
function addWithRollback<T>(behaviorSubject: BehaviorSubject<readonly T[]>, value: T): sourcegraph.Unsubscribable {
    behaviorSubject.next([...behaviorSubject.value, value])
    return new Subscription(() => behaviorSubject.next(behaviorSubject.value.filter(item => item !== value)))
}

/**
 * Takes a stream of providers, calls them using `invokeProvider` and merges the results using `mergeResults`.
 *
 * @param providersObservable observable of provider collection (expected to emit if a provider was added or removed)
 * @param invokeProvider specifies how to get results from a provider (usually a closure over provider arguments)
 * @param mergeResults specifies how provider results should be aggregated
 * @param logErrors if console.error should be used for reporting errors from providers
 * @returns observable of aggregated results from all providers based on mergeResults function
 */
export function callProviders<TProvider, TProviderResult, TMergedResult>(
    invokeProvider: (provider: TProvider) => sourcegraph.ProviderResult<TProviderResult>,
    mergeResults: (providerResults: readonly Exclude<TProviderResult, null | undefined>[]) => TMergedResult,
    logErrors: boolean = true
): OperatorFunction<readonly RegisteredProvider<TProvider>[], MaybeLoadingResult<TMergedResult>> {
    return providersObservable =>
        providersObservable
            .pipe(
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
                    result: mergeResults(results.filter(allOf(isNot(isExactly(LOADING)), isDefined))),
                })),
                distinctUntilChanged((a, b) => isEqual(a, b))
            )
}
