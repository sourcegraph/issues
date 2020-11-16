import {
    ContextResolver,
    createHoverifier,
    findPositionsFromEvents,
    Hoverifier,
    HoverState,
    MaybeLoadingResult,
    DiffPart,
} from '@sourcegraph/codeintellify'
import { TextDocumentDecoration } from '@sourcegraph/extension-api-types'
import * as H from 'history'
import * as React from 'react'
import { render as reactDOMRender } from 'react-dom'
import {
    asyncScheduler,
    combineLatest,
    EMPTY,
    from,
    Observable,
    of,
    Subject,
    Subscription,
    Unsubscribable,
    concat,
    BehaviorSubject,
} from 'rxjs'
import {
    catchError,
    concatAll,
    concatMap,
    filter,
    finalize,
    map,
    mergeMap,
    observeOn,
    switchMap,
    withLatestFrom,
    tap,
    startWith,
    distinctUntilChanged,
    retryWhen,
    mapTo,
} from 'rxjs/operators'
import { ActionItemAction } from '../../../../../shared/src/actions/ActionItem'
import { DecorationMapByLine } from '../../../../../shared/src/api/client/services/decoration'
import { CodeEditorData, CodeEditorWithPartialModel } from '../../../../../shared/src/api/client/services/viewerService'
import {
    isPrivateRepoPublicSourcegraphComErrorLike,
    isRepoNotFoundErrorLike,
} from '../../../../../shared/src/backend/errors'
import {
    CommandListClassProps,
    CommandListPopoverButtonClassProps,
} from '../../../../../shared/src/commandPalette/CommandList'
import { CompletionWidgetClassProps } from '../../../../../shared/src/components/completion/CompletionWidget'
import { asObservable } from '../../../../../shared/src/util/rxjs/asObservable'
import { ApplyLinkPreviewOptions } from '../../../../../shared/src/components/linkPreviews/linkPreviews'
import { Controller } from '../../../../../shared/src/extensions/controller'
import { registerHighlightContributions } from '../../../../../shared/src/highlight/contributions'
import { getHoverActions, registerHoverContributions } from '../../../../../shared/src/hover/actions'
import { HoverContext, HoverOverlay, HoverOverlayClassProps } from '../../../../../shared/src/hover/HoverOverlay'
import { getModeFromPath } from '../../../../../shared/src/languages'
import { URLToFileContext } from '../../../../../shared/src/platform/context'
import { TelemetryProps } from '../../../../../shared/src/telemetry/telemetryService'
import { isDefined, isInstanceOf, property } from '../../../../../shared/src/util/types'
import {
    FileSpec,
    UIPositionSpec,
    RawRepoSpec,
    RepoSpec,
    ResolvedRevisionSpec,
    RevisionSpec,
    toRootURI,
    toURIWithPath,
    ViewStateSpec,
} from '../../../../../shared/src/util/url'
import { observeStorageKey } from '../../../browser-extension/web-extension-api/storage'
import { isExtension, isInPage } from '../../context'
import { SourcegraphIntegrationURLs, BrowserPlatformContext } from '../../platform/context'
import { toTextDocumentIdentifier, toTextDocumentPositionParameters } from '../../backend/extension-api-conversion'
import { CodeViewToolbar, CodeViewToolbarClassProps } from '../../components/CodeViewToolbar'
import { resolveRevision, retryWhenCloneInProgressError } from '../../repo/backend'
import { EventLogger, ConditionalTelemetryService } from '../../tracking/eventLogger'
import { MutationRecordLike, querySelectorOrSelf } from '../../util/dom'
import { featureFlags } from '../../util/featureFlags'
import { bitbucketServerCodeHost } from '../bitbucket/codeHost'
import { githubCodeHost } from '../github/codeHost'
import { gitlabCodeHost } from '../gitlab/codeHost'
import { phabricatorCodeHost } from '../phabricator/codeHost'
import { CodeView, trackCodeViews, fetchFileContentForDiffOrFileInfo } from './codeViews'
import { ContentView, handleContentViews } from './contentViews'
import { applyDecorations, initializeExtensions, renderCommandPalette, renderGlobalDebug } from './extensions'
import { ViewOnSourcegraphButtonClassProps, ViewOnSourcegraphButton } from './ViewOnSourcegraphButton'
import {
    createPrivateCodeHoverAlert,
    getActiveHoverAlerts,
    onHoverAlertDismissed,
    userNeedsToSetupPrivateInstance,
} from './hoverAlerts'
import {
    handleNativeTooltips,
    NativeTooltip,
    nativeTooltipsEnabledFromSettings,
    registerNativeTooltipContributions,
} from './nativeTooltips'
import { handleTextFields, TextField } from './textFields'
import { delayUntilIntersecting, ViewResolver } from './views'
import { IS_LIGHT_THEME } from './consts'
import { NotificationType, HoverAlert } from 'sourcegraph'
import { isHTTPAuthError } from '../../../../../shared/src/backend/fetch'
import { asError } from '../../../../../shared/src/util/errors'
import { resolveRepoNamesForDiffOrFileInfo, defaultRevisionToCommitID } from './util/fileInfo'
import { wrapRemoteObservable } from '../../../../../shared/src/api/client/api/common'
import { HoverMerged } from '../../../../../shared/src/api/client/types/hover'
import { isFirefox, observeSourcegraphURL } from '../../util/context'
import { shouldOverrideSendTelemetry, observeOptionFlag } from '../../util/optionFlags'
import { BackgroundPageApi } from '../../../browser-extension/web-extension-api/types'
import { background } from '../../../browser-extension/web-extension-api/runtime'
import { ReferenceCounter } from '../../../../../shared/src/util/ReferenceCounter'

