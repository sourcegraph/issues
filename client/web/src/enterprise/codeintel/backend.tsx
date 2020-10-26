import * as GQL from '../../../../shared/src/graphql/schema'
import {
    dataOrThrowErrors,
    gql,
    createInvalidGraphQLMutationResponseError,
} from '../../../../shared/src/graphql/graphql'
import { map } from 'rxjs/operators'
import { Observable } from 'rxjs'
import { queryGraphQL, mutateGraphQL } from '../../backend/graphql'

// Create an expected subtype including only the fields that we use in this component so
// that storybook tests do not need to define a full IGitTree type (which is very large).
export type Upload = Omit<GQL.ILSIFUpload, '__typename' | 'projectRoot'> & {
    projectRoot: {
        url: string
        path: string
        repository: {
            url: string
            name: string
        }
        commit: {
            url: string
            oid: string
            abbreviatedOID: string
        }
    } | null
}

export type UploadConnection = Omit<GQL.ILSIFUploadConnection, '__typename' | 'nodes'> & {
    nodes: Upload[]
}

/**
 * Return LSIF uploads. If a repository is given, only uploads for that repository will be returned. Otherwise,
 * uploads across all repositories are returned.
 */
export function fetchLsifUploads({
    repository,
    query,
    state,
    isLatestForRepo,
    first,
    after,
}: { repository?: string } & GQL.ILsifUploadsOnRepositoryArguments): Observable<UploadConnection> {
    if (!repository) {
        return queryGraphQL(
            gql`
                query LsifUploads(
                    $state: LSIFUploadState
                    $isLatestForRepo: Boolean
                    $first: Int
                    $after: String
                    $query: String
                ) {
                    lsifUploads(
                        query: $query
                        state: $state
                        isLatestForRepo: $isLatestForRepo
                        first: $first
                        after: $after
                    ) {
                        nodes {
                            id
                            state
                            projectRoot {
                                url
                                path
                                repository {
                                    url
                                    name
                                }
                                commit {
                                    url
                                    oid
                                    abbreviatedOID
                                }
                            }
                            inputCommit
                            inputRoot
                            inputIndexer
                            uploadedAt
                            startedAt
                            finishedAt
                            placeInQueue
                        }

                        totalCount
                        pageInfo {
                            endCursor
                            hasNextPage
                        }
                    }
                }
            `,
            { query, state, isLatestForRepo, first, after }
        ).pipe(
            map(dataOrThrowErrors),
            map(({ lsifUploads }) => lsifUploads)
        )
    }

    return queryGraphQL(
        gql`
            query LsifUploadsWithRepo(
                $repository: ID!
                $state: LSIFUploadState
                $isLatestForRepo: Boolean
                $first: Int
                $after: String
                $query: String
            ) {
                node(id: $repository) {
                    __typename
                    ... on Repository {
                        lsifUploads(
                            query: $query
                            state: $state
                            isLatestForRepo: $isLatestForRepo
                            first: $first
                            after: $after
                        ) {
                            nodes {
                                id
                                state
                                projectRoot {
                                    url
                                    path
                                    repository {
                                        url
                                        name
                                    }
                                    commit {
                                        url
                                        oid
                                        abbreviatedOID
                                    }
                                }
                                inputCommit
                                inputRoot
                                inputIndexer
                                uploadedAt
                                startedAt
                                finishedAt
                                placeInQueue
                            }

                            totalCount
                            pageInfo {
                                endCursor
                                hasNextPage
                            }
                        }
                    }
                }
            }
        `,
        { repository, query, state, isLatestForRepo, first, after }
    ).pipe(
        map(dataOrThrowErrors),
        map(({ node }) => {
            if (!node) {
                throw new Error('Invalid repository')
            }
            if (node.__typename !== 'Repository') {
                throw new Error(`The given ID is a ${node.__typename}, not a Repository`)
            }

            return node.lsifUploads
        })
    )
}

export function fetchLsifUpload({ id }: { id: string }): Observable<Upload | null> {
    return queryGraphQL(
        gql`
            query LsifUpload($id: ID!) {
                node(id: $id) {
                    __typename
                    ... on LSIFUpload {
                        id
                        projectRoot {
                            url
                            path
                            repository {
                                url
                                name
                            }
                            commit {
                                url
                                oid
                                abbreviatedOID
                            }
                        }
                        inputCommit
                        inputRoot
                        inputIndexer
                        state
                        failure
                        uploadedAt
                        startedAt
                        finishedAt
                        isLatestForRepo
                        placeInQueue
                    }
                }
            }
        `,
        { id }
    ).pipe(
        map(dataOrThrowErrors),
        map(({ node }) => {
            if (!node) {
                return null
            }
            if (node.__typename !== 'LSIFUpload') {
                throw new Error(`The given ID is a ${node.__typename}, not an LSIFUpload`)
            }

            return node
        })
    )
}

