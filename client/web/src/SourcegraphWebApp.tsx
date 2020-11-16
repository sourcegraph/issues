import 'focus-visible'

import { ShortcutProvider } from '@slimsag/react-shortcuts'
import ServerIcon from 'mdi-react/ServerIcon'
import * as React from 'react'
import { hot } from 'react-hot-loader/root'
import { Route } from 'react-router'
import { BrowserRouter } from 'react-router-dom'
import { combineLatest, from, Subscription, fromEvent, of } from 'rxjs'
import { startWith, switchMap } from 'rxjs/operators'
import { setLinkComponent } from '../../shared/src/components/Link'
import {
    Controller as ExtensionsController,
    createController as createExtensionsController,
} from '../../shared/src/extensions/controller'
import { Notifications } from '../../shared/src/notifications/Notifications'
import { PlatformContext } from '../../shared/src/platform/context'
import { EMPTY_SETTINGS_CASCADE, SettingsCascadeProps } from '../../shared/src/settings/settings'
import { authenticatedUser, AuthenticatedUser } from './auth'
import { ErrorBoundary } from './components/ErrorBoundary'
import { FeedbackText } from './components/FeedbackText'
import { HeroPage } from './components/HeroPage'
import { RouterLinkOrAnchor } from './components/RouterLinkOrAnchor'
import { Tooltip } from '../../branded/src/components/tooltip/Tooltip'
import { ExtensionAreaRoute } from './extensions/extension/ExtensionArea'
import { ExtensionAreaHeaderNavItem } from './extensions/extension/ExtensionAreaHeader'
import { ExtensionsAreaRoute } from './extensions/ExtensionsArea'
import { ExtensionsAreaHeaderActionButton } from './extensions/ExtensionsAreaHeader'
import { Layout, LayoutProps } from './Layout'
import { updateUserSessionStores } from './marketing/util'
import { OrgAreaRoute } from './org/area/OrgArea'
import { OrgAreaHeaderNavItem } from './org/area/OrgHeader'
import { createPlatformContext } from './platform/context'
import { fetchHighlightedFileLines } from './repo/backend'
import { RepoContainerRoute } from './repo/RepoContainer'
import { RepoHeaderActionButton } from './repo/RepoHeader'
import { RepoRevisionContainerRoute } from './repo/RepoRevisionContainer'
import { LayoutRouteProps } from './routes'
import { search, searchStream, fetchSavedSearches, fetchRecentSearches, fetchRecentFileViews } from './search/backend'
import { SiteAdminAreaRoute } from './site-admin/SiteAdminArea'
import { SiteAdminSideBarGroups } from './site-admin/SiteAdminSidebar'
import { ThemePreference } from './theme'
import { eventLogger } from './tracking/eventLogger'
import { withActivation } from './tracking/withActivation'
import { UserAreaRoute } from './user/area/UserArea'
import { UserAreaHeaderNavItem } from './user/area/UserAreaHeader'
import { UserSettingsAreaRoute } from './user/settings/UserSettingsArea'
import { UserSettingsSidebarItems } from './user/settings/UserSettingsSidebar'
import {
    parseSearchURLPatternType,
    searchURLIsCaseSensitive,
    parseSearchURLVersionContext,
    resolveVersionContext,
} from './search'
import { KeyboardShortcutsProps } from './keyboardShortcuts/keyboardShortcuts'
import { QueryState } from './search/helpers'
import { RepoSettingsAreaRoute } from './repo/settings/RepoSettingsArea'
import { RepoSettingsSideBarGroup } from './repo/settings/RepoSettingsSidebar'
import { FiltersToTypeAndValue } from '../../shared/src/search/interactive/util'
import { generateFiltersQuery } from '../../shared/src/util/url'
import { NotificationType } from '../../shared/src/api/client/services/notifications'
import { VersionContext } from './schema/site.schema'
import { globbingEnabledFromSettings } from './util/globbing'
import {
    SITE_SUBJECT_NO_ADMIN,
    viewerSubjectFromSettings,
    defaultPatternTypeFromSettings,
    experimentalFeaturesFromSettings,
} from './util/settings'
import { SearchPatternType } from '../../shared/src/graphql-operations'
import { HTTPStatusError } from '../../shared/src/backend/fetch'