registerHighlightContributions()

export interface OverlayPosition {
    top: number
    left: number
}

/**
 * A function that gets the mount location for elements being mounted to the DOM.
 *
 * - If the mount doesn't belong into the container, it must return `null`.
 * - If the mount already exists in the container, it must return the existing mount.
 * - If the mount does not exist yet in the container, it must create and return it.
 *
 * Caveats:
 * - The passed element might be the mount itself
 * - The passed element might be an element _within_ the mount
 */
export type MountGetter = (container: HTMLElement) => HTMLElement | null

/**
 * The context the code host is in on the current page.
 */
export type CodeHostContext = RawRepoSpec & Partial<RevisionSpec> & { privateRepository: boolean }

export type CodeHostType = 'github' | 'phabricator' | 'bitbucket-server' | 'gitlab'

/** Information for adding code intelligence to code views on arbitrary code hosts. */
export interface CodeHost extends ApplyLinkPreviewOptions {
    /**
     * The type of the code host. This will be added as a className to the overlay mount.
     * Use {@link CodeHost#name} if you need a human-readable name for the code host to display in the UI.
     */
    type: CodeHostType

    /**
     * A human-readable name for the code host, to be displayed in the UI.
     */
    name: string

    /**
     * Basic contextual information for the current code host.
     */
    getContext?: () => CodeHostContext

    /**
     * Mount getter for the repository "View on Sourcegraph" button.
     *
     * If undefined, the "View on Sourcegraph" button won't be rendered on the code host.
     */
    getViewContextOnSourcegraphMount?: MountGetter

    /**
     * Optional class name for the contextual link to Sourcegraph.
     */
    viewOnSourcegraphButtonClassProps?: ViewOnSourcegraphButtonClassProps

    /**
     * Checks to see if the current context the code is running in is within
     * the given code host.
     */
    check: () => boolean

    /**
     * CSS classes for ActionItem buttons in the hover overlay to customize styling
     */
    hoverOverlayClassProps?: HoverOverlayClassProps

    /**
     * Resolve {@link CodeView}s from the DOM.
     */
    codeViewResolvers: ViewResolver<CodeView>[]

    /**
     * Resolve {@link ContentView}s from the DOM.
     */
    contentViewResolvers?: ViewResolver<ContentView>[]

    /**
     * Resolve {@link TextField}s from the DOM.
     */
    textFieldResolvers?: ViewResolver<TextField>[]

    /**
     * Resolves {@link NativeTooltip}s from the DOM.
     */
    nativeTooltipResolvers?: ViewResolver<NativeTooltip>[]

    /**
     * Adjust the position of the hover overlay. Useful for fixed headers or other
     * elements that throw off the position of the tooltip within the relative
     * element.
     */
    adjustOverlayPosition?: (position: OverlayPosition) => OverlayPosition

    // Extensions related input

    /**
     * Mount getter for the command palette button for extensions.
     *
     * If undefined, the command palette button won't be rendered on the code host.
     */
    getCommandPaletteMount?: MountGetter

    /**
     * Returns a selector used to determine the mount location of the hover overlay in the DOM.
     *
     * If undefined, or when null is returned, the hover overlay container will be mounted to <body>.
     */
    getHoverOverlayMountLocation?: () => string | null

    /**
     * Construct the URL to the specified file.
     *
     * @param sourcegraphURL The URL of the Sourcegraph instance.
     * @param target The target to build a URL for.
     * @param context Context information about this invocation.
     */
    urlToFile?: (
        sourcegraphURL: string,
        target: RepoSpec & RawRepoSpec & RevisionSpec & FileSpec & Partial<UIPositionSpec> & Partial<ViewStateSpec>,
        context: URLToFileContext
    ) => string

    notificationClassNames: Record<NotificationType, string>

    /**
     * CSS classes for the command palette to customize styling
     */
    commandPaletteClassProps?: CommandListPopoverButtonClassProps & CommandListClassProps

    /**
     * CSS classes for the code view toolbar to customize styling
     */
    codeViewToolbarClassProps?: CodeViewToolbarClassProps

    /**
     * CSS classes for the completion widget to customize styling
     */
    completionWidgetClassProps?: CompletionWidgetClassProps

    /**
     * Whether or not code views need to be tokenized. Defaults to false.
     */
    codeViewsRequireTokenization?: boolean
}

/**
 * A blob (single file `FileInfo`) or a diff (with a head `FileInfo` and/or base `FileInfo`)
 */
export type DiffOrBlobInfo<T extends FileInfo = FileInfo> = BlobInfo<T> | DiffInfo<T>
export interface BlobInfo<T extends FileInfo = FileInfo> {
    blob: T
}
export type DiffInfo<T extends FileInfo = FileInfo> =
    // `base?: undefined` avoids making `{ head: T; base: T }` assignable to this type
    { head: T; base?: undefined } | { base: T; head?: undefined } | { head: T; base: T }

