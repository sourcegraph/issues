import {
    dataOrThrowErrors,
    gql,
    createInvalidGraphQLMutationResponseError,
} from '../../../../../shared/src/graphql/graphql'
import { map } from 'rxjs/operators'
import { Observable } from 'rxjs'
import { queryGraphQL, mutateGraphQL } from '../../../backend/graphql'
import {
    DeleteLsifUploadForRepoResult,
    LsifUploadsForRepoVariables,
    LsifUploadsForRepoResult,
} from '../../../graphql-operations'

/**
 * Fetch LSIF uploads for a repository.
 */
export function fetchLsifUploads({
    repository,
    query,
    state,
    isLatestForRepo,
    first,
    after,
}: LsifUploadsForRepoVariables): Observable<
    (LsifUploadsForRepoResult['node'] & { __typename: 'Repository' })['lsifUploads']
> {
    return queryGraphQL<LsifUploadsForRepoResult>(
        gql`
            query LsifUploadsForRepo(
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
                                    commit {
                                        abbreviatedOID
                                        url
                                    }
                                    path
                                    url
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

/**
 * Delete an LSIF upload by id.
 */
export function deleteLsifUpload({ id }: { id: string }): Observable<void> {
    return mutateGraphQL<DeleteLsifUploadForRepoResult>(
        gql`
            mutation DeleteLsifUploadForRepo($id: ID!) {
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
