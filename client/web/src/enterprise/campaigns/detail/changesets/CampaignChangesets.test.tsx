import React from 'react'
import { CampaignChangesets } from './CampaignChangesets'
import * as H from 'history'
import { of } from 'rxjs'
import { NOOP_TELEMETRY_SERVICE } from '../../../../../../shared/src/telemetry/telemetryService'
import { mount } from 'enzyme'
import {
    ChangesetExternalState,
    ChangesetPublicationState,
    ChangesetReconcilerState,
} from '../../../../graphql-operations'

describe('CampaignChangesets', () => {
    const history = H.createMemoryHistory()
    test('renders', () =>
        expect(
            mount(
                <CampaignChangesets
                    queryChangesets={() =>
                        of({
                            totalCount: 1,
                            nodes: [
                                {
                                    id: '0',
                                    __typename: 'HiddenExternalChangeset',
                                    createdAt: new Date('2020-01-03').toISOString(),
                                    externalState: ChangesetExternalState.OPEN,
                                    nextSyncAt: null,
                                    publicationState: ChangesetPublicationState.PUBLISHED,
                                    reconcilerState: ChangesetReconcilerState.COMPLETED,
                                    updatedAt: new Date('2020-01-04').toISOString(),
                                },
                            ],
                            pageInfo: {
                                endCursor: null,
                                hasNextPage: false,
                            },
                        })
                    }
                    campaignID="123"
                    viewerCanAdminister={true}
                    history={history}
                    location={history.location}
                    isLightTheme={true}
                    extensionsController={undefined as any}
                    platformContext={undefined as any}
                    telemetryService={NOOP_TELEMETRY_SERVICE}
                />
            )
        ).toMatchSnapshot())
})
