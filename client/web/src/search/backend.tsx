import { Observable, of, combineLatest, defer, from } from 'rxjs'
import { catchError, map, switchMap, publishReplay, refCount } from 'rxjs/operators'
import { dataOrThrowErrors, gql } from '../../../shared/src/graphql/graphql'
import * as GQL from '../../../shared/src/graphql/schema'
import { asError, createAggregateError, ErrorLike } from '../../../shared/src/util/errors'
import { memoizeObservable } from '../../../shared/src/util/memoizeObservable'
import { mutateGraphQL, queryGraphQL, requestGraphQL } from '../backend/graphql'
import { SearchSuggestion } from '../../../shared/src/search/suggestions'
import { Remote } from 'comlink'
import { FlatExtensionHostAPI } from '../../../shared/src/api/contract'
import { wrapRemoteObservable } from '../../../shared/src/api/client/api/common'
import { DeployType } from '../jscontext'
import { SearchPatternType, EventLogsDataResult, EventLogsDataVariables } from '../graphql-operations'
import * as SearchStream from './stream'

export function search(
    query: string,
    version: string,
    patternType: SearchPatternType,
    versionContext: string | undefined,
    extensionHostPromise: Promise<Remote<FlatExtensionHostAPI>>
): Observable<GQL.ISearchResults | ErrorLike> {
    const transformedQuery = from(extensionHostPromise).pipe(
        switchMap(extensionHost => wrapRemoteObservable(extensionHost.transformSearchQuery(query)))
    )

    return transformedQuery.pipe(
        switchMap(query =>
            queryGraphQL(
                gql`
                    query Search(
                        $query: String!
                        $version: SearchVersion!
                        $patternType: SearchPatternType!
                        $versionContext: String
                    ) {
                        search(
                            query: $query
                            version: $version
                            patternType: $patternType
                            versionContext: $versionContext
                        ) {
                            results {
                                __typename
                                limitHit
                                matchCount
                                approximateResultCount
                                missing {
                                    name
                                }
                                cloning {
                                    name
                                }
                                repositoriesCount
                                timedout {
                                    name
                                }
                                indexUnavailable
                                dynamicFilters {
                                    value
                                    label
                                    count
                                    limitHit
                                    kind
                                }
                                results {
                                    __typename
                                    ... on Repository {
                                        id
                                        name
                                        # TODO: Make this a proper fragment, blocked by https://github.com/graph-gophers/graphql-go/issues/241.
                                        # beginning of genericSearchResultInterfaceFields inline fragment
                                        label {
                                            html
                                        }
                                        url
                                        icon
                                        detail {
                                            html
                                        }
                                        matches {
                                            url
                                            body {
                                                text
                                                html
                                            }
                                            highlights {
                                                line
                                                character
                                                length
                                            }
                                        }
                                        # end of genericSearchResultInterfaceFields inline fragment
                                    }
                                    ... on FileMatch {
                                        file {
                                            path
                                            url
                                            commit {
                                                oid
                                            }
                                        }
                                        repository {
                                            name
                                            url
                                        }
                                        revSpec {
                                            __typename
                                            ... on GitRef {
                                                displayName
                                                url
                                            }
                                            ... on GitRevSpecExpr {
                                                expr
                                                object {
                                                    commit {
                                                        url
                                                    }
                                                }
                                            }
                                            ... on GitObject {
                                                abbreviatedOID
                                                commit {
                                                    url
                                                }
                                            }
                                        }
                                        limitHit
                                        symbols {
                                            name
                                            containerName
                                            url
                                            kind
                                        }
                                        lineMatches {
                                            preview
                                            lineNumber
                                            offsetAndLengths
                                        }
                                    }
                                    ... on CommitSearchResult {
                                        # TODO: Make this a proper fragment, blocked by https://github.com/graph-gophers/graphql-go/issues/241.
                                        # beginning of genericSearchResultInterfaceFields inline fragment
                                        label {
                                            html
                                        }
                                        url
                                        icon
                                        detail {
                                            html
                                        }
                                        matches {
                                            url
                                            body {
                                                text
                                                html
                                            }
                                            highlights {
                                                line
                                                character
                                                length
                                            }
                                        }
                                        # end of genericSearchResultInterfaceFields inline fragment
                                    }
                                }
                                alert {
                                    title
                                    description
                                    proposedQueries {
                                        description
                                        query
                                    }
                                }
                                elapsedMilliseconds
                            }
                        }
                    }
                `,
                { query, version, patternType, versionContext }
            ).pipe(
                map(({ data, errors }) => {
                    if (!data || !data.search || !data.search.results) {
                        throw createAggregateError(errors)
                    }
                    return data.search.results
                }),
                catchError(error => [asError(error)])
            )
        )
    )
}

