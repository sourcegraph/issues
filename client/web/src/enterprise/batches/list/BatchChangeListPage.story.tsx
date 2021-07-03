import { useCallback } from '@storybook/addons'
import { storiesOf } from '@storybook/react'
import React from 'react'
import { of } from 'rxjs'

import { EnterpriseWebStory } from '../../components/EnterpriseWebStory'

import { BatchChangeListPage } from './BatchChangeListPage'
import { nodes } from './testData'

const { add } = storiesOf('web/batches/BatchChangeListPage', module)
    .addDecorator(story => <div className="p-3 container">{story()}</div>)
    .addParameters({
        chromatic: {
            viewports: [320, 576, 978, 1440],
        },
    })

const queryBatchChanges = () =>
    of({
        batchChanges: {
            totalCount: Object.values(nodes).length,
            nodes: Object.values(nodes),
            pageInfo: { endCursor: null, hasNextPage: false },
        },
        totalCount: Object.values(nodes).length,
    })

const batchChangesNotLicensed = () => of(false)

const batchChangesLicensed = () => of(true)

add('List of batch changes', () => (
    <EnterpriseWebStory>
        {props => (
            <BatchChangeListPage
                {...props}
                isSourcegraphDotCom={false}
                headingElement="h1"
                queryBatchChanges={queryBatchChanges}
                areBatchChangesLicensed={batchChangesLicensed}
            />
        )}
    </EnterpriseWebStory>
))

add('Dotcom page', () => (
    <EnterpriseWebStory>
        {props => (
            <BatchChangeListPage
                {...props}
                isSourcegraphDotCom={true}
                headingElement="h1"
                queryBatchChanges={queryBatchChanges}
                areBatchChangesLicensed={batchChangesLicensed}
            />
        )}
    </EnterpriseWebStory>
))

add('Licensing not enforced', () => (
    <EnterpriseWebStory>
        {props => (
            <BatchChangeListPage
                {...props}
                isSourcegraphDotCom={false}
                headingElement="h1"
                queryBatchChanges={queryBatchChanges}
                areBatchChangesLicensed={batchChangesNotLicensed}
            />
        )}
    </EnterpriseWebStory>
))

add('No batch changes', () => {
    const queryBatchChanges = useCallback(
        () =>
            of({
                batchChanges: {
                    totalCount: 0,
                    nodes: [],
                    pageInfo: {
                        endCursor: null,
                        hasNextPage: false,
                    },
                },
                totalCount: 0,
            }),
        []
    )
    return (
        <EnterpriseWebStory>
            {props => (
                <BatchChangeListPage
                    {...props}
                    isSourcegraphDotCom={false}
                    headingElement="h1"
                    queryBatchChanges={queryBatchChanges}
                    areBatchChangesLicensed={batchChangesLicensed}
                />
            )}
        </EnterpriseWebStory>
    )
})

add('All batch changes tab empty', () => {
    const queryBatchChanges = useCallback(
        () =>
            of({
                batchChanges: {
                    totalCount: 0,
                    nodes: [],
                    pageInfo: {
                        endCursor: null,
                        hasNextPage: false,
                    },
                },
                totalCount: 0,
            }),
        []
    )
    return (
        <EnterpriseWebStory>
            {props => (
                <BatchChangeListPage
                    {...props}
                    isSourcegraphDotCom={false}
                    headingElement="h1"
                    queryBatchChanges={queryBatchChanges}
                    areBatchChangesLicensed={batchChangesLicensed}
                    openTab="batchChanges"
                />
            )}
        </EnterpriseWebStory>
    )
})
