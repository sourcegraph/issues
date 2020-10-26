/* eslint rxjs/no-ignored-subscription: warn */
import { Subject, Observable } from 'rxjs'
import {
    debounceTime,
    distinctUntilChanged,
    filter,
    map,
    mergeMap,
    publishReplay,
    refCount,
    repeat,
    switchMap,
    take,
    toArray,
} from 'rxjs/operators'
import { dataOrThrowErrors, gql } from '../../../../shared/src/graphql/graphql'
import * as GQL from '../../../../shared/src/graphql/schema'
import { PlatformContext } from '../../../../shared/src/platform/context'
import { isDefined } from '../../../../shared/src/util/types'

interface BaseSuggestion {
    title: string
    description?: string

    /**
     * The URL that is navigated to when the user selects this
     * suggestion.
     */
    url: string

    /**
     * A label describing the action taken when navigating to
     * the URL (e.g., "go to repository").
     */
    urlLabel: string
}

interface SymbolSuggestion extends BaseSuggestion {
    type: 'symbol'
    kind: string
}

interface RepoSuggestion extends BaseSuggestion {
    type: 'repo'
}

interface FileSuggestion extends BaseSuggestion {
    type: 'file'
}

interface DirectorySuggestion extends BaseSuggestion {
    type: 'dir'
}

export type Suggestion = SymbolSuggestion | RepoSuggestion | FileSuggestion | DirectorySuggestion

/**
 * Returns all but the last element of path, or "." if that would be the empty path.
 */
function dirname(path: string): string | undefined {
    return path.split('/').slice(0, -1).join('/') || '.'
}

/**
 * Returns the last element of path, or "." if path is empty.
 */
function basename(path: string): string {
    return path.split('/').slice(-1)[0] || '.'
}

function createSuggestion(item: GQL.SearchSuggestion): Suggestion | null {
    switch (item.__typename) {
        case 'Repository': {
            return {
                type: 'repo',
                title: item.name,
                url: item.url,
                urlLabel: 'go to repository',
            }
        }
        case 'File': {
            const descriptionParts: string[] = []
            const directory = dirname(item.path)
            if (directory !== undefined && directory !== '.') {
                descriptionParts.push(`${directory}/`)
            }
            descriptionParts.push(basename(item.repository.name))
            if (item.isDirectory) {
                return {
                    type: 'dir',
                    title: item.name,
                    description: descriptionParts.join(' — '),
                    url: item.url,
                    urlLabel: 'go to dir',
                }
            }
            return {
                type: 'file',
                title: item.name,
                description: descriptionParts.join(' — '),
                url: item.url,
                urlLabel: 'go to file',
            }
        }
        case 'Symbol': {
            return {
                type: 'symbol',
                kind: item.kind,
                title: item.name,
                description: `${item.containerName || item.location.resource.path} — ${basename(
                    item.location.resource.repository.name
                )}`,
                url: item.url,
                urlLabel: 'go to definition',
            }
        }
        default:
            return null
    }
}

const symbolsFragment = gql`
    fragment SymbolFields on Symbol {
        __typename
        name
        containerName
        url
        kind
        location {
            resource {
                path
                repository {
                    name
                }
            }
            url
        }
    }
`

const fetchSuggestions = (
    query: string,
    first: number,
    requestGraphQL: PlatformContext['requestGraphQL']
): Observable<GQL.SearchSuggestion> =>
    requestGraphQL<GQL.IQuery>({
        request: gql`
            query SearchSuggestions($query: String!, $first: Int!) {
                search(query: $query) {
                    suggestions(first: $first) {
                        ... on Repository {
                            __typename
                            name
                            url
                        }
                        ... on File {
                            __typename
                            path
                            name
                            isDirectory
                            url
                            repository {
                                name
                            }
                        }
                        ... on Symbol {
                            ...SymbolFields
                        }
                    }
                }
            }
            ${symbolsFragment}
        `,
        variables: {
            query,
            first,
        },
        mightContainPrivateInfo: true,
    }).pipe(
        map(dataOrThrowErrors),
        mergeMap(({ search }) => {
            if (!search || !search.suggestions) {
                throw new Error('No search suggestions')
            }
            return search.suggestions
        })
    )

interface SuggestionInput {
    query: string
    handler: (suggestion: Suggestion[]) => void
}

export const createSuggestionFetcher = (
    first = 5,
    requestGraphQL: PlatformContext['requestGraphQL']
): ((input: SuggestionInput) => void) => {
    const fetcher = new Subject<SuggestionInput>()

    fetcher
        .pipe(
            distinctUntilChanged(),
            debounceTime(200),
            switchMap(({ query, handler }) =>
                fetchSuggestions(query, first, requestGraphQL).pipe(
                    take(first),
                    map(createSuggestion),
                    // createSuggestion will return null if we get a type we don't recognize
                    filter(isDefined),
                    toArray(),
                    map((suggestions: Suggestion[]) => ({
                        suggestions,
                        suggestHandler: handler,
                    })),
                    publishReplay(),
                    refCount()
                )
            ),
            // But resubscribe afterwards
            repeat()
        )
        .subscribe(({ suggestions, suggestHandler }) => suggestHandler(suggestions))

    return input => fetcher.next(input)
}
