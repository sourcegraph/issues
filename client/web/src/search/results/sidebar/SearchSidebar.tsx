import classNames from 'classnames'
import React, { useCallback } from 'react'
import { useHistory } from 'react-router'
import StickyBox from 'react-sticky-box'

import { Filter } from '@sourcegraph/shared/src/search/stream'
import { VersionContextProps } from '@sourcegraph/shared/src/search/util'
import { SettingsCascadeProps } from '@sourcegraph/shared/src/settings/settings'
import { TelemetryProps } from '@sourcegraph/shared/src/telemetry/telemetryService'
import { useLocalStorage } from '@sourcegraph/shared/src/util/useLocalStorage'

import { CaseSensitivityProps, PatternTypeProps, SearchContextProps } from '../..'
import { AuthenticatedUser } from '../../../auth'
import { FeatureFlagProps } from '../../../featureFlags/featureFlags'
import { QueryState, submitSearch, toggleSearchFilter } from '../../helpers'

import { getDynamicFilterLinks, getRepoFilterLinks, getSearchSnippetLinks } from './FilterLink'
import { getQuickLinks } from './QuickLink'
import { getSearchReferenceFactory } from './SearchReference'
import styles from './SearchSidebar.module.scss'
import { SearchSidebarSection } from './SearchSidebarSection'
import { getSearchTypeLinks } from './SearchTypeLink'

const SEARCH_SIDEBAR_VISIBILITY_KEY = 'SearchProduct.SearchSidebar.Visibility'

export interface SearchSidebarProps
    extends Omit<PatternTypeProps, 'setPatternType'>,
        Omit<CaseSensitivityProps, 'setCaseSensitivity'>,
        VersionContextProps,
        Pick<SearchContextProps, 'selectedSearchContextSpec'>,
        SettingsCascadeProps,
        TelemetryProps,
        FeatureFlagProps {
    authenticatedUser: AuthenticatedUser | null
    query: string
    filters?: Filter[]
    className?: string
    navbarSearchQueryState: QueryState
    onNavbarQueryChange: (queryState: QueryState) => void
    isSourcegraphDotCom: boolean
}

enum SectionID {
    SEARCH_REFERENCE,
    SEARCH_TYPES,
    DYNAMIC_FILTERS,
    REPOSITORIES,
    SEARCH_SNIPPETS,
    QUICK_LINKS,
}

export const SearchSidebar: React.FunctionComponent<SearchSidebarProps> = props => {
    const history = useHistory()
    const [openSections, setOpenSections] = useLocalStorage<{ [key in SectionID]?: boolean }>(
        SEARCH_SIDEBAR_VISIBILITY_KEY,
        {}
    )

    const onFilterClicked = useCallback(
        (value: string) => {
            const newQuery = toggleSearchFilter(props.query, value)
            submitSearch({ ...props, query: newQuery, source: 'filter', history })
        },
        [history, props]
    )

    const onDynamicFilterClicked = useCallback(
        (value: string) => {
            props.telemetryService.log('DynamicFilterClicked', {
                search_filter: { value },
            })

            onFilterClicked(value)
        },
        [onFilterClicked, props.telemetryService]
    )

    const onSnippetClicked = useCallback(
        (value: string) => {
            props.telemetryService.log('SearchSnippetClicked')
            onFilterClicked(value)
        },
        [onFilterClicked, props.telemetryService]
    )

    const persistToggleState = useCallback(
        (id: SectionID, open: boolean) => {
            setOpenSections(openSections => ({ ...openSections, [id]: open }))
        },
        [setOpenSections]
    )
    const onSearchReferenceToggle = useCallback(
        open => {
            persistToggleState(SectionID.SEARCH_REFERENCE, open)
            props.telemetryService.log(open ? 'SearchReferenceOpened' : 'SearchReferenceClosed')
        },
        [persistToggleState, props.telemetryService]
    )

    return (
        <div className={classNames(styles.searchSidebar, props.className)}>
            <StickyBox className={styles.searchSidebarStickyBox}>
                {props.featureFlags.get('search-reference') && (
                    <SearchSidebarSection
                        className={styles.searchSidebarItem}
                        header="Search reference"
                        showSearch={true}
                        open={openSections[SectionID.SEARCH_REFERENCE] ?? true}
                        onToggle={onSearchReferenceToggle}
                    >
                        {getSearchReferenceFactory(props)}
                    </SearchSidebarSection>
                )}
                {!props.featureFlags.get('search-reference') && (
                    <SearchSidebarSection
                        className={styles.searchSidebarItem}
                        header="Search types"
                        open={openSections[SectionID.SEARCH_TYPES] ?? true}
                        onToggle={open => persistToggleState(SectionID.SEARCH_TYPES, open)}
                    >
                        {getSearchTypeLinks(props)}
                    </SearchSidebarSection>
                )}
                <SearchSidebarSection
                    className={styles.searchSidebarItem}
                    header="Dynamic filters"
                    open={openSections[SectionID.DYNAMIC_FILTERS] ?? true}
                    onToggle={open => persistToggleState(SectionID.DYNAMIC_FILTERS, open)}
                >
                    {getDynamicFilterLinks(props.filters, onDynamicFilterClicked)}
                </SearchSidebarSection>
                <SearchSidebarSection
                    className={styles.searchSidebarItem}
                    header="Repositories"
                    open={openSections[SectionID.REPOSITORIES] ?? true}
                    onToggle={open => persistToggleState(SectionID.REPOSITORIES, open)}
                    showSearch={true}
                >
                    {getRepoFilterLinks(props.filters, onDynamicFilterClicked)}
                </SearchSidebarSection>
                <SearchSidebarSection
                    className={styles.searchSidebarItem}
                    header="Search snippets"
                    open={openSections[SectionID.REPOSITORIES] ?? true}
                    onToggle={open => persistToggleState(SectionID.REPOSITORIES, open)}
                >
                    {getSearchSnippetLinks(props.settingsCascade, onSnippetClicked)}
                </SearchSidebarSection>
                <SearchSidebarSection
                    className={styles.searchSidebarItem}
                    header="Quicklinks"
                    open={openSections[SectionID.QUICK_LINKS] ?? true}
                    onToggle={open => persistToggleState(SectionID.QUICK_LINKS, open)}
                >
                    {getQuickLinks(props.settingsCascade)}
                </SearchSidebarSection>
            </StickyBox>
        </div>
    )
}
