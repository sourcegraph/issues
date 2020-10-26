import { CodeIntelUploadsPage } from './CodeIntelUploadsPage'
import { of } from 'rxjs'
import { storiesOf } from '@storybook/react'
import { SuiteFunction } from 'mocha'
import { Upload } from './backend'
import * as H from 'history'
import React from 'react'
import webStyles from '../../SourcegraphWebApp.scss'
import { SourcegraphContext } from '../../jscontext'
import { LSIFUploadState } from '../../../../shared/src/graphql-operations'
import { NOOP_TELEMETRY_SERVICE } from '../../../../shared/src/telemetry/telemetryService'

window.context = {} as SourcegraphContext & SuiteFunction

const { add } = storiesOf('web/Codeintel administration/CodeIntelUploads', module).addDecorator(story => (
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

const upload: Pick<Upload, 'projectRoot' | 'inputCommit' | 'inputRoot' | 'inputIndexer' | 'isLatestForRepo'> = {
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
    inputRoot: 'web/',
    inputIndexer: 'lsif-tsc',
    isLatestForRepo: false,
}

add('List', () => (
    <CodeIntelUploadsPage
        {...commonProps}
        fetchLsifUploads={() =>
            of({
                nodes: [
                    {
                        ...upload,
                        id: '1',
                        state: LSIFUploadState.COMPLETED,
                        uploadedAt: '2020-06-15T12:20:30+00:00',
                        startedAt: '2020-06-15T12:25:30+00:00',
                        finishedAt: '2020-06-15T12:30:30+00:00',
                        failure: null,
                        placeInQueue: null,
                    },
                    {
                        ...upload,
                        id: '2',
                        state: LSIFUploadState.ERRORED,
                        uploadedAt: '2020-06-15T12:20:30+00:00',
                        startedAt: '2020-06-15T12:25:30+00:00',
                        finishedAt: '2020-06-15T12:30:30+00:00',
                        failure: 'Whoops! The server encountered a boo-boo handling this input.',
                        placeInQueue: null,
                    },
                    {
                        ...upload,
                        id: '3',
                        state: LSIFUploadState.PROCESSING,
                        uploadedAt: '2020-06-15T12:20:30+00:00',
                        startedAt: '2020-06-15T12:25:30+00:00',
                        finishedAt: null,
                        failure: null,
                        placeInQueue: null,
                    },
                    {
                        ...upload,
                        id: '4',
                        state: LSIFUploadState.QUEUED,
                        uploadedAt: '2020-06-15T12:20:30+00:00',
                        startedAt: null,
                        finishedAt: null,
                        placeInQueue: 3,
                        failure: null,
                    },
                    {
                        ...upload,
                        id: '5',
                        state: LSIFUploadState.UPLOADING,
                        uploadedAt: '2020-06-15T12:20:30+00:00',
                        startedAt: null,
                        finishedAt: null,
                        failure: null,
                        placeInQueue: null,
                    },
                ],
                totalCount: 10,
                pageInfo: {
                    __typename: 'PageInfo',
                    endCursor: 'fakenextpage',
                    hasNextPage: true,
                },
            })
        }
    />
))
