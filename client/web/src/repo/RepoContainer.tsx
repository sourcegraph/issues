import AlertCircleIcon from 'mdi-react/AlertCircleIcon'
import MapSearchIcon from 'mdi-react/MapSearchIcon'
import React, { useMemo, useState, useEffect, useCallback, useRef } from 'react'
import { escapeRegExp, uniqueId } from 'lodash'
import { Route, RouteComponentProps, Switch } from 'react-router'
import { Observable, NEVER, ObservableInput, of } from 'rxjs'
import { catchError, map, startWith } from 'rxjs/operators'
import { redirectToExternalHost } from '.'
import {
    isRepoNotFoundErrorLike,
    isRepoSeeOtherErrorLike,
    isCloneInProgressErrorLike,
} from '../../../shared/src/backend/errors'
import { ActivationProps } from '../../../shared/src/components/activation/Activation'
import { ExtensionsControllerProps } from '../../../shared/src/extensions/controller'
import * as GQL from '../../../shared/src/graphql/schema'
import { PlatformContextProps } from '../../../shared/src/platform/context'
import { SettingsCascadeProps } from '../../../shared/src/settings/settings'
import { ErrorLike, isErrorLike, asError } from '../../../shared/src/util/errors'
import { encodeURIComponentExceptSlashes, makeRepoURI } from '../../../shared/src/util/url'
import { ErrorBoundary } from '../components/ErrorBoundary'
import { HeroPage } from '../components/HeroPage'
import {
    searchQueryForRepoRevision,
    PatternTypeProps,
    CaseSensitivityProps,
    InteractiveSearchProps,
    repoFilterForRepoRevision,
    CopyQueryButtonProps,
    quoteIfNeeded,
} from '../search'
import { RouteDescriptor } from '../util/contributions'
import { parseBrowserRepoURL } from '../util/url'
import { GoToCodeHostAction } from './actions/GoToCodeHostAction'
import { fetchFileExternalLinks, fetchRepository, resolveRevision } from './backend'
import { RepoHeader, RepoHeaderActionButton, RepoHeaderContributionsLifecycleProps } from './RepoHeader'
import { RepoRevisionContainer, RepoRevisionContainerRoute } from './RepoRevisionContainer'
import { RepositoryNotFoundPage } from './RepositoryNotFoundPage'
import { ThemeProps } from '../../../shared/src/theme'
import { RepoSettingsAreaRoute } from './settings/RepoSettingsArea'
import { RepoSettingsSideBarGroup } from './settings/RepoSettingsSidebar'
import { ErrorMessage } from '../components/alerts'
import { QueryState } from '../search/helpers'
import { FiltersToTypeAndValue, FilterType } from '../../../shared/src/search/interactive/util'
import * as H from 'history'
import { VersionContextProps } from '../../../shared/src/search/util'
import { BreadcrumbSetters, BreadcrumbsProps } from '../components/Breadcrumbs'
import { useObservable, useEventObservable } from '../../../shared/src/util/useObservable'
import { repeatUntil } from '../../../shared/src/util/rxjs/repeatUntil'
import { RepoHeaderContributionPortal } from './RepoHeaderContributionPortal'
import { Link } from '../../../shared/src/components/Link'
import { UncontrolledPopover } from 'reactstrap'
import MenuDownIcon from 'mdi-react/MenuDownIcon'
import { RepositoriesPopover } from './RepositoriesPopover'
import { displayRepoName, splitPath } from '../../../shared/src/components/RepoFileLink'
import { AuthenticatedUser } from '../auth'
import { TelemetryProps } from '../../../shared/src/telemetry/telemetryService'
import { ExternalLinkFields } from '../graphql-operations'
import { browserExtensionInstalled } from '../tracking/analyticsUtils'
import { InstallBrowserExtensionAlert } from './actions/InstallBrowserExtensionAlert'
import { IS_CHROME } from '../marketing/util'
import { useLocalStorage } from '../util/useLocalStorage'
import { Settings } from '../schema/settings.schema'

