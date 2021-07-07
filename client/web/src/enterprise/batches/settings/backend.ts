import { Observable } from 'rxjs'
import { map } from 'rxjs/operators'

import { dataOrThrowErrors, gql } from '@sourcegraph/shared/src/graphql/graphql'

import { watchQuery } from '../../../backend/graphql'
import {
    BatchChangesCodeHostsFields,
    GlobalBatchChangesCodeHostsResult,
    GlobalBatchChangesCodeHostsVariables,
    UserBatchChangesCodeHostsResult,
    UserBatchChangesCodeHostsVariables,
} from '../../../graphql-operations'

export const batchChangesCredentialFieldsFragment = gql`
    fragment BatchChangesCredentialFields on BatchChangesCredential {
        id
        sshPublicKey
        isSiteCredential
    }
`

export const CREATE_BATCH_CHANGES_CREDENTIAL = gql`
    mutation CreateBatchChangesCredential(
        $user: ID
        $credential: String!
        $externalServiceKind: ExternalServiceKind!
        $externalServiceURL: String!
    ) {
        createBatchChangesCredential(
            user: $user
            credential: $credential
            externalServiceKind: $externalServiceKind
            externalServiceURL: $externalServiceURL
        ) {
            ...BatchChangesCredentialFields
        }
    }

    ${batchChangesCredentialFieldsFragment}
`

export const DELETE_BATCH_CHANGES_CREDENTIAL = gql`
    mutation DeleteBatchChangesCredential($id: ID!) {
        deleteBatchChangesCredential(batchChangesCredential: $id) {
            alwaysNil
        }
    }
`

const batchChangesCodeHostsFieldsFragment = gql`
    fragment BatchChangesCodeHostsFields on BatchChangesCodeHostConnection {
        totalCount
        pageInfo {
            hasNextPage
            endCursor
        }
        nodes {
            ...BatchChangesCodeHostFields
        }
    }

    fragment BatchChangesCodeHostFields on BatchChangesCodeHost {
        externalServiceKind
        externalServiceURL
        requiresSSH
        credential {
            ...BatchChangesCredentialFields
        }
    }

    ${batchChangesCredentialFieldsFragment}
`

export const queryUserBatchChangesCodeHosts = ({
    user,
    first,
    after,
}: UserBatchChangesCodeHostsVariables): Observable<BatchChangesCodeHostsFields> =>
    watchQuery<UserBatchChangesCodeHostsResult, UserBatchChangesCodeHostsVariables>(
        gql`
            query UserBatchChangesCodeHosts($user: ID!, $first: Int, $after: String) {
                node(id: $user) {
                    __typename
                    ... on User {
                        id
                        batchChangesCodeHosts(first: $first, after: $after) {
                            ...BatchChangesCodeHostsFields
                        }
                    }
                }
            }

            ${batchChangesCodeHostsFieldsFragment}
        `,
        {
            user,
            first,
            after,
        }
    ).pipe(
        map(dataOrThrowErrors),
        map(data => {
            if (data.node === null) {
                throw new Error('User not found')
            }
            if (data.node.__typename !== 'User') {
                throw new Error(`Node is a ${data.node.__typename}, not a User`)
            }
            return data.node.batchChangesCodeHosts
        })
    )

export const queryGlobalBatchChangesCodeHosts = ({
    first,
    after,
}: GlobalBatchChangesCodeHostsVariables): Observable<BatchChangesCodeHostsFields> =>
    watchQuery<GlobalBatchChangesCodeHostsResult, GlobalBatchChangesCodeHostsVariables>(
        gql`
            query GlobalBatchChangesCodeHosts($first: Int, $after: String) {
                batchChangesCodeHosts(first: $first, after: $after) {
                    ...BatchChangesCodeHostsFields
                }
            }

            ${batchChangesCodeHostsFieldsFragment}
        `,
        {
            first,
            after,
        }
    ).pipe(
        map(dataOrThrowErrors),
        map(data => data.batchChangesCodeHosts)
    )