export interface SourcegraphWebAppProps extends KeyboardShortcutsProps {
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
    showCampaigns: boolean
}

interface SourcegraphWebAppState extends SettingsCascadeProps {
    error?: Error

    /** The currently authenticated user (or null if the viewer is anonymous). */
    authenticatedUser?: AuthenticatedUser | null

    viewerSubject: LayoutProps['viewerSubject']

    /** The user's preference for the theme (light, dark or following system theme) */
    themePreference: ThemePreference

    /**
     * Whether the OS uses light theme, synced from a media query.
     * If the browser/OS does not this, will default to true.
     */
    systemIsLightTheme: boolean

    /**
     * The current search query in the navbar.
     */
    navbarSearchQueryState: QueryState

    /**
     * The current search pattern type.
     */
    searchPatternType: SearchPatternType

    /**
     * Whether the current search is case sensitive.
     */
    searchCaseSensitivity: boolean

    /**
     * filtersInQuery is the source of truth for the filter values currently in the query.
     *
     * The data structure is a map, where the key is a uniquely assigned string in the form `repoType-numberOfFilterAdded`.
     * The value is a data structure containing the fields {`type`, `value`, `editable`}.
     * `type` is the field type of the filter (repo, file, etc.) `value` is the current value for that particular filter,
     * and `editable` is whether the corresponding filter input is currently editable in the UI.
     * */
    filtersInQuery: FiltersToTypeAndValue

    /**
     * Whether interactive search mode is activated
     */
    interactiveSearchMode: boolean

    /**
     * Whether to display the option to toggle between interactive and omni search modes.
     */
    splitSearchModes: boolean

    /**
     * Whether to display the copy query button.
     */
    copyQueryButton: boolean

    /*
     * The version context the instance is in. If undefined, it means no version context is selected.
     */
    versionContext?: string

    /**
     * Available version contexts defined in the site configuration.
     */
    availableVersionContexts?: VersionContext[]

    /**
     * The previously used version context, as specified in localStorage.
     */
    previousVersionContext: string | null

    /**
     * Whether the experimental search streaming API should be used.
     */
    searchStreaming: boolean

    showRepogroupHomepage: boolean

    showOnboardingTour: boolean

    showEnterpriseHomePanels: boolean

    /**
     * Whether globbing is enabled for filters.
     */
    globbing: boolean

    /**
     * Whether we show the mulitiline editor at /search/console
     */
    showMultilineSearchConsole: boolean

    /**
     * Whether we show the mulitiline editor at /search/query-builder
     */
    showQueryBuilder: boolean

    /**
     * Wether to enable enable contextual syntax highlighting and hovers for search queries
     */
    enableSmartQuery: boolean
}

const notificationClassNames = {
    [NotificationType.Log]: 'alert alert-secondary',
    [NotificationType.Success]: 'alert alert-success',
    [NotificationType.Info]: 'alert alert-info',
    [NotificationType.Warning]: 'alert alert-warning',
    [NotificationType.Error]: 'alert alert-danger',
}

const LIGHT_THEME_LOCAL_STORAGE_KEY = 'light-theme'
const SEARCH_MODE_KEY = 'sg-search-mode'
const LAST_VERSION_CONTEXT_KEY = 'sg-last-version-context'

/** Reads the stored theme preference from localStorage */
const readStoredThemePreference = (): ThemePreference => {
    const value = localStorage.getItem(LIGHT_THEME_LOCAL_STORAGE_KEY)
    // Handle both old and new preference values
    switch (value) {
        case 'true':
        case 'light':
            return ThemePreference.Light
        case 'false':
        case 'dark':
            return ThemePreference.Dark
        default:
            return ThemePreference.System
    }
}

