import { assert } from 'chai'
import { Range, Selection } from 'monaco-editor'

import { FILTERS, FilterType } from '@sourcegraph/shared/src/search/query/filters'

import { QueryChangeSource, QueryState } from '../../helpers'

import { FilterInfo, parsePlaceholder, updateQueryWithFilter } from './SearchReference'

/**
 * Automatically sets cursor position and selections from example query.
 */
function queryStateFromExample(query: string, showSuggestions = false): QueryState {
    const positions: { [char: string]: number } = {}
    const magicCharacters = {
        selectionStart: '[',
        selectionEnd: ']',
        rangeStart: '{',
        rangeEnd: '}',
    }
    let offset = 0

    const cleanedQuery = query.replaceAll(/[[\]{|}]/g, (match, index: number) => {
        positions[match] = index + offset + 1
        offset -= 1
        return ''
    })

    const { [magicCharacters.selectionStart]: selectionStart } = positions

    const selection = new Selection(1, selectionStart, 1, positions[magicCharacters.selectionEnd])

    return {
        query: cleanedQuery,
        changeSource: QueryChangeSource.searchReference,
        selection,
        showSuggestions,
        revealRange: new Range(1, positions[magicCharacters.rangeStart], 1, positions[magicCharacters.rangeEnd]),
    }
}

function createFilterInfo(type: FilterType, placeholder: string): FilterInfo {
    return {
        type,
        placeholder: parsePlaceholder(placeholder),
        description: '',
    }
}

describe('repeatable filters', () => {
    it('appends placeholder filter and selects placeholder', () => {
        assert.deepStrictEqual(
            updateQueryWithFilter({ query: 'foo' }, createFilterInfo(FilterType.after, '{test}'), false, FILTERS),
            queryStateFromExample('foo {after:[test]}')
        )
    })

    it('appends suggestions filter', () => {
        assert.deepStrictEqual(
            updateQueryWithFilter({ query: 'foo' }, createFilterInfo(FilterType.lang, '{lang}'), false, FILTERS),
            queryStateFromExample('foo {lang:[]}', true)
        )
    })
})

describe('unique filters', () => {
    it('appends placeholder filter and selects placeholder', () => {
        assert.deepStrictEqual(
            updateQueryWithFilter({ query: 'foo' }, createFilterInfo(FilterType.repogroup, '{test}'), false, FILTERS),
            queryStateFromExample('foo {repogroup:[test]}')
        )
    })

    it('updates existing placeholder filter and selects value', () => {
        assert.deepStrictEqual(
            updateQueryWithFilter(
                { query: 'repogroup:value foo' },
                createFilterInfo(FilterType.repogroup, '{test}'),
                false,
                FILTERS
            ),
            queryStateFromExample('{repogroup:[value]} foo')
        )
    })

    it('appends suggestions filter', () => {
        assert.deepStrictEqual(
            updateQueryWithFilter({ query: 'foo' }, createFilterInfo(FilterType.case, '{test}'), false, FILTERS),
            queryStateFromExample('foo {case:[]}', true)
        )
    })

    it('updates existing suggestions filter', () => {
        assert.deepStrictEqual(
            updateQueryWithFilter(
                { query: 'case:yes foo' },
                createFilterInfo(FilterType.case, '{test}'),
                false,
                FILTERS
            ),
            queryStateFromExample('{case:[]} foo', true)
        )
    })
})