export interface FileInfo {
    /**
     * The path for the repo the file belongs to.
     */
    rawRepoName: string
    /**
     * The path for the file path for a given `codeView`.
     */
    filePath: string
    /**
     * The commit that the code view is at.
     */
    commitID: string
    /**
     * The revision the code view is at.
     */
    revision?: string
}

export interface FileInfoWithRepoName extends FileInfo, RepoSpec {}
export interface FileInfoWithContent extends FileInfoWithRepoName {
    content?: string
}

export interface CodeIntelligenceProps extends TelemetryProps {
    platformContext: Pick<
        BrowserPlatformContext,
        | 'forceUpdateTooltip'
        | 'urlToFile'
        | 'sideloadedExtensionURL'
        | 'requestGraphQL'
        | 'settings'
        | 'refreshSettings'
        | 'sourcegraphURL'
    >
    codeHost: CodeHost
    extensionsController: Controller
    showGlobalDebug?: boolean
}

export const createOverlayMount = (codeHostName: string, container: HTMLElement): HTMLElement => {
    const mount = document.createElement('div')
    mount.classList.add('hover-overlay-mount', `hover-overlay-mount__${codeHostName}`, 'theme-light')
    container.append(mount)
    return mount
}

export const createGlobalDebugMount = (): HTMLElement => {
    const mount = document.createElement('div')
    mount.className = 'global-debug'
    document.body.append(mount)
    return mount
}

/**
 * Prepares the page for code intelligence. It creates the hoverifier, injects
 * and mounts the hover overlay and then returns the hoverifier.
 */
function initCodeIntelligence({
    mutations,
    codeHost,
    platformContext,
    extensionsController,
    render,
    telemetryService,
    hoverAlerts,
}: Pick<CodeIntelligenceProps, 'codeHost' | 'platformContext' | 'extensionsController' | 'telemetryService'> & {
    render: typeof reactDOMRender
    hoverAlerts: Observable<HoverAlert>[]
    mutations: Observable<MutationRecordLike[]>
}): {
    hoverifier: Hoverifier<RepoSpec & RevisionSpec & FileSpec & ResolvedRevisionSpec, HoverMerged, ActionItemAction>
    subscription: Unsubscribable
} {
    const subscription = new Subscription()

    /** Emits when the close button was clicked */
    const closeButtonClicks = new Subject<MouseEvent>()

    /** Emits whenever the ref callback for the hover element is called */
    const hoverOverlayElements = new Subject<HTMLElement | null>()

    const relativeElement = document.body

    const containerComponentUpdates = new Subject<void>()

    subscription.add(
        registerHoverContributions({
            extensionsController,
            platformContext,
            history: H.createBrowserHistory(),
            locationAssign: location.assign.bind(location),
        })
    )

    // Code views come and go, but there is always a single hoverifier on the page
    const hoverifier = createHoverifier<
        RepoSpec & RevisionSpec & FileSpec & ResolvedRevisionSpec,
        HoverMerged,
        ActionItemAction
    >({
        closeButtonClicks,
        hoverOverlayElements,
        hoverOverlayRerenders: containerComponentUpdates.pipe(
            withLatestFrom(hoverOverlayElements),
            map(([, hoverOverlayElement]) => ({ hoverOverlayElement, relativeElement })),
            filter(property('hoverOverlayElement', isDefined))
        ),
        getHover: ({ line, character, part, ...rest }) =>
            concat(
                [{ isLoading: true, result: null }],
                combineLatest([
                    from(extensionsController.extHostAPI).pipe(
                        switchMap(extensionHost =>
                            wrapRemoteObservable(
                                extensionHost.getHover(
                                    toTextDocumentPositionParameters({ ...rest, position: { line, character } })
                                )
                            )
                        )
                    ),
                    getActiveHoverAlerts([
                        ...hoverAlerts,
                        ...(userNeedsToSetupPrivateInstance(codeHost, platformContext.sourcegraphURL)
                            ? [of(createPrivateCodeHoverAlert(codeHost))]
                            : []),
                    ]),
                ]).pipe(
                    map(
                        ([{ isLoading, result: hoverMerged }, alerts]): MaybeLoadingResult<HoverMerged | null> => ({
                            isLoading,
                            result: hoverMerged || alerts?.length ? { contents: [], ...hoverMerged, alerts } : null,
                        })
                    )
                )
            ),
        getDocumentHighlights: ({ line, character, part, ...rest }) =>
            from(extensionsController.extHostAPI).pipe(
                switchMap(extensionHost =>
                    wrapRemoteObservable(
                        extensionHost.getDocumentHighlights(
                            toTextDocumentPositionParameters({ ...rest, position: { line, character } })
                        )
                    )
                )
            ),
        getActions: context => getHoverActions({ extensionsController, platformContext }, context),
        pinningEnabled: true,
        tokenize: codeHost.codeViewsRequireTokenization,
    })

    class HoverOverlayContainer extends React.Component<{}, HoverState<HoverContext, HoverMerged, ActionItemAction>> {
        private subscription = new Subscription()
        private nextOverlayElement = hoverOverlayElements.next.bind(hoverOverlayElements)
        private nextCloseButtonClick = closeButtonClicks.next.bind(closeButtonClicks)

        constructor(props: {}) {
            super(props)
            this.state = hoverifier.hoverState
            this.subscription.add(
                hoverifier.hoverStateUpdates.subscribe(update => {
                    this.setState(update)
                })
            )
        }
        public componentDidMount(): void {
            containerComponentUpdates.next()
        }
        public componentWillUnmount(): void {
            this.subscription.unsubscribe()
        }
        public componentDidUpdate(): void {
            containerComponentUpdates.next()
        }
        public render(): JSX.Element | null {
            const hoverOverlayProps = this.getHoverOverlayProps()
            return hoverOverlayProps ? (
                <HoverOverlay
                    {...hoverOverlayProps}
                    {...codeHost.hoverOverlayClassProps}
                    telemetryService={telemetryService}
                    isLightTheme={IS_LIGHT_THEME}
                    hoverRef={this.nextOverlayElement}
                    extensionsController={extensionsController}
                    platformContext={platformContext}
                    location={H.createLocation(window.location)}
                    onCloseButtonClick={this.nextCloseButtonClick}
                    onAlertDismissed={onHoverAlertDismissed}
                />
            ) : null
        }
        private getHoverOverlayProps(): HoverState<HoverContext, HoverMerged, ActionItemAction>['hoverOverlayProps'] {
            if (!this.state.hoverOverlayProps) {
                return undefined
            }
            let { overlayPosition, ...rest } = this.state.hoverOverlayProps
            // TODO: is adjustOverlayPosition needed or could it be solved with a better relativeElement?
            if (overlayPosition && codeHost.adjustOverlayPosition) {
                overlayPosition = codeHost.adjustOverlayPosition(overlayPosition)
            }
            return { ...rest, overlayPosition }
        }
    }

    const { getHoverOverlayMountLocation } = codeHost
    if (!getHoverOverlayMountLocation) {
        // This renders to document.body, which we can assume is never removed,
        // so we don't need to subscribe to mutations.
        const overlayMount = createOverlayMount(codeHost.type, document.body)
        render(<HoverOverlayContainer />, overlayMount)
    } else {
        let previousMount: HTMLElement | null = null
        subscription.add(
            observeHoverOverlayMountLocation(getHoverOverlayMountLocation, mutations).subscribe(mountLocation => {
                // Remove the previous mount if it exists,
                // to avoid displaying duplicate hovers.
                if (previousMount) {
                    previousMount.remove()
                }
                const mount = createOverlayMount(codeHost.type, mountLocation)
                previousMount = mount
                render(<HoverOverlayContainer />, mount)
            })
        )
    }

    return { hoverifier, subscription }
}

