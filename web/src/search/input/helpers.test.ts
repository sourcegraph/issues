import { convertPlainTextToInteractiveQuery, escapeDoubleQuote } from './helpers'

describe('Search input helpers', () => {
    describe('convertPlainTextToInteractiveQuery', () => {
        test('converts query with no filters', () => {
            const newQuery = convertPlainTextToInteractiveQuery('foo')
            expect(newQuery.navbarQuery === 'foo' && newQuery.filtersInQuery === {})
        })
        test('converts query with one filter', () => {
            const newQuery = convertPlainTextToInteractiveQuery('foo case:yes')
            expect(
                newQuery.navbarQuery === 'foo' &&
                newQuery.filtersInQuery ===
                {
                    case: {
                        type: 'case' as const,
                        value: 'yes',
                        editable: false,
                        negated: false,
                    },
                }
            )
        })
        test('converts query with multiple filters', () => {
            const newQuery = convertPlainTextToInteractiveQuery('foo case:yes archived:no')
            expect(
                newQuery.navbarQuery === 'foo' &&
                newQuery.filtersInQuery ===
                {
                    case: {
                        type: 'case' as const,
                        value: 'yes',
                        editable: false,
                        negated: false,
                    },
                    archived: {
                        type: 'archived' as const,
                        value: 'no',
                        editable: false,
                        negated: false,
                    },
                }
            )
        })

        test('converts query with invalid filters, without adding invalid filters to filtersInQuery', () => {
            const newQuery = convertPlainTextToInteractiveQuery('foo case:yes archived:no asdf:no')
            expect(
                newQuery.navbarQuery === 'foo asdf:no' &&
                newQuery.filtersInQuery ===
                {
                    case: {
                        type: 'case' as const,
                        value: 'yes',
                        editable: false,
                        negated: false,
                    },
                    archived: {
                        type: 'archived' as const,
                        value: 'no',
                        editable: false,
                        negated: false,
                    },
                }
            )
        })
    })

    describe('escapeDoubleQuote', () => {
        test('Correctly converts input when it is one double quote', () => {
            const newQuery = escapeDoubleQuote('"')
            expect(newQuery === '"\\""')
        })
        test('Does not convert input when it is actually a quoted input', () => {
            const newQuery = escapeDoubleQuote('"test query"')
            expect(newQuery === '"test query"')
        })

        test('Does not convert input when it is two double quotes', () => {
            const newQuery = escapeDoubleQuote('""')
            expect(newQuery === '"test query"')
        })
    })
})
