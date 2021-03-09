import { queryUserBatchChangesCodeHosts } from './backend'
import { QueryObserverResult, useQuery } from 'react-query'
import { BatchChangesCodeHostsFields, UserAreaUserFields } from '../../../graphql-operations'

interface BatchChanges {
    user: Pick<UserAreaUserFields, 'id'>
    node: {
        batchChangesCodeHosts: {
            nodes: BatchChangesCodeHostsFields
        }
    }
}

export const useBatchChanges = (userID: string): QueryObserverResult => {
    const result = useQuery<BatchChanges, unknown, BatchChangesCodeHostsFields>(
        [queryUserBatchChangesCodeHosts, { user: userID, first: 10, after: '1' }],
        {
            select: data => data.node.batchChangesCodeHosts.nodes,
        }
    )

    return result
}
