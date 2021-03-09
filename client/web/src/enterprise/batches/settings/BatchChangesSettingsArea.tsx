/* eslint-disable @typescript-eslint/no-unsafe-member-access */
/* eslint-disable @typescript-eslint/no-unsafe-call */
import React from 'react'
import { RouteComponentProps } from 'react-router'
import { PageHeader } from '../../../components/PageHeader'
import { PageTitle } from '../../../components/PageTitle'
import { BatchChangesCodeHostFields, UserAreaUserFields } from '../../../graphql-operations'
import { BatchChangesIcon } from '../icons'
import { queryUserBatchChangesCodeHosts } from './backend'
import { useBatchChanges } from './useBatchCanges'
import { CodeHostConnectionNode } from './CodeHostConnectionNode'

export interface BatchChangesSettingsAreaProps extends Pick<RouteComponentProps, 'history' | 'location'> {
    user: Pick<UserAreaUserFields, 'id'>
    queryUserBatchChangesCodeHosts?: typeof queryUserBatchChangesCodeHosts
}

/** The page area for all batch changes settings. It's shown in the user settings sidebar. */
export const BatchChangesSettingsArea: React.FunctionComponent<BatchChangesSettingsAreaProps> = () => {
    const userID = 'VXNlcjox' // TODO: We need a useGetUserID or similar
    const { status, data, isFetching }: { data: any; status: string; isFetching: boolean } = useBatchChanges(userID)

    if (status === 'loading') {
        return <div>loading</div>
    }

    return data ? (
        <div className="web-content test-batches-settings-page">
            <PageTitle title="Batch changes settings" />
            <PageHeader path={[{ icon: BatchChangesIcon, text: 'Batch changes' }]} className="mb-3" />
            <h2>Code host tokens</h2>
            <p>Add authentication tokens to enable batch changes changeset creation on your code hosts.</p>
            <div>{isFetching ? 'Background Updating...' : ' '}</div>
            {data.map((batch: BatchChangesCodeHostFields, index: number) => (
                <div key={index}>
                    <CodeHostConnectionNode node={batch} userID={userID} />
                </div>
            ))}

            <p>
                Code host not present? Site admins can add a code host in{' '}
                <a href="https://docs.sourcegraph.com/admin/external_service" target="_blank" rel="noopener noreferrer">
                    the manage repositories settings
                </a>
                .
            </p>
        </div>
    ) : null
}
