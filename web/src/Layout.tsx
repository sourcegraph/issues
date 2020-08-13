import { LoadingSpinner } from '@sourcegraph/react-loading-spinner'
import React, { Suspense } from 'react'
import { Redirect, Route, RouteComponentProps, Switch, matchPath } from 'react-router'
import { Observable } from 'rxjs'
import { ActivationProps } from '../../shared/src/components/activation/Activation'
import { FetchFileCtx } from '../../shared/src/components/CodeExcerpt'
import { ExtensionsControllerProps } from '../../shared/src/extensions/controller'
import * as GQL from '../../shared/src/graphql/schema'
import { ResizablePanel } from '../../shared/src/panel/Panel'
import { PlatformContextProps } from '../../shared/src/platform/context'
import { SettingsCascadeProps } from '../../shared/src/settings/settings'
import { ErrorLike } from '../../shared/src/util/errors'
import { parseHash } from '../../shared/src/util/url'
import { ErrorBoundary } from './components/ErrorBoundary'
import { useScrollToLocationHash } from './components/useScrollToLocationHash'
import { GlobalContributions } from './contributions'
import { ExploreSectionDescriptor } from './explore/ExploreArea'
import { ExtensionAreaRoute } from './extensions/extension/ExtensionArea'
import { ExtensionAreaHeaderNavItem } from './extensions/extension/ExtensionAreaHeader'
import { ExtensionsAreaRoute } from './extensions/ExtensionsArea'
import { ExtensionsAreaHeaderActionButton } from './extensions/ExtensionsAreaHeader'
import { GlobalAlerts } from './global/GlobalAlerts'
import { GlobalDebug } from './global/GlobalDebug'
import { KeyboardShortcutsHelp } from './keyboardShortcuts/KeyboardShortcutsHelp'
import { IntegrationsToast } from './marketing/IntegrationsToast'
import { GlobalNavbar } from './nav/GlobalNavbar'
import { OrgAreaRoute } from './org/area/OrgArea'
import { OrgAreaHeaderNavItem } from './org/area/OrgHeader'
import { fetchHighlightedFileLines } from './repo/backend'
import { RepoContainerRoute } from './repo/RepoContainer'
import { RepoHeaderActionButton } from './repo/RepoHeader'
import { RepoRevisionContainerRoute } from './repo/RepoRevisionContainer'
import { LayoutRouteProps } from './routes'
import {
    parseSearchURLQuery,
    PatternTypeProps,
    InteractiveSearchProps,
    CaseSensitivityProps,
    SmartSearchFieldProps,
    CopyQueryButtonProps,
    RepogroupHomepageProps,
} from './search'
import { SiteAdminAreaRoute } from './site-admin/SiteAdminArea'
import { SiteAdminSideBarGroups } from './site-admin/SiteAdminSidebar'
import { EventLogger, EventLoggerProps } from './tracking/eventLogger'
import { UserAreaRoute } from './user/area/UserArea'
import { UserAreaHeaderNavItem } from './user/area/UserAreaHeader'
import { UserSettingsAreaRoute } from './user/settings/UserSettingsArea'
import { UserSettingsSidebarItems } from './user/settings/UserSettingsSidebar'
import { parseBrowserRepoURL } from './util/url'
import { SurveyToast } from './marketing/SurveyToast'
import { ThemeProps } from '../../shared/src/theme'
import { ThemePreferenceProps } from './theme'
import { KeyboardShortcutsProps, KEYBOARD_SHORTCUT_SHOW_HELP } from './keyboardShortcuts/keyboardShortcuts'
import { QueryState } from './search/helpers'
import { RepoSettingsAreaRoute } from './repo/settings/RepoSettingsArea'
import { VersionContextProps } from '../../shared/src/search/util'
import { VersionContext } from './schema/site.schema'
import { RepoSettingsSideBarGroup } from './repo/settings/RepoSettingsSidebar'
import { Settings } from './schema/settings.schema'
import { Remote } from 'comlink'
import { FlatExtHostAPI } from '../../shared/src/api/contract'