setLinkComponent(RouterLinkOrAnchor)

const LayoutWithActivation = window.context.sourcegraphDotComMode ? Layout : withActivation(Layout)

/**
 * The root component.
 *
 * This is the non-hot-reload component. It is wrapped in `hot(...)` below to make it
 * hot-reloadable in development.
 */
class ColdSourcegraphWebApp extends React.Component<SourcegraphWebAppProps, SourcegraphWebAppState> {
    private readonly subscriptions = new Subscription()
    private readonly darkThemeMediaList = window.matchMedia('(prefers-color-scheme: dark)')
    private readonly platformContext: PlatformContext = createPlatformContext()
    private readonly extensionsController: ExtensionsController = createExtensionsController(this.platformContext)

    constructor(props: SourcegraphWebAppProps) {
        super(props)
        this.subscriptions.add(this.extensionsController)

        // The patternType in the URL query parameter. If none is provided, default to literal.
        // This will be updated with the default in settings when the web app mounts.
        const urlPatternType = parseSearchURLPatternType(window.location.search) || SearchPatternType.literal
        const urlCase = searchURLIsCaseSensitive(window.location.search)
        const currentSearchMode = localStorage.getItem(SEARCH_MODE_KEY)
        const availableVersionContexts = window.context.experimentalFeatures.versionContexts
        const previousVersionContext = localStorage.getItem(LAST_VERSION_CONTEXT_KEY)
        const resolvedVersionContext = availableVersionContexts
            ? parseSearchURLVersionContext(window.location.search) ||
              resolveVersionContext(previousVersionContext || undefined, availableVersionContexts) ||
              undefined
            : undefined

        this.state = {
            themePreference: readStoredThemePreference(),
            systemIsLightTheme: !this.darkThemeMediaList.matches,
            navbarSearchQueryState: { query: '', cursorPosition: 0 },
            settingsCascade: EMPTY_SETTINGS_CASCADE,
            viewerSubject: SITE_SUBJECT_NO_ADMIN,
            searchPatternType: urlPatternType,
            searchCaseSensitivity: urlCase,
            filtersInQuery: {},
            splitSearchModes: false,
            interactiveSearchMode: currentSearchMode ? currentSearchMode === 'interactive' : false,
            copyQueryButton: false,
            versionContext: resolvedVersionContext,
            availableVersionContexts,
            previousVersionContext,
            searchStreaming: false,
            showRepogroupHomepage: false,
            showOnboardingTour: false,
            showEnterpriseHomePanels: false,
            globbing: false,
            showMultilineSearchConsole: false,
            showQueryBuilder: false,
            enableSmartQuery: false,
        }
    }

    /** Returns whether Sourcegraph should be in light theme */
    private isLightTheme(): boolean {
        return this.state.themePreference === 'system'
            ? this.state.systemIsLightTheme
            : this.state.themePreference === 'light'
    }

    public componentDidMount(): void {
        updateUserSessionStores()

        document.documentElement.classList.add('theme')

        this.subscriptions.add(
            combineLatest([from(this.platformContext.settings), authenticatedUser.pipe(startWith(null))]).subscribe(
                ([settingsCascade, authenticatedUser]) => {
                    this.setState(state => ({
                        settingsCascade,
                        authenticatedUser,
                        ...experimentalFeaturesFromSettings(settingsCascade),
                        globbing: globbingEnabledFromSettings(settingsCascade),
                        searchPatternType: defaultPatternTypeFromSettings(settingsCascade) || state.searchPatternType,
                        viewerSubject: viewerSubjectFromSettings(settingsCascade, authenticatedUser),
                    }))
                },
                () => this.setState({ authenticatedUser: null })
            )
        )

        // React to OS theme change
        this.subscriptions.add(
            fromEvent<MediaQueryListEvent>(this.darkThemeMediaList, 'change').subscribe(event => {
                this.setState({ systemIsLightTheme: !event.matches })
            })
        )

        /**
         * Listens for uncaught 401 errors when a user when a user was previously authenticated.
         *
         * Don't subscribe to this event when there wasn't an authenticated user,
         * as it could lead to an infinite loop of 401 -> reload -> 401
         */
        this.subscriptions.add(
            authenticatedUser
                .pipe(
                    switchMap(authenticatedUser =>
                        authenticatedUser ? fromEvent<ErrorEvent>(window, 'error') : of(null)
                    )
                )
                .subscribe(event => {
                    if (event?.error instanceof HTTPStatusError && event.error.status === 401) {
                        location.reload()
                    }
                })
        )

        // Send initial versionContext to extensions
        this.extensionsController.extHostAPI
            .then(extensionHostApi => extensionHostApi.setVersionContext(this.state.versionContext))
            .catch(console.error)
    }

