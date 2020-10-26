/* eslint-disable id-length */
import { Observable, fromEvent, Subscription, OperatorFunction, pipe } from 'rxjs'
import { defaultIfEmpty, map, scan } from 'rxjs/operators'
import * as GQL from '../../../shared/src/graphql/schema'
import { SearchPatternType } from '../graphql-operations'

// This is an initial proof of concept implementation of search streaming.
// The protocol and implementation is still in the design phase. Feel free to
// change anything and everything here. We are iteratively improving this
// until it is no longer a proof of concept and instead works well.

type SearchEvent =
    | { type: 'filematches'; matches: FileMatch[] }
    | { type: 'repomatches'; matches: RepositoryMatch[] }
    | { type: 'commitmatches'; matches: CommitMatch[] }
    | { type: 'symbolmatches'; matches: FileSymbolMatch[] }
    | { type: 'filters'; filters: Filter[] }
    | { type: 'alert'; alert: Alert }

interface FileMatch extends RepositoryMatch {
    name: string
    repository: string
    branches?: string[]
    version?: string
    lineMatches: LineMatch[]
}

interface LineMatch {
    line: string
    lineNumber: number
    offsetAndLengths: number[][]
}

interface FileSymbolMatch extends Omit<FileMatch, 'lineMatches'> {
    symbols: SymbolMatch[]
}

interface SymbolMatch {
    url: string
    name: string
    containerName: string
    kind: string
}

type MarkdownText = string

/**
 * Our batch based client requests generic fields from GraphQL to represent repo and commit/diff matches.
 * We currently are only using it for commit. To simplify the PoC we are keeping this interface for commits.
 *
 * @see GQL.IGenericSearchResultInterface
 */
interface CommitMatch {
    icon: string
    label: MarkdownText
    url: string
    detail: MarkdownText

    content: MarkdownText
    ranges: number[][]
}

type RepositoryMatch = Pick<FileMatch, 'repository' | 'branches'>

interface Filter {
    value: string
    label: string
    count: number
    limitHit: boolean
    kind: string
}

interface Alert {
    title: string
    description?: string
    proposedQueries: ProposedQuery[]
}

interface ProposedQuery {
    description?: string
    query: string
}

const toGQLLineMatch = (line: LineMatch): GQL.ILineMatch => ({
    __typename: 'LineMatch',
    limitHit: false,
    lineNumber: line.lineNumber,
    offsetAndLengths: line.offsetAndLengths,
    preview: line.line,
})

function toGQLFileMatchBase(fm: Omit<FileMatch, 'lineMatches'>): GQL.IFileMatch {
    let revision = ''
    if (fm.branches) {
        const branch = fm.branches[0]
        if (branch !== '') {
            revision = '@' + branch
        }
    } else if (fm.version) {
        revision = '@' + fm.version
    }

    // eslint-disable-next-line @typescript-eslint/consistent-type-assertions
    const file: GQL.IGitBlob = {
        path: fm.name,
        // /github.com/gorilla/mux@v1.7.2/-/blob/mux_test.go
        // TODO return in response?
        url: '/' + fm.repository + revision + '/-/blob/' + fm.name,
        commit: {
            oid: fm.version || '',
        },
    } as GQL.IGitBlob
    // eslint-disable-next-line @typescript-eslint/consistent-type-assertions
    const repository: GQL.IRepository = {
        name: fm.repository,
    } as GQL.IRepository
    return {
        __typename: 'FileMatch',
        file,
        repository,
        revSpec: null,
        resource: fm.name,
        symbols: [],
        lineMatches: [],
        limitHit: false,
    }
}

const toGQLFileMatch = (fm: FileMatch): GQL.IFileMatch => ({
    ...toGQLFileMatchBase(fm),
    lineMatches: fm.lineMatches.map(toGQLLineMatch),
})

