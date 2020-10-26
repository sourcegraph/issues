import { getMonacoTokens } from './tokens'
import { parseSearchQuery, ParseSuccess, Sequence } from './parser'

describe('getMonacoTokens()', () => {
    test('returns the tokens for a parsed search query', () => {
        expect(
            getMonacoTokens(
                (parseSearchQuery('r:^github.com/sourcegraph f:code_intelligence trackViews') as ParseSuccess<Sequence>)
                    .token
            )
        ).toStrictEqual([
            {
                scopes: 'keyword',
                startIndex: 0,
            },
            {
                scopes: 'identifier',
                startIndex: 2,
            },
            {
                scopes: 'whitespace',
                startIndex: 25,
            },
            {
                scopes: 'keyword',
                startIndex: 26,
            },
            {
                scopes: 'identifier',
                startIndex: 28,
            },
            {
                scopes: 'whitespace',
                startIndex: 45,
            },
            {
                scopes: 'identifier',
                startIndex: 46,
            },
        ])
    })

    test('search query containing parenthesized parameters', () => {
        expect(getMonacoTokens((parseSearchQuery('r:a (f:b and c)') as ParseSuccess<Sequence>).token)).toStrictEqual([
            {
                scopes: 'keyword',
                startIndex: 0,
            },
            {
                scopes: 'identifier',
                startIndex: 2,
            },
            {
                scopes: 'whitespace',
                startIndex: 3,
            },
            {
                scopes: 'identifier',
                startIndex: 4,
            },
            {
                scopes: 'keyword',
                startIndex: 5,
            },
            {
                scopes: 'identifier',
                startIndex: 7,
            },
            {
                scopes: 'whitespace',
                startIndex: 8,
            },
            {
                scopes: 'operator',
                startIndex: 9,
            },
            {
                scopes: 'whitespace',
                startIndex: 12,
            },
            {
                scopes: 'identifier',
                startIndex: 13,
            },
            {
                scopes: 'identifier',
                startIndex: 14,
            },
        ])
    })
})