/**
 * Props passed to sub-routes of {@link RepoContainer}.
 */
export interface RepoContainerContext
    extends RepoHeaderContributionsLifecycleProps,
        SettingsCascadeProps,
        ExtensionsControllerProps,
        PlatformContextProps,
        ThemeProps,
        HoverThresholdProps,
        TelemetryProps,
        ActivationProps,
        PatternTypeProps,
        CaseSensitivityProps,
        CopyQueryButtonProps,
        VersionContextProps,
        BreadcrumbSetters {
    repo: GQL.IRepository
    authenticatedUser: AuthenticatedUser | null
    repoSettingsAreaRoutes: readonly RepoSettingsAreaRoute[]
    repoSettingsSidebarGroups: readonly RepoSettingsSideBarGroup[]

    /** The URL route match for {@link RepoContainer}. */
    routePrefix: string

    onDidUpdateRepository: (update: Partial<GQL.IRepository>) => void
    onDidUpdateExternalLinks: (externalLinks: ExternalLinkFields[] | undefined) => void

    globbing: boolean
}

/** A sub-route of {@link RepoContainer}. */
export interface RepoContainerRoute extends RouteDescriptor<RepoContainerContext> {}

const RepoPageNotFound: React.FunctionComponent = () => (
    <HeroPage icon={MapSearchIcon} title="404: Not Found" subtitle="The repository page was not found." />
)

interface RepoContainerProps
    extends RouteComponentProps<{ repoRevAndRest: string }>,
        SettingsCascadeProps<Settings>,
        PlatformContextProps,
        TelemetryProps,
        ExtensionsControllerProps,
        ActivationProps,
        ThemeProps,
        ExtensionAlertProps,
        PatternTypeProps,
        CaseSensitivityProps,
        InteractiveSearchProps,
        CopyQueryButtonProps,
        VersionContextProps,
        BreadcrumbSetters,
        BreadcrumbsProps {
    repoContainerRoutes: readonly RepoContainerRoute[]
    repoRevisionContainerRoutes: readonly RepoRevisionContainerRoute[]
    repoHeaderActionButtons: readonly RepoHeaderActionButton[]
    repoSettingsAreaRoutes: readonly RepoSettingsAreaRoute[]
    repoSettingsSidebarGroups: readonly RepoSettingsSideBarGroup[]
    authenticatedUser: AuthenticatedUser | null
    onNavbarQueryChange: (state: QueryState) => void
    history: H.History
    globbing: boolean
}

export const HOVER_COUNT_KEY = 'hover-count'
const HAS_DISMISSED_ALERT_KEY = 'has-dismissed-extension-alert'

export const HOVER_THRESHOLD = 5

export interface HoverThresholdProps {
    /**
     * Called when a hover with content is shown.
     */
    onHoverShown?: () => void
}

export interface ExtensionAlertProps {
    onExtensionAlertDismissed: () => void
}

/**
 * Renders a horizontal bar and content for a repository page.
 */
