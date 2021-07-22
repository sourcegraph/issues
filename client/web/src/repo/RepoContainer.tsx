import classNames from 'classnames'
import * as H from 'history'
import { escapeRegExp } from 'lodash'
import AlertCircleIcon from 'mdi-react/AlertCircleIcon'
import ChevronDownIcon from 'mdi-react/ChevronDownIcon'
import MapSearchIcon from 'mdi-react/MapSearchIcon'
import MenuDownIcon from 'mdi-react/MenuDownIcon'
import SourceRepositoryIcon from 'mdi-react/SourceRepositoryIcon'
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Route, RouteComponentProps, Switch } from 'react-router'
import { UncontrolledPopover } from 'reactstrap'
import { NEVER, ObservableInput, of } from 'rxjs'
import { catchError, switchMap } from 'rxjs/operators'

import {
    isCloneInProgressErrorLike,
    isRepoNotFoundErrorLike,
    isRepoSeeOtherErrorLike,
} from '@sourcegraph/shared/src/backend/errors'
import { ActivationProps } from '@sourcegraph/shared/src/components/activation/Activation'
import { Link } from '@sourcegraph/shared/src/components/Link'
import { displayRepoName } from '@sourcegraph/shared/src/components/RepoFileLink'
import { ExtensionsControllerProps } from '@sourcegraph/shared/src/extensions/controller'
import { PlatformContextProps } from '@sourcegraph/shared/src/platform/context'
import { escapeSpaces } from '@sourcegraph/shared/src/search/query/filters'
import { VersionContextProps } from '@sourcegraph/shared/src/search/util'
import { SettingsCascadeProps } from '@sourcegraph/shared/src/settings/settings'
import { TelemetryProps } from '@sourcegraph/shared/src/telemetry/telemetryService'
import { ThemeProps } from '@sourcegraph/shared/src/theme'
import { isFirefox } from '@sourcegraph/shared/src/util/browserDetection'
import { asError, ErrorLike, isErrorLike } from '@sourcegraph/shared/src/util/errors'
import { repeatUntil } from '@sourcegraph/shared/src/util/rxjs/repeatUntil'
import { encodeURIPathComponent, makeRepoURI } from '@sourcegraph/shared/src/util/url'
import { useLocalStorage } from '@sourcegraph/shared/src/util/useLocalStorage'
import { useObservable } from '@sourcegraph/shared/src/util/useObservable'
import { useRedesignToggle } from '@sourcegraph/shared/src/util/useRedesignToggle'

import { AuthenticatedUser } from '../auth'
import { BatchChangesProps } from '../batches/batches'
import { ErrorMessage } from '../components/alerts'
import { BreadcrumbSetters, BreadcrumbsProps } from '../components/Breadcrumbs'
import { ErrorBoundary } from '../components/ErrorBoundary'
import { FuzzyFinder } from '../components/fuzzyFinder/FuzzyFinder'
import { HeroPage } from '../components/HeroPage'
import { ActionItemsBarProps, useWebActionItems } from '../extensions/components/ActionItemsBar'
import { ExternalLinkFields, RepositoryFields } from '../graphql-operations'
import { IS_CHROME } from '../marketing/util'
import { Settings } from '../schema/settings.schema'
import { CaseSensitivityProps, PatternTypeProps, SearchContextProps, searchQueryForRepoRevision } from '../search'
import { QueryState } from '../search/helpers'
import { browserExtensionInstalled } from '../tracking/analyticsUtils'
import { RouteDescriptor } from '../util/contributions'
import { parseBrowserRepoURL } from '../util/url'

import { GoToCodeHostAction } from './actions/GoToCodeHostAction'
import { InstallBrowserExtensionAlert, isFirefoxCampaignActive } from './actions/InstallBrowserExtensionAlert'
import { fetchFileExternalLinks, fetchRepository, resolveRevision } from './backend'
import { RepoHeader, RepoHeaderActionButton, RepoHeaderContributionsLifecycleProps } from './RepoHeader'
import { RepoHeaderContributionPortal } from './RepoHeaderContributionPortal'
import { RepoRevisionContainer, RepoRevisionContainerRoute } from './RepoRevisionContainer'
import { RepositoriesPopover } from './RepositoriesPopover'
import { RepositoryNotFoundPage } from './RepositoryNotFoundPage'
import { RepoSettingsAreaRoute } from './settings/RepoSettingsArea'
import { RepoSettingsSideBarGroup } from './settings/RepoSettingsSidebar'