export function searchStream(
    query: string,
    version: string,
    patternType: SearchPatternType,
    versionContext: string | undefined,
    extensionHostPromise: Promise<Remote<FlatExtensionHostAPI>>
): Observable<GQL.ISearchResults | ErrorLike> {
    const transformedQuery = from(extensionHostPromise).pipe(
        switchMap(extensionHost => wrapRemoteObservable(extensionHost.transformSearchQuery(query)))
    )

    return transformedQuery.pipe(
        switchMap(query =>
            SearchStream.search(query, version, patternType, versionContext).pipe(
                SearchStream.switchToGQLISearchResults
            )
        )
    )
}

/**
 * Repogroups to include in search suggestions.
 *
 * defer() is used here to avoid calling queryGraphQL in tests,
 * which would fail when accessing window.context.xhrHeaders.
 */
const repogroupSuggestions = defer(() =>
    queryGraphQL(gql`
        query RepoGroups {
            repoGroups {
                __typename
                name
            }
        }
    `)
).pipe(
    map(dataOrThrowErrors),
    map(({ repoGroups }) => repoGroups),
    publishReplay(1),
    refCount()
)

export function fetchSuggestions(query: string): Observable<SearchSuggestion[]> {
    return combineLatest([
        repogroupSuggestions,
        queryGraphQL(
            gql`
                query SearchSuggestions($query: String!) {
                    search(query: $query) {
                        suggestions {
                            __typename
                            ... on Repository {
                                name
                            }
                            ... on File {
                                path
                                name
                                isDirectory
                                url
                                repository {
                                    name
                                }
                            }
                            ... on Symbol {
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
                                }
                            }
                        }
                    }
                }
            `,
            { query }
        ).pipe(
            map(({ data, errors }) => {
                if (!data?.search?.suggestions) {
                    throw createAggregateError(errors)
                }
                return data.search.suggestions
            })
        ),
    ]).pipe(map(([repogroups, dynamicSuggestions]) => [...repogroups, ...dynamicSuggestions]))
}

export function fetchReposByQuery(query: string): Observable<{ name: string; url: string }[]> {
    return queryGraphQL(
        gql`
            query ReposByQuery($query: String!) {
                search(query: $query) {
                    results {
                        repositories {
                            name
                            url
                        }
                    }
                }
            }
        `,
        { query }
    ).pipe(
        map(({ data, errors }) => {
            if (!data || !data.search || !data.search.results || !data.search.results.repositories) {
                throw createAggregateError(errors)
            }
            return data.search.results.repositories
        })
    )
}

const savedSearchFragment = gql`
    fragment SavedSearchFields on SavedSearch {
        id
        description
        notify
        notifySlack
        query
        namespace {
            __typename
            id
            namespaceName
        }
        slackWebhookURL
    }
`

export function fetchSavedSearches(): Observable<GQL.ISavedSearch[]> {
    return queryGraphQL(gql`
        query savedSearches {
            savedSearches {
                ...SavedSearchFields
            }
        }
        ${savedSearchFragment}
    `).pipe(
        map(({ data, errors }) => {
            if (!data || !data.savedSearches) {
                throw createAggregateError(errors)
            }
            return data.savedSearches
        })
    )
}

export function fetchSavedSearch(id: GQL.ID): Observable<GQL.ISavedSearch> {
    return queryGraphQL(
        gql`
            query SavedSearch($id: ID!) {
                node(id: $id) {
                    ... on SavedSearch {
                        id
                        description
                        query
                        notify
                        notifySlack
                        slackWebhookURL
                        namespace {
                            id
                        }
                    }
                }
            }
        `,
        { id }
    ).pipe(
        map(dataOrThrowErrors),
        map(data => data.node as GQL.ISavedSearch)
    )
}

export function createSavedSearch(
    description: string,
    query: string,
    notify: boolean,
    notifySlack: boolean,
    userId: GQL.ID | null,
    orgId: GQL.ID | null
): Observable<void> {
    return mutateGraphQL(
        gql`
            mutation CreateSavedSearch(
                $description: String!
                $query: String!
                $notifyOwner: Boolean!
                $notifySlack: Boolean!
                $userID: ID
                $orgID: ID
            ) {
                createSavedSearch(
                    description: $description
                    query: $query
                    notifyOwner: $notifyOwner
                    notifySlack: $notifySlack
                    userID: $userID
                    orgID: $orgID
                ) {
                    ...SavedSearchFields
                }
            }
            ${savedSearchFragment}
        `,
        {
            description,
            query,
            notifyOwner: notify,
            notifySlack,
            userID: userId,
            orgID: orgId,
        }
    ).pipe(
        map(dataOrThrowErrors),
        map(() => undefined)
    )
}

