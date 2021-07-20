import { Observable } from 'rxjs'
import { map } from 'rxjs/operators'

import { dataOrThrowErrors, gql } from '@sourcegraph/shared/src/graphql/graphql'

import { requestGraphQL } from '../../../backend/graphql'
import {
    BatchSpecExecutionByIDResult,
    BatchSpecExecutionByIDVariables,
    BatchSpecExecutionFields,
    Scalars,
} from '../../../graphql-operations'

const batchSpecExecutionFieldsFragment = gql`
    fragment BatchSpecExecutionFields on BatchSpecExecution {
        id
        inputSpec
        state
        createdAt
        startedAt
        finishedAt
        failure
        steps {
            setup {
                ...BatchSpecExecutionLogEntryFields
            }
            srcPreview {
                ...BatchSpecExecutionLogEntryFields
            }
            teardown {
                ...BatchSpecExecutionLogEntryFields
            }
        }
        placeInQueue
        batchSpec {
            applyURL
        }
        initiator {
            id
            url
            displayName
        }
        namespace {
            id
            url
            namespaceName
        }
    }

    fragment BatchSpecExecutionLogEntryFields on ExecutionLogEntry {
        key
        command
        startTime
        exitCode
        durationMilliseconds
        out
    }
`

export const fetchBatchSpecExecution = (id: Scalars['ID']): Observable<BatchSpecExecutionFields | null> =>
    requestGraphQL<BatchSpecExecutionByIDResult, BatchSpecExecutionByIDVariables>(
        gql`
            query BatchSpecExecutionByID($id: ID!) {
                node(id: $id) {
                    __typename
                    ...BatchSpecExecutionFields
                }
            }
            ${batchSpecExecutionFieldsFragment}
        `,
        { id }
    ).pipe(
        map(dataOrThrowErrors),
        map(({ node }) => {
            if (!node) {
                return null
            }
            if (node.__typename !== 'BatchSpecExecution') {
                throw new Error(`Node is a ${node.__typename}, not a BatchSpecExecution`)
            }
            return node
        })
    )
