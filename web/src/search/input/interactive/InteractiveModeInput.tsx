import * as React from 'react'
import * as H from 'history'
import { QueryState, submitSearch } from '../../helpers'
import * as GQL from '../../../../../shared/src/graphql/schema'
import { Form } from '../../../components/Form'
import { AddFilterRow } from './AddFilterRow'
import { SelectedFiltersRow } from './SelectedFiltersRow'
import { SearchButton } from '../SearchButton'
import { ThemeProps } from '../../../../../shared/src/theme'
import { Link } from '../../../../../shared/src/components/Link'
import { NavLinks } from '../../../nav/NavLinks'
import { showDotComMarketing } from '../../../util/features'
import { SettingsCascadeProps } from '../../../../../shared/src/settings/settings'
import { KeyboardShortcutsProps, KEYBOARD_SHORTCUT_FOCUS_SEARCHBAR } from '../../../keyboardShortcuts/keyboardShortcuts'
import { ExtensionsControllerProps } from '../../../../../shared/src/extensions/controller'
import { PlatformContextProps } from '../../../../../shared/src/platform/context'
import { ThemePreferenceProps } from '../../../theme'
import { EventLoggerProps } from '../../../tracking/eventLogger'
import { ActivationProps } from '../../../../../shared/src/components/activation/Activation'
import { FiltersToTypeAndValue, FilterType } from '../../../../../shared/src/search/interactive/util'
import { QueryInput } from '../QueryInput'
import {
    parseSearchURLQuery,
    InteractiveSearchProps,
    PatternTypeProps,
    CaseSensitivityProps,
    CopyQueryButtonProps,
} from '../..'
import { SearchModeToggle } from './SearchModeToggle'
import { uniqueId } from 'lodash'
import { convertPlainTextToInteractiveQuery } from '../helpers'
import { isSingularFilter } from '../../../../../shared/src/search/parser/filters'
import { VersionContextDropdown } from '../../../nav/VersionContextDropdown'
import { VersionContextProps } from '../../../../../shared/src/search/util'
import { VersionContext } from '../../../schema/site.schema'
import { globbingEnabledFromSettings } from '../../../util/globbing'
import { OptionalAuthProps } from '../../../auth'

interface InteractiveModeProps
    extends SettingsCascadeProps,
        KeyboardShortcutsProps,
        ExtensionsControllerProps<'executeCommand' | 'services'>,
        PlatformContextProps<'forceUpdateTooltip' | 'settings'>,
        ThemeProps,
        ThemePreferenceProps,
        EventLoggerProps,
        ActivationProps,
        PatternTypeProps,
        CaseSensitivityProps,
        CopyQueryButtonProps,
        Pick<InteractiveSearchProps, 'filtersInQuery' | 'onFiltersInQueryChange' | 'toggleSearchMode'>,
        OptionalAuthProps,
        VersionContextProps {
    location: H.Location
    history: H.History
    navbarSearchState: QueryState
    onNavbarQueryChange: (userQuery: QueryState) => void

    /** Whether globbing is enabled for filters. */
    globbing: boolean

    /** Whether to hide the selected filters and add filter rows. */
    lowProfile: boolean

    // For NavLinks
    authRequired?: boolean
    showCampaigns: boolean
    isSourcegraphDotCom: boolean

    setVersionContext: (versionContext: string | undefined) => void
    availableVersionContexts: VersionContext[] | undefined

    /** Whether to display the interactive mode input centered on the page, as on the search homepage. */
    homepageMode?: boolean
}

interface InteractiveModeState {
    /** Count of the total number of filters ever added in this component.*/
    numFiltersAdded: number
}

export class InteractiveModeInput extends React.Component<InteractiveModeProps, InteractiveModeState> {
    constructor(props: InteractiveModeProps) {
        super(props)

        const searchParameters = new URLSearchParams(props.location.search)
        let filtersInQuery: FiltersToTypeAndValue = {}

        const query = searchParameters.get('q')
        if (query !== null && query.length > 0) {
            const { filtersInQuery: newFiltersInQuery, navbarQuery } = convertPlainTextToInteractiveQuery(query)
            filtersInQuery = { ...filtersInQuery, ...newFiltersInQuery }
            this.props.onNavbarQueryChange({ query: navbarQuery, cursorPosition: navbarQuery.length })
        }

        this.props.onFiltersInQueryChange(filtersInQuery)
    }

    /**
     * Adds a new filter to the top-level filtersInQuery state field.
     */
    private addNewFilter = (filterType: FilterType): void => {
        let filterKey: string = uniqueId(filterType)
        if (isSingularFilter(filterType)) {
            filterKey = filterType
            // Singular filters can only be specified at most once per query,
            // so we don't need to append a uniqueId.
            if (this.props.filtersInQuery[filterKey]) {
                // If the finite filter already exists in the query, just make the
                // existing one editable.
                const newFiltersInQuery = this.props.filtersInQuery
                newFiltersInQuery[filterKey].editable = true
                this.props.onFiltersInQueryChange(newFiltersInQuery)
                return
            }
        }

        this.props.onFiltersInQueryChange({
            ...this.props.filtersInQuery,
            [filterKey]: { type: filterType, value: '', editable: true },
        })
    }

