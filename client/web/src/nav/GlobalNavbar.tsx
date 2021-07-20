import * as H from 'history'
import BarChartIcon from 'mdi-react/BarChartIcon'
import MagnifyIcon from 'mdi-react/MagnifyIcon'
import PuzzleOutlineIcon from 'mdi-react/PuzzleOutlineIcon'
import React, { useEffect, useMemo } from 'react'
import { of } from 'rxjs'
import { startWith } from 'rxjs/operators'

import { isErrorLike } from '@sourcegraph/codeintellify/lib/errors'
import { ContributableMenu } from '@sourcegraph/shared/src/api/protocol'
import { ActivationProps } from '@sourcegraph/shared/src/components/activation/Activation'
import { ActivationDropdown } from '@sourcegraph/shared/src/components/activation/ActivationDropdown'
import { Link } from '@sourcegraph/shared/src/components/Link'
import { LinkOrSpan } from '@sourcegraph/shared/src/components/LinkOrSpan'
import { ExtensionsControllerProps } from '@sourcegraph/shared/src/extensions/controller'
import { PlatformContextProps } from '@sourcegraph/shared/src/platform/context'
import { omitFilter } from '@sourcegraph/shared/src/search/query/transformer'
import { VersionContextProps } from '@sourcegraph/shared/src/search/util'
import { SettingsCascadeProps } from '@sourcegraph/shared/src/settings/settings'
import { TelemetryProps } from '@sourcegraph/shared/src/telemetry/telemetryService'
import { ThemeProps } from '@sourcegraph/shared/src/theme'
import { useObservable } from '@sourcegraph/shared/src/util/useObservable'
import { useRedesignToggle } from '@sourcegraph/shared/src/util/useRedesignToggle'
import { WebCommandListPopoverButton } from '@sourcegraph/web/src/components/shared'
import { FeedbackPrompt } from '@sourcegraph/web/src/nav/Feedback/FeedbackPrompt'
import { StatusMessagesNavItem } from '@sourcegraph/web/src/nav/StatusMessagesNavItem'
import { NavGroup, NavItem, NavBar, NavLink, NavActions, NavAction } from '@sourcegraph/wildcard/src/components/NavBar'

import { AuthenticatedUser } from '../auth'
import { BatchChangesNavItem } from '../batches/BatchChangesNavItem'
import { CodeMonitoringProps } from '../code-monitoring'
import { CodeMonitoringLogo } from '../code-monitoring/CodeMonitoringLogo'
import { BrandLogo } from '../components/branding/BrandLogo'
import { FeatureFlagProps } from '../featureFlags/featureFlags'
import {
    KeyboardShortcutsProps,
    KEYBOARD_SHORTCUT_SHOW_COMMAND_PALETTE,
    KEYBOARD_SHORTCUT_SWITCH_THEME,
} from '../keyboardShortcuts/keyboardShortcuts'
import { LayoutRouteProps } from '../routes'
import { Settings } from '../schema/settings.schema'
import { VersionContext } from '../schema/site.schema'
import {
    PatternTypeProps,
    CaseSensitivityProps,
    OnboardingTourProps,
    ParsedSearchQueryProps,
    isSearchContextSpecAvailable,
    getGlobalSearchContextFilter,
    SearchContextInputProps,
} from '../search'
import { QueryState } from '../search/helpers'
import { SearchNavbarItem } from '../search/input/SearchNavbarItem'
import { ThemePreferenceProps } from '../theme'
import { userExternalServicesEnabledFromTags } from '../user/settings/cloud-ga'
import { UserSettingsSidebarItems } from '../user/settings/UserSettingsSidebar'
import { showDotComMarketing } from '../util/features'

import { NavLinks } from './NavLinks'
import { ExtensionAlertAnimationProps, UserNavItem } from './UserNavItem'

interface Props
    extends SettingsCascadeProps<Settings>,
        PlatformContextProps,
        ExtensionsControllerProps,
        KeyboardShortcutsProps,
        TelemetryProps,
        ThemeProps,
        ThemePreferenceProps,
        ExtensionAlertAnimationProps,
        ActivationProps,
        Pick<ParsedSearchQueryProps, 'parsedSearchQuery'>,
        PatternTypeProps,
        CaseSensitivityProps,
        VersionContextProps,
        SearchContextInputProps,
        CodeMonitoringProps,
        OnboardingTourProps,
        FeatureFlagProps {
    history: H.History
    location: H.Location<{ query: string }>
    authenticatedUser: AuthenticatedUser | null
    authRequired: boolean
    navbarSearchQueryState: QueryState
    onNavbarQueryChange: (queryState: QueryState) => void
    isSourcegraphDotCom: boolean
    showSearchBox: boolean
    showBatchChanges: boolean
    routes: readonly LayoutRouteProps<{}>[]

    // Whether globbing is enabled for filters.
    globbing: boolean

    userSettingsSideBarItems?: UserSettingsSidebarItems

    /**
     * Which variation of the global navbar to render.
     *
     * 'low-profile' renders the the navbar with no border or background. Used on the search
     * homepage.
     *
     * 'low-profile-with-logo' renders the low-profile navbar but with the homepage logo. Used on repogroup pages.
     */
    // TODO: after redesign refresh is done remove no-search-input variant
    variant: 'default' | 'low-profile' | 'low-profile-with-logo' | 'no-search-input'

    setVersionContext: (versionContext: string | undefined) => Promise<void>
    availableVersionContexts: VersionContext[] | undefined

    minimalNavLinks?: boolean
    isSearchAutoFocusRequired?: boolean
    branding?: typeof window.context.branding

    /** For testing only. Used because reactstrap's Popover is incompatible with react-test-renderer. */
    hideNavLinks: boolean
}