function toGQLSymbol(symbol: SymbolMatch): GQL.ISymbol {
    // eslint-disable-next-line @typescript-eslint/consistent-type-assertions
    return {
        __typename: 'Symbol',
        ...symbol,
    } as GQL.ISymbol
}

const toGQLSymbolMatch = (fm: FileSymbolMatch): GQL.IFileMatch => ({
    ...toGQLFileMatchBase(fm),
    symbols: fm.symbols.map(toGQLSymbol),
})

// eslint-disable-next-line @typescript-eslint/consistent-type-assertions
const toMarkdown = (text: string | MarkdownText): GQL.IMarkdown => ({ __typename: 'Markdown', text } as GQL.IMarkdown)

// copy-paste from search_repositories.go. When we move away from GQL types this shouldn't be part of the API.
const repoIcon =
    'data:image/svg+xml;base64,PHN2ZyB2ZXJzaW9uPSIxLjEiIGlkPSJMYXllcl8xIiB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHhtbG5zOnhsaW5rPSJodHRwOi8vd3d3LnczLm9yZy8xOTk5L3hsaW5rIiB4PSIwcHgiIHk9IjBweCIKCSB2aWV3Qm94PSIwIDAgNjQgNjQiIHN0eWxlPSJlbmFibGUtYmFja2dyb3VuZDpuZXcgMCAwIDY0IDY0OyIgeG1sOnNwYWNlPSJwcmVzZXJ2ZSI+Cjx0aXRsZT5JY29ucyA0MDA8L3RpdGxlPgo8Zz4KCTxwYXRoIGQ9Ik0yMywyMi40YzEuMywwLDIuNC0xLjEsMi40LTIuNHMtMS4xLTIuNC0yLjQtMi40Yy0xLjMsMC0yLjQsMS4xLTIuNCwyLjRTMjEuNywyMi40LDIzLDIyLjR6Ii8+Cgk8cGF0aCBkPSJNMzUsMjYuNGMxLjMsMCwyLjQtMS4xLDIuNC0yLjRzLTEuMS0yLjQtMi40LTIuNHMtMi40LDEuMS0yLjQsMi40UzMzLjcsMjYuNCwzNSwyNi40eiIvPgoJPHBhdGggZD0iTTIzLDQyLjRjMS4zLDAsMi40LTEuMSwyLjQtMi40cy0xLjEtMi40LTIuNC0yLjRzLTIuNCwxLjEtMi40LDIuNFMyMS43LDQyLjQsMjMsNDIuNHoiLz4KCTxwYXRoIGQ9Ik01MCwxNmgtMS41Yy0wLjMsMC0wLjUsMC4yLTAuNSwwLjV2MzVjMCwwLjMtMC4yLDAuNS0wLjUsMC41aC0yN2MtMC41LDAtMS0wLjItMS40LTAuNmwtMC42LTAuNmMtMC4xLTAuMS0wLjEtMC4yLTAuMS0wLjQKCQljMC0wLjMsMC4yLTAuNSwwLjUtMC41SDQ0YzEuMSwwLDItMC45LDItMlYxMmMwLTEuMS0wLjktMi0yLTJIMTRjLTEuMSwwLTIsMC45LTIsMnYzNi4zYzAsMS4xLDAuNCwyLjEsMS4yLDIuOGwzLjEsMy4xCgkJYzEuMSwxLjEsMi43LDEuOCw0LjIsMS44SDUwYzEuMSwwLDItMC45LDItMlYxOEM1MiwxNi45LDUxLjEsMTYsNTAsMTZ6IE0xOSwyMGMwLTIuMiwxLjgtNCw0LTRjMS40LDAsMi44LDAuOCwzLjUsMgoJCWMxLjEsMS45LDAuNCw0LjMtMS41LDUuNFYzM2MxLTAuNiwyLjMtMC45LDQtMC45YzEsMCwyLTAuNSwyLjgtMS4zQzMyLjUsMzAsMzMsMjkuMSwzMywyOHYtMC42Yy0xLjItMC43LTItMi0yLTMuNQoJCWMwLTIuMiwxLjgtNCw0LTRjMi4yLDAsNCwxLjgsNCw0YzAsMS41LTAuOCwyLjctMiwzLjVoMGMtMC4xLDIuMS0wLjksNC40LTIuNSw2Yy0xLjYsMS42LTMuNCwyLjQtNS41LDIuNWMtMC44LDAtMS40LDAuMS0xLjksMC4zCgkJYy0wLjIsMC4xLTEsMC44LTEuMiwwLjlDMjYuNiwzOCwyNywzOC45LDI3LDQwYzAsMi4yLTEuOCw0LTQsNHMtNC0xLjgtNC00YzAtMS41LDAuOC0yLjcsMi0zLjRWMjMuNEMxOS44LDIyLjcsMTksMjEuNCwxOSwyMHoiLz4KPC9nPgo8L3N2Zz4K'

