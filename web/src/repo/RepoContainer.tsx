import AlertCircleIcon from 'mdi-react/AlertCircleIcon'
import MapSearchIcon from 'mdi-react/MapSearchIcon'
import React, { useMemo, useState, useEffect, useCallback } from 'react'
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
import { makeRepoURI } from '../../../shared/src/util/url'
import { ErrorBoundary } from '../components/ErrorBoundary'
import { HeroPage } from '../components/HeroPage'
import {
    searchQueryForRepoRevision,
    PatternTypeProps,
    CaseSensitivityProps,
    InteractiveSearchProps,
    repoFilterForRepoRevision,
    CopyQueryButtonProps,
} from '../search'
import { RouteDescriptor } from '../util/contributions'
import { parseBrowserRepoURL } from '../util/url'
import { GoToCodeHostAction } from './actions/GoToCodeHostAction'
import { fetchRepository, resolveRevision } from './backend'
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
import SourceRepositoryIcon from 'mdi-react/SourceRepositoryIcon'

/**
 * Props passed to sub-routes of {@link RepoContainer}.
 */
export interface RepoContainerContext
    extends RepoHeaderContributionsLifecycleProps,
        SettingsCascadeProps,
        ExtensionsControllerProps,
        PlatformContextProps,
        ThemeProps,
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
        SettingsCascadeProps,
        PlatformContextProps,
        TelemetryProps,
        ExtensionsControllerProps,
        ActivationProps,
        ThemeProps,
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

    const childBreadcrumbSetters = props.useBreadcrumb(
        useMemo(() => {
            if (isErrorLike(repoOrError) || !repoOrError) {
                return
            }

            const [repoDirectory, repoBase] = splitPath(displayRepoName(repoOrError.name))

            return {
                key: 'repository',
                divider: null,
                noHome: true,
                element: (
                    <>
                        <Link
                            to={
                                resolvedRevisionOrError && !isErrorLike(resolvedRevisionOrError)
                                    ? resolvedRevisionOrError.rootTreeURL
                                    : repoOrError.url
                            }
                            className="h5 mb-0 d-inline-flex align-items-center test-repo-breadcrumb__repo"
                        >
                            <SourceRepositoryIcon className="icon-inline mr-1" />
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
                        <UncontrolledPopover placement="bottom-start" target="repo-popover" trigger="legacy">
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

    // Update the workspace roots service to reflect the current repo / resolved revision
    useEffect(() => {
        props.extensionsController.services.workspace.roots.next(
            resolvedRevisionOrError && !isErrorLike(resolvedRevisionOrError)
                ? [
                      {
                          uri: makeRepoURI({
                              repoName,
                              revision: resolvedRevisionOrError.commitID,
                          }),
                          inputRevision: revision || '',
                      },
                  ]
                : []
        )
        // Clear the Sourcegraph extensions model's roots when navigating away.
        return () => props.extensionsController.services.workspace.roots.next([])
    }, [props.extensionsController.services.workspace.roots, repoName, resolvedRevisionOrError, revision])

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
                query = `${query.trimEnd()} file:${globbing ? filePath : '^' + escapeRegExp(filePath)}`
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

    const repoMatchURL = `/${repoOrError.name}`

    const context: RepoContainerContext = {
        ...props,
        ...repoHeaderContributionsLifecycleProps,
        ...childBreadcrumbSetters,
        repo: repoOrError,
        routePrefix: repoMatchURL,
        onDidUpdateExternalLinks: setExternalLinks,
        onDidUpdateRepository: nextRepoOrErrorUpdate,
    }

    return (
        <div className="repo-container test-repo-container w-100 d-flex flex-column">
            <RepoHeader
                {...props}
                actionButtons={props.repoHeaderActionButtons}
                revision={revision}
                repo={repoOrError}
                resolvedRev={resolvedRevisionOrError}
                onLifecyclePropsChange={setRepoHeaderContributionsLifecycleProps}
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