    public componentWillUnmount(): void {
        this.subscriptions.unsubscribe()
        document.documentElement.classList.remove('theme', 'theme-light', 'theme-dark')
    }

    public componentDidUpdate(): void {
        localStorage.setItem(LIGHT_THEME_LOCAL_STORAGE_KEY, this.state.themePreference)
        document.documentElement.classList.toggle('theme-light', this.isLightTheme())
        document.documentElement.classList.toggle('theme-dark', !this.isLightTheme())
    }

    private toggleSearchMode = (event: React.MouseEvent<HTMLAnchorElement>): void => {
        event.preventDefault()
        localStorage.setItem(SEARCH_MODE_KEY, this.state.interactiveSearchMode ? 'plain' : 'interactive')

        eventLogger.log('SearchModeToggled', { mode: this.state.interactiveSearchMode ? 'plain' : 'interactive' })

        if (this.state.interactiveSearchMode) {
            const queries = [this.state.navbarSearchQueryState.query, generateFiltersQuery(this.state.filtersInQuery)]
            const newQuery = queries.filter(query => query.length > 0).join(' ')

            this.setState(state => ({
                interactiveSearchMode: !state.interactiveSearchMode,
                navbarSearchQueryState: { query: newQuery, cursorPosition: newQuery.length },
                filtersInQuery: {},
            }))
        } else {
            this.setState(state => ({
                interactiveSearchMode: !state.interactiveSearchMode,
            }))
        }
    }

