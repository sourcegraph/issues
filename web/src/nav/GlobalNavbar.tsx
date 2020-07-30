import * as H from 'history'
import * as React from 'react'
import { Subscription } from 'rxjs'
import { ActivationProps } from '../../../shared/src/components/activation/Activation'
import { ExtensionsControllerProps } from '../../../shared/src/extensions/controller'
import { PlatformContextProps } from '../../../shared/src/platform/context'
import { SettingsCascadeProps } from '../../../shared/src/settings/settings'
import { authRequired, OptionalAuthProps } from '../auth'
import {
    parseSearchURLQuery,
    PatternTypeProps,
    InteractiveSearchProps,
    CaseSensitivityProps,
    SmartSearchFieldProps,
    CopyQueryButtonProps,
} from '../search'
import { SearchNavbarItem } from '../search/input/SearchNavbarItem'
import { EventLoggerProps } from '../tracking/eventLogger'
import { showDotComMarketing } from '../util/features'
import { NavLinks } from './NavLinks'
import { ThemeProps } from '../../../shared/src/theme'
import { ThemePreferenceProps } from '../theme'
import { KeyboardShortcutsProps } from '../keyboardShortcuts/keyboardShortcuts'
import { QueryState } from '../search/helpers'
import { InteractiveModeInput } from '../search/input/interactive/InteractiveModeInput'
import { FiltersToTypeAndValue } from '../../../shared/src/search/interactive/util'
import { SearchModeToggle } from '../search/input/interactive/SearchModeToggle'
import { Link } from '../../../shared/src/components/Link'
import { convertPlainTextToInteractiveQuery } from '../search/input/helpers'
import { VersionContextDropdown } from './VersionContextDropdown'
import { VersionContextProps } from '../../../shared/src/search/util'
import { VersionContext } from '../schema/site.schema'

interface Props
    extends SettingsCascadeProps,
        PlatformContextProps,
        ExtensionsControllerProps,
        KeyboardShortcutsProps,
        EventLoggerProps,
        ThemeProps,
        ThemePreferenceProps,
        ActivationProps,
        PatternTypeProps,
        CaseSensitivityProps,
        InteractiveSearchProps,
        SmartSearchFieldProps,
        CopyQueryButtonProps,
        VersionContextProps,
        OptionalAuthProps {
    history: H.History
    location: H.Location<{ query: string }>
    navbarSearchQueryState: QueryState
    onNavbarQueryChange: (queryState: QueryState) => void
    isSourcegraphDotCom: boolean
    isSearchRelatedPage: boolean
    showCampaigns: boolean

    // Whether globbing is enabled for filters.
    globbing: boolean

    /**
     * Which variation of the global navbar to render.
     *
     * 'low-profile' renders the the navbar with no border or background. Used on the search
     * homepage.
     *
     * 'low-profile-with-logo' renders the low-profile navbar but with the homepage logo. Used on repogroup pages.
     */
    variant: 'default' | 'low-profile' | 'low-profile-with-logo' | 'no-search-input'

    splitSearchModes: boolean
    interactiveSearchMode: boolean
    toggleSearchMode: (event: React.MouseEvent<HTMLAnchorElement>) => void
    setVersionContext: (versionContext: string | undefined) => void
    availableVersionContexts: VersionContext[] | undefined

    /** For testing only. Used because reactstrap's Popover is incompatible with react-test-renderer. */
    hideNavLinks: boolean
}

interface State {
    authRequired?: boolean
}

export class GlobalNavbar extends React.PureComponent<Props, State> {
    public state: State = {}

    private subscriptions = new Subscription()

    constructor(props: Props) {
        super(props)

        // In interactive search mode, the InteractiveModeInput component will handle updating the inputs.
        if (!props.interactiveSearchMode) {
            // Reads initial state from the props (i.e. URL parameters).
            const query = parseSearchURLQuery(props.location.search || '')
            if (query) {
                props.onNavbarQueryChange({ query, cursorPosition: query.length })
            } else {
                // If we have no component state, then we may have gotten unmounted during a route change.
                const query = props.location.state ? props.location.state.query : ''
                props.onNavbarQueryChange({
                    query,
                    cursorPosition: query.length,
                })
            }
        }
    }

    public componentDidMount(): void {
        this.subscriptions.add(authRequired.subscribe(authRequired => this.setState({ authRequired })))
    }