function toGQLRepositoryMatch(repo: RepositoryMatch): GQL.IRepository {
    const branch = repo?.branches?.[0]
    const revision = branch ? `@${branch}` : ''
    const label = repo.repository + revision

    // We only need to return the subset defined in IGenericSearchResultInterface
    const gqlRepo: unknown = {
        __typename: 'Repository',
        icon: repoIcon,
        label: toMarkdown(`[${label}](/${label})`),
        url: '/' + label,
        detail: toMarkdown('Repository name match'),
        matches: [],
    }

    return gqlRepo as GQL.IRepository
}

//
function toGQLCommitMatch(commit: CommitMatch): GQL.ICommitSearchResult {
    const match = {
        __typename: 'SearchResultMatch',
        url: commit.url,
        body: toMarkdown(commit.content),
        highlights: commit.ranges.map(([line, character, length]) => ({
            __typename: 'IHighlight',
            line,
            character,
            length,
        })),
    }

    // We only need to return the subset defined in IGenericSearchResultInterface
    const gqlCommit: unknown = {
        __typename: 'CommitSearchResult',
        icon: commit.icon,
        label: toMarkdown(commit.label),
        url: commit.url,
        detail: toMarkdown(commit.detail),
        matches: [match],
    }

    return gqlCommit as GQL.ICommitSearchResult
}

const toGQLSearchFilter = (filter: Omit<Filter, 'type'>): GQL.ISearchFilter => ({
    __typename: 'SearchFilter',
    ...filter,
})

const toGQLSearchAlert = (alert: Alert): GQL.ISearchAlert => ({
    __typename: 'SearchAlert',
    title: alert.title,
    description: alert.description || null,
    proposedQueries:
        alert.proposedQueries?.map(pq => ({
            __typename: 'SearchQueryDescription',
            description: pq.description || null,
            query: pq.query,
        })) || null,
})

const emptyGQLSearchResults: GQL.ISearchResults = {
    __typename: 'SearchResults',
    matchCount: 0,
    resultCount: 0,
    approximateResultCount: '',
    limitHit: false,
    sparkline: [],
    repositories: [],
    repositoriesCount: 0,
    repositoriesSearched: [],
    indexedRepositoriesSearched: [],
    cloning: [],
    missing: [],
    timedout: [],
    indexUnavailable: false,
    alert: null,
    elapsedMilliseconds: 0,
    dynamicFilters: [],
    results: [],
    pageInfo: { __typename: 'PageInfo', endCursor: null, hasNextPage: false },
}

/**
 * Converts a stream of SearchEvents into an aggregated GQL.ISearchResult
 */
