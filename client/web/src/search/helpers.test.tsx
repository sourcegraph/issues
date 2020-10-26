import * as H from 'history'
import {
    getSearchTypeFromQuery,
    toggleSearchType,
    filterStaticSuggestions,
    insertSuggestionInQuery,
    validFilterAndValueBeforeCursor,
    isFuzzyWordSearch,
    formatQueryForFuzzySearch,
    filterAliasForSearch,
    toggleSearchFilter,
    submitSearch,
} from './helpers'
import { SearchType } from './results/SearchResults'
import { searchFilterSuggestions } from './searchFilterSuggestions'
import { filterAliases, isolatedFuzzySearchFilters } from './input/Suggestion'
import assert from 'assert'
import { SearchPatternType } from '../../../shared/src/graphql/schema'

jest.mock('../tracking/eventLogger', () => ({ eventLogger: { log: () => undefined } }))

describe('search/helpers', () => {
    describe('queryIndexOfScope()', () => {
        test.skip('should return the index of a scope if contained in the query', () => {
            /* noop */
        })
        test.skip('should return the index of a scope if at the beginning of the query', () => {
            /* noop */
        })
        test.skip('should return the -1 if the scope is not contained in the query', () => {
            /* noop */
        })
        test.skip('should return the -1 if the scope is contained as a substring of another scope', () => {
            /* noop */
        })
    })

    describe('submitSearch()', () => {
        test('should update history', () => {
            const history = H.createMemoryHistory({})
            submitSearch({
                history,
                query: 'querystring',
                patternType: SearchPatternType.literal,
                caseSensitive: false,
                versionContext: undefined,
                activation: undefined,
                filtersInQuery: undefined,
                source: 'home',
                searchParameters: undefined,
            })
            expect(history.location.search).toEqual('?q=querystring&patternType=literal')
        })
        test('should keep trace param when updating history', () => {
            const history = H.createMemoryHistory({ initialEntries: ['/?trace=1'] })
            submitSearch({
                history,
                query: 'querystring',
                patternType: SearchPatternType.literal,
                caseSensitive: false,
                versionContext: undefined,
                activation: undefined,
                filtersInQuery: undefined,
                source: 'home',
                searchParameters: undefined,
            })
            expect(history.location.search).toEqual('?q=querystring&patternType=literal&trace=1')
        })
    })

    const searchTypes: NonNullable<SearchType>[] = ['diff', 'commit', 'symbol', 'repo', 'path']

    describe('getSearchTypeFromQuery()', () => {
        test('parses the search type in simple queries', () => {
            for (const searchType of searchTypes) {
                expect(getSearchTypeFromQuery(`type:${searchType}`)).toEqual(searchType)
            }
        })

        test('returns null when no search type specified', () => {
            expect(getSearchTypeFromQuery('code')).toEqual(null)
        })

        test('parses the search type in complex queries', () => {
            expect(getSearchTypeFromQuery('test type:diff')).toEqual('diff')
            expect(getSearchTypeFromQuery('type:diff test')).toEqual('diff')
            expect(getSearchTypeFromQuery('repo:^github.com/sourcegraph/sourcegraph type:diff test')).toEqual('diff')
            expect(getSearchTypeFromQuery('type:diff repo:^github.com/sourcegraph/sourcegraph test')).toEqual('diff')
        })

        test('returns symbol when multiple search types, including symbol, are specified', () => {
            // Edge case. If there are multiple type filters and `type:symbol` is one of them, symbol results always get returned.
            expect(
                getSearchTypeFromQuery('type:diff type:symbol repo:^github.com/sourcegraph/sourcegraph test')
            ).toEqual('symbol')
        })

        test('returns the first search type specified when multiple search types, not including symbol, are specified', () => {
            expect(
                getSearchTypeFromQuery('type:diff type:commit repo:^github.com/sourcegraph/sourcegraph test')
            ).toEqual('diff')
        })
    })

    describe('toggleSearchType()', () => {
        test('returns the original query when the query already contains the correct type', () => {
            expect(toggleSearchType('test', null)).toEqual('test')

            for (const searchType of searchTypes) {
                expect(toggleSearchType(`test type:${searchType}`, searchType)).toEqual(`test type:${searchType}`)
            }
        })

        test('appends type:$TYPE to the query when no type exists in the query', () => {
            for (const searchType of searchTypes) {
                expect(toggleSearchType('test', searchType)).toEqual(`test type:${searchType}`)
            }
        })

        test('replaces existing type in query with new type in simple queries', () => {
            expect(toggleSearchType('test type:commit', 'diff')).toEqual('test type:diff')
            expect(toggleSearchType('type:diff test', 'commit')).toEqual('type:commit test')
        })

        test('replaces existing type in query with new type in complex queries', () => {
            expect(toggleSearchType('test type:symbol repo:^sourcegraph/test', 'diff')).toEqual(
                'test type:diff repo:^sourcegraph/test'
            )
            expect(toggleSearchType('test type:symbol repo:^sourcegraph/test', null)).toEqual(
                'test  repo:^sourcegraph/test'
            )
        })

        test('replaces the first type in query with new type in queries with multiple type fields', () => {
            expect(toggleSearchType('test type:symbol type:commit repo:^sourcegraph/test', 'diff')).toEqual(
                'test type:diff type:commit repo:^sourcegraph/test'
            )
        })
    })

    describe('suggestions', () => {
        const filterQuery = 'test r test'

        const getArchivedSuggestions = () =>
            filterStaticSuggestions({ query: 'archived:', cursorPosition: 9 }, searchFilterSuggestions)
        const getFilterSuggestionStartingWithR = () =>
            filterStaticSuggestions({ query: filterQuery, cursorPosition: 6 }, searchFilterSuggestions)

        describe(`${filterStaticSuggestions.name}()`, () => {
            test('filters suggestions for filters starting with "r"', () => {
                const filtersStartingWithR = searchFilterSuggestions.filters.values
                    .map(filter => filter.value)
                    .filter(filter => filter.startsWith('r'))
                expect(getFilterSuggestionStartingWithR().map(suggestion => suggestion.value)).toEqual(
                    expect.arrayContaining(filtersStartingWithR)
                )
            })

            test('filters suggestions for filter aliases', () => {
                const [{ value }] = filterStaticSuggestions({ query: 'l', cursorPosition: 1 }, searchFilterSuggestions)
                expect(value).toBe(`${filterAliases.l}:`)
            })

            test('does not throw for query ":"', () => {
                expect(() =>
                    filterStaticSuggestions({ query: ':', cursorPosition: 1 }, searchFilterSuggestions)
                ).not.toThrowError()
            })

            test('filters suggestions for word "test"', () => {
                expect(
                    filterStaticSuggestions({ query: filterQuery, cursorPosition: 4 }, searchFilterSuggestions)
                ).toHaveLength(0)
            })

            test('filters suggestions for the "archived:" filter', () => {
                const archivedSuggestions = getArchivedSuggestions()
                expect(archivedSuggestions).toEqual(expect.arrayContaining(searchFilterSuggestions.archived.values))
            })
        })

        describe(`${insertSuggestionInQuery.name}()`, () => {
            describe('inserts suggestions for a filter name', () => {
                const suggestion = getFilterSuggestionStartingWithR().find(({ value }) => value === 'repo:')
                assert(suggestion)
                const { query: newQuery } = insertSuggestionInQuery('test r test', suggestion, 6)
                expect(newQuery).toBe(`test ${suggestion.value} test`)
            })
            test('inserts suggestion for a filter value', () => {
                const [suggestion] = getArchivedSuggestions()
                const { query: newQuery } = insertSuggestionInQuery('test archived: test', suggestion, 14)
                expect(newQuery).toBe(`test archived:${suggestion.value} test`)
            })
        })
    })

    describe(`${validFilterAndValueBeforeCursor.name}()`, () => {
        const query = 'archived:yes QueryInput'
        it('returns values when a filter value is being typed', () => {
            expect(validFilterAndValueBeforeCursor({ query, cursorPosition: 10 })).toEqual({
                filterIndex: 0,
                filterAndValue: 'archived:y',
                matchedFilter: 'archived',
                resolvedFilterType: 'archived',
                value: 'y',
            })
        })
        it('returns values when a filter is selected but no value char is typed yet', () => {
            expect(validFilterAndValueBeforeCursor({ query, cursorPosition: 9 })).toEqual({
                filterIndex: 0,
                filterAndValue: 'archived:',
                matchedFilter: 'archived',
                resolvedFilterType: 'archived',
                value: '',
            })
        })
        it('does not return a value if typed whitespace char', () => {
            expect(validFilterAndValueBeforeCursor({ query, cursorPosition: 13 })).toStrictEqual(null)
        })
        it('correctly resolves filter aliases', () => {
            const query = 'l:go'
            expect(validFilterAndValueBeforeCursor({ query, cursorPosition: query.length })).toEqual({
                filterIndex: 0,
                filterAndValue: query,
                matchedFilter: 'l',
                resolvedFilterType: 'lang',
                value: 'go',
            })
        })
        it('correctly formats negated filters', () => {
            const query = '-f:package.json'
            expect(validFilterAndValueBeforeCursor({ query, cursorPosition: query.length })).toEqual({
                filterIndex: 0,
                filterAndValue: query,
                matchedFilter: '-f',
                resolvedFilterType: 'file',
                value: 'package.json',
            })
        })
        it('returns correct values for filter query without a value', () => {
            const query = '-f'
            expect(validFilterAndValueBeforeCursor({ query, cursorPosition: query.length })).toEqual({
                filterIndex: 0,
                filterAndValue: query,
                matchedFilter: query,
                resolvedFilterType: 'file',
                value: '',
            })
        })
    })

    describe('isFuzzyWordSearch', () => {
        const query = 'Query lang:g'
        it('returns false if typing a filter value', () =>
            expect(isFuzzyWordSearch({ query, cursorPosition: 12 })).toBe(false))
        it('returns true if typing a non filter type or value', () =>
            expect(isFuzzyWordSearch({ query, cursorPosition: 5 })).toBe(true))
    })

    describe('formatQueryForFuzzySearch', () => {
        const formatForSearchWithFilter = (filter: string) =>
            formatQueryForFuzzySearch({
                query: `archived:Yes ${filter}:value Props`,
                // 19 is position until after ':value'
                cursorPosition: 19 + filter.length,
            })

        it('isolates filters that are in isolatedFuzzySearchFilters', () => {
            expect(isolatedFuzzySearchFilters.map(formatForSearchWithFilter)).toStrictEqual(
                isolatedFuzzySearchFilters.map(filterType => filterType + ':value')
            )
        })
        it('replaces filter being typed with its `filterAliasForSearch`', () => {
            expect(Object.keys(filterAliasForSearch).map(formatForSearchWithFilter)).toStrictEqual(
                Object.values(filterAliasForSearch).map(alias => `archived:Yes ${alias!}:value Props`)
            )
        })
        it('return absolute filter if filter being typed is negated (e.g: `-file`)', () => {
            expect(
                formatQueryForFuzzySearch({
                    query: 'l:javascript -file:index.js archived:No',
                    cursorPosition: 27,
                })
            ).toBe('l:javascript file:index.js archived:No')
        })
    })

    describe('toggleSearchFilter', () => {
        it('adds filter if it is not already in query', () => {
            expect(toggleSearchFilter('repo:test ', 'lang:c++')).toStrictEqual('repo:test lang:c++ ')
        })

        it('adds filter if it is not already in query, even if it matches substring for an existing filter', () => {
            expect(toggleSearchFilter('repo:test lang:c++ ', 'lang:c')).toStrictEqual('repo:test lang:c++ lang:c ')
        })

        it('removes filter from query it it exists', () => {
            expect(toggleSearchFilter('repo:test lang:c++ lang:c ', 'lang:c')).toStrictEqual('repo:test lang:c++')
        })
    })
})
