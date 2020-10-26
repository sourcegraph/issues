import { escapeRegExp } from 'lodash'
import { FiltersToTypeAndValue } from '../../../shared/src/search/interactive/util'
import { parseCaseSensitivityFromQuery, parsePatternTypeFromQuery } from '../../../shared/src/util/url'
import { replaceRange } from '../../../shared/src/util/strings'
import { discreteValueAliases } from '../../../shared/src/search/parser/filters'
import { VersionContext } from '../schema/site.schema'
import { SearchPatternType } from '../../../shared/src/graphql-operations'
import { Observable } from 'rxjs'
import { ISavedSearch } from '../../../shared/src/graphql/schema'
import { EventLogResult } from './backend'

/**
 * Parses the query out of the URL search params (the 'q' parameter). In non-interactive mode, if the 'q' parameter is not present, it
 * returns undefined. When parsing for interactive mode, each filter's individual query parameter
 * will be parsed and detected.
 *
 * @param query the URL query parameters
 * @param interactiveMode whether to parse the search URL query in interactive mode, reading query params such as `repo=` and `file=`.
 * @param navbarQueryOnly whether to only parse the query for the main query input, i.e. only the value passed to the `q=`
 * URL query parameter, as this represents the query that appears in the main query input in both modes.
 *
 */
export function parseSearchURLQuery(query: string): string | undefined {
    const searchParameters = new URLSearchParams(query)
    return searchParameters.get('q') || undefined
}

/**
 * Parses the pattern type out of the URL search params (the 'patternType' parameter). If the 'pattern' parameter
 * is not present, or it is an invalid value, it returns undefined.
 */
export function parseSearchURLPatternType(query: string): SearchPatternType | undefined {
    const searchParameters = new URLSearchParams(query)
    const patternType = searchParameters.get('patternType')
    if (
        patternType !== SearchPatternType.literal &&
        patternType !== SearchPatternType.regexp &&
        patternType !== SearchPatternType.structural
    ) {
        return undefined
    }
    return patternType
}

/**
 * Parses the version context out of the URL search params (the 'c' parameter). If the version context
 * is not present, return undefined.
 */
export function parseSearchURLVersionContext(query: string): string | undefined {
    const searchParameters = new URLSearchParams(query)
    const context = searchParameters.get('c')
    return context ?? undefined
}

export function searchURLIsCaseSensitive(query: string): boolean {
    const queryCaseSensitivity = parseCaseSensitivityFromQuery(query)
    if (queryCaseSensitivity) {
        // if `case:` filter exists in the query, override the existing case: query param
        return discreteValueAliases.yes.includes(queryCaseSensitivity.value)
    }
    const searchParameters = new URLSearchParams(query)
    const caseSensitive = searchParameters.get('case')
    return discreteValueAliases.yes.includes(caseSensitive || '')
}

/**
 * parseSearchURL takes a URL's search querystring and returns
 * an object containing:
 * - the canonical, user-visible query (with `patternType` and `case` filters excluded),
 * - the effective pattern type, and
 * - the effective case sensitivity of the query.
 *
 * @param urlSearchQuery a URL's query string.
 */
export function parseSearchURL(
    urlSearchQuery: string
): {
    query: string | undefined
    patternType: SearchPatternType | undefined
    caseSensitive: boolean
    versionContext: string | undefined
} {
    let finalQuery = parseSearchURLQuery(urlSearchQuery) || ''
    let patternType = parseSearchURLPatternType(urlSearchQuery)
    let caseSensitive = searchURLIsCaseSensitive(urlSearchQuery)

    const patternTypeInQuery = parsePatternTypeFromQuery(finalQuery)
    if (patternTypeInQuery) {
        // Any `patterntype:` filter in the query should override the patternType= URL query parameter if it exists.
        finalQuery = replaceRange(finalQuery, patternTypeInQuery.range)
        patternType = patternTypeInQuery.value as SearchPatternType
    }

    const caseInQuery = parseCaseSensitivityFromQuery(finalQuery)
    if (caseInQuery) {
        // Any `case:` filter in the query should override the case= URL query parameter if it exists.
        finalQuery = replaceRange(finalQuery, caseInQuery.range)

        if (discreteValueAliases.yes.includes(caseInQuery.value)) {
            caseSensitive = true
        } else if (discreteValueAliases.no.includes(caseInQuery.value)) {
            caseSensitive = false
        }
    }

    return {
        query: finalQuery,
        patternType,
        caseSensitive,
        versionContext: parseSearchURLVersionContext(urlSearchQuery),
    }
}

export function repoFilterForRepoRevision(repoName: string, globbing: boolean, revision?: string): string {
    if (globbing) {
        return `${quoteIfNeeded(`${repoName}${revision ? `@${abbreviateOID(revision)}` : ''}`)}`
    }
    return `${quoteIfNeeded(`^${escapeRegExp(repoName)}$${revision ? `@${abbreviateOID(revision)}` : ''}`)}`
}

export function searchQueryForRepoRevision(repoName: string, globbing: boolean, revision?: string): string {
    return `repo:${repoFilterForRepoRevision(repoName, globbing, revision)} `
}

function abbreviateOID(oid: string): string {
    if (oid.length === 40) {
        return oid.slice(0, 7)
    }
    return oid
}

export function quoteIfNeeded(string: string): string {
    if (/[ "']/.test(string)) {
        return JSON.stringify(string)
    }
    return string
}

export interface PatternTypeProps {
    patternType: SearchPatternType
    setPatternType: (patternType: SearchPatternType) => void
}

export interface CaseSensitivityProps {
    caseSensitive: boolean
    setCaseSensitivity: (caseSensitive: boolean) => void
}

export interface InteractiveSearchProps {
    filtersInQuery: FiltersToTypeAndValue
    onFiltersInQueryChange: (filtersInQuery: FiltersToTypeAndValue) => void
    splitSearchModes: boolean
    interactiveSearchMode: boolean
    toggleSearchMode: (event: React.MouseEvent<HTMLAnchorElement>) => void
}

export interface CopyQueryButtonProps {
    copyQueryButton: boolean
}

export interface RepogroupHomepageProps {
    showRepogroupHomepage: boolean
}

export interface OnboardingTourProps {
    showOnboardingTour: boolean
}

export interface ShowQueryBuilderProps {
    showQueryBuilder: boolean
}

export interface HomePanelsProps {
    showEnterpriseHomePanels: boolean
    fetchSavedSearches: () => Observable<ISavedSearch[]>
    fetchRecentSearches: (userId: string, first: number) => Observable<EventLogResult | null>
    fetchRecentFileViews: (userId: string, first: number) => Observable<EventLogResult | null>

    /** Function that returns current time (for stability in visual tests). */
    now?: () => Date
}

/**
 * Verifies whether a version context exists on an instance.
 *
 * For URLs that have a `c=$X` parameter, we must check that
 * the version $X actually exists before trying to search with it.
 *
 * If the version context doesn't exist or there are no available version contexts, return undefined to
 * use the default context.
 *
 * @param versionContext The version context to verify.
 * @param availableVersionContexts A list of all version contexts defined in site configuration.
 */
export function resolveVersionContext(
    versionContext: string | undefined,
    availableVersionContexts: VersionContext[] | undefined
): string | undefined {
    if (
        !versionContext ||
        !availableVersionContexts ||
        !availableVersionContexts.map(versionContext => versionContext.name).includes(versionContext) ||
        versionContext === 'default'
    ) {
        return undefined
    }

    return versionContext
}
