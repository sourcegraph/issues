import {
    FiltersToTypeAndValue,
    FilterType,
    isNegatedFilter,
    resolveNegatedFilter,
} from '../../../../shared/src/search/interactive/util'
import { parseSearchQuery } from '../../../../shared/src/search/parser/parser'
import { uniqueId } from 'lodash'
import { validateFilter, isSingularFilter } from '../../../../shared/src/search/parser/filters'

/**
 * Converts a plain text query into a an object containing the two components
 * of an interactive mode query:
 * - navbarQuery: any non-filter values in the query, which appears in the main query input
 * - filtersInQuery: an object containing key-value pairs of filters and their values
 *
 * @param query a plain text query.
 */
export function convertPlainTextToInteractiveQuery(
    query: string
): { filtersInQuery: FiltersToTypeAndValue; navbarQuery: string } {
    const parsedQuery = parseSearchQuery(query)

    const newFiltersInQuery: FiltersToTypeAndValue = {}
    let newNavbarQuery = ''

    if (parsedQuery.type === 'success') {
        for (const member of parsedQuery.token.members) {
            if (
                member.token.type === 'filter' &&
                member.token.filterValue &&
                validateFilter(member.token.filterType.token.value, member.token.filterValue).valid
            ) {
                const filterType = member.token.filterType.token.value as FilterType
                newFiltersInQuery[isSingularFilter(filterType) ? filterType : uniqueId(filterType)] = {
                    type: isNegatedFilter(filterType) ? resolveNegatedFilter(filterType) : filterType,
                    value: query.slice(member.token.filterValue.range.start, member.token.filterValue.range.end),
                    editable: false,
                    negated: isNegatedFilter(filterType),
                }
            } else if (
                member.token.type !== 'filter' ||
                (member.token.type === 'filter' &&
                    !validateFilter(member.token.filterType.token.value, member.token.filterValue).valid)
            ) {
                newNavbarQuery = [newNavbarQuery, query.slice(member.range.start, member.range.end)]
                    .filter(query => query.length > 0)
                    .join('')
            }
        }
    }

    return { filtersInQuery: newFiltersInQuery, navbarQuery: newNavbarQuery.trim() }
}
