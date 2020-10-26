import { getHoverResult } from './hover'
import { parseSearchQuery, ParseSuccess, Sequence } from './parser'

describe('getHoverResult()', () => {
    test('returns hover contents for filters', () => {
        const parsedQuery = (parseSearchQuery('repo:sourcegraph file:code_intelligence') as ParseSuccess<Sequence>)
            .token
        expect(getHoverResult(parsedQuery, { column: 4 })).toStrictEqual({
            contents: [
                {
                    value: 'Include only results from repositories matching the given search pattern.',
                },
            ],
            range: {
                endColumn: 17,
                endLineNumber: 1,
                startColumn: 1,
                startLineNumber: 1,
            },
        })
        expect(getHoverResult(parsedQuery, { column: 30 })).toStrictEqual({
            contents: [
                {
                    value: 'Include only results from files matching the given search pattern.',
                },
            ],
            range: {
                endColumn: 40,
                endLineNumber: 1,
                startColumn: 18,
                startLineNumber: 1,
            },
        })
    })
})