export const RepoContainer: React.FunctionComponent<RepoContainerProps> = props => {
    const { repoName, revision, rawRevision, filePath, commitRange, position, range } = parseBrowserRepoURL(
        location.pathname + location.search + location.hash
    )

    // Fetch repository upon mounting the component.
    const initialRepoOrError = useObservable(
        useMemo(
            () =>
                fetchRepository({ repoName }).pipe(
                    catchError(
                        (error): ObservableInput<ErrorLike> => {
                            const redirect = isRepoSeeOtherErrorLike(error)
                            if (redirect) {
                                redirectToExternalHost(redirect)
                                return NEVER
                            }
                            return of(asError(error))
                        }
                    )
                ),
            [repoName]
        )
    )

    // Allow partial updates of the repository from components further down the tree.
    const [nextRepoOrErrorUpdate, repoOrError] = useEventObservable(
        useCallback(
            (repoOrErrorUpdates: Observable<Partial<GQL.IRepository>>) =>
                repoOrErrorUpdates.pipe(
                    map((update): GQL.IRepository | ErrorLike | undefined =>
                        isErrorLike(initialRepoOrError) || initialRepoOrError === undefined
                            ? initialRepoOrError
                            : { ...initialRepoOrError, ...update }
                    ),
                    startWith(initialRepoOrError)
                ),
            [initialRepoOrError]
        )
    )

    const resolvedRevisionOrError = useObservable(
        React.useMemo(
            () =>
                resolveRevision({ repoName, revision }).pipe(
                    catchError(error => {
                        if (isCloneInProgressErrorLike(error)) {
                            return of<ErrorLike>(asError(error))
                        }
                        throw error
                    }),
                    repeatUntil(value => !isCloneInProgressErrorLike(value), { delay: 1000 }),
                    catchError(error => of<ErrorLike>(asError(error)))
                ),
            [repoName, revision]
        )
    )

    // The external links to show in the repository header, if any.
    const [externalLinks, setExternalLinks] = useState<ExternalLinkFields[] | undefined>()

    // The lifecycle props for repo header contributions.
    const [repoHeaderContributionsLifecycleProps, setRepoHeaderContributionsLifecycleProps] = useState<
        RepoHeaderContributionsLifecycleProps
    >()

    const repositoryBreadcrumbSetters = props.useBreadcrumb(
        useMemo(
            () => ({
                key: 'repositories',
                element: <>Repositories</>,
            }),
            []
        )
    )

    const childBreadcrumbSetters = repositoryBreadcrumbSetters.useBreadcrumb(
        useMemo(() => {
            if (isErrorLike(repoOrError) || !repoOrError) {
                return
            }

            const [repoDirectory, repoBase] = splitPath(displayRepoName(repoOrError.name))

            return {
                key: 'repository',
                element: (
                    <>
                        <Link
                            to={
                                resolvedRevisionOrError && !isErrorLike(resolvedRevisionOrError)
                                    ? resolvedRevisionOrError.rootTreeURL
                                    : repoOrError.url
                            }
                            className="repo-header__repo"
                        >
                            {repoDirectory ? `${repoDirectory}/` : ''}
                            <span className="font-weight-semibold">{repoBase}</span>
                        </Link>
                        <button
                            type="button"
                            id="repo-popover"
                            className="btn btn-icon px-0"
                            aria-label="Change repository"
                        >
                            <MenuDownIcon className="icon-inline" />
                        </button>
                        <UncontrolledPopover
                            placement="bottom-start"
                            target="repo-popover"
                            trigger="legacy"
                            hideArrow={true}
                            popperClassName="border-0"
                        >
                            <RepositoriesPopover
                                currentRepo={repoOrError.id}
                                history={props.history}
                                location={props.location}
                            />
                        </UncontrolledPopover>
                    </>
                ),
            }
        }, [repoOrError, resolvedRevisionOrError, props.history, props.location])
    )

    // Update the workspace roots via the extension host API to reflect the current repo / resolved revision
    useEffect(() => {
        const workspaceRootURI =
            resolvedRevisionOrError && !isErrorLike(resolvedRevisionOrError)
                ? makeRepoURI({ repoName, revision: resolvedRevisionOrError.commitID })
                : undefined
        if (workspaceRootURI) {
            props.extensionsController.extHostAPI
                .then(extensionHostAPI =>
                    extensionHostAPI.addWorkspaceRoot({
                        uri: workspaceRootURI,
                    })
                )
                .catch(console.error)
        }

        // Clear the Sourcegraph extensions workspace's roots when navigating away.
        return () => {
            if (workspaceRootURI) {
                props.extensionsController.extHostAPI
                    .then(extensionHostAPI => extensionHostAPI.removeWorkspaceRoot(workspaceRootURI))
                    .catch(console.error)
            }
        }
    }, [props.extensionsController.extHostAPI, repoName, resolvedRevisionOrError, revision])

    // Update the navbar query to reflect the current repo / revision
    const { splitSearchModes, interactiveSearchMode, globbing, onFiltersInQueryChange, onNavbarQueryChange } = props
    useEffect(() => {
        if (splitSearchModes && interactiveSearchMode) {
            const filters: FiltersToTypeAndValue = {
                [uniqueId('repo')]: {
                    type: FilterType.repo,
                    value: repoFilterForRepoRevision(repoName, globbing, revision),
                    editable: false,
                },
            }
            if (filePath) {
                filters[uniqueId('file')] = {
                    type: FilterType.file,
                    value: globbing ? filePath : `^${escapeRegExp(filePath)}`,
                    editable: false,
                }
            }
            onFiltersInQueryChange(filters)
            onNavbarQueryChange({
                query: '',
                cursorPosition: 0,
            })
        } else {
            let query = searchQueryForRepoRevision(repoName, globbing, revision)
            if (filePath) {
                query = `${query.trimEnd()} file:${quoteIfNeeded(globbing ? filePath : '^' + escapeRegExp(filePath))}`
            }
            onNavbarQueryChange({
                query,
                cursorPosition: query.length,
            })
        }
    }, [
        revision,
        filePath,
        repoName,
        onFiltersInQueryChange,
        onNavbarQueryChange,
        splitSearchModes,
        globbing,
        interactiveSearchMode,
    ])

    const isBrowserExtensionInstalled = useObservable(browserExtensionInstalled)
    const codeHostIntegrationMessaging =
        (!isErrorLike(props.settingsCascade.final) &&
            props.settingsCascade.final?.['alerts.codeHostIntegrationMessaging']) ||
        'browser-extension'

    // Browser extension discoverability features (alert, popover for `GoToCodeHostAction)
    const [hasDismissedExtensionAlert, setHasDismissedExtensionAlert] = useLocalStorage(HAS_DISMISSED_ALERT_KEY, false)
    const [hasDismissedPopover, setHasDismissedPopover] = useState(false)
    const [hoverCount, setHoverCount] = useLocalStorage(HOVER_COUNT_KEY, 0)
    const canShowPopover =
        !hasDismissedPopover &&
        isBrowserExtensionInstalled === false &&
        codeHostIntegrationMessaging === 'browser-extension' &&
        hoverCount >= HOVER_THRESHOLD
    const showExtensionAlert = useMemo(
        () => isBrowserExtensionInstalled === false && !hasDismissedExtensionAlert && hoverCount >= HOVER_THRESHOLD,
        // Intentionally use useMemo() here without a dependency on hoverCount to only show the alert on the next reload,
        // to not cause an annoying layout shift from displaying the alert.
        // eslint-disable-next-line react-hooks/exhaustive-deps
        [hasDismissedExtensionAlert, isBrowserExtensionInstalled]
    )

    const { onExtensionAlertDismissed } = props

    // Increment hovers that the user has seen. Enable browser extension discoverability
    // features after hover count threshold is reached (e.g. alerts, popovers)
    // Store hover count in ref to avoid circular dependency
    // hoverCount -> onHoverShown -> WebHoverOverlay (onHoverShown in useEffect deps) -> onHoverShown()
    const hoverCountReference = useRef(hoverCount)
    hoverCountReference.current = hoverCount
    const onHoverShown = useCallback(() => {
        const count = hoverCountReference.current + 1
        if (count > HOVER_THRESHOLD) {
            // No need to keep updating localStorage
            return
        }
        setHoverCount(count)
    }, [setHoverCount])

    const onPopoverDismissed = useCallback(() => {
        setHasDismissedPopover(true)
    }, [])

    const onAlertDismissed = useCallback(() => {
        onExtensionAlertDismissed()
        setHasDismissedExtensionAlert(true)
    }, [onExtensionAlertDismissed, setHasDismissedExtensionAlert])

    if (!repoOrError) {
        // Render nothing while loading
        return null
    }

    const viewerCanAdminister = !!props.authenticatedUser && props.authenticatedUser.siteAdmin

    if (isErrorLike(repoOrError)) {
        // Display error page
        if (isRepoNotFoundErrorLike(repoOrError)) {
            return <RepositoryNotFoundPage repo={repoName} viewerCanAdminister={viewerCanAdminister} />
        }
        return (
            <HeroPage
                icon={AlertCircleIcon}
                title="Error"
                subtitle={<ErrorMessage error={repoOrError} history={props.history} />}
            />
        )
    }

    const repoMatchURL = '/' + encodeURIComponentExceptSlashes(repoName)

    const context: RepoContainerContext = {
        ...props,
        ...repoHeaderContributionsLifecycleProps,
        ...childBreadcrumbSetters,
        onHoverShown,
        repo: repoOrError,
        routePrefix: repoMatchURL,
        onDidUpdateExternalLinks: setExternalLinks,
        onDidUpdateRepository: nextRepoOrErrorUpdate,
    }

    return (
        <div className="repo-container test-repo-container w-100 d-flex flex-column">
            {showExtensionAlert && (
                <InstallBrowserExtensionAlert
                    isChrome={IS_CHROME}
                    onAlertDismissed={onAlertDismissed}
                    externalURLs={repoOrError.externalURLs}
                    codeHostIntegrationMessaging={codeHostIntegrationMessaging}
                />
            )}
            <RepoHeader
                {...props}
                actionButtons={props.repoHeaderActionButtons}
                revision={revision}
                repo={repoOrError}
                resolvedRev={resolvedRevisionOrError}
                onLifecyclePropsChange={setRepoHeaderContributionsLifecycleProps}
                isAlertDisplayed={showExtensionAlert}
            />
            <RepoHeaderContributionPortal
                position="right"
                priority={2}
                {...repoHeaderContributionsLifecycleProps}
                element={
                    <GoToCodeHostAction
                        key="go-to-code-host"
                        repo={repoOrError}
                        // We need a revision to generate code host URLs, if revision isn't available, we use the default branch or HEAD.
                        revision={rawRevision || repoOrError.defaultBranch?.displayName || 'HEAD'}
                        filePath={filePath}
                        commitRange={commitRange}
                        position={position}
                        range={range}
                        externalLinks={externalLinks}
                        fetchFileExternalLinks={fetchFileExternalLinks}
                        canShowPopover={canShowPopover}
                        onPopoverDismissed={onPopoverDismissed}
                    />
                }
            />
            <ErrorBoundary location={props.location}>
                <Switch>
                    {/* eslint-disable react/jsx-no-bind */}
                    {[
                        '',
                        ...(rawRevision ? [`@${rawRevision}`] : []), // must exactly match how the revision was encoded in the URL
                        '/-/blob',
                        '/-/tree',
                        '/-/commits',
                    ].map(routePath => (
                        <Route
                            path={`${repoMatchURL}${routePath}`}
                            key="hardcoded-key" // see https://github.com/ReactTraining/react-router/issues/4578#issuecomment-334489490
                            exact={routePath === ''}
                            render={routeComponentProps => (
                                <RepoRevisionContainer
                                    {...routeComponentProps}
                                    {...context}
                                    {...childBreadcrumbSetters}
                                    routes={props.repoRevisionContainerRoutes}
                                    revision={revision || ''}
                                    resolvedRevisionOrError={resolvedRevisionOrError}
                                    // must exactly match how the revision was encoded in the URL
                                    routePrefix={`${repoMatchURL}${rawRevision ? `@${rawRevision}` : ''}`}
                                />
                            )}
                        />
                    ))}
                    {props.repoContainerRoutes.map(
                        ({ path, render, exact, condition = () => true }) =>
                            condition(context) && (
                                <Route
                                    path={context.routePrefix + path}
                                    key="hardcoded-key" // see https://github.com/ReactTraining/react-router/issues/4578#issuecomment-334489490
                                    exact={exact}
                                    // RouteProps.render is an exception
                                    render={routeComponentProps => render({ ...context, ...routeComponentProps })}
                                />
                            )
                    )}
                    <Route key="hardcoded-key" component={RepoPageNotFound} />
                    {/* eslint-enable react/jsx-no-bind */}
                </Switch>
            </ErrorBoundary>
        </div>
    )
}