/**
 * Returns an Observable that emits the element where
 * the hover overlay mount should be appended, taking account
 * mutations and {@link CodeHost#getHoverOverlayMountLocation}.
 *
 * The caller is responsible for removing the previous mount if it exists.
 *
 * This is useful to mount the hover overlay to a different container than document.body,
 * so that it is affected by the visibility changes of that container.
 *
 * Related issue: https://gitlab.com/gitlab-org/gitlab/issues/193433
 *
 * Example use case on GitLab:
 * 1. User visits https://gitlab.com/gitlab-org/gitaly/-/merge_requests/1575. `div.tab-pane.diffs` doesn't exist yet (it'll be lazy-loaded) -> Mount the  hover overlay is to `document.body`.
 * 2. User visits the 'Changes' tab -> Unmount from `document.body`, mount to `div.tab-pane.diffs`.
 * 3. User visits the 'Overview' tab again -> `div.tab-pane.diffs` is hidden, and as a result so is the hover overlay.
 * 4. User navigates away from the merge request (soft-reload), `div.tab-pane.diffs` is removed -> Mount to `document.body` again.
 */
export function observeHoverOverlayMountLocation(
    getMountLocationSelector: NonNullable<CodeHost['getHoverOverlayMountLocation']>,
    mutations: Observable<MutationRecordLike[]>
): Observable<HTMLElement> {
    return mutations.pipe(
        concatAll(),
        map(({ addedNodes, removedNodes }): HTMLElement | null => {
            // If no selector can be used to determine the mount location
            // return document.body as the mount location.
            const selector = getMountLocationSelector()
            if (selector === null) {
                return document.body
            }
            // If any of the added nodes match the selector, return it
            // as the new mount location.
            for (const addedNode of addedNodes) {
                if (!(addedNode instanceof HTMLElement)) {
                    continue
                }
                const mountLocation = querySelectorOrSelf<HTMLElement>(addedNode, selector)
                if (mountLocation) {
                    return mountLocation
                }
            }
            // If any of the removed nodes match the selector,
            // return document.body as the new mount location.
            for (const removedNode of removedNodes) {
                if (!(removedNode instanceof HTMLElement)) {
                    continue
                }
                if (querySelectorOrSelf<HTMLElement>(removedNode, selector)) {
                    return document.body
                }
            }
            // Neither added nodes nor removed nodes match the selector,
            // don't return a new mount location.
            return null
        }),
        filter(isDefined),
        startWith(document.body),
        distinctUntilChanged()
    )
}

