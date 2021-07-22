import React, { Suspense, useCallback, useEffect, useMemo } from 'react'
import { Redirect, Route, RouteComponentProps, Switch, matchPath } from 'react-router'
import { Observable } from 'rxjs'

import { ResizablePanel } from '@sourcegraph/branded/src/components/panel/Panel'
import { LoadingSpinner } from '@sourcegraph/react-loading-spinner'
import { ActivationProps } from '@sourcegraph/shared/src/components/activation/Activation'
import { FetchFileParameters } from '@sourcegraph/shared/src/components/CodeExcerpt'
import { ExtensionsControllerProps } from '@sourcegraph/shared/src/extensions/controller'
import * as GQL from '@sourcegraph/shared/src/graphql/schema'
import { PlatformContextProps } from '@sourcegraph/shared/src/platform/context'
import { SettingsCascadeProps } from '@sourcegraph/shared/src/settings/settings'
import { TelemetryProps } from '@sourcegraph/shared/src/telemetry/telemetryService'
import { ThemeProps } from '@sourcegraph/shared/src/theme'
import { parseQueryAndHash } from '@sourcegraph/shared/src/util/url'
import { useObservable } from '@sourcegraph/shared/src/util/useObservable'
import { useRedesignToggle } from '@sourcegraph/shared/src/util/useRedesignToggle'

import { AuthenticatedUser, authRequired as authRequiredObservable } from './auth'
import { CodeMonitoringProps } from './code-monitoring'
import { useBreadcrumbs } from './components/Breadcrumbs'
import { ErrorBoundary } from './components/ErrorBoundary'
import { useScrollToLocationHash } from './components/useScrollToLocationHash'
import { GlobalContributions } from './contributions'
import { ExtensionAreaRoute } from './extensions/extension/ExtensionArea'
import { ExtensionAreaHeaderNavItem } from './extensions/extension/ExtensionAreaHeader'
import { ExtensionsAreaRoute } from './extensions/ExtensionsArea'
import { ExtensionsAreaHeaderActionButton } from './extensions/ExtensionsAreaHeader'
import { FeatureFlagProps } from './featureFlags/featureFlags'
import { GlobalAlerts } from './global/GlobalAlerts'
import { GlobalDebug } from './global/GlobalDebug'
import { KeyboardShortcutsProps, KEYBOARD_SHORTCUT_SHOW_HELP } from './keyboardShortcuts/keyboardShortcuts'
import { KeyboardShortcutsHelp } from './keyboardShortcuts/KeyboardShortcutsHelp'
import { SurveyToast } from './marketing/SurveyToast'
import { GlobalNavbar } from './nav/GlobalNavbar'
import { useExtensionAlertAnimation } from './nav/UserNavItem'
import { OrgAreaRoute } from './org/area/OrgArea'
import { OrgAreaHeaderNavItem } from './org/area/OrgHeader'
import { fetchHighlightedFileLineRanges } from './repo/backend'
import { RepoContainerRoute } from './repo/RepoContainer'
import { RepoHeaderActionButton } from './repo/RepoHeader'
import { RepoRevisionContainerRoute } from './repo/RepoRevisionContainer'
import { RepoSettingsAreaRoute } from './repo/settings/RepoSettingsArea'
import { RepoSettingsSideBarGroup } from './repo/settings/RepoSettingsSidebar'
import { LayoutRouteProps, LayoutRouteComponentProps } from './routes'
import { Settings } from './schema/settings.schema'
import {
    parseSearchURLQuery,
    PatternTypeProps,
    CaseSensitivityProps,
    RepogroupHomepageProps,
    OnboardingTourProps,
    HomePanelsProps,
    SearchStreamingProps,
    ParsedSearchQueryProps,
    MutableVersionContextProps,
    parseSearchURL,
    SearchContextProps,
    getGlobalSearchContextFilter,
} from './search'
import { QueryState } from './search/helpers'
import { SiteAdminAreaRoute } from './site-admin/SiteAdminArea'
import { SiteAdminSideBarGroups } from './site-admin/SiteAdminSidebar'
import { ThemePreferenceProps } from './theme'
import { UserAreaRoute } from './user/area/UserArea'
import { UserAreaHeaderNavItem } from './user/area/UserAreaHeader'
import { UserSettingsAreaRoute } from './user/settings/UserSettingsArea'
import { UserSettingsSidebarItems } from './user/settings/UserSettingsSidebar'
import { isMacPlatform, UserExternalServicesOrRepositoriesUpdateProps } from './util'
import { parseBrowserRepoURL } from './util/url'

