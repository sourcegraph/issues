import { parseSearchQuery } from './parser'

describe('parseSearchQuery()', () => {
    test('empty', () =>
        expect(parseSearchQuery('')).toMatchObject({
            range: {
                start: 0,
                end: 1,
            },
            token: {
                members: [],
                type: 'sequence',
            },
            type: 'success',
        }))

    test('whitespace', () =>
        expect(parseSearchQuery('  ')).toMatchObject({
            range: {
                start: 0,
                end: 2,
            },
            token: {
                members: [
                    {
                        range: {
                            end: 2,
                            start: 0,
                        },
                        token: {
                            type: 'whitespace',
                        },
                    },
                ],
                type: 'sequence',
            },
            type: 'success',
        }))

    test('literal', () =>
        expect(parseSearchQuery('a')).toMatchObject({
            range: {
                start: 0,
                end: 1,
            },
            token: {
                members: [
                    {
                        range: {
                            start: 0,
                            end: 1,
                        },
                        token: {
                            type: 'literal',
                            value: 'a',
                        },
                    },
                ],
                type: 'sequence',
            },
            type: 'success',
        }))

    test('triple quotes', () => {
        expect(parseSearchQuery('"""')).toMatchObject({
            range: {
                end: 3,
                start: 0,
            },
            token: {
                members: [
                    {
                        range: {
                            end: 3,
                            start: 0,
                        },
                        token: {
                            type: 'literal',
                            value: '"""',
                        },
                    },
                ],
                type: 'sequence',
            },
            type: 'success',
        })
    })

    test('filter', () =>
        expect(parseSearchQuery('f:b')).toMatchObject({
            range: {
                end: 3,
                start: 0,
            },
            token: {
                members: [
                    {
                        range: {
                            end: 3,
                            start: 0,
                        },
                        token: {
                            filterType: {
                                range: {
                                    end: 1,
                                    start: 0,
                                },
                                token: {
                                    type: 'literal',
                                    value: 'f',
                                },
                                type: 'success',
                            },
                            filterValue: {
                                range: {
                                    end: 3,
                                    start: 2,
                                },
                                token: {
                                    type: 'literal',
                                    value: 'b',
                                },
                                type: 'success',
                            },
                            type: 'filter',
                        },
                    },
                ],
                type: 'sequence',
            },
            type: 'success',
        }))

    test('negated filter', () =>
        expect(parseSearchQuery('-f:b')).toMatchObject({
            range: {
                end: 4,
                start: 0,
            },
            token: {
                members: [
                    {
                        range: {
                            end: 4,
                            start: 0,
                        },
                        token: {
                            filterType: {
                                range: {
                                    end: 2,
                                    start: 0,
                                },
                                token: {
                                    type: 'literal',
                                    value: '-f',
                                },
                                type: 'success',
                            },
                            filterValue: {
                                range: {
                                    end: 4,
                                    start: 3,
                                },
                                token: {
                                    type: 'literal',
                                    value: 'b',
                                },
                                type: 'success',
                            },
                            type: 'filter',
                        },
                    },
                ],
                type: 'sequence',
            },
            type: 'success',
        }))

    test('filter with quoted value', () => {
        expect(parseSearchQuery('f:"b"')).toMatchObject({
            range: {
                end: 5,
                start: 0,
            },
            token: {
                members: [
                    {
                        range: {
                            end: 5,
                            start: 0,
                        },
                        token: {
                            filterType: {
                                range: {
                                    end: 1,
                                    start: 0,
                                },
                                token: {
                                    type: 'literal',
                                    value: 'f',
                                },
                                type: 'success',
                            },
                            filterValue: {
                                range: {
                                    end: 5,
                                    start: 2,
                                },
                                token: {
                                    quotedValue: 'b',
                                    type: 'quoted',
                                },
                                type: 'success',
                            },
                            type: 'filter',
                        },
                    },
                ],
                type: 'sequence',
            },
            type: 'success',
        })
    })

    test('filter with a value ending with a colon', () => {
        expect(parseSearchQuery('f:a:')).toStrictEqual({
            range: {
                end: 4,
                start: 0,
            },
            token: {
                members: [
                    {
                        range: {
                            end: 4,
                            start: 0,
                        },
                        token: {
                            filterType: {
                                range: {
                                    end: 1,
                                    start: 0,
                                },
                                token: {
                                    type: 'literal',
                                    value: 'f',
                                },
                                type: 'success',
                            },
                            filterValue: {
                                range: {
                                    end: 4,
                                    start: 2,
                                },
                                token: {
                                    type: 'literal',
                                    value: 'a:',
                                },
                                type: 'success',
                            },
                            type: 'filter',
                        },
                    },
                ],
                type: 'sequence',
            },
            type: 'success',
        })
    })

    test('filter where the value is a colon', () => {
        expect(parseSearchQuery('f::')).toStrictEqual({
            range: {
                end: 3,
                start: 0,
            },
            token: {
                members: [
                    {
                        range: {
                            end: 3,
                            start: 0,
                        },
                        token: {
                            filterType: {
                                range: {
                                    end: 1,
                                    start: 0,
                                },
                                token: {
                                    type: 'literal',
                                    value: 'f',
                                },
                                type: 'success',
                            },
                            filterValue: {
                                range: {
                                    end: 3,
                                    start: 2,
                                },
                                token: {
                                    type: 'literal',
                                    value: ':',
                                },
                                type: 'success',
                            },
                            type: 'filter',
                        },
                    },
                ],
                type: 'sequence',
            },
            type: 'success',
        })
    })

    test('quoted', () =>
        expect(parseSearchQuery('"a:b"')).toMatchObject({
            range: {
                end: 5,
                start: 0,
            },
            token: {
                members: [
                    {
                        range: {
                            end: 5,
                            start: 0,
                        },
                        token: {
                            quotedValue: 'a:b',
                            type: 'quoted',
                        },
                    },
                ],
                type: 'sequence',
            },
            type: 'success',
        }))

    test('quoted (escaped quotes)', () =>
        expect(parseSearchQuery('"-\\"a\\":b"')).toMatchObject({
            range: {
                end: 10,
                start: 0,
            },
            token: {
                members: [
                    {
                        range: {
                            end: 10,
                            start: 0,
                        },
                        token: {
                            quotedValue: '-\\"a\\":b',
                            type: 'quoted',
                        },
                    },
                ],
                type: 'sequence',
            },
            type: 'success',
        }))

    test('complex query', () =>
        expect(parseSearchQuery('repo:^github\\.com/gorilla/mux$ lang:go -file:mux.go Router')).toMatchObject({
            range: {
                end: 58,
                start: 0,
            },
            token: {
                members: [
                    {
                        range: {
                            end: 30,
                            start: 0,
                        },
                        token: {
                            filterType: {
                                range: {
                                    end: 4,
                                    start: 0,
                                },
                                token: {
                                    type: 'literal',
                                    value: 'repo',
                                },
                                type: 'success',
                            },
                            filterValue: {
                                range: {
                                    end: 30,
                                    start: 5,
                                },
                                token: {
                                    type: 'literal',
                                    value: '^github\\.com/gorilla/mux$',
                                },
                                type: 'success',
                            },
                            type: 'filter',
                        },
                    },
                    {
                        range: {
                            end: 31,
                            start: 30,
                        },
                        token: {
                            type: 'whitespace',
                        },
                    },
                    {
                        range: {
                            end: 38,
                            start: 31,
                        },
                        token: {
                            filterType: {
                                range: {
                                    end: 35,
                                    start: 31,
                                },
                                token: {
                                    type: 'literal',
                                    value: 'lang',
                                },
                                type: 'success',
                            },
                            filterValue: {
                                range: {
                                    end: 38,
                                    start: 36,
                                },
                                token: {
                                    type: 'literal',
                                    value: 'go',
                                },
                                type: 'success',
                            },
                            type: 'filter',
                        },
                    },
                    {
                        range: {
                            end: 39,
                            start: 38,
                        },
                        token: {
                            type: 'whitespace',
                        },
                    },
                    {
                        range: {
                            end: 51,
                            start: 39,
                        },
                        token: {
                            filterType: {
                                range: {
                                    end: 44,
                                    start: 39,
                                },
                                token: {
                                    type: 'literal',
                                    value: '-file',
                                },
                                type: 'success',
                            },
                            filterValue: {
                                range: {
                                    end: 51,
                                    start: 45,
                                },
                                token: {
                                    type: 'literal',
                                    value: 'mux.go',
                                },
                                type: 'success',
                            },
                            type: 'filter',
                        },
                    },
                    {
                        range: {
                            end: 52,
                            start: 51,
                        },
                        token: {
                            type: 'whitespace',
                        },
                    },
                    {
                        range: {
                            end: 58,
                            start: 52,
                        },
                        token: {
                            type: 'literal',
                            value: 'Router',
                        },
                    },
                ],
                type: 'sequence',
            },
            type: 'success',
        }))

    test('parenthesized parameters', () => {
        expect(parseSearchQuery('repo:a (file:b and c)')).toMatchObject({
            range: {
                end: 21,
                start: 0,
            },
            token: {
                members: [
                    {
                        range: {
                            end: 6,
                            start: 0,
                        },
                        token: {
                            filterType: {
                                range: {
                                    end: 4,
                                    start: 0,
                                },
                                token: {
                                    type: 'literal',
                                    value: 'repo',
                                },
                                type: 'success',
                            },
                            filterValue: {
                                range: {
                                    end: 6,
                                    start: 5,
                                },
                                token: {
                                    type: 'literal',
                                    value: 'a',
                                },
                                type: 'success',
                            },
                            type: 'filter',
                        },
                    },
                    {
                        range: {
                            end: 7,
                            start: 6,
                        },
                        token: {
                            type: 'whitespace',
                        },
                    },
                    {
                        range: {
                            end: 8,
                            start: 7,
                        },
                        token: {
                            type: 'openingParen',
                        },
                    },
                    {
                        range: {
                            end: 14,
                            start: 8,
                        },
                        token: {
                            filterType: {
                                range: {
                                    end: 12,
                                    start: 8,
                                },
                                token: {
                                    type: 'literal',
                                    value: 'file',
                                },
                                type: 'success',
                            },
                            filterValue: {
                                range: {
                                    end: 14,
                                    start: 13,
                                },
                                token: {
                                    type: 'literal',
                                    value: 'b',
                                },
                                type: 'success',
                            },
                            type: 'filter',
                        },
                    },
                    {
                        range: {
                            end: 15,
                            start: 14,
                        },
                        token: {
                            type: 'whitespace',
                        },
                    },
                    {
                        range: {
                            end: 18,
                            start: 15,
                        },
                        token: {
                            type: 'operator',
                            value: 'and',
                        },
                    },
                    {
                        range: {
                            end: 19,
                            start: 18,
                        },
                        token: {
                            type: 'whitespace',
                        },
                    },
                    {
                        range: {
                            end: 20,
                            start: 19,
                        },
                        token: {
                            type: 'literal',
                            value: 'c',
                        },
                    },
                    {
                        range: {
                            end: 21,
                            start: 20,
                        },
                        token: {
                            type: 'closingParen',
                        },
                    },
                ],
                type: 'sequence',
            },
            type: 'success',
        })
    })

    test('nested parenthesized parameters', () => {
        expect(parseSearchQuery('(a and (b or c) and d)')).toMatchObject({
            range: {
                end: 22,
                start: 0,
            },
            token: {
                members: [
                    {
                        range: {
                            end: 1,
                            start: 0,
                        },
                        token: {
                            type: 'openingParen',
                        },
                    },
                    {
                        range: {
                            end: 2,
                            start: 1,
                        },
                        token: {
                            type: 'literal',
                            value: 'a',
                        },
                    },
                    {
                        range: {
                            end: 3,
                            start: 2,
                        },
                        token: {
                            type: 'whitespace',
                        },
                    },
                    {
                        range: {
                            end: 6,
                            start: 3,
                        },
                        token: {
                            type: 'operator',
                        },
                    },
                    {
                        range: {
                            end: 7,
                            start: 6,
                        },
                        token: {
                            type: 'whitespace',
                        },
                    },
                    {
                        range: {
                            end: 8,
                            start: 7,
                        },
                        token: {
                            type: 'openingParen',
                        },
                    },
                    {
                        range: {
                            end: 9,
                            start: 8,
                        },
                        token: {
                            type: 'literal',
                            value: 'b',
                        },
                    },
                    {
                        range: {
                            end: 10,
                            start: 9,
                        },
                        token: {
                            type: 'whitespace',
                        },
                    },
                    {
                        range: {
                            end: 12,
                            start: 10,
                        },
                        token: {
                            type: 'operator',
                        },
                    },
                    {
                        range: {
                            end: 13,
                            start: 12,
                        },
                        token: {
                            type: 'whitespace',
                        },
                    },
                    {
                        range: {
                            end: 14,
                            start: 13,
                        },
                        token: {
                            type: 'literal',
                            value: 'c',
                        },
                    },
                    {
                        range: {
                            end: 15,
                            start: 14,
                        },
                        token: {
                            type: 'closingParen',
                        },
                    },
                    {
                        range: {
                            end: 16,
                            start: 15,
                        },
                        token: {
                            type: 'whitespace',
                        },
                    },
                    {
                        range: {
                            end: 19,
                            start: 16,
                        },
                        token: {
                            type: 'operator',
                        },
                    },
                    {
                        range: {
                            end: 20,
                            start: 19,
                        },
                        token: {
                            type: 'whitespace',
                        },
                    },
                    {
                        range: {
                            end: 21,
                            start: 20,
                        },
                        token: {
                            type: 'literal',
                            value: 'd',
                        },
                    },
                    {
                        range: {
                            end: 22,
                            start: 21,
                        },
                        token: {
                            type: 'closingParen',
                        },
                    },
                ],
                type: 'sequence',
            },
            type: 'success',
        })
    })

    test('do not treat links as filters', () => {
        expect(parseSearchQuery('http://example.com repo:a')).toMatchObject({
            range: {
                end: 25,
                start: 0,
            },
            token: {
                members: [
                    {
                        range: {
                            end: 18,
                            start: 0,
                        },
                        token: {
                            type: 'literal',
                            value: 'http://example.com',
                        },
                    },
                    {
                        range: {
                            end: 19,
                            start: 18,
                        },
                        token: {
                            type: 'whitespace',
                        },
                    },
                    {
                        range: {
                            end: 25,
                            start: 19,
                        },
                        token: {
                            filterType: {
                                range: {
                                    end: 23,
                                    start: 19,
                                },
                            },
                        },
                    },
                ],
                type: 'sequence',
            },
            type: 'success',
        })
    })

    test('interpret C-style comments', () => {
        const query = `// saucegraph is best graph
repo:sourcegraph
// search for thing
thing`
        expect(parseSearchQuery(query, true)).toMatchObject({
            range: {
                end: 70,
                start: 0,
            },
            token: {
                members: [
                    {
                        range: {
                            start: 0,
                            end: 27,
                        },
                        token: {
                            type: 'comment',
                            value: '// saucegraph is best graph',
                        },
                    },
                    {
                        range: {
                            start: 27,
                            end: 28,
                        },
                        token: {
                            type: 'whitespace',
                        },
                    },
                    {
                        range: {
                            start: 28,
                            end: 44,
                        },
                        token: {
                            filterType: {
                                range: {
                                    start: 28,
                                    end: 32,
                                },
                                token: {
                                    type: 'literal',
                                    value: 'repo',
                                },
                            },
                            filterValue: {
                                range: {
                                    start: 33,
                                    end: 44,
                                },
                                token: {
                                    type: 'literal',
                                    value: 'sourcegraph',
                                },
                                type: 'success',
                            },
                            type: 'filter',
                        },
                    },
                    {
                        range: {
                            start: 44,
                            end: 45,
                        },
                        token: {
                            type: 'whitespace',
                        },
                    },
                    {
                        range: {
                            start: 45,
                            end: 64,
                        },
                        token: {
                            type: 'comment',
                            value: '// search for thing',
                        },
                    },
                    {
                        range: {
                            start: 64,
                            end: 65,
                        },
                        token: {
                            type: 'whitespace',
                        },
                    },
                    {
                        range: {
                            start: 65,
                            end: 70,
                        },
                        token: {
                            type: 'literal',
                            value: 'thing',
                        },
                    },
                ],
                type: 'sequence',
            },
            type: 'success',
        })
    })

    test('do not interpret C-style comments', () => {
        expect(parseSearchQuery('// thing')).toMatchObject({
            range: {
                end: 8,
                start: 0,
            },
            token: {
                members: [
                    {
                        range: {
                            start: 0,
                            end: 2,
                        },
                        token: {
                            type: 'literal',
                            value: '//',
                        },
                    },
                    {
                        range: {
                            start: 2,
                            end: 3,
                        },
                        token: {
                            type: 'whitespace',
                        },
                    },
                    {
                        range: {
                            start: 3,
                            end: 8,
                        },
                        token: {
                            type: 'literal',
                            value: 'thing',
                        },
                    },
                ],
                type: 'sequence',
            },
            type: 'success',
        })
    })
})