export const switchToGQLISearchResults: OperatorFunction<SearchEvent, GQL.ISearchResults> = pipe(
    scan((results: GQL.ISearchResults, newEvent: SearchEvent) => {
        switch (newEvent.type) {
            case 'filematches':
                return {
                    ...results,
                    // File matches are additive
                    results: results.results.concat(newEvent.matches.map(toGQLFileMatch)),
                }

            case 'repomatches':
                return {
                    ...results,
                    // Repository matches are additive
                    results: results.results.concat(newEvent.matches.map(toGQLRepositoryMatch)),
                }

            case 'commitmatches':
                return {
                    ...results,
                    // Generic matches are additive
                    results: results.results.concat(newEvent.matches.map(toGQLCommitMatch)),
                }

            case 'symbolmatches':
                return {
                    ...results,
                    // symbol matches are additive
                    results: results.results.concat(newEvent.matches.map(toGQLSymbolMatch)),
                }

            case 'filters':
                return {
                    ...results,
                    // New filter results replace all previous ones
                    dynamicFilters: newEvent.filters.map(toGQLSearchFilter),
                }

            case 'alert':
                return {
                    ...results,
                    alert: toGQLSearchAlert(newEvent.alert),
                }
        }
    }, emptyGQLSearchResults),
    defaultIfEmpty(emptyGQLSearchResults)
)

const observeMessages = <T extends {}>(eventSource: EventSource, eventName: SearchEvent['type']): Observable<T> =>
    fromEvent(eventSource, eventName).pipe(
        map((event: Event) => {
            if (!(event instanceof MessageEvent)) {
                throw new TypeError(`internal error: expected MessageEvent in streaming search ${eventName}`)
            }
            try {
                const parsedData = JSON.parse(event.data) as T
                return parsedData
            } catch {
                throw new Error(`Could not parse ${eventName} message data in streaming search`)
            }
        })
    )

/**
 * Initiates a streaming search. This is a type safe wrapper around Sourcegraph's streaming search API (using Server Sent Events).
 * The observable will emit each event returned from the backend.
 *
 * @param query the search query to send to Sourcegraph's backend.
 */
export function search(
    query: string,
    version: string,
    patternType: SearchPatternType,
    versionContext: string | undefined
): Observable<SearchEvent> {
    return new Observable<SearchEvent>(observer => {
        const parameters = [
            ['q', query],
            ['v', version],
            ['t', patternType as string],
        ]
        if (versionContext) {
            parameters.push(['vc', versionContext])
        }
        const parameterEncoded = parameters.map(([k, v]) => k + '=' + encodeURIComponent(v)).join('&')

        const eventSource = new EventSource('/search/stream?' + parameterEncoded)
        const subscriptions = new Subscription()
        subscriptions.add(
            observeMessages<FileMatch[]>(eventSource, 'filematches')
                .pipe(map(matches => ({ type: 'filematches' as const, matches })))
                .subscribe(observer)
        )
        subscriptions.add(
            observeMessages<FileSymbolMatch[]>(eventSource, 'symbolmatches')
                .pipe(map(matches => ({ type: 'symbolmatches' as const, matches })))
                .subscribe(observer)
        )
        subscriptions.add(
            observeMessages<RepositoryMatch[]>(eventSource, 'repomatches')
                .pipe(map(matches => ({ type: 'repomatches' as const, matches })))
                .subscribe(observer)
        )
        subscriptions.add(
            observeMessages<CommitMatch[]>(eventSource, 'commitmatches')
                .pipe(map(matches => ({ type: 'commitmatches' as const, matches })))
                .subscribe(observer)
        )
        subscriptions.add(
            observeMessages<Filter[]>(eventSource, 'filters')
                .pipe(map(filters => ({ type: 'filters' as const, filters })))
                .subscribe(observer)
        )
        subscriptions.add(
            observeMessages<Alert>(eventSource, 'alert')
                .pipe(map(alert => ({ type: 'alert' as const, alert })))
                .subscribe(observer)
        )
        subscriptions.add(
            fromEvent(eventSource, 'done').subscribe(() => {
                observer.complete()
                eventSource.close()
            })
        )
        subscriptions.add(
            fromEvent(eventSource, 'error').subscribe(error => {
                observer.error(error)
                eventSource.close()
            })
        )
        return () => {
            subscriptions.unsubscribe()
            eventSource.close()
        }
    })
}