export function updateSavedSearch(
    id: GQL.ID,
    description: string,
    query: string,
    notify: boolean,
    notifySlack: boolean,
    userId: GQL.ID | null,
    orgId: GQL.ID | null
): Observable<void> {
    return mutateGraphQL(
        gql`
            mutation UpdateSavedSearch(
                $id: ID!
                $description: String!
                $query: String!
                $notifyOwner: Boolean!
                $notifySlack: Boolean!
                $userID: ID
                $orgID: ID
            ) {
                updateSavedSearch(
                    id: $id
                    description: $description
                    query: $query
                    notifyOwner: $notifyOwner
                    notifySlack: $notifySlack
                    userID: $userID
                    orgID: $orgID
                ) {
                    ...SavedSearchFields
                }
            }
            ${savedSearchFragment}
        `,
        {
            id,
            description,
            query,
            notifyOwner: notify,
            notifySlack,
            userID: userId,
            orgID: orgId,
        }
    ).pipe(
        map(dataOrThrowErrors),
        map(() => undefined)
    )
}

export function deleteSavedSearch(id: GQL.ID): Observable<void> {
    return mutateGraphQL(
        gql`
            mutation DeleteSavedSearch($id: ID!) {
                deleteSavedSearch(id: $id) {
                    alwaysNil
                }
            }
        `,
        { id }
    ).pipe(
        map(dataOrThrowErrors),
        map(() => undefined)
    )
}

export const highlightCode = memoizeObservable(
    (context: {
        code: string
        fuzzyLanguage: string
        disableTimeout: boolean
        isLightTheme: boolean
    }): Observable<string> =>
        queryGraphQL(
            gql`
                query highlightCode(
                    $code: String!
                    $fuzzyLanguage: String!
                    $disableTimeout: Boolean!
                    $isLightTheme: Boolean!
                ) {
                    highlightCode(
                        code: $code
                        fuzzyLanguage: $fuzzyLanguage
                        disableTimeout: $disableTimeout
                        isLightTheme: $isLightTheme
                    )
                }
            `,
            context
        ).pipe(
            map(({ data, errors }) => {
                if (!data || !data.highlightCode) {
                    throw createAggregateError(errors)
                }
                return data.highlightCode
            })
        ),
    context =>
        `${context.code}:${context.fuzzyLanguage}:${String(context.disableTimeout)}:${String(context.isLightTheme)}`
)

/**
 * Returns true if search performance and accuracy are limited because this is a
 * single-node Docker deployment that is configured with more than 100 repositories.
 */
export function shouldDisplayPerformanceWarning(deployType: DeployType): Observable<boolean> {
    if (deployType !== 'docker-container') {
        return of(false)
    }
    const manyReposWarningLimit = 100
    return queryGraphQL(
        gql`
            query ManyReposWarning($first: Int) {
                repositories(first: $first) {
                    nodes {
                        id
                    }
                }
            }
        `,
        {
            first: manyReposWarningLimit + 1,
        }
    ).pipe(
        map(dataOrThrowErrors),
        map(data => (data.repositories.nodes || []).length > manyReposWarningLimit)
    )
}

export interface EventLogResult {
    totalCount: number
    nodes: { argument: string | null; timestamp: string; url: string }[]
    pageInfo: { hasNextPage: boolean }
}

function fetchEvents(userId: GQL.ID, first: number, eventName: string): Observable<EventLogResult | null> {
    if (!userId) {
        return of(null)
    }

    const result = requestGraphQL<EventLogsDataResult, EventLogsDataVariables>(
        gql`
            query EventLogsData($userId: ID!, $first: Int, $eventName: String!) {
                node(id: $userId) {
                    ... on User {
                        eventLogs(first: $first, eventName: $eventName) {
                            nodes {
                                argument
                                timestamp
                                url
                            }
                            pageInfo {
                                hasNextPage
                            }
                            totalCount
                        }
                    }
                }
            }
        `,
        { userId, first: first ?? null, eventName }
    )

    return result.pipe(
        map(dataOrThrowErrors),
        map(
            (data: EventLogsDataResult): EventLogResult => {
                if (!data.node) {
                    throw new Error('User not found')
                }
                return data.node.eventLogs
            }
        )
    )
}

export function fetchRecentSearches(userId: GQL.ID, first: number): Observable<EventLogResult | null> {
    return fetchEvents(userId, first, 'SearchResultsQueried')
}

export function fetchRecentFileViews(userId: GQL.ID, first: number): Observable<EventLogResult | null> {
    return fetchEvents(userId, first, 'ViewBlob')
}