    public render(): React.ReactFragment | null {
        if (window.pageError && window.pageError.statusCode !== 404) {
            const statusCode = window.pageError.statusCode
            const statusText = window.pageError.statusText
            const errorMessage = window.pageError.error
            const errorID = window.pageError.errorID

            let subtitle: JSX.Element | undefined
            if (errorID) {
                subtitle = <FeedbackText headerText="Sorry, there's been a problem." />
            }
            if (errorMessage) {
                subtitle = (
                    <div className="app__error">
                        {subtitle}
                        {subtitle && <hr className="my-3" />}
                        <pre>{errorMessage}</pre>
                    </div>
                )
            } else {
                subtitle = <div className="app__error">{subtitle}</div>
            }
            return <HeroPage icon={ServerIcon} title={`${statusCode}: ${statusText}`} subtitle={subtitle} />
        }

        const { authenticatedUser } = this.state
        if (authenticatedUser === undefined) {
            return null
        }

        const { children, ...props } = this.props

        return (
            <ErrorBoundary location={null}>
                <ShortcutProvider>
                    <BrowserRouter key={0}>
                        {/* eslint-disable react/jsx-no-bind */}
                        <Route
                            path="/"
                            render={routeComponentProps => (
                                <LayoutWithActivation
                                    {...props}
                                    {...routeComponentProps}
                                    authenticatedUser={authenticatedUser}
                                    viewerSubject={this.state.viewerSubject}
                                    settingsCascade={this.state.settingsCascade}
                                    showCampaigns={this.props.showCampaigns}
                                    // Theme
                                    isLightTheme={this.isLightTheme()}
                                    themePreference={this.state.themePreference}
                                    onThemePreferenceChange={this.onThemePreferenceChange}
                                    // Search query
                                    navbarSearchQueryState={this.state.navbarSearchQueryState}
                                    onNavbarQueryChange={this.onNavbarQueryChange}
                                    fetchHighlightedFileLines={fetchHighlightedFileLines}
                                    searchRequest={this.state.searchStreaming ? searchStream : search}
                                    searchStreaming={this.state.searchStreaming}
                                    // Extensions
                                    platformContext={this.platformContext}
                                    extensionsController={this.extensionsController}
                                    telemetryService={eventLogger}
                                    isSourcegraphDotCom={window.context.sourcegraphDotComMode}
                                    patternType={this.state.searchPatternType}
                                    caseSensitive={this.state.searchCaseSensitivity}
                                    splitSearchModes={this.state.splitSearchModes}
                                    interactiveSearchMode={this.state.interactiveSearchMode}
                                    toggleSearchMode={this.toggleSearchMode}
                                    filtersInQuery={this.state.filtersInQuery}
                                    onFiltersInQueryChange={this.onFiltersInQueryChange}
                                    setPatternType={this.setPatternType}
                                    setCaseSensitivity={this.setCaseSensitivity}
                                    copyQueryButton={this.state.copyQueryButton}
                                    versionContext={this.state.versionContext}
                                    setVersionContext={this.setVersionContext}
                                    availableVersionContexts={this.state.availableVersionContexts}
                                    previousVersionContext={this.state.previousVersionContext}
                                    showRepogroupHomepage={this.state.showRepogroupHomepage}
                                    showOnboardingTour={this.state.showOnboardingTour}
                                    showEnterpriseHomePanels={this.state.showEnterpriseHomePanels}
                                    globbing={this.state.globbing}
                                    showMultilineSearchConsole={this.state.showMultilineSearchConsole}
                                    showQueryBuilder={this.state.showQueryBuilder}
                                    enableSmartQuery={this.state.enableSmartQuery}
                                    fetchSavedSearches={fetchSavedSearches}
                                    fetchRecentSearches={fetchRecentSearches}
                                    fetchRecentFileViews={fetchRecentFileViews}
                                />
                            )}
                        />
                        {/* eslint-enable react/jsx-no-bind */}
                    </BrowserRouter>
                    <Tooltip key={1} />
                    <Notifications
                        key={2}
                        extensionsController={this.extensionsController}
                        notificationClassNames={notificationClassNames}
                    />
                </ShortcutProvider>
            </ErrorBoundary>
        )
    }

    private onThemePreferenceChange = (themePreference: ThemePreference): void => {
        this.setState({ themePreference })
    }

    private onNavbarQueryChange = (navbarSearchQueryState: QueryState): void => {
        this.setState({ navbarSearchQueryState })
    }

    private onFiltersInQueryChange = (filtersInQuery: FiltersToTypeAndValue): void => {
        this.setState({ filtersInQuery })
    }

    private setPatternType = (patternType: SearchPatternType): void => {
        this.setState({
            searchPatternType: patternType,
        })
    }

    private setCaseSensitivity = (caseSensitive: boolean): void => {
        this.setState({
            searchCaseSensitivity: caseSensitive,
        })
    }

    private setVersionContext = (versionContext: string | undefined): void => {
        const resolvedVersionContext = resolveVersionContext(versionContext, this.state.availableVersionContexts)
        if (!resolvedVersionContext) {
            localStorage.removeItem(LAST_VERSION_CONTEXT_KEY)
            this.setState({ versionContext: undefined, previousVersionContext: null })
        } else {
            localStorage.setItem(LAST_VERSION_CONTEXT_KEY, resolvedVersionContext)
            this.setState({ versionContext: resolvedVersionContext, previousVersionContext: resolvedVersionContext })
        }

        this.extensionsController.extHostAPI
            .then(extensionHostAPI => extensionHostAPI.setVersionContext(resolvedVersionContext))
            .catch(console.error)
    }
}

export const SourcegraphWebApp = hot(ColdSourcegraphWebApp)