export interface LayoutProps
    extends RouteComponentProps<{}>,
        SettingsCascadeProps<Settings>,
        PlatformContextProps,
        ExtensionsControllerProps,
        KeyboardShortcutsProps,
        ThemeProps,
        TelemetryProps,
        ThemePreferenceProps,
        ActivationProps,
        ParsedSearchQueryProps,
        PatternTypeProps,
        CaseSensitivityProps,
        MutableVersionContextProps,
        RepogroupHomepageProps,
        OnboardingTourProps,
        SearchContextProps,
        HomePanelsProps,
        SearchStreamingProps,
        CodeMonitoringProps,
        SearchContextProps,
        UserExternalServicesOrRepositoriesUpdateProps,
        FeatureFlagProps {
    extensionAreaRoutes: readonly ExtensionAreaRoute[]
    extensionAreaHeaderNavItems: readonly ExtensionAreaHeaderNavItem[]
    extensionsAreaRoutes: readonly ExtensionsAreaRoute[]
    extensionsAreaHeaderActionButtons: readonly ExtensionsAreaHeaderActionButton[]
    siteAdminAreaRoutes: readonly SiteAdminAreaRoute[]
    siteAdminSideBarGroups: SiteAdminSideBarGroups
    siteAdminOverviewComponents: readonly React.ComponentType[]
    userAreaHeaderNavItems: readonly UserAreaHeaderNavItem[]
    userAreaRoutes: readonly UserAreaRoute[]
    userSettingsSideBarItems: UserSettingsSidebarItems
    userSettingsAreaRoutes: readonly UserSettingsAreaRoute[]
    orgAreaHeaderNavItems: readonly OrgAreaHeaderNavItem[]
    orgAreaRoutes: readonly OrgAreaRoute[]
    repoContainerRoutes: readonly RepoContainerRoute[]
    repoRevisionContainerRoutes: readonly RepoRevisionContainerRoute[]
    repoHeaderActionButtons: readonly RepoHeaderActionButton[]
    repoSettingsAreaRoutes: readonly RepoSettingsAreaRoute[]
    repoSettingsSidebarGroups: readonly RepoSettingsSideBarGroup[]
    routes: readonly LayoutRouteProps<any>[]

    authenticatedUser: AuthenticatedUser | null

    /**
     * The subject GraphQL node ID of the viewer, which is used to look up the viewer's settings. This is either
     * the site's GraphQL node ID (for anonymous users) or the authenticated user's GraphQL node ID.
     */
    viewerSubject: Pick<GQL.ISettingsSubject, 'id' | 'viewerCanAdminister'>

    // Search
    navbarSearchQueryState: QueryState
    onNavbarQueryChange: (queryState: QueryState) => void
    fetchHighlightedFileLineRanges: (parameters: FetchFileParameters, force?: boolean) => Observable<string[][]>

    globbing: boolean
    showMultilineSearchConsole: boolean
    showSearchNotebook: boolean
    showQueryBuilder: boolean
    isSourcegraphDotCom: boolean
    showBatchChanges: boolean
    fetchSavedSearches: () => Observable<GQL.ISavedSearch[]>
    children?: never
}

