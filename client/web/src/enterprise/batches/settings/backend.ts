import { gql } from 'graphql-request'
import { map, mapTo } from 'rxjs/operators'
import { dataOrThrowErrors } from '../../../../../shared/src/graphql/graphql'
import { requestGraphQL } from '../../../backend/graphql'
import {
    BatchChangesCredentialFields,
    CreateBatchChangesCredentialResult,
    CreateBatchChangesCredentialVariables,
    DeleteBatchChangesCredentialResult,
    DeleteBatchChangesCredentialVariables,
    Scalars,
} from '../../../graphql-operations'

export const batchChangesCredentialFieldsFragment = gql`
    fragment BatchChangesCredentialFields on BatchChangesCredential {
        id
        createdAt
        sshPublicKey
    }
`

export function createBatchChangesCredential(
    args: CreateBatchChangesCredentialVariables
): Promise<BatchChangesCredentialFields> {
    return requestGraphQL<CreateBatchChangesCredentialResult, CreateBatchChangesCredentialVariables>(
        gql`
            mutation CreateBatchChangesCredential(
                $user: ID!
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
        `,
        args
    )
        .pipe(
            map(dataOrThrowErrors),
            map(data => data.createBatchChangesCredential)
        )
        .toPromise()
}

export function deleteBatchChangesCredential(id: Scalars['ID']): Promise<void> {
    return requestGraphQL<DeleteBatchChangesCredentialResult, DeleteBatchChangesCredentialVariables>(
        gql`
            mutation DeleteBatchChangesCredential($id: ID!) {
                deleteBatchChangesCredential(batchChangesCredential: $id) {
                    alwaysNil
                }
            }
        `,
        { id }
    )
        .pipe(map(dataOrThrowErrors), mapTo(undefined))
        .toPromise()
}

export const queryUserBatchChangesCodeHosts = gql`
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
