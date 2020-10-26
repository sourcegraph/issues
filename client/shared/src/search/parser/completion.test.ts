import * as Monaco from 'monaco-editor'
import { getCompletionItems, repositoryCompletionItemKind } from './completion'
import { parseSearchQuery, ParseSuccess, Sequence } from './parser'
import { NEVER, of } from 'rxjs'
import { SearchSuggestion } from '../../graphql/schema'

describe('getCompletionItems()', () => {
    test('returns only static filter type completions when the token matches a known filter', async () => {
        expect(
            (
                await getCompletionItems(
                    (parseSearchQuery('re') as ParseSuccess<Sequence>).token,
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
            'count',
            'file',
            '-file',
            'fork',
            'index',
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
            'stable',
            'timeout',
            'type',
            'visibility',
        ])
    })

    test("returns static filter type completions along with dynamically fetched completions when the token doesn't match a filter", async () => {
        expect(
            (
                await getCompletionItems(
                    (parseSearchQuery('reposi') as ParseSuccess<Sequence>).token,
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
            'count',
            'file',
            '-file',
            'fork',
            'index',
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
            'stable',
            'timeout',
            'type',
            'visibility',
            'github.com/sourcegraph/jsonrpc2',
            'RepoRoutes',
        ])
    })

    test('returns suggestions for an empty query', async () => {
        expect(
            (
                await getCompletionItems(
                    (parseSearchQuery('') as ParseSuccess<Sequence>).token,
                    { column: 1 },
                    NEVER,
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
            'count',
            'file',
            '-file',
            'fork',
            'index',
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
            'stable',
            'timeout',
            'type',
            'visibility',
        ])
    })

    test('returns suggestions on whitespace', async () => {
        expect(
            (
                await getCompletionItems(
                    (parseSearchQuery('a ') as ParseSuccess<Sequence>).token,
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
            'count',
            'file',
            '-file',
            'fork',
            'index',
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
            'stable',
            'timeout',
            'type',
            'visibility',
            'github.com/sourcegraph/jsonrpc2',
        ])
    })

    test('returns static filter type completions for case-insensitive query', async () => {
        expect(
            (
                await getCompletionItems(
                    (parseSearchQuery('rE') as ParseSuccess<Sequence>).token,
                    { column: 3 },
                    of([]),
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
            'count',
            'file',
            '-file',
            'fork',
            'index',
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
            'stable',
            'timeout',
            'type',
            'visibility',
        ])
    })

    test('returns completions for filters with discrete values', async () => {
        expect(
            (
                await getCompletionItems(
                    (parseSearchQuery('case:y') as ParseSuccess<Sequence>).token,
                    { column: 7 },
                    NEVER,
                    false
                )
            )?.suggestions.map(({ label }) => label)
        ).toStrictEqual(['yes', 'no'])
    })

    test('returns completions for filters with static suggestions', async () => {
        expect(
            (
                await getCompletionItems(
                    (parseSearchQuery('lang:') as ParseSuccess<Sequence>).token,
                    {
                        column: 6,
                    },
                    of([]),
                    false
                )
            )?.suggestions.map(({ label }) => label)
        ).toStrictEqual([
            'c',
            'cpp',
            'csharp',
            'css',
            'go',
            'graphql',
            'haskell',
            'html',
            'java',
            'javascript',
            'json',
            'lua',
            'markdown',
            'php',
            'powershell',
            'python',
            'r',
            'ruby',
            'rust',
            'sass',
            'swift',
            'typescript',
        ])
    })

    test('returns dynamically fetched completions', async () => {
        expect(
            (
                await getCompletionItems(
                    (parseSearchQuery('file:c') as ParseSuccess<Sequence>).token,
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
                    (parseSearchQuery('jsonrpc') as ParseSuccess<Sequence>).token,
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
                    (parseSearchQuery('f:^jsonrpc') as ParseSuccess<Sequence>).token,
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
                    (parseSearchQuery('main.go') as ParseSuccess<Sequence>).token,
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
                    (parseSearchQuery('f:') as ParseSuccess<Sequence>).token,
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
})