export const Layout: React.FunctionComponent<LayoutProps> = props => {
    const routeMatch = props.routes.find(({ path, exact }) => matchPath(props.location.pathname, { path, exact }))?.path
    const isSearchRelatedPage = (routeMatch === '/:repoRevAndRest+' || routeMatch?.startsWith('/search')) ?? false
    const minimalNavLinks = routeMatch === '/cncf'
    const isSearchHomepage = props.location.pathname === '/search' && !parseSearchURLQuery(props.location.search)
    const isSearchConsolePage = routeMatch?.startsWith('/search/console')
    const isSearchNotebookPage = routeMatch?.startsWith('/search/notebook')

    // Update parsedSearchQuery, patternType, caseSensitivity, versionContext, and selectedSearchContextSpec based on current URL
    const {
        history,
        parsedSearchQuery: currentQuery,
        patternType: currentPatternType,
        caseSensitive: currentCaseSensitive,
        versionContext: currentVersionContext,
        selectedSearchContextSpec,
        location,
        setParsedSearchQuery,
        setPatternType,
        setCaseSensitivity,
        setVersionContext,
        setSelectedSearchContextSpec,
    } = props

    const { query = '', patternType, caseSensitive, versionContext } = useMemo(() => parseSearchURL(location.search), [
        location.search,
    ])

    const searchContextSpec = useMemo(() => getGlobalSearchContextFilter(query)?.spec, [query])

    useEffect(() => {
        if (query !== currentQuery) {
            setParsedSearchQuery(query)
        }

        // Only override filters from URL if there is a search query
        if (query) {
            if (patternType && patternType !== currentPatternType) {
                setPatternType(patternType)
            }

            if (caseSensitive !== currentCaseSensitive) {
                setCaseSensitivity(caseSensitive)
            }

            if (versionContext !== currentVersionContext) {
                setVersionContext(versionContext).catch(error => {
                    console.error('Error sending version context to extensions', error)
                })
            }

            if (searchContextSpec && searchContextSpec !== selectedSearchContextSpec) {
                setSelectedSearchContextSpec(searchContextSpec)
            }
        }
    }, [
        history,
        caseSensitive,
        currentCaseSensitive,
        currentPatternType,
        currentQuery,
        currentVersionContext,
        selectedSearchContextSpec,
        patternType,
        query,
        setCaseSensitivity,
        setParsedSearchQuery,
        setPatternType,
        setVersionContext,
        versionContext,
        setSelectedSearchContextSpec,
        searchContextSpec,
    ])

    // Hack! Hardcode these routes into cmd/frontend/internal/app/ui/router.go
    const repogroupPages = [
        '/refactor-python2-to-3',
        '/kubernetes',
        '/golang',
        '/react-hooks',
        '/android',
        '/stanford',
        '/stackstorm',
        '/temporal',
        '/o3de',
        '/cncf',
    ]
    const isRepogroupPage = repogroupPages.includes(props.location.pathname)

    // TODO add a component layer as the parent of the Layout component rendering "top-level" routes that do not render the navbar,
    // so that Layout can always render the navbar.
    const needsSiteInit = window.context?.needsSiteInit
    const isSiteInit = props.location.pathname === '/site-admin/init'
    const isSignInOrUp =
        props.location.pathname === '/sign-in' ||
        props.location.pathname === '/sign-up' ||
        props.location.pathname === '/password-reset' ||
        props.location.pathname === '/post-sign-up'

    // TODO Change this behavior when we have global focus management system
    // Need to know this for disable autofocus on nav search input
    // and preserve autofocus for first textarea at survey page, creation UI etc.
    const isSearchAutoFocusRequired = routeMatch === '/survey/:score?' || routeMatch === '/insights'

    const authRequired = useObservable(authRequiredObservable)

    const hideGlobalSearchInput: boolean =
        props.location.pathname === '/stats' ||
        props.location.pathname === '/search/query-builder' ||
        props.location.pathname === '/search/console' ||
        props.location.pathname === '/search/notebook'

    const breadcrumbProps = useBreadcrumbs()

    const [isRedesignEnabled] = useRedesignToggle()

    // Control browser extension discoverability animation here.
    // `Layout` is the lowest common ancestor of `UserNavItem` (target) and `RepoContainer` (trigger)
    const { isExtensionAlertAnimating, startExtensionAlertAnimation } = useExtensionAlertAnimation()
    const onExtensionAlertDismissed = useCallback(() => {
        startExtensionAlertAnimation()
    }, [startExtensionAlertAnimation])

    useScrollToLocationHash(props.location)
    // Remove trailing slash (which is never valid in any of our URLs).
    if (props.location.pathname !== '/' && props.location.pathname.endsWith('/')) {
        return <Redirect to={{ ...props.location, pathname: props.location.pathname.slice(0, -1) }} />
    }

    const context: LayoutRouteComponentProps<any> = {
        ...props,
        ...breadcrumbProps,
        onExtensionAlertDismissed,
        isRedesignEnabled,
        isMacPlatform,
    }

    return (
        <div className="layout">
            <KeyboardShortcutsHelp
                keyboardShortcutForShow={KEYBOARD_SHORTCUT_SHOW_HELP}
                keyboardShortcuts={props.keyboardShortcuts}
            />
            <GlobalAlerts authenticatedUser={props.authenticatedUser} settingsCascade={props.settingsCascade} />
            {!isSiteInit && <SurveyToast authenticatedUser={props.authenticatedUser} />}
            {!isSiteInit && !isSignInOrUp && (
                <GlobalNavbar
                    {...props}
                    authRequired={!!authRequired}
                    showSearchBox={
                        isSearchRelatedPage &&
                        !isSearchHomepage &&
                        !isRepogroupPage &&
                        !isSearchConsolePage &&
                        !isSearchNotebookPage
                    }
                    variant={
                        hideGlobalSearchInput
                            ? 'no-search-input'
                            : isSearchHomepage
                            ? 'low-profile'
                            : isRepogroupPage
                            ? 'low-profile-with-logo'
                            : 'default'
                    }
                    hideNavLinks={false}
                    minimalNavLinks={minimalNavLinks}
                    isSearchAutoFocusRequired={!isSearchAutoFocusRequired}
                    isExtensionAlertAnimating={isExtensionAlertAnimating}
                />
            )}
            {needsSiteInit && !isSiteInit && <Redirect to="/site-admin/init" />}
            <ErrorBoundary location={props.location}>
                <Suspense
                    fallback={
                        <div className="flex flex-1">
                            <LoadingSpinner className="icon-inline m-2" />
                        </div>
                    }
                >
                    <Switch>
                        {props.routes.map(
                            ({ render, condition = () => true, ...route }) =>
                                condition(context) && (
                                    <Route
                                        {...route}
                                        key="hardcoded-key" // see https://github.com/ReactTraining/react-router/issues/4578#issuecomment-334489490
                                        component={undefined}
                                        render={routeComponentProps => (
                                            <div className="layout__app-router-container">
                                                {render({ ...context, ...routeComponentProps })}
                                            </div>
                                        )}
                                    />
                                )
                        )}
                    </Switch>
                </Suspense>
            </ErrorBoundary>
            {parseQueryAndHash(props.location.search, props.location.hash).viewState &&
                props.location.pathname !== '/sign-in' && (
                    <ResizablePanel
                        {...props}
                        repoName={`git://${parseBrowserRepoURL(props.location.pathname).repoName}`}
                        fetchHighlightedFileLineRanges={fetchHighlightedFileLineRanges}
                    />
                )}
            <GlobalContributions
                key={3}
                extensionsController={props.extensionsController}
                platformContext={props.platformContext}
                history={props.history}
            />
            <GlobalDebug {...props} />
        </div>
    )
}