export const GlobalNavbar: React.FunctionComponent<Props> = ({
    authRequired,
    showSearchBox,
    navbarSearchQueryState,
    caseSensitive,
    patternType,
    onNavbarQueryChange,
    hideNavLinks,
    variant,
    isLightTheme,
    branding,
    location,
    history,
    minimalNavLinks,
    isSourcegraphDotCom,
    ...props
}) => {
    // Workaround: can't put this in optional parameter value because of https://github.com/babel/babel/issues/11166
    branding = branding ?? window.context?.branding

    const query = props.parsedSearchQuery

    const globalSearchContextSpec = useMemo(() => getGlobalSearchContextFilter(query), [query])

    // Design Refresh will include repositories section as part of the user navigation bar
    // This filter makes sure repositories feature flag is active.
    const showRepositorySection = useMemo(
        () => !!props.userSettingsSideBarItems?.find(item => item.label === 'Repositories'),
        [props.userSettingsSideBarItems]
    )

    const isSearchContextAvailable = useObservable(
        useMemo(
            () =>
                globalSearchContextSpec
                    ? // While we wait for the result of the `isSearchContextSpecAvailable` call, we assume the context is available
                      // to prevent flashing and moving content in the query bar. This optimizes for the most common use case where
                      // user selects a search context from the dropdown.
                      // See https://github.com/sourcegraph/sourcegraph/issues/19918 for more info.
                      isSearchContextSpecAvailable(globalSearchContextSpec.spec).pipe(startWith(true))
                    : of(false),
            [globalSearchContextSpec]
        )
    )

    useEffect(() => {
        // On a non-search related page or non-repo page, we clear the query in
        // the main query input to avoid misleading users
        // that the query is relevant in any way on those pages.
        if (!showSearchBox) {
            onNavbarQueryChange({ query: '' })
            return
        }
        // Do nothing if there is no query in the URL
        if (!query) {
            return
        }

        // If a global search context spec is available to the user, we omit it from the
        // query and move it to the search contexts dropdown
        const finalQuery =
            globalSearchContextSpec && isSearchContextAvailable && props.showSearchContext
                ? omitFilter(query, globalSearchContextSpec.filter)
                : query

        onNavbarQueryChange({ query: finalQuery })
    }, [
        showSearchBox,
        onNavbarQueryChange,
        query,
        globalSearchContextSpec,
        isSearchContextAvailable,
        props.showSearchContext,
    ])

    const [isRedesignEnabled] = useRedesignToggle()

    const settings = !isErrorLike(props.settingsCascade.final) ? props.settingsCascade.final : null
    const codeInsights =
        settings?.experimentalFeatures?.codeInsights && settings?.['insights.displayLocation.insightsPage'] !== false

    const logo = (
        <LinkOrSpan to={authRequired ? undefined : '/search'} className="global-navbar__logo-link">
            <BrandLogo
                branding={branding}
                isLightTheme={isLightTheme}
                variant="symbol"
                className="global-navbar__logo"
            />
        </LinkOrSpan>
    )
    const navLinks = !authRequired && !hideNavLinks && (
        <NavLinks
            showDotComMarketing={showDotComMarketing}
            minimalNavLinks={minimalNavLinks}
            location={location}
            history={history}
            isLightTheme={isLightTheme}
            isSourcegraphDotCom={isSourcegraphDotCom}
            {...props}
        />
    )

    const searchNavBar = (
        <SearchNavbarItem
            {...props}
            navbarSearchState={navbarSearchQueryState}
            onChange={onNavbarQueryChange}
            location={location}
            history={history}
            isLightTheme={isLightTheme}
            patternType={patternType}
            caseSensitive={caseSensitive}
            isSourcegraphDotCom={isSourcegraphDotCom}
        />
    )

    if (isRedesignEnabled) {
        return (
            <>
                <NavBar
                    logo={
                        <BrandLogo
                            branding={branding}
                            isLightTheme={isLightTheme}
                            variant="symbol"
                            className="global-navbar__logo"
                        />
                    }
                >
                    <NavGroup>
                        <NavItem icon={MagnifyIcon}>
                            <NavLink to="/search">Code Search</NavLink>
                        </NavItem>
                        {props.enableCodeMonitoring && (
                            <NavItem icon={CodeMonitoringLogo}>
                                <NavLink to="/code-monitoring">Monitoring</NavLink>
                            </NavItem>
                        )}
                        {props.showBatchChanges && <BatchChangesNavItem isSourcegraphDotCom={isSourcegraphDotCom} />}
                        {codeInsights && (
                            <NavItem icon={BarChartIcon}>
                                <NavLink to="/insights/dashboards/all">Insights</NavLink>
                            </NavItem>
                        )}
                        <NavItem icon={PuzzleOutlineIcon}>
                            <NavLink to="/extensions">Extensions</NavLink>
                        </NavItem>
                        {props.activation && (
                            <NavItem>
                                <ActivationDropdown activation={props.activation} history={history} />
                            </NavItem>
                        )}
                    </NavGroup>
                    <NavActions>
                        {!props.authenticatedUser && (
                            <>
                                {showDotComMarketing && (
                                    <NavAction>
                                        <Link
                                            className="global-navbar__link font-weight-medium"
                                            to="/help"
                                            target="_blank"
                                        >
                                            Docs
                                        </Link>
                                    </NavAction>
                                )}

                                <NavAction>
                                    <Link
                                        className="global-navbar__link"
                                        to="https://about.sourcegraph.com"
                                        rel="noreferrer noopener"
                                        target="_blank"
                                    >
                                        About
                                    </Link>
                                </NavAction>
                            </>
                        )}
                        {props.authenticatedUser && (
                            <NavAction>
                                <FeedbackPrompt routes={props.routes} />
                            </NavAction>
                        )}
                        <NavAction>
                            <WebCommandListPopoverButton
                                {...props}
                                location={location}
                                buttonClassName="btn btn-link p-0 m-0"
                                menu={ContributableMenu.CommandPalette}
                                keyboardShortcutForShow={KEYBOARD_SHORTCUT_SHOW_COMMAND_PALETTE}
                            />
                        </NavAction>
                        {props.authenticatedUser &&
                            (props.authenticatedUser.siteAdmin ||
                                userExternalServicesEnabledFromTags(props.authenticatedUser.tags)) && (
                                <NavAction>
                                    <StatusMessagesNavItem
                                        user={{
                                            id: props.authenticatedUser.id,
                                            username: props.authenticatedUser.username,
                                            isSiteAdmin: props.authenticatedUser?.siteAdmin || false,
                                        }}
                                        history={history}
                                    />
                                </NavAction>
                            )}
                        {!props.authenticatedUser ? (
                            <>
                                <NavAction>
                                    <div>
                                        <Link className="btn btn-sm btn-outline-secondary mr-1" to="/sign-in">
                                            Log in
                                        </Link>
                                        <Link
                                            className="btn btn-sm btn-outline-secondary global-navbar__sign-up"
                                            to="/sign-up"
                                        >
                                            Sign up
                                        </Link>
                                    </div>
                                </NavAction>
                            </>
                        ) : (
                            <NavAction>
                                <UserNavItem
                                    {...props}
                                    location={location}
                                    isLightTheme={isLightTheme}
                                    authenticatedUser={props.authenticatedUser}
                                    showDotComMarketing={showDotComMarketing}
                                    showRepositorySection={showRepositorySection}
                                    codeHostIntegrationMessaging={
                                        (!isErrorLike(props.settingsCascade.final) &&
                                            props.settingsCascade.final?.['alerts.codeHostIntegrationMessaging']) ||
                                        'browser-extension'
                                    }
                                    keyboardShortcutForSwitchTheme={KEYBOARD_SHORTCUT_SWITCH_THEME}
                                />
                            </NavAction>
                        )}
                    </NavActions>
                </NavBar>
                {showSearchBox && (
                    <div className="w-100 px-3 pt-2">
                        <div className="pb-2 border-bottom">{searchNavBar}</div>
                    </div>
                )}
            </>
        )
    }

    return (
        <div
            className={`global-navbar ${
                variant === 'low-profile' || variant === 'low-profile-with-logo'
                    ? ''
                    : 'global-navbar--bg border-bottom'
            } py-1`}
        >
            {variant === 'low-profile' || variant === 'low-profile-with-logo' ? (
                <>
                    {variant === 'low-profile-with-logo' && logo}
                    <div className="flex-1" />
                    {navLinks}
                </>
            ) : variant === 'no-search-input' ? (
                <>
                    {logo}
                    <div className="nav-item flex-1">
                        <Link to="/search" className="nav-link">
                            Search
                        </Link>
                    </div>
                    {navLinks}
                </>
            ) : (
                <>
                    {logo}
                    {authRequired ? (
                        <div className="flex-1" />
                    ) : (
                        <div className="global-navbar__search-box-container d-none d-sm-flex flex-row">
                            {searchNavBar}
                        </div>
                    )}
                    {navLinks}
                </>
            )}
        </div>
    )
}
