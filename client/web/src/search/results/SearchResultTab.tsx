import * as React from 'react'
import * as H from 'history'
import { SearchType } from './SearchResults'
import { NavLink } from 'react-router-dom'
import { toggleSearchType } from '../helpers'
import { buildSearchURLQuery, generateFiltersQuery } from '../../../../shared/src/util/url'
import { constant } from 'lodash'
import { PatternTypeProps, CaseSensitivityProps, parseSearchURLQuery, InteractiveSearchProps } from '..'
import { parseSearchQuery } from '../../../../shared/src/search/parser/parser'
import { VersionContextProps } from '../../../../shared/src/search/util'

interface Props
    extends Omit<PatternTypeProps, 'setPatternType'>,
        Omit<CaseSensitivityProps, 'setCaseSensitivity'>,
        Pick<InteractiveSearchProps, 'filtersInQuery'>,
        VersionContextProps {
    location: H.Location
    type: SearchType
    query: string
}

const typeToProse: Record<Exclude<SearchType, null>, string> = {
    diff: 'Diffs',
    commit: 'Commits',
    symbol: 'Symbols',
    repo: 'Repositories',
    path: 'Filenames',
}

export const SearchResultTabHeader: React.FunctionComponent<Props> = ({
    location,
    type,
    query,
    filtersInQuery,
    patternType,
    caseSensitive,
    versionContext,
}) => {
    const fullQuery = [query, generateFiltersQuery(filtersInQuery)].filter(query => query.length > 0).join(' ')
    const caseToggledQuery = toggleSearchType(fullQuery, type)
    const builtURLQuery = buildSearchURLQuery(caseToggledQuery, patternType, caseSensitive, versionContext)

    const currentQuery = parseSearchURLQuery(location.search) || ''
    const parsedQuery = parseSearchQuery(currentQuery)
    let typeInQuery: SearchType = null

    if (parsedQuery.type === 'success') {
        // Parse any `type:` filter that exists in a query so
        // we can check whether this tab should be active.
        for (const member of parsedQuery.token.members) {
            if (
                member.token.type === 'filter' &&
                member.token.filterType.token.value === 'type' &&
                member.token.filterValue
            ) {
                typeInQuery =
                    member.token.filterValue.token.type === 'literal'
                        ? (member.token.filterValue.token.value as SearchType)
                        : (member.token.filterValue.token.quotedValue as SearchType)
            }
        }
    }

    const isActiveFunc = constant(typeInQuery === type)
    return (
        <li className="nav-item test-search-result-tab">
            <NavLink
                to={{ pathname: '/search', search: builtURLQuery }}
                className={`nav-link test-search-result-tab-${String(type)}`}
                activeClassName="active test-search-result-tab--active"
                isActive={isActiveFunc}
            >
                {type ? typeToProse[type] : 'Code'}
            </NavLink>
        </li>
    )
}