export interface HandleCodeHostOptions extends CodeIntelligenceProps {
    mutations: Observable<MutationRecordLike[]>
    sourcegraphURL: string
    render: typeof reactDOMRender
    minimalUI: boolean
    background: Pick<BackgroundPageApi, 'notifyPrivateRepository' | 'openOptionsPage'>
}

export function handleCodeHost({
    mutations,
    codeHost,
    extensionsController,
    platformContext,
    showGlobalDebug,
    sourcegraphURL,
    telemetryService,
    render,
    minimalUI,
    background,
}: HandleCodeHostOptions): Subscription {
    const history = H.createBrowserHistory()
    const subscriptions = new Subscription()
    const { requestGraphQL } = platformContext

    if (isExtension) {
        // Notify the background page that we are on a private repository
        // This information will be used to alert the user when using Sourcegraph Cloud
        // while on a private repository.
        const isPrivateRepo = !codeHost.getContext || codeHost.getContext().privateRepository
        background.notifyPrivateRepository(isPrivateRepo).catch(error => {
            console.error('Error notifying background page of private repository:', error)
        })
    }

    const addedElements = mutations.pipe(
        concatAll(),
        concatMap(mutation => mutation.addedNodes),
        filter(isInstanceOf(HTMLElement))
    )

    const nativeTooltipsEnabled = codeHost.nativeTooltipResolvers
        ? nativeTooltipsEnabledFromSettings(platformContext.settings)
        : of(false)

    const hoverAlerts: Observable<HoverAlert>[] = []

    if (codeHost.nativeTooltipResolvers) {
        const { subscription, nativeTooltipsAlert } = handleNativeTooltips(
            mutations,
            nativeTooltipsEnabled,
            codeHost,
            platformContext.sourcegraphURL
        )
        subscriptions.add(subscription)
        hoverAlerts.push(nativeTooltipsAlert)
        subscriptions.add(registerNativeTooltipContributions(extensionsController))
    }

    const { hoverifier, subscription } = initCodeIntelligence({
        codeHost,
        extensionsController,
        platformContext,
        telemetryService,
        render,
        hoverAlerts,
        mutations,
    })
    subscriptions.add(hoverifier)
    subscriptions.add(subscription)

    // Inject UI components
    // Render command palette
    if (codeHost.getCommandPaletteMount && !minimalUI) {
        subscriptions.add(
            addedElements.pipe(map(codeHost.getCommandPaletteMount), filter(isDefined)).subscribe(
                renderCommandPalette({
                    extensionsController,
                    history,
                    platformContext,
                    telemetryService,
                    render,
                    ...codeHost.commandPaletteClassProps,
                    notificationClassNames: codeHost.notificationClassNames,
                })
            )
        )
    }

    // Render extension debug menu
    // This renders to document.body, which we can assume is never removed,
    // so we don't need to subscribe to mutations.
    if (showGlobalDebug) {
        const mount = createGlobalDebugMount()
        renderGlobalDebug({ extensionsController, platformContext, history, sourcegraphURL, render })(mount)
    }

    const signInCloses = new Subject<void>()
    const nextSignInClose = signInCloses.next.bind(signInCloses)

    // Try to fetch settings and refresh them when a sign in tab was closed
    subscriptions.add(
        concat([null], signInCloses)
            .pipe(
                switchMap(() =>
                    from(platformContext.refreshSettings()).pipe(
                        catchError(error => {
                            console.error('Refreshing settings failed', error)
                            return []
                        })
                    )
                )
            )
            .subscribe()
    )

    /** The number of code views that were detected on the page (not necessarily initialized) */
    const codeViewCount = new BehaviorSubject<number>(0)

    // Render view on Sourcegraph button
    if (codeHost.getViewContextOnSourcegraphMount && codeHost.getContext) {
        const { getContext, viewOnSourcegraphButtonClassProps } = codeHost

        /** Whether or not the repo exists on the configured Sourcegraph instance. */
        const repoExistsOrErrors = signInCloses.pipe(
            startWith(null),
            switchMap(() => {
                const { rawRepoName, revision } = getContext()
                return resolveRevision({ repoName: rawRepoName, revision, requestGraphQL }).pipe(
                    retryWhenCloneInProgressError(),
                    mapTo(true),
                    catchError(error => {
                        if (isRepoNotFoundErrorLike(error)) {
                            return [false]
                        }
                        return [asError(error)]
                    }),
                    startWith(undefined)
                )
            })
        )
        const onConfigureSourcegraphClick: React.MouseEventHandler<HTMLAnchorElement> = async event => {
            event.preventDefault()
            // If we're here, then `isInPage` should have been checked already,
            // but we double check to be sure and to indicate the intent, for
            // when we might refactor this, that it must only be called in the
            // extension.
            if (isExtension) {
                await background.openOptionsPage()
            }
        }

        subscriptions.add(
            combineLatest([
                repoExistsOrErrors,
                addedElements.pipe(map(codeHost.getViewContextOnSourcegraphMount), filter(isDefined)),
                // Only show sign in button when there is no other code view on the page that is displaying it
                codeViewCount.pipe(
                    map(count => count === 0),
                    distinctUntilChanged()
                ),
            ]).subscribe(([repoExistsOrError, mount, showSignInButton]) => {
                render(
                    <ViewOnSourcegraphButton
                        {...viewOnSourcegraphButtonClassProps}
                        codeHostType={codeHost.type}
                        getContext={getContext}
                        minimalUI={minimalUI}
                        sourcegraphURL={sourcegraphURL}
                        repoExistsOrError={repoExistsOrError}
                        showSignInButton={showSignInButton}
                        // The bound function is constant
                        // eslint-disable-next-line react/jsx-no-bind
                        onSignInClose={nextSignInClose}
                        onConfigureSourcegraphClick={isInPage ? undefined : onConfigureSourcegraphClick}
                    />,
                    mount
                )
            })
        )
    }

    /** A stream of added or removed code views with the resolved file info */
    const codeViews = mutations.pipe(
        trackCodeViews(codeHost),
        tap(codeViewEvent => {
            codeViewCount.next(codeViewCount.value + 1)
            codeViewEvent.subscriptions.add(() => codeViewCount.next(codeViewCount.value - 1))
        }),
        // Delay emitting code views until they are in the viewport, or within 4000 vertical
        // pixels of the viewport's top or bottom edges.
        delayUntilIntersecting({ rootMargin: '4000px 0px' }),
        mergeMap(codeViewEvent =>
            asObservable(() =>
                codeViewEvent.resolveFileInfo(codeViewEvent.element, platformContext.requestGraphQL)
            ).pipe(
                mergeMap(diffOrBlobInfo =>
                    resolveRepoNamesForDiffOrFileInfo(diffOrBlobInfo, platformContext.requestGraphQL)
                ),
                mergeMap(diffOrBlobInfo =>
                    fetchFileContentForDiffOrFileInfo(diffOrBlobInfo, platformContext.requestGraphQL).pipe(
                        map(diffOrBlobInfo => ({
                            diffOrBlobInfo,
                            ...codeViewEvent,
                        }))
                    )
                ),
                catchError(error => {
                    // Ignore PrivateRepoPublicSourcegraph errors (don't initialize those code views)
                    if (isPrivateRepoPublicSourcegraphComErrorLike(error)) {
                        return EMPTY
                    }
                    throw error
                }),
                tap({
                    error: error => {
                        if (codeViewEvent.getToolbarMount) {
                            const mount = codeViewEvent.getToolbarMount(codeViewEvent.element)
                            render(
                                <CodeViewToolbar
                                    {...codeHost.codeViewToolbarClassProps}
                                    fileInfoOrError={error}
                                    sourcegraphURL={sourcegraphURL}
                                    telemetryService={telemetryService}
                                    platformContext={platformContext}
                                    extensionsController={extensionsController}
                                    buttonProps={codeViewEvent.toolbarButtonProps}
                                    // The bound function is constant
                                    // eslint-disable-next-line react/jsx-no-bind
                                    onSignInClose={nextSignInClose}
                                    location={H.createLocation(window.location)}
                                />,
                                mount
                            )
                        }
                    },
                }),
                // Retry auth errors after the user closed a sign-in tab
                retryWhen(errors =>
                    errors.pipe(
                        // Don't swallow non-auth errors
                        tap(error => {
                            if (!isHTTPAuthError(error)) {
                                throw error
                            }
                        }),
                        switchMap(() => signInCloses)
                    )
                ),
                catchError(error => {
                    // Log errors but don't break the handling of other code views
                    console.error('Could not resolve file info for code view', error)
                    return []
                })
            )
        ),
        observeOn(asyncScheduler)
    )

    /** Keep a count of existing references to workspace roots. */
    const rootReferenceCounter = new ReferenceCounter()

    /**
     * Adds root referenced by a code editor to the workspace.
     */
    const addRootReference = (uri: string, inputRevision: string | undefined): void => {
        if (rootReferenceCounter.increment(uri)) {
            extensionsController.extHostAPI
                .then(extensionHostAPI => extensionHostAPI.addWorkspaceRoot({ uri, inputRevision }))
                .catch(console.error)
        }
    }

    /**
     * Deletes a reference to a workspace root from a code editor.
     */
    const deleteRootReference = (uri: string): void => {
        if (rootReferenceCounter.decrement(uri)) {
            extensionsController.extHostAPI
                .then(extensionHostApi => extensionHostApi.removeWorkspaceRoot(uri))
                .catch(console.error)
        }
    }

    subscriptions.add(
        codeViews.subscribe(codeViewEvent => {
            console.log('Code view added')
            codeViewEvent.subscriptions.add(() => console.log('Code view removed'))

            const { element, diffOrBlobInfo, getPositionAdjuster, getToolbarMount, toolbarButtonProps } = codeViewEvent

            const initializeModelAndViewerForFileInfo = (
                fileInfo: FileInfoWithContent & FileInfoWithRepoName
            ): CodeEditorWithPartialModel => {
                const uri = toURIWithPath(fileInfo)

                // Add model
                const languageId = getModeFromPath(fileInfo.filePath)
                const model = { uri, languageId, text: fileInfo.content }
                // Only add the model if it doesn't exist
                // (there may be several code views on the page pointing to the same model)
                if (!extensionsController.services.model.hasModel(uri)) {
                    extensionsController.services.model.addModel(model)
                }

                // Add viewer
                const editorData: CodeEditorData = {
                    type: 'CodeEditor' as const,
                    resource: uri,
                    selections: codeViewEvent.getSelections ? codeViewEvent.getSelections(codeViewEvent.element) : [],
                    isActive: true,
                }
                const editorId = extensionsController.services.viewer.addViewer(editorData)

                // Add root ref
                const rootURI = toRootURI(fileInfo)
                addRootReference(rootURI, fileInfo.revision)

                // Subscribe for removal
                codeViewEvent.subscriptions.add(() => {
                    deleteRootReference(rootURI)
                    extensionsController.services.viewer.removeViewer(editorId)
                })

                return {
                    ...editorData,
                    ...editorId,
                    model,
                }
            }

            const initializeModelAndViewerForDiffOrFileInfo = (
                diffOrFileInfo: DiffOrBlobInfo<FileInfoWithContent>
            ): CodeEditorWithPartialModel => {
                if ('blob' in diffOrFileInfo) {
                    return initializeModelAndViewerForFileInfo(diffOrFileInfo.blob)
                }
                if (diffOrFileInfo.head && diffOrFileInfo.base) {
                    // For diffs, both editors are created (for head and base)
                    // but only one of them is returned and later passed into
                    // the `scope` of the CodeViewToolbar component.
                    const editor = initializeModelAndViewerForFileInfo(diffOrFileInfo.head)
                    initializeModelAndViewerForFileInfo(diffOrFileInfo.base)
                    return editor
                }
                if (diffOrFileInfo.base) {
                    return initializeModelAndViewerForFileInfo(diffOrFileInfo.base)
                }
                return initializeModelAndViewerForFileInfo(diffOrFileInfo.head)
            }

            const codeEditorWithPartialModel = initializeModelAndViewerForDiffOrFileInfo(diffOrBlobInfo)

            const domFunctions = {
                ...codeViewEvent.dom,
                // If any parent element has the sourcegraph-extension-element
                // class then that element does not have any code. We
                // must check for "any parent element" because extensions
                // create their DOM changes before the blob is tokenized
                // into multiple elements.
                getCodeElementFromTarget: (target: HTMLElement): HTMLElement | null =>
                    target.closest('.sourcegraph-extension-element') !== null
                        ? null
                        : codeViewEvent.dom.getCodeElementFromTarget(target),
            }

            const applyDecorationsForFileInfo = (fileInfo: FileInfoWithContent, diffPart?: DiffPart): void => {
                let decorationsByLine: DecorationMapByLine = new Map()
                const update = (decorations?: TextDocumentDecoration[] | null): void => {
                    try {
                        decorationsByLine = applyDecorations(
                            domFunctions,
                            element,
                            decorations || [],
                            decorationsByLine,
                            diffPart
                        )
                    } catch (error) {
                        console.error('Could not apply decorations to code view', codeViewEvent.element, error)
                    }
                }
                codeViewEvent.subscriptions.add(
                    extensionsController.services.textDocumentDecoration
                        .getDecorations(toTextDocumentIdentifier(fileInfo))
                        // Make sure extensions get cleaned up un unsubscription
                        .pipe(finalize(update))
                        // The nested subscribe cannot be replaced with a switchMap()
                        // We manage the subscription correctly.
                        // eslint-disable-next-line rxjs/no-nested-subscribe
                        .subscribe(update)
                )
            }

            // Apply decorations coming from extensions
            if (!minimalUI) {
                if ('blob' in diffOrBlobInfo) {
                    applyDecorationsForFileInfo(diffOrBlobInfo.blob)
                } else {
                    if (diffOrBlobInfo.head) {
                        applyDecorationsForFileInfo(diffOrBlobInfo.head, 'head')
                    }
                    if (diffOrBlobInfo.base) {
                        applyDecorationsForFileInfo(diffOrBlobInfo.base, 'base')
                    }
                }
            }

            // Add hover code intelligence
            const resolveContext: ContextResolver<RepoSpec & RevisionSpec & FileSpec & ResolvedRevisionSpec> = ({
                part,
            }) => {
                if ('blob' in diffOrBlobInfo) {
                    return defaultRevisionToCommitID(diffOrBlobInfo.blob)
                }
                if (diffOrBlobInfo.head && part === 'head') {
                    return defaultRevisionToCommitID(diffOrBlobInfo.head)
                }
                if (diffOrBlobInfo.base && part === 'base') {
                    return defaultRevisionToCommitID(diffOrBlobInfo.base)
                }
                throw new Error(`Could not resolve context for diff part ${JSON.stringify(part)}`)
            }

            const adjustPosition = getPositionAdjuster?.(platformContext.requestGraphQL)
            let hoverSubscription = new Subscription()
            codeViewEvent.subscriptions.add(
                // eslint-disable-next-line rxjs/no-nested-subscribe
                nativeTooltipsEnabled.subscribe(useNativeTooltips => {
                    hoverSubscription.unsubscribe()
                    if (!useNativeTooltips) {
                        hoverSubscription = hoverifier.hoverify({
                            dom: domFunctions,
                            positionEvents: of(element).pipe(
                                findPositionsFromEvents({
                                    domFunctions,
                                    tokenize: codeHost.codeViewsRequireTokenization !== false,
                                })
                            ),
                            resolveContext,
                            adjustPosition,
                            scrollBoundaries: codeViewEvent.getScrollBoundaries
                                ? codeViewEvent.getScrollBoundaries(codeViewEvent.element)
                                : [],
                        })
                    }
                })
            )
            codeViewEvent.subscriptions.add(hoverSubscription)

            element.classList.add('sg-mounted')

            // Render toolbar
            if (getToolbarMount && !minimalUI) {
                const mount = getToolbarMount(element)
                render(
                    <CodeViewToolbar
                        {...codeHost.codeViewToolbarClassProps}
                        fileInfoOrError={diffOrBlobInfo}
                        sourcegraphURL={sourcegraphURL}
                        telemetryService={telemetryService}
                        platformContext={platformContext}
                        extensionsController={extensionsController}
                        buttonProps={toolbarButtonProps}
                        location={H.createLocation(window.location)}
                        scope={codeEditorWithPartialModel}
                        // The bound function is constant
                        // eslint-disable-next-line react/jsx-no-bind
                        onSignInClose={nextSignInClose}
                    />,
                    mount
                )
            }
        })
    )

    // Show link previews on content views (feature-flagged).
    subscriptions.add(
        handleContentViews(
            from(featureFlags.isEnabled('experimentalLinkPreviews')).pipe(
                switchMap(enabled => (enabled ? mutations : []))
            ),
            { extensionsController },
            codeHost
        )
    )

    // Show completions in text fields (feature-flagged).
    subscriptions.add(
        handleTextFields(
            from(featureFlags.isEnabled('experimentalTextFieldCompletion')).pipe(
                switchMap(enabled => (enabled ? mutations : []))
            ),
            { extensionsController },
            codeHost
        )
    )

    return subscriptions
}

