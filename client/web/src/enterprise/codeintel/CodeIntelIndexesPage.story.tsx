import { CodeIntelIndexesPage } from './CodeIntelIndexesPage'
import { Index } from './backend'
import { of } from 'rxjs'
import { storiesOf } from '@storybook/react'
import { SuiteFunction } from 'mocha'
import * as H from 'history'
import React from 'react'
import webStyles from '../../SourcegraphWebApp.scss'
import { SourcegraphContext } from '../../jscontext'
import { LSIFIndexState } from '../../../../shared/src/graphql-operations'
import { NOOP_TELEMETRY_SERVICE } from '../../../../shared/src/telemetry/telemetryService'

window.context = {} as SourcegraphContext & SuiteFunction

const { add } = storiesOf('web/Codeintel administration/CodeIntelIndexes', module).addDecorator(story => (
    <>
        <div className="theme-light container">{story()}</div>
        <style>{webStyles}</style>
    </>
))

const history = H.createMemoryHistory()

const commonProps = {
    history,
    location: history.location,
    match: {
        params: { id: '' },
        isExact: true,
        path: '',
        url: '',
    },
    now: () => new Date('2020-06-15T15:25:00+00:00'),
    telemetryService: NOOP_TELEMETRY_SERVICE,
}

const index: Pick<Index, 'projectRoot' | 'inputCommit'> = {
    projectRoot: {
        url: '',
        path: 'web/',
        repository: {
            url: '',
            name: 'github.com/sourcegraph/sourcegraph',
        },
        commit: {
            url: '',
            oid: '9ea5e9f0e0344f8197622df6b36faf48ccd02570',
            abbreviatedOID: '9ea5e9f',
        },
    },
    inputCommit: '9ea5e9f0e0344f8197622df6b36faf48ccd02570',
}

add('List', () => (
    <CodeIntelIndexesPage
        {...commonProps}
        fetchLsifIndexes={() =>
            of({
                nodes: [
                    {
                        ...index,
                        id: '1',
                        state: LSIFIndexState.COMPLETED,
                        queuedAt: '2020-06-15T12:20:30+00:00',
                        startedAt: '2020-06-15T12:25:30+00:00',
                        finishedAt: '2020-06-15T12:30:30+00:00',
                        failure: null,
                        placeInQueue: null,
                    },
                    {
                        ...index,
                        id: '2',
                        state: LSIFIndexState.ERRORED,
                        queuedAt: '2020-06-15T12:20:30+00:00',
                        startedAt: '2020-06-15T12:25:30+00:00',
                        finishedAt: '2020-06-15T12:30:30+00:00',
                        failure: 'Whoops! The server encountered a boo-boo handling this input.',
                        placeInQueue: null,
                    },
                    {
                        ...index,
                        id: '3',
                        state: LSIFIndexState.PROCESSING,
                        queuedAt: '2020-06-15T12:20:30+00:00',
                        startedAt: '2020-06-15T12:25:30+00:00',
                        finishedAt: null,
                        failure: null,
                        placeInQueue: null,
                    },
                    {
                        ...index,
                        id: '4',
                        state: LSIFIndexState.QUEUED,
                        queuedAt: '2020-06-15T12:20:30+00:00',
                        startedAt: null,
                        finishedAt: null,
                        placeInQueue: 3,
                        failure: null,
                    },
                ],
                totalCount: 8,
                pageInfo: {
                    __typename: 'PageInfo',
                    endCursor: 'fakenextpage',
                    hasNextPage: true,
                },
            })
        }
    />
))
