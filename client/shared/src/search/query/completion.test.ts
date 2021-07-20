/* eslint-disable no-template-curly-in-string */
import * as Monaco from 'monaco-editor'
import { NEVER, of } from 'rxjs'

import { SearchSuggestion } from '../../graphql/schema'

import { getCompletionItems, repositoryCompletionItemKind } from './completion'
import { POPULAR_LANGUAGES } from './languageFilter'
import { scanSearchQuery, ScanSuccess, ScanResult } from './scanner'
import { Token } from './token'

expect.addSnapshotSerializer({
    serialize: value => JSON.stringify(value, null, 2),
    test: () => true,
})

const toSuccess = (result: ScanResult<Token[]>): Token[] => (result as ScanSuccess<Token[]>).term

describe('getCompletionItems()', () => {
    test('returns only static filter type completions when the token matches a known filter', async () => {
        expect(
            (
                await getCompletionItems(
                    toSuccess(scanSearchQuery('re')),
                    { column: 3 },
                    of([
                        {
                            __typename: 'Repository',
                            name: 'github.com/sourcegraph/jsonrpc2',
                        },
                        {
                            __typename: 'Symbol',
                            name: 'RepoRoutes',
                            kind: 'VARIABLE',
                            location: {
                                resource: {
                                    repository: {
                                        name: 'github.com/sourcegraph/jsonrpc2',
                                    },
                                },
                            },
                        },
                    ] as SearchSuggestion[]),
                    false
                )
            )?.suggestions.map(({ label }) => label)
        ).toStrictEqual([
            'after',
            'archived',
            'author',
            '-author',
            'before',
            'case',
            'committer',
            '-committer',
            'content',
            '-content',
            'context',
            'count',
            'file',
            '-file',
            'fork',
            'lang',
            '-lang',
            'message',
            '-message',
            'patterntype',
            'repo',
            '-repo',
            'repogroup',
            'repohascommitafter',
            'repohasfile',
            '-repohasfile',
            'rev',
            'select',
            'timeout',
            'type',
            'visibility',
        ])
    })

    test("returns static filter type completions along with dynamically fetched completions when the token doesn't match a filter", async () => {
        expect(
            (
                await getCompletionItems(
                    toSuccess(scanSearchQuery('reposi')),
                    { column: 7 },
                    of([
                        {
                            __typename: 'Repository',
                            name: 'github.com/sourcegraph/jsonrpc2',
                        },
                        {
                            __typename: 'Symbol',
                            name: 'RepoRoutes',
                            kind: 'VARIABLE',
                            location: {
                                resource: {
                                    repository: {
                                        name: 'github.com/sourcegraph/jsonrpc2',
                                    },
                                },
                            },
                        },
                    ] as SearchSuggestion[]),
                    false
                )
            )?.suggestions.map(({ label }) => label)
        ).toStrictEqual([
            'after',
            'archived',
            'author',
            '-author',
            'before',
            'case',
            'committer',
            '-committer',
            'content',
            '-content',
            'context',
            'count',
            'file',
            '-file',
            'fork',
            'lang',
            '-lang',
            'message',
            '-message',
            'patterntype',
            'repo',
            '-repo',
            'repogroup',
            'repohascommitafter',
            'repohasfile',
            '-repohasfile',
            'rev',
            'select',
            'timeout',
            'type',
            'visibility',
            'github.com/sourcegraph/jsonrpc2',
            'RepoRoutes',
        ])
    })

    test('returns suggestions for an empty query', async () => {
        expect(
            (await getCompletionItems(toSuccess(scanSearchQuery('')), { column: 1 }, NEVER, false))?.suggestions.map(
                ({ label }) => label
            )
        ).toStrictEqual([
            'after',
            'archived',
            'author',
            '-author',
            'before',
            'case',
            'committer',
            '-committer',
            'content',
            '-content',
            'context',
            'count',
            'file',
            '-file',
            'fork',
            'lang',
            '-lang',
            'message',
            '-message',
            'patterntype',
            'repo',
            '-repo',
            'repogroup',
            'repohascommitafter',
            'repohasfile',
            '-repohasfile',
            'rev',
            'select',
            'timeout',
            'type',
            'visibility',
        ])
    })

    test('returns suggestions on whitespace', async () => {
        expect(
            (
                await getCompletionItems(
                    toSuccess(scanSearchQuery('a ')),
                    { column: 3 },
                    of([
                        {
                            __typename: 'Repository',
                            name: 'github.com/sourcegraph/jsonrpc2',
                        },
                    ] as SearchSuggestion[]),
                    false
                )
            )?.suggestions.map(({ label }) => label)
        ).toStrictEqual([
            'after',
            'archived',
            'author',
            '-author',
            'before',
            'case',
            'committer',
            '-committer',
            'content',
            '-content',
            'context',
            'count',
            'file',
            '-file',
            'fork',
            'lang',
            '-lang',
            'message',
            '-message',
            'patterntype',
            'repo',
            '-repo',
            'repogroup',
            'repohascommitafter',
            'repohasfile',
            '-repohasfile',
            'rev',
            'select',
            'timeout',
            'type',
            'visibility',
            'github.com/sourcegraph/jsonrpc2',
        ])
    })

    test('returns static filter type completions for case-insensitive query', async () => {
        expect(
            (await getCompletionItems(toSuccess(scanSearchQuery('rE')), { column: 3 }, of([]), false))?.suggestions.map(
                ({ label }) => label
            )
        ).toStrictEqual([
            'after',
            'archived',
            'author',
            '-author',
            'before',
            'case',
            'committer',
            '-committer',
            'content',
            '-content',
            'context',
            'count',
            'file',
            '-file',
            'fork',
            'lang',
            '-lang',
            'message',
            '-message',
            'patterntype',
            'repo',
            '-repo',
            'repogroup',
            'repohascommitafter',
            'repohasfile',
            '-repohasfile',
            'rev',
            'select',
            'timeout',
            'type',
            'visibility',
        ])
    })

    test('returns completions for filters with discrete values', async () => {
        expect(
            (
                await getCompletionItems(toSuccess(scanSearchQuery('case:y')), { column: 7 }, NEVER, false)
            )?.suggestions.map(({ label }) => label)
        ).toStrictEqual(['yes', 'no'])
    })

    test('returns completions for filters with static suggestions', async () => {
        expect(
            (
                await getCompletionItems(
                    toSuccess(scanSearchQuery('lang:')),
                    {
                        column: 6,
                    },
                    of([]),
                    false
                )
            )?.suggestions.map(({ label }) => label)
        ).toStrictEqual(POPULAR_LANGUAGES)
    })

    test('returns completions in order of discrete value definition, not alphabetically', async () => {
        expect(
            (
                await getCompletionItems(
                    toSuccess(scanSearchQuery('select:')),
                    {
                        column: 8,
                    },
                    of([]),
                    false
                )
            )?.suggestions.map(({ label }) => label)
        ).toStrictEqual(['repo', 'file', 'content', 'symbol', 'commit'])
    })

    test('returns dynamically fetched completions', async () => {
        expect(
            (
                await getCompletionItems(
                    toSuccess(scanSearchQuery('file:c')),
                    { column: 7 },
                    of([
                        {
                            __typename: 'File',
                            path: 'connect.go',
                            name: 'connect.go',
                            repository: {
                                name: 'github.com/sourcegraph/jsonrpc2',
                            },
                        },
                    ] as SearchSuggestion[]),
                    false
                )
            )?.suggestions.map(({ label, insertText }) => ({ label, insertText }))
        ).toStrictEqual([{ label: 'connect.go', insertText: '^connect\\.go$ ' }])
    })

    test('inserts valid filters when selecting a file or repository suggestion', async () => {
        expect(
            (
                await getCompletionItems(
                    toSuccess(scanSearchQuery('jsonrpc')),
                    { column: 8 },
                    of([
                        {
                            __typename: 'File',
                            path: 'jsonrpc2.go',
                            name: 'jsonrpc2.go',
                            repository: {
                                name: 'github.com/sourcegraph/jsonrpc2',
                            },
                        },
                        {
                            __typename: 'Repository',
                            name: 'github.com/sourcegraph/jsonrpc2.go',
                        },
                    ] as SearchSuggestion[]),
                    false
                )
            )?.suggestions
                .filter(
                    ({ kind }) =>
                        kind === Monaco.languages.CompletionItemKind.File || kind === repositoryCompletionItemKind
                )
                .map(({ insertText }) => insertText)
        ).toStrictEqual(['file:^jsonrpc2\\.go$ ', 'repo:^github\\.com/sourcegraph/jsonrpc2\\.go$ '])
    })

    test('sets current filter value as filterText', async () => {
        expect(
            (
                await getCompletionItems(
                    toSuccess(scanSearchQuery('f:^jsonrpc')),
                    { column: 11 },
                    of([
                        {
                            __typename: 'File',
                            path: 'jsonrpc2.go',
                            name: 'jsonrpc2.go',
                            repository: {
                                name: 'github.com/sourcegraph/jsonrpc2',
                            },
                        },
                    ] as SearchSuggestion[]),
                    false
                )
            )?.suggestions.map(({ filterText }) => filterText)
        ).toStrictEqual(['^jsonrpc'])
    })

    test('includes file path in insertText with fuzzy completions', async () => {
        expect(
            (
                await getCompletionItems(
                    toSuccess(scanSearchQuery('main.go')),
                    { column: 7 },
                    of([
                        {
                            __typename: 'File',
                            path: 'some/path/main.go',
                            name: 'main.go',
                            repository: {
                                name: 'github.com/sourcegraph/jsonrpc2',
                            },
                        },
                    ] as SearchSuggestion[]),
                    false
                )
            )?.suggestions
                .filter(({ insertText }) => insertText.includes('some/path'))
                .map(({ insertText }) => insertText)
        ).toStrictEqual(['file:^some/path/main\\.go$ '])
    })

    test('includes file path in insertText when completing filter value', async () => {
        expect(
            (
                await getCompletionItems(
                    toSuccess(scanSearchQuery('f:')),
                    { column: 2 },
                    of([
                        {
                            __typename: 'File',
                            path: 'some/path/main.go',
                            name: 'main.go',
                            repository: {
                                name: 'github.com/sourcegraph/jsonrpc2',
                            },
                        },
                    ] as SearchSuggestion[]),
                    false
                )
            )?.suggestions.map(({ insertText }) => insertText)
        ).toStrictEqual(['^some/path/main\\.go$ '])
    })

    test('escapes spaces in repo value', async () => {
        expect(
            (
                await getCompletionItems(
                    toSuccess(scanSearchQuery('repo:')),
                    { column: 5 },
                    of([
                        {
                            __typename: 'Repository',
                            name: 'repo/with a space',
                        },
                    ] as SearchSuggestion[]),
                    false
                )
            )?.suggestions.map(({ insertText }) => insertText)
        ).toMatchInlineSnapshot(`
            [
              "contains.file(\${1:CHANGELOG}) ",
              "contains.content(\${1:TODO}) ",
              "contains(file:\${1:CHANGELOG} content:\${2:fix}) ",
              "contains.commit.after(\${1:1 month ago}) ",
              "^repo/with\\\\ a\\\\ space$ "
            ]
        `)
    })

    test('Sourcegraph.com GH repo completions', async () => {
        expect(
            (
                await getCompletionItems(toSuccess(scanSearchQuery('repo:')), { column: 5 }, of([]), false, true)
            )?.suggestions.map(({ insertText }) => insertText)
        ).toMatchInlineSnapshot(`
            [
              "^github\\\\.com/\${1:ORGANIZATION}/.* ",
              "^github\\\\.com/\${1:ORGANIZATION}/\${2:REPO-NAME}$ ",
              "\${1:STRING} ",
              "contains.file(\${1:CHANGELOG}) ",
              "contains.content(\${1:TODO}) ",
              "contains(file:\${1:CHANGELOG} content:\${2:fix}) ",
              "contains.commit.after(\${1:1 month ago}) "
            ]
        `)
    })
})