const SHOW_DEBUG = (): boolean => localStorage.getItem('debug') !== null

const CODE_HOSTS: CodeHost[] = [bitbucketServerCodeHost, githubCodeHost, gitlabCodeHost, phabricatorCodeHost]
export const determineCodeHost = (): CodeHost | undefined => CODE_HOSTS.find(codeHost => codeHost.check())

export function injectCodeIntelligenceToCodeHost(
    mutations: Observable<MutationRecordLike[]>,
    codeHost: CodeHost,
    { sourcegraphURL, assetsURL }: SourcegraphIntegrationURLs,
    isExtension: boolean,
    showGlobalDebug = SHOW_DEBUG()
): Subscription {
    const subscriptions = new Subscription()
    const { platformContext, extensionsController } = initializeExtensions(
        codeHost,
        { sourcegraphURL, assetsURL },
        isExtension
    )
    const { requestGraphQL } = platformContext
    subscriptions.add(extensionsController)

    const overrideSendTelemetry = observeSourcegraphURL(isExtension).pipe(
        map(sourcegraphUrl => shouldOverrideSendTelemetry(isFirefox(), isExtension, sourcegraphUrl))
    )

    const observeSendTelemetry = combineLatest([overrideSendTelemetry, observeOptionFlag('sendTelemetry')]).pipe(
        map(([override, sendTelemetry]) => {
            if (override) {
                return true
            }
            return sendTelemetry
        })
    )

    const innerTelemetryService = new EventLogger(isExtension, requestGraphQL)
    const telemetryService = new ConditionalTelemetryService(innerTelemetryService, observeSendTelemetry)
    subscriptions.add(telemetryService)

    let codeHostSubscription: Subscription
    // In the browser extension, observe whether the `disableExtension` storage flag is set.
    // In the native integration, this flag does not exist.
    const extensionDisabled = isExtension ? observeStorageKey('sync', 'disableExtension') : of(false)

    // RFC 68: hide some UI features in the GitLab native integration.
    // This can be overridden using the `sourcegraphMinimalUI` local storage flag.
    const minimalUIStorageFlag = localStorage.getItem('sourcegraphMinimalUI')
    const minimalUI =
        minimalUIStorageFlag !== null ? minimalUIStorageFlag === 'true' : codeHost.type === 'gitlab' && !isExtension
    subscriptions.add(
        extensionDisabled.subscribe(disableExtension => {
            if (disableExtension) {
                // We don't need to unsubscribe if the extension starts with disabled state.
                if (codeHostSubscription) {
                    codeHostSubscription.unsubscribe()
                }
                console.log('Browser extension is disabled')
            } else {
                codeHostSubscription = handleCodeHost({
                    mutations,
                    codeHost,
                    extensionsController,
                    platformContext,
                    showGlobalDebug,
                    sourcegraphURL,
                    telemetryService,
                    render: reactDOMRender,
                    minimalUI,
                    background,
                })
                subscriptions.add(codeHostSubscription)
                console.log(`${isExtension ? 'Browser extension' : 'Native integration'} is enabled`)
            }
        })
    )
    return subscriptions
}
