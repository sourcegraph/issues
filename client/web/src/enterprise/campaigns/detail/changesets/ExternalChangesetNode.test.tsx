import * as H from 'history'
import React from 'react'
import { ExternalChangesetNode } from './ExternalChangesetNode'
import { shallow } from 'enzyme'
import {
    ChangesetReviewState,
    ChangesetPublicationState,
    ChangesetReconcilerState,
    ChangesetExternalState,
    ChangesetCheckState,
} from '../../../../graphql-operations'

describe('ExternalChangesetNode', () => {
    const history = H.createMemoryHistory({ keyLength: 0 })
    const location = H.createLocation('/campaigns')
    test('renders an externalchangeset', () => {
        expect(
            shallow(
                <ExternalChangesetNode
                    isLightTheme={true}
                    history={history}
                    location={location}
                    viewerCanAdminister={true}
                    node={{
                        __typename: 'ExternalChangeset',
                        id: 'TestExternalChangeset',
                        reviewState: ChangesetReviewState.PENDING,
                        publicationState: ChangesetPublicationState.PUBLISHED,
                        reconcilerState: ChangesetReconcilerState.COMPLETED,
                        externalState: ChangesetExternalState.OPEN,
                        externalURL: {
                            url: 'https://github.com/sourcegraph/sourcegraph/pull/111111',
                        },
                        title: 'Remove lodash',
                        body: 'We should remove lodash',
                        checkState: ChangesetCheckState.FAILED,
                        error: null,
                        externalID: '123',
                        diffStat: {
                            added: 100,
                            changed: 200,
                            deleted: 100,
                        },
                        labels: [
                            {
                                color: '93ba13',
                                description: 'Something is broken',
                                text: 'bug',
                            },
                        ],
                        repository: {
                            name: 'sourcegraph',
                            url: 'github.com/sourcegraph/sourcegraph',
                            id: 'TestRepository',
                        },
                        createdAt: new Date('2020-01-01').toISOString(),
                        updatedAt: new Date('2020-01-01').toISOString(),
                        nextSyncAt: null,
                        currentSpec: { id: 'spec-rand-id-1' },
                    }}
                />
            )
        ).toMatchSnapshot()
    })
})
