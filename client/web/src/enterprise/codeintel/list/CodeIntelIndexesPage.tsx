import React, { FunctionComponent, useCallback, useEffect } from 'react'
import { RouteComponentProps } from 'react-router'

import { TelemetryProps } from '@sourcegraph/shared/src/telemetry/telemetryService'
import { Container, PageHeader } from '@sourcegraph/wildcard'

import {
    FilteredConnection,
    FilteredConnectionFilter,
    FilteredConnectionQueryArguments,
} from '../../../components/FilteredConnection'
import { PageTitle } from '../../../components/PageTitle'
import { LsifIndexFields, LSIFIndexState } from '../../../graphql-operations'

import { fetchLsifIndexes as defaultFetchLsifIndexes } from './backend'
import { CodeIntelIndexNode, CodeIntelIndexNodeProps } from './CodeIntelIndexNode'

export interface CodeIntelIndexesPageProps extends RouteComponentProps<{}>, TelemetryProps {
    repo?: { id: string }
    fetchLsifIndexes?: typeof defaultFetchLsifIndexes
    now?: () => Date
}

const filters: FilteredConnectionFilter[] = [
    {
        id: 'filters',
        label: 'Index state',
        type: 'select',
        values: [
            {
                label: 'All',
                value: 'all',
                tooltip: 'Show all indexes',
                args: {},
            },
            {
                label: 'Completed',
                value: 'completed',
                tooltip: 'Show completed indexes only',
                args: { state: LSIFIndexState.COMPLETED },
            },
            {
                label: 'Errored',
                value: 'errored',
                tooltip: 'Show errored indexes only',
                args: { state: LSIFIndexState.ERRORED },
            },
            {
                label: 'Processing',
                value: 'processing',
                tooltip: 'Show processing indexes only',
                args: { state: LSIFIndexState.PROCESSING },
            },
            {
                label: 'Queued',
                value: 'queued',
                tooltip: 'Show queued indexes only',
                args: { state: LSIFIndexState.QUEUED },
            },
        ],
    },
]

export const CodeIntelIndexesPage: FunctionComponent<CodeIntelIndexesPageProps> = ({
    repo,
    fetchLsifIndexes = defaultFetchLsifIndexes,
    now,
    telemetryService,
    ...props
}) => {
    useEffect(() => telemetryService.logViewEvent('CodeIntelIndexes'), [telemetryService])

    const queryIndexes = useCallback(
        (args: FilteredConnectionQueryArguments) => fetchLsifIndexes({ repository: repo?.id, ...args }),
        [repo?.id, fetchLsifIndexes]
    )

    return (
        <div className="code-intel-indexes">
            <PageTitle title="Precise code intelligence auto-index records" />
            <PageHeader
                headingElement="h2"
                path={[{ text: 'Precise code intelligence auto-index records' }]}
                description={
                    <>
                        Popular repositories are indexed automatically on{' '}
                        <a href="https://sourcegraph.com" target="_blank" rel="noreferrer noopener">
                            Sourcegraph.com
                        </a>
                        .
                    </>
                }
                className="mb-3"
            />

            <Container>
                <div className="list-group position-relative">
                    <FilteredConnection<LsifIndexFields, Omit<CodeIntelIndexNodeProps, 'node'>>
                        listComponent="div"
                        listClassName="codeintel-indexes__grid mb-3"
                        noun="index"
                        pluralNoun="indexes"
                        nodeComponent={CodeIntelIndexNode}
                        nodeComponentProps={{ now }}
                        queryConnection={queryIndexes}
                        history={props.history}
                        location={props.location}
                        cursorPaging={true}
                        filters={filters}
                        defaultFilter="All"
                    />
                </div>
            </Container>
        </div>
    )
}
