import MapSearchIcon from 'mdi-react/MapSearchIcon'
import React, { FunctionComponent, useCallback, useEffect, useMemo } from 'react'
import { RouteComponentProps } from 'react-router'
import { of } from 'rxjs'

import { TelemetryProps } from '@sourcegraph/shared/src/telemetry/telemetryService'
import { useObservable } from '@sourcegraph/shared/src/util/useObservable'
import { Container, PageHeader } from '@sourcegraph/wildcard'

import {
    FilteredConnection,
    FilteredConnectionFilter,
    FilteredConnectionQueryArguments,
} from '../../../components/FilteredConnection'
import { PageTitle } from '../../../components/PageTitle'
import { LsifUploadFields, LSIFUploadState } from '../../../graphql-operations'

import {
    fetchLsifUploads as defaultFetchLsifUploads,
    fetchCommitGraphMetadata as defaultFetchCommitGraphMetadata,
} from './backend'
import { CodeIntelUploadNode, CodeIntelUploadNodeProps } from './CodeIntelUploadNode'
import { CommitGraphMetadata } from './CommitGraphMetadata'

export interface CodeIntelUploadsPageProps extends RouteComponentProps<{}>, TelemetryProps {
    repo?: { id: string }
    fetchLsifUploads?: typeof defaultFetchLsifUploads
    fetchCommitGraphMetadata?: typeof defaultFetchCommitGraphMetadata
    now?: () => Date
}

const filters: FilteredConnectionFilter[] = [
    {
        id: 'filters',
        label: 'Upload state',
        type: 'select',
        values: [
            {
                label: 'All',
                value: 'all',
                tooltip: 'Show all uploads',
                args: {},
            },
            {
                label: 'Current',
                value: 'current',
                tooltip: 'Show current uploads only',
                args: { isLatestForRepo: true },
            },
            {
                label: 'Completed',
                value: 'completed',
                tooltip: 'Show completed uploads only',
                args: { state: LSIFUploadState.COMPLETED },
            },
            {
                label: 'Errored',
                value: 'errored',
                tooltip: 'Show errored uploads only',
                args: { state: LSIFUploadState.ERRORED },
            },
            {
                label: 'Processing',
                value: 'processing',
                tooltip: 'Show processing uploads only',
                args: { state: LSIFUploadState.PROCESSING },
            },
            {
                label: 'Queued',
                value: 'queued',
                tooltip: 'Show queued uploads only',
                args: { state: LSIFUploadState.QUEUED },
            },
            {
                label: 'Uploading',
                value: 'uploading',
                tooltip: 'Show uploading uploads only',
                args: { state: LSIFUploadState.UPLOADING },
            },
            {
                label: 'Deleting',
                value: 'deleting',
                tooltip: 'Show uploads queued for deletion',
                args: { state: LSIFUploadState.DELETING },
            },
        ],
    },
]

export const CodeIntelUploadsPage: FunctionComponent<CodeIntelUploadsPageProps> = ({
    repo,
    fetchLsifUploads = defaultFetchLsifUploads,
    fetchCommitGraphMetadata = defaultFetchCommitGraphMetadata,
    now,
    telemetryService,
    ...props
}) => {
    useEffect(() => telemetryService.logViewEvent('CodeIntelUploads'), [telemetryService])

    const queryUploads = useCallback(
        (args: FilteredConnectionQueryArguments) => fetchLsifUploads({ repository: repo?.id, ...args }),
        [repo?.id, fetchLsifUploads]
    )

    const commitGraphMetadata = useObservable(
        useMemo(() => (repo ? fetchCommitGraphMetadata({ repository: repo?.id }) : of(undefined)), [
            repo,
            fetchCommitGraphMetadata,
        ])
    )

    return (
        <div className="code-intel-uploads">
            <PageTitle title="Precise code intelligence uploads" />
            <PageHeader
                headingElement="h2"
                path={[{ text: 'Precise code intelligence uploads' }]}
                description={
                    <>
                        Current uploads provide code intelligence for the latest commit on the default branch and are
                        used in cross-repository <em>Find References</em> requests. Non-current uploads may still
                        provide code intelligence for historic and branch commits.
                    </>
                }
                className="mb-3"
            />

            <Container>
                {repo && commitGraphMetadata && (
                    <CommitGraphMetadata
                        stale={commitGraphMetadata.stale}
                        updatedAt={commitGraphMetadata.updatedAt}
                        now={now}
                    />
                )}

                <div className="list-group position-relative">
                    <FilteredConnection<LsifUploadFields, Omit<CodeIntelUploadNodeProps, 'node'>>
                        listComponent="div"
                        listClassName="codeintel-uploads__grid mb-3"
                        noun="upload"
                        pluralNoun="uploads"
                        nodeComponent={CodeIntelUploadNode}
                        nodeComponentProps={{ now }}
                        queryConnection={queryUploads}
                        history={props.history}
                        location={props.location}
                        cursorPaging={true}
                        filters={filters}
                        defaultFilter="current"
                        emptyElement={<EmptyLSIFUploadsElement />}
                    />
                </div>
            </Container>
        </div>
    )
}

const EmptyLSIFUploadsElement: React.FunctionComponent = () => (
    <p className="text-muted text-center w-100 mb-0 mt-1">
        <MapSearchIcon className="mb-2" />
        <br />
        No uploads yet. Enable precise code intelligence by{' '}
        <a
            href="https://docs.sourcegraph.com/code_intelligence/explanations/precise_code_intelligence"
            target="_blank"
            rel="noreferrer noopener"
        >
            uploading LSIF data
        </a>
        .
    </p>
)
