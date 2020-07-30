import * as GQL from '../../../../shared/src/graphql/schema'
import {
    dataOrThrowErrors,
    gql,
    createInvalidGraphQLMutationResponseError,
} from '../../../../shared/src/graphql/graphql'
import { map } from 'rxjs/operators'
import { Observable } from 'rxjs'
import { queryGraphQL, mutateGraphQL } from '../../backend/graphql'
import {
    LsifUploadsResult,
    LsifUploadsWithRepoResult,
    LsifUploadResult,
    LsifUploadConnectionFields,
    LsifIndexResult,
    LsifIndexesResult,
    LsifIndexesVariables,
    LsifIndexesWithRepoResult,
    LsifIndexesWithRepoVariables,
    LsifIndexConnectionFields,
    LsifUploadsVariables,
    LsifUploadsWithRepoVariables,
    DeleteLsifUploadResult,
    DeleteLsifIndexResult,
} from '../../graphql-operations'

// Create an expected subtype including only the fields that we use in this component so
// that storybook tests do not need to define a full IGitTree type (which is very large).
export type Upload = Omit<GQL.LSIFUpload, '__typename' | 'projectRoot'> & {
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

export type UploadConnection = Omit<GQL.LSIFUploadConnection, '__typename' | 'nodes'> & {
    nodes: Upload[]
}

const lsifUploadConnectionFields = gql`
    fragment LsifUploadConnectionFields on LSIFUploadConnection {
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
`

/**
 * Return LSIF uploads. If a repository is given, only uploads for that repository will be returned. Otherwise,
 * uploads across all repositories are returned.
 */
export function fetchLsifUploads(
    variables: LsifUploadsVariables | LsifUploadsWithRepoVariables
): Observable<LsifUploadConnectionFields> {
    if (!('repository' in variables) || !variables.repository) {
        return queryGraphQL<LsifUploadsResult>(
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
                        ...LsifUploadConnectionFields
                    }
                }
                ${lsifUploadConnectionFields}
            `,
            variables
        ).pipe(
            map(dataOrThrowErrors),
            map(({ lsifUploads }) => lsifUploads)
        )
    }

    return queryGraphQL<LsifUploadsWithRepoResult>(
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
                            ...LsifUploadConnectionFields
                        }
                    }
                }
            }
            ${lsifUploadConnectionFields}
        `,
        variables
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
    return queryGraphQL<LsifUploadResult>(
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
    return mutateGraphQL<DeleteLsifUploadResult>(
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

export type Index = LsifIndexConnectionFields['nodes'][number]

export type IndexConnection = Omit<GQL.LSIFIndexConnection, '__typename' | 'nodes'> & {
    nodes: Index[]
}

const lsifIndexFields = gql`
    fragment LsifIndexFields on LSIFIndex {
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
        failure
        startedAt
        finishedAt
        placeInQueue
    }
`

const lsifIndexConnectionFields = gql`
    fragment LsifIndexConnectionFields on LSIFIndexConnection {
        nodes {
            ...LsifIndexFields
        }
        totalCount
        pageInfo {
            endCursor
            hasNextPage
        }
    }
    ${lsifIndexFields}
`

/**
 * Return LSIF indexes. If a repository is given, only indexes for that repository will be returned. Otherwise,
 * indexes across all repositories are returned.
 */
export function fetchLsifIndexes(
    variables: LsifIndexesVariables | LsifIndexesWithRepoVariables
): Observable<LsifIndexConnectionFields> {
    if ('repository' in variables && variables.repository) {
        return queryGraphQL<LsifIndexesResult>(
            gql`
                query LsifIndexes($state: LSIFIndexState, $first: Int, $after: String, $query: String) {
                    lsifIndexes(query: $query, state: $state, first: $first, after: $after) {
                        ...LsifIndexConnectionFields
                    }
                }
                ${lsifIndexConnectionFields}
            `,
            variables
        ).pipe(
            map(dataOrThrowErrors),
            map(({ lsifIndexes }) => lsifIndexes)
        )
    }

    return queryGraphQL<LsifIndexesWithRepoResult>(
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
                            ...LsifIndexConnectionFields
                        }
                    }
                }
            }
            ${lsifIndexConnectionFields}
        `,
        variables
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
    return queryGraphQL<LsifIndexResult>(
        gql`
            query LsifIndex($id: ID!) {
                node(id: $id) {
                    __typename
                    ... on LSIFIndex {
                        ...LsifIndexFields
                    }
                }
            }
            ${lsifIndexFields}
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
    return mutateGraphQL<DeleteLsifIndexResult>(
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