    public componentDidUpdate(previousProps: Props): void {
        if (previousProps.location !== this.props.location) {
            if (!this.props.isSearchRelatedPage) {
                // On a non-search related page or non-repo page, we clear the query in
                // the main query input and interactive mode UI to avoid misleading users
                // that the query is relevant in any way on those pages.
                this.props.onNavbarQueryChange({ query: '', cursorPosition: 0 })
                this.props.onFiltersInQueryChange({})
            }
        }

        if (previousProps.location.search !== this.props.location.search) {
            const query = parseSearchURLQuery(this.props.location.search || '')
            if (query) {
                if (this.props.interactiveSearchMode) {
                    let filtersInQuery: FiltersToTypeAndValue = {}
                    const { filtersInQuery: newFiltersInQuery, navbarQuery } = convertPlainTextToInteractiveQuery(query)
                    filtersInQuery = { ...filtersInQuery, ...newFiltersInQuery }
                    this.props.onNavbarQueryChange({ query: navbarQuery, cursorPosition: navbarQuery.length })

                    this.props.onFiltersInQueryChange(filtersInQuery)
                } else {
                    this.props.onNavbarQueryChange({ query, cursorPosition: query.length })
                }
            }
        }
    }

    public componentWillUnmount(): void {
        this.subscriptions.unsubscribe()
    }

    public render(): JSX.Element | null {
        let logoSource = '/.assets/img/sourcegraph-mark.svg'
        let logoLinkClassName = 'global-navbar__logo-link global-navbar__logo-animated'
        const logoWithNameSource = '/.assets/img/sourcegraph-head-logo.svg'
        const logoWithNameLightSource = '/.assets/img/sourcegraph-light-head-logo.svg'

        const branding = window.context ? window.context.branding : null
        if (branding) {
            if (this.props.isLightTheme) {
                if (branding.light?.symbol) {
                    logoSource = branding.light.symbol
                }
            } else if (branding.dark?.symbol) {
                logoSource = branding.dark.symbol
            }
            if (branding.disableSymbolSpin) {
                logoLinkClassName = 'global-navbar__logo-link'
            }
        }

        const logo = <img className="global-navbar__logo" src={logoSource} />
        const logoWithNameLink = (
            <Link to="/search">
                <img
                    className="global-navbar__logo-with-name pl-2"
                    src={this.props.isLightTheme ? logoWithNameLightSource : logoWithNameSource}
                />
            </Link>
        )

        const logoLink = !this.state.authRequired ? (
            <Link to="/search" className={logoLinkClassName}>
                {logo}
            </Link>
        ) : (
            <div className={logoLinkClassName}>{logo}</div>
        )
        const navLinks = !this.state.authRequired && !this.props.hideNavLinks && (
            <NavLinks {...this.props} showDotComMarketing={showDotComMarketing} />
        )

        return (
            <div
                className={`global-navbar ${
                    this.props.variant === 'low-profile' || this.props.variant === 'low-profile-with-logo'
                        ? ''
                        : 'global-navbar--bg border-bottom'
                } py-1`}
            >
                {this.props.variant === 'low-profile' || this.props.variant === 'low-profile-with-logo' ? (
                    <>
                        {this.props.variant === 'low-profile-with-logo' && (
                            <div className="nav-item flex-1">{logoWithNameLink}</div>
                        )}
                        <div className="flex-1" />
                        {!this.state.authRequired && !this.props.hideNavLinks && (
                            <NavLinks {...this.props} showDotComMarketing={showDotComMarketing} />
                        )}
                    </>
                ) : this.props.variant === 'no-search-input' ? (
                    <>
                        {logoLink}
                        <div className="nav-item flex-1">
                            <Link to="/search" className="nav-link">
                                Search
                            </Link>
                        </div>
                        {navLinks}
                    </>
                ) : (
                    <>
                        {this.props.splitSearchModes && this.props.interactiveSearchMode ? (
                            !this.state.authRequired && (
                                <InteractiveModeInput
                                    {...this.props}
                                    authRequired={this.state.authRequired}
                                    navbarSearchState={this.props.navbarSearchQueryState}
                                    onNavbarQueryChange={this.props.onNavbarQueryChange}
                                    lowProfile={!this.props.isSearchRelatedPage}
                                    versionContext={this.props.versionContext}
                                />
                            )
                        ) : (
                            <>
                                {logoLink}
                                {!this.state.authRequired && (
                                    <div className="global-navbar__search-box-container d-none d-sm-flex flex-row">
                                        {this.props.splitSearchModes && (
                                            <SearchModeToggle
                                                {...this.props}
                                                interactiveSearchMode={this.props.interactiveSearchMode}
                                            />
                                        )}
                                        <VersionContextDropdown
                                            history={this.props.history}
                                            navbarSearchQuery={this.props.navbarSearchQueryState.query}
                                            caseSensitive={this.props.caseSensitive}
                                            patternType={this.props.patternType}
                                            versionContext={this.props.versionContext}
                                            setVersionContext={this.props.setVersionContext}
                                            availableVersionContexts={this.props.availableVersionContexts}
                                        />
                                        <SearchNavbarItem
                                            {...this.props}
                                            navbarSearchState={this.props.navbarSearchQueryState}
                                            onChange={this.props.onNavbarQueryChange}
                                            smartSearchField={this.props.smartSearchField}
                                        />
                                    </div>
                                )}
                                {navLinks}
                            </>
                        )}
                    </>
                )}
            </div>
        )
    }
}