    /**
     * onFilterEdited updates the top-level filtersInQuery object with new values
     * when new filter values are submitted by the user.
     *
     * Also conducts a new search with the updated query.
     */
    private onFilterEdited = (filterKey: string, value: string): void => {
        const newFiltersInQuery = {
            ...this.props.filtersInQuery,
            [filterKey]: {
                ...this.props.filtersInQuery[filterKey],
                value,
                editable: false,
            },
        }
        // Update the top-level filtersInQuery with the new values
        this.props.onFiltersInQueryChange(newFiltersInQuery)

        // Submit a search with the new values
        submitSearch({
            ...this.props,
            source: 'nav',
            query: this.props.navbarSearchState.query,
            filtersInQuery: newFiltersInQuery,
        })
    }

    private onFilterDeleted = (filterKey: string): void => {
        const filterWasEmpty =
            this.props.filtersInQuery[filterKey].value === '' || !this.props.filtersInQuery[filterKey]
        const newFiltersInQuery = { ...this.props.filtersInQuery }
        delete newFiltersInQuery[filterKey]

        this.props.onFiltersInQueryChange(newFiltersInQuery)

        if (!filterWasEmpty) {
            // Submit a search with the new values
            submitSearch({
                ...this.props,
                source: 'nav',
                query: this.props.navbarSearchState.query,
                filtersInQuery: newFiltersInQuery,
            })
        }
    }

    /**
     * toggleFilterEditable updates the top-level filtersInQuery object with
     * the new `editable` state of a single filter when its edit state is
     * being toggled.
     */
    private toggleFilterEditable = (filterKey: string): void => {
        this.props.onFiltersInQueryChange({
            ...this.props.filtersInQuery,
            [filterKey]: {
                ...this.props.filtersInQuery[filterKey],
                editable: !this.props.filtersInQuery[filterKey].editable,
            },
        })
    }

    private toggleFilterNegated = (filterKey: string): void => {
        this.props.onFiltersInQueryChange({
            ...this.props.filtersInQuery,
            [filterKey]: {
                ...this.props.filtersInQuery[filterKey],
                negated: !this.props.filtersInQuery[filterKey].negated,
            },
        })
    }

    private onSubmit = (event: React.FormEvent<HTMLFormElement>): void => {
        event.preventDefault()
        submitSearch({
            ...this.props,
            source: 'nav',
            query: this.props.navbarSearchState.query,
        })
    }

    public render(): JSX.Element | null {
        const homepageMode =
            this.props.homepageMode ||
            (this.props.location.pathname === '/search' && !parseSearchURLQuery(this.props.location.search))

        let logoSource = '/.assets/img/sourcegraph-mark.svg'
        let logoLinkClassName = 'global-navbar__logo-link global-navbar__logo-animated'

        const { branding } = window.context
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

        return (
            <div className="interactive-mode-input test-interactive-mode-input">
                <div className={!homepageMode ? 'interactive-mode-input__top-nav' : ''}>
                    {!homepageMode &&
                        (this.props.authRequired ? (
                            <div className={logoLinkClassName}>{logo}</div>
                        ) : (
                            <Link to="/search" className={logoLinkClassName}>
                                {logo}
                            </Link>
                        ))}
                    <div
                        className={`d-none d-sm-flex flex-row ${
                            !homepageMode ? 'interactive-mode-input__search-box-container' : ''
                        }`}
                    >
                        <Form onSubmit={this.onSubmit} className="flex-grow-1">
                            <div className="d-flex align-items-start">
                                <SearchModeToggle {...this.props} interactiveSearchMode={true} />
                                <VersionContextDropdown
                                    history={this.props.history}
                                    navbarSearchQuery={this.props.navbarSearchState.query}
                                    caseSensitive={this.props.caseSensitive}
                                    patternType={this.props.patternType}
                                    versionContext={this.props.versionContext}
                                    setVersionContext={this.props.setVersionContext}
                                    availableVersionContexts={this.props.availableVersionContexts}
                                />
                                <QueryInput
                                    {...this.props}
                                    location={this.props.location}
                                    history={this.props.history}
                                    value={this.props.navbarSearchState}
                                    hasGlobalQueryBehavior={true}
                                    onChange={this.props.onNavbarQueryChange}
                                    patternType={this.props.patternType}
                                    setPatternType={this.props.setPatternType}
                                    caseSensitive={this.props.caseSensitive}
                                    setCaseSensitivity={this.props.setCaseSensitivity}
                                    autoFocus={true}
                                    filtersInQuery={this.props.filtersInQuery}
                                    withoutSuggestions={true}
                                    withSearchModeToggle={true}
                                    keyboardShortcutForFocus={KEYBOARD_SHORTCUT_FOCUS_SEARCHBAR}
                                />
                                <SearchButton noHelp={true} />
                            </div>
                        </Form>
                    </div>
                    {!this.props.authRequired && !homepageMode && (
                        <NavLinks {...this.props} showDotComMarketing={showDotComMarketing} />
                    )}
                </div>
                {!this.props.lowProfile && (
                    <div>
                        <SelectedFiltersRow
                            globbing={globbingEnabledFromSettings(this.props.settingsCascade)}
                            filtersInQuery={this.props.filtersInQuery}
                            navbarQuery={this.props.navbarSearchState}
                            onSubmit={this.onSubmit}
                            onFilterEdited={this.onFilterEdited}
                            onFilterDeleted={this.onFilterDeleted}
                            toggleFilterEditable={this.toggleFilterEditable}
                            toggleFilterNegated={this.toggleFilterNegated}
                            isHomepage={homepageMode}
                        />
                        <AddFilterRow onAddNewFilter={this.addNewFilter} isHomepage={homepageMode} />
                    </div>
                )}
            </div>
        )
    }
}