export function deleteLsifUpload({ id }: { id: string }): Observable<void> {
    return mutateGraphQL(
        gql`
            mutation DeleteLsifUpload($id: ID!) {
                deleteLSIFUpload(id: $id) {
                    alwaysNil
                }
            }
        `,
        { id }
    ).pipe(
        map(dataOrThrowErrors),
        map(data => {
            if (!data.deleteLSIFUpload) {
                throw createInvalidGraphQLMutationResponseError('DeleteLsifUpload')
            }
        })
    )
}

// Create an expected subtype including only the fields that we use in this component so
// that storybook tests do not need to define a full IGitTree type (which is very large).
export type Index = Omit<GQL.ILSIFIndex, '__typename' | 'projectRoot'> & {
    projectRoot: {
        url: string
        path: string
        repository: {
            url: string
            name: string
        }
        commit: {
            url: string
            oid: string
            abbreviatedOID: string
        }
    } | null
}

export type IndexConnection = Omit<GQL.ILSIFIndexConnection, '__typename' | 'nodes'> & {
    nodes: Index[]
}

/**
 * Return LSIF indexes. If a repository is given, only indexes for that repository will be returned. Otherwise,
 * indexes across all repositories are returned.
 */
export function fetchLsifIndexes({
    repository,
    query,
    state,
    first,
    after,
}: { repository?: string } & GQL.ILsifIndexesOnRepositoryArguments): Observable<IndexConnection> {
    if (!repository) {
        return queryGraphQL(
            gql`
                query LsifIndexes($state: LSIFIndexState, $first: Int, $after: String, $query: String) {
                    lsifIndexes(query: $query, state: $state, first: $first, after: $after) {
                        nodes {
                            id
                            state
                            projectRoot {
                                url
                                path
                                repository {
                                    url
                                    name
                                }
                                commit {
                                    url
                                    oid
                                    abbreviatedOID
                                }
                            }
                            inputCommit
                            queuedAt
                            startedAt
                            finishedAt
                            placeInQueue
                        }

                        totalCount
                        pageInfo {
                            endCursor
                            hasNextPage
                        }
                    }
                }
            `,
            { query, state, first, after }
        ).pipe(
            map(dataOrThrowErrors),
            map(({ lsifIndexes }) => lsifIndexes)
        )
    }

    return queryGraphQL(
        gql`
            query LsifIndexesWithRepo(
                $repository: ID!
                $state: LSIFIndexState
                $first: Int
                $after: String
                $query: String
            ) {
                node(id: $repository) {
                    __typename
                    ... on Repository {
                        lsifIndexes(query: $query, state: $state, first: $first, after: $after) {
                            nodes {
                                id
                                state
                                projectRoot {
                                    url
                                    path
                                    repository {
                                        url
                                        name
                                    }
                                    commit {
                                        url
                                        oid
                                        abbreviatedOID
                                    }
                                }
                                inputCommit
                                queuedAt
                                startedAt
                                finishedAt
                                placeInQueue
                            }

                            totalCount
                            pageInfo {
                                endCursor
                                hasNextPage
                            }
                        }
                    }
                }
            }
        `,
        { repository, query, state, first, after }
    ).pipe(
        map(dataOrThrowErrors),
        map(({ node }) => {
            if (!node) {
                throw new Error('Invalid repository')
            }
            if (node.__typename !== 'Repository') {
                throw new Error(`The given ID is a ${node.__typename}, not a Repository`)
            }

            return node.lsifIndexes
        })
    )
}

export function fetchLsifIndex({ id }: { id: string }): Observable<Index | null> {
    return queryGraphQL(
        gql`
            query LsifIndex($id: ID!) {
                node(id: $id) {
                    __typename
                    ... on LSIFIndex {
                        id
                        projectRoot {
                            url
                            path
                            repository {
                                url
                                name
                            }
                            commit {
                                url
                                oid
                                abbreviatedOID
                            }
                        }
                        inputCommit
                        state
                        failure
                        queuedAt
                        startedAt
                        finishedAt
                        placeInQueue
                    }
                }
            }
        `,
        { id }
    ).pipe(
        map(dataOrThrowErrors),
        map(({ node }) => {
            if (!node) {
                return null
            }
            if (node.__typename !== 'LSIFIndex') {
                throw new Error(`The given ID is a ${node.__typename}, not an LSIFIndex`)
            }

            return node
        })
    )
}

export function deleteLsifIndex({ id }: { id: string }): Observable<void> {
    return mutateGraphQL(
        gql`
            mutation DeleteLsifIndex($id: ID!) {
                deleteLSIFIndex(id: $id) {
                    alwaysNil
                }
            }
        `,
        { id }
    ).pipe(
        map(dataOrThrowErrors),
        map(data => {
            if (!data.deleteLSIFIndex) {
                throw createInvalidGraphQLMutationResponseError('DeleteLsifIndex')
            }
        })
    )
}
