/* eslint-disable id-length */
import { Observable, fromEvent, Subscription, OperatorFunction, pipe, Subscriber } from 'rxjs'
import { defaultIfEmpty, map, scan } from 'rxjs/operators'
import * as GQL from '../../../shared/src/graphql/schema'
import { SearchPatternType } from '../graphql-operations'

// This is an initial proof of concept implementation of search streaming.
// The protocol and implementation is still in the design phase. Feel free to
// change anything and everything here. We are iteratively improving this
// until it is no longer a proof of concept and instead works well.

export type SearchEvent =
    | { type: 'filematches'; data: FileMatch[] }
    | { type: 'repomatches'; data: RepositoryMatch[] }
    | { type: 'commitmatches'; data: CommitMatch[] }
    | { type: 'symbolmatches'; data: FileSymbolMatch[] }
    | { type: 'progress'; data: Progress }
    | { type: 'filters'; data: Filter[] }
    | { type: 'alert'; data: Alert }
    | { type: 'error'; data: Error }
    | { type: 'done'; data: {} }

interface FileMatch {
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

/**
 * An aggregate type representing a progress update.
 * Should be replaced when a new ones come in.
 */
export interface Progress {
    /**
     * True if this is the final progress update for this stream
     */
    done: boolean

    /**
     * The number of repositories matching the repo: filter. Is set once they
     * are resolved.
     */
    repositoriesCount?: number

    // The number of non-overlapping matches. If skipped is non-empty, then
    // this is a lower bound.
    matchCount: number

    // Wall clock time in milliseconds for this search.
    durationMs: number