export interface LayoutProps
    extends RouteComponentProps<{}>,
        SettingsCascadeProps<Settings>,
        PlatformContextProps,
        ExtensionsControllerProps,
        KeyboardShortcutsProps,
        ThemeProps,
        EventLoggerProps,
        ThemePreferenceProps,
        ActivationProps,
        PatternTypeProps,
        CaseSensitivityProps,
        InteractiveSearchProps,
        SmartSearchFieldProps,
        CopyQueryButtonProps,
        VersionContextProps,
        RepogroupHomepageProps {
    exploreSections: readonly ExploreSectionDescriptor[]
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

    authenticatedUser: GQL.IUser | null

    /**
     * The subject GraphQL node ID of the viewer, which is used to look up the viewer's settings. This is either
     * the site's GraphQL node ID (for anonymous users) or the authenticated user's GraphQL node ID.
     */
    viewerSubject: Pick<GQL.ISettingsSubject, 'id' | 'viewerCanAdminister'>

    telemetryService: EventLogger

    // Search
    navbarSearchQueryState: QueryState
    onNavbarQueryChange: (queryState: QueryState) => void
    fetchHighlightedFileLines: (ctx: FetchFileCtx, force?: boolean) => Observable<string[]>
    searchRequest: (
        query: QueryState['query'],
        version: string,
        patternType: GQL.SearchPatternType,
        versionContext: string | undefined,
        extensionHostPromise: Promise<Remote<FlatExtHostAPI>>
    ) => Observable<GQL.ISearchResults | ErrorLike>
    setVersionContext: (versionContext: string | undefined) => void
    availableVersionContexts: VersionContext[] | undefined
    previousVersionContext: string | null
    globbing: boolean
    isSourcegraphDotCom: boolean
    showCampaigns: boolean
    children?: never
}

export const Layout: React.FunctionComponent<LayoutProps> = props => {
    const routeMatch = props.routes.find(({ path, exact }) => matchPath(props.location.pathname, { path, exact }))?.path
    const isSearchRelatedPage = (routeMatch === '/:repoRevAndRest+' || routeMatch?.startsWith('/search')) ?? false
    const isSearchHomepage = props.location.pathname === '/search' && !parseSearchURLQuery(props.location.search)

    // Hack! Hardcode these routes into cmd/frontend/internal/app/ui/router.go
    const repogroupPages = ['/refactor-python2-to-3', '/kubernetes', '/golang', '/react-hooks', '/android', '/stanford']
    const isRepogroupPage = repogroupPages.includes(props.location.pathname)

    const needsSiteInit = window.context.needsSiteInit
    const isSiteInit = props.location.pathname === '/site-admin/init'

    const hideGlobalSearchInput: boolean =
        props.location.pathname === '/stats' || props.location.pathname === '/search/query-builder'

    useScrollToLocationHash(props.location)
    // Remove trailing slash (which is never valid in any of our URLs).
    if (props.location.pathname !== '/' && props.location.pathname.endsWith('/')) {
        return <Redirect to={{ ...props.location, pathname: props.location.pathname.slice(0, -1) }} />
    }

    return (
        <div className="layout">
            <KeyboardShortcutsHelp
                keyboardShortcutForShow={KEYBOARD_SHORTCUT_SHOW_HELP}
                keyboardShortcuts={props.keyboardShortcuts}
            />
            <GlobalAlerts
                isSiteAdmin={!!props.authenticatedUser && props.authenticatedUser.siteAdmin}
                settingsCascade={props.settingsCascade}
                history={props.history}
            />
            {!needsSiteInit && !isSiteInit && !!props.authenticatedUser && (
                <IntegrationsToast history={props.history} />
            )}
            {!isSiteInit && <SurveyToast authenticatedUser={props.authenticatedUser} />}
            {!isSiteInit && (
                <GlobalNavbar
                    {...props}
                    isSearchRelatedPage={isSearchRelatedPage}
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
                />
            )}
            {needsSiteInit && !isSiteInit && <Redirect to="/site-admin/init" />}
            <ErrorBoundary location={props.location}>
                <Suspense fallback={<LoadingSpinner className="icon-inline m-2" />}>
                    <Switch>
                        {/* eslint-disable react/jsx-no-bind */}
                        {props.routes.map(
                            ({ render, condition = () => true, ...route }) =>
                                condition(props) && (
                                    <Route
                                        {...route}
                                        key="hardcoded-key" // see https://github.com/ReactTraining/react-router/issues/4578#issuecomment-334489490
                                        component={undefined}
                                        render={routeComponentProps => (
                                            <div className="layout__app-router-container">
                                                {render({ ...props, ...routeComponentProps })}
                                            </div>
                                        )}
                                    />
                                )
                        )}
                        {/* eslint-enable react/jsx-no-bind */}
                    </Switch>
                </Suspense>
            </ErrorBoundary>
            {parseHash(props.location.hash).viewState && props.location.pathname !== '/sign-in' && (
                <ResizablePanel
                    {...props}
                    repoName={`git://${parseBrowserRepoURL(props.location.pathname).repoName}`}
                    fetchHighlightedFileLines={fetchHighlightedFileLines}
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
