import { getDiagnostics } from './diagnostics'
import { parseSearchQuery, ParseSuccess, Sequence } from './parser'
import { SearchPatternType } from '../../graphql-operations'

describe('getDiagnostics()', () => {
    test('do not raise invalid filter type', () => {
        expect(
            getDiagnostics(
                (parseSearchQuery('repos:^github.com/sourcegraph') as ParseSuccess<Sequence>).token,
                SearchPatternType.literal
            )
        ).toStrictEqual([])
    })

    test('invalid filter value', () => {
        expect(
            getDiagnostics((parseSearchQuery('case:maybe') as ParseSuccess<Sequence>).token, SearchPatternType.literal)
        ).toStrictEqual([
            {
                endColumn: 5,
                endLineNumber: 1,
                message: 'Invalid filter value, expected one of: yes, no.',
                severity: 8,
                startColumn: 1,
                startLineNumber: 1,
            },
        ])
    })

    test('search query containing colon, regexp pattern type, suggest quotes', () => {
        expect(
            getDiagnostics(
                (parseSearchQuery('Configuration::doStuff(...)') as ParseSuccess<Sequence>).token,
                SearchPatternType.regexp
            )
        ).toStrictEqual([
            {
                endColumn: 27,
                endLineNumber: 1,
                message: 'Quoting the query may help if you want a literal match.',
                severity: 4,
                startColumn: 1,
                startLineNumber: 1,
            },
        ])
    })

    test('search query containing colon, literal pattern type, do not raise error', () => {
        expect(
            getDiagnostics(
                (parseSearchQuery('Configuration::doStuff(...)') as ParseSuccess<Sequence>).token,
                SearchPatternType.literal
            )
        ).toStrictEqual([])
    })

    test('search query containing quoted token, regexp pattern type', () => {
        expect(
            getDiagnostics(
                (parseSearchQuery('"Configuration::doStuff(...)"') as ParseSuccess<Sequence>).token,
                SearchPatternType.regexp
            )
        ).toStrictEqual([])
    })

    test('search query containing parenthesized parameterss', () => {
        expect(
            getDiagnostics(
                (parseSearchQuery('repo:a (file:b and c)') as ParseSuccess<Sequence>).token,
                SearchPatternType.regexp
            )
        ).toStrictEqual([])
    })

    test('search query containing quoted token, literal pattern type', () => {
        expect(
            getDiagnostics(
                (parseSearchQuery('"Configuration::doStuff(...)"') as ParseSuccess<Sequence>).token,
                SearchPatternType.literal
            )
        ).toStrictEqual([
            {
                endColumn: 30,
                endLineNumber: 1,
                message: 'Your search is interpreted literally and contains quotes. Did you mean to search for quotes?',
                severity: 4,
                startColumn: 1,
                startLineNumber: 1,
            },
        ])
    })
})