    /**
     * A description of shards or documents that were skipped. This has a
     * deterministic ordering. More important reasons will be listed first. If
     * a search is repeated, the final skipped list will be the same.
     * However, within a search stream when a new skipped reason is found, it
     * may appear anywhere in the list.
     */
    skipped: Skipped[]
}

interface Skipped {
    /**
     * Why a document/shard/repository was skipped. We group counts by reason.
     *
     * - document-match-limit :: we found too many matches in a document, so we stopped searching it.
     * - shard-match-limit :: we found too many matches in a shard/repository, so we stopped searching it.
     * - repository-limit :: we did not search a repository because the set of repositories to search was too large.
     * - shard-timeout :: we ran out of time before searching a shard/repository.
     * - repository-cloning :: we could not search a repository because it is not cloned.
     * - repository-missing :: we could not search a repository because it is not cloned and we failed to find it on the remote code host.
     * - excluded-fork :: we did not search a repository because it is a fork.
     * - excluded-archive :: we did not search a repository because it is archived.
     */
    reason:
        | 'document-match-limit'
        | 'shard-match-limit'
        | 'repository-limit'
        | 'shard-timedout'
        | 'repository-cloning'
        | 'repository-missing'
        | 'excluded-fork'
        | 'excluded-archive'
    /**
     * A short message. eg 1,200 timed out.
     */
    title: string
    /**
     * A message to show the user. Usually includes information explaining the reason,
     * count as well as a sample of the missing items.
     */
    message: string
    severity: 'info' | 'warn'
    /**
     * a suggested query expression to remedy the skip. eg "archived:yes" or "timeout:2m".
     */
    suggested?: {
        title: string
        queryExpression: string
    }
}

export interface Filter {
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

export interface AggregateStreamingSearchResults {
    results: GQL.SearchResult[]
    alert?: Alert
    filters: Filter[]
    progress: Progress
}

const emptyAggregateResults: AggregateStreamingSearchResults = {
    results: [],
    filters: [],
    progress: {
        done: false,
        durationMs: 0,
        matchCount: 0,
        skipped: [],
    },
}

/**
 * Converts a stream of SearchEvents into AggregateStreamingSearchResults
 */
const switchAggregateSearchResults: OperatorFunction<SearchEvent, AggregateStreamingSearchResults> = pipe(
    scan((results: AggregateStreamingSearchResults, newEvent: SearchEvent) => {
        switch (newEvent.type) {
            case 'filematches':
                return {
                    ...results,
                    // File matches are additive
                    results: results.results.concat(newEvent.data.map(toGQLFileMatch)),
                }

            case 'repomatches':
                return {
                    ...results,
                    // Repository matches are additive
                    results: results.results.concat(newEvent.data.map(toGQLRepositoryMatch)),
                }

            case 'commitmatches':
                return {
                    ...results,
                    // Commit matches are additive
                    results: results.results.concat(newEvent.data.map(toGQLCommitMatch)),
                }

            case 'symbolmatches':
                return {
                    ...results,
                    // Symbol matches are additive
                    results: results.results.concat(newEvent.data.map(toGQLSymbolMatch)),
                }

            case 'progress':
                return {
                    ...results,
                    // Progress updates replace
                    progress: newEvent.data,
                }

            case 'filters':
                return {
                    ...results,
                    // New filter results replace all previous ones
                    filters: newEvent.data,
                }

            case 'alert':
                return {
                    ...results,
                    alert: newEvent.data,
                }

            default:
                return results
        }
    }, emptyAggregateResults),
    defaultIfEmpty(emptyAggregateResults)
)

const observeMessages = <T extends SearchEvent>(
    type: T['type'],
    eventSource: EventSource,
    observer: Subscriber<SearchEvent>
): Subscription =>
    fromEvent(eventSource, type)
        .pipe(
            map((event: Event) => {
                if (!(event instanceof MessageEvent)) {
                    throw new TypeError(`internal error: expected MessageEvent in streaming search ${type}`)
                }
                try {
                    const parsedData = JSON.parse(event.data) as T['data']
                    return parsedData
                } catch {
                    throw new Error(`Could not parse ${type} message data in streaming search`)
                }
            }),
            map(data => ({ type, data } as T))
        )
        .subscribe(observer)

type MessageHandler<EventType extends SearchEvent['type'] = SearchEvent['type']> = (
    type: EventType,
    eventSource: EventSource,
    observer: Subscriber<SearchEvent>
) => Subscription

const messageHandlers: {
    [EventType in SearchEvent['type']]: MessageHandler<EventType>
} = {
    done: (type, eventSource, observer) =>
        fromEvent(eventSource, type).subscribe(() => {
            observer.complete()
            eventSource.close()
        }),
    error: (type, eventSource, observer) =>
        fromEvent(eventSource, type)
            .pipe(
                map(
                    (event: Event): Error => {
                        if (event instanceof MessageEvent && typeof event.data === 'string') {
                            // Error sent by the server as a named event
                            return new Error(event.data)
                        }
                        if (event instanceof ErrorEvent) {
                            // Client-side error (eg. failed to connect)
                            const { lineno, filename, message, colno, error } = event
                            return Object.assign(new Error(message), { lineno, filename, message, colno, error })
                        }
                        return Object.assign(new Error('Unknown error in streaming search'), { event })
                    }
                )
            )
            .subscribe(error => {
                observer.error(error)
                eventSource.close()
            }),
    filematches: observeMessages,
    symbolmatches: observeMessages,
    repomatches: observeMessages,
    commitmatches: observeMessages,
    progress: observeMessages,
    filters: observeMessages,
    alert: observeMessages,
}

/**
 * Initiates a streaming search. This is a type safe wrapper around Sourcegraph's streaming search API (using Server Sent Events).
 * The observable will emit each event returned from the backend.
 *
 * @param query the search query to send to Sourcegraph's backend.
 */
function search(
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
        for (const [eventType, handleMessages] of Object.entries(messageHandlers)) {
            subscriptions.add(
                (handleMessages as MessageHandler)(eventType as SearchEvent['type'], eventSource, observer)
            )
        }
        return () => {
            subscriptions.unsubscribe()
            eventSource.close()
        }
    })
}

/** Initiate a streaming search and aggregate the results */
export function aggregateStreamingSearch(
    query: string,
    version: string,
    patternType: SearchPatternType,
    versionContext: string | undefined
): Observable<AggregateStreamingSearchResults> {
    return search(query, version, patternType, versionContext).pipe(switchAggregateSearchResults)
}