import { redirectToExternalHost } from '.'

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
        VersionContextProps,
        Pick<SearchContextProps, 'selectedSearchContextSpec'>,
        BreadcrumbSetters,
        BatchChangesProps,
        ActionItemsBarProps {
    repo: RepositoryFields
    authenticatedUser: AuthenticatedUser | null
    repoSettingsAreaRoutes: readonly RepoSettingsAreaRoute[]
    repoSettingsSidebarGroups: readonly RepoSettingsSideBarGroup[]

    /** The URL route match for {@link RepoContainer}. */
    routePrefix: string

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
        VersionContextProps,
        Pick<SearchContextProps, 'selectedSearchContextSpec'>,
        BreadcrumbSetters,
        BatchChangesProps,
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
const HAS_DISMISSED_FIREFOX_ALERT_KEY = 'has-dismissed-firefox-addon-alert'

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
    const [isRedesignEnabled] = useRedesignToggle()
    const { repoName, revision, rawRevision, filePath, commitRange, position, range } = parseBrowserRepoURL(
        location.pathname + location.search + location.hash
    )

    // Fetch repository upon mounting the component.
    const repoOrError = useObservable(
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

    const resolvedRevisionOrError = useObservable(
        useMemo(
            () =>
                of(undefined)
                    .pipe(
                        // Wrap in switchMap so we don't break the observable chain when
                        // catchError returns a new observable, so repeatUntil will
                        // properly resubscribe to the outer observable and re-fetch.
                        switchMap(() =>
                            resolveRevision({ repoName, revision }).pipe(
                                catchError(error => {
                                    if (isCloneInProgressErrorLike(error)) {
                                        return of<ErrorLike>(asError(error))
                                    }
                                    throw error
                                })
                            )
                        )
                    )
                    .pipe(
                        repeatUntil(value => !isCloneInProgressErrorLike(value), { delay: 1000 }),
                        catchError(error => of<ErrorLike>(asError(error)))
                    ),
            [repoName, revision]
        )
    )

    // The external links to show in the repository header, if any.
    const [externalLinks, setExternalLinks] = useState<ExternalLinkFields[] | undefined>()

    // The lifecycle props for repo header contributions.
    const [
        repoHeaderContributionsLifecycleProps,
        setRepoHeaderContributionsLifecycleProps,
    ] = useState<RepoHeaderContributionsLifecycleProps>()

    const childBreadcrumbSetters = props.useBreadcrumb(
        useMemo(() => {
            if (isErrorLike(repoOrError) || !repoOrError) {
                return
            }

            return {
                key: 'repository',
                element: (
                    <>
                        <div className={classNames('d-inline-flex', isRedesignEnabled && 'btn-group')}>
                            <Link
                                to={
                                    resolvedRevisionOrError && !isErrorLike(resolvedRevisionOrError)
                                        ? resolvedRevisionOrError.rootTreeURL
                                        : repoOrError.url
                                }
                                className={classNames(
                                    'text-nowrap test-repo-header-repo-link',
                                    isRedesignEnabled ? 'btn btn-sm btn-outline-secondary' : 'font-weight-bold'
                                )}
                            >
                                <SourceRepositoryIcon className="icon-inline" /> {displayRepoName(repoOrError.name)}
                            </Link>
                            <button
                                type="button"
                                id="repo-popover"
                                className={classNames(
                                    'btn repo-container__repo-change',
                                    isRedesignEnabled ? 'btn-sm btn-outline-secondary' : 'btn-icon'
                                )}
                                aria-label="Change repository"
                            >
                                {isRedesignEnabled ? (
                                    <ChevronDownIcon className="icon-inline" />
                                ) : (
                                    <MenuDownIcon className="icon-inline" />
                                )}
                            </button>
                        </div>
                        <UncontrolledPopover
                            placement="bottom-start"
                            target="repo-popover"
                            trigger="legacy"
                            hideArrow={true}
                            fade={false}
                            popperClassName="border-0"
                        >
                            <RepositoriesPopover currentRepo={repoOrError.id} />
                        </UncontrolledPopover>
                    </>
                ),
            }
        }, [repoOrError, resolvedRevisionOrError, isRedesignEnabled])
    )

    // Update the workspace roots service to reflect the current repo / resolved revision
    useEffect(() => {
        const workspaceRootUri =
            resolvedRevisionOrError &&
            !isErrorLike(resolvedRevisionOrError) &&
            makeRepoURI({
                repoName,
                revision: resolvedRevisionOrError.commitID,
            })

        if (workspaceRootUri) {
            props.extensionsController.extHostAPI
                .then(extensionHostAPI =>
                    extensionHostAPI.addWorkspaceRoot({
                        uri: workspaceRootUri,
                        inputRevision: revision || '',
                    })
                )
                .catch(error => {
                    console.error('Error adding workspace root', error)
                })
        }

        // Clear the Sourcegraph extensions model's roots when navigating away.
        return () => {
            if (workspaceRootUri) {
                props.extensionsController.extHostAPI
                    .then(extensionHostAPI => extensionHostAPI.removeWorkspaceRoot(workspaceRootUri))
                    .catch(error => {
                        console.error('Error removing workspace root', error)
                    })
            }
        }
    }, [props.extensionsController, repoName, resolvedRevisionOrError, revision])

    // Update the navbar query to reflect the current repo / revision
    const { globbing, onNavbarQueryChange } = props
    useEffect(() => {
        let query = searchQueryForRepoRevision(repoName, globbing, revision)
        if (filePath) {
            query = `${query.trimEnd()} file:${escapeSpaces(globbing ? filePath : '^' + escapeRegExp(filePath))}`
        }
        onNavbarQueryChange({
            query,
        })
    }, [revision, filePath, repoName, onNavbarQueryChange, globbing])

    const { useActionItemsBar, useActionItemsToggle } = useWebActionItems()

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

    const [hasDismissedFirefoxAlert, setHasDismissedFirefoxAlert] = useLocalStorage(
        HAS_DISMISSED_FIREFOX_ALERT_KEY,
        false
    )
    const showFirefoxAddonAlert = isFirefox() && !hasDismissedFirefoxAlert && isFirefoxCampaignActive(Date.now())

    const onAlertDismissed = useCallback(() => {
        onExtensionAlertDismissed()
        setHasDismissedExtensionAlert(true)
        // TEMPORARY
        setHasDismissedFirefoxAlert(true)
    }, [onExtensionAlertDismissed, setHasDismissedExtensionAlert, setHasDismissedFirefoxAlert])

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
        return <HeroPage icon={AlertCircleIcon} title="Error" subtitle={<ErrorMessage error={repoOrError} />} />
    }

    const repoMatchURL = '/' + encodeURIPathComponent(repoName)

    const context: RepoContainerContext = {
        ...props,
        ...repoHeaderContributionsLifecycleProps,
        ...childBreadcrumbSetters,
        onHoverShown,
        repo: repoOrError,
        routePrefix: repoMatchURL,
        onDidUpdateExternalLinks: setExternalLinks,
        useActionItemsBar,
    }

    return (
        <div className="repo-container test-repo-container w-100 d-flex flex-column action-items">
            {!isErrorLike(props.settingsCascade.final) &&
                props.settingsCascade.final?.experimentalFeatures?.fuzzyFinder &&
                resolvedRevisionOrError &&
                !isErrorLike(resolvedRevisionOrError) && (
                    <FuzzyFinder
                        repoName={repoName}
                        commitID={resolvedRevisionOrError.commitID}
                        caseInsensitiveFileCountThreshold={
                            props.settingsCascade.final?.experimentalFeatures
                                ?.fuzzyFinderCaseInsensitiveFileCountThreshold
                        }
                    />
                )}
            {(showExtensionAlert || showFirefoxAddonAlert) && (
                <InstallBrowserExtensionAlert
                    isChrome={IS_CHROME}
                    onAlertDismissed={onAlertDismissed}
                    externalURLs={repoOrError.externalURLs}
                    codeHostIntegrationMessaging={codeHostIntegrationMessaging}
                    showFirefoxAddonAlert={showFirefoxAddonAlert}
                />
            )}
            <RepoHeader
                actionButtons={props.repoHeaderActionButtons}
                useActionItemsToggle={useActionItemsToggle}
                breadcrumbs={props.breadcrumbs}
                revision={revision}
                repo={repoOrError}
                resolvedRev={resolvedRevisionOrError}
                onLifecyclePropsChange={setRepoHeaderContributionsLifecycleProps}
                isAlertDisplayed={showExtensionAlert}
                location={props.location}
                history={props.history}
                settingsCascade={props.settingsCascade}
                authenticatedUser={props.authenticatedUser}
                platformContext={props.platformContext}
                extensionsController={props.extensionsController}
                telemetryService={props.telemetryService}
            />
            <RepoHeaderContributionPortal
                position="right"
                priority={2}
                id="go-to-code-host"
                {...repoHeaderContributionsLifecycleProps}
            >
                {({ actionType }) => (
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
                        actionType={actionType}
                        repoName={repoName}
                    />
                )}
            </RepoHeaderContributionPortal>
            <ErrorBoundary location={props.location}>
                <Switch>
                    {[
                        '',
                        ...(rawRevision ? [`@${rawRevision}`] : []), // must exactly match how the revision was encoded in the URL
                        '/-/blob',
                        '/-/tree',
                        '/-/commits',
                        '/-/docs',
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
                                    useActionItemsBar={useActionItemsBar}
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
                </Switch>
            </ErrorBoundary>
        </div>
    )
}
